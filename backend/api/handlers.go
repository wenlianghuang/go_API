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
	"github.com/gorilla/websocket"
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
	// 2. 🔥 【關鍵步驟】 將 GORM Model 轉換為 DTO
	resp := ToDeviceResponse(device)
	//WriteJSON(w, http.StatusOK, device)
	WriteJSON(w, http.StatusOK, resp)
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
	s.Hub.Broadcast(ToTelemetryResponse(*telemetry)) // 使用 DTO 推播乾淨的數據
}

// HandleGetTelemetry 處理取得遙測數據的請求
func (s *Server) HandleGetTelemetry(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid telemetry ID")
		return
	}

	telemetry, err := s.Store.GetTelemetryByID(uint(id))
	if err != nil {
		WriteError(w, http.StatusNotFound, "Telemetry not found")
		return
	}
	WriteJSON(w, http.StatusOK, telemetry)
}

// 處理刪除請求
func (s *Server) HandleDeleteDevice(w http.ResponseWriter, r *http.Request) {
	// 1. 解析 ID
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid device ID")
		return
	}

	// 2. 呼叫 Store 執行原子性刪除
	// 因為我们在介面 (store/db.go) 定義了，所以這裡可以呼叫 s.Store.DeleteDeviceWithAllData
	if err := s.Store.DeleteDeviceWithAllData(uint(id)); err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to delete device: "+err.Error())
		return
	}

	// 3. 回傳成功訊息
	WriteJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// HandleUpdateDevice 處理更新整個設備的請求
func (s *Server) HandleUpdateDevice(w http.ResponseWriter, r *http.Request) {
	type UpdateDeviceRequest struct {
		Name       string `json:"name"`
		Type       string `json:"type"`
		MacAddress string `json:"mac_address"`
		IsActive   bool   `json:"is_active"`
	}

	// 1. 解析 ID
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid device ID")
		return
	}

	// 2. 解析請求體
	var req UpdateDeviceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	// 3. PUT 要求所有必填字段都必須提供
	if req.Name == "" || req.MacAddress == "" {
		WriteError(w, http.StatusBadRequest, "Name and MacAddress are required for PUT request")
		return
	}

	// 4. 驗證設備是否存在
	_, err = s.Store.GetDeviceByID(uint(id))
	if err != nil {
		WriteError(w, http.StatusNotFound, "Device not found")
		return
	}

	// 5. 轉換成 Domain Model
	device := &model.Device{
		Name:       req.Name,
		Type:       req.Type,
		MacAddress: req.MacAddress,
		IsActive:   req.IsActive,
	}

	// 6. 更新設備
	if err := s.Store.UpdateDevice(uint(id), device); err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to update device: "+err.Error())
		return
	}

	// 7. 回傳更新後的設備資訊
	updatedDevice, err := s.Store.GetDeviceByID(uint(id))
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to fetch updated device")
		return
	}

	resp := ToDeviceResponse(updatedDevice)
	WriteJSON(w, http.StatusOK, resp)
}

// HandlePatchDevice 處理 PATCH 請求 - 部分更新（只需要提供要更新的字段）
func (s *Server) HandlePatchDevice(w http.ResponseWriter, r *http.Request) {
	type PatchDeviceRequest struct {
		Name       *string `json:"name,omitempty"` // 使用指針，nil 表示不更新
		Type       *string `json:"type,omitempty"`
		MacAddress *string `json:"mac_address,omitempty"`
		IsActive   *bool   `json:"is_active,omitempty"`
	}

	// 1. 解析 ID
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid device ID")
		return
	}

	// 2. 驗證設備是否存在
	_, err = s.Store.GetDeviceByID(uint(id))
	if err != nil {
		WriteError(w, http.StatusNotFound, "Device not found")
		return
	}

	// 3. 解析請求體
	var req PatchDeviceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	// 4. 構建更新映射（只包含提供的字段）
	updates := make(map[string]interface{})
	if req.Name != nil {
		if *req.Name == "" {
			WriteError(w, http.StatusBadRequest, "Name cannot be empty")
			return
		}
		updates["name"] = *req.Name
	}
	if req.Type != nil {
		updates["type"] = *req.Type
	}
	if req.MacAddress != nil {
		if *req.MacAddress == "" {
			WriteError(w, http.StatusBadRequest, "MacAddress cannot be empty")
			return
		}
		updates["mac_address"] = *req.MacAddress
	}
	if req.IsActive != nil {
		updates["is_active"] = *req.IsActive
	}

	// 5. 如果沒有任何更新字段
	if len(updates) == 0 {
		WriteError(w, http.StatusBadRequest, "At least one field must be provided for update")
		return
	}

	// 6. 執行部分更新
	if err := s.Store.PatchDevice(uint(id), updates); err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to update device: "+err.Error())
		return
	}

	// 7. 回傳更新後的設備資訊
	updatedDevice, err := s.Store.GetDeviceByID(uint(id))
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to fetch updated device")
		return
	}

	resp := ToDeviceResponse(updatedDevice)
	WriteJSON(w, http.StatusOK, resp)
}

