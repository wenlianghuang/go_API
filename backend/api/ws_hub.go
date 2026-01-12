// backend/api/ws_hub.go

package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
)

type Hub struct {
	// 本地訂閱：Topic -> 客戶端集合
	topics map[string]map[*websocket.Conn]bool
	mu     sync.Mutex

	// Redis 客戶端
	rdb *redis.Client

	// Redis 監聽的模式列表
	redisPatterns []string

	// 🆕 優雅停機支援
	pubsub *redis.PubSub      // Redis 訂閱對象，用於關閉
	ctx    context.Context    // 控制 Redis 監聽器的生命週期
	cancel context.CancelFunc // 用於取消 context，觸發停機
	wg     sync.WaitGroup     // 用於等待 goroutine 完成
}

func NewHub(rdb *redis.Client) *Hub {
	// 定義要監聽的 Redis 模式列表
	patterns := []string{
		"device:*", // 設備相關訊息
		"value:*",  // 數值相關訊息
		// 可以在這裡添加更多模式
	}

	// 🆕 創建可取消的 context，用於控制停機
	ctx, cancel := context.WithCancel(context.Background())

	h := &Hub{
		topics:        make(map[string]map[*websocket.Conn]bool),
		rdb:           rdb,
		redisPatterns: patterns,
		ctx:           ctx,    // 🆕 儲存 context
		cancel:        cancel, // 🆕 儲存 cancel 函數
	}

	// 🔥 關鍵：啟動背景任務監聽 Redis
	// 🆕 使用 WaitGroup 追蹤 goroutine，以便停機時等待
	h.wg.Add(1)
	go h.listenToRedis()

	return h
}

// listenToRedis 監聽 Redis 的廣播，收到後分發給本地連線
// 🆕 支援優雅停機：當 h.ctx 被取消時，會自動退出
func (h *Hub) listenToRedis() {
	// 🆕 確保在函數退出時標記 WaitGroup 完成
	defer h.wg.Done()

	// 🆕 使用 Hub 的 context 代替 Background context
	// 這樣當 context 被取消時，訂閱會自動關閉
	h.pubsub = h.rdb.PSubscribe(h.ctx, h.redisPatterns...)
	defer h.pubsub.Close()

	ch := h.pubsub.Channel()
	fmt.Printf("👷 Redis 監聽器已啟動，監聽模式: %v\n", h.redisPatterns)

	// 🆕 使用 select 監聽 context 取消信號和 Redis 訊息
	for {
		select {
		case <-h.ctx.Done():
			// 🆕 收到停機信號，優雅退出
			fmt.Println("🛑 Redis 監聽器收到停機信號，準備關閉...")
			return

		case msg, ok := <-ch:
			// 🆕 檢查 channel 是否已關閉
			if !ok {
				log.Printf("⚠️ Redis 監聽器 channel 已關閉")
				return
			}
			// 當 Redis 收到訊息時，msg.Channel 就是 Topic (如 device:1 或 value:22.5)
			// msg.Payload 就是數據內容
			if msg == nil {
				log.Printf("⚠️ Redis 收到 nil 訊息，可能連接已關閉")
				return
			}
			// 檢查是否為錯誤訊息（go-redis 會在錯誤時發送特殊格式的訊息）
			if msg.Payload == "" && msg.Channel == "" {
				log.Printf("⚠️ Redis 收到空訊息")
				continue
			}
			// 正常處理訊息
			h.broadcastToLocal(msg.Channel, msg.Payload)
		}
	}
}

// matchPattern 檢查 topic 是否匹配 pattern
// 例如: value:22.5 匹配 value:*
func matchPattern(topic, pattern string) bool {
	// 如果完全匹配，直接返回 true
	if topic == pattern {
		return true
	}

	// 處理通配符模式
	if strings.HasSuffix(pattern, "*") {
		prefix := strings.TrimSuffix(pattern, "*")
		return strings.HasPrefix(topic, prefix)
	}

	return false
}

// broadcastToLocal 僅發送給「連線到這台伺服器」的用戶
// 支援模式匹配：如果訂閱了 value:*，會收到 value:22.5 的訊息
func (h *Hub) broadcastToLocal(topic string, payload string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// 收集需要發送訊息的連接
	clientsToNotify := make(map[*websocket.Conn]bool)

	// 遍歷所有訂閱的 topics，找出匹配的訂閱者
	for subscribedTopic, clients := range h.topics {
		// 檢查是否完全匹配或模式匹配
		if matchPattern(topic, subscribedTopic) {
			for conn := range clients {
				clientsToNotify[conn] = true
			}
		}
	}

	if len(clientsToNotify) == 0 {
		return
	}

	// 收集需要移除的連接（避免在迭代時修改 map）
	var toRemove []*websocket.Conn

	for conn := range clientsToNotify {
		err := conn.WriteMessage(websocket.TextMessage, []byte(payload))
		if err != nil {
			log.Printf("⚠️ 發送訊息到 WebSocket 失敗: %v", err)
			toRemove = append(toRemove, conn)
		}
	}

	// 移除失敗的連接
	for _, conn := range toRemove {
		// 從所有 topics 中移除這個連接
		for topicKey, clients := range h.topics {
			if clients[conn] {
				delete(clients, conn)
				// 如果該 topic 沒有客戶端了，清理空的 map
				if len(clients) == 0 {
					delete(h.topics, topicKey)
				}
			}
		}
		if err := conn.Close(); err != nil {
			log.Printf("⚠️ 關閉失敗的 WebSocket 連接時發生錯誤: %v", err)
		}
	}
}

