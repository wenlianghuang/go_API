# Storage Interface 實現說明

## 📋 概述

本文檔解釋為什麼 `MemoryStore` 和 `GormStore` 都需要實現 `Storage` interface 的所有方法，即使生產環境可能只使用其中一個實現。

## 🔍 Go 的 Interface 實現機制

### 1. Interface 定義（合約）

在 `db.go` 中，我們定義了 `Storage` interface：

```go
type Storage interface {
    // User 相關
    Create(u model.User) error
    Get(id string) (model.User, error)
    GetUserByEmail(email string) (model.User, error)
    List() ([]model.User, error)

    // 設備相關
    CreateDevice(dev *model.Device) error
    GetDeviceByID(id uint) (*model.Device, error)
    ListDevices() ([]model.Device, error)
    DeleteDeviceWithAllData(id uint) error
    UpdateDevice(id uint, device *model.Device) error
    PatchDevice(id uint, updates map[string]interface{}) error

    // 遙測數據相關
    ListTelemetries() ([]model.Telemetry, error)
    AddTelemetry(data *model.Telemetry) error
    GetTelemetryByID(id uint) (*model.Telemetry, error)
    PatchTelemetry(id uint, updates map[string]interface{}) error
    DeleteTelemetry(id uint) error  // ← 這是合約的一部分
}
```

### 2. 兩個不同的實現

項目中有兩個實現 `Storage` interface 的類型：

- **`MemoryStore`**（在 `db.go` 中）- 用於測試和開發，數據存儲在內存中
- **`GormStore`**（在 `gorm_store.go` 中）- 用於生產環境，使用 GORM 操作 PostgreSQL

### 3. 為什麼兩個都要實現？

#### Go 的隱式 Interface 實現

在 Go 中，interface 實現是**隱式的（implicit）**，不是顯式的。這意味著：

- 不需要明確聲明 `implements Storage`
- 只要類型實現了 interface 中定義的所有方法，它就自動實現了該 interface
- **編譯器會在編譯時檢查所有實現是否完整**

#### 編譯時檢查

當你運行 `go build` 或 `go test` 時，編譯器會檢查：

```bash
# 編譯器會檢查：
# 1. GormStore 是否實現了 Storage 的所有方法？ ✅
# 2. MemoryStore 是否實現了 Storage 的所有方法？ ❌ 如果缺少方法會編譯失敗
# 3. MockStore（測試用）是否實現了 Storage 的所有方法？ ✅

# 結果：如果任何一個實現缺少方法，整個項目編譯失敗！
```

#### 實際使用場景

```go
// 場景 1: 生產環境（main.go）
gormStore, err := store.NewGormStore(db)  // 使用 GormStore
srv := api.NewServer(gormStore, redisAddr) // 傳入 Storage interface

// 場景 2: 單元測試（handler_test.go）
mockStore := new(MockStore)                // 使用 MockStore
srv := api.NewServer(mockStore, redisAddr) // 傳入 Storage interface

// 場景 3: 開發測試（可能使用 MemoryStore）
memoryStore := store.NewMemoryStore()      // 使用 MemoryStore
srv := api.NewServer(memoryStore, redisAddr) // 傳入 Storage interface
```

關鍵點：
- `Server` 接受的是 `store.Storage` **interface**，不是具體的 `GormStore` 或 `MemoryStore`
- 編譯器會檢查**所有**實現 `Storage` 的類型是否完整實現了所有方法
- 即使生產環境只用 `GormStore`，如果 `MemoryStore` 缺少方法，編譯也會失敗

## 🎯 為什麼這樣設計？

### 1. 類型安全

Go 在編譯時保證類型安全，避免運行時錯誤。如果某個實現缺少方法，編譯時就會發現問題。

### 2. 可替換性

由於所有實現都完整實現了 interface，你可以輕鬆切換不同的實現：

```go
// 可以輕鬆切換
var store store.Storage

// 開發環境
store = store.NewMemoryStore()

// 生產環境
store = store.NewGormStore(db)

// 測試環境
store = new(MockStore)

// 所有這些都可以傳給 Server，因為它們都實現了 Storage interface
srv := api.NewServer(store, redisAddr)
```

### 3. 測試友好

在測試中，你可以使用 `MemoryStore` 或 `MockStore`，它們都必須實現完整的 interface，確保測試覆蓋所有方法。

## ⚠️ 常見錯誤

### 錯誤示例

```go
// ❌ 錯誤：只實現了部分方法
type MemoryStore struct {
    // ...
}

func (s *MemoryStore) Create(u model.User) error { ... }
func (s *MemoryStore) Get(id string) (model.User, error) { ... }
// 缺少 DeleteTelemetry 方法

// 編譯錯誤：
// MemoryStore does not implement Storage (missing DeleteTelemetry method)
```

### 正確做法

```go
// ✅ 正確：實現所有方法
type MemoryStore struct {
    // ...
}

func (s *MemoryStore) Create(u model.User) error { ... }
func (s *MemoryStore) Get(id string) (model.User, error) { ... }
// ... 其他方法
func (s *MemoryStore) DeleteTelemetry(id uint) error { ... } // 必須實現
```

## 📝 添加新方法時的步驟

當你在 `Storage` interface 中添加新方法時，必須：

1. ✅ 在 `Storage` interface 中定義方法簽名
2. ✅ 在 `GormStore` 中實現該方法
3. ✅ 在 `MemoryStore` 中實現該方法
4. ✅ 在測試用的 `MockStore` 中實現該方法（如果有的話）

## 🔗 相關文件

- `backend/store/db.go` - Storage interface 定義和 MemoryStore 實現
- `backend/store/gorm_store.go` - GormStore 實現
- `backend/api/handler_test.go` - 測試用的 MockStore 實現
- `backend/main_test.go` - 另一個測試用的 MockStore 實現

## 📚 參考資料

- [Go 官方文檔：Interfaces](https://go.dev/tour/methods/9)
- [Effective Go: Interfaces](https://go.dev/doc/effective_go#interfaces)
