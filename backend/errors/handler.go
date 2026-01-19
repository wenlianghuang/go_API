package errors

import (
	"encoding/json"
	"net/http"
)

// ErrorResponse 错误响应结构
type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

// ErrorDetail 错误详情
type ErrorDetail struct {
	Code    string                 `json:"code"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
}

// HandleError 统一处理错误并写入HTTP响应
// 如果错误是 AppError 类型，使用其状态码和错误信息
// 否则返回 500 内部服务器错误
func HandleError(w http.ResponseWriter, err error) {
	var appErr AppError
	var ok bool

	// 尝试将错误转换为 AppError
	if appErr, ok = err.(AppError); !ok {
		// 如果不是 AppError，包装为内部错误
		appErr = NewInternalError("unknown operation", err)
	}

	// 构建错误响应
	response := ErrorResponse{
		Error: ErrorDetail{
			Code:    appErr.Code(),
			Message: appErr.Message(),
			Details: appErr.Details(),
		},
	}

	// 设置响应头
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(appErr.StatusCode())

	// 写入JSON响应
	json.NewEncoder(w).Encode(response)
}

// HandleValidationErrors 处理验证错误（用于 validator.ValidationErrors）
func HandleValidationErrors(w http.ResponseWriter, validationErrors interface{}) {
	// 这里可以扩展以支持验证错误的特殊处理
	// 目前使用通用的错误处理
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
