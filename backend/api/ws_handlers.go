package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/websocket"
)

// ==========================================
// WebSocket 相關的 Handlers
// ==========================================

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true }, // 允許所有來源連線
}

// HandleWS 處理 WebSocket 連線
func (s *Server) HandleWS(w http.ResponseWriter, r *http.Request) {
	// 記錄 WebSocket 連線建立 metrics（手動記錄，因為繞過了 MetricsMiddleware）
	RequestCounter.WithLabelValues(r.Method, "/ws", "Switching Protocols").Inc()

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Printf("❌ WS 升級失敗: %v\n", err)
		return
	}

	// 註冊客戶端並預設訂閱所有 device:* 頻道
	s.Hub.AddClient(conn)

	// 保持連線，直到客戶端斷開
	defer s.Hub.RemoveClient(conn)

	// 處理客戶端訊息（訂閱/取消訂閱等）
	for {
		messageType, message, err := conn.ReadMessage()
		if err != nil {
			// 連接關閉或讀取錯誤
			break
		}

		// 處理文字訊息（JSON 格式的訂閱請求）
		if messageType == websocket.TextMessage {
			var msg map[string]interface{}
			if err := json.Unmarshal(message, &msg); err == nil {
				// 處理訂閱請求
				if action, ok := msg["action"].(string); ok {
					switch action {
					case "subscribe":
						if topic, ok := msg["topic"].(string); ok {
							s.Hub.Subscribe(topic, conn)
							// 回傳確認訊息
							response := map[string]interface{}{
								"status": "subscribed",
								"topic":  topic,
							}
							if data, err := json.Marshal(response); err == nil {
								conn.WriteMessage(websocket.TextMessage, data)
							}
						}
					case "unsubscribe":
						if topic, ok := msg["topic"].(string); ok {
							s.Hub.Unsubscribe(topic, conn)
							// 回傳確認訊息
							response := map[string]interface{}{
								"status": "unsubscribed",
								"topic":  topic,
							}
							if data, err := json.Marshal(response); err == nil {
								conn.WriteMessage(websocket.TextMessage, data)
							}
						}
					}
				}
			}
		}
	}
}
