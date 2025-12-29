# WebSocket 實時推播

本文件詳細說明 WebSocket 和 Redis Pub/Sub 的實時推播機制。

## 🔌 WebSocket 基本知識

### 什麼是 WebSocket？

WebSocket 是一種通訊協定，允許伺服器與客戶端建立持久性的雙向連線。與傳統的 HTTP 請求-回應模式不同，WebSocket 可以讓伺服器主動推送數據給客戶端，無需客戶端不斷輪詢（Polling）。

### 為什麼使用 WebSocket？

1. **實時性**：數據可以即時推送到所有連線的客戶端
2. **效率**：避免客戶端頻繁發送 HTTP 請求
3. **雙向通訊**：伺服器和客戶端都可以主動發送訊息
4. **低延遲**：建立連線後，數據傳輸延遲極低

### 本專案的應用場景

- 當創建新的遙測數據時，自動推播給所有監聽的客戶端
- 當更新遙測數據時，自動通知所有客戶端數據變更
- 適合用於 IoT 儀表板、即時監控系統等場景

## 🏗 WebSocket 架構設計

本專案採用 **Hub 模式（Hub Pattern）** 結合 **Redis Pub/Sub** 來實現跨服務器的 WebSocket 廣播：

```
┌─────────────────────────────────────────────────────────┐
│                    API Server 1                          │
│  ┌──────────────────────────────────────────────────┐   │
│  │         WebSocket Hub                           │   │
│  │  - 管理本地 WebSocket 連線                      │   │
│  │  - 訂閱 Redis 頻道                              │   │
│  └──────────────┬──────────────────────────────────┘   │
│                 │                                        │
│    ┌────────────┼────────────┐                          │
│    │            │            │                          │
│ ┌──▼───┐    ┌──▼───┐    ┌──▼───┐                      │
│ │Client1│   │Client2│   │Client3│                      │
│ └──────┘    └──────┘    └──────┘                      │
└──────────────┬─────────────────────────────────────────┘
               │
               │ Publish/Subscribe
               │
┌──────────────▼─────────────────────────────────────────┐
│                    Redis Server                         │
│  - 作為訊息中介層                                       │
│  - 支援多個模式訂閱 (device:*, value:*)                │
└──────────────┬─────────────────────────────────────────┘
               │
               │ Publish/Subscribe
               │
┌──────────────▼─────────────────────────────────────────┐
│                    API Server 2                         │
│  ┌──────────────────────────────────────────────────┐   │
│  │         WebSocket Hub                           │   │
│  │  - 管理本地 WebSocket 連線                      │   │
│  │  - 訂閱 Redis 頻道                              │   │
│  └──────────────┬──────────────────────────────────┘   │
│                 │                                        │
│    ┌────────────┼────────────┐                          │
│    │            │            │                          │
│ ┌──▼───┐    ┌──▼───┐    ┌──▼───┐                      │
│ │Client4│   │Client5│   │Client6│                      │
│ └──────┘    └──────┘    └──────┘                      │
└─────────────────────────────────────────────────────────┘
```

**核心組件**

1. **Hub** (`api/ws_hub.go`)：管理所有 WebSocket 連線和 Redis 訂閱
2. **Handler** (`api/handlers.go`)：處理 WebSocket 連線升級和客戶端訂閱請求
3. **Redis Pub/Sub**：實現跨服務器的訊息廣播
4. **Broadcast**：當遙測數據變更時，透過 Redis 推播給所有服務器的客戶端

## 📡 Redis Publish/Subscribe 架構

### 為什麼需要 Redis？

在微服務或水平擴展的環境中，可能有多個 API 服務器實例同時運行。如果只使用本地 Hub，每個服務器只能推播給連接到自己的客戶端，無法實現跨服務器的廣播。

**解決方案：使用 Redis Pub/Sub**

1. **集中式訊息中介**：所有服務器都連接到同一個 Redis 實例
2. **跨服務器廣播**：當一個服務器發布訊息時，所有服務器都能收到
3. **模式訂閱**：支援使用通配符訂閱多個頻道（如 `device:*`、`value:*`）

