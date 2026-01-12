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
	Username string `json:"username" validate:"required,min=3,max=50,alphanum" example:"john_doe"`
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
