package main

import (
	"fmt"
	"net/http"

	"my-api/api"
	"my-api/store"
)

func main() {
	// 1. 初始化資料庫 (這裡用記憶體模擬)
	db := store.NewMemoryStore()

	// 2. 初始化 Server (注入資料庫依賴)
	srv := api.NewServer(db)

	// 3. 啟動服務
	fmt.Println("🚀 Server is running on port :8080")
	if err := http.ListenAndServe(":8080", srv.Router); err != nil {
		fmt.Println("Error starting server:", err)
	}
}
