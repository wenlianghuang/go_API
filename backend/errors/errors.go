package errors

import (
	"fmt"
)

// AppError 定义应用程序错误的接口
type AppError interface {
	error
	Code() string
	StatusCode() int
	Message() string
	Details() map[string]interface{}
}

// BaseError 基础错误结构体，实现 AppError 接口
type BaseError struct {
	code       string
	statusCode int
	message    string
	details    map[string]interface{}
	err        error // 原始错误（可选）
}

// Error 实现 error 接口
func (e *BaseError) Error() string {
	if e.err != nil {
		return fmt.Sprintf("%s: %v", e.message, e.err)
	}
	return e.message
}

// Code 返回错误码
func (e *BaseError) Code() string {
	return e.code
}

// StatusCode 返回HTTP状态码
func (e *BaseError) StatusCode() int {
	return e.statusCode
}

// Message 返回错误消息
func (e *BaseError) Message() string {
	return e.message
}

// Details 返回错误详情
func (e *BaseError) Details() map[string]interface{} {
	if e.details == nil {
		return make(map[string]interface{})
	}
	return e.details
}

// Unwrap 返回原始错误（用于 errors.Unwrap）
func (e *BaseError) Unwrap() error {
	return e.err
}

// NewBaseError 创建基础错误
func NewBaseError(code string, statusCode int, message string, details map[string]interface{}, err error) *BaseError {
	return &BaseError{
		code:       code,
		statusCode: statusCode,
		message:    message,
		details:    details,
		err:        err,
	}
}

// 错误码常量定义
const (
	// 通用错误码 (000-099)
	ErrCodeValidation      = "ERR_001"
	ErrCodeUnauthorized    = "ERR_002"
	ErrCodeNotFound        = "ERR_003"
	ErrCodeForbidden       = "ERR_004"
	ErrCodeConflict        = "ERR_005"
	ErrCodeInternal        = "ERR_006"
	ErrCodeBadRequest      = "ERR_007"
	ErrCodeTooManyRequests = "ERR_008"

	// 认证相关错误码 (100-199)
	ErrCodeUserExists         = "ERR_101"
	ErrCodeInvalidCredentials = "ERR_102"
	ErrCodeTokenExpired       = "ERR_103"
	ErrCodeTokenInvalid       = "ERR_104"
	ErrCodeUserNotFound       = "ERR_105"

	// 设备相关错误码 (200-299)
	ErrCodeDeviceNotFound     = "ERR_201"
	ErrCodeDeviceCreateFailed = "ERR_202"
	ErrCodeDeviceUpdateFailed = "ERR_203"
	ErrCodeDeviceDeleteFailed = "ERR_204"
	ErrCodeInvalidDeviceID    = "ERR_205"

	// 遥测数据相关错误码 (300-399)
	ErrCodeTelemetryNotFound     = "ERR_301"
	ErrCodeDeviceMismatch        = "ERR_302"
	ErrCodeTelemetryCreateFailed = "ERR_303"
	ErrCodeTelemetryUpdateFailed = "ERR_304"
	ErrCodeTelemetryDeleteFailed = "ERR_305"
	ErrCodeInvalidTimeFormat     = "ERR_306"
)
