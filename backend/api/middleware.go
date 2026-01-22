package api

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"
)

// 1. 定義私有的 Key 型別，防止外部套件衝突
type contextKey string

// 定義具體的 Key 值
const UserIDKey contextKey = "userID"

// 2. AuthMiddleware: 驗證 JWT Token 並注入 User ID
func (s *Server) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// A. 取得 Authorization Header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			WriteError(w, http.StatusUnauthorized, "Missing Authorization header")
			return
		}

		// B. 解析 Token (格式必須是 "Bearer <JWT-token>")
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			WriteError(w, http.StatusUnauthorized, "Invalid token format, expected 'Bearer <token>'")
			return
		}

		tokenString := parts[1]

		// C. 驗證並解析 JWT Token（使用 Server 的 JWTService）
		claims, err := s.AuthService.JWTGenerator.ValidateJWT(tokenString)
		if err != nil {
			WriteError(w, http.StatusUnauthorized, "Invalid or expired token: "+err.Error())
			return
		}

		// D. 【關鍵】將解析出的 UserID 注入到 Context 中
		// 這樣後續的 Handler 就可以通過 GetUserIDFromContext 取得當前用戶 ID
		ctx := context.WithValue(r.Context(), UserIDKey, claims.GetUserID())

		// 可選：也可以將完整的 claims 注入 context（如果需要 username, email 等）
		// 這裡我們只注入 UserID，保持簡單

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

// ZapHTTPLogger logs one line per HTTP request using zap.
//
// Notes:
// - WebSocket requests are handled in main.go and bypass chi middleware.
// - This middleware uses chi's RequestID middleware (if enabled) to attach request_id.
func ZapHTTPLogger(log *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			ww := &statusCapturingResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}
			next.ServeHTTP(ww, r)

			routePattern := chi.RouteContext(r.Context()).RoutePattern()
			if routePattern == "" {
				routePattern = r.URL.Path
			}

			fields := []zap.Field{
				zap.String("method", r.Method),
				zap.String("route", routePattern),
				zap.String("path", r.URL.Path),
				zap.Int("status", ww.statusCode),
				zap.Duration("duration", time.Since(start)),
			}

			if reqID := chimiddleware.GetReqID(r.Context()); reqID != "" {
				fields = append(fields, zap.String("request_id", reqID))
			}

			// Best-effort client IP
			if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
				fields = append(fields, zap.String("x_forwarded_for", ip))
			} else if ip := r.Header.Get("X-Real-IP"); ip != "" {
				fields = append(fields, zap.String("x_real_ip", ip))
			} else if r.RemoteAddr != "" {
				// RemoteAddr is "IP:port"
				fields = append(fields, zap.String("remote_addr", r.RemoteAddr))
			}

			// If response size is known, include it (Write() tracked)
			if ww.bytesWritten > 0 {
				fields = append(fields, zap.Int64("bytes", ww.bytesWritten))
			}

			// Log level by status code
			switch {
			case ww.statusCode >= 500:
				log.Error("http request completed", fields...)
			case ww.statusCode >= 400:
				log.Warn("http request completed", fields...)
			default:
				log.Info("http request completed", fields...)
			}
		})
	}
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

type statusCapturingResponseWriter struct {
	http.ResponseWriter
	statusCode    int
	bytesWritten  int64
	headerWritten bool
}

func (w *statusCapturingResponseWriter) WriteHeader(code int) {
	w.statusCode = code
	w.headerWritten = true
	w.ResponseWriter.WriteHeader(code)
}

func (w *statusCapturingResponseWriter) Write(p []byte) (int, error) {
	if !w.headerWritten {
		w.WriteHeader(http.StatusOK)
	}
	n, err := w.ResponseWriter.Write(p)
	w.bytesWritten += int64(n)
	return n, err
}

// Optional: if the underlying ResponseWriter supports Hijacker (websocket), preserve it.
// (Chi middlewares can break Hijacker support if they wrap ResponseWriter incorrectly.)
// This wrapper preserves Hijacker only by embedding; if needed, implement explicit Hijack forwarding.
var _ http.ResponseWriter = (*statusCapturingResponseWriter)(nil)
