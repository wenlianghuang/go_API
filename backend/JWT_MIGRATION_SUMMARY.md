# JWT 認證系統遷移總結

## 📝 改動概要

本次更新將簡單的 Bearer Token 認證（`secret-token-123`）升級為完整的 JWT 認證系統。

---

## 🔄 主要變更

### 1. **新增文件**

#### `api/jwt.go`
JWT 工具函數庫，提供：
- `GenerateJWT()` - 生成 JWT token
- `ValidateJWT()` - 驗證並解析 token
- `RefreshJWT()` - 刷新 token

#### `docs/JWT_AUTH.md`
完整的 JWT 認證系統使用文檔

#### `test-jwt-auth.sh`
自動化測試腳本

### 2. **修改文件**

#### `api/dto.go`
新增認證相關的數據結構：
```go
type RegisterRequest  // 註冊請求
type LoginRequest     // 登入請求
type AuthResponse     // 認證響應（包含 token）
```

#### `api/handlers.go`
新增三個認證處理函數：
```go
func HandleRegister()      // 用戶註冊
func HandleLogin()         // 用戶登入
func HandleRefreshToken()  // 刷新 token
```

#### `api/middleware.go`
更新 `AuthMiddleware`：
- ❌ 舊：檢查硬編碼的 `secret-token-123`
- ✅ 新：驗證 JWT token 並解析用戶信息

#### `api/server.go`
新增路由：
```go
POST /auth/register  // 註冊（公開）
POST /auth/login     // 登入（公開）
POST /auth/refresh   // 刷新（需要認證）
```

#### `store/db.go` & `store/gorm_store.go`
新增方法：
```go
GetUserByEmail(email string) (model.User, error)
```

---

## 🚀 如何使用

### 快速開始

#### 1. 啟動服務器
```bash
cd backend
go run main.go
```

#### 2. 註冊新用戶
```bash
curl -X POST http://localhost:8080/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "username": "myuser",
    "email": "user@example.com",
    "password": "password123"
  }'
```

**響應：**
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "expires_at": "2026-01-13T12:00:00Z",
  "user": {
    "id": "usr_1736683200000000000",
    "username": "myuser",
    "email": "user@example.com"
  }
}
```

#### 3. 使用 Token 訪問 API
```bash
# 保存 token
export TOKEN="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."

# 訪問受保護的端點
curl -X GET http://localhost:8080/devices \
  -H "Authorization: Bearer $TOKEN"
```

#### 4. 運行自動測試
```bash
cd backend
./test-jwt-auth.sh
```

---

## 📋 API 端點對比

### 認證相關（新增）

| 端點 | 方法 | 認證 | 說明 |
|------|------|------|------|
| `/auth/register` | POST | ❌ 不需要 | 註冊新用戶 |
| `/auth/login` | POST | ❌ 不需要 | 用戶登入 |
| `/auth/refresh` | POST | ✅ 需要 | 刷新 token |

### 其他端點（認證方式變更）

所有原有的受保護端點仍然存在，但認證方式改變：

| 端點 | 舊方式 | 新方式 |
|------|--------|--------|
| `/devices` | `Bearer secret-token-123` | `Bearer <JWT-token>` |
| `/users` | `Bearer secret-token-123` | `Bearer <JWT-token>` |
| `/me` | `Bearer secret-token-123` | `Bearer <JWT-token>` |
| 所有其他受保護端點 | `Bearer secret-token-123` | `Bearer <JWT-token>` |

---

## 🔐 認證流程對比

### 舊流程 ❌

```
1. 硬編碼使用 token: "secret-token-123"
2. 每個請求都使用同一個 token
3. 沒有用戶登入/註冊
4. 沒有密碼保護
5. Token 永不過期
```

### 新流程 ✅

```
1. 用戶註冊 → 獲得 JWT token
2. 使用 JWT token 訪問 API
3. Token 5分鐘後過期
4. 可以刷新 token 延長有效期
5. 密碼使用 bcrypt 加密存儲
6. Token 包含用戶身份信息
```

---

## 💡 關鍵技術

### JWT (JSON Web Token)

JWT 是一個包含三部分的字符串：
```
eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.     # Header
eyJ1c2VyX2lkIjoiMTIzIiwiZW1haWwiOiJ0ZXN0  # Payload (用戶信息)
SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV         # Signature (簽名)
```

### 密碼加密

使用 `bcrypt` 算法：
```go
// 註冊時
hashedPassword := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

