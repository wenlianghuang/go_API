package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"my-api/api" // 假設這是你放 Server 的地方
	"my-api/model"
	"my-api/store"

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

	// 3. 自動遷移 (Auto Migration) - GORM 神技
	// 這行程式碼會自動在資料庫建立 devices 和 telemetries 資料表
	// 甚至當你修改 struct 欄位時，它也會試著幫你修改表結構
	if err := db.AutoMigrate(&model.Device{}, &model.Telemetry{}, &model.User{}); err != nil {
		log.Fatalf("資料庫遷移失敗: %v", err)
	}

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

	// 9. 啟動兩個獨立的 server
	var wg sync.WaitGroup
	wg.Add(2)

	// API Server - 處理所有業務請求（包含 WebSocket）
	go func() {
		defer wg.Done()
		fmt.Printf("🚀 API Server running on :%s\n", apiPort)
		if err := http.ListenAndServe(":"+apiPort, mainHandler); err != nil {
			log.Fatalf("API Server failed: %v", err)
		}
	}()

	// Metrics Server - 只處理 Prometheus metrics 請求
	go func() {
		defer wg.Done()
		metricsMux := http.NewServeMux()
		metricsMux.Handle("/metrics", api.MetricsHandler())
		fmt.Printf("📊 Metrics Server running on :%s/metrics\n", metricsPort)
		if err := http.ListenAndServe(":"+metricsPort, metricsMux); err != nil {
			log.Fatalf("Metrics Server failed: %v", err)
		}
	}()

	// 等待兩個 server（實際上會一直運行）
	wg.Wait()
}
