# JWT 認證系統使用說明

本文檔詳細說明如何使用新的 JWT (JSON Web Token) 認證系統。

## 📋 目錄

1. [系統概覽](#系統概覽)
2. [主要變更](#主要變更)
3. [API 端點](#api-端點)
4. [使用流程](#使用流程)
5. [技術實現](#技術實現)
6. [安全注意事項](#安全注意事項)

---

## 系統概覽

之前的認證系統使用硬編碼的 token (`secret-token-123`)，現在已經升級為基於 JWT 的完整認證系統。

### 主要優勢

- ✅ **安全性提升**：使用加密的 JWT token，支援過期時間
- ✅ **密碼保護**：使用 bcrypt 加密存儲用戶密碼
- ✅ **標準化**：遵循業界標準的 JWT 認證流程
- ✅ **可擴展**：支援 token 刷新、撤銷等功能
- ✅ **用戶信息**：Token 中包含用戶身份信息

---

## 主要變更

### 1. 新增的文件

#### `api/jwt.go` - JWT 工具函數
```go
// 主要功能：
- GenerateJWT()    // 生成 JWT token
- ValidateJWT()    // 驗證並解析 JWT token
- RefreshJWT()     // 刷新 token
```

### 2. 更新的文件

#### `api/dto.go` - 新增認證相關的 DTO
```go
type RegisterRequest struct {
    Username string `json:"username"`
    Email    string `json:"email"`
    Password string `json:"password"`
}

type LoginRequest struct {
    Email    string `json:"email"`
    Password string `json:"password"`
}

type AuthResponse struct {
    Token     string       `json:"token"`
    ExpiresAt time.Time    `json:"expires_at"`
    User      UserResponse `json:"user"`
}
```

#### `api/handlers.go` - 新增認證相關的 Handler
- `HandleRegister()` - 處理用戶註冊
- `HandleLogin()` - 處理用戶登入
- `HandleRefreshToken()` - 處理 token 刷新

#### `api/middleware.go` - 更新認證中間件
- 從簡單的 token 驗證升級為 JWT 驗證
- 自動解析 JWT 並注入用戶信息到 Context

#### `api/server.go` - 新增認證路由
```go
// 公開路由（不需要認證）
POST /auth/register  // 用戶註冊
POST /auth/login     // 用戶登入

// 私有路由（需要 JWT token）
POST /auth/refresh   // 刷新 token
GET  /me            // 獲取當前用戶信息
```

#### `store/db.go` 和 `store/gorm_store.go` - 新增用戶查詢方法
- `GetUserByEmail()` - 通過 email 查找用戶（用於登入）

### 3. 依賴變更

新增的 Go 包：
- `golang.org/x/crypto/bcrypt` - 密碼加密
- `github.com/golang-jwt/jwt/v5` - JWT 處理（已存在）

---

## API 端點

### 1. 用戶註冊

**端點：** `POST /auth/register`

**請求示例：**
```bash
curl -X POST http://localhost:8080/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "username": "john_doe",
    "email": "john@example.com",
    "password": "SecurePassword123"
  }'
```

**響應示例：**
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "expires_at": "2026-01-13T12:00:00Z",
  "user": {
    "id": "usr_1736683200000000000",
    "username": "john_doe",
    "email": "john@example.com",
    "created_at": "2026-01-12T12:00:00Z"
  }
}
```

**注意事項：**
- 密碼長度至少 6 個字符
- Email 必須唯一（不能重複註冊）
- 註冊成功後會自動返回 JWT token

### 2. 用戶登入

**端點：** `POST /auth/login`

**請求示例：**
```bash
curl -X POST http://localhost:8080/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "john@example.com",
    "password": "SecurePassword123"
  }'
```

**響應示例：**
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "expires_at": "2026-01-13T12:00:00Z",
  "user": {
    "id": "usr_1736683200000000000",
    "username": "john_doe",
    "email": "john@example.com",
    "created_at": "2026-01-12T12:00:00Z"
  }
}
```

**錯誤處理：**
- `401 Unauthorized` - Email 或密碼錯誤

### 3. 刷新 Token

**端點：** `POST /auth/refresh`

**請求示例：**
```bash
curl -X POST http://localhost:8080/auth/refresh \
  -H "Authorization: Bearer <your-old-token>"
```

**響應示例：**
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "expires_at": "2026-01-13T12:00:00Z",
  "user": {
    "id": "usr_1736683200000000000",
    "username": "john_doe",
    "email": "john@example.com",
    "created_at": "2026-01-12T12:00:00Z"
  }
}
```

### 4. 使用 JWT Token 訪問受保護的端點

所有需要認證的端點都需要在 Header 中包含 JWT token：

**請求示例：**
```bash
curl -X GET http://localhost:8080/devices \
  -H "Authorization: Bearer <your-jwt-token>"
```

**受保護的端點列表：**
- `GET /me` - 獲取當前用戶信息
- `GET /users` - 獲取用戶列表
- `GET /users/{id}` - 獲取特定用戶
- `GET /devices` - 獲取設備列表
- `POST /devices` - 創建設備
- `GET /devices/{id}` - 獲取特定設備
- `PUT /devices/{id}` - 更新設備
- `PATCH /devices/{id}` - 部分更新設備
- `DELETE /devices/{id}` - 刪除設備
- `GET /telemetries` - 獲取遙測數據
- `POST /telemetries` - 創建遙測數據
- 等等...

---

## 使用流程

### 完整的使用流程

```
1. 用戶註冊
   POST /auth/register
   ↓
   收到 JWT token

2. 使用 token 訪問 API
   GET /devices
   Header: Authorization: Bearer <token>
   ↓
   獲取數據

3. Token 快過期時刷新
   POST /auth/refresh
   Header: Authorization: Bearer <old-token>
   ↓
   收到新的 JWT token

4. 繼續使用新 token 訪問 API
```

### Postman 使用示例

#### 步驟 1：註冊用戶
1. 新建請求：`POST http://localhost:8080/auth/register`
2. 選擇 Body → raw → JSON
3. 輸入：
```json
{
  "username": "testuser",
  "email": "test@example.com",
  "password": "test123456"
}
```
4. 發送請求
5. 複製響應中的 `token` 值

#### 步驟 2：使用 Token 訪問 API
1. 新建請求：`GET http://localhost:8080/devices`
2. 選擇 Authorization → Type: Bearer Token
3. 貼上剛才複製的 token
4. 發送請求

#### 步驟 3：測試其他端點
使用同樣的 token 測試其他需要認證的端點。

---

## 技術實現

### JWT Token 結構

JWT token 包含以下信息：

```json
{
  "user_id": "usr_1736683200000000000",
  "username": "john_doe",
  "email": "john@example.com",
  "exp": 1736683200,
  "iat": 1736596800,
  "nbf": 1736596800,
  "iss": "my-api",
  "sub": "usr_1736683200000000000",
  "jti": "usr_1736683200000000000-20260112120000"
}
```

**字段說明：**
- `user_id`, `username`, `email` - 用戶信息（自定義 claims）
- `exp` (Expires At) - 過期時間（5分鐘後）
- `iat` (Issued At) - 簽發時間
- `nbf` (Not Before) - 生效時間
- `iss` (Issuer) - 簽發者（"my-api"）
- `sub` (Subject) - 主題（用戶 ID）
- `jti` (JWT ID) - Token 唯一標識符

### 密碼安全

使用 `bcrypt` 進行密碼加密：

```go
// 註冊時加密
hashedPassword, _ := bcrypt.GenerateFromPassword(
    []byte(password), 
    bcrypt.DefaultCost
)

// 登入時驗證
err := bcrypt.CompareHashAndPassword(
    []byte(hashedPassword), 
    []byte(inputPassword)
)
```

### 中間件工作流程

```
HTTP Request
    ↓
1. 檢查 Authorization Header
    ↓
2. 解析 "Bearer <token>"
    ↓
3. 驗證 JWT token
    - 檢查簽名
    - 檢查過期時間
    - 解析 claims
    ↓
4. 將 UserID 注入 Context
    ↓
5. 調用下一個 Handler
    ↓
Handler 可以通過 GetUserIDFromContext() 獲取當前用戶 ID
```

---

## 安全注意事項

### ⚠️ 重要：生產環境配置

#### 1. 修改 JWT Secret

在 `api/jwt.go` 中，將 JWT Secret 改為環境變數：

```go
// ❌ 不安全（當前實現）
const JWTSecret = "your-secret-key-change-this-in-production"

// ✅ 安全（推薦）
var JWTSecret = os.Getenv("JWT_SECRET")
```

**在生產環境中設置環境變數：**
```bash
export JWT_SECRET="your-very-long-and-random-secret-key-here"
```

#### 2. 使用 HTTPS

在生產環境中，**必須**使用 HTTPS 來保護 token 傳輸：

```go
// 在 server 配置中
server := &http.Server{
    Addr:    ":443",
    Handler: router,
    TLSConfig: &tls.Config{
        MinVersion: tls.VersionTLS12,
    },
}
server.ListenAndServeTLS("cert.pem", "key.pem")
```

#### 3. Token 過期時間

根據應用需求調整 token 過期時間：

```go
// 當前設置：5分鐘（開發測試用）
token, _ := GenerateJWT(userID, username, email, 5)

// 建議：
// - 開發環境：5-15 分鐘（方便測試）
// - 生產環境：60-120 分鐘（1-2小時）
// - 配合 refresh token 機制
token, _ := GenerateJWT(userID, username, email, 60)  // 1小時
```

#### 4. 密碼策略

強化密碼要求：

```go
// 當前：最少 6 個字符
if len(req.Password) < 6 {
    return errors.New("密碼太短")
}

// 建議：
// - 最少 8-12 個字符
// - 包含大小寫字母、數字、特殊字符
// - 使用密碼強度檢查庫
```

#### 5. Token 撤銷機制

實現 token 黑名單（使用 Redis）：

```go
// 用戶登出時，將 token 加入黑名單
func (s *Server) HandleLogout(w http.ResponseWriter, r *http.Request) {
    token := extractToken(r)
    claims, _ := ValidateJWT(token)
    
    // 將 token JTI 存入 Redis，過期時間與 token 相同
    s.Redis.Set(
        "blacklist:"+claims.ID, 
        "true", 
        time.Until(claims.ExpiresAt.Time)
    )
}

// 中間件中檢查黑名單
func (s *Server) AuthMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // ... 驗證 JWT ...
        
        // 檢查是否在黑名單中
        val, err := s.Redis.Get("blacklist:" + claims.ID).Result()
        if err == nil && val == "true" {
            WriteError(w, http.StatusUnauthorized, "Token has been revoked")
            return
        }
        
        // ... 繼續處理 ...
    })
}
```

#### 6. Rate Limiting

防止暴力破解，添加速率限制：

```go
// 限制登入嘗試次數
// 可以使用 Redis 或 github.com/ulule/limiter
```

---

## 測試範例

### 完整測試流程

```bash
# 1. 啟動服務器
cd backend
go run main.go

# 2. 註冊新用戶
curl -X POST http://localhost:8080/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "username": "testuser",
    "email": "test@example.com",
    "password": "password123"
  }'

# 保存返回的 token
export TOKEN="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."

# 3. 使用 token 訪問受保護的端點
curl -X GET http://localhost:8080/me \
  -H "Authorization: Bearer $TOKEN"

# 4. 測試設備端點
curl -X GET http://localhost:8080/devices \
  -H "Authorization: Bearer $TOKEN"

# 5. 測試錯誤情況（無 token）
curl -X GET http://localhost:8080/devices
# 應返回 401 Unauthorized

# 6. 測試錯誤情況（錯誤的 token）
curl -X GET http://localhost:8080/devices \
  -H "Authorization: Bearer invalid-token"
# 應返回 401 Unauthorized

# 7. 刷新 token
curl -X POST http://localhost:8080/auth/refresh \
  -H "Authorization: Bearer $TOKEN"
```

---

## 常見問題 (FAQ)

### Q1: 舊的 `secret-token-123` 還能用嗎？
**A:** 不能。系統已經完全切換到 JWT 認證。所有請求必須使用新的 JWT token。

### Q2: Token 過期後怎麼辦？
**A:** 使用 `/auth/refresh` 端點刷新 token，或者重新登入。

### Q3: 如何在前端存儲 token？
**A:** 
- 推薦：使用 `httpOnly` cookie（更安全）
- 備選：localStorage 或 sessionStorage（需注意 XSS 風險）

### Q4: 可以同時使用多個 token 嗎？
**A:** 可以。每次登入都會生成新的 token，舊的 token 在過期前仍然有效（除非實現了 token 撤銷機制）。

### Q5: 如何實現「記住我」功能？
**A:** 為「記住我」選項生成較長過期時間的 token（如 7天或30天）。

---

## 遷移指南

### 從舊系統遷移

如果你之前使用 `Authorization: Bearer secret-token-123`：

**步驟 1：** 註冊或登入獲取 JWT token
```bash
curl -X POST http://localhost:8080/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email": "your@email.com", "password": "yourpassword"}'
```

**步驟 2：** 替換所有 API 請求中的 token
```bash
# 舊方式 ❌
curl -H "Authorization: Bearer secret-token-123" ...

# 新方式 ✅
curl -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..." ...
```

**步驟 3：** 實現 token 刷新邏輯
在你的客戶端代碼中，當收到 `401 Unauthorized` 時：
1. 嘗試刷新 token
2. 如果刷新失敗，重新導向到登入頁面

---

## 總結

✅ 已完成的改進：
- JWT 認證系統
- 密碼加密存儲
- 用戶註冊和登入
- Token 刷新機制
- 安全的中間件

🚀 建議的下一步改進：
- 實現 token 撤銷（黑名單）
- 添加速率限制
- 實現「記住我」功能
- 添加 Email 驗證
- 實現密碼重置功能
- 添加用戶角色和權限管理

---

## 相關文檔

- [API 文檔](./API.md)
- [Swagger UI](http://localhost:8080/swagger/)
- [WebSocket 文檔](./WEBSOCKET.md)
- [部署文檔](./DEPLOYMENT.md)

---

**最後更新：** 2026-01-12
**作者：** Your Team
