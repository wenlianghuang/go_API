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
	Name       string `gorm:"size:255;not null"`
	Type       string `gorm:"size:50"`                     // e.g., "Sensor", "Camera"
	MacAddress string `gorm:"uniqueIndex;not null"` // 唯一索引
	IsActive   bool   `gorm:"default:true"`
	UserID     string // Foreign key to link to a User

	// 這裡展示 GORM 的關聯：一對多 (One Device has many Telemetry data)
	Telemetries []Telemetry `gorm:"foreignKey:DeviceID"`
}

// Telemetry 代表設備傳回的數據點
type Telemetry struct {
	gorm.Model
	DeviceID   uint      // 外鍵
	DataType   string    // e.g., "Temperature", "Humidity"
	Value      float64   
	RecordedAt time.Time 
}
