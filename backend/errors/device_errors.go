package errors

import (
	"fmt"
	"net/http"
)

// DeviceNotFoundError 设备未找到错误
type DeviceNotFoundError struct {
	*BaseError
	DeviceID uint
}

// NewDeviceNotFoundError 创建设备未找到错误
func NewDeviceNotFoundError(deviceID uint) *DeviceNotFoundError {
	details := map[string]interface{}{
		"device_id": deviceID,
	}

	return &DeviceNotFoundError{
		BaseError: NewBaseError(
			ErrCodeDeviceNotFound,
			http.StatusNotFound,
			fmt.Sprintf("Device with ID %d not found", deviceID),
			details,
			nil,
		),
		DeviceID: deviceID,
	}
}

// DeviceCreateFailedError 设备创建失败错误
type DeviceCreateFailedError struct {
	*BaseError
	Reason string
	Err    error
}

// NewDeviceCreateFailedError 创建设备创建失败错误
func NewDeviceCreateFailedError(reason string, err error) *DeviceCreateFailedError {
	details := map[string]interface{}{
		"reason": reason,
	}
	if err != nil {
		details["error"] = err.Error()
	}

	return &DeviceCreateFailedError{
		BaseError: NewBaseError(
			ErrCodeDeviceCreateFailed,
			http.StatusInternalServerError,
			fmt.Sprintf("Failed to create device: %s", reason),
			details,
			err,
		),
		Reason: reason,
		Err:    err,
	}
}

// DeviceUpdateFailedError 设备更新失败错误
type DeviceUpdateFailedError struct {
	*BaseError
	DeviceID uint
	Reason   string
	Err      error
}

// NewDeviceUpdateFailedError 创建设备更新失败错误
func NewDeviceUpdateFailedError(deviceID uint, reason string, err error) *DeviceUpdateFailedError {
	details := map[string]interface{}{
		"device_id": deviceID,
		"reason":    reason,
	}
	if err != nil {
		details["error"] = err.Error()
	}

	return &DeviceUpdateFailedError{
		BaseError: NewBaseError(
			ErrCodeDeviceUpdateFailed,
			http.StatusInternalServerError,
			fmt.Sprintf("Failed to update device %d: %s", deviceID, reason),
			details,
			err,
		),
		DeviceID: deviceID,
		Reason:   reason,
		Err:      err,
	}
}

// DeviceDeleteFailedError 设备删除失败错误
type DeviceDeleteFailedError struct {
	*BaseError
	DeviceID uint
	Reason   string
	Err      error
}

// NewDeviceDeleteFailedError 创建设备删除失败错误
func NewDeviceDeleteFailedError(deviceID uint, reason string, err error) *DeviceDeleteFailedError {
	details := map[string]interface{}{
		"device_id": deviceID,
		"reason":    reason,
	}
	if err != nil {
		details["error"] = err.Error()
	}

	return &DeviceDeleteFailedError{
		BaseError: NewBaseError(
			ErrCodeDeviceDeleteFailed,
			http.StatusInternalServerError,
			fmt.Sprintf("Failed to delete device %d: %s", deviceID, reason),
			details,
			err,
		),
		DeviceID: deviceID,
		Reason:   reason,
		Err:      err,
	}
}

// InvalidDeviceIDError 无效设备ID错误
type InvalidDeviceIDError struct {
	*BaseError
	DeviceID string
}

// NewInvalidDeviceIDError 创建无效设备ID错误
func NewInvalidDeviceIDError(deviceID string) *InvalidDeviceIDError {
	details := map[string]interface{}{
		"device_id": deviceID,
	}

	return &InvalidDeviceIDError{
		BaseError: NewBaseError(
			ErrCodeInvalidDeviceID,
			http.StatusBadRequest,
			fmt.Sprintf("Invalid device ID: %s", deviceID),
			details,
			nil,
		),
		DeviceID: deviceID,
	}
}
