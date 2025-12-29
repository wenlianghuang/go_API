# 故障排除指南

本文件提供常見問題的解決方案。

## 🔧 資料庫連接失敗

### 症狀

- API 啟動時無法連接到 PostgreSQL
- 日誌顯示連接錯誤

### 解決方案

1. **檢查 PostgreSQL 是否運行**
   ```bash
   docker ps | grep postgres-iot
   # 或
   ps aux | grep postgres
   ```

2. **驗證 `DB_DSN` 環境變數是否正確**
   ```bash
   docker exec iot-api env | grep DB_DSN
   ```

3. **確認資料庫用戶權限**
   - 檢查 PostgreSQL 用戶是否有足夠權限
   - 確認數據庫是否存在

4. **檢查容器是否在同一個 network**
   ```bash
   docker network inspect iot-network
   ```

5. **從 API 容器測試連接**
   ```bash
   docker exec iot-api ping -c 3 postgres-iot
   ```

## 🐳 Docker 容器無法啟動

### 症狀

- `docker-compose up` 失敗
- 容器立即退出

### 解決方案

1. **檢查 Dockerfile 是否正確**
   - 確認 Dockerfile 語法正確
   - 檢查基礎鏡像是否存在

2. **查看容器日誌**
   ```bash
   docker logs <container-name>
   # 或
   docker-compose logs api
   ```

3. **確認端口是否被占用**
   ```bash
   # macOS/Linux
   lsof -i :8080
   lsof -i :5432
   
   # Windows
   netstat -ano | findstr :8080
   ```

4. **檢查容器名稱是否衝突**
   ```bash
   docker ps -a | grep iot-api
   # 如果存在，刪除舊容器
   docker rm <container-name>
   ```

## 🔐 API 返回 401 未授權

### 症狀

- API 請求返回 `401 Unauthorized`
- 無法訪問需要認證的端點

### 解決方案

1. **確認請求 Header 中包含正確的 Token**
   ```bash
   curl -H "Authorization: Bearer secret-token-123" http://localhost:8080/devices
   ```

2. **檢查中間件是否正確配置**
   - 確認路由是否正確掛載了 `AuthMiddleware`
   - 檢查 Token 驗證邏輯

3. **確認 Token 格式正確**
   ```
   Authorization: Bearer secret-token-123
   ```
   注意：`Bearer` 後面有一個空格

## 💾 容器數據庫是空的

### 症狀

- 使用 `go run main.go` 可以訪問數據
- Docker 容器中查詢不到數據

### 原因

- 本地運行連接到 `localhost:5432`（本地 PostgreSQL，有數據）
- Docker 容器連接到 `postgres-iot:5432`（容器內的 PostgreSQL，可能是新的空數據庫）

### 解決方案

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

## 🔍 查詢設備返回 "Device not found"

### 症狀

- GET `/devices/{id}` 返回 404
- 但設備確實存在

### 可能原因

1. 數據庫中確實沒有該設備（ID 不存在）
2. 容器連接到不同的數據庫實例（本地 vs 容器）

### 排查步驟

```bash
# 1. 檢查容器數據庫中的設備
docker exec -it postgres-iot psql -U postgres -d iot_db -c "SELECT id, name FROM devices;"

# 2. 檢查本地數據庫中的設備
psql -U postgres -d iot_db -c "SELECT id, name FROM devices;"

# 3. 先列出所有設備，確認有哪些 ID
curl http://localhost:8080/devices \
  -H "Authorization: Bearer secret-token-123"
```

## 🔴 Redis 連接失敗

### 症狀

- WebSocket 推播功能無法正常工作
- 日誌顯示 Redis 連接錯誤

### 解決方案

1. **檢查 Redis 是否運行**
   ```bash
   docker ps | grep redis
   # 或
   redis-cli ping
   ```

2. **確認 Redis 連接地址**
   - Docker Compose 環境：使用 `redis:6379`
   - 本地開發：使用 `localhost:6379`

3. **檢查網絡連接**
   ```bash
   docker network inspect iot-network
   # 確認 API 和 Redis 容器在同一個網絡
   ```

4. **測試 Redis 連接**
   ```bash
   docker exec iot-api ping -c 3 redis
   # 或
   redis-cli -h localhost -p 6379 ping
   ```

## 🌐 WebSocket 無法連線

### 症狀

- 無法連接到 `ws://localhost:8080/ws`
- 連接立即斷開

### 解決方案

1. **確認 API 服務正在運行**
   ```bash
   curl http://localhost:8080/
   ```

2. **檢查端口是否正確**
   - 確認 API 監聽在 `:8080`
   - 檢查防火牆設置

3. **確認 WebSocket 端點是否正確配置**
   ```bash
   # 檢查路由配置
   grep -r "/ws" api/server.go
   ```

4. **查看 API 日誌**
   ```bash
   docker-compose logs api
   # 或
   docker logs iot-api
   ```

## 📦 Volume 數據管理

### 查看 volume 使用情況

```bash
docker system df -v
```

### 備份 volume 數據

```bash
docker run --rm -v postgres-data:/data -v $(pwd):/backup alpine \
  tar czf /backup/postgres-backup-$(date +%Y%m%d).tar.gz -C /data .
```

### 恢復 volume 數據

```bash
docker run --rm -v postgres-data:/data -v $(pwd):/backup alpine \
  tar xzf /backup/postgres-backup-YYYYMMDD.tar.gz -C /data
```

### 刪除 volume（謹慎操作）

```bash
# 停止相關容器
docker-compose down

# 刪除 volume（會刪除所有數據！）
docker volume rm postgres-data
```

## 🔄 常見錯誤訊息

### "connection refused"

**原因**：服務未啟動或端口錯誤

**解決方案**：
- 檢查服務是否運行
- 確認端口配置正確

### "no such host"

**原因**：容器名稱或網絡配置錯誤

**解決方案**：
- 確認容器名稱正確
- 檢查容器是否在同一個網絡

### "permission denied"

**原因**：文件權限或數據庫權限問題

**解決方案**：
- 檢查文件權限
- 確認數據庫用戶權限

## 📞 獲取更多幫助

如果以上解決方案都無法解決問題，請：

1. **查看完整日誌**
   ```bash
   docker-compose logs
   # 或
   docker logs <container-name>
   ```

2. **檢查系統資源**
   ```bash
   docker stats
   ```

3. **重啟服務**
   ```bash
   docker-compose restart
   # 或
   docker restart <container-name>
   ```

## 🔗 相關文檔

- [部署指南](./DEPLOYMENT.md)
- [API 文檔](./API.md)
- [WebSocket 文檔](./WEBSOCKET.md)
- [架構設計](./ARCHITECTURE.md)

