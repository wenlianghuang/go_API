# 資料庫遷移指南

本專案使用 [golang-migrate](https://github.com/golang-migrate/migrate) 進行資料庫版本管理，替代 GORM AutoMigrate。

## 📁 遷移文件說明

```
backend/
├── migrations/              # ✅ 必需：遷移 SQL 腳本（源代碼）
│   ├── 000001_*.sql        #   這些文件會被複製到 Docker 容器
│   ├── 000002_*.sql        #   必須提交到 Git（版本控制）
│   └── ...
└── scripts/
    ├── docker-migrate.sh   # ✅ 推薦：Docker 環境使用
    └── migrate.sh          # ⚠️ 可選：本地開發或 CI 使用
```

**重要說明**：
- **`migrations/` 目錄**：✅ **絕對不能刪除**，這些是遷移腳本的源代碼，Docker 構建時會複製到容器內
- **`docker-migrate.sh`**：✅ **推薦使用**，在 Docker 容器內執行遷移，確保連接到正確的資料庫
- **`migrate.sh`**：⚠️ **可選**，僅在本地有 PostgreSQL 實例或 CI/CD 環境需要時使用

## 🐳 Docker 環境（推薦）

### 自動遷移

遷移會在容器啟動時自動執行：
```bash
# 清除現有資料（如果需要）
docker-compose down -v

# 啟動服務（會自動執行遷移）
docker-compose up --build
```

### 手動執行遷移（推薦使用）

使用 `docker-migrate.sh` 腳本在 Docker 容器內執行遷移：

```bash
# 查看當前版本（會顯示 Docker 資料庫狀態）
./scripts/docker-migrate.sh version

# 執行向上遷移
./scripts/docker-migrate.sh up

# 執行向下遷移（回滾）
./scripts/docker-migrate.sh down

# 強制設定版本
./scripts/docker-migrate.sh force 3
```

**為什麼推薦使用 `docker-migrate.sh`？**
- ✅ 確保連接到 Docker 容器內的資料庫
- ✅ 避免本地 PostgreSQL 實例的干擾
- ✅ 使用與 API 容器相同的環境
- ✅ 自動顯示 Docker 資料庫的 `schema_migrations` 狀態

## 💻 本地開發（可選）

如果您在本地運行 PostgreSQL（非 Docker），可以使用 `migrate.sh`：

1. **安裝 migrate CLI**：
   ```bash
   # macOS
   brew install golang-migrate
   
   # 或使用 Go install
   go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
   ```

2. **執行遷移**：
   ```bash
   cd backend
   chmod +x scripts/migrate.sh
   ./scripts/migrate.sh up
   ```

3. **查看當前版本**（會同時顯示本地和 Docker 狀態）：
   ```bash
   ./scripts/migrate.sh version
   ```

**注意**：如果您的團隊只在 Docker 環境中工作，可以忽略此部分。

## 📝 建立新的遷移

無論使用哪種方式，建立新遷移的方法相同：

```bash
# 使用本地 migrate 工具建立新遷移
./scripts/migrate.sh create add_new_column
```

這會在 `migrations/` 目錄下建立兩個檔案：
- `XXXXXX_add_new_column.up.sql` - 向上遷移（新增變更）
- `XXXXXX_add_new_column.down.sql` - 向下遷移（回滾變更）

**重要**：建立遷移後，需要重新構建 Docker 鏡像才能使用：
```bash
docker-compose build api
docker-compose up -d api
```

## 🔧 常見操作

### Docker 環境（推薦）

```bash
./scripts/docker-migrate.sh version    # 查看版本
./scripts/docker-migrate.sh up         # 執行遷移
./scripts/docker-migrate.sh down       # 回滾遷移
./scripts/docker-migrate.sh force 2    # 強制設定版本
```

### 本地開發（可選）

```bash
./scripts/migrate.sh version           # 查看版本（會顯示本地和 Docker 狀態）
./scripts/migrate.sh up                # 執行遷移
./scripts/migrate.sh down              # 回滾遷移
./scripts/migrate.sh force 2           # 強制設定版本
```

## 📂 遷移檔案結構

```
backend/migrations/
├── 000001_create_users_table.up.sql
├── 000001_create_users_table.down.sql
├── 000002_create_devices_table.up.sql
├── 000002_create_devices_table.down.sql
├── 000003_create_telemetries_table.up.sql
├── 000003_create_telemetries_table.down.sql
├── 000004_add_device_location.up.sql
└── 000004_add_device_location.down.sql
```

**文件說明**：
- `.up.sql`：向上遷移（前進），例如：創建表、添加列、創建索引
- `.down.sql`：向下遷移（回滾），例如：刪除表、刪除列、刪除索引
- 每個 `.up.sql` 都必須有對應的 `.down.sql` 用於回滾

## ✅ 優勢

✅ **版本控制**：每個 schema 變更都有明確的版本號  
✅ **可回滾**：透過 `.down.sql` 可以安全地撤銷變更  
✅ **多人協作**：團隊成員可以追蹤誰做了什麼變更  
✅ **安全刪除欄位**：可以建立遷移來移除不需要的欄位  
✅ **環境一致性**：開發、測試、生產環境使用相同的遷移腳本  
✅ **審查機制**：Schema 變更透過 Git PR 進行 code review

## ⚠️ 重要注意事項

- **不要刪除 `migrations/` 目錄**：這些是源代碼，Docker 構建需要，必須提交到 Git
- **不要手動修改資料庫 schema**：所有變更都應透過遷移檔案
- **向下遷移需謹慎**：在生產環境回滾可能導致資料遺失
- **保留 GORM tags**：model 定義中的 GORM tags 仍用於查詢和關聯
- **測試遷移**：在本地先測試 up 和 down 後再提交
- **推薦使用 `docker-migrate.sh`**：確保連接到正確的資料庫，避免本地 PostgreSQL 實例的干擾

## 🔍 故障排除

### 問題：執行 down 後，Docker 資料庫的版本沒有更新

**原因**：本地 `migrate` 工具可能連接到錯誤的資料庫實例。

**解決方案**：使用 `docker-migrate.sh` 而不是 `migrate.sh`：
```bash
# 使用 Docker 方式執行遷移
./scripts/docker-migrate.sh down
```

### 問題：遷移版本顯示 "dirty"

**原因**：上次遷移執行失敗或中斷。

**解決方案**：
```bash
# 檢查當前狀態
./scripts/docker-migrate.sh version

# 如果版本是 4 但表結構不對，強制設置回版本 3
./scripts/docker-migrate.sh force 3

# 重新執行遷移
./scripts/docker-migrate.sh up
```

### 問題：建立新遷移後，Docker 容器內看不到

**原因**：Docker 鏡像需要重新構建。

**解決方案**：
```bash
# 重新構建 API 鏡像
docker-compose build api

# 重啟服務
docker-compose up -d api
```

## 📚 相關資源

- [golang-migrate 官方文檔](https://github.com/golang-migrate/migrate)
- [PostgreSQL 文檔](https://www.postgresql.org/docs/)
- [GORM 文檔](https://gorm.io/docs/)
