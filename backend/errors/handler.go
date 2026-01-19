package errors

import (
	"encoding/json"
	"net/http"
)

// ErrorResponse 錯誤響應結構
type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

// ErrorDetail 錯誤詳情
type ErrorDetail struct {
	Code    string                 `json:"code"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
}

// HandleError 統一處理錯誤並寫入HTTP響應
// 如果錯誤是 AppError 類型，使用其狀態碼和錯誤資訊
// 否則返回 500 內部伺服器錯誤
func HandleError(w http.ResponseWriter, err error) {
	var appErr AppError
	var ok bool

	// 嘗試將錯誤轉換為 AppError
	if appErr, ok = err.(AppError); !ok {
		// 如果不是 AppError，包裝為內部錯誤
		appErr = NewInternalError("unknown operation", err)
	}

	// 構建錯誤響應
	response := ErrorResponse{
		Error: ErrorDetail{
			Code:    appErr.Code(),
			Message: appErr.Message(),
			Details: appErr.Details(),
		},
	}

	// 設置響應頭
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(appErr.StatusCode())

	// 寫入JSON響應
	json.NewEncoder(w).Encode(response)
}

// HandleValidationErrors 處理驗證錯誤（用於 validator.ValidationErrors）
func HandleValidationErrors(w http.ResponseWriter, validationErrors interface{}) {
	// 這裡可以擴展以支援驗證錯誤的特殊處理
	// 目前使用通用的錯誤處理
	details := map[string]interface{}{
		"validation_errors": validationErrors,
	}

	err := NewBaseError(
		ErrCodeValidation,
		http.StatusBadRequest,
		"Validation failed",
		details,
		nil,
	)

	HandleError(w, err)
}
