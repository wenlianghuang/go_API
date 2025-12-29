# Docker 部署指南

本文件詳細說明如何使用 Docker 部署 IoT Device Management API。

## 🐳 方式一：使用 Docker Compose（推薦）

這是最簡單的部署方式，適合開發和生產環境。

### 前置需求

- Docker
- Docker Compose

### 部署步驟

1. **確保 `docker-compose.yml` 文件存在**

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

  redis:
    image: redis:alpine
    ports:
      - "6379:6379"

volumes:
  postgres_data:
```

2. **啟動服務**

```bash
docker-compose up -d
```

3. **驗證部署**

```bash
# 查看服務狀態
docker-compose ps

# 查看日誌
docker-compose logs api

# 測試 API
curl http://localhost:8080/
```

### 從本地數據庫導入數據（可選）

如果你本地已經有數據庫，想要導入到容器中：

```bash
# 1. 確保服務已啟動
docker-compose up -d

# 2. 等待 PostgreSQL 完全啟動
sleep 5

# 3. 執行導入腳本
./import-local-db.sh
```

**導入腳本說明**：
- 腳本會自動檢查本地是否有 `iot_db` 數據庫
- 如果存在，會自動導出並導入到容器數據庫
- 如果 docker DB 已經有了，可以用 Y/N 來確定要不要替代他
- 如果不存在，會跳過導入並提示

**手動導入（如果腳本不可用）**：

```bash
# 從本地數據庫導出並導入到容器
pg_dump -U postgres -h localhost -d iot_db | \
  docker compose exec -T postgres psql -U postgres -d iot_db
```

### 常用 Docker Compose 命令

```bash
# 啟動服務（後台運行）
docker-compose up -d

# 查看服務狀態
docker-compose ps

# 查看日誌
docker-compose logs
docker-compose logs api        # 只看 API 日誌
docker-compose logs -f api     # 實時查看 API 日誌

# 停止服務
docker-compose stop

# 停止並刪除容器
docker-compose down

# 停止並刪除容器和 volumes（會刪除數據！）
docker-compose down -v

# 重新構建並啟動
docker-compose build
docker-compose up -d

# 重啟服務
docker-compose restart api
```

## 🐳 方式二：單獨使用 Docker（使用 Docker Volumes 持久化數據）

適合需要更多控制權的場景。

### 步驟 0：清理舊容器（如果存在）

```bash
# 停止並刪除舊容器（如果存在）
docker stop iot-api postgres-iot 2>/dev/null || true
docker rm iot-api postgres-iot 2>/dev/null || true
```

### 步驟 1：創建 Docker Volume（數據持久化）

```bash
# 創建 volume 用於持久化 PostgreSQL 數據
# 即使容器被刪除，數據也會保留在 volume 中
docker volume create postgres-data
```

**為什麼需要 Volume？**
- 每次創建新的 PostgreSQL 容器時，它都是全新的空數據庫
- 使用 Volume 可以讓數據持久化，即使容器刪除重建，數據也不會丟失

### 步驟 2：創建 Docker Network

```bash
# 創建網絡，讓容器可以互相通信
docker network create iot-network
```

### 步驟 3：啟動 PostgreSQL 容器（使用 Volume）

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

### 步驟 4：啟動 Redis 容器

```bash
docker run -d \
  --name redis-iot \
  --network iot-network \
  -p 6379:6379 \
  redis:alpine
```

### 步驟 5：從本地數據庫導入數據（可選）

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

### 步驟 6：構建 API 鏡像

```bash
# 進入 backend 目錄
cd /Users/matthuang/Desktop/go_API/backend

# 構建 Docker 鏡像
docker build -t iot-api:latest .
```

### 步驟 7：啟動 API 容器

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

### 步驟 8：驗證部署

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

## 🐳 方式三：只使用 API 容器（連接到本地 PostgreSQL）

如果你本地已經有 PostgreSQL 在運行，只想將 API 容器化，可以跳過 PostgreSQL 容器的創建步驟。

### 前置條件

- 本地已安裝並運行 PostgreSQL
- 本地已有 `iot_db` 數據庫（或創建一個新的）
- 本地已安裝並運行 Redis（或使用 Docker 運行 Redis）

### 步驟 1：啟動 Redis（如果本地沒有）

```bash
docker run -d \
  --name redis-iot \
  -p 6379:6379 \
  redis:alpine
```

### 步驟 2：構建 API 鏡像

```bash
# 進入 backend 目錄
cd /Users/matthuang/Desktop/go_API/backend

# 構建 Docker 鏡像
docker build -t iot-api:latest .
```

### 步驟 3：啟動 API 容器（連接到本地 PostgreSQL）

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

### 步驟 4：驗證部署

```bash
# 1. 檢查容器狀態
docker ps

# 2. 查看 API 容器日誌（確認連接成功）
docker logs iot-api

# 3. 測試 API
curl http://localhost:8080/
```

### 重要說明

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

4. **Redis 配置**：
   - 如果使用 Docker 運行 Redis，API 容器需要連接到 `host.docker.internal:6379`
   - 需要修改 `api/server.go` 中的 Redis 連接地址

5. **優點**：
   - 不需要創建和管理 PostgreSQL 容器
   - 直接使用本地已有的數據庫和數據
   - 適合開發環境，數據管理更方便

6. **缺點**：
   - 依賴本地環境，不夠隔離
   - 不同開發者環境可能不一致
   - 不適合生產環境部署

## 💾 數據持久化說明

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

## 📦 Dockerfile 說明

本專案使用**多階段構建（Multi-stage Build）**：

1. **Builder 階段**：使用 `golang:1.25-alpine` 編譯應用程式
2. **Runner 階段**：使用輕量級的 `alpine:latest` 運行應用程式

這樣可以大幅減少最終映像的大小。

## 🔗 相關文檔

- [故障排除](../docs/TROUBLESHOOTING.md)
- [API 文檔](../docs/API.md)
- [架構設計](../docs/ARCHITECTURE.md)

