package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-playground/validator/v10"
)

// 全局验证器实例
var validate *validator.Validate

// 初始化验证器
func init() {
	validate = validator.New()

	validate.RegisterValidation("alphanum_underscore", func(fl validator.FieldLevel) bool {
		value := fl.Field().String()
		for _, char := range value {
			if !((char >= 'a' && char <= 'z') ||
				(char >= 'A' && char <= 'Z') ||
				(char >= '0' && char <= '9') ||
				char == '_' || char == '.' || char == ' ') {
				return false
			}
		}
		return true
	})
}

// ValidateStruct 验证结构体
func ValidateStruct(s interface{}) error {
	return validate.Struct(s)
}

// ValidationError 验证错误响应结构
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// ValidationErrorResponse 完整的验证错误响应
type ValidationErrorResponse struct {
	Error   string            `json:"error"`
	Details []ValidationError `json:"details"`
}

// FormatValidationErrors 格式化验证错误为友好的消息
func FormatValidationErrors(err error) []ValidationError {
	var errors []ValidationError

	if validationErrors, ok := err.(validator.ValidationErrors); ok {
		for _, e := range validationErrors {
			errors = append(errors, ValidationError{
				Field:   jsonFieldName(e.Field()),
				Message: formatErrorMessage(e),
			})
		}
	}

	return errors
}

// jsonFieldName 将结构体字段名转换为 JSON 字段名
// 简化版本：转换为小写并添加下划线
func jsonFieldName(field string) string {
	// 将驼峰命名转换为蛇形命名
	var result strings.Builder
	for i, r := range field {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result.WriteRune('_')
		}
		result.WriteRune(r)
	}
	return strings.ToLower(result.String())
}

// formatErrorMessage 根据验证标签格式化友好的错误消息
func formatErrorMessage(e validator.FieldError) string {
	field := e.Field()

	switch e.Tag() {
	case "required":
		return fmt.Sprintf("%s is required", field)
	case "email":
		return fmt.Sprintf("%s must be a valid email address", field)
	case "min":
		if e.Type().String() == "string" {
			return fmt.Sprintf("%s must be at least %s characters long", field, e.Param())
		}
		return fmt.Sprintf("%s must be at least %s", field, e.Param())
	case "max":
		if e.Type().String() == "string" {
			return fmt.Sprintf("%s must be at most %s characters long", field, e.Param())
		}
		return fmt.Sprintf("%s must be at most %s", field, e.Param())
	case "len":
		return fmt.Sprintf("%s must be exactly %s characters long", field, e.Param())
	case "alphanum":
		return fmt.Sprintf("%s must contain only alphanumeric characters", field)
	case "alpha":
		return fmt.Sprintf("%s must contain only letters", field)
	case "numeric":
		return fmt.Sprintf("%s must be a number", field)
	case "mac":
		return fmt.Sprintf("%s must be a valid MAC address", field)
	case "ip":
		return fmt.Sprintf("%s must be a valid IP address", field)
	case "url":
		return fmt.Sprintf("%s must be a valid URL", field)
	case "uuid":
		return fmt.Sprintf("%s must be a valid UUID", field)
	case "gt":
		return fmt.Sprintf("%s must be greater than %s", field, e.Param())
	case "gte":
		return fmt.Sprintf("%s must be greater than or equal to %s", field, e.Param())
	case "lt":
		return fmt.Sprintf("%s must be less than %s", field, e.Param())
	case "lte":
		return fmt.Sprintf("%s must be less than or equal to %s", field, e.Param())
	case "oneof":
		return fmt.Sprintf("%s must be one of: %s", field, e.Param())
	case "eqfield":
		return fmt.Sprintf("%s must be equal to %s", field, e.Param())
	case "datetime":
		return fmt.Sprintf("%s must be a valid datetime in format: %s", field, e.Param())
	case "alphanum_underscore":
		return fmt.Sprintf("%s must contain only alphanumeric characters and underscores", field)
	default:
		return fmt.Sprintf("%s is invalid", field)
	}
}

// ValidateAndDecode 解码 JSON 并验证请求体
// 这是一个辅助函数，将解码和验证合并为一步
func ValidateAndDecode(r *http.Request, v interface{}) error {
	// 1. 解码 JSON
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		return fmt.Errorf("invalid JSON format: %w", err)
	}

	// 2. 验证结构体
	if err := ValidateStruct(v); err != nil {
		return err
	}

	return nil
}

// WriteValidationError 写入格式化的验证错误响应
func WriteValidationError(w http.ResponseWriter, err error) {
	errors := FormatValidationErrors(err)

	response := ValidationErrorResponse{
		Error:   "Validation failed",
		Details: errors,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	json.NewEncoder(w).Encode(response)
}

// HandleValidationError 统一处理验证和解码错误
// 自动判断错误类型并返回适当的响应
func HandleValidationError(w http.ResponseWriter, err error) {
	if validationErr, ok := err.(validator.ValidationErrors); ok {
		// 验证错误
		WriteValidationError(w, validationErr)
	} else {
		// JSON 解码错误或其他错误
		WriteError(w, http.StatusBadRequest, "Invalid request format: "+err.Error())
	}
}
