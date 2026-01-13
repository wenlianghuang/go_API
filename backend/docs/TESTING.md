# Go 測試最佳實踐指南

本文件介紹本專案使用的測試策略和工具，包括 **Table-Driven Tests** 和 **Testify** 測試框架。

## 目錄

- [測試工具](#測試工具)
- [Table-Driven Tests](#table-driven-tests)
- [Testify 框架](#testify-框架)
- [Assert vs Require](#assert-vs-require)
- [實際範例](#實際範例)
- [執行測試](#執行測試)
- [效能測試](#效能測試)

---

## 測試工具

### 1. Go 標準測試庫

Go 內建的 `testing` 包提供基本的測試功能：

```go
import "testing"

func TestExample(t *testing.T) {
    if got != want {
        t.Errorf("got %v, want %v", got, want)
    }
}
```

### 2. Testify - 強大的測試工具包

[Testify](https://github.com/stretchr/testify) 是 Go 生態系統中最受歡迎的測試工具包，提供：

- **assert**: 斷言失敗時繼續執行測試
- **require**: 斷言失敗時立即終止測試
- **mock**: 建立 mock 物件
- **suite**: 測試套件功能

#### 安裝

```bash
go get github.com/stretchr/testify/assert
go get github.com/stretchr/testify/require
```

---

## Table-Driven Tests

Table-Driven Tests 是 Go 社群推薦的測試模式，將測試案例定義在一個結構體切片中。

### 優點

✅ **易於維護**：所有測試案例集中在一起  
✅ **易於擴展**：新增測試案例只需添加一條記錄  
✅ **減少重複代碼**：測試邏輯只寫一次  
✅ **清晰易讀**：測試案例一目了然  

### 基本結構

```go
func TestFunction(t *testing.T) {
    tests := []struct {
        name    string  // 測試案例名稱
        input   string  // 輸入參數
        want    string  // 期望輸出
        wantErr bool    // 是否期望錯誤
    }{
        {
            name:    "Valid input",
            input:   "test",
            want:    "result",
            wantErr: false,
        },
        {
            name:    "Invalid input",
            input:   "",
            want:    "",
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := Function(tt.input)
            
            if tt.wantErr {
                assert.Error(t, err)
                return
            }
            
            assert.NoError(t, err)
            assert.Equal(t, tt.want, got)
        })
    }
}
```

---

## Testify 框架

### Assert - 驗證但繼續執行

`assert` 會在驗證失敗時記錄錯誤，但**不會終止測試**，適合用於需要檢查多個條件的情況。

```go
import "github.com/stretchr/testify/assert"

func TestWithAssert(t *testing.T) {
    result := calculate(10)
    
    assert.NotNil(t, result)        // 如果失敗，繼續執行
    assert.Equal(t, 100, result)     // 仍會執行
    assert.Greater(t, result, 0)     // 仍會執行
}
```

### Require - 驗證且立即終止

`require` 會在驗證失敗時**立即終止測試**，適合用於後續測試依賴的前置條件。

```go
import "github.com/stretchr/testify/require"

func TestWithRequire(t *testing.T) {
    result := getData()
    
    require.NotNil(t, result)       // 如果失敗，立即終止
    // 如果 result 是 nil，後續代碼不會執行
    assert.Equal(t, "expected", result.Name)
}
```

---

## Assert vs Require

### 使用時機

| 情境 | 使用 | 原因 |
|------|------|------|
| 測試資料準備 | `require` | 資料無效時後續測試無意義 |
| 生成測試 Token | `require` | Token 無效時無法繼續測試 |
| 多個獨立驗證 | `assert` | 想看到所有失敗的驗證 |
| 驗證多個欄位 | `assert` | 顯示所有不匹配的欄位 |
| nil 檢查 | `require` | 避免後續程式碼 panic |

### 實例對比

#### ❌ 不好的做法（容易 panic）

```go
func TestBad(t *testing.T) {
    claims, err := ValidateJWT(token)
    assert.NoError(t, err)           // 如果失敗，繼續執行
    assert.Equal(t, "user123", claims.UserID)  // claims 可能是 nil，會 panic！
}
```

#### ✅ 好的做法（安全且清晰）

```go
func TestGood(t *testing.T) {
    claims, err := ValidateJWT(token)
    require.NoError(t, err)          // 失敗立即終止
    require.NotNil(t, claims)        // 確保 claims 不是 nil
    
    // 以下代碼安全執行
    assert.Equal(t, "user123", claims.UserID)
    assert.Equal(t, "testuser", claims.Username)
    assert.Equal(t, "test@example.com", claims.Email)
}
```

---

## 實際範例

### 範例 1：JWT Token 生成測試

```go
func TestGenerateJWT(t *testing.T) {
    oldSecret := JWTSecret
    JWTSecret = "test-secret-key"
    defer func() { JWTSecret = oldSecret }()

    tests := []struct {
        name            string
        userID          string
        username        string
        email           string
        expirationHours int
        wantErr         bool
    }{
        {
            name:            "Valid token generation",
            userID:          "user123",
            username:        "testuser",
            email:           "test@example.com",
            expirationHours: 24,
            wantErr:         false,
        },
        {
            name:            "Short expiration",
            userID:          "user456",
            username:        "shortuser",
            email:           "short@example.com",
            expirationHours: 1,
            wantErr:         false,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            token, err := GenerateJWT(tt.userID, tt.username, tt.email, tt.expirationHours)

            if tt.wantErr {
                assert.Error(t, err)
                return
            }
            
            // 使用 require 確保 token 生成成功
            require.NoError(t, err)
            require.NotEmpty(t, token)

            // 驗證 token 可以被解析
            claims, err := ValidateJWT(token)
            require.NoError(t, err)
            require.NotNil(t, claims)

            // 使用 assert 驗證所有欄位
            assert.Equal(t, tt.userID, claims.UserID)
            assert.Equal(t, tt.username, claims.Username)
            assert.Equal(t, tt.email, claims.Email)
        })
    }
}
```

### 範例 2：Token 驗證測試

```go
func TestValidateJWT(t *testing.T) {
    oldSecret := JWTSecret
    JWTSecret = "test-secret-key"
    defer func() { JWTSecret = oldSecret }()

    // 準備測試資料 - 使用 require 確保成功
    validToken, err := GenerateJWT("user123", "testuser", "test@example.com", 24)
    require.NoError(t, err)

    expiredToken, err := GenerateJWT("user456", "expired", "expired@example.com", -1)
    require.NoError(t, err)

    tests := []struct {
        name          string
        token         string
        wantErr       bool
        errorContains string
        expectedUser  string
    }{
        {
            name:         "Valid token",
            token:        validToken,
            wantErr:      false,
            expectedUser: "user123",
        },
        {
            name:          "Expired token",
            token:         expiredToken,
            wantErr:       true,
            errorContains: "token",
        },
        {
            name:          "Empty token",
            token:         "",
            wantErr:       true,
            errorContains: "token",
        },
        {
            name:          "Malformed token",
            token:         "invalid.token.string",
            wantErr:       true,
            errorContains: "token",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            claims, err := ValidateJWT(tt.token)

            if tt.wantErr {
                assert.Error(t, err)
                if tt.errorContains != "" {
                    assert.Contains(t, strings.ToLower(err.Error()),
                        strings.ToLower(tt.errorContains))
                }
                return
            }

            assert.NoError(t, err)
            assert.NotNil(t, claims)
            assert.Equal(t, tt.expectedUser, claims.UserID)
        })
    }
}
```

### 範例 3：複雜的測試設置

```go
func TestRefreshJWT(t *testing.T) {
    oldSecret := JWTSecret
    JWTSecret = "test-secret-key"
    defer func() { JWTSecret = oldSecret }()

    tests := []struct {
        name          string
        setupToken    func() string  // 動態生成測試 token
        wantErr       bool
        errorContains string
    }{
        {
            name: "Refresh valid token",
            setupToken: func() string {
                token, _ := GenerateJWT("user123", "testuser", "test@example.com", 24)
                return token
            },
            wantErr: false,
        },
        {
            name: "Refresh expired token",
            setupToken: func() string {
                token, _ := GenerateJWT("user456", "expired", "expired@example.com", -1)
                return token
            },
            wantErr:       true,
            errorContains: "token",
        },
        {
            name: "Refresh with wrong secret",
            setupToken: func() string {
                oldSecret := JWTSecret
                JWTSecret = "different-secret"
                token, _ := GenerateJWT("user789", "diffuser", "diff@example.com", 24)
                JWTSecret = oldSecret
                return token
            },
            wantErr:       true,
            errorContains: "signature",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            oldToken := tt.setupToken()
            
            if !tt.wantErr {
                time.Sleep(10 * time.Millisecond)  // 確保時間戳不同
            }

            newToken, err := RefreshJWT(oldToken)

            if tt.wantErr {
                assert.Error(t, err)
                if tt.errorContains != "" {
                    assert.Contains(t, strings.ToLower(err.Error()),
                        strings.ToLower(tt.errorContains))
                }
                return
            }

            // 驗證成功的情況
            require.NoError(t, err)
            require.NotEmpty(t, newToken)
            assert.NotEqual(t, oldToken, newToken)

            // 驗證新 token 有效
            newClaims, err := ValidateJWT(newToken)
            require.NoError(t, err)
            require.NotNil(t, newClaims)

            // 驗證 claims 被保留
            oldClaims, _ := ValidateJWT(oldToken)
            assert.Equal(t, oldClaims.UserID, newClaims.UserID)
            assert.Equal(t, oldClaims.Username, newClaims.Username)
            assert.Equal(t, oldClaims.Email, newClaims.Email)
        })
    }
}
```

---

## 執行測試

### 執行所有測試

```bash
cd backend
JWT_SECRET=test-secret go test ./...
```

### 執行特定套件的測試

```bash
JWT_SECRET=test-secret go test ./api
```

### 執行特定測試

```bash
JWT_SECRET=test-secret go test ./api -run TestGenerateJWT
```

### 執行符合模式的多個測試

```bash
JWT_SECRET=test-secret go test ./api -run "^TestGenerateJWT|TestValidateJWT"
```

### 詳細輸出模式

```bash
JWT_SECRET=test-secret go test -v ./api
```

### 顯示覆蓋率

```bash
JWT_SECRET=test-secret go test -cover ./...
```

### 產生覆蓋率報告

```bash
JWT_SECRET=test-secret go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

---

## 效能測試

### Benchmark 測試

使用 Go 的 benchmark 功能測試性能：

```go
func BenchmarkGenerateJWT(b *testing.B) {
    oldSecret := JWTSecret
    JWTSecret = "test-secret-key"
    defer func() { JWTSecret = oldSecret }()

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, _ = GenerateJWT("user123", "testuser", "test@example.com", 24)
    }
}

func BenchmarkValidateJWT(b *testing.B) {
    oldSecret := JWTSecret
    JWTSecret = "test-secret-key"
    defer func() { JWTSecret = oldSecret }()

    token, _ := GenerateJWT("user123", "testuser", "test@example.com", 24)

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, _ = ValidateJWT(token)
    }
}
```

### 執行 Benchmark

```bash
cd backend
JWT_SECRET=test-secret go test -bench=. ./api
```

### Benchmark 輸出範例

```
BenchmarkGenerateJWT-8     5000    235467 ns/op
BenchmarkValidateJWT-8    10000    156234 ns/op
```

解讀：
- `-8`：使用 8 個 CPU 核心
- `5000`：執行了 5000 次
- `235467 ns/op`：平均每次操作耗時 235 微秒

---

## 常用 Testify 斷言方法

### 相等性斷言

```go
assert.Equal(t, expected, actual)           // 值相等
assert.NotEqual(t, unexpected, actual)      // 值不相等
assert.Exactly(t, expected, actual)         // 類型和值都相等
assert.NotNil(t, object)                    // 不是 nil
assert.Nil(t, object)                       // 是 nil
```

### 布林斷言

```go
assert.True(t, condition)                   // 條件為真
assert.False(t, condition)                  // 條件為假
```

### 錯誤斷言

```go
assert.NoError(t, err)                      // 無錯誤
assert.Error(t, err)                        // 有錯誤
assert.EqualError(t, err, "error message")  // 錯誤訊息相等
assert.ErrorContains(t, err, "substring")   // 錯誤訊息包含子字串
```

### 字串斷言

```go
assert.Contains(t, "hello world", "world")  // 包含子字串
assert.NotContains(t, "hello", "xyz")       // 不包含子字串
assert.Empty(t, str)                        // 字串為空
assert.NotEmpty(t, str)                     // 字串不為空
```

### 數字斷言

```go
assert.Greater(t, actual, expected)         // 大於
assert.GreaterOrEqual(t, actual, expected)  // 大於等於
assert.Less(t, actual, expected)            // 小於
assert.LessOrEqual(t, actual, expected)     // 小於等於
```

### 集合斷言

```go
assert.Len(t, list, expectedLength)         // 長度相等
assert.ElementsMatch(t, expected, actual)   // 元素匹配（順序無關）
```

---

## 測試最佳實踐

### 1. 使用描述性的測試名稱

```go
// ❌ 不好
{name: "test1", ...}

// ✅ 好
{name: "Valid token generation", ...}
{name: "Expired token should fail", ...}
```

### 2. 適當使用 require 和 assert

```go
// ✅ 好的做法
func TestExample(t *testing.T) {
    result := getData()
    require.NotNil(t, result)  // 必須成功，否則後續無意義
    
    assert.Equal(t, "value1", result.Field1)  // 想看所有失敗
    assert.Equal(t, "value2", result.Field2)
    assert.Equal(t, "value3", result.Field3)
}
```

### 3. 清理測試環境

```go
func TestWithCleanup(t *testing.T) {
    // 保存原始狀態
    oldValue := GlobalVar
    defer func() { GlobalVar = oldValue }()  // 恢復
    
    // 執行測試
    GlobalVar = "test-value"
    // ...
}
```

### 4. 獨立的測試案例

每個測試案例應該獨立，不依賴其他測試的執行順序或狀態。

```go
// ✅ 好 - 每個測試獨立設置
func TestIndependent(t *testing.T) {
    tests := []struct {
        name  string
        setup func()
        // ...
    }{
        {
            name: "Test case 1",
            setup: func() {
                // 設置測試環境
            },
        },
    }
    // ...
}
```

### 5. 測試邊界條件

```go
tests := []struct {
    name  string
    input int
}{
    {name: "Zero value", input: 0},
    {name: "Negative value", input: -1},
    {name: "Maximum value", input: math.MaxInt},
    {name: "Normal value", input: 100},
}
```

---

## 參考資源

- [Go Testing Package](https://pkg.go.dev/testing)
- [Testify GitHub](https://github.com/stretchr/testify)
- [Table Driven Tests in Go](https://dave.cheney.net/2019/05/07/prefer-table-driven-tests)
- [Effective Go - Testing](https://go.dev/doc/effective_go#testing)

---

## 總結

- ✅ 使用 **Table-Driven Tests** 組織測試案例
- ✅ 使用 **Testify** 讓斷言更清晰易讀
- ✅ 關鍵前置條件使用 `require`，一般驗證使用 `assert`
- ✅ 每個測試案例都要有描述性的名稱
- ✅ 測試應該獨立且可重複執行
- ✅ 測試邊界條件和錯誤情況
- ✅ 使用 benchmark 測試性能關鍵代碼

遵循這些最佳實踐，可以編寫出高品質、易維護的測試代碼！
