package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"my-api/model"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"
	"golang.org/x/crypto/bcrypt"
)

// ==========================================
// 認證相關的 Handlers
// ==========================================

// HandleRegister 處理用戶註冊請求
// @Summary      用戶註冊
// @Description  註冊一個新的用戶帳號並返回 JWT token
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        user  body      RegisterRequest  true  "註冊資訊"
// @Success      201   {object}  AuthResponse
// @Failure      400   {object}  ErrorResponse
// @Failure      409   {object}  ErrorResponse  "用戶已存在"
// @Failure      500   {object}  ErrorResponse
// @Router       /auth/register [post]
func (s *Server) HandleRegister(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	// 驗證必填字段
	if req.Username == "" || req.Email == "" || req.Password == "" {
		WriteError(w, http.StatusBadRequest, "Username, Email and Password are required")
		return
	}

	// 檢查密碼長度
	if len(req.Password) < 6 {
		WriteError(w, http.StatusBadRequest, "Password must be at least 6 characters")
		return
	}

	// 檢查用戶是否已存在
	if _, err := s.Store.GetUserByEmail(req.Email); err == nil {
		WriteError(w, http.StatusConflict, "User with this email already exists")
		return
	}

	// 對密碼進行哈希加密
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to hash password")
		return
	}

	// 創建用戶
	user := model.User{
		ID:        fmt.Sprintf("usr_%d", time.Now().UnixNano()),
		Username:  req.Username,
		Email:     req.Email,
		Password:  string(hashedPassword),
		CreatedAt: time.Now(),
	}

	// 保存到數據庫
	if err := s.Store.Create(user); err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to create user: "+err.Error())
		return
	}

	// 生成 JWT token
	token, err := GenerateJWT(user.ID, user.Username, user.Email, 5) // 5分鐘過期
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to generate token")
		return
	}

	// 返回響應
	response := AuthResponse{
		Token:     token,
		ExpiresAt: time.Now().Add(5 * time.Minute), // 與 GenerateJWT 的參數一致
		User:      ToUserResponse(user),
	}

	WriteJSON(w, http.StatusCreated, response)
}

// HandleLogin 處理用戶登入請求
// @Summary      用戶登入
// @Description  使用郵箱和密碼登入，返回 JWT token
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        credentials  body      LoginRequest  true  "登入憑證"
// @Success      200          {object}  AuthResponse
// @Failure      400          {object}  ErrorResponse
// @Failure      401          {object}  ErrorResponse  "郵箱或密碼錯誤"
// @Failure      500          {object}  ErrorResponse
// @Router       /auth/login [post]
func (s *Server) HandleLogin(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	// 驗證必填字段
	if req.Email == "" || req.Password == "" {
		WriteError(w, http.StatusBadRequest, "Email and Password are required")
		return
	}

	// 根據 email 查找用戶
	user, err := s.Store.GetUserByEmail(req.Email)
	if err != nil {
		WriteError(w, http.StatusUnauthorized, "Invalid email or password")
		return
	}

	// 驗證密碼
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		WriteError(w, http.StatusUnauthorized, "Invalid email or password")
		return
	}

	// 生成 JWT token
	token, err := GenerateJWT(user.ID, user.Username, user.Email, 5) // 5分鐘過期
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to generate token")
		return
	}

	// 返回響應
	response := AuthResponse{
		Token:     token,
		ExpiresAt: time.Now().Add(5 * time.Minute),
		User:      ToUserResponse(user),
	}

	WriteJSON(w, http.StatusOK, response)
}

// HandleRefreshToken 處理刷新 token 請求
// @Summary      刷新 JWT token
// @Description  使用舊的 token 刷新獲取新的 token（延長過期時間）
// @Tags         auth
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  AuthResponse
// @Failure      401  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Router       /auth/refresh [post]
func (s *Server) HandleRefreshToken(w http.ResponseWriter, r *http.Request) {
	// 從 header 中獲取 token
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		WriteError(w, http.StatusUnauthorized, "Missing Authorization header")
		return
	}

	// 解析 Bearer token
	var tokenString string
	if _, err := fmt.Sscanf(authHeader, "Bearer %s", &tokenString); err != nil {
		WriteError(w, http.StatusUnauthorized, "Invalid token format")
		return
	}

	// 刷新 token
	newToken, err := RefreshJWT(tokenString)
	if err != nil {
		WriteError(w, http.StatusUnauthorized, "Invalid or expired token")
		return
	}

	// 解析新 token 以獲取用戶信息
	claims, err := ValidateJWT(newToken)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to parse new token")
		return
	}

	// 從數據庫獲取用戶最新信息
	user, err := s.Store.Get(claims.UserID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to fetch user")
		return
	}

	// 返回響應
	response := AuthResponse{
		Token:     newToken,
		ExpiresAt: time.Now().Add(5 * time.Minute),
		User:      ToUserResponse(user),
	}

	WriteJSON(w, http.StatusOK, response)
}

