package api

import (
	"context"
	"fmt"
	"my-api/service"
	"my-api/store"
	"net/http"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/redis/go-redis/v9"
	httpSwagger "github.com/swaggo/http-swagger"
)

// Server 結構體持有所有的依賴 (Router 和 Storage)
type Server struct {
	Router           *chi.Mux
	Store            store.Storage             // 注意：這裡依賴的是 Storage Interface，而不是具體的 struct
	AuthService      *service.AuthService      // 認證服務
	DeviceService    *service.DeviceService    // 設備服務
	TelemetryService *service.TelemetryService // 遙測數據服務
	TaskChan         chan uint                 // 存放設備 ID 的任務通道
	Hub              *Hub                      // 存放 WebSocket 的 Hub

	// 🆕 優雅停機支援
	workerWg     sync.WaitGroup     // 用於等待 worker goroutine 完成
	workerCtx    context.Context    // 控制 worker 的生命週期
	workerCancel context.CancelFunc // 用於取消 context，觸發 worker 停機
}

// NewServer 初始化 Server 並掛載路由
func NewServer(store store.Storage, redisAddr string) *Server {
	// 1. 初始化 Redis
	rdb := redis.NewClient(&redis.Options{
		Addr: redisAddr, // 使用傳入的 Redis 位址
	})

	// 🆕 創建可取消的 context，用於控制 worker 停機
	ctx, cancel := context.WithCancel(context.Background())

	s := &Server{
		Router:           chi.NewRouter(),
		Store:            store,
		AuthService:      service.NewAuthService(store, NewJWTService()),
		DeviceService:    service.NewDeviceService(store),
		TelemetryService: service.NewTelemetryService(store),
		TaskChan:         make(chan uint, 100), // 設定通道大小為 100
		Hub:              NewHub(rdb),
		workerCtx:        ctx,    // 🆕 儲存 context
		workerCancel:     cancel, // 🆕 儲存 cancel 函數
	}

	// 🆕 使用 WaitGroup 追蹤 worker goroutine，以便停機時等待
	s.workerWg.Add(1)
	go s.startWorker()

	s.mountRoutes()
	return s
}

// startWorker 是一個永遠在背景運行的消費者
// 🆕 支援優雅停機：當 s.workerCtx 被取消或 TaskChan 關閉時，會自動退出
func (s *Server) startWorker() {
	// 🆕 確保在函數退出時標記 WaitGroup 完成
	defer s.workerWg.Done()
	fmt.Println("👷 Worker 已啟動，等待任務中...")

	// 🆕 使用 select 監聽 context 取消信號和任務通道
	for {
		select {
		case <-s.workerCtx.Done():
			// 🆕 收到停機信號，處理完 channel 中剩餘的任務後退出
			fmt.Println("🛑 Worker 收到停機信號，處理剩餘任務中...")
			// 持續從 TaskChan 讀取，直到 channel 被關閉
			for id := range s.TaskChan {
				fmt.Printf("👷 Worker 正在處理剩餘任務，設備 ID: %d\n", id)
				time.Sleep(5 * time.Second) // 模擬耗時計算
				fmt.Printf("👷 Worker 處理 ID: %d 完成\n", id)
			}
			fmt.Println("✅ Worker 已處理完所有剩餘任務")
			return

		case id, ok := <-s.TaskChan:
			// 🆕 檢查 channel 是否已關閉
			if !ok {
				fmt.Println("✅ TaskChan 已關閉，Worker 退出")
				return
			}
			// 正常處理任務
			fmt.Printf("👷 Worker 正在處理設備 ID: %d\n", id)
			time.Sleep(5 * time.Second) // 模擬耗時計算
			fmt.Printf("👷 Worker 處理 ID: %d 完成\n", id)
		}
	}
}

