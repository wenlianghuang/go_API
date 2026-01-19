package errors

import (
	"fmt"
	"net/http"
)

// NotFoundError 资源未找到错误
type NotFoundError struct {
	*BaseError
	Resource string
	ID       interface{}
}

// NewNotFoundError 创建资源未找到错误
func NewNotFoundError(resource string, id interface{}) *NotFoundError {
	details := map[string]interface{}{
		"resource": resource,
	}
	if id != nil {
		details["id"] = id
	}

	return &NotFoundError{
		BaseError: NewBaseError(
			ErrCodeNotFound,
			http.StatusNotFound,
			fmt.Sprintf("%s not found", resource),
			details,
			nil,
		),
		Resource: resource,
		ID:       id,
	}
}

// ValidationError 验证错误
type ValidationError struct {
	*BaseError
	Field   string
	Reason  string
}

// NewValidationError 创建验证错误
func NewValidationError(field, reason string) *ValidationError {
	details := map[string]interface{}{
		"field":  field,
		"reason": reason,
	}

	return &ValidationError{
		BaseError: NewBaseError(
			ErrCodeValidation,
			http.StatusBadRequest,
			fmt.Sprintf("Validation failed for field '%s': %s", field, reason),
			details,
			nil,
		),
		Field:  field,
		Reason: reason,
	}
}

// UnauthorizedError 未授权错误
type UnauthorizedError struct {
	*BaseError
	Reason string
}

// NewUnauthorizedError 创建未授权错误
func NewUnauthorizedError(reason string) *UnauthorizedError {
	details := map[string]interface{}{
		"reason": reason,
	}

	return &UnauthorizedError{
		BaseError: NewBaseError(
			ErrCodeUnauthorized,
			http.StatusUnauthorized,
			reason,
			details,
			nil,
		),
		Reason: reason,
	}
}

// ForbiddenError 禁止访问错误
type ForbiddenError struct {
	*BaseError
	Reason string
}

// NewForbiddenError 创建禁止访问错误
func NewForbiddenError(reason string) *ForbiddenError {
	details := map[string]interface{}{
		"reason": reason,
	}

	return &ForbiddenError{
		BaseError: NewBaseError(
			ErrCodeForbidden,
			http.StatusForbidden,
			reason,
			details,
			nil,
		),
		Reason: reason,
	}
}

// ConflictError 冲突错误（资源已存在等）
type ConflictError struct {
	*BaseError
	Resource string
	Field    string
	Value    interface{}
}

// NewConflictError 创建冲突错误
func NewConflictError(resource, field string, value interface{}) *ConflictError {
	details := map[string]interface{}{
		"resource": resource,
		"field":    field,
		"value":    value,
	}

	return &ConflictError{
		BaseError: NewBaseError(
			ErrCodeConflict,
			http.StatusConflict,
			fmt.Sprintf("%s with %s '%v' already exists", resource, field, value),
			details,
			nil,
		),
		Resource: resource,
		Field:    field,
		Value:    value,
	}
}

// InternalError 内部服务器错误
type InternalError struct {
	*BaseError
	Operation string
	Err       error
}

// NewInternalError 创建内部服务器错误
func NewInternalError(operation string, err error) *InternalError {
	details := map[string]interface{}{
		"operation": operation,
	}
	if err != nil {
		details["error"] = err.Error()
	}

	return &InternalError{
		BaseError: NewBaseError(
			ErrCodeInternal,
			http.StatusInternalServerError,
			fmt.Sprintf("Internal server error during %s", operation),
			details,
			err,
		),
		Operation: operation,
		Err:       err,
	}
}

// BadRequestError 错误请求
type BadRequestError struct {
	*BaseError
	Reason string
}

// NewBadRequestError 创建错误请求
func NewBadRequestError(reason string) *BadRequestError {
	details := map[string]interface{}{
		"reason": reason,
	}

	return &BadRequestError{
		BaseError: NewBaseError(
			ErrCodeBadRequest,
			http.StatusBadRequest,
			reason,
			details,
			nil,
		),
		Reason: reason,
	}
}
