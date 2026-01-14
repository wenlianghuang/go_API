package store

import (
	"errors"
	"my-api/model"

	"gorm.io/gorm"
)

// GormStore 是 Storage 介面的一個實作
type GormStore struct {
	db *gorm.DB
}

// NewGormStore 是一個工廠函式
func NewGormStore(db *gorm.DB) (*GormStore, error) {
	// 資料庫遷移現在由 golang-migrate 處理
	// 請確保在啟動應用程式前已執行遷移：./scripts/migrate.sh up
	return &GormStore{db: db}, nil
}

// Create 實作建立使用者
func (s *GormStore) Create(u model.User) error {
	return s.db.Create(&u).Error
}

// Get 實作查詢單一使用者
func (s *GormStore) Get(id string) (model.User, error) {
	var user model.User
	result := s.db.Where("id = ?", id).First(&user)
	if result.Error != nil {
		return model.User{}, result.Error
	}
	return user, nil
}

// List 實作列表查詢使用者
func (s *GormStore) List() ([]model.User, error) {
	var users []model.User
	result := s.db.Find(&users)
	return users, result.Error
}

// GetUserByEmail 實作通過 email 查詢使用者
func (s *GormStore) GetUserByEmail(email string) (model.User, error) {
	var user model.User
	result := s.db.Where("email = ?", email).First(&user)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return model.User{}, errors.New("user not found")
	}
	if result.Error != nil {
		return model.User{}, result.Error
	}
	return user, nil
}

// CreateDevice 實作建立設備
func (s *GormStore) CreateDevice(dev *model.Device) error {
	// GORM 的 Create 會自動處理 SQL Insert
	result := s.db.Create(dev)
	if result.Error != nil {
		// 這裡可以做錯誤轉換，例如檢查是不是重複的 MacAddress
		return result.Error
	}
	return nil
}

// GetDeviceByID 實作查詢單一設備
func (s *GormStore) GetDeviceByID(id uint) (*model.Device, error) {
	var dev model.Device

	// Preload("Telemetries"): 這就是 GORM 的強大之處
	// 它會自動幫你執行兩次查詢，把該設備關聯的數據也一起抓出來 (Eager Loading)
	result := s.db.Preload("Telemetries").First(&dev, id)

	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, errors.New("device not found")
	}
	if result.Error != nil {
		return nil, result.Error
	}
	return &dev, nil
}

// ListDevices 實作列表查詢
func (s *GormStore) ListDevices() ([]model.Device, error) {
	var devices []model.Device
	// Find 會抓取所有資料
	result := s.db.Find(&devices)
	return devices, result.Error
}

func (s *GormStore) AddTelemetry(data *model.Telemetry) error {
	return s.db.Create(data).Error
}

// ListTelemetries 實作列表查詢遙測數據
func (s *GormStore) ListTelemetries() ([]model.Telemetry, error) {
	var telemetries []model.Telemetry
	if err := s.db.Find(&telemetries).Error; err != nil {
		return nil, err
	}
	return telemetries, nil
}

// GetTelemetryByID 實作查詢單一遙測數據
func (s *GormStore) GetTelemetryByID(id uint) (*model.Telemetry, error) {
	var telemetry model.Telemetry
	result := s.db.First(&telemetry, id)

	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, errors.New("telemetry not found")
	}
	if result.Error != nil {
		return nil, result.Error
	}
	return &telemetry, nil
}

// DeleteDeviceWithAllData 刪除設備及其所有遙測數據 (原子性操作)
func (s *GormStore) DeleteDeviceWithAllData(id uint) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Delete(&model.Telemetry{}, "device_id = ?", id).Error; err != nil {
			return err
		}
		if err := tx.Delete(&model.Device{}, id).Error; err != nil {
			return err
		}
		return nil
	})
}

// UpdateDevice 更新整個設備（所有字段）- PUT 使用
func (s *GormStore) UpdateDevice(id uint, device *model.Device) error {
	// 先檢查設備是否存在
	var existingDevice model.Device
	result := s.db.First(&existingDevice, id)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return errors.New("device not found")
	}
	if result.Error != nil {
		return result.Error
	}

	// 更新設備的所有字段（除了 ID 和 CreatedAt）
	device.ID = id // 確保 ID 不會被更新
	updateResult := s.db.Model(&existingDevice).Updates(model.Device{
		Name:       device.Name,
		Type:       device.Type,
		MacAddress: device.MacAddress,
		IsActive:   device.IsActive,
	})

	if updateResult.Error != nil {
		return updateResult.Error
	}
	return nil
}

// PatchDevice 部分更新設備（只更新提供的字段）- PATCH 使用
func (s *GormStore) PatchDevice(id uint, updates map[string]interface{}) error {
	// 先檢查設備是否存在
	var existingDevice model.Device
	result := s.db.First(&existingDevice, id)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return errors.New("device not found")
	}
	if result.Error != nil {
		return result.Error
	}

	// 只更新提供的字段
	updateResult := s.db.Model(&existingDevice).Updates(updates)
	if updateResult.Error != nil {
		return updateResult.Error
	}
	return nil
}

// PatchTelemetry 部分更新遙測數據（只更新提供的字段）- PATCH 使用
func (s *GormStore) PatchTelemetry(id uint, updates map[string]interface{}) error {
	// 先檢查遙測數據是否存在
	var existingTelemetry model.Telemetry
	result := s.db.First(&existingTelemetry, id)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return errors.New("telemetry not found")
	}
	if result.Error != nil {
		return result.Error
	}

	// 只更新提供的字段
	updateResult := s.db.Model(&existingTelemetry).Updates(updates)
	if updateResult.Error != nil {
		return updateResult.Error
	}
	return nil
}
