// backend/api/ws_hub.go

package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"

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
}

func NewHub(rdb *redis.Client) *Hub {
	// 定義要監聽的 Redis 模式列表
	patterns := []string{
		"device:*", // 設備相關訊息
		"value:*",  // 數值相關訊息
		// 可以在這裡添加更多模式
	}

	h := &Hub{
		topics:        make(map[string]map[*websocket.Conn]bool),
		rdb:           rdb,
		redisPatterns: patterns,
	}

	// 🔥 關鍵：啟動背景任務監聽 Redis
	go h.listenToRedis()

	return h
}

// listenToRedis 監聽 Redis 的廣播，收到後分發給本地連線
func (h *Hub) listenToRedis() {
	ctx := context.Background()

	// 使用 Pattern Subscribe 同時監聽多個模式
	pubsub := h.rdb.PSubscribe(ctx, h.redisPatterns...)
	defer pubsub.Close()

	ch := pubsub.Channel()
	fmt.Printf("👷 Redis 監聽器已啟動，監聽模式: %v\n", h.redisPatterns)

	for msg := range ch {
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
	// Channel 關閉時記錄日誌（可能是正常關閉或錯誤）
	log.Printf("⚠️ Redis 監聽器 channel 已關閉")
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
