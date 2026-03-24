package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"my-api/api"
	"my-api/config"
	"my-api/model"
	"my-api/store"
)

// 1. 定義一個 MockStore
// 這個 struct 專門給測試用，我們可以控制它何時報錯
type MockStore struct {
	// 這裡可以用來注入我們想要模擬的行為，例如 "是否要讓 Create 失敗"
	ShouldError bool
}

// 實作 Storage 介面的 Create 方法
func (m *MockStore) Create(ctx context.Context, u model.User) error {
	if m.ShouldError {
		return errors.New("mock database error")
	}
	return nil
}

// 實作其他方法以滿足介面 (雖然這次測試用不到)
func (m *MockStore) Get(ctx context.Context, id string) (model.User, error) { return model.User{}, nil }
func (m *MockStore) GetUserByEmail(ctx context.Context, email string) (model.User, error) {
	// 如果 ShouldError 為 true，返回錯誤表示用戶已存在
	if m.ShouldError {
		return model.User{}, errors.New("user already exists")
	}
	// 否則返回錯誤表示用戶不存在（這樣註冊才能成功）
	return model.User{}, errors.New("user not found")
}
func (m *MockStore) List(ctx context.Context) ([]model.User, error) { return nil, nil }

// 實作 Storage 介面的設備相關方法
func (m *MockStore) CreateDevice(ctx context.Context, dev *model.Device) error { return nil }
func (m *MockStore) GetDeviceByID(ctx context.Context, id uint) (*model.Device, error) {
	return nil, nil
}
func (m *MockStore) ListDevices(ctx context.Context) ([]model.Device, error)        { return nil, nil }
func (m *MockStore) ListTelemetries(ctx context.Context) ([]model.Telemetry, error) { return nil, nil }
func (m *MockStore) AddTelemetry(ctx context.Context, data *model.Telemetry) error  { return nil }
func (m *MockStore) GetTelemetryByID(ctx context.Context, id uint) (*model.Telemetry, error) {
	return nil, nil
}
func (m *MockStore) DeleteDeviceWithAllData(ctx context.Context, id uint) error {
	if m.ShouldError {
		return errors.New("mock delete error")
	}
	return nil
}

func (m *MockStore) UpdateDevice(ctx context.Context, id uint, device *model.Device) error {
	if m.ShouldError {
		return errors.New("mock update error")
	}
	return nil
}

func (m *MockStore) PatchDevice(ctx context.Context, id uint, updates map[string]interface{}) error {
	if m.ShouldError {
		return errors.New("mock patch error")
	}
	return nil
}
func (m *MockStore) PatchTelemetry(ctx context.Context, id uint, updates map[string]interface{}) error {
	if m.ShouldError {
		return errors.New("mock patch error")
	}
	return nil
}
func (m *MockStore) DeleteTelemetry(ctx context.Context, id uint) error {
	if m.ShouldError {
		return errors.New("mock delete error")
	}
	return nil
}

// ExecTx 執行一個事務
func (m *MockStore) ExecTx(ctx context.Context, fn func(txStorage store.Storage) error) error {
	if m.ShouldError {
		return errors.New("mock transaction error")
	}
	return fn(m)
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
func TestHandleCreateUser(t *testing.T) {
	// 定義測試表格 (Table)
	tests := []struct {
		name           string                 // 測試名稱
		inputBody      map[string]interface{} // 輸入的 JSON 資料
		mockShouldErr  bool                   // 是否模擬資料庫錯誤
		expectedStatus int                    // 預期收到的 HTTP 狀態碼
	}{
		{
			name:           "Success_CreateUser",
			inputBody:      map[string]interface{}{"username": "testuser", "email": "test@example.com", "password": "password123"},
			mockShouldErr:  false,
			expectedStatus: http.StatusCreated, // 預期 201
		},
		{
			name:           "Fail_MissingFields",
			inputBody:      map[string]interface{}{"username": ""}, // 缺少 email 和 password
			mockShouldErr:  false,
			expectedStatus: http.StatusBadRequest, // 預期 400
		},
		{
			name:           "Fail_DatabaseError",
			inputBody:      map[string]interface{}{"username": "testuser", "email": "db_error@test.com", "password": "password123"},
			mockShouldErr:  true,                           // 模擬資料庫壞掉
			expectedStatus: http.StatusInternalServerError, // 預期 500
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 1. 準備依賴 (Arrange)
			// 使用我們的 MockStore，而不是真實的 MemoryStore
			mockStore := &MockStore{ShouldError: tt.mockShouldErr}
			cfg := createTestConfig()
			srv := api.NewServer(mockStore, cfg, nil)

			// 2. 準備請求 (Act)
			// 把 map 轉成 json body
			bodyBytes, _ := json.Marshal(tt.inputBody)
			req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewBuffer(bodyBytes))
			req.Header.Set("Content-Type", "application/json")

			// httptest.NewRecorder 是一個 "假的" ResponseWriter
			// 它會把 Handler 寫入的資料通通記在記憶體裡，方便我們檢查
			rr := httptest.NewRecorder()

			// 直接呼叫 Handler (這裡不需要經過 Router，直接測函數邏輯)
			// 注意：現在測試 HandleRegister，因為 HandleCreateUser 已經 deprecated
			handler := http.HandlerFunc(srv.HandleRegister)
			handler.ServeHTTP(rr, req)

			// 3. 驗證結果 (Assert)
			if status := rr.Code; status != tt.expectedStatus {
				t.Errorf("handler returned wrong status code: got %v want %v",
					status, tt.expectedStatus)
			}

			// 如果是成功案例，我們可以進一步檢查回傳的 JSON 內容
			if !tt.mockShouldErr && tt.expectedStatus == http.StatusCreated {
				var response map[string]interface{}
				json.NewDecoder(rr.Body).Decode(&response)

				// HandleRegister 返回的是 AuthResponse，包含 user 對象
				if user, ok := response["user"].(map[string]interface{}); ok {
					if user["username"] != tt.inputBody["username"] {
						t.Errorf("handler returned unexpected body: got username %v want %v",
							user["username"], tt.inputBody["username"])
					}
				}
			}
		})
	}
}
