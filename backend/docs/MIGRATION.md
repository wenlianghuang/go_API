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

## 🆕 創建新表的完整流程

以下說明如何從零開始創建一個新的資料庫表，包含遷移文件、Go Model 和 Store 層的完整步驟。

### 步驟 1：創建遷移文件

```bash
cd /Users/matthuang/Desktop/go_API/backend

# 創建新的遷移文件（例如：創建 notifications 表）
./scripts/migrate.sh create create_notifications_table
```

這會生成兩個檔案：
- `000005_create_notifications_table.up.sql`（創建表）
- `000005_create_notifications_table.down.sql`（刪除表）

### 步驟 2：編寫 UP 遷移（創建表）

編輯 `migrations/000005_create_notifications_table.up.sql`：

```sql
CREATE TABLE notifications (
    id SERIAL PRIMARY KEY,
    user_id VARCHAR(255) NOT NULL,
    title VARCHAR(255) NOT NULL,
    message TEXT NOT NULL,
    type VARCHAR(50) DEFAULT 'info',  -- 'info', 'warning', 'error'
    is_read BOOLEAN DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE,
    
    CONSTRAINT fk_notifications_user FOREIGN KEY (user_id) 
        REFERENCES users(id) ON DELETE CASCADE
);

-- 創建索引
CREATE INDEX idx_notifications_user_id ON notifications(user_id);
CREATE INDEX idx_notifications_is_read ON notifications(is_read);
CREATE INDEX idx_notifications_created_at ON notifications(created_at);
CREATE INDEX idx_notifications_deleted_at ON notifications(deleted_at);
```

### 步驟 3：編寫 DOWN 遷移（刪除表）

編輯 `migrations/000005_create_notifications_table.down.sql`：

```sql
DROP TABLE IF EXISTS notifications;
```

### 步驟 4：創建 Go Model

在 `model/` 目錄下創建新檔案，例如 `model/notification.go`：

```go
package model

import (
	"time"
	"gorm.io/gorm"
)

// Notification 代表一個通知
type Notification struct {
	gorm.Model
	
	UserID  string `gorm:"not null" json:"user_id"`
	Title   string `gorm:"size:255;not null" json:"title"`
	Message string `gorm:"type:text;not null" json:"message"`
	Type    string `gorm:"size:50;default:'info'" json:"type"` // 'info', 'warning', 'error'
	IsRead  bool   `gorm:"default:false" json:"is_read"`
	
	// 關聯（可選）
	User User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}
```

### 步驟 5：更新 Store 層（如果需要）

在 `store/gorm_store.go` 中添加相關方法：

```go
// CreateNotification 創建通知
func (s *GormStore) CreateNotification(notif *model.Notification) error {
	return s.db.Create(notif).Error
}

// GetNotificationByID 查詢單一通知
func (s *GormStore) GetNotificationByID(id uint) (*model.Notification, error) {
	var notif model.Notification
	result := s.db.First(&notif, id)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, errors.New("notification not found")
	}
	if result.Error != nil {
		return nil, result.Error
	}
	return &notif, nil
}

// ListNotificationsByUserID 查詢用戶的所有通知
func (s *GormStore) ListNotificationsByUserID(userID string) ([]model.Notification, error) {
	var notifications []model.Notification
	result := s.db.Where("user_id = ?", userID).Order("created_at DESC").Find(&notifications)
	return notifications, result.Error
}
```

### 步驟 6：重新構建 Docker 鏡像

```bash
# 重新構建 API 鏡像（包含新的遷移文件）
docker-compose build api

# 或者完全重建
docker-compose build --no-cache api
```

### 步驟 7：執行遷移

```bash
# 使用 Docker 方式執行遷移（推薦）
./scripts/docker-migrate.sh up

# 或者如果容器已啟動，遷移會自動執行
docker-compose up -d api
```

### 步驟 8：驗證遷移

```bash
# 檢查遷移版本
./scripts/docker-migrate.sh version
# 應該顯示：5

# 檢查表是否創建
docker-compose exec postgres psql -U postgres -d iot_db -c "\d notifications"

# 檢查表結構
docker-compose exec postgres psql -U postgres -d iot_db -c "\d+ notifications"
```

### 步驟 9：測試回滾（可選但推薦）

```bash
# 測試向下遷移（回滾）
./scripts/docker-migrate.sh down

# 驗證表被刪除
docker-compose exec postgres psql -U postgres -d iot_db -c "\d notifications"
# 應該顯示：relation "notifications" does not exist

# 再次執行向上遷移
./scripts/docker-migrate.sh up

# 驗證表重新創建
docker-compose exec postgres psql -U postgres -d iot_db -c "\d notifications"
```

### 步驟 10：提交到 Git

```bash
# 添加新檔案
git add migrations/000005_*.sql
git add model/notification.go
git add store/gorm_store.go  # 如果有修改

# 提交
git commit -m "feat: add notifications table migration"
```

### 完整流程總結

```
1. 創建遷移文件
   ↓
2. 編寫 .up.sql（CREATE TABLE）
   ↓
3. 編寫 .down.sql（DROP TABLE）
   ↓
4. 創建 Go Model
   ↓
5. 更新 Store 層（可選）
   ↓
6. 重新構建 Docker 鏡像
   ↓
7. 執行遷移（up）
   ↓
8. 驗證遷移成功
   ↓
9. 測試回滾（down/up）
   ↓
10. 提交到 Git
```

### 重要提示

1. **遷移文件命名**：使用描述性名稱，例如 `create_notifications_table`、`add_user_avatar_column`
2. **外鍵約束**：如果有外鍵，確保引用的表已存在（按遷移順序）
3. **索引**：為常用查詢欄位創建索引
4. **軟刪除**：如果需要軟刪除，添加 `deleted_at` 欄位
5. **時間戳**：通常包含 `created_at` 和 `updated_at`
6. **測試回滾**：在提交前測試 `down` 遷移

### 快速參考範例

```bash
# 1. 創建遷移
./scripts/migrate.sh create create_notifications_table

# 2. 編輯 up.sql 和 down.sql（手動編輯檔案）

# 3. 創建 model/notification.go（手動創建檔案）

# 4. 重新構建
docker-compose build api

# 5. 執行遷移
./scripts/docker-migrate.sh up

# 6. 驗證
./scripts/docker-migrate.sh version
docker-compose exec postgres psql -U postgres -d iot_db -c "\d notifications"
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