// ==========================================
// 用戶管理相關的 Handlers (保留舊的，用於兼容)
// ==========================================

// CreateUserRequest 創建用戶請求結構（已廢棄，請使用 /auth/register）
type CreateUserRequest struct {
	Username string `json:"username" example:"john_doe" binding:"required"`
	Email    string `json:"email" example:"john@example.com" binding:"required"`
}

// HandleCreateUser 處理建立使用者的請求（已廢棄，請使用 /auth/register）
func (s *Server) HandleCreateUser(w http.ResponseWriter, r *http.Request) {
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
	user := model.User{
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
// @Summary      獲取所有用戶列表
// @Description  獲取系統中所有用戶的列表
// @Tags         users
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Success      200  {array}   UserResponse
// @Failure      401  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Router       /users [get]
func (s *Server) HandleListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := s.Store.List()
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to fetch users")
		return
	}
	WriteJSON(w, http.StatusOK, users)
}

// HandleGetUser 取得單一使用者
// @Summary      獲取單個用戶
// @Description  根據用戶 ID 獲取用戶詳細資訊
// @Tags         users
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string  true  "用戶 ID"
// @Success      200  {object}  UserResponse
// @Failure      401  {object}  ErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Router       /users/{id} [get]
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
// @Summary      獲取當前用戶資訊
// @Description  獲取當前已認證用戶的資訊
// @Tags         users
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  map[string]string
// @Failure      401  {object}  ErrorResponse
// @Router       /me [get]
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

// CreateDeviceRequest 創建設備請求結構
type CreateDeviceRequest struct {
	Name       string `json:"name" example:"Temperature Sensor 1" binding:"required"`
	Type       string `json:"type" example:"Sensor"`
	MacAddress string `json:"mac_address" example:"00:11:22:33:44:55" binding:"required"`
	IsActive   bool   `json:"is_active" example:"true"`
}

// HandleCreateDevice 處理建立設備的請求
// @Summary      創建新設備
// @Description  創建一個新的 IoT 設備
// @Tags         devices
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        device  body      CreateDeviceRequest  true  "設備資訊"
// @Success      201     {object}  DeviceResponse
// @Failure      400     {object}  ErrorResponse
// @Failure      401     {object}  ErrorResponse
// @Failure      500     {object}  ErrorResponse
// @Router       /devices [post]
func (s *Server) HandleCreateDevice(w http.ResponseWriter, r *http.Request) {

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
// @Summary      獲取所有設備列表
// @Description  獲取系統中所有 IoT 設備的列表
// @Tags         devices
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Success      200  {array}   DeviceResponse
// @Failure      401  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Router       /devices [get]
func (s *Server) HandleListDevices(w http.ResponseWriter, r *http.Request) {
	devices, err := s.Store.ListDevices()
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to fetch devices")
		return
	}
	WriteJSON(w, http.StatusOK, devices)
}

// HandleGetDevice 取得單一設備
// @Summary      獲取單個設備
// @Description  根據設備 ID 獲取設備詳細資訊，包含遙測數據
// @Tags         devices
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      int  true  "設備 ID"
// @Success      200  {object}  DeviceResponse
// @Failure      400  {object}  ErrorResponse
// @Failure      401  {object}  ErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Router       /devices/{id} [get]
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

// CreateTelemetryRequest 創建遙測數據請求結構
type CreateTelemetryRequest struct {
	DeviceID   uint    `json:"device_id" example:"1" binding:"required"`
	DataType   string  `json:"data_type" example:"Temperature" binding:"required"`
	Value      float64 `json:"value" example:"25.5"`
	RecordedAt string  `json:"recorded_at,omitempty" example:"2024-01-01T00:00:00Z"` // 可選，如果沒有提供則使用當前時間
}

