package api

import (
	"context"
	"net/http"
	"strings"
)

// 1. 定義私有的 Key 型別，防止外部套件衝突
type contextKey string

// 定義具體的 Key 值
const UserIDKey contextKey = "userID"

// 2. AuthMiddleware: 驗證並注入 User ID
func (s *Server) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// A. 取得 Header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			WriteError(w, http.StatusUnauthorized, "Missing Authorization header")
			return
		}

		// B. 解析 Token (通常格式是 "Bearer <token>")
		// 這裡我們先簡單模擬：假設 Token 必須是 "Bearer secret-token-123"
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			WriteError(w, http.StatusUnauthorized, "Invalid token format")
			return
		}

		token := parts[1]

		// C. 驗證 Token (真實場景這裡會解密 JWT 或查 Redis)
		// 這裡我們模擬：如果 token 是 "secret-token-123"，代表 UserID 是 "user_admin"
		if token != "secret-token-123" {
			WriteError(w, http.StatusUnauthorized, "Invalid token")
			return
		}

		userID := "user_admin" // 模擬解出來的 ID

		// D. 【關鍵】將 UserID 注入 Context
		// r.WithContext 會建立一個新的 Request 副本，並帶有新的 Context
		ctx := context.WithValue(r.Context(), UserIDKey, userID)

		// E. 呼叫下一個 Handler，並傳入帶有新 Context 的 Request
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetUserIDFromContext 是一個 Helper，方便 Handler 取得當前用戶 ID
func GetUserIDFromContext(ctx context.Context) (string, bool) {
	// 從 Context 拿出來的是 interface{}，必須斷言成 string
	userID, ok := ctx.Value(UserIDKey).(string)
	return userID, ok
}
