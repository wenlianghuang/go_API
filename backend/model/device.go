package model

import (
	"time"

	"gorm.io/gorm"
)

// Device 代表一個實體 IoT 設備
type Device struct {
	// gorm.Model 會自動幫你加入 ID (uint), CreatedAt, UpdatedAt, DeletedAt
	gorm.Model

	// 使用 struct tags 定義資料庫欄位特性
	Name       string `json:"name" gorm:"size:255;not null"`
	Type       string `json:"type" gorm:"size:50"`                     // e.g., "Sensor", "Camera"
	MacAddress string `json:"mac_address" gorm:"uniqueIndex;not null"` // 唯一索引
	IsActive   bool   `json:"is_active" gorm:"default:true"`

	// 這裡展示 GORM 的關聯：一對多 (One Device has many Telemetry data)
	Telemetries []Telemetry `json:"telemetries,omitempty" gorm:"foreignKey:DeviceID"`
}

// Telemetry 代表設備傳回的數據點
type Telemetry struct {
	gorm.Model
	DeviceID   uint      `json:"device_id"` // 外鍵
	DataType   string    `json:"data_type"` // e.g., "Temperature", "Humidity"
	Value      float64   `json:"value"`
	RecordedAt time.Time `json:"recorded_at"`
}
