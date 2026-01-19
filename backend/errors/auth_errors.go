package errors

import (
	"fmt"
	"net/http"
)

// UserExistsError 用户已存在错误
type UserExistsError struct {
	*BaseError
	Email string
}

// NewUserExistsError 创建用户已存在错误
func NewUserExistsError(email string) *UserExistsError {
	details := map[string]interface{}{
		"email": email,
	}

	return &UserExistsError{
		BaseError: NewBaseError(
			ErrCodeUserExists,
			http.StatusConflict,
			fmt.Sprintf("User with email '%s' already exists", email),
			details,
			nil,
		),
		Email: email,
	}
}

// InvalidCredentialsError 无效凭据错误
type InvalidCredentialsError struct {
	*BaseError
}

// NewInvalidCredentialsError 创建无效凭据错误
func NewInvalidCredentialsError() *InvalidCredentialsError {
	return &InvalidCredentialsError{
		BaseError: NewBaseError(
			ErrCodeInvalidCredentials,
			http.StatusUnauthorized,
			"Invalid email or password",
			map[string]interface{}{},
			nil,
		),
	}
}

// TokenExpiredError Token过期错误
type TokenExpiredError struct {
	*BaseError
}

// NewTokenExpiredError 创建Token过期错误
func NewTokenExpiredError() *TokenExpiredError {
	return &TokenExpiredError{
		BaseError: NewBaseError(
			ErrCodeTokenExpired,
			http.StatusUnauthorized,
			"Token has expired",
			map[string]interface{}{},
			nil,
		),
	}
}

// TokenInvalidError Token无效错误
type TokenInvalidError struct {
	*BaseError
	Reason string
}

// NewTokenInvalidError 创建Token无效错误
func NewTokenInvalidError(reason string) *TokenInvalidError {
	details := map[string]interface{}{
		"reason": reason,
	}

	return &TokenInvalidError{
		BaseError: NewBaseError(
			ErrCodeTokenInvalid,
			http.StatusUnauthorized,
			fmt.Sprintf("Invalid token: %s", reason),
			details,
			nil,
		),
		Reason: reason,
	}
}

// UserNotFoundError 用户未找到错误（认证相关）
type UserNotFoundError struct {
	*BaseError
	Email string
}

// NewUserNotFoundError 创建用户未找到错误
func NewUserNotFoundError(email string) *UserNotFoundError {
	details := map[string]interface{}{
		"email": email,
	}

	return &UserNotFoundError{
		BaseError: NewBaseError(
			ErrCodeUserNotFound,
			http.StatusNotFound,
			fmt.Sprintf("User with email '%s' not found", email),
			details,
			nil,
		),
		Email: email,
	}
}
