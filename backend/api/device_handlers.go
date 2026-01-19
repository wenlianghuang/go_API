package api

import (
	"net/http"
	"strconv"
	"strings"

	"my-api/model"
	"my-api/service"

	"github.com/go-chi/chi/v5"
)

// ==========================================
// 設備相關的 Handlers
// ==========================================

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

	// 使用新的驗證工具
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

	// 判斷是否應該使用預設的 IsActive 值
	// 如果 URL query 參數中沒有 "is_active" 且請求體中的 IsActive 是 false，則使用預設值 true
	defaultIsActive := !r.URL.Query().Has("is_active") && !req.IsActive

	// 調用 service 執行業務邏輯
	result, err := s.DeviceService.CreateDevice(r.Context(), service.CreateDeviceInput{
		Name:       req.Name,
		Type:       req.Type,
		MacAddress: req.MacAddress,
		IsActive:   req.IsActive,
		UserID:     userID,
	}, defaultIsActive)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// 返回響應
	WriteJSON(w, http.StatusCreated, result.Device)
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
	devices, err := s.Store.ListDevices(r.Context())
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

	device, err := s.Store.GetDeviceByID(r.Context(), uint(id))
	if err != nil {
		WriteError(w, http.StatusNotFound, "Device not found")
		return
	}
	// 2. 🔥 【關鍵步驟】 將 GORM Model 轉換為 DTO
	resp := ToDeviceResponse(device)
	//WriteJSON(w, http.StatusOK, device)
	WriteJSON(w, http.StatusOK, resp)
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

	// 2. 解析請求體並驗證
	var req UpdateDeviceRequest
	if err := ValidateAndDecode(r, &req); err != nil {
		HandleValidationError(w, err)
		return
	}

	// 4. 驗證設備是否存在
	_, err = s.Store.GetDeviceByID(r.Context(), uint(id))
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
	if err := s.Store.UpdateDevice(r.Context(), uint(id), device); err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to update device: "+err.Error())
		return
	}

	// 7. 回傳更新後的設備資訊
	updatedDevice, err := s.Store.GetDeviceByID(r.Context(), uint(id))
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to fetch updated device")
		return
	}

	resp := ToDeviceResponse(updatedDevice)
	WriteJSON(w, http.StatusOK, resp)
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

	// 2. 解析請求體並驗證
	var req PatchDeviceRequest
	if err := ValidateAndDecode(r, &req); err != nil {
		HandleValidationError(w, err)
		return
	}

	// 3. 調用 service 執行業務邏輯
	result, err := s.DeviceService.PatchDevice(r.Context(), uint(id), service.PatchDeviceInput{
		Name:       req.Name,
		Type:       req.Type,
		MacAddress: req.MacAddress,
		IsActive:   req.IsActive,
	})
	if err != nil {
		// 處理錯誤並轉換為適當的 HTTP 狀態碼
		errMsg := err.Error()
		if strings.Contains(errMsg, "device not found") {
			WriteError(w, http.StatusNotFound, "Device not found")
			return
		}
		if strings.Contains(errMsg, "at least one field must be provided") {
			WriteError(w, http.StatusBadRequest, "At least one field must be provided for update")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// 4. 構建響應並返回
	resp := ToDeviceResponse(result.Device)
	WriteJSON(w, http.StatusOK, resp)
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
	if err := s.Store.DeleteDeviceWithAllData(r.Context(), uint(id)); err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to delete device: "+err.Error())
		return
	}

	// 3. 回傳成功訊息
	WriteJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
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
