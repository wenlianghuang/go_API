# GitHub Actions 部署配置指南

本文檔說明如何配置 GitHub Actions 以支援資料庫遷移和部署。

## 📋 工作流說明

`deploy.yaml` 包含三個作業：

1. **push-to-ghcr**：建構並推送 Docker 映像檔到 GitHub Container Registry
2. **run-migrations**：執行資料庫遷移（在部署前）
3. **deploy**：部署應用到生產環境

## 🔐 需要配置的 GitHub Secrets

在 GitHub 倉庫中，進入 **Settings → Secrets and variables → Actions**，新增以下 Secret：

### 必需配置

#### `PRODUCTION_DATABASE_URL`

生產資料庫的連接字串。

**格式**：
```
postgres://username:password@host:port/database?sslmode=require
```

**範例**：
```
postgres://postgres:your-secure-password@db.example.com:5432/iot_db?sslmode=require
```

**⚠️ 重要：不能使用 localhost**

在 GitHub Actions 中，**絕對不能使用 `localhost` 或 `127.0.0.1`**，因為：

- GitHub Actions 在雲端伺服器上執行，不是您的本地機器
- `localhost` 指向運行器本身，而不是您的資料庫伺服器
- 會導致錯誤：`dial tcp [::1]:5432: connect: connection refused`

**正確的 host 格式**：
- ✅ 使用公網 IP：`postgres://user:pass@123.45.67.89:5432/db`
- ✅ 使用域名：`postgres://user:pass@db.example.com:5432/db`
- ✅ 使用雲服務地址：`postgres://user:pass@xxx.rds.amazonaws.com:5432/db`
- ❌ **不能使用**：`localhost`、`127.0.0.1`、`[::1]`

**使用自己的網路 IP**

如果您想使用自己的網路 IP 連接資料庫，需要滿足以下條件：

1. **資料庫伺服器有公網 IP**
   - 不是私有 IP（如 `192.168.x.x`、`10.x.x.x`）
   - 或已配置端口轉發/NAT

2. **防火牆允許外部連接**
   ```bash
   # Ubuntu/Debian
   sudo ufw allow 5432/tcp
   
   # CentOS/RHEL
   sudo firewall-cmd --add-port=5432/tcp --permanent
   sudo firewall-cmd --reload
   ```

3. **PostgreSQL 配置允許遠程連接**
   - `postgresql.conf`：`listen_addresses = '*'`
   - `pg_hba.conf`：允許外部 IP 連接

4. **獲取公網 IP**
   ```bash
   curl ifconfig.me
   # 或
   curl ipinfo.io/ip
   ```

**安全建議**：
- ✅ 使用 `sslmode=require` 或 `sslmode=verify-full` 確保加密連接
- ✅ 使用強密碼
- ✅ 定期輪換密碼
- ✅ 限制資料庫防火牆只允許 GitHub Actions IP 範圍
- ✅ 考慮使用雲資料庫服務（AWS RDS、Google Cloud SQL 等）更安全

### 可選配置（如果使用分離的資料庫參數）

如果不想使用完整的連接字串，可以分別配置：

- `PRODUCTION_DB_HOST`：資料庫主機地址
- `PRODUCTION_DB_PORT`：資料庫端口（預設：5432）
- `PRODUCTION_DB_USER`：資料庫使用者名稱
- `PRODUCTION_DB_PASSWORD`：資料庫密碼
- `PRODUCTION_DB_NAME`：資料庫名稱

如果使用這些分離的參數，需要修改 `deploy.yaml` 中的 `DATABASE_URL` 建構方式。

## 🚀 工作流執行流程

```
Push to main 或 CICD-test 分支
    ↓
1. Build Docker Image
    ↓
2. Push to GHCR
    ↓
3. Run Database Migrations
    ├─ Verify secret exists
    ├─ Install golang-migrate
    ├─ Check current version
    ├─ Execute migrations
    └─ Verify final version
    ↓
4. Deploy Application
    └─ (新增您的部署腳本)
```

