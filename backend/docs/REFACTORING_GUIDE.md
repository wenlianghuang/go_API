# Go API 架構重構教學指南

本文件詳細說明如何實作**結構化配置管理**和**全面導入 Context 傳遞**兩個核心重構，以優化 Go API 架構。

---

## 目錄

1. [結構化配置管理 (Centralized Config)](#1-結構化配置管理-centralized-config)
2. [全面導入 Context 傳遞 (Context Propagation)](#2-全面導入-context-傳遞-context-propagation)
3. [最佳實踐與注意事項](#3-最佳實踐與注意事項)

---

## 1. 結構化配置管理 (Centralized Config)

### 1.1 概述

**結構化配置管理**是指將應用程式的所有配置集中管理，從環境變數中讀取並驗證，確保關鍵配置在啟動時就存在（快速失敗原則）。

### 1.2 實作步驟

#### 步驟 1：建立配置結構體

在 `backend/config/config.go` 中定義配置結構：

```go
package config

import (
	"fmt"
	"os"
)

// Config 應用程式配置結構
type Config struct {
	Database DatabaseConfig
	Redis    RedisConfig
	App      AppConfig
}

// DatabaseConfig 資料庫配置
type DatabaseConfig struct {
	DSN string
}

// RedisConfig Redis 配置
type RedisConfig struct {
	Addr string
}

// AppConfig 應用程式配置
type AppConfig struct {
	APIPort     string
	MetricsPort string
	JWTSecret   string
}
```

#### 步驟 2：實作 LoadConfig 函式

實作 `LoadConfig()` 函式，從環境變數讀取配置並進行驗證：

```go
// LoadConfig 載入配置並驗證（快速失敗）
func LoadConfig() (*Config, error) {
	cfg := &Config{}

	// 載入 Database 配置
	cfg.Database.DSN = os.Getenv("DB_DSN")
	if cfg.Database.DSN == "" {
		return nil, fmt.Errorf("DB_DSN is required but not set")
	}

	// 載入 Redis 配置（有預設值）
	cfg.Redis.Addr = os.Getenv("REDIS_ADDR")
	if cfg.Redis.Addr == "" {
		cfg.Redis.Addr = "localhost:6379" // 預設值
	}

	// 載入 App 配置
	cfg.App.APIPort = os.Getenv("API_PORT")
	if cfg.App.APIPort == "" {
		cfg.App.APIPort = "8080" // 預設值
	}

	cfg.App.MetricsPort = os.Getenv("METRICS_PORT")
	if cfg.App.MetricsPort == "" {
		cfg.App.MetricsPort = "9090" // 預設值
	}

	// JWT Secret 是關鍵配置，必須存在
	cfg.App.JWTSecret = os.Getenv("JWT_SECRET")
	if cfg.App.JWTSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET is required but not set")
	}

	return cfg, nil
}
```

**關鍵要點：**
- ✅ **快速失敗 (Fail-Fast)**：關鍵配置（如 `DB_DSN`、`JWT_SECRET`）缺失時立即返回錯誤
- ✅ **預設值**：非關鍵配置（如 `REDIS_ADDR`、`API_PORT`）可以設定預設值
- ✅ **結構化**：使用巢狀結構體組織配置，提高可讀性和維護性

#### 步驟 3：修改 main.go

在 `main.go` 中最早呼叫 `LoadConfig()`：

```go
package main

import (
	"log"
	"my-api/config"
	// ... 其他 imports
)

func main() {
	// 1. 載入配置（快速失敗）
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("無法載入配置: %v", err)
	}

	// 2. 使用配置連接資料庫
	db, err := gorm.Open(postgres.Open(cfg.Database.DSN), &gorm.Config{})
	// ...

	// 3. 初始化 Server（注入配置）
	srv := api.NewServer(gormStore, cfg)

	// 4. 使用配置中的端口啟動服務
	apiServer := &http.Server{
		Addr: ":" + cfg.App.APIPort,
		// ...
	}
	// ...
}
```

#### 步驟 4：修改 Server 結構

在 `backend/api/server.go` 中讓 `Server` 持有配置：

```go
type Server struct {
	Router           *chi.Mux
	Store            store.Storage
	AuthService      *service.AuthService
	DeviceService    *service.DeviceService
	TelemetryService *service.TelemetryService
	Config           *config.Config  // 🆕 新增配置欄位
	// ... 其他欄位
}

// NewServer 初始化 Server
func NewServer(store store.Storage, cfg *config.Config) *Server {
	// 使用配置初始化 Redis
	rdb := redis.NewClient(&redis.Options{
		Addr: cfg.Redis.Addr, // 使用配置中的 Redis 位址
	})

	s := &Server{
		Router:      chi.NewRouter(),
		Store:       store,
		Config:      cfg, // 🆕 儲存配置
		AuthService: service.NewAuthService(store, NewJWTService(cfg)),
		// ...
	}
	// ...
}
```

#### 步驟 5：修改 JWT Service

在 `backend/api/jwt.go` 中讓 `JWTService` 從配置中獲取 secret：

```go
// JWTService 實現 service.JWTGenerator 接口
type JWTService struct {
	secret string  // 🆕 將 secret 儲存在結構體中
}

// NewJWTService 創建一個新的 JWTService 實例
func NewJWTService(cfg *config.Config) *JWTService {
	return &JWTService{
		secret: cfg.App.JWTSecret, // 🆕 從配置中獲取
	}
}

// ValidateJWT 實現 service.JWTGenerator 接口
func (j *JWTService) ValidateJWT(tokenString string) (service.JWTClaims, error) {
	claims, err := ValidateJWT(j.secret, tokenString) // 🆕 使用結構體中的 secret
	// ...
}
```

### 1.3 環境變數設定

建立 `.env` 檔案（或使用系統環境變數）：

```bash
# 資料庫配置（必填）
DB_DSN=host=localhost user=postgres password=postgres dbname=iot_db port=5432 sslmode=disable

# Redis 配置（選填，有預設值）
REDIS_ADDR=localhost:6379

# 應用程式配置
API_PORT=8080
METRICS_PORT=9090

# JWT 配置（必填）
JWT_SECRET=your-super-secret-jwt-key-change-in-production
```

### 1.4 優點

- ✅ **集中管理**：所有配置在一個地方，易於維護
- ✅ **快速失敗**：關鍵配置缺失時立即報錯，避免運行時錯誤
- ✅ **類型安全**：使用結構體定義配置，編譯時檢查
- ✅ **易於測試**：可以輕鬆建立測試用的配置物件
- ✅ **環境隔離**：不同環境（開發、測試、生產）使用不同的環境變數

---

## 2. 全面導入 Context 傳遞 (Context Propagation)

### 2.1 概述

**Context 傳遞**是指在整個應用程式層級（Handler → Service → Store）中傳遞 `context.Context`，用於：
- 請求超時控制
- 取消操作
- 傳遞請求範圍的值（如用戶 ID、追蹤 ID）
- 資料庫查詢追蹤

### 2.2 實作步驟

#### 步驟 1：修改 Storage 介面

在 `backend/store/db.go` 中為所有方法添加 `ctx context.Context` 參數：

```go
package store

import (
	"context"  // 🆕 導入 context
	"my-api/model"
)

// Storage 定義了資料庫的行為 (Interface)
type Storage interface {
	// User 相關
	Create(ctx context.Context, u model.User) error
	Get(ctx context.Context, id string) (model.User, error)
	GetUserByEmail(ctx context.Context, email string) (model.User, error)
	List(ctx context.Context) ([]model.User, error)

	// 設備相關
	CreateDevice(ctx context.Context, dev *model.Device) error
	GetDeviceByID(ctx context.Context, id uint) (*model.Device, error)
	ListDevices(ctx context.Context) ([]model.Device, error)
	UpdateDevice(ctx context.Context, id uint, device *model.Device) error
	PatchDevice(ctx context.Context, id uint, updates map[string]interface{}) error
	DeleteDeviceWithAllData(ctx context.Context, id uint) error

	// 遙測數據相關
	ListTelemetries(ctx context.Context) ([]model.Telemetry, error)
	AddTelemetry(ctx context.Context, data *model.Telemetry) error
	GetTelemetryByID(ctx context.Context, id uint) (*model.Telemetry, error)
	PatchTelemetry(ctx context.Context, id uint, updates map[string]interface{}) error
	DeleteTelemetry(ctx context.Context, id uint) error
}
```

**關鍵要點：**
- ✅ `context.Context` 必須是第一個參數
- ✅ 所有方法都需要添加 `ctx` 參數
- ✅ 介面定義後，所有實作都必須符合介面

#### 步驟 2：修改 GormStore 實作

在 `backend/store/gorm_store.go` 中為所有 GORM 查詢添加 `.WithContext(ctx)`：

```go
package store

import (
	"context"  // 🆕 導入 context
	"my-api/model"
	"gorm.io/gorm"
)

// Create 實作建立使用者
func (s *GormStore) Create(ctx context.Context, u model.User) error {
	return s.db.WithContext(ctx).Create(&u).Error  // 🆕 使用 WithContext
}

// Get 實作查詢單一使用者
func (s *GormStore) Get(ctx context.Context, id string) (model.User, error) {
	var user model.User
	result := s.db.WithContext(ctx).First(&user, "id = ?", id)  // 🆕 使用 WithContext
	// ...
}

// CreateDevice 實作建立設備
func (s *GormStore) CreateDevice(ctx context.Context, dev *model.Device) error {
	result := s.db.WithContext(ctx).Create(dev)  // 🆕 使用 WithContext
	return result.Error
}

// GetDeviceByID 實作查詢單一設備
func (s *GormStore) GetDeviceByID(ctx context.Context, id uint) (*model.Device, error) {
	var dev model.Device
	result := s.db.WithContext(ctx).Preload("Telemetries").First(&dev, id)  // 🆕 使用 WithContext
	// ...
}

// ListDevices 實作列表查詢
func (s *GormStore) ListDevices(ctx context.Context) ([]model.Device, error) {
	var devices []model.Device
	result := s.db.WithContext(ctx).Find(&devices)  // 🆕 使用 WithContext
	return devices, result.Error
}
```

**關鍵要點：**
- ✅ 所有 GORM 查詢都必須使用 `.WithContext(ctx)`
- ✅ 這讓 GORM 能夠追蹤查詢、支援超時和取消操作

#### 步驟 3：修改 Service 層

在 `backend/service/` 下的所有服務方法中添加 `ctx context.Context` 參數：

**範例：`backend/service/device_service.go`**

```go
package service

import (
	"context"  // 🆕 導入 context
	"fmt"
	"my-api/model"
	"my-api/store"
)

// CreateDevice 處理創建設備業務邏輯
func (s *DeviceService) CreateDevice(ctx context.Context, input CreateDeviceInput, defaultIsActive bool) (*CreateDeviceResult, error) {
	// 轉換成 Domain Model
	device := &model.Device{
		Name:       input.Name,
		Type:       input.Type,
		MacAddress: input.MacAddress,
		IsActive:   input.IsActive,
		UserID:     input.UserID,
	}

	// 呼叫資料庫層（傳遞 context）
	if err := s.store.CreateDevice(ctx, device); err != nil {  // 🆕 傳遞 ctx
		return nil, fmt.Errorf("failed to create device: %w", err)
	}

	return &CreateDeviceResult{Device: device}, nil
}

// PatchDevice 處理部分更新設備業務邏輯
func (s *DeviceService) PatchDevice(ctx context.Context, deviceID uint, input PatchDeviceInput) (*PatchDeviceResult, error) {
	// 驗證設備是否存在（傳遞 context）
	_, err := s.store.GetDeviceByID(ctx, deviceID)  // 🆕 傳遞 ctx
	if err != nil {
		return nil, fmt.Errorf("device not found")
	}

	// 執行部分更新（傳遞 context）
	if err := s.store.PatchDevice(ctx, deviceID, updates); err != nil {  // 🆕 傳遞 ctx
		return nil, fmt.Errorf("failed to update device: %w", err)
	}

	// 獲取更新後的設備（傳遞 context）
	updatedDevice, err := s.store.GetDeviceByID(ctx, deviceID)  // 🆕 傳遞 ctx
	// ...
}
```

**範例：`backend/service/auth_service.go`**

```go
// Register 處理用戶註冊
func (s *AuthService) Register(ctx context.Context, input RegisterInput) (*RegisterResult, error) {
	// 檢查用戶是否已存在（傳遞 context）
	_, err := s.Store.GetUserByEmail(ctx, input.Email)  // 🆕 傳遞 ctx
	// ...

	// 創建用戶（傳遞 context）
	if err := s.Store.Create(ctx, user); err != nil {  // 🆕 傳遞 ctx
		return nil, fmt.Errorf("failed to create user: %w", err)
	}
	// ...
}

// Login 處理用戶登入
func (s *AuthService) Login(ctx context.Context, input LoginInput) (*LoginResult, error) {
	// 查詢用戶（傳遞 context）
	user, err := s.Store.GetUserByEmail(ctx, input.Email)  // 🆕 傳遞 ctx
	// ...
}
```

#### 步驟 4：修改 Handler 層

在 `backend/api/` 下的所有 Handler 方法中從 `r.Context()` 獲取 context 並傳遞給 Service：

**範例：`backend/api/device_handlers.go`**

```go
package api

import (
	"net/http"
	"my-api/service"
)

// HandleCreateDevice 處理建立設備的請求
func (s *Server) HandleCreateDevice(w http.ResponseWriter, r *http.Request) {
	var req CreateDeviceRequest

	// 解析請求
	if err := ValidateAndDecode(r, &req); err != nil {
		HandleValidationError(w, err)
		return
	}

	// 從 context 中獲取當前登入用戶的 ID（由 AuthMiddleware 注入）
	userID, ok := GetUserIDFromContext(r.Context())
	if !ok {
		WriteError(w, http.StatusUnauthorized, "User ID not found in context")
		return
	}

	// 調用 service 執行業務邏輯（傳遞 context）
	result, err := s.DeviceService.CreateDevice(
		r.Context(),  // 🆕 從 Request 獲取 context
		service.CreateDeviceInput{
			Name:       req.Name,
			Type:       req.Type,
			MacAddress: req.MacAddress,
			IsActive:   req.IsActive,
			UserID:     userID,
		},
		defaultIsActive,
	)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusCreated, result.Device)
}

// HandleListDevices 取得所有設備
func (s *Server) HandleListDevices(w http.ResponseWriter, r *http.Request) {
	devices, err := s.Store.ListDevices(r.Context())  // 🆕 傳遞 context
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to fetch devices")
		return
	}
	WriteJSON(w, http.StatusOK, devices)
}

// HandleGetDevice 取得單一設備
func (s *Server) HandleGetDevice(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid device ID")
		return
	}

	device, err := s.Store.GetDeviceByID(r.Context(), uint(id))  // 🆕 傳遞 context
	if err != nil {
		WriteError(w, http.StatusNotFound, "Device not found")
		return
	}
	WriteJSON(w, http.StatusOK, device)
}
```

**範例：`backend/api/auth_handlers.go`**

```go
// HandleRegister 處理用戶註冊
func (s *Server) HandleRegister(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := ValidateAndDecode(r, &req); err != nil {
		HandleValidationError(w, err)
		return
	}

	// 調用 service（傳遞 context）
	result, err := s.AuthService.Register(
		r.Context(),  // 🆕 從 Request 獲取 context
		service.RegisterInput{
			Username: req.Username,
			Email:    req.Email,
			Password: req.Password,
		},
	)
	// ...
}

// HandleLogin 處理用戶登入
func (s *Server) HandleLogin(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := ValidateAndDecode(r, &req); err != nil {
		HandleValidationError(w, err)
		return
	}

	// 調用 service（傳遞 context）
	result, err := s.AuthService.Login(
		r.Context(),  // 🆕 從 Request 獲取 context
		service.LoginInput{
			Email:    req.Email,
			Password: req.Password,
		},
	)
	// ...
}
```

### 2.3 Context 的進階用法

#### 2.3.1 傳遞請求範圍的值

在 Middleware 中將用戶 ID 注入到 context：

```go
// backend/api/middleware.go

type contextKey string
const UserIDKey contextKey = "userID"

// AuthMiddleware 驗證 JWT Token 並注入 User ID
func (s *Server) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 驗證 JWT Token
		claims, err := s.AuthService.JWTGenerator.ValidateJWT(tokenString)
		// ...

		// 將 UserID 注入到 Context
		ctx := context.WithValue(r.Context(), UserIDKey, claims.GetUserID())

		// 呼叫下一個 Handler，傳入帶有新 Context 的 Request
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetUserIDFromContext 從 Context 獲取用戶 ID
func GetUserIDFromContext(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value(UserIDKey).(string)
	return userID, ok
}
```

在 Handler 中使用：

```go
func (s *Server) HandleCreateDevice(w http.ResponseWriter, r *http.Request) {
	// 從 context 中獲取用戶 ID
	userID, ok := GetUserIDFromContext(r.Context())
	if !ok {
		WriteError(w, http.StatusUnauthorized, "User ID not found in context")
		return
	}

	// 使用 userID 創建設備
	// ...
}
```

#### 2.3.2 請求超時控制

在 Server 層面設定請求超時：

```go
// 在 main.go 或 server.go 中
func (s *Server) setupRoutes() {
	s.Router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 設定 30 秒超時
			ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
			defer cancel()

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	})
}
```

### 2.4 優點

- ✅ **請求追蹤**：可以追蹤整個請求的生命週期
- ✅ **超時控制**：可以設定請求超時，避免長時間阻塞
- ✅ **取消操作**：可以取消正在進行的操作
- ✅ **傳遞值**：可以在不同層級間傳遞請求範圍的值（如用戶 ID、追蹤 ID）
- ✅ **資料庫追蹤**：GORM 可以追蹤查詢，便於除錯和性能分析

---

## 3. 最佳實踐與注意事項

### 3.1 配置管理最佳實踐

1. **快速失敗原則**
   - 關鍵配置（如資料庫連接、JWT Secret）必須在啟動時驗證
   - 使用 `log.Fatalf` 立即終止程式，避免運行時錯誤

2. **環境變數命名**
   - 使用大寫字母和底線（如 `DB_DSN`、`JWT_SECRET`）
   - 保持命名一致性和可讀性

3. **預設值處理**
   - 非關鍵配置可以設定合理的預設值
   - 關鍵配置不應有預設值，必須明確設定

4. **配置驗證**
   - 驗證配置格式（如 URL、端口範圍）
   - 使用結構體標籤進行驗證（可選）

### 3.2 Context 傳遞最佳實踐

1. **Context 作為第一個參數**
   - 所有需要 context 的函式都應將 `context.Context` 作為第一個參數
   - 這是 Go 的慣例

2. **不要儲存 Context**
   - 不要將 context 儲存在結構體中
   - Context 應該作為參數傳遞

3. **不要傳遞 nil Context**
   - 如果函式需要 context，應該接受 `context.Context` 參數
   - 如果不需要，可以使用 `context.Background()` 或 `context.TODO()`

4. **Context 的超時和取消**
   - 使用 `context.WithTimeout` 設定超時
   - 使用 `context.WithCancel` 實現取消操作
   - 記得呼叫 `defer cancel()`

5. **Context 值的類型安全**
   - 使用自定義類型作為 context key，避免衝突
   ```go
   type contextKey string
   const UserIDKey contextKey = "userID"
   ```

### 3.3 測試注意事項

#### 測試配置管理

```go
func TestLoadConfig(t *testing.T) {
	// 設定環境變數
	os.Setenv("DB_DSN", "test-dsn")
	os.Setenv("JWT_SECRET", "test-secret")
	defer func() {
		os.Unsetenv("DB_DSN")
		os.Unsetenv("JWT_SECRET")
	}()

	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if cfg.Database.DSN != "test-dsn" {
		t.Errorf("Database.DSN = %v, want test-dsn", cfg.Database.DSN)
	}
}
```

#### 測試 Context 傳遞

```go
func TestCreateDevice(t *testing.T) {
	ctx := context.Background()
	mockStore := &MockStore{}
	service := NewDeviceService(mockStore)

	input := CreateDeviceInput{
		Name:       "Test Device",
		Type:       "Sensor",
		MacAddress: "00:11:22:33:44:55",
		IsActive:   true,
		UserID:     "usr_123",
	}

	result, err := service.CreateDevice(ctx, input, false)
	if err != nil {
		t.Fatalf("CreateDevice() error = %v", err)
	}

	if result.Device.Name != "Test Device" {
		t.Errorf("Device.Name = %v, want Test Device", result.Device.Name)
	}
}
```

### 3.4 常見錯誤與解決方案

#### 錯誤 1：忘記傳遞 Context

```go
// ❌ 錯誤：沒有傳遞 context
device, err := s.store.GetDeviceByID(deviceID)

// ✅ 正確：傳遞 context
device, err := s.store.GetDeviceByID(ctx, deviceID)
```

#### 錯誤 2：忘記使用 WithContext

```go
// ❌ 錯誤：GORM 查詢沒有使用 WithContext
result := s.db.Create(&device)

// ✅ 正確：使用 WithContext
result := s.db.WithContext(ctx).Create(&device)
```

#### 錯誤 3：配置驗證不足

```go
// ❌ 錯誤：沒有驗證關鍵配置
cfg.App.JWTSecret = os.Getenv("JWT_SECRET")

// ✅ 正確：驗證關鍵配置
cfg.App.JWTSecret = os.Getenv("JWT_SECRET")
if cfg.App.JWTSecret == "" {
	return nil, fmt.Errorf("JWT_SECRET is required but not set")
}
```

---

## 4. 總結

### 4.1 結構化配置管理

- ✅ 集中管理所有配置
- ✅ 快速失敗驗證
- ✅ 類型安全的配置結構
- ✅ 易於測試和維護

### 4.2 Context 傳遞

- ✅ 完整的請求追蹤
- ✅ 超時和取消支援
- ✅ 請求範圍的值傳遞
- ✅ 資料庫查詢追蹤

### 4.3 重構後的架構優勢

1. **可維護性**：配置集中管理，易於修改和擴展
2. **可測試性**：可以輕鬆建立測試用的配置和 context
3. **可追蹤性**：完整的 context 傳遞支援請求追蹤和除錯
4. **穩定性**：快速失敗驗證避免運行時錯誤
5. **擴展性**：易於添加新的配置項和 context 值

---

## 5. 參考資源

- [Go Context Package](https://pkg.go.dev/context)
- [GORM Context Support](https://gorm.io/docs/context.html)
- [12-Factor App: Config](https://12factor.net/config)
- [Go Best Practices: Context](https://go.dev/blog/context)

---

**最後更新：** 2026-01-18  
**作者：** Go API 開發團隊
