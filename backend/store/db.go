package store

import (
	"context"
	"fmt"
	"my-api/model"
	"reflect"
	"sync"
)

// Storage 定義了資料庫的行為 (Interface)，這是為了以後可以隨時換成 Postgres/MySQL
type Storage interface {
	// User 相關
	Create(ctx context.Context, u model.User) error
	Get(ctx context.Context, id string) (model.User, error)
	GetUserByEmail(ctx context.Context, email string) (model.User, error)
	List(ctx context.Context) ([]model.User, error)

	// 設備相關
	CreateDevice(ctx context.Context, dev *model.Device) error
	GetDeviceByID(ctx context.Context, id uint) (*model.Device, error)
	ListDevices(ctx context.Context) ([]model.Device, error)

	// 定義刪除功能的合約
	// 這樣 Server 才知道可以呼叫這個方法
	DeleteDeviceWithAllData(ctx context.Context, id uint) error
	// 更新整個設備（所有字段）- PUT 使用
	UpdateDevice(ctx context.Context, id uint, device *model.Device) error
	// 部分更新設備（只更新提供的字段）- PATCH 使用
	PatchDevice(ctx context.Context, id uint, updates map[string]interface{}) error
	// 數據相關
	ListTelemetries(ctx context.Context) ([]model.Telemetry, error)
	AddTelemetry(ctx context.Context, data *model.Telemetry) error
	GetTelemetryByID(ctx context.Context, id uint) (*model.Telemetry, error)
	// 部分更新遙測數據（只更新提供的字段）- PATCH 使用
	PatchTelemetry(ctx context.Context, id uint, updates map[string]interface{}) error

	// 刪除遙測數據
	DeleteTelemetry(ctx context.Context, id uint) error

	// 執行一個事務
	ExecTx(ctx context.Context, fn func(txStorage Storage) error) error
}

// MemoryStore 是 Storage 的一個實作 (存在記憶體中)
type MemoryStore struct {
	mu              sync.RWMutex
	users           map[string]model.User
	devices         map[uint]*model.Device
	telemetries     map[uint]*model.Telemetry
	nextDeviceID    uint
	nextTelemetryID uint
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		users:           make(map[string]model.User),
		devices:         make(map[uint]*model.Device),
		telemetries:     make(map[uint]*model.Telemetry),
		nextDeviceID:    1,
		nextTelemetryID: 1,
	}
}

func (s *MemoryStore) Create(ctx context.Context, u model.User) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.users[u.ID]; ok {
		return fmt.Errorf("user already exists")
	}
	s.users[u.ID] = u
	return nil
}

func (s *MemoryStore) Get(ctx context.Context, id string) (model.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	user, ok := s.users[id]
	if !ok {
		return model.User{}, fmt.Errorf("user not found")
	}
	return user, nil
}

func (s *MemoryStore) List(ctx context.Context) ([]model.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var users []model.User
	for _, u := range s.users {
		users = append(users, u)
	}
	return users, nil
}

func (s *MemoryStore) GetUserByEmail(ctx context.Context, email string) (model.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, u := range s.users {
		if u.Email == email {
			return u, nil
		}
	}
	return model.User{}, fmt.Errorf("user not found")
}

func (s *MemoryStore) ListTelemetries(ctx context.Context) ([]model.Telemetry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var telemetries []model.Telemetry
	for _, tel := range s.telemetries {
		telemetries = append(telemetries, *tel)
	}
	return telemetries, nil
}

func (s *MemoryStore) CreateDevice(ctx context.Context, dev *model.Device) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	dev.ID = s.nextDeviceID
	s.devices[dev.ID] = dev
	s.nextDeviceID++
	return nil
}

func (s *MemoryStore) GetDeviceByID(ctx context.Context, id uint) (*model.Device, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	dev, ok := s.devices[id]
	if !ok {
		return nil, fmt.Errorf("device not found")
	}
	return dev, nil
}

func (s *MemoryStore) ListDevices(ctx context.Context) ([]model.Device, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var devices []model.Device
	for _, dev := range s.devices {
		devices = append(devices, *dev)
	}
	return devices, nil
}

func (s *MemoryStore) DeleteDeviceWithAllData(ctx context.Context, id uint) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.devices[id]; !ok {
		return fmt.Errorf("device not found")
	}
	delete(s.devices, id)
	// Also delete associated telemetry
	for tid, tel := range s.telemetries {
		if tel.DeviceID == id {
			delete(s.telemetries, tid)
		}
	}
	return nil
}

func (s *MemoryStore) UpdateDevice(ctx context.Context, id uint, device *model.Device) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.devices[id]; !ok {
		return fmt.Errorf("device not found")
	}
	device.ID = id
	s.devices[id] = device
	return nil
}

func (s *MemoryStore) PatchDevice(ctx context.Context, id uint, updates map[string]interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	dev, ok := s.devices[id]
	if !ok {
		return fmt.Errorf("device not found")
	}
	// This is a simplified implementation of patch
	for k, v := range updates {
		// Use reflection to update fields, this is a simple version
		field := reflect.ValueOf(dev).Elem().FieldByName(k)
		if field.IsValid() && field.CanSet() {
			val := reflect.ValueOf(v)
			if field.Type() == val.Type() {
				field.Set(val)
			}
		}
	}
	s.devices[id] = dev
	return nil
}

func (s *MemoryStore) AddTelemetry(ctx context.Context, data *model.Telemetry) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	data.ID = s.nextTelemetryID
	s.telemetries[data.ID] = data
	s.nextTelemetryID++
	return nil
}

func (s *MemoryStore) GetTelemetryByID(ctx context.Context, id uint) (*model.Telemetry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	tel, ok := s.telemetries[id]
	if !ok {
		return nil, fmt.Errorf("telemetry not found")
	}
	return tel, nil
}

func (s *MemoryStore) PatchTelemetry(ctx context.Context, id uint, updates map[string]interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	tel, ok := s.telemetries[id]
	if !ok {
		return fmt.Errorf("telemetry not found")
	}
	// Simplified patch implementation
	for k, v := range updates {
		field := reflect.ValueOf(tel).Elem().FieldByName(k)
		if field.IsValid() && field.CanSet() {
			val := reflect.ValueOf(v)
			if field.Type() == val.Type() {
				field.Set(val)
			}
		}
	}
	s.telemetries[id] = tel
	return nil
}

func (s *MemoryStore) DeleteTelemetry(ctx context.Context, id uint) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.telemetries[id]; !ok {
		return fmt.Errorf("telemetry not found")
	}
	delete(s.telemetries, id)
	return nil
}
