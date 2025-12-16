package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"my-api/api" // 假設這是你放 Server 的地方
	"my-api/model"
	"my-api/store"
)

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
	if err := db.AutoMigrate(&model.Device{}, &model.Telemetry{}); err != nil {
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
	gormStore := store.NewGormStore(db)

	// 6. 初始化 Server (注入 GormStore)
	// Server 根本不知道底層換成了 Postgres，這就是介面的威力
	srv := api.NewServer(gormStore)

	// 7. 啟動
	fmt.Println("🚀 IoT Server running on :8080")
	http.ListenAndServe(":8080", srv.Router)
}
