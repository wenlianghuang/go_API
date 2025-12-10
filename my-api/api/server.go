package api

import (
	"my-api/store"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// Server 結構體持有所有的依賴 (Router 和 Storage)
type Server struct {
	Router *chi.Mux
	Store  store.Storage // 注意：這裡依賴的是 Storage Interface，而不是具體的 struct
}

// NewServer 初始化 Server 並掛載路由
func NewServer(store store.Storage) *Server {
	s := &Server{
		Router: chi.NewRouter(),
		Store:  store,
	}

	s.mountRoutes()
	return s
}

func (s *Server) mountRoutes() {
	s.Router.Use(middleware.Logger)
	s.Router.Use(middleware.Recoverer)
	s.Router.Use(middleware.RequestID)

	// === 1. 公開路由 (Public Routes) ===
	// 任何人都可以訪問，不需要 Token
	s.Router.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Welcome to the Mortgage System API"))
	})

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

		// Telemetry 相關路由（需要認證）
		r.Post("/telemetries", s.HandleCreateTelemetry)
	})
}
