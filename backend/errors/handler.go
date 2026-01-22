package errors

import (
	"encoding/json"
	"net/http"
)

// ErrorLogger 定義錯誤記錄的介面（避免直接依賴 zap）
// 實作者可以包裝任何 logger（zap, logrus, stdlib log 等）
type ErrorLogger interface {
	// LogError 記錄錯誤，level 由實作者決定
	// statusCode: HTTP 狀態碼（用於決定 log level）
	// fields: 錯誤相關的欄位（key-value pairs）
	LogError(statusCode int, fields map[string]interface{})
}

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
// 如果提供了 logger，會同時記錄錯誤到後端 log
// 如果錯誤是 AppError 類型，使用其狀態碼和錯誤資訊
// 否則返回 500 內部伺服器錯誤
func HandleError(w http.ResponseWriter, err error, logger ErrorLogger) {
	var appErr AppError
	var ok bool

	// 嘗試將錯誤轉換為 AppError
	if appErr, ok = err.(AppError); !ok {
		// 如果不是 AppError，包裝為內部錯誤
		appErr = NewInternalError("unknown operation", err)
	}

	// 🆕 記錄錯誤到後端 log（如果提供了 logger）
	if logger != nil {
		fields := map[string]interface{}{
			"error_code":    appErr.Code(),
			"error_message": appErr.Message(),
		}

		// 如果有錯誤詳情，加入
		if details := appErr.Details(); len(details) > 0 {
			fields["error_details"] = details
		}

		// 如果有原始錯誤（透過 Unwrap），加入
		if unwrapped := Unwrap(appErr); unwrapped != nil {
			fields["underlying_error"] = unwrapped.Error()
		}

		// 記錄錯誤（level 由 logger 實作根據 statusCode 決定）
		logger.LogError(appErr.StatusCode(), fields)
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

// Unwrap 輔助函數，用於取得被包裝的原始錯誤
func Unwrap(err error) error {
	type unwrapper interface {
		Unwrap() error
	}
	if u, ok := err.(unwrapper); ok {
		return u.Unwrap()
	}
	return nil
}

// HandleValidationErrors 處理驗證錯誤（用於 validator.ValidationErrors）
func HandleValidationErrors(w http.ResponseWriter, validationErrors interface{}, logger ErrorLogger) {
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

	HandleError(w, err, logger)
}