// Subscribe 僅在本地註冊訂閱關係
func (h *Hub) Subscribe(topic string, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.topics[topic] == nil {
		h.topics[topic] = make(map[*websocket.Conn]bool)
	}
	h.topics[topic][conn] = true
	fmt.Printf("🌐 本地用戶訂閱了 Redis 頻道: %s\n", topic)
}

// Unsubscribe 取消訂閱特定頻道
func (h *Hub) Unsubscribe(topic string, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()

	clients, ok := h.topics[topic]
	if !ok {
		return
	}

	delete(clients, conn)
	fmt.Printf("🌐 本地用戶取消訂閱 Redis 頻道: %s\n", topic)

	// 如果該 topic 沒有客戶端了，清理空的 map
	if len(clients) == 0 {
		delete(h.topics, topic)
	}
}

// Publish 不再直接發給本地，而是發給 Redis
func (h *Hub) Publish(ctx context.Context, topic string, v interface{}) error {
	data, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("JSON 序列化失敗: %w", err)
	}
	// 🔥 將訊息發布到 Redis，所有伺服器都會收到
	if err := h.rdb.Publish(ctx, topic, data).Err(); err != nil {
		return fmt.Errorf("Redis Publish 失敗: %w", err)
	}
	return nil
}

// AddClient 註冊新連線並預設訂閱所有 device:* 頻道
// 這樣客戶端連接後就能收到所有設備的訊息
func (h *Hub) AddClient(conn *websocket.Conn) {
	fmt.Printf("🌐 新的 WebSocket 客戶端已連線\n")
	// 預設訂閱所有 device:* 頻道，讓客戶端能收到所有設備的訊息
	h.Subscribe("device:*", conn)
	// Update metrics
	ActiveConnections.Inc()
}

// RemoveClient 移除連線，從所有 topics 中清除該連接
func (h *Hub) RemoveClient(conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// 遍歷所有 topics，移除該連接
	for topic, clients := range h.topics {
		if clients[conn] {
			delete(clients, conn)
			// 如果該 topic 沒有客戶端了，清理空的 map
			if len(clients) == 0 {
				delete(h.topics, topic)
			}
		}
	}

	// 關閉連接
	if err := conn.Close(); err != nil {
		log.Printf("⚠️ 關閉 WebSocket 連接時發生錯誤: %v", err)
	}
	fmt.Printf("🌐 WebSocket 客戶端已斷開，已從所有 topics 中移除\n")
	// Update metrics
	ActiveConnections.Dec()
}

// Broadcast 推播訊息給所有訂閱者（為了向後兼容）
// 內部使用 Publish 將訊息發布到 Redis，所有伺服器都會收到並分發
// 如果提供了 deviceID，會發布到具體的 topic (device:{id})，否則發布到 device:*
func (h *Hub) Broadcast(v interface{}) {
	h.BroadcastToDevice(0, v)
}

// BroadcastToDevice 推播訊息到特定設備的訂閱者
// 如果 deviceID 為 0，則發布到 device:*（所有設備）
// 否則發布到 device:{deviceID}
func (h *Hub) BroadcastToDevice(deviceID uint, v interface{}) {
	ctx := context.Background()
	var topic string
	if deviceID == 0 {
		topic = "device:*"
	} else {
		topic = fmt.Sprintf("device:%d", deviceID)
	}
	if err := h.Publish(ctx, topic, v); err != nil {
		log.Printf("❌ Broadcast 失敗 (topic: %s): %v", topic, err)
	}
}

// 🆕 Shutdown 優雅停機：關閉所有連接並等待 goroutine 退出
// 此方法會：
// 1. 取消 context，停止 Redis 監聽器
// 2. 關閉所有 WebSocket 連接（通知客戶端服務器正在關閉）
// 3. 等待 Redis 監聽器 goroutine 完全退出
func (h *Hub) Shutdown(ctx context.Context) error {
	fmt.Println("🔄 正在關閉 WebSocket Hub...")

	// 步驟 1：取消 context，觸發 Redis 監聽器停止
	h.cancel()

	// 步驟 2：收集所有 WebSocket 連接並關閉
	h.mu.Lock()
	var allConns []*websocket.Conn
	// 遍歷所有 topics，收集所有連接
	for _, clients := range h.topics {
		for conn := range clients {
			allConns = append(allConns, conn)
		}
	}
	// 清空 topics map（因為我們要關閉所有連接）
	h.topics = make(map[string]map[*websocket.Conn]bool)
	h.mu.Unlock()

	// 發送 WebSocket 關閉幀給所有客戶端
	// CloseGoingAway (1001) 表示服務器正在關閉
	closeMessage := websocket.FormatCloseMessage(websocket.CloseGoingAway, "Server is shutting down")
	for _, conn := range allConns {
		// 發送關閉消息（設定 1 秒超時）
		_ = conn.WriteControl(websocket.CloseMessage, closeMessage, time.Now().Add(time.Second))
		// 關閉連接
		_ = conn.Close()
	}
	fmt.Printf("✅ 已關閉 %d 個 WebSocket 連接\n", len(allConns))

	// 步驟 3：等待 Redis 監聽器 goroutine 退出（帶超時保護）
	done := make(chan struct{})
	go func() {
		h.wg.Wait() // 等待所有 goroutine 完成
		close(done)
	}()

	select {
	case <-done:
		// 所有 goroutine 已完成
		fmt.Println("✅ Redis 監聽器已安全關閉")
		return nil
	case <-ctx.Done():
		// 超時了，但 goroutine 可能還在運行
		return fmt.Errorf("Hub 停機超時，但已發送停止信號")
	}
}