func (s *Server) HandleAnalyzeDevice(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, _ := strconv.ParseUint(idStr, 10, 32)

	// 將任務丟進通道，如果通道滿了會暫時阻塞
	s.TaskChan <- uint(id)

	WriteJSON(w, http.StatusAccepted, map[string]string{
		"message": "Task queued",
	})
}

// backend/api/handlers.go

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true }, // 允許所有來源連線
}

// HandleWS 處理 WebSocket 連線
func (s *Server) HandleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Printf("❌ WS 升級失敗: %v\n", err)
		return
	}

	s.Hub.AddClient(conn)

	// 保持連線，直到客戶端斷開
	defer s.Hub.RemoveClient(conn)
	for {
		// 這裡可以讀取客戶端傳來的訊息，目前我們先放著讓它維持連線
		if _, _, err := conn.ReadMessage(); err != nil {
			break
		}
	}
}

// HandlePatchTelemetry 處理 PATCH 請求 - 部分更新遙測數據（只需要提供要更新的字段）
func (s *Server) HandlePatchTelemetry(w http.ResponseWriter, r *http.Request) {
	type PatchTelemetryRequest struct {
		DeviceID   *uint    `json:"device_id,omitempty"` // 使用指針，nil 表示不更新
		DataType   *string  `json:"data_type,omitempty"`
		Value      *float64 `json:"value,omitempty"`
		RecordedAt *string  `json:"recorded_at,omitempty"`
	}

	// 1. 解析 ID
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid telemetry ID")
		return
	}

	// 2. 驗證遙測數據是否存在
	_, err = s.Store.GetTelemetryByID(uint(id))
	if err != nil {
		WriteError(w, http.StatusNotFound, "Telemetry not found")
		return
	}

	// 3. 解析請求體
	var req PatchTelemetryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	// 4. 構建更新映射（只包含提供的字段）
	updates := make(map[string]interface{})
	if req.DeviceID != nil {
		if *req.DeviceID == 0 {
			WriteError(w, http.StatusBadRequest, "DeviceID cannot be zero")
			return
		}
		// 驗證設備是否存在
		_, err := s.Store.GetDeviceByID(*req.DeviceID)
		if err != nil {
			WriteError(w, http.StatusNotFound, fmt.Sprintf("Device with ID %d not found", *req.DeviceID))
			return
		}
		updates["device_id"] = *req.DeviceID
	}
	if req.DataType != nil {
		if *req.DataType == "" {
			WriteError(w, http.StatusBadRequest, "DataType cannot be empty")
			return
		}
		updates["data_type"] = *req.DataType
	}
	if req.Value != nil {
		updates["value"] = *req.Value
	}
	if req.RecordedAt != nil {
		parsedTime, err := time.Parse(time.RFC3339, *req.RecordedAt)
		if err != nil {
			WriteError(w, http.StatusBadRequest, "Invalid recorded_at format, expected RFC3339")
			return
		}
		updates["recorded_at"] = parsedTime
	}

	// 5. 如果沒有任何更新字段
	if len(updates) == 0 {
		WriteError(w, http.StatusBadRequest, "At least one field must be provided for update")
		return
	}

	// 6. 執行部分更新
	if err := s.Store.PatchTelemetry(uint(id), updates); err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to update telemetry: "+err.Error())
		return
	}

	// 7. 回傳更新後的遙測數據資訊
	updatedTelemetry, err := s.Store.GetTelemetryByID(uint(id))
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to fetch updated telemetry")
		return
	}

	WriteJSON(w, http.StatusOK, updatedTelemetry)
}

// Handle Get telemetry