### Redis 工作流程

```
1. API Server 1 收到創建遙測數據的請求
   ↓
2. 數據存入 PostgreSQL
   ↓
3. Hub.Publish() 將訊息發布到 Redis (topic: device:1)
   ↓
4. Redis 將訊息廣播給所有訂閱該模式的服務器
   ↓
5. 所有服務器的 listenToRedis() 收到訊息
   ↓
6. 各服務器的 broadcastToLocal() 分發給本地訂閱的客戶端
   ↓
7. 所有連線的 WebSocket 客戶端收到訊息
```

### Redis 模式訂閱

系統預設監聽以下模式：

- `device:*` - 設備相關訊息（如 `device:1`、`device:2`）
- `value:*` - 數值相關訊息（如 `value:22.5`、`value:30.0`）

可以在 `NewHub()` 函數中添加更多模式：

```go
patterns := []string{
    "device:*",  // 設備相關訊息
    "value:*",   // 數值相關訊息
    "sensor:*",  // 感測器相關訊息（可擴展）
    "alert:*",   // 警報相關訊息（可擴展）
}
```

### 模式匹配機制

系統支援模式匹配功能，允許客戶端訂閱模式 topic：

- 訂閱 `value:*` 可以收到所有以 `value:` 開頭的訊息（如 `value:22.5`、`value:30.0`）
- 訂閱 `device:1` 只會收到該設備的訊息
- 訂閱 `device:*` 會收到所有設備的訊息

## 💻 WebSocket 代碼實現

### 1. Hub 結構 (`api/ws_hub.go`)

```go
type Hub struct {
    // 本地訂閱：Topic -> 客戶端集合
    topics map[string]map[*websocket.Conn]bool
    mu     sync.Mutex
    
    // Redis 客戶端
    rdb *redis.Client
    
    // Redis 監聽的模式列表
    redisPatterns []string
}
```

**主要方法：**

- `AddClient(conn)`: 註冊新的 WebSocket 連線（預設訂閱 `device:*`）
- `RemoveClient(conn)`: 移除並關閉連線
- `Subscribe(topic, conn)`: 訂閱特定 topic（支援模式匹配）
- `Unsubscribe(topic, conn)`: 取消訂閱特定 topic
- `Publish(topic, v)`: 將訊息發布到 Redis
- `Broadcast(v)`: 推播訊息給所有訂閱者（透過 Redis）
- `BroadcastToDevice(deviceID, v)`: 推播訊息到特定設備
- `listenToRedis()`: 背景監聽 Redis 訊息並分發給本地客戶端

**線程安全設計：**

使用 `sync.Mutex` 確保在多個 goroutine 同時存取 `topics` map 時不會發生競態條件（Race Condition）。

### 2. WebSocket Handler (`api/handlers.go`)

```go
var upgrader = websocket.Upgrader{
    CheckOrigin: func(r *http.Request) bool { return true }, // 允許所有來源連線
}

func (s *Server) HandleWS(w http.ResponseWriter, r *http.Request) {
    // 1. 將 HTTP 連線升級為 WebSocket
    conn, err := upgrader.Upgrade(w, r, nil)
    
    // 2. 註冊到 Hub（預設訂閱 device:*）
    s.Hub.AddClient(conn)
    
    // 3. 保持連線，處理客戶端訊息
    defer s.Hub.RemoveClient(conn)
    for {
        messageType, message, err := conn.ReadMessage()
        if err != nil {
            break
        }
        
        // 處理訂閱/取消訂閱請求
        if messageType == websocket.TextMessage {
            var msg map[string]interface{}
            if err := json.Unmarshal(message, &msg); err == nil {
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
                            conn.WriteMessage(websocket.TextMessage, 
                                json.Marshal(response))
                        }
                    case "unsubscribe":
                        if topic, ok := msg["topic"].(string); ok {
                            s.Hub.Unsubscribe(topic, conn)
                            // 回傳確認訊息
                            response := map[string]interface{}{
                                "status": "unsubscribed",
                                "topic":  topic,
                            }
                            conn.WriteMessage(websocket.TextMessage, 
                                json.Marshal(response))
                        }
                    }
                }
            }
        }
    }
}
```

