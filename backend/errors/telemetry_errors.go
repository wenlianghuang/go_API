package errors

import (
	"fmt"
	"net/http"
)

// TelemetryNotFoundError 遥测数据未找到错误
type TelemetryNotFoundError struct {
	*BaseError
	TelemetryID uint
}

// NewTelemetryNotFoundError 创建遥测数据未找到错误
func NewTelemetryNotFoundError(telemetryID uint) *TelemetryNotFoundError {
	details := map[string]interface{}{
		"telemetry_id": telemetryID,
	}

	return &TelemetryNotFoundError{
		BaseError: NewBaseError(
			ErrCodeTelemetryNotFound,
			http.StatusNotFound,
			fmt.Sprintf("Telemetry with ID %d not found", telemetryID),
			details,
			nil,
		),
		TelemetryID: telemetryID,
	}
}

// DeviceMismatchError 设备不匹配错误
type DeviceMismatchError struct {
	*BaseError
	TelemetryID      uint
	DeviceID         uint
	ExpectedDeviceID uint
}

// NewDeviceMismatchError 创建设备不匹配错误
func NewDeviceMismatchError(telemetryID uint, deviceID, expectedDeviceID uint) *DeviceMismatchError {
	details := map[string]interface{}{
		"telemetry_id":       telemetryID,
		"device_id":          deviceID,
		"expected_device_id": expectedDeviceID,
	}

	return &DeviceMismatchError{
		BaseError: NewBaseError(
			ErrCodeDeviceMismatch,
			http.StatusBadRequest,
			fmt.Sprintf("Device ID %d does not match expected device ID %d for telemetry %d", deviceID, expectedDeviceID, telemetryID),
			details,
			nil,
		),
		TelemetryID:      telemetryID,
		DeviceID:         deviceID,
		ExpectedDeviceID: expectedDeviceID,
	}
}

// TelemetryCreateFailedError 遥测数据创建失败错误
type TelemetryCreateFailedError struct {
	*BaseError
	Reason string
	Err    error
}

// NewTelemetryCreateFailedError 创建遥测数据创建失败错误
func NewTelemetryCreateFailedError(reason string, err error) *TelemetryCreateFailedError {
	details := map[string]interface{}{
		"reason": reason,
	}
	if err != nil {
		details["error"] = err.Error()
	}

	return &TelemetryCreateFailedError{
		BaseError: NewBaseError(
			ErrCodeTelemetryCreateFailed,
			http.StatusInternalServerError,
			fmt.Sprintf("Failed to create telemetry: %s", reason),
			details,
			err,
		),
		Reason: reason,
		Err:    err,
	}
}

// TelemetryUpdateFailedError 遥测数据更新失败错误
type TelemetryUpdateFailedError struct {
	*BaseError
	TelemetryID uint
	Reason      string
	Err         error
}

// NewTelemetryUpdateFailedError 创建遥测数据更新失败错误
func NewTelemetryUpdateFailedError(telemetryID uint, reason string, err error) *TelemetryUpdateFailedError {
	details := map[string]interface{}{
		"telemetry_id": telemetryID,
		"reason":       reason,
	}
	if err != nil {
		details["error"] = err.Error()
	}

	return &TelemetryUpdateFailedError{
		BaseError: NewBaseError(
			ErrCodeTelemetryUpdateFailed,
			http.StatusInternalServerError,
			fmt.Sprintf("Failed to update telemetry %d: %s", telemetryID, reason),
			details,
			err,
		),
		TelemetryID: telemetryID,
		Reason:      reason,
		Err:         err,
	}
}

// TelemetryDeleteFailedError 遥测数据删除失败错误
type TelemetryDeleteFailedError struct {
	*BaseError
	TelemetryID uint
	Reason      string
	Err         error
}

// NewTelemetryDeleteFailedError 创建遥测数据删除失败错误
func NewTelemetryDeleteFailedError(telemetryID uint, reason string, err error) *TelemetryDeleteFailedError {
	details := map[string]interface{}{
		"telemetry_id": telemetryID,
		"reason":       reason,
	}
	if err != nil {
		details["error"] = err.Error()
	}

	return &TelemetryDeleteFailedError{
		BaseError: NewBaseError(
			ErrCodeTelemetryDeleteFailed,
			http.StatusInternalServerError,
			fmt.Sprintf("Failed to delete telemetry %d: %s", telemetryID, reason),
			details,
			err,
		),
		TelemetryID: telemetryID,
		Reason:      reason,
		Err:         err,
	}
}

// InvalidTimeFormatError 无效时间格式错误
type InvalidTimeFormatError struct {
	*BaseError
	TimeString string
	Format     string
}

// NewInvalidTimeFormatError 创建无效时间格式错误
func NewInvalidTimeFormatError(timeString, format string) *InvalidTimeFormatError {
	details := map[string]interface{}{
		"time_string": timeString,
		"format":      format,
	}

	return &InvalidTimeFormatError{
		BaseError: NewBaseError(
			ErrCodeInvalidTimeFormat,
			http.StatusBadRequest,
			fmt.Sprintf("Invalid time format '%s', expected format: %s", timeString, format),
			details,
			nil,
		),
		TimeString: timeString,
		Format:     format,
	}
}
