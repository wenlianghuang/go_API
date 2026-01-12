# 🚀 JWT 認證快速開始指南

## 1️⃣ 啟動服務器

```bash
cd backend
go run main.go
```

## 2️⃣ 註冊用戶

```bash
curl -X POST http://localhost:8080/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "username": "myuser",
    "email": "user@example.com",
    "password": "password123"
  }'
```

**保存返回的 token！**

## 3️⃣ 使用 Token

```bash
# 設置 token 環境變數（替換成你的實際 token）
export TOKEN="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."

# 訪問受保護的端點
curl -H "Authorization: Bearer $TOKEN" http://localhost:8080/devices
curl -H "Authorization: Bearer $TOKEN" http://localhost:8080/me
```

## 4️⃣ 或使用登入

如果已經註冊過：

```bash
curl -X POST http://localhost:8080/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "password123"
  }'
```

## 5️⃣ 自動化測試

```bash
./test-jwt-auth.sh
```

## 📌 重要端點

| 端點 | 方法 | 需要認證 | 說明 |
|------|------|----------|------|
| `/auth/register` | POST | ❌ | 註冊新用戶 |
| `/auth/login` | POST | ❌ | 用戶登入 |
| `/auth/refresh` | POST | ✅ | 刷新 token |
| `/me` | GET | ✅ | 獲取當前用戶信息 |
| `/devices` | GET | ✅ | 獲取設備列表 |
| `/users` | GET | ✅ | 獲取用戶列表 |

## ⚠️ 注意事項

- Token 有效期：**5分鐘**
- 密碼最少：**6個字符**
- 格式：`Authorization: Bearer <token>`

## 📚 完整文檔

- 詳細說明：`docs/JWT_AUTH.md`
- 遷移總結：`JWT_MIGRATION_SUMMARY.md`
- Swagger UI：http://localhost:8080/swagger/

## 💡 Postman 使用

1. 註冊/登入獲取 token
2. 在 Authorization 選項卡選擇 "Bearer Token"
3. 貼上 token
4. 發送請求

---

**就這麼簡單！🎉**