// HandleCreateTelemetry 處理建立遙測數據的請求
// @Summary      創建遙測數據
// @Description  為指定設備創建新的遙測數據記錄
// @Tags         telemetries
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        telemetry  body      CreateTelemetryRequest  true  "遙測數據資訊"
// @Success      201        {object}  TelemetryResponse
// @Failure      400        {object}  ErrorResponse
// @Failure      401        {object}  ErrorResponse
// @Failure      404        {object}  ErrorResponse
// @Failure      500        {object}  ErrorResponse
// @Router       /telemetries [post]
func (s *Server) HandleCreateTelemetry(w http.ResponseWriter, r *http.Request) {

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
	// 使用 deviceID 發布到具體的 topic (device:{id})
	s.Hub.BroadcastToDevice(telemetry.DeviceID, ToTelemetryResponse(*telemetry))
}

// HandleListTelemetries 處理取得所有遙測數據的請求
// @Summary      獲取所有遙測數據列表
// @Description  獲取系統中所有遙測數據的列表
// @Tags         telemetries
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Success      200  {array}  TelemetryResponse
// @Failure      401  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Router       /telemetries [get]
func (s *Server) HandleListTelemetries(w http.ResponseWriter, r *http.Request) {
	telemetries, err := s.Store.ListTelemetries()
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to fetch telemetries")
		return
	}
	WriteJSON(w, http.StatusOK, telemetries)
}

// HandleGetTelemetry 處理取得遙測數據的請求
// @Summary      獲取單個遙測數據
// @Description  根據遙測數據 ID 獲取詳細資訊
// @Tags         telemetries
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      int  true  "遙測數據 ID"
// @Success      200  {object}  TelemetryResponse
// @Failure      400  {object}  ErrorResponse
// @Failure      401  {object}  ErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Router       /telemetries/{id} [get]
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

// HandleDeleteDevice 處理刪除設備請求
// @Summary      刪除設備
// @Description  刪除指定設備及其所有相關的遙測數據
// @Tags         devices
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      int  true  "設備 ID"
// @Success      200  {object}  map[string]string
// @Failure      400  {object}  ErrorResponse
// @Failure      401  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Router       /devices/{id} [delete]
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

// UpdateDeviceRequest 更新設備請求結構
type UpdateDeviceRequest struct {
	Name       string `json:"name" example:"Temperature Sensor 1" binding:"required"`
	Type       string `json:"type" example:"Sensor"`
	MacAddress string `json:"mac_address" example:"00:11:22:33:44:55" binding:"required"`
	IsActive   bool   `json:"is_active" example:"true"`
}

