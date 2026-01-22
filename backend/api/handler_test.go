package api

import (
	"bytes"
	"context"
	"encoding/json"
	"my-api/config"
	"my-api/model"
	"my-api/store"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// 使用 testify/mock 重新定義 MockStore
type MockStore struct {
	mock.Mock
}

// User 相關方法
func (m *MockStore) Create(ctx context.Context, u model.User) error {
	args := m.Called(ctx, mock.Anything) // 註冊時 ID 是動態生成的，所以用 Anything
	return args.Error(0)
}

func (m *MockStore) Get(ctx context.Context, id string) (model.User, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(model.User), args.Error(1)
}

func (m *MockStore) GetUserByEmail(ctx context.Context, email string) (model.User, error) {
	args := m.Called(ctx, email)
	return args.Get(0).(model.User), args.Error(1)
}

func (m *MockStore) List(ctx context.Context) ([]model.User, error) {
	args := m.Called(ctx)
	return args.Get(0).([]model.User), args.Error(1)
}

// 設備相關方法
func (m *MockStore) CreateDevice(ctx context.Context, dev *model.Device) error {
	args := m.Called(ctx, dev)
	return args.Error(0)
}

func (m *MockStore) GetDeviceByID(ctx context.Context, id uint) (*model.Device, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Device), args.Error(1)
}

func (m *MockStore) ListDevices(ctx context.Context) ([]model.Device, error) {
	args := m.Called(ctx)
	return args.Get(0).([]model.Device), args.Error(1)
}

func (m *MockStore) DeleteDeviceWithAllData(ctx context.Context, id uint) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockStore) UpdateDevice(ctx context.Context, id uint, device *model.Device) error {
	args := m.Called(ctx, id, device)
	return args.Error(0)
}

func (m *MockStore) PatchDevice(ctx context.Context, id uint, updates map[string]interface{}) error {
	args := m.Called(ctx, id, updates)
	return args.Error(0)
}

// 遙測數據相關方法
func (m *MockStore) ListTelemetries(ctx context.Context) ([]model.Telemetry, error) {
	args := m.Called(ctx)
	return args.Get(0).([]model.Telemetry), args.Error(1)
}

func (m *MockStore) AddTelemetry(ctx context.Context, data *model.Telemetry) error {
	args := m.Called(ctx, data)
	return args.Error(0)
}

func (m *MockStore) GetTelemetryByID(ctx context.Context, id uint) (*model.Telemetry, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Telemetry), args.Error(1)
}

func (m *MockStore) PatchTelemetry(ctx context.Context, id uint, updates map[string]interface{}) error {
	args := m.Called(ctx, id, updates)
	return args.Error(0)
}

func (m *MockStore) DeleteTelemetry(ctx context.Context, id uint) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// ExecTx 執行一個事務
func (m *MockStore) ExecTx(ctx context.Context, fn func(txStorage store.Storage) error) error {
	args := m.Called(ctx, fn)
	if args.Get(0) == nil {
		// 如果沒有設置返回值，直接執行函數（用於簡單的測試場景）
		return fn(m)
	}
	return args.Error(0)
}

// createTestConfig 創建測試用的配置
func createTestConfig() *config.Config {
	return &config.Config{
		Database: config.DatabaseConfig{
			DSN: "test-dsn",
		},
		Redis: config.RedisConfig{
			Addr: "localhost:6379",
		},
		App: config.AppConfig{
			APIPort:     "8080",
			MetricsPort: "9090",
			JWTSecret:   "test-secret",
		},
	}
}

func TestHandleRegister_Success(t *testing.T) {
	// 1. 初始化
	mockStore := new(MockStore)
	cfg := createTestConfig()
	srv := NewServer(mockStore, cfg, nil)

	// 2. 定義 Mock 行為 (Expectations)
	// 當 Handler 呼叫 GetUserByEmail 時，回傳「找不到記錄」
	mockStore.On("GetUserByEmail", mock.Anything, "new@example.com").Return(model.User{}, gorm.ErrRecordNotFound)
	// 當 Handler 呼叫 Create 時，回傳 nil (成功)
	mockStore.On("Create", mock.Anything, mock.Anything).Return(nil)

	// 3. 準備請求
	reqBody := RegisterRequest{
		Username: "newuser",
		Email:    "new@example.com",
		Password: "password123",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	// 4. 執行
	srv.HandleRegister(w, req)

	// 5. 驗證 (Assert)
	assert.Equal(t, http.StatusCreated, w.Code)

	var resp AuthResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp.Token) // 確保真的有產生 JWT
	assert.Equal(t, "newuser", resp.User.Username)

	// 關鍵：驗證 Mock 的方法是否有被按照預期呼叫
	mockStore.AssertExpectations(t)
}

func TestHandleLogin_Success(t *testing.T) {
	// 1. 初始化
	mockStore := new(MockStore)
	cfg := createTestConfig()
	srv := NewServer(mockStore, cfg, nil)

	// 生成真正的 bcrypt 哈希值用於測試
	// 密碼是 "password123"
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	require.NoError(t, err, "Failed to generate bcrypt hash for testing")

	// 2. 定義 Mock 行為 (Expectations)
	// 當 Handler 呼叫 GetUserByEmail 時，回傳「找到記錄」
	mockStore.On("GetUserByEmail", mock.Anything, "existing@example.com").Return(model.User{
		ID:       "usr_123",
		Username: "existinguser",
		Email:    "existing@example.com",
		Password: string(hashedPassword), // 使用真正的 bcrypt 哈希值
	}, nil)

	// 3. 準備請求
	reqBody := LoginRequest{
		Email:    "existing@example.com",
		Password: "password123",
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	// 4. 執行
	srv.HandleLogin(w, req)

	// 5. 驗證 (Assert)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp AuthResponse
	err = json.Unmarshal(w.Body.Bytes(), &resp)

	assert.NoError(t, err)
	assert.NotEmpty(t, resp.Token) // 確保真的有產生 JWT
	assert.Equal(t, "existinguser", resp.User.Username)

	// 關鍵：驗證 Mock 的方法是否有被按照預期呼叫
	mockStore.AssertExpectations(t)
}