**客戶端訂閱範例：**

```json
// 訂閱特定 topic
{
  "action": "subscribe",
  "topic": "value:*"
}

// 取消訂閱
{
  "action": "unsubscribe",
  "topic": "value:*"
}
```

### 3. Redis 監聽器 (`api/ws_hub.go`)

```go
// listenToRedis 監聽 Redis 的廣播，收到後分發給本地連線
func (h *Hub) listenToRedis() {
    ctx := context.Background()
    
    // 使用 Pattern Subscribe 同時監聽多個模式
    pubsub := h.rdb.PSubscribe(ctx, h.redisPatterns...)
    defer pubsub.Close()
    
    ch := pubsub.Channel()
    
    for msg := range ch {
        // 當 Redis 收到訊息時，msg.Channel 就是 Topic
        // msg.Payload 就是數據內容（JSON 格式）
        h.broadcastToLocal(msg.Channel, msg.Payload)
    }
}
```

**模式匹配函數：**

```go
// matchPattern 檢查 topic 是否匹配 pattern
// 例如: value:22.5 匹配 value:*
func matchPattern(topic, pattern string) bool {
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
```

### 4. 自動推播機制

當創建或更新遙測數據時，系統會透過 Redis 自動推播給所有服務器的 WebSocket 客戶端：

```go
// 在 HandleCreateTelemetry 中
// 使用 deviceID 發布到具體的 topic (device:{id})
s.Hub.BroadcastToDevice(telemetry.DeviceID, ToTelemetryResponse(*telemetry))

// 在 HandlePatchTelemetry 中
s.Hub.BroadcastToDevice(updatedTelemetry.DeviceID, ToTelemetryResponse(*updatedTelemetry))
```

**推播流程：**

1. Handler 調用 `Hub.BroadcastToDevice()` 或 `Hub.Publish()`
2. Hub 將數據序列化為 JSON 並發布到 Redis
3. Redis 將訊息廣播給所有訂閱該模式的服務器
4. 各服務器的 `listenToRedis()` 收到訊息
5. `broadcastToLocal()` 使用模式匹配找出訂閱的客戶端
6. 訊息發送給所有匹配的 WebSocket 客戶端

**推播的數據格式：**

```json
{
  "id": 1,
  "data_type": "Temperature",
  "value": 25.5,
  "recorded_at": "2024-01-15T10:30:00Z"
}
```

### 5. Server 整合 (`api/server.go`)

```go
type Server struct {
    Router   *chi.Mux
    Store    store.Storage
    TaskChan chan uint
    Hub      *Hub  // WebSocket Hub
}

func NewServer(store store.Storage) *Server {
    // 1. 初始化 Redis 客戶端
    rdb := redis.NewClient(&redis.Options{
        Addr: "redis:6379", // Docker 內部的 service name
    })
    
    s := &Server{
        Router:   chi.NewRouter(),
        Store:    store,
        TaskChan: make(chan uint, 100),
        Hub:      NewHub(rdb),  // 初始化 Hub（傳入 Redis 客戶端）
    }
    // ...
}
```

**路由配置：**

```go
// WebSocket 端點是公開的，不需要認證
s.Router.Get("/ws", s.HandleWS)
```

## 🔄 WebSocket 連線流程

