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
func NewGormStore(db *gorm.DB) *GormStore {
	return &GormStore{db: db}
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

// Create 實作建立使用者
func (s *GormStore) Create(u User) error {
	return s.db.Create(&u).Error
}

// Get 實作查詢單一使用者
func (s *GormStore) Get(id string) (User, error) {
	var user User
	result := s.db.Where("id = ?", id).First(&user)
	if result.Error != nil {
		return User{}, result.Error
	}
	return user, nil
}

// List 實作列表查詢使用者
func (s *GormStore) List() ([]User, error) {
	var users []User
	result := s.db.Find(&users)
	return users, result.Error
}
