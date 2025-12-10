package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"my-api/model"
	"my-api/store"

	"github.com/go-chi/chi/v5"
)

// HandleCreateUser 處理建立使用者的請求
func (s *Server) HandleCreateUser(w http.ResponseWriter, r *http.Request) {
	// 定義接收前端資料的結構 (DTO)
	type CreateUserRequest struct {
		Username string `json:"username"`
		Email    string `json:"email"`
	}

	var req CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	// 簡單驗證
	if req.Username == "" || req.Email == "" {
		WriteError(w, http.StatusBadRequest, "Username and Email are required")
		return
	}

	// 轉換成 Domain Model
	user := store.User{
		ID:        fmt.Sprintf("usr_%d", time.Now().UnixNano()), // 簡單生成 ID
		Username:  req.Username,
		Email:     req.Email,
		CreatedAt: time.Now(),
	}

	// 呼叫資料庫層
	if err := s.Store.Create(user); err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusCreated, user)
}

// HandleListUsers 取得所有使用者
func (s *Server) HandleListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := s.Store.List()
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to fetch users")
		return
	}
	WriteJSON(w, http.StatusOK, users)
}

// HandleGetUser 取得單一使用者
func (s *Server) HandleGetUser(w http.ResponseWriter, r *http.Request) {
	// 從 URL 參數中取得 id
	id := chi.URLParam(r, "id")

	user, err := s.Store.Get(id)
	if err != nil {
		WriteError(w, http.StatusNotFound, "User not found")
		return
	}

	WriteJSON(w, http.StatusOK, user)
}

// HandleMe 回傳當前登入者的資訊
func (s *Server) HandleMe(w http.ResponseWriter, r *http.Request) {
	// 使用 Helper 安全地取出 ID
	userID, ok := GetUserIDFromContext(r.Context())

	if !ok {
		// 理論上經過 AuthMiddleware 不會發生這種事，但為了防禦性程式設計還是要寫
		WriteError(w, http.StatusInternalServerError, "User ID not found in context")
		return
	}

	// 回傳簡單的訊息
	response := map[string]string{
		"message": "You are authenticated!",
		"user_id": userID,
	}
	WriteJSON(w, http.StatusOK, response)
}

// HandleCreateDevice 處理建立設備的請求
func (s *Server) HandleCreateDevice(w http.ResponseWriter, r *http.Request) {
	type CreateDeviceRequest struct {
		Name       string `json:"name"`
		Type       string `json:"type"`
		MacAddress string `json:"mac_address"`
		IsActive   bool   `json:"is_active"`
	}

	var req CreateDeviceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	// 簡單驗證
	if req.Name == "" || req.MacAddress == "" {
		WriteError(w, http.StatusBadRequest, "Name and MacAddress are required")
		return
	}

	// 轉換成 Domain Model
	device := &model.Device{
		Name:       req.Name,
		Type:       req.Type,
		MacAddress: req.MacAddress,
		IsActive:   req.IsActive,
	}

	// 如果沒有指定 IsActive，預設為 true
	if !r.URL.Query().Has("is_active") && !req.IsActive {
		device.IsActive = true
	}

	// 呼叫資料庫層
	if err := s.Store.CreateDevice(device); err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusCreated, device)
}

// HandleListDevices 取得所有設備
func (s *Server) HandleListDevices(w http.ResponseWriter, r *http.Request) {
	devices, err := s.Store.ListDevices()
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to fetch devices")
		return
	}
	WriteJSON(w, http.StatusOK, devices)
}

// HandleGetDevice 取得單一設備
func (s *Server) HandleGetDevice(w http.ResponseWriter, r *http.Request) {
	// 從 URL 參數中取得 id
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid device ID")
		return
	}

	device, err := s.Store.GetDeviceByID(uint(id))
	if err != nil {
		WriteError(w, http.StatusNotFound, "Device not found")
		return
	}

	WriteJSON(w, http.StatusOK, device)
}

// HandleCreateTelemetry 處理建立遙測數據的請求
func (s *Server) HandleCreateTelemetry(w http.ResponseWriter, r *http.Request) {
	type CreateTelemetryRequest struct {
		DeviceID   uint    `json:"device_id"`
		DataType   string  `json:"data_type"`
		Value      float64 `json:"value"`
		RecordedAt string  `json:"recorded_at,omitempty"` // 可選，如果沒有提供則使用當前時間
	}

	var req CreateTelemetryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, fmt.Sprintf("Invalid request payload: %v", err))
		return
	}

	// 簡單驗證
	if req.DeviceID == 0 || req.DataType == "" {
		WriteError(w, http.StatusBadRequest, "DeviceID and DataType are required")
		return
	}

	// 驗證設備是否存在
	_, err := s.Store.GetDeviceByID(req.DeviceID)
	if err != nil {
		WriteError(w, http.StatusNotFound, fmt.Sprintf("Device with ID %d not found", req.DeviceID))
		return
	}

	// 處理時間戳
	recordedAt := time.Now()
	if req.RecordedAt != "" {
		parsedTime, err := time.Parse(time.RFC3339, req.RecordedAt)
		if err != nil {
			// 如果解析失敗，使用當前時間
			recordedAt = time.Now()
		} else {
			recordedAt = parsedTime
		}
	}

	// 轉換成 Domain Model
	telemetry := &model.Telemetry{
		DeviceID:   req.DeviceID,
		DataType:   req.DataType,
		Value:      req.Value,
		RecordedAt: recordedAt,
	}

	// 呼叫資料庫層
	if err := s.Store.AddTelemetry(telemetry); err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusCreated, telemetry)
}