```
1. 客戶端發起 WebSocket 連線請求
   ws://localhost:8080/ws
   ↓
2. 伺服器接收 HTTP 請求
   ↓
3. Upgrader 將 HTTP 連線升級為 WebSocket
   ↓
4. Hub.AddClient() 註冊新連線（預設訂閱 device:*）
   ↓
5. 保持連線，等待訊息
   ↓
6. （可選）客戶端發送訂閱請求訂閱其他 topic
   ↓
7. 當遙測數據變更時
   ↓
8. Handler 調用 Hub.BroadcastToDevice() 或 Hub.Publish()
   ↓
9. Hub 將訊息發布到 Redis（topic: device:{id}）
   ↓
10. Redis 將訊息廣播給所有訂閱 device:* 模式的服務器
   ↓
11. 各服務器的 listenToRedis() 收到訊息
   ↓
12. broadcastToLocal() 使用模式匹配找出訂閱的客戶端
   ↓
13. 訊息發送給所有匹配的 WebSocket 客戶端
   ↓
14. 客戶端收到 JSON 格式的遙測數據
   ↓
15. 客戶端斷線時，Hub.RemoveClient() 清理連線
```

## ⚙️ Redis 配置

### Docker Compose 配置

在 `docker-compose.yml` 中，Redis 服務配置如下：

```yaml
redis:
  image: redis:alpine
  ports:
    - "6379:6379"
```

### 連接配置

API 服務器連接到 Redis 的配置在 `api/server.go` 中：

```go
rdb := redis.NewClient(&redis.Options{
    Addr: "redis:6379", // Docker 內部的 service name
})
```

**注意**：
- 在 Docker Compose 環境中，使用服務名稱 `redis` 作為主機名
- 在本地開發環境中，可能需要改為 `localhost:6379`
- 生產環境建議使用環境變數配置 Redis 地址

### 監聽的模式

系統預設監聽以下 Redis 模式：

- `device:*` - 設備相關訊息
- `value:*` - 數值相關訊息

可以在 `api/ws_hub.go` 的 `NewHub()` 函數中修改或添加模式。

## 📖 使用方式

1. **確保 Redis 服務正在運行**（Docker Compose 會自動啟動）

2. **在瀏覽器安裝 WebSocket 客戶端**
   - Chrome: Simple WebSocket Client 擴充功能
   - 或使用任何 WebSocket 客戶端工具

3. **連線到 WebSocket 端點**
   ```
   ws://localhost:8080/ws
   ```

4. **（可選）發送訂閱請求來訂閱特定 topic**
   ```json
   {
     "action": "subscribe",
     "topic": "value:*"
   }
   ```

5. **使用 API 創建或更新遙測數據**，客戶端會自動收到推播訊息

### 測試推播功能

```bash
# 1. 確保 WebSocket 客戶端已連線

# 2. 創建新的遙測數據（會自動推播）
curl -X POST http://localhost:8080/telemetries \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer secret-token-123" \
  -d '{
    "device_id": 1,
    "data_type": "Temperature",
    "value": 25.5
  }'

# 3. 更新遙測數據（會自動推播）
curl -X PATCH http://localhost:8080/telemetries/1 \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer secret-token-123" \
  -d '{
    "value": 26.0
  }'
```

## ⚠️ 注意事項

1. **跨域設定**：目前 `CheckOrigin` 允許所有來源連線，生產環境應限制允許的來源
2. **錯誤處理**：推播失敗時會自動移除該連線，避免影響其他客戶端
3. **連線管理**：使用 `defer` 確保連線關閉時會自動清理
4. **數據格式**：推播的數據使用 DTO 格式，確保只傳送必要的字段
5. **Redis 連線**：確保 Redis 服務正常運行，否則 WebSocket 推播功能將無法正常工作
6. **模式匹配**：訂閱模式 topic（如 `value:*`）時，會收到所有匹配的訊息；訂閱具體 topic（如 `value:22.5`）時，只會收到該 topic 的訊息
7. **訂閱確認**：客戶端發送 `subscribe` 請求後，會收到確認訊息；發送 `subscribed` 不會收到回應（應使用 `subscribe`）
8. **水平擴展**：使用 Redis Pub/Sub 後，可以輕鬆水平擴展多個 API 服務器實例，所有實例都能收到並分發訊息

## 🔗 相關文檔

- [API 文檔](./API.md)
- [架構設計](./ARCHITECTURE.md)
- [故障排除](./TROUBLESHOOTING.md)

