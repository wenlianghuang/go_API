package api

import (
	"fmt"
	"my-api/store"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/redis/go-redis/v9"
	httpSwagger "github.com/swaggo/http-swagger"
)

// Server 結構體持有所有的依賴 (Router 和 Storage)
type Server struct {
	Router   *chi.Mux
	Store    store.Storage // 注意：這裡依賴的是 Storage Interface，而不是具體的 struct
	TaskChan chan uint     // 存放設備 ID 的任務通道
	Hub      *Hub          // 存放 WebSocket 的 Hub
}

// NewServer 初始化 Server 並掛載路由
func NewServer(store store.Storage) *Server {
	// 1. 初始化 Redis
	rdb := redis.NewClient(&redis.Options{
		Addr: "redis:6379", // Docker 內部的 service name
	})
	s := &Server{
		Router:   chi.NewRouter(),
		Store:    store,
		TaskChan: make(chan uint, 100), // 設定通道大小為 100
		Hub:      NewHub(rdb),
	}

	go s.startWorker()

	s.mountRoutes()
	return s
}

// startWorker 是一個永遠在背景運行的消費者
func (s *Server) startWorker() {
	fmt.Println("👷 Worker 已啟動，等待任務中...")
	for id := range s.TaskChan { // 不斷從通道拿 ID
		fmt.Printf("👷 Worker 正在處理設備 ID: %d\n", id)
		time.Sleep(5 * time.Second) // 模擬耗時計算
		fmt.Printf("👷 Worker 處理 ID: %d 完成\n", id)
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
		w.Write([]byte("Welcome to the Mortgage System API"))
	})

	// Swagger UI 路由
	s.Router.Get("/swagger/*", httpSwagger.WrapHandler)

	// 假設註冊也是公開的
	s.Router.Post("/users", s.HandleCreateUser)

	// === 2. 私有路由 (Private Routes) ===
	// 這裡面的所有路由，都會先經過 AuthMiddleware
	s.Router.Group(func(r chi.Router) {
		// 掛載中間件
		r.Use(s.AuthMiddleware)

		// User 相關路由
		r.Get("/users", s.HandleListUsers)    // 只有管理員能看列表
		r.Get("/users/{id}", s.HandleGetUser) // 只有管理員能查詳情
		r.Get("/me", s.HandleMe)              // 測試 Context 注入用

		// Device 相關路由（所有端點都需要認證）
		r.Post("/devices", s.HandleCreateDevice)
		r.Get("/devices", s.HandleListDevices)
		r.Get("/devices/{id}", s.HandleGetDevice)
		r.Put("/devices/{id}", s.HandleUpdateDevice)  // PUT: 完整更新（所有字段必須提供）
		r.Patch("/devices/{id}", s.HandlePatchDevice) // PATCH: 部分更新（只需提供要更新的字段）

		// 註冊 DELETE 路由
		r.Delete("/devices/{id}", s.HandleDeleteDevice)
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
