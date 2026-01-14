package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"my-api/api"   // 假設這是你放 Server 的地方
	"my-api/store" // 資料存取層

	_ "my-api/docs" // swagger docs
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
	// 1. 設定資料庫連線資訊
	// 這裡使用環境變數，如果沒設定則使用預設值 (本機測試用)
	dsn := os.Getenv("DB_DSN")
	if dsn == "" {
		dsn = "host=localhost user=postgres password=password dbname=iot_db port=5432 sslmode=disable TimeZone=Asia/Taipei"
	}

	// 2. 連接資料庫
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("無法連接資料庫: %v", err)
	}
	fmt.Println("✅ 成功連接到 PostgreSQL")

	// 3. 資料庫遷移現在使用 golang-migrate
	// 請在啟動應用程式前執行：./scripts/migrate.sh up
	// 或在 Docker 環境中會自動執行遷移

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
		log.Fatalf("無法初始化 GormStore: %v", err)
	}

	// 6. 初始化 Server (注入 GormStore 和 Redis 位址)
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379" // 本地開發預設值
	}
	srv := api.NewServer(gormStore, redisAddr)

	// 7. 設定 port（從環境變數讀取，預設值為 8080 和 9090）
	apiPort := os.Getenv("API_PORT")
	if apiPort == "" {
		apiPort = "8080"
	}

	metricsPort := os.Getenv("METRICS_PORT")
	if metricsPort == "" {
		metricsPort = "9090"
	}

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

	// 9. 創建 HTTP Server 實例（使用 http.Server 物件）
	apiServer := &http.Server{
		Addr:         ":" + apiPort,
		Handler:      mainHandler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	metricsMux := http.NewServeMux()
	metricsMux.Handle("/metrics", api.MetricsHandler())
	metricsServer := &http.Server{
		Addr:         ":" + metricsPort,
		Handler:      metricsMux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// 10. 設定系統信號監聽
	// 創建一個 channel 來接收系統信號
	quit := make(chan os.Signal, 1)
	// 監聽 SIGINT (Ctrl+C) 和 SIGTERM (Docker/K8s stop)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// 11. 啟動兩個獨立的 server（使用 goroutine）
	var wg sync.WaitGroup
	wg.Add(2)

	// API Server - 處理所有業務請求（包含 WebSocket）
	go func() {
		defer wg.Done()
		fmt.Printf("🚀 API Server running on :%s\n", apiPort)
		if err := apiServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("API Server failed: %v", err)
		}
	}()

	// Metrics Server - 只處理 Prometheus metrics 請求
	go func() {
		defer wg.Done()
		fmt.Printf("📊 Metrics Server running on :%s/metrics\n", metricsPort)
		if err := metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Metrics Server failed: %v", err)
		}
	}()

	// 12. 阻塞等待系統信號
	sig := <-quit
	fmt.Printf("\n🛑 收到停機信號: %v\n", sig)
	fmt.Println("⏳ 開始優雅停機流程...")

	// 13. 創建帶超時的 context（給予 30 秒完成停機）
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	// 14. 並行關閉兩個服務器
	var shutdownWg sync.WaitGroup
	shutdownWg.Add(2)

	// 關閉 API Server（包含背景任務）
	go func() {
		defer shutdownWg.Done()
		fmt.Println("🔄 正在關閉 API Server...")

		// 🆕 步驟 1：先關閉 HTTP Server（停止接受新請求）
		if err := apiServer.Shutdown(shutdownCtx); err != nil {
			log.Printf("❌ HTTP Server 停機失敗: %v", err)
		} else {
			fmt.Println("✅ HTTP Server 已安全關閉")
		}

		// 🆕 步驟 2：再關閉 Server 的背景任務（Worker + WebSocket Hub）
		// 這會關閉：
		// - Background Worker（處理設備分析任務）
		// - WebSocket Hub（包括所有 WebSocket 連接）
		// - Redis 監聽器
		// - Redis 連接
		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Printf("❌ Server 背景任務停機失敗: %v", err)
		}
	}()

	// 關閉 Metrics Server
	go func() {
		defer shutdownWg.Done()
		fmt.Println("🔄 正在關閉 Metrics Server...")
		if err := metricsServer.Shutdown(shutdownCtx); err != nil {
			log.Printf("❌ Metrics Server 停機失敗: %v", err)
		} else {
			fmt.Println("✅ Metrics Server 已安全關閉")
		}
	}()

	// 等待所有服務器完成關閉
	shutdownWg.Wait()

	// 15. 關閉資料庫連線
	fmt.Println("🔄 正在關閉資料庫連線...")
	if err := sqlDB.Close(); err != nil {
		log.Printf("❌ 資料庫關閉失敗: %v", err)
	} else {
		fmt.Println("✅ 資料庫連線已關閉")
	}

	fmt.Println("👋 程式已完全停止")
}
