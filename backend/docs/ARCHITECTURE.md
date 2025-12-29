# 架構設計

本文件詳細說明 IoT Device Management API 的架構設計和核心設計原則。

## 🏗 分層架構

本專案採用**分層架構（Layered Architecture）**，將業務邏輯與數據訪問分離：

```
┌─────────────────────────────────────┐
│         API Layer (api/)             │
│  - Handlers (請求處理)               │
│  - Middleware (認證、日誌)           │
│  - DTO (數據傳輸對象)                │
└──────────────┬──────────────────────┘
               │
┌──────────────▼──────────────────────┐
│      Domain Layer (model/)           │
│  - Device (設備模型)                 │
│  - Telemetry (遙測數據模型)          │
└──────────────┬──────────────────────┘
               │
┌──────────────▼──────────────────────┐
│    Data Access Layer (store/)        │
│  - Storage Interface (接口)          │
│  - GormStore (GORM 實現)             │
└──────────────┬──────────────────────┘
               │
┌──────────────▼──────────────────────┐
│         Database (PostgreSQL)        │
└─────────────────────────────────────┘
```

## 🎯 核心設計原則

### 1. 依賴注入（Dependency Injection）

`Server` 依賴 `Storage` 接口，而非具體實現：

```go
type Server struct {
    Store store.Storage // 依賴接口，而非具體實現
    // ...
}
```

**優點**：
- 可以輕鬆切換不同的數據存儲方案（PostgreSQL、MySQL、Memory 等）
- 便於單元測試（可以使用 Mock 實現）
- 降低耦合度

### 2. 接口隔離（Interface Segregation）

`Storage` 接口定義了所有數據操作契約：

```go
type Storage interface {
    Create(user store.User) error
    List() ([]store.User, error)
    Get(id string) (store.User, error)
    CreateDevice(device *model.Device) error
    // ... 其他方法
}
```

**優點**：
- 實現類只需滿足接口即可
- 接口定義清晰，易於理解
- 符合 SOLID 原則

### 3. 單一職責（Single Responsibility）

每層都有明確的職責：

- **API Layer**：負責請求處理、參數驗證、響應格式化
- **Domain Layer**：定義業務模型
- **Data Access Layer**：負責數據庫操作
- **Database**：數據持久化

**優點**：
- 代碼組織清晰
- 易於維護和擴展
- 符合單一職責原則

## 🔄 代碼流程說明

### 請求處理流程

```
1. HTTP 請求進入
   ↓
2. Chi Router 匹配路由
   ↓
3. Middleware 處理（Logger, Recoverer, AuthMiddleware）
   ↓
4. Handler 處理請求
   ├── 解析請求參數
   ├── 驗證輸入數據
   ├── 轉換 DTO → Domain Model
   ├── 調用 Store 層
   └── 返回響應（DTO）
   ↓
5. Store 層執行數據庫操作
   ↓
6. 返回結果給 Handler
   ↓
7. Handler 格式化響應
   ↓
8. 返回 HTTP 響應
```

### 範例：創建設備流程

```go
// 1. 請求進入 Handler
HandleCreateDevice(w, r)
   ↓
// 2. 解析 JSON 請求體
json.NewDecoder(r.Body).Decode(&req)
   ↓
// 3. 驗證必填字段
if req.Name == "" || req.MacAddress == "" { ... }
   ↓
// 4. 轉換為 Domain Model
device := &model.Device{...}
   ↓
// 5. 調用 Store 層
s.Store.CreateDevice(device)
   ↓
// 6. GormStore 執行數據庫操作
db.Create(device)
   ↓
// 7. 返回結果
WriteJSON(w, http.StatusCreated, device)
```

### 認證流程

```
1. 請求進入 AuthMiddleware
   ↓
2. 檢查 Authorization Header
   ↓
3. 解析 Bearer Token
   ↓
4. 驗證 Token（目前為簡單驗證）
   ↓
5. 將 UserID 注入 Context
   ↓
6. 傳遞給下一個 Handler
   ↓
7. Handler 可從 Context 取得 UserID
```

## 📁 項目結構

```
backend/
├── api/                    # API 層
│   ├── handlers.go        # 請求處理器
│   ├── server.go          # 路由配置
│   ├── middleware.go      # 中間件（認證等）
│   ├── response.go        # 響應工具函數
│   ├── dto.go             # 數據傳輸對象
│   └── ws_hub.go          # WebSocket Hub（管理連線和推播）
├── model/                  # 領域模型
│   └── device.go          # Device 和 Telemetry 模型
├── store/                  # 數據訪問層
│   ├── db.go              # Storage 接口定義
│   └── gorm_store.go      # GORM 實現
├── main.go                 # 應用程式入口
├── main_test.go           # 單元測試
├── Dockerfile             # Docker 構建文件
├── go.mod                 # Go 模組依賴
└── README.md              # 本文件
```

## 🔌 WebSocket 架構

本專案使用 **Hub 模式** 結合 **Redis Pub/Sub** 來實現跨服務器的 WebSocket 廣播。

詳細說明請參考 [WebSocket 文檔](./WEBSOCKET.md)。

## 🗄️ 數據模型

### Device（設備）

```go
type Device struct {
    ID        uint      `gorm:"primaryKey"`
    Name      string    `gorm:"not null"`
    Type      string
    MacAddress string   `gorm:"uniqueIndex;not null"`
    IsActive  bool      `gorm:"default:true"`
    CreatedAt time.Time
    UpdatedAt time.Time
    Telemetries []Telemetry `gorm:"foreignKey:DeviceID"`
}
```

### Telemetry（遙測數據）

```go
type Telemetry struct {
    ID         uint      `gorm:"primaryKey"`
    DeviceID   uint      `gorm:"not null;index"`
    DataType   string    `gorm:"not null"`
    Value      float64   `gorm:"not null"`
    RecordedAt time.Time `gorm:"not null"`
    CreatedAt  time.Time
    Device     Device    `gorm:"foreignKey:DeviceID"`
}
```

## 🔐 認證機制

目前使用簡單的 Bearer Token 認證：

```go
Authorization: Bearer secret-token-123
```

**注意**：生產環境應使用 JWT 或其他更安全的認證機制。

## 📊 DTO（數據傳輸對象）

使用 DTO 來控制 API 響應的格式，只返回必要的字段：

```go
type DeviceResponse struct {
    ID        uint      `json:"id"`
    Name      string    `json:"name"`
    Type      string    `json:"type"`
    MacAddress string   `json:"mac_address"`
    IsActive  bool      `json:"is_active"`
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
}
```

**優點**：
- 隱藏內部實現細節
- 控制 API 版本
- 減少數據傳輸量

## 🔗 相關文檔

- [API 文檔](./API.md)
- [WebSocket 文檔](./WEBSOCKET.md)
- [部署指南](./DEPLOYMENT.md)
- [故障排除](./TROUBLESHOOTING.md)