// HandleUpdateDevice 處理更新整個設備的請求
// @Summary      完整更新設備
// @Description  使用 PUT 方法完整更新設備資訊（所有字段必須提供）
// @Tags         devices
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id      path      int                 true  "設備 ID"
// @Param        device  body      UpdateDeviceRequest  true  "設備資訊"
// @Success      200     {object}  DeviceResponse
// @Failure      400     {object}  ErrorResponse
// @Failure      401     {object}  ErrorResponse
// @Failure      404     {object}  ErrorResponse
// @Failure      500     {object}  ErrorResponse
// @Router       /devices/{id} [put]
func (s *Server) HandleUpdateDevice(w http.ResponseWriter, r *http.Request) {

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

// PatchDeviceRequest 部分更新設備請求結構
type PatchDeviceRequest struct {
	Name       *string `json:"name,omitempty" example:"Temperature Sensor 1"` // 使用指針，nil 表示不更新
	Type       *string `json:"type,omitempty" example:"Sensor"`
	MacAddress *string `json:"mac_address,omitempty" example:"00:11:22:33:44:55"`
	IsActive   *bool   `json:"is_active,omitempty" example:"true"`
}

// HandlePatchDevice 處理 PATCH 請求 - 部分更新（只需要提供要更新的字段）
// @Summary      部分更新設備
// @Description  使用 PATCH 方法部分更新設備資訊（只需提供要更新的字段）
// @Tags         devices
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id      path      int                 true  "設備 ID"
// @Param        device  body      PatchDeviceRequest  true  "要更新的設備欄位"
// @Success      200     {object}  DeviceResponse
// @Failure      400     {object}  ErrorResponse
// @Failure      401     {object}  ErrorResponse
// @Failure      404     {object}  ErrorResponse
// @Failure      500     {object}  ErrorResponse
// @Router       /devices/{id} [patch]
func (s *Server) HandlePatchDevice(w http.ResponseWriter, r *http.Request) {

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

// HandleAnalyzeDevice 處理設備分析請求
// @Summary      分析設備
// @Description  將設備分析任務加入佇列進行處理
// @Tags         devices
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      int  true  "設備 ID"
// @Success      202  {object}  map[string]string
// @Failure      400  {object}  ErrorResponse
// @Failure      401  {object}  ErrorResponse
// @Router       /devices/{id}/analyze [post]
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
	// 記錄 WebSocket 連線建立 metrics（手動記錄，因為繞過了 MetricsMiddleware）
	RequestCounter.WithLabelValues(r.Method, "/ws", "Switching Protocols").Inc()

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Printf("❌ WS 升級失敗: %v\n", err)
		return
	}

	// 註冊客戶端並預設訂閱所有 device:* 頻道
	s.Hub.AddClient(conn)

	// 保持連線，直到客戶端斷開
	defer s.Hub.RemoveClient(conn)

	// 處理客戶端訊息（訂閱/取消訂閱等）
	for {
		messageType, message, err := conn.ReadMessage()
		if err != nil {
			// 連接關閉或讀取錯誤
			break
		}

		// 處理文字訊息（JSON 格式的訂閱請求）
		if messageType == websocket.TextMessage {
			var msg map[string]interface{}
			if err := json.Unmarshal(message, &msg); err == nil {
				// 處理訂閱請求
				if action, ok := msg["action"].(string); ok {
					switch action {
					case "subscribe":
						if topic, ok := msg["topic"].(string); ok {
							s.Hub.Subscribe(topic, conn)
							// 回傳確認訊息
							response := map[string]interface{}{
								"status": "subscribed",
								"topic":  topic,
							}
							if data, err := json.Marshal(response); err == nil {
								conn.WriteMessage(websocket.TextMessage, data)
							}
						}
					case "unsubscribe":
						if topic, ok := msg["topic"].(string); ok {
							s.Hub.Unsubscribe(topic, conn)
							// 回傳確認訊息
							response := map[string]interface{}{
								"status": "unsubscribed",
								"topic":  topic,
							}
							if data, err := json.Marshal(response); err == nil {
								conn.WriteMessage(websocket.TextMessage, data)
							}
						}
					}
				}
			}
		}
	}
}

// PatchTelemetryRequest 部分更新遙測數據請求結構
type PatchTelemetryRequest struct {
	DeviceID   *uint    `json:"device_id,omitempty" example:"1"` // 使用指針，nil 表示不更新
	DataType   *string  `json:"data_type,omitempty" example:"Temperature"`
	Value      *float64 `json:"value,omitempty" example:"25.5"`
	RecordedAt *string  `json:"recorded_at,omitempty" example:"2024-01-01T00:00:00Z"`
}

// HandlePatchTelemetry 處理 PATCH 請求 - 部分更新遙測數據（只需要提供要更新的字段）
// @Summary      部分更新遙測數據
// @Description  使用 PATCH 方法部分更新遙測數據資訊（只需提供要更新的字段）
// @Tags         telemetries
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id         path      int                   true  "遙測數據 ID"
// @Param        telemetry  body      PatchTelemetryRequest  true  "要更新的遙測數據欄位"
// @Success      200        {object}  TelemetryResponse
// @Failure      400        {object}  ErrorResponse
// @Failure      401        {object}  ErrorResponse
// @Failure      404        {object}  ErrorResponse
// @Failure      500        {object}  ErrorResponse
// @Router       /telemetries/{id} [patch]
func (s *Server) HandlePatchTelemetry(w http.ResponseWriter, r *http.Request) {

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
	// 使用 deviceID 發布到具體的 topic (device:{id})
	s.Hub.BroadcastToDevice(updatedTelemetry.DeviceID, ToTelemetryResponse(*updatedTelemetry))
}

// Handle Get telemetry
