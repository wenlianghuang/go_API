package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"my-api/api"
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
func (m *MockStore) Create(u store.User) error {
	if m.ShouldError {
		return errors.New("mock database error")
	}
	return nil
}

// 實作其他方法以滿足介面 (雖然這次測試用不到)
func (m *MockStore) Get(id string) (store.User, error) { return store.User{}, nil }
func (m *MockStore) List() ([]store.User, error)       { return nil, nil }

// 實作 Storage 介面的設備相關方法
func (m *MockStore) CreateDevice(dev *model.Device) error         { return nil }
func (m *MockStore) GetDeviceByID(id uint) (*model.Device, error) { return nil, nil }
func (m *MockStore) ListDevices() ([]model.Device, error)         { return nil, nil }
func (m *MockStore) AddTelemetry(data *model.Telemetry) error     { return nil }

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
			inputBody:      map[string]interface{}{"username": "testuser", "email": "test@example.com"},
			mockShouldErr:  false,
			expectedStatus: http.StatusCreated, // 預期 201
		},
		{
			name:           "Fail_MissingFields",
			inputBody:      map[string]interface{}{"username": ""}, // 缺少 email
			mockShouldErr:  false,
			expectedStatus: http.StatusBadRequest, // 預期 400
		},
		{
			name:           "Fail_DatabaseError",
			inputBody:      map[string]interface{}{"username": "testuser", "email": "db_error@test.com"},
			mockShouldErr:  true,                           // 模擬資料庫壞掉
			expectedStatus: http.StatusInternalServerError, // 預期 500
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 1. 準備依賴 (Arrange)
			// 使用我們的 MockStore，而不是真實的 MemoryStore
			mockStore := &MockStore{ShouldError: tt.mockShouldErr}
			srv := api.NewServer(mockStore)

			// 2. 準備請求 (Act)
			// 把 map 轉成 json body
			bodyBytes, _ := json.Marshal(tt.inputBody)
			req := httptest.NewRequest(http.MethodPost, "/users", bytes.NewBuffer(bodyBytes))
			req.Header.Set("Content-Type", "application/json")

			// httptest.NewRecorder 是一個 "假的" ResponseWriter
			// 它會把 Handler 寫入的資料通通記在記憶體裡，方便我們檢查
			rr := httptest.NewRecorder()

			// 直接呼叫 Handler (這裡不需要經過 Router，直接測函數邏輯)
			// 注意：我們是測 srv.HandleCreateUser，這就是依賴注入的好處
			handler := http.HandlerFunc(srv.HandleCreateUser)
			handler.ServeHTTP(rr, req)

			// 3. 驗證結果 (Assert)
			if status := rr.Code; status != tt.expectedStatus {
				t.Errorf("handler returned wrong status code: got %v want %v",
					status, tt.expectedStatus)
			}

			// 如果是成功案例，我們可以進一步檢查回傳的 JSON 內容
			if !tt.mockShouldErr && tt.expectedStatus == http.StatusCreated {
				var createdUser store.User
				json.NewDecoder(rr.Body).Decode(&createdUser)

				if createdUser.Username != tt.inputBody["username"] {
					t.Errorf("handler returned unexpected body: got username %v want %v",
						createdUser.Username, tt.inputBody["username"])
				}
			}
		})
	}
}
