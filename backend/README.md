# IoT Device Management API

一個基於 Go 語言開發的 IoT 設備管理系統 API，使用 PostgreSQL 作為資料庫，採用分層架構設計，支援設備管理、遙測數據收集等功能。

## 📋 目錄

- [技術棧](#技術棧)
- [項目結構](#項目結構)
- [架構設計](#架構設計)
- [API 端點](#api-端點)
- [代碼流程說明](#代碼流程說明)
- [環境變數配置](#環境變數配置)
- [本地開發](#本地開發)
- [Docker 部署](#docker-部署)
- [測試](#測試)

## 🛠 技術棧

- **語言**: Go 1.25.1
- **Web 框架**: Chi Router
- **ORM**: GORM
- **資料庫**: PostgreSQL
- **容器化**: Docker

## 📁 項目結構

```
backend/
├── api/                    # API 層
│   ├── handlers.go        # 請求處理器
│   ├── server.go          # 路由配置
│   ├── middleware.go      # 中間件（認證等）
│   ├── response.go        # 響應工具函數
│   └── dto.go             # 數據傳輸對象
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

## 🏗 架構設計

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

### 核心設計原則

1. **依賴注入（Dependency Injection）**
   - `Server` 依賴 `Storage` 接口，而非具體實現
   - 可以輕鬆切換不同的數據存儲方案（PostgreSQL、MySQL、Memory 等）

2. **接口隔離（Interface Segregation）**
   - `Storage` 接口定義了所有數據操作契約
   - 實現類只需滿足接口即可

3. **單一職責（Single Responsibility）**
   - 每層都有明確的職責
   - Handler 負責請求處理，Store 負責數據訪問

## 📡 API 端點

### 公開端點（無需認證）

| 方法 | 路徑 | 說明 |
|------|------|------|
| GET | `/` | API 歡迎訊息 |
| POST | `/users` | 創建使用者 |

### 私有端點（需要認證）

所有私有端點都需要在 Header 中提供：
```
Authorization: Bearer secret-token-123
```

#### 使用者相關

| 方法 | 路徑 | 說明 |
|------|------|------|
| GET | `/users` | 取得所有使用者列表 |
| GET | `/users/{id}` | 取得單一使用者 |
| GET | `/me` | 取得當前登入者資訊 |

#### 設備相關

| 方法 | 路徑 | 說明 |
|------|------|------|
| POST | `/devices` | 創建設備 |
| GET | `/devices` | 取得所有設備列表 |
| GET | `/devices/{id}` | 取得單一設備（包含遙測數據） |
| PUT | `/devices/{id}` | 完整更新設備（所有字段必須提供） |
| PATCH | `/devices/{id}` | 部分更新設備（只需提供要更新的字段） |
| DELETE | `/devices/{id}` | 刪除設備及其所有遙測數據 |

#### 遙測數據相關

| 方法 | 路徑 | 說明 |
|------|------|------|
| POST | `/telemetries` | 創建遙測數據 |

### API 請求範例

#### 創建設備
```bash
POST /devices
Authorization: Bearer secret-token-123
Content-Type: application/json

{
  "name": "Temperature Sensor 1",
  "type": "Sensor",
  "mac_address": "00:11:22:33:44:55",
  "is_active": true
}
```

#### 部分更新設備（PATCH）
```bash
PATCH /devices/1
Authorization: Bearer secret-token-123
Content-Type: application/json

{
  "name": "Updated Sensor Name"
}
```

#### 完整更新設備（PUT）
```bash
PUT /devices/1
Authorization: Bearer secret-token-123
Content-Type: application/json

{
  "name": "Temperature Sensor 1",
  "type": "Sensor",
  "mac_address": "00:11:22:33:44:55",
  "is_active": false
}
```

#### 創建遙測數據
```bash
POST /telemetries
Authorization: Bearer secret-token-123
Content-Type: application/json

{
  "device_id": 1,
  "data_type": "Temperature",
  "value": 25.5,
  "recorded_at": "2024-01-15T10:30:00Z"
}
```

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

## ⚙️ 環境變數配置

| 變數名 | 說明 | 預設值 |
|--------|------|--------|
| `DB_DSN` | PostgreSQL 連接字串 | `host=localhost user=postgres password=password dbname=iot_db port=5432 sslmode=disable TimeZone=Asia/Taipei` |

### 環境變數格式

```
DB_DSN=host=<host> user=<user> password=<password> dbname=<dbname> port=<port> sslmode=<sslmode> TimeZone=<timezone>
```

## 💻 本地開發

### 前置需求

- Go 1.25.1 或更高版本
- PostgreSQL 12 或更高版本

### 安裝步驟

1. **克隆專案**
   ```bash
   git clone <repository-url>
   cd go_API/backend
   ```

2. **安裝依賴**
   ```bash
   go mod download
   ```

3. **設置資料庫**
   ```bash
   # 創建資料庫
   createdb iot_db
   
   # 或使用 psql
   psql -U postgres
   CREATE DATABASE iot_db;
   ```

4. **設置環境變數（可選）**
   ```bash
   export DB_DSN="host=localhost user=postgres password=your_password dbname=iot_db port=5432 sslmode=disable TimeZone=Asia/Taipei"
   ```

5. **運行應用程式**
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

## 🐳 Docker 部署

### 方式一：使用 Docker Compose（推薦）

創建 `docker-compose.yml` 文件：

```yaml
version: '3.8'

services:
  postgres:
    image: postgres:15-alpine
    environment:
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: password
      POSTGRES_DB: iot_db
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U postgres"]
      interval: 10s
      timeout: 5s
      retries: 5

  api:
    build:
      context: .
      dockerfile: Dockerfile
    ports:
      - "8080:8080"
    environment:
      DB_DSN: "host=postgres user=postgres password=password dbname=iot_db port=5432 sslmode=disable TimeZone=Asia/Taipei"
    depends_on:
      postgres:
        condition: service_healthy
    restart: unless-stopped

volumes:
  postgres_data:
```

執行：
```bash
docker-compose up -d
```

### 方式二：單獨使用 Docker（使用 Docker Volumes 持久化數據）

#### 步驟 0：清理舊容器（如果存在）

```bash
# 停止並刪除舊容器（如果存在）
docker stop iot-api postgres-iot 2>/dev/null || true
docker rm iot-api postgres-iot 2>/dev/null || true
```

#### 步驟 1：創建 Docker Volume（數據持久化）

```bash
# 創建 volume 用於持久化 PostgreSQL 數據
# 即使容器被刪除，數據也會保留在 volume 中
docker volume create postgres-data
```

**為什麼需要 Volume？**
- 每次創建新的 PostgreSQL 容器時，它都是全新的空數據庫
- 使用 Volume 可以讓數據持久化，即使容器刪除重建，數據也不會丟失

#### 步驟 2：創建 Docker Network

```bash
# 創建網絡，讓容器可以互相通信
docker network create iot-network
```

#### 步驟 3：啟動 PostgreSQL 容器（使用 Volume）

```bash
# 啟動 PostgreSQL 容器，掛載 volume 以持久化數據
docker run -d \
  --name postgres-iot \
  --network iot-network \
  -e POSTGRES_USER=postgres \
  -e POSTGRES_PASSWORD=password \
  -e POSTGRES_DB=iot_db \
  -p 5432:5432 \
  -v postgres-data:/var/lib/postgresql/data \
  postgres:15-alpine
```

等待數據庫啟動（約 5-10 秒）：

```bash
# 等待數據庫完全啟動
echo "等待 PostgreSQL 啟動..."
sleep 10

# 驗證數據庫是否啟動成功
docker exec postgres-iot pg_isready -U postgres
```

#### 步驟 4：從本地數據庫導入數據（可選）

如果你本地已有數據庫，想要導入到容器中：

```bash
# 方法 A：使用 pg_dump 和管道（推薦）
pg_dump -U postgres -h localhost -d iot_db 2>/dev/null | \
  docker exec -i postgres-iot psql -U postgres -d iot_db

# 或者方法 B：先導出再導入
# pg_dump -U postgres -h localhost -d iot_db > backup.sql
# docker exec -i postgres-iot psql -U postgres -d iot_db < backup.sql
```

**注意**：如果本地沒有數據庫或不需要導入，可以跳過此步驟。後續可以通過 API 創建設備。

#### 步驟 5：構建 API 鏡像

```bash
# 進入 backend 目錄
cd /Users/matthuang/Desktop/go_API/backend

# 構建 Docker 鏡像
docker build -t iot-api:latest .
```

#### 步驟 6：啟動 API 容器

**選項 A：連接到容器內的 PostgreSQL（推薦用於生產環境）**

```bash
# 啟動 API 容器，連接到 PostgreSQL 容器
docker run -d \
  --name iot-api \
  --network iot-network \
  -p 8080:8080 \
  -e DB_DSN="host=postgres-iot user=postgres password=password dbname=iot_db port=5432 sslmode=disable TimeZone=Asia/Taipei" \
  iot-api:latest
```

**重要說明**：
- `host=postgres-iot`：使用容器名稱作為主機名（在 Docker Network 中）
- 容器必須在同一個 network 中才能通過名稱互相訪問

**選項 B：連接到本地 PostgreSQL（適合開發環境）**

如果你本地已經有 PostgreSQL 在運行，想讓 API 容器連接到本地數據庫：

**macOS / Windows：**

```bash
# macOS 和 Windows 可以使用 host.docker.internal 直接訪問主機
docker run -d \
  --name iot-api \
  -p 8080:8080 \
  -e DB_DSN="host=host.docker.internal user=postgres password=password dbname=iot_db port=5432 sslmode=disable TimeZone=Asia/Taipei" \
  iot-api:latest
```

**Linux：**

```bash
# Linux 需要使用 --add-host 或 --network host
# 方法 1：使用 --add-host（推薦）
docker run -d \
  --name iot-api \
  --add-host=host.docker.internal:host-gateway \
  -p 8080:8080 \
  -e DB_DSN="host=host.docker.internal user=postgres password=password dbname=iot_db port=5432 sslmode=disable TimeZone=Asia/Taipei" \
  iot-api:latest

# 方法 2：使用 --network host（容器直接使用主機網絡）
docker run -d \
  --name iot-api \
  --network host \
  -e DB_DSN="host=localhost user=postgres password=password dbname=iot_db port=5432 sslmode=disable TimeZone=Asia/Taipei" \
  iot-api:latest
```

**注意事項**：
- 使用本地 PostgreSQL 時，不需要創建 PostgreSQL 容器和 Docker Network
- 確保本地 PostgreSQL 允許來自 Docker 容器的連接（檢查 `pg_hba.conf`）
- macOS/Windows 的 `host.docker.internal` 是 Docker Desktop 提供的特殊主機名
- Linux 需要手動添加 `--add-host` 或使用 `--network host`

#### 步驟 7：驗證部署

```bash
# 1. 檢查容器狀態
docker ps

# 2. 查看 API 容器日誌（確認連接成功）
docker logs iot-api

# 3. 查看 PostgreSQL 容器日誌
docker logs postgres-iot

# 4. 測試 API（如果數據庫是空的，先創建一個設備）
curl http://localhost:8080/

# 5. 檢查數據庫中的數據
docker exec -it postgres-iot psql -U postgres -d iot_db -c "SELECT COUNT(*) FROM devices;"
```

### 完整部署腳本（一鍵執行）

你也可以創建一個 `deploy.sh` 腳本，一鍵執行所有步驟：

```bash
#!/bin/bash

set -e  # 遇到錯誤立即退出

echo "🧹 清理舊容器..."
docker stop iot-api postgres-iot 2>/dev/null || true
docker rm iot-api postgres-iot 2>/dev/null || true

echo "📦 創建 Docker Volume..."
docker volume create postgres-data 2>/dev/null || echo "Volume 已存在"

echo "🌐 創建 Docker Network..."
docker network create iot-network 2>/dev/null || echo "Network 已存在"

echo "🐘 啟動 PostgreSQL 容器..."
docker run -d \
  --name postgres-iot \
  --network iot-network \
  -e POSTGRES_USER=postgres \
  -e POSTGRES_PASSWORD=password \
  -e POSTGRES_DB=iot_db \
  -p 5432:5432 \
  -v postgres-data:/var/lib/postgresql/data \
  postgres:15-alpine

echo "⏳ 等待 PostgreSQL 啟動..."
sleep 10
docker exec postgres-iot pg_isready -U postgres

echo "📥 導入本地數據（如果存在）..."
if pg_dump -U postgres -h localhost -d iot_db >/dev/null 2>&1; then
  echo "發現本地數據庫，正在導入..."
  pg_dump -U postgres -h localhost -d iot_db | \
    docker exec -i postgres-iot psql -U postgres -d iot_db
  echo "✅ 數據導入完成"
else
  echo "⚠️  未發現本地數據庫，跳過導入"
fi

echo "🔨 構建 API 鏡像..."
cd /Users/matthuang/Desktop/go_API/backend
docker build -t iot-api:latest .

echo "🚀 啟動 API 容器..."
docker run -d \
  --name iot-api \
  --network iot-network \
  -p 8080:8080 \
  -e DB_DSN="host=postgres-iot user=postgres password=password dbname=iot_db port=5432 sslmode=disable TimeZone=Asia/Taipei" \
  iot-api:latest

echo "⏳ 等待 API 啟動..."
sleep 5

echo "✅ 部署完成！"
echo ""
echo "📊 檢查容器狀態："
docker ps --filter "name=iot-api" --filter "name=postgres-iot"

echo ""
echo "🧪 測試 API："
curl -s http://localhost:8080/ || echo "API 尚未就緒，請稍候再試"

echo ""
echo "📝 查看日誌："
echo "  API 日誌: docker logs iot-api"
echo "  PostgreSQL 日誌: docker logs postgres-iot"
```

保存為 `deploy.sh`，然後執行：

```bash
chmod +x deploy.sh
./deploy.sh
```


### 方式三：只使用 API 容器（連接到本地 PostgreSQL）

如果你本地已經有 PostgreSQL 在運行，只想將 API 容器化，可以跳過 PostgreSQL 容器的創建步驟。

#### 前置條件

- 本地已安裝並運行 PostgreSQL
- 本地已有 `iot_db` 數據庫（或創建一個新的）

#### 步驟 1：構建 API 鏡像

```bash
# 進入 backend 目錄
cd /Users/matthuang/Desktop/go_API/backend

# 構建 Docker 鏡像
docker build -t iot-api:latest .
```

#### 步驟 2：啟動 API 容器（連接到本地 PostgreSQL）

**macOS / Windows：**

```bash
# macOS 和 Windows 可以使用 host.docker.internal 直接訪問主機
docker run -d \
  --name iot-api \
  -p 8080:8080 \
  -e DB_DSN="host=host.docker.internal user=postgres password=password dbname=iot_db port=5432 sslmode=disable TimeZone=Asia/Taipei" \
  iot-api:latest
```

**Linux：**

```bash
# 方法 1：使用 --add-host（推薦）
docker run -d \
  --name iot-api \
  --add-host=host.docker.internal:host-gateway \
  -p 8080:8080 \
  -e DB_DSN="host=host.docker.internal user=postgres password=password dbname=iot_db port=5432 sslmode=disable TimeZone=Asia/Taipei" \
  iot-api:latest

# 方法 2：使用 --network host（容器直接使用主機網絡）
docker run -d \
  --name iot-api \
  --network host \
  -e DB_DSN="host=localhost user=postgres password=password dbname=iot_db port=5432 sslmode=disable TimeZone=Asia/Taipei" \
  iot-api:latest
```

#### 步驟 3：驗證部署

```bash
# 1. 檢查容器狀態
docker ps

# 2. 查看 API 容器日誌（確認連接成功）
docker logs iot-api

# 3. 測試 API
curl http://localhost:8080/
```

#### 重要說明

1. **macOS/Windows**：
   - `host.docker.internal` 是 Docker Desktop 提供的特殊主機名
   - 可以直接訪問主機上的服務（如本地 PostgreSQL）

2. **Linux**：
   - 方法 1（`--add-host`）：手動添加 `host.docker.internal` 映射到主機網關
   - 方法 2（`--network host`）：容器直接使用主機網絡，可以直接訪問 `localhost`
   - 使用 `--network host` 時，`-p 8080:8080` 端口映射會被忽略

3. **PostgreSQL 配置**：
   - 確保本地 PostgreSQL 允許來自 Docker 容器的連接
   - 檢查 `pg_hba.conf` 文件，確保允許來自 Docker 網絡的連接
   - 如果使用 `--network host`，PostgreSQL 需要允許來自 `localhost` 的連接

4. **優點**：
   - 不需要創建和管理 PostgreSQL 容器
   - 直接使用本地已有的數據庫和數據
   - 適合開發環境，數據管理更方便

5. **缺點**：
   - 依賴本地環境，不夠隔離
   - 不同開發者環境可能不一致
   - 不適合生產環境部署

### 數據持久化說明

使用 Docker Volume 的好處：

1. **數據持久化**：即使刪除容器，數據也會保留在 volume 中
2. **容器重建**：如果需要重建容器但保留數據，只需刪除容器，volume 會保留數據
3. **備份方便**：可以輕鬆備份 volume 數據

查看和管理 volume：

```bash
# 查看所有 volumes
docker volume ls

# 查看 volume 詳情
docker volume inspect postgres-data

# 備份 volume（可選）
docker run --rm -v postgres-data:/data -v $(pwd):/backup alpine \
  tar czf /backup/postgres-backup.tar.gz -C /data .

# 刪除 volume（謹慎操作，會刪除所有數據）
docker volume rm postgres-data
```

### Dockerfile 說明

本專案使用**多階段構建（Multi-stage Build）**：

1. **Builder 階段**：使用 `golang:1.25-alpine` 編譯應用程式
2. **Runner 階段**：使用輕量級的 `alpine:latest` 運行應用程式

這樣可以大幅減少最終映像的大小。

## 🧪 測試

### 運行單元測試

```bash
go test ./...
```

### 運行特定測試

```bash
go test -v ./main_test.go
```

### 測試覆蓋率

```bash
go test -cover ./...
```

## 📝 注意事項

1. **認證 Token**：目前使用硬編碼的 token `secret-token-123`，生產環境應使用 JWT 或其他安全機制
2. **資料庫遷移**：應用程式啟動時會自動執行 `AutoMigrate`，生產環境建議使用專業的遷移工具
3. **錯誤處理**：目前為簡化版本，生產環境需要更完善的錯誤處理和日誌記錄
4. **連接池**：已設置連接池參數，可根據實際負載調整

## 🔧 故障排除

### 資料庫連接失敗

- 檢查 PostgreSQL 是否運行：`docker ps | grep postgres-iot`
- 驗證 `DB_DSN` 環境變數是否正確：`docker exec iot-api env | grep DB_DSN`
- 確認資料庫用戶權限
- 檢查容器是否在同一個 network：`docker network inspect iot-network`
- 從 API 容器測試連接：`docker exec iot-api ping -c 3 postgres-iot`

### Docker 容器無法啟動

- 檢查 Dockerfile 是否正確
- 查看容器日誌：`docker logs <container-name>`
- 確認端口是否被占用：`lsof -i :8080` 或 `lsof -i :5432`
- 檢查容器名稱是否衝突：`docker ps -a | grep iot-api`

### API 返回 401 未授權

- 確認請求 Header 中包含 `Authorization: Bearer secret-token-123`
- 檢查中間件是否正確配置

### 容器數據庫是空的

**問題**：使用 `go run main.go` 可以訪問數據，但 Docker 容器中查詢不到數據

**原因**：
- 本地運行連接到 `localhost:5432`（本地 PostgreSQL，有數據）
- Docker 容器連接到 `postgres-iot:5432`（容器內的 PostgreSQL，可能是新的空數據庫）

**解決方案**：
1. **使用 Docker Volume 持久化數據**（推薦）
   ```bash
   # 確保使用 volume 掛載
   docker run -d ... -v postgres-data:/var/lib/postgresql/data ...
   ```

2. **從本地數據庫導入數據**
   ```bash
   pg_dump -U postgres -h localhost -d iot_db | \
     docker exec -i postgres-iot psql -U postgres -d iot_db
   ```

3. **讓容器連接到本地數據庫**（開發環境）
   ```bash
   # Mac/Windows 使用 host.docker.internal
   docker run -d ... \
     -e DB_DSN="host=host.docker.internal user=postgres ..." ...
   
   # Linux 需要添加 --add-host
   docker run -d ... --add-host=host.docker.internal:host-gateway ...
   ```

### 查詢設備返回 "Device not found"

**可能原因**：
1. 數據庫中確實沒有該設備（ID 不存在）
2. 容器連接到不同的數據庫實例（本地 vs 容器）

**排查步驟**：
```bash
# 1. 檢查容器數據庫中的設備
docker exec -it postgres-iot psql -U postgres -d iot_db -c "SELECT id, name FROM devices;"

# 2. 檢查本地數據庫中的設備
psql -U postgres -d iot_db -c "SELECT id, name FROM devices;"

# 3. 先列出所有設備，確認有哪些 ID
curl http://localhost:8080/devices \
  -H "Authorization: Bearer secret-token-123"
```

### Volume 數據管理

```bash
# 查看 volume 使用情況
docker system df -v

# 備份 volume 數據
docker run --rm -v postgres-data:/data -v $(pwd):/backup alpine \
  tar czf /backup/postgres-backup-$(date +%Y%m%d).tar.gz -C /data .

# 恢復 volume 數據
docker run --rm -v postgres-data:/data -v $(pwd):/backup alpine \
  tar xzf /backup/postgres-backup-YYYYMMDD.tar.gz -C /data
```

## 📚 相關資源

- [Go 官方文檔](https://go.dev/doc/)
- [Chi Router 文檔](https://github.com/go-chi/chi)
- [GORM 文檔](https://gorm.io/docs/)
- [PostgreSQL 文檔](https://www.postgresql.org/docs/)

## 📄 授權

本專案僅供學習和參考使用。