// 登入時驗證
err := bcrypt.CompareHashAndPassword(hashedPassword, inputPassword)
```

### 中間件驗證

```go
func AuthMiddleware(next http.Handler) http.Handler {
    // 1. 提取 token
    // 2. 驗證 JWT
    // 3. 解析用戶信息
    // 4. 注入到 Context
    // 5. 調用下一個 Handler
}
```

---

## ⚠️ 重要注意事項

### 生產環境配置

#### 1. 更改 JWT Secret

**當前（開發環境）：**
```go
const JWTSecret = "your-secret-key-change-this-in-production"
```

**生產環境應改為：**
```go
var JWTSecret = os.Getenv("JWT_SECRET")
```

然後設置環境變數：
```bash
export JWT_SECRET="your-super-secret-random-key-at-least-32-characters"
```

#### 2. 使用 HTTPS

生產環境**必須**使用 HTTPS 來保護 token 傳輸。

#### 3. 調整 Token 過期時間

根據需求調整過期時間：
```go
// 開發環境：5分鐘
token := GenerateJWT(userID, username, email, 5)

// 生產環境建議：1-2小時 + refresh token
token := GenerateJWT(userID, username, email, 2)
```

#### 4. 添加 Rate Limiting

防止暴力破解登入：
```go
// 建議使用 rate limiting middleware
// 例如：github.com/ulule/limiter
```

---

## 📚 文檔資源

- **完整文檔**：`docs/JWT_AUTH.md`
- **測試腳本**：`test-jwt-auth.sh`
- **Swagger UI**：http://localhost:8080/swagger/

---

## 🧪 測試清單

運行測試腳本或手動測試以下場景：

- [x] 用戶註冊
- [x] 用戶登入
- [x] 使用 token 訪問受保護的端點
- [x] 無效 token 被拒絕
- [x] 錯誤密碼被拒絕
- [x] Token 刷新
- [x] Token 過期處理

---

## 🔧 疑難排解

### 問題 1：編譯錯誤
```bash
# 確保所有依賴已安裝
go mod tidy
go mod download
```

### 問題 2：Token 無效
- 檢查 token 是否包含 "Bearer " 前綴
- 檢查 token 是否過期（5分鐘）
- 確認 JWT_SECRET 沒有被更改

### 問題 3：密碼錯誤
- 密碼最少 6 個字符
- 密碼區分大小寫

---

## 📞 聯繫方式

如有問題，請查看：
1. `docs/JWT_AUTH.md` - 完整文檔
2. `docs/TROUBLESHOOTING.md` - 疑難排解
3. Swagger UI - http://localhost:8080/swagger/

---

## ✅ 總結

### 主要改進

| 特性 | 舊系統 | 新系統 |
|------|--------|--------|
| 認證方式 | 硬編碼 token | JWT token |
| 用戶管理 | ❌ 無 | ✅ 註冊/登入 |
| 密碼保護 | ❌ 無 | ✅ bcrypt 加密 |
| Token 過期 | ❌ 永不過期 | ✅ 5分鐘過期   |
| Token 刷新 | ❌ 不支援 | ✅ 支援 |
| 用戶信息 | ❌ 無 | ✅ 包含在 token 中 |
| 安全性 | ⚠️ 低 | ✅ 高 |

### 新增功能

✅ JWT 認證系統  
✅ 用戶註冊和登入  
✅ 密碼加密存儲  
✅ Token 自動過期  
✅ Token 刷新機制  
✅ 用戶信息管理  

### 向後兼容

⚠️ **不兼容**：舊的 `secret-token-123` 不再有效  
✅ **兼容**：所有其他 API 端點保持不變  
✅ **兼容**：只需更新 Authorization header 即可  

---

**更新日期：** 2026-01-12  
**版本：** v2.0.0 (JWT Authentication)