**分支條件說明**：

- `push-to-ghcr`：所有配置的分支都會執行
- `run-migrations`：只在 `main` 或 `CICD-test` 分支執行
- `deploy`：只在 `main` 或 `CICD-test` 分支執行

如果推送到其他分支，`run-migrations` 和 `deploy` 會被跳過（這是正常的）。

## 📝 遷移步驟說明

### 1. 驗證 Secret 存在
```bash
# 檢查 PRODUCTION_DATABASE_URL 是否配置
```
- 如果未配置，工作流會立即失敗並顯示錯誤訊息

### 2. 安裝 golang-migrate
```bash
# 下載並安裝 migrate CLI 工具
```

### 3. 檢查當前版本
```bash
migrate -path ./migrations -database "$DATABASE_URL" version
```
- 顯示當前資料庫的遷移版本
- 如果資料庫是新的，可能顯示錯誤（這是正常的，使用 `continue-on-error: true`）

### 4. 執行遷移
```bash
migrate -path ./migrations -database "$DATABASE_URL" up
```
- 執行所有待處理的遷移
- 自動更新 `schema_migrations` 表

### 5. 驗證版本
```bash
migrate -path ./migrations -database "$DATABASE_URL" version
```
- 確認遷移成功執行
- 顯示最終的遷移版本號

## ⚠️ 重要注意事項

### 遷移安全

1. **備份資料庫**：在生產環境執行遷移前，建議先備份資料庫
2. **測試遷移**：在測試環境先驗證遷移腳本
3. **回滾計劃**：準備回滾方案（使用 `.down.sql` 檔案）
4. **監控**：遷移後檢查應用是否正常運行

### 錯誤處理

如果遷移失敗：
1. 檢查 `schema_migrations` 表的狀態（可能顯示 `dirty`）
2. 使用 `migrate force` 命令修復狀態（謹慎使用）
3. 檢查資料庫連接和權限
4. 查看遷移日誌

### 權限要求

資料庫使用者需要以下權限：
- `CREATE TABLE`：建立表
- `ALTER TABLE`：修改表結構
- `CREATE INDEX`：建立索引
- `DROP TABLE`：刪除表（用於回滾）
- 對 `schema_migrations` 表的讀寫權限

## 🔧 自訂部署步驟

在 `deploy.yaml` 的 `deploy` 作業中，新增您的實際部署腳本：

### 範例 1：SSH 部署
```yaml
- name: Deploy via SSH
  uses: appleboy/ssh-action@master
  with:
    host: ${{ secrets.SSH_HOST }}
    username: ${{ secrets.SSH_USER }}
    key: ${{ secrets.SSH_PRIVATE_KEY }}
    script: |
      docker pull ghcr.io/${{ github.repository }}/my-api:${{ github.sha }}
      docker-compose up -d
```

### 範例 2：Kubernetes 部署
```yaml
- name: Deploy to Kubernetes
  uses: azure/k8s-deploy@v4
  with:
    manifests: |
      k8s/deployment.yaml
    images: |
      ghcr.io/${{ github.repository }}/my-api:${{ github.sha }}
```

### 範例 3：Docker Compose 部署
```yaml
- name: Deploy with Docker Compose
  run: |
    ssh user@server << 'EOF'
      cd /path/to/app
      docker-compose pull
      docker-compose up -d
    EOF
```

## 📊 監控和驗證

部署後，建議驗證：

1. **應用健康檢查**：
   ```bash
   curl https://your-api.com/health
   ```

2. **資料庫連接**：
   ```bash
   psql "$DATABASE_URL" -c "SELECT version();"
   ```

3. **遷移版本**：
   ```bash
   migrate -path ./migrations -database "$DATABASE_URL" version
   ```

## 🆘 故障排除

### 問題：遷移步驟被跳過（Skipped）

