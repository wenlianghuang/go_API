package api

import (
	"my-api/model"
	"time"
)

type TelemetryResponse struct {
	ID        uint      `json:"id"`
	DataType  string    `json:"data_type"`
	Value     float64   `json:"value"`
	Timestamp time.Time `json:"recorded_at"`
}

type DeviceResponse struct {
	ID         uint   `json:"id"`
	Name       string `json:"name"`
	Type       string `json:"type"`
	MacAddress string `json:"mac_address"`
	IsActive   bool   `json:"is_active"`
	// 這裡只放入 DTO 版本的 Telemetry
	Telemetries []TelemetryResponse `json:"telemetries"`
}

// ==========================================
// 轉換函式 (Mapper Functions)
// ==========================================

// ToTelemetryResponse 負責將內部模型轉換為外部 DTO
func ToTelemetryResponse(t model.Telemetry) TelemetryResponse {
	return TelemetryResponse{
		ID:        t.ID,
		DataType:  t.DataType,
		Value:     t.Value,
		Timestamp: t.RecordedAt, // 注意：我們將 model.Telemetry 中的 RecordedAt 字段 (儘管在你的模型中未定義，我們假設它存在或使用 CreatedAt) 映射到 Timestamp
	}
}

// ToDeviceResponse 負責將內部模型轉換為外部 DTO
func ToDeviceResponse(d *model.Device) DeviceResponse {
	// 處理關聯數據：將 model.Telemetry 列表轉換為 TelemetryResponse 列表
	telemetryResponses := make([]TelemetryResponse, len(d.Telemetries))
	for i, t := range d.Telemetries {
		telemetryResponses[i] = ToTelemetryResponse(t)
	}

	return DeviceResponse{
		ID:          d.ID,
		Name:        d.Name,
		MacAddress:  d.MacAddress,
		IsActive:    d.IsActive,
		Type:        d.Type,
		Telemetries: telemetryResponses,
	}
}
