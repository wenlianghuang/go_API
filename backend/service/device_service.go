package service

import (
	"context"
	"my-api/errors"
	"my-api/model"
	"my-api/store"
)

// DeviceService 處理設備相關的業務邏輯
type DeviceService struct {
	store store.Storage
}

// NewDeviceService 創建一個新的 DeviceService 實例
func NewDeviceService(store store.Storage) *DeviceService {
	return &DeviceService{
		store: store,
	}
}

// CreateDeviceInput 創建設備輸入參數
type CreateDeviceInput struct {
	Name       string
	Type       string
	MacAddress string
	IsActive   bool
	UserID     string
}

// CreateDeviceResult 創建設備結果
type CreateDeviceResult struct {
	Device *model.Device
}

// CreateDevice 處理創建設備業務邏輯
// 返回 CreateDeviceResult 和錯誤
func (s *DeviceService) CreateDevice(ctx context.Context, input CreateDeviceInput, defaultIsActive bool) (*CreateDeviceResult, error) {
	// 驗證 UserID 不為空
	if input.UserID == "" {
		return nil, errors.NewBadRequestError("user ID is required")
	}

	// 轉換成 Domain Model
	device := &model.Device{
		Name:       input.Name,
		Type:       input.Type,
		MacAddress: input.MacAddress,
		IsActive:   input.IsActive,
		UserID:     input.UserID,
	}

	// 如果沒有指定 IsActive，使用預設值
	if !input.IsActive && defaultIsActive {
		device.IsActive = true
	}

	// 呼叫資料庫層
	if err := s.store.CreateDevice(ctx, device); err != nil {
		return nil, errors.NewDeviceCreateFailedError("database operation failed", err)
	}

	// 構建結果
	result := &CreateDeviceResult{
		Device: device,
	}

	return result, nil
}

// PatchDeviceInput 部分更新設備輸入參數
type PatchDeviceInput struct {
	Name       *string
	Type       *string
	MacAddress *string
	IsActive   *bool
}

// PatchDeviceResult 部分更新設備結果
type PatchDeviceResult struct {
	Device *model.Device
}

// PatchDevice 處理部分更新設備業務邏輯
// 返回 PatchDeviceResult 和錯誤
func (s *DeviceService) PatchDevice(ctx context.Context, deviceID uint, input PatchDeviceInput) (*PatchDeviceResult, error) {
	// 驗證設備是否存在
	_, err := s.store.GetDeviceByID(ctx, deviceID)
	if err != nil {
		return nil, errors.NewDeviceNotFoundError(deviceID)
	}

	// 構建更新映射（只包含提供的字段）
	updates := make(map[string]interface{})
	if input.Name != nil {
		updates["name"] = *input.Name
	}
	if input.Type != nil {
		updates["type"] = *input.Type
	}
	if input.MacAddress != nil {
		updates["mac_address"] = *input.MacAddress
	}
	if input.IsActive != nil {
		updates["is_active"] = *input.IsActive
	}

	// 如果沒有任何更新字段
	if len(updates) == 0 {
		return nil, errors.NewBadRequestError("at least one field must be provided for update")
	}

	// 執行部分更新
	if err := s.store.PatchDevice(ctx, deviceID, updates); err != nil {
		return nil, errors.NewDeviceUpdateFailedError(deviceID, "database operation failed", err)
	}

	// 獲取更新後的設備資訊
	updatedDevice, err := s.store.GetDeviceByID(ctx, deviceID)
	if err != nil {
		return nil, errors.NewInternalError("fetch updated device", err)
	}

	// 構建結果
	result := &PatchDeviceResult{
		Device: updatedDevice,
	}

	return result, nil
}

// CreateDeviceWithInitLog 創建設備並同時寫入初始 Log（在同一個事務內）
// 返回 CreateDeviceResult 和錯誤
func (s *DeviceService) CreateDeviceWithInitLog(ctx context.Context, input CreateDeviceInput, defaultIsActive bool) (*CreateDeviceResult, error) {
	// 驗證 UserID 不為空
	if input.UserID == "" {
		return nil, errors.NewBadRequestError("user ID is required")
	}

	// 轉換成 Domain Model
	device := &model.Device{
		Name:       input.Name,
		Type:       input.Type,
		MacAddress: input.MacAddress,
		IsActive:   input.IsActive,
		UserID:     input.UserID,
	}

	// 如果沒有指定 IsActive，使用預設值
	if !input.IsActive && defaultIsActive {
		device.IsActive = true
	}

	// 呼叫我們剛寫的 ExecTx
	// 注意這裡的 txStore，它是專門為這次事務臨時創建的
	err := s.store.ExecTx(ctx, func(txStore store.Storage) error {
		// 1. 創建設備 (操作 Devices 表)
		// 使用 txStore，而不是 s.store
		if err := txStore.CreateDevice(ctx, device); err != nil {
			return err // 返回錯誤會觸發 Rollback
		}
		// 2. 寫入初始 Log (操作 Telemetries 表)
		// 使用同一個 txStore，確保在同一個事務內
		initTelemetry := &model.Telemetry{
			DeviceID: device.ID, // 這裡能拿到 ID，因為上面已經執行了 (但在事務提交前這 ID 只在當前 Session 有效)
			DataType: "System",
			Value:    1.0, // 1.0 代表初始化成功
		}
		if err := txStore.AddTelemetry(ctx, initTelemetry); err != nil {
			return err // 如果這裡失敗，上面的 CreateDevice 也會被撤銷！
		}

		return nil // 函數結束，自動 Commit
	})

	if err != nil {
		return nil, errors.NewDeviceCreateFailedError("database operation failed", err)
	}

	// 構建結果
	result := &CreateDeviceResult{
		Device: device,
	}

	return result, nil
}
