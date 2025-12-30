package api

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
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

// MetricsMiddleware records HTTP request metrics
func MetricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// 檢測是否為 WebSocket 升級請求
		// WebSocket 升級需要直接訪問原始 ResponseWriter，不能包裝
		isWebSocket := r.Header.Get("Upgrade") == "websocket" ||
			strings.Contains(strings.ToLower(r.Header.Get("Connection")), "upgrade") ||
			r.URL.Path == "/ws"

		var ww http.ResponseWriter
		var statusCode int

		if isWebSocket {
			// WebSocket 請求：使用原始 ResponseWriter，不包裝
			// 這樣 WebSocket.Upgrade() 可以正確設置響應頭
			ww = w
			statusCode = http.StatusSwitchingProtocols // 101 Switching Protocols
		} else {
			// 一般 HTTP 請求：包裝 ResponseWriter 以捕獲狀態碼
			ww = &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
			statusCode = http.StatusOK
		}

		// Call the next handler
		next.ServeHTTP(ww, r)

		// 獲取路由模式
		routePattern := chi.RouteContext(r.Context()).RoutePattern()
		if routePattern == "" {
			routePattern = r.URL.Path
		}

		if isWebSocket {
			// WebSocket 連線：記錄連線建立（使用 101 Switching Protocols）
			// 不記錄持續時間，因為 WebSocket 是持久連線
			// 連線數由 websocket_active_connections gauge 追蹤
			RequestCounter.WithLabelValues(r.Method, routePattern, "Switching Protocols").Inc()
		} else {
			// 一般 HTTP 請求：記錄完整的 metrics
			duration := time.Since(start).Seconds()

			// 獲取實際狀態碼（從包裝的 ResponseWriter）
			if rw, ok := ww.(*responseWriter); ok {
				statusCode = rw.statusCode
			}

			// 記錄 metrics
			RequestCounter.WithLabelValues(r.Method, routePattern, http.StatusText(statusCode)).Inc()
			RequestDuration.WithLabelValues(r.Method, routePattern).Observe(duration)
		}
	})
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}
