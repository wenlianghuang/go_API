package api

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"my-api/service"

	"github.com/go-chi/chi/v5"
)

// ==========================================
// 遙測數據相關的 Handlers
// ==========================================

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

	// 使用新的驗證工具
	if err := ValidateAndDecode(r, &req); err != nil {
		HandleValidationError(w, err)
		return
	}

	// 調用 service 執行業務邏輯
	result, err := s.TelemetryService.CreateTelemetry(r.Context(), service.CreateTelemetryInput{
		DeviceID:   req.DeviceID,
		DataType:   req.DataType,
		Value:      req.Value,
		RecordedAt: req.RecordedAt,
	})
	if err != nil {
		// 處理錯誤並轉換為適當的 HTTP 狀態碼
		errMsg := err.Error()
		if strings.Contains(errMsg, "device with ID") && strings.Contains(errMsg, "not found") {
			WriteError(w, http.StatusNotFound, fmt.Sprintf("Device with ID %d not found", req.DeviceID))
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// 返回響應
	WriteJSON(w, http.StatusCreated, result.Telemetry)
	// 使用 deviceID 發布到具體的 topic (device:{id}) - WebSocket 廣播保留在 handler 層
	s.Hub.BroadcastToDevice(result.Telemetry.DeviceID, ToTelemetryResponse(*result.Telemetry))
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
	telemetries, err := s.Store.ListTelemetries(r.Context())
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

	telemetry, err := s.Store.GetTelemetryByID(r.Context(), uint(id))
	if err != nil {
		WriteError(w, http.StatusNotFound, "Telemetry not found")
		return
	}
	WriteJSON(w, http.StatusOK, telemetry)
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

	// 2. 解析請求體並驗證
	var req PatchTelemetryRequest
	if err := ValidateAndDecode(r, &req); err != nil {
		HandleValidationError(w, err)
		return
	}

	// 3. 調用 service 執行業務邏輯
	result, err := s.TelemetryService.PatchTelemetry(r.Context(), uint(id), service.PatchTelemetryInput{
		DeviceID:   req.DeviceID,
		DataType:   req.DataType,
		Value:      req.Value,
		RecordedAt: req.RecordedAt,
	})
	if err != nil {
		// 處理錯誤並轉換為適當的 HTTP 狀態碼
		errMsg := err.Error()
		if strings.Contains(errMsg, "telemetry not found") {
			WriteError(w, http.StatusNotFound, "Telemetry not found")
			return
		}
		if strings.Contains(errMsg, "device with ID") && strings.Contains(errMsg, "not found") {
			// 提取設備 ID
			var deviceID uint
			if req.DeviceID != nil {
				deviceID = *req.DeviceID
			}
			WriteError(w, http.StatusNotFound, fmt.Sprintf("Device with ID %d not found", deviceID))
			return
		}
		if strings.Contains(errMsg, "invalid recorded_at format") {
			WriteError(w, http.StatusBadRequest, "Invalid recorded_at format, expected RFC3339")
			return
		}
		if strings.Contains(errMsg, "at least one field must be provided") {
			WriteError(w, http.StatusBadRequest, "At least one field must be provided for update")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// 4. 返回響應
	WriteJSON(w, http.StatusOK, result.Telemetry)
	// 使用 deviceID 發布到具體的 topic (device:{id}) - WebSocket 廣播保留在 handler 層
	s.Hub.BroadcastToDevice(result.Telemetry.DeviceID, ToTelemetryResponse(*result.Telemetry))
}

// HandleDeleteTelemetry 處理刪除遙測數據的請求
// @Summary      刪除遙測數據
// @Description  根據遙測數據 ID 刪除指定的遙測數據記錄
// @Tags         telemetries
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      int  true  "遙測數據 ID"
// @Success      200  {object}  map[string]string
// @Failure      400  {object}  ErrorResponse
// @Failure      401  {object}  ErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Router       /telemetries/{id} [delete]
func (s *Server) HandleDeleteTelemetry(w http.ResponseWriter, r *http.Request) {
	// 1. 解析 ID
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid telemetry ID")
		return
	}

	// 2. 調用 service 執行業務邏輯
	err = s.TelemetryService.DeleteTelemetry(r.Context(), uint(id))
	if err != nil {
		// 處理錯誤並轉換為適當的 HTTP 狀態碼
		errMsg := err.Error()
		if strings.Contains(errMsg, "telemetry not found") {
			WriteError(w, http.StatusNotFound, "Telemetry not found")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// 3. 返回成功響應
	WriteJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}
