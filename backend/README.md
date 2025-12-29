# IoT Device Management API

一個基於 Go 語言開發的 IoT 設備管理系統 API，使用 PostgreSQL 作為資料庫，採用分層架構設計，支援設備管理、遙測數據收集和實時推播等功能。

## 🚀 快速開始

### 使用 Docker Compose（推薦）

最簡單的啟動方式：

```bash
# 啟動所有服務（PostgreSQL、Redis、API）
docker-compose up -d

# 查看服務狀態
docker-compose ps

# 查看日誌
docker-compose logs -f api
```

服務將在以下地址啟動：
- API: `http://localhost:8080`
- PostgreSQL: `localhost:5432`
- Redis: `localhost:6379`

### 本地開發

1. **安裝依賴**
   ```bash
   go mod download
   ```

2. **設置資料庫**
   ```bash
   createdb iot_db
   # 或使用 psql
   psql -U postgres
   CREATE DATABASE iot_db;
   ```

3. **設置環境變數（可選）**
   ```bash
   export DB_DSN="host=localhost user=postgres password=your_password dbname=iot_db port=5432 sslmode=disable TimeZone=Asia/Taipei"
   ```

4. **運行應用程式**
   ```bash
   go run main.go
   ```

   應用程式將在 `http://localhost:8080` 啟動

### 測試 API

```bash
# 測試公開端點
curl http://localhost:8080/

# 創建使用者
curl -X POST http://localhost:8080/users \
  -H "Content-Type: application/json" \
  -d '{"username":"testuser","email":"test@example.com"}'

# 創建設備（需要認證）
curl -X POST http://localhost:8080/devices \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer secret-token-123" \
  -d '{"name":"Sensor 1","type":"Sensor","mac_address":"00:11:22:33:44:55","is_active":true}'
```

## 📚 文檔導航

- 📖 [API 文檔](docs/API.md) - 完整的 API 端點說明和請求範例
- 🐳 [部署指南](docs/DEPLOYMENT.md) - Docker 部署詳細說明（三種方式）
- 🔌 [WebSocket & Redis](docs/WEBSOCKET.md) - 實時推播機制詳細說明
- 🏗 [架構設計](docs/ARCHITECTURE.md) - 系統架構和設計原則
- 🔧 [故障排除](docs/TROUBLESHOOTING.md) - 常見問題解決方案

## 🛠 技術棧

- **語言**: Go 1.25.1
- **Web 框架**: Chi Router
- **ORM**: GORM
- **資料庫**: PostgreSQL
- **訊息佇列**: Redis (Pub/Sub)
- **容器化**: Docker

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
├── docs/                   # 文檔目錄
│   ├── API.md             # API 文檔
│   ├── DEPLOYMENT.md      # 部署指南
│   ├── WEBSOCKET.md       # WebSocket 文檔
│   ├── ARCHITECTURE.md    # 架構設計
│   └── TROUBLESHOOTING.md # 故障排除
├── main.go                 # 應用程式入口
├── main_test.go           # 單元測試
├── Dockerfile             # Docker 構建文件
├── docker-compose.yml     # Docker Compose 配置
├── go.mod                 # Go 模組依賴
└── README.md              # 本文件
```

## 🏗 架構概述

本專案採用**分層架構（Layered Architecture）**：

```
API Layer (api/)      → 請求處理、認證、響應格式化
Domain Layer (model/) → 業務模型定義
Data Access (store/)  → 數據庫操作
Database (PostgreSQL) → 數據持久化
```

**核心特性**：
- 依賴注入：使用接口而非具體實現
- 接口隔離：清晰的接口定義
- 單一職責：每層職責明確

詳細說明請參考 [架構設計文檔](docs/ARCHITECTURE.md)。

## 🔌 WebSocket 實時推播

系統支援透過 WebSocket 和 Redis Pub/Sub 實現跨服務器的實時數據推播：

- 當創建或更新遙測數據時，自動推播給所有訂閱的客戶端
- 支援模式訂閱（如 `device:*`、`value:*`）
- 支援水平擴展多個 API 服務器實例

**快速使用**：

1. 連線到 `ws://localhost:8080/ws`
2. （可選）訂閱特定 topic：
   ```json
   {
     "action": "subscribe",
     "topic": "value:*"
   }
   ```
3. 使用 API 創建遙測數據，客戶端會自動收到推播

詳細說明請參考 [WebSocket 文檔](docs/WEBSOCKET.md)。

## ⚙️ 環境變數配置

| 變數名 | 說明 | 預設值 |
|--------|------|--------|
| `DB_DSN` | PostgreSQL 連接字串 | `host=localhost user=postgres password=password dbname=iot_db port=5432 sslmode=disable TimeZone=Asia/Taipei` |

### 環境變數格式

```
DB_DSN=host=<host> user=<user> password=<password> dbname=<dbname> port=<port> sslmode=<sslmode> TimeZone=<timezone>
```

## 🧪 測試

```bash
# 運行所有測試
go test ./...

# 運行特定測試
go test -v ./main_test.go

# 測試覆蓋率
go test -cover ./...
```

## 📝 注意事項

1. **認證 Token**：目前使用硬編碼的 token `secret-token-123`，生產環境應使用 JWT 或其他安全機制
2. **資料庫遷移**：應用程式啟動時會自動執行 `AutoMigrate`，生產環境建議使用專業的遷移工具
3. **錯誤處理**：目前為簡化版本，生產環境需要更完善的錯誤處理和日誌記錄
4. **連接池**：已設置連接池參數，可根據實際負載調整

## 🔗 相關資源

- [Go 官方文檔](https://go.dev/doc/)
- [Chi Router 文檔](https://github.com/go-chi/chi)
- [GORM 文檔](https://gorm.io/docs/)
- [PostgreSQL 文檔](https://www.postgresql.org/docs/)

## 📄 授權

本專案僅供學習和參考使用。

## 🏛 目前整體 Go Backend 的架構

- [Go Backend 架構圖](https://tinyurl.com/24m5vnr2)
