package api

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/gorilla/websocket"
)

// Hub 管理所有的活動連線
type Hub struct {
	// 儲存所有連線中的客戶端，使用 map 方便刪除
	clients map[*websocket.Conn]bool
	// 互斥鎖，確保在併發環境下存取 map 是安全的
	mu sync.Mutex
}

func NewHub() *Hub {
	return &Hub{
		clients: make(map[*websocket.Conn]bool),
	}
}

// AddClient 註冊新連線
func (h *Hub) AddClient(conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.clients[conn] = true
	fmt.Println("🌐 新的 WebSocket 客戶端已連線")
}

// RemoveClient 移除連線
func (h *Hub) RemoveClient(conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.clients, conn)
	conn.Close()
	fmt.Println("🌐 WebSocket 客戶端已斷開")
}

// Broadcast 推播訊息給所有人
func (h *Hub) Broadcast(v interface{}) {
	h.mu.Lock()
	defer h.mu.Unlock()

	data, _ := json.Marshal(v)
	for conn := range h.clients {
		err := conn.WriteMessage(websocket.TextMessage, data)
		if err != nil {
			fmt.Printf("❌ 推播失敗: %v\n", err)
			conn.Close()
			delete(h.clients, conn)
		}
	}
}
