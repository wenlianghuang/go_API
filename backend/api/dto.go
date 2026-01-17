package api

import (
	"my-api/model"
	"time"
)

// ==========================================
// 認證相關的 DTO
// ==========================================

// RegisterRequest 註冊請求結構
type RegisterRequest struct {
	Username string `json:"username" validate:"required,min=3,max=50,alphanum_underscore" example:"john_doe"`
	Email    string `json:"email" validate:"required,email" example:"john@example.com"`
	Password string `json:"password" validate:"required,min=6,max=100" example:"SecurePassword123"`
}

// LoginRequest 登入請求結構
type LoginRequest struct {
	Email    string `json:"email" validate:"required,email" example:"john@example.com"`
	Password string `json:"password" validate:"required,min=6" example:"SecurePassword123"`
}

// AuthResponse 認證響應結構（包含 JWT token）
type AuthResponse struct {
	Token     string       `json:"token"`
	ExpiresAt time.Time    `json:"expires_at"`
	User      UserResponse `json:"user"`
}

// ==========================================
// 使用者相關的 DTO
// ==========================================

// CreateUserRequest 創建用戶請求結構（已廢棄，請使用 /auth/register）
type CreateUserRequest struct {
	Username string `json:"username" validate:"required,min=3,max=50" example:"john_doe"`
	Email    string `json:"email" validate:"required,email" example:"john@example.com"`
}

// UserResponse 使用者響應結構
type UserResponse struct {
	ID        string    `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

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
// 設備相關的 Request DTOs
// ==========================================

// CreateDeviceRequest 創建設備請求結構
type CreateDeviceRequest struct {
	Name       string `json:"name" validate:"required,min=1,max=100" example:"Temperature Sensor 1"`
	Type       string `json:"type" validate:"omitempty" example:"Sensor"`
	MacAddress string `json:"mac_address" validate:"required,mac" example:"00:11:22:33:44:55"`
	IsActive   bool   `json:"is_active" example:"true"`
}

// UpdateDeviceRequest 更新設備請求結構
type UpdateDeviceRequest struct {
	Name       string `json:"name" validate:"required,min=1,max=100" example:"Temperature Sensor 1"`
	Type       string `json:"type" validate:"omitempty" example:"Sensor"`
	MacAddress string `json:"mac_address" validate:"required,mac" example:"00:11:22:33:44:55"`
	IsActive   bool   `json:"is_active" example:"true"`
}

// PatchDeviceRequest 部分更新設備請求結構
type PatchDeviceRequest struct {
	Name       *string `json:"name,omitempty" validate:"omitempty,min=1,max=100" example:"Temperature Sensor 1"` // 使用指針，nil 表示不更新
	Type       *string `json:"type,omitempty" validate:"omitempty" example:"Sensor"`
	MacAddress *string `json:"mac_address,omitempty" validate:"omitempty,mac" example:"00:11:22:33:44:55"`
	IsActive   *bool   `json:"is_active,omitempty" example:"true"`
}

// ==========================================
// 遙測數據相關的 Request DTOs
// ==========================================

// CreateTelemetryRequest 創建遙測數據請求結構
type CreateTelemetryRequest struct {
	DeviceID   uint    `json:"device_id" validate:"required,gt=0" example:"1"`
	DataType   string  `json:"data_type" validate:"required,min=1,max=50" example:"Temperature"`
	Value      float64 `json:"value" validate:"required" example:"25.5"`
	RecordedAt string  `json:"recorded_at,omitempty" validate:"omitempty,datetime=2006-01-02T15:04:05Z07:00" example:"2024-01-01T00:00:00Z"` // 可選，如果沒有提供則使用當前時間
}

// PatchTelemetryRequest 部分更新遙測數據請求結構
type PatchTelemetryRequest struct {
	DeviceID   *uint    `json:"device_id,omitempty" validate:"omitempty,gt=0" example:"1"` // 使用指針，nil 表示不更新
	DataType   *string  `json:"data_type,omitempty" validate:"omitempty,min=1,max=50" example:"Temperature"`
	Value      *float64 `json:"value,omitempty" example:"25.5"`
	RecordedAt *string  `json:"recorded_at,omitempty" validate:"omitempty,datetime=2006-01-02T15:04:05Z07:00" example:"2024-01-01T00:00:00Z"`
}

// ==========================================
// 轉換函式 (Mapper Functions)
// ==========================================

// ToUserResponse 負責將內部模型轉換為外部 DTO
func ToUserResponse(u model.User) UserResponse {
	return UserResponse{
		ID:        u.ID,
		Username:  u.Username,
		Email:     u.Email,
		CreatedAt: u.CreatedAt,
	}
}

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