**可能原因**：
- 推送到未配置的分支（不是 `main` 或 `CICD-test`）
- 作業的 `if` 條件不滿足

**解決方案**：
1. 確認推送到正確的分支
2. 檢查 `deploy.yaml` 中的 `if` 條件
3. 如果需要其他分支也執行，修改條件：
   ```yaml
   if: github.ref == 'refs/heads/main' || github.ref == 'refs/heads/your-branch'
   ```

### 問題：連接錯誤 `dial tcp [::1]:5432: connect: connection refused`

**原因**：
- 使用了 `localhost` 或 `127.0.0.1` 作為 host
- GitHub Actions 運行器無法連接到本地資料庫

**解決方案**：
1. ✅ 使用公網 IP 或域名
2. ✅ 確認資料庫伺服器可從網際網路訪問
3. ✅ 檢查防火牆配置
4. ✅ 驗證連接字串格式正確

**正確格式範例**：
```bash
# ✅ 正確
postgres://user:pass@123.45.67.89:5432/db?sslmode=require
postgres://user:pass@db.example.com:5432/db?sslmode=require

# ❌ 錯誤
postgres://user:pass@localhost:5432/db
postgres://user:pass@127.0.0.1:5432/db
```

### 問題：遷移步驟失敗

**可能原因**：
- 資料庫連接字串錯誤
- 資料庫不可訪問
- 權限不足
- 遷移檔案有語法錯誤

**解決方案**：
1. 檢查 `PRODUCTION_DATABASE_URL` Secret 是否正確
2. 驗證資料庫網路連接
3. 檢查資料庫使用者權限
4. 在本地測試遷移腳本
5. 確認 host 不是 `localhost`

### 問題：遷移顯示 "dirty" 狀態

**解決方案**：
```bash
# 在本地或通過 SSH 連接到伺服器
migrate -path ./migrations -database "$DATABASE_URL" force <version>
```

### 問題：部署後應用無法啟動

**檢查清單**：
- [ ] 資料庫遷移是否成功
- [ ] 環境變數是否正確配置
- [ ] 應用日誌是否有錯誤
- [ ] 資料庫連接是否正常

## 🔒 安全最佳實踐

### 資料庫連接安全

1. **使用 SSL 連接**（必須）
   ```bash
   postgres://user:pass@host:5432/db?sslmode=require
   ```

2. **限制 IP 訪問**
   - 在資料庫防火牆中只允許 GitHub Actions IP 範圍
   - GitHub Actions IP：https://api.github.com/meta

3. **使用強密碼**
   - 至少 16 位，包含大小寫字母、數字、特殊字元

4. **定期輪換密碼**
   - 定期更新資料庫密碼
   - 同步更新 GitHub Secret

### 替代方案比較

| 方案 | 可行性 | 安全性 | 推薦度 | 說明 |
|------|--------|--------|--------|------|
| 自己的公網 IP | ✅ 可以 | ⚠️ 中等 | ⭐⭐⭐ | 需要配置防火牆和 SSL |
| 雲資料庫服務 | ✅ 可以 | ✅ 高 | ⭐⭐⭐⭐⭐ | AWS RDS、Google Cloud SQL 等 |
| VPN/專用網路 | ✅ 可以 | ✅ 高 | ⭐⭐⭐⭐ | Tailscale、WireGuard 等 |
| SSH 隧道 | ✅ 可以 | ✅ 高 | ⭐⭐⭐ | 需要配置 SSH 金鑰 |

**推薦**：使用雲資料庫服務或 VPN，比直接暴露公網 IP 更安全。

## 📚 相關文檔

- [資料庫遷移指南](../backend/docs/MIGRATION.md)
- [GitHub Actions 文檔](https://docs.github.com/en/actions)
- [golang-migrate 文檔](https://github.com/golang-migrate/migrate)
- [PostgreSQL 遠程連接配置](https://www.postgresql.org/docs/current/runtime-config-connection.html)
