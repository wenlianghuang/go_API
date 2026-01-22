package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"my-api/api" // 假設這是你放 Server 的地方
	"my-api/config"
	"my-api/logger"
	"my-api/store" // 資料存取層

	_ "my-api/docs" // swagger docs

	"go.uber.org/zap"
)

// @title           IoT API
// @version         1.0
// @description     IoT 設備管理系統 API 文檔
// @termsOfService  http://swagger.io/terms/

// @contact.name   API Support
// @contact.url    http://www.swagger.io/support
// @contact.email  support@swagger.io

// @license.name  Apache 2.0
// @license.url   http://www.apache.org/licenses/LICENSE-2.0.html

// @host      localhost:8080
// @BasePath  /

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.

func main() {
	// 1. 載入配置（快速失敗）
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("無法載入配置: %v", err)
	}

	// 1.1 初始化 structured logger（zap）
	zlog, err := logger.New(cfg)
	if err != nil {
		log.Fatalf("無法初始化 logger: %v", err)
	}
	defer func() {
		_ = zlog.Sync()
	}()

	// 2. 連接資料庫
	db, err := gorm.Open(postgres.Open(cfg.Database.DSN), &gorm.Config{})
	if err != nil {
		zlog.Fatal("無法連接資料庫", zap.Error(err))
	}
	zlog.Info("成功連接到 PostgreSQL")

	// 3. 資料庫遷移現在使用 golang-migrate
	// 請在啟動應用程式前執行：./scripts/docker-migrate.sh up

	// 4. 設定連線池 (Connection Pool) - 生產環境必備
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatal(err)
	}
	sqlDB.SetMaxIdleConns(10)  // 空閒時保留10個連線
	sqlDB.SetMaxOpenConns(100) // 高流量時最多開100個連線

	// 5. 初始化 Store (使用 GormStore)
	gormStore, err := store.NewGormStore(db)
	if err != nil {
		zlog.Fatal("無法初始化 GormStore", zap.Error(err))
	}

	// 6. 初始化 Server (注入配置)
	srv := api.NewServer(gormStore, cfg, zlog)

	// 8. 創建組合的 HTTP handler，將 WebSocket 和 API 路由組合
	// 使用自定義的 handler 來確保 WebSocket 路由優先處理
	mainHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// WebSocket 路由：優先處理，不經過任何中間件
		if r.URL.Path == "/ws" {
			srv.HandleWS(w, r)
			return
		}
		// 其他路由：使用 Chi router（包含所有中間件）
		srv.Router.ServeHTTP(w, r)
	})

	// 7. 創建 HTTP Server 實例（使用 http.Server 物件）
	apiServer := &http.Server{
		Addr:         ":" + cfg.App.APIPort,
		Handler:      mainHandler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	metricsMux := http.NewServeMux()
	metricsMux.Handle("/metrics", api.MetricsHandler())
	metricsServer := &http.Server{
		Addr:         ":" + cfg.App.MetricsPort,
		Handler:      metricsMux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// 8. 設定系統信號監聽
	// 創建一個 channel 來接收系統信號
	quit := make(chan os.Signal, 1)
	// 監聽 SIGINT (Ctrl+C) 和 SIGTERM (Docker/K8s stop)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// 9. 啟動兩個獨立的 server（使用 goroutine）
	var wg sync.WaitGroup
	wg.Add(2)

	// API Server - 處理所有業務請求（包含 WebSocket）
	go func() {
		defer wg.Done()
		zlog.Info("API server listening", zap.String("addr", ":"+cfg.App.APIPort))
		if err := apiServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			zlog.Fatal("API server failed", zap.Error(err))
		}
	}()

	// Metrics Server - 只處理 Prometheus metrics 請求
	go func() {
		defer wg.Done()
		zlog.Info("metrics server listening", zap.String("addr", ":"+cfg.App.MetricsPort), zap.String("path", "/metrics"))
		if err := metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			zlog.Fatal("metrics server failed", zap.Error(err))
		}
	}()

	// 10. 阻塞等待系統信號
	sig := <-quit
	zlog.Info("received shutdown signal", zap.String("signal", sig.String()))
	zlog.Info("starting graceful shutdown")

	// 11. 創建帶超時的 context（給予 30 秒完成停機）
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	// 12. 並行關閉兩個服務器
	var shutdownWg sync.WaitGroup
	shutdownWg.Add(2)

	// 關閉 API Server（包含背景任務）
	go func() {
		defer shutdownWg.Done()
		zlog.Info("shutting down api server")

		// 🆕 步驟 1：先關閉 HTTP Server（停止接受新請求）
		if err := apiServer.Shutdown(shutdownCtx); err != nil {
			zlog.Error("http server shutdown failed", zap.Error(err))
		} else {
			zlog.Info("http server shut down")
		}

		// 🆕 步驟 2：再關閉 Server 的背景任務（Worker + WebSocket Hub）
		// 這會關閉：
		// - Background Worker（處理設備分析任務）
		// - WebSocket Hub（包括所有 WebSocket 連接）
		// - Redis 監聽器
		// - Redis 連接
		if err := srv.Shutdown(shutdownCtx); err != nil {
			zlog.Error("server background shutdown failed", zap.Error(err))
		}
	}()

	// 關閉 Metrics Server
	go func() {
		defer shutdownWg.Done()
		zlog.Info("shutting down metrics server")
		if err := metricsServer.Shutdown(shutdownCtx); err != nil {
			zlog.Error("metrics server shutdown failed", zap.Error(err))
		} else {
			zlog.Info("metrics server shut down")
		}
	}()

	// 等待所有服務器完成關閉
	shutdownWg.Wait()

	// 13. 關閉資料庫連線
	zlog.Info("closing database connection")
	if err := sqlDB.Close(); err != nil {
		zlog.Error("database close failed", zap.Error(err))
	} else {
		zlog.Info("database connection closed")
	}

	zlog.Info("shutdown complete")
}