func (s *Server) mountRoutes() {
	// 注意：WebSocket 路由 (/ws) 不在這裡註冊
	// 因為 WebSocket 升級需要直接訪問原始 ResponseWriter（實現 http.Hijacker）
	// Chi 的中間件會包裝 ResponseWriter，導致無法實現 Hijacker 接口
	// WebSocket 將在 main.go 中使用標準 http.ServeMux 單獨處理

	// 其他中間件（應用於所有其他路由）
	s.Router.Use(middleware.Logger)
	s.Router.Use(middleware.Recoverer)
	s.Router.Use(middleware.RequestID)
	s.Router.Use(MetricsMiddleware) // Add metrics middleware

	// === 1. 公開路由 (Public Routes) ===
	// 任何人都可以訪問，不需要 Token
	s.Router.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Welcome to the IoT API System"))
	})

	// Swagger UI 路由
	s.Router.Get("/swagger/*", httpSwagger.WrapHandler)

	// === 認證路由 (Authentication Routes) ===
	// 這些路由是公開的，用於用戶註冊和登入
	s.Router.Post("/auth/register", s.HandleRegister) // 用戶註冊
	s.Router.Post("/auth/login", s.HandleLogin)       // 用戶登入

	// 舊的用戶創建路由（已廢棄，保留用於向後兼容）
	s.Router.Post("/users", s.HandleCreateUser)

	// === 2. 私有路由 (Private Routes) ===
	// 這裡面的所有路由，都會先經過 AuthMiddleware（需要有效的 JWT token）
	s.Router.Group(func(r chi.Router) {
		// 掛載 JWT 驗證中間件
		r.Use(s.AuthMiddleware)

		// === 認證相關路由（需要已登入） ===
		r.Post("/auth/refresh", s.HandleRefreshToken) // 刷新 token

		// === User 相關路由（需要已登入） ===
		r.Get("/users", s.HandleListUsers)    // 獲取所有用戶列表
		r.Get("/users/{id}", s.HandleGetUser) // 獲取特定用戶資訊
		r.Get("/me", s.HandleMe)              // 獲取當前登入用戶資訊

		// Device 相關路由（所有端點都需要認證）
		r.Post("/devices", s.HandleCreateDevice)
		r.Get("/devices", s.HandleListDevices)
		r.Get("/devices/{id}", s.HandleGetDevice)
		r.Put("/devices/{id}", s.HandleUpdateDevice)  // PUT: 完整更新（所有字段必須提供）
		r.Patch("/devices/{id}", s.HandlePatchDevice) // PATCH: 部分更新（只需提供要更新的字段）

		// 註冊 DELETE 路由
		r.Delete("/devices/{id}", s.HandleDeleteDevice)

		// Get all telemetries
		r.Get("/telemetries", s.HandleListTelemetries)
		// Get telemetry
		r.Get("/telemetries/{id}", s.HandleGetTelemetry)
		// Telemetry 相關路由（需要認證）
		r.Post("/telemetries", s.HandleCreateTelemetry)
		// Patch telemetry
		r.Patch("/telemetries/{id}", s.HandlePatchTelemetry)
		// Test Analyze Device
		r.Post("/devices/{id}/analyze", s.HandleAnalyzeDevice)

	})
}

// 🆕 Shutdown 優雅停機：關閉所有背景任務
// 此方法會按順序：
// 1. 通知 worker 不再接受新任務（取消 context）
// 2. 關閉 TaskChan，讓 worker 知道不會有新任務了
// 3. 等待 worker 處理完現有任務
// 4. 關閉 WebSocket Hub（包括所有連接和 Redis 監聽器）
// 5. 關閉 Redis 連接
func (s *Server) Shutdown(ctx context.Context) error {
	fmt.Println("🔄 正在關閉 Server 背景任務...")

	// 步驟 1：取消 worker context，通知不再接受新任務
	s.workerCancel()

	// 步驟 2：關閉 TaskChan（讓 worker 知道不會有新任務了）
	close(s.TaskChan)

	// 步驟 3：等待 worker 完成（帶超時保護）
	workerDone := make(chan struct{})
	go func() {
		s.workerWg.Wait() // 等待所有 worker goroutine 完成
		close(workerDone)
	}()

	select {
	case <-workerDone:
		fmt.Println("✅ Worker 已安全關閉")
	case <-ctx.Done():
		// 超時了，但我們會繼續關閉其他組件
		fmt.Println("⚠️ Worker 停機超時（但已發送停止信號）")
	}

	// 步驟 4：關閉 WebSocket Hub（包括所有連接和 Redis 監聽器）
	if err := s.Hub.Shutdown(ctx); err != nil {
		// 記錄錯誤但繼續關閉流程
		fmt.Printf("⚠️ Hub 停機時發生錯誤: %v\n", err)
	}

	// 步驟 5：關閉 Redis 連接
	fmt.Println("🔄 正在關閉 Redis 連接...")
	if err := s.Hub.rdb.Close(); err != nil {
		return fmt.Errorf("Redis 關閉失敗: %w", err)
	}
	fmt.Println("✅ Redis 連接已關閉")

	fmt.Println("✅ Server 背景任務已全部關閉")
	return nil
}
