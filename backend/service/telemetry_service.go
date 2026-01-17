package service

import (
	"fmt"
	"my-api/model"
	"my-api/store"
	"time"
)

// TelemetryService 處理遙測數據相關的業務邏輯
type TelemetryService struct {
	store store.Storage
}

// NewTelemetryService 創建一個新的 TelemetryService 實例
func NewTelemetryService(store store.Storage) *TelemetryService {
	return &TelemetryService{
		store: store,
	}
}

// CreateTelemetryInput 創建遙測數據輸入參數
type CreateTelemetryInput struct {
	DeviceID   uint
	DataType   string
	Value      float64
	RecordedAt string // 可選，RFC3339 格式的時間字符串
}

// CreateTelemetryResult 創建遙測數據結果
type CreateTelemetryResult struct {
	Telemetry *model.Telemetry
}

// CreateTelemetry 處理創建遙測數據業務邏輯
// 返回 CreateTelemetryResult 和錯誤
func (s *TelemetryService) CreateTelemetry(input CreateTelemetryInput) (*CreateTelemetryResult, error) {
	// 驗證設備是否存在
	_, err := s.store.GetDeviceByID(input.DeviceID)
	if err != nil {
		return nil, fmt.Errorf("device with ID %d not found", input.DeviceID)
	}

	// 處理時間戳
	recordedAt := time.Now()
	if input.RecordedAt != "" {
		parsedTime, err := time.Parse(time.RFC3339, input.RecordedAt)
		if err != nil {
			// 如果解析失敗，使用當前時間
			recordedAt = time.Now()
		} else {
			recordedAt = parsedTime
		}
	}

	// 轉換成 Domain Model
	telemetry := &model.Telemetry{
		DeviceID:   input.DeviceID,
		DataType:   input.DataType,
		Value:      input.Value,
		RecordedAt: recordedAt,
	}

	// 呼叫資料庫層
	if err := s.store.AddTelemetry(telemetry); err != nil {
		return nil, fmt.Errorf("failed to create telemetry: %w", err)
	}

	// 構建結果
	result := &CreateTelemetryResult{
		Telemetry: telemetry,
	}

	return result, nil
}

// PatchTelemetryInput 部分更新遙測數據輸入參數
type PatchTelemetryInput struct {
	DeviceID   *uint
	DataType   *string
	Value      *float64
	RecordedAt *string // RFC3339 格式的時間字符串
}

// PatchTelemetryResult 部分更新遙測數據結果
type PatchTelemetryResult struct {
	Telemetry *model.Telemetry
}

// PatchTelemetry 處理部分更新遙測數據業務邏輯
// 返回 PatchTelemetryResult 和錯誤
func (s *TelemetryService) PatchTelemetry(telemetryID uint, input PatchTelemetryInput) (*PatchTelemetryResult, error) {
	// 驗證遙測數據是否存在
	_, err := s.store.GetTelemetryByID(telemetryID)
	if err != nil {
		return nil, fmt.Errorf("telemetry not found")
	}

	// 構建更新映射（只包含提供的字段）
	updates := make(map[string]interface{})
	if input.DeviceID != nil {
		// 驗證設備是否存在
		_, err := s.store.GetDeviceByID(*input.DeviceID)
		if err != nil {
			return nil, fmt.Errorf("device with ID %d not found", *input.DeviceID)
		}
		updates["device_id"] = *input.DeviceID
	}
	if input.DataType != nil {
		updates["data_type"] = *input.DataType
	}
	if input.Value != nil {
		updates["value"] = *input.Value
	}
	if input.RecordedAt != nil {
		parsedTime, err := time.Parse(time.RFC3339, *input.RecordedAt)
		if err != nil {
			return nil, fmt.Errorf("invalid recorded_at format, expected RFC3339: %w", err)
		}
		updates["recorded_at"] = parsedTime
	}

	// 如果沒有任何更新字段
	if len(updates) == 0 {
		return nil, fmt.Errorf("at least one field must be provided for update")
	}

	// 執行部分更新
	if err := s.store.PatchTelemetry(telemetryID, updates); err != nil {
		return nil, fmt.Errorf("failed to update telemetry: %w", err)
	}

	// 獲取更新後的遙測數據資訊
	updatedTelemetry, err := s.store.GetTelemetryByID(telemetryID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch updated telemetry: %w", err)
	}

	// 構建結果
	result := &PatchTelemetryResult{
		Telemetry: updatedTelemetry,
	}

	return result, nil
}
