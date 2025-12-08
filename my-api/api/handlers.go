package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"my-api/store"

	"github.com/go-chi/chi/v5"
)

// HandleCreateUser 處理建立使用者的請求
func (s *Server) HandleCreateUser(w http.ResponseWriter, r *http.Request) {
	// 定義接收前端資料的結構 (DTO)
	type CreateUserRequest struct {
		Username string `json:"username"`
		Email    string `json:"email"`
	}

	var req CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	// 簡單驗證
	if req.Username == "" || req.Email == "" {
		WriteError(w, http.StatusBadRequest, "Username and Email are required")
		return
	}

	// 轉換成 Domain Model
	user := store.User{
		ID:        fmt.Sprintf("usr_%d", time.Now().UnixNano()), // 簡單生成 ID
		Username:  req.Username,
		Email:     req.Email,
		CreatedAt: time.Now(),
	}

	// 呼叫資料庫層
	if err := s.Store.Create(user); err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusCreated, user)
}

// HandleListUsers 取得所有使用者
func (s *Server) HandleListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := s.Store.List()
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to fetch users")
		return
	}
	WriteJSON(w, http.StatusOK, users)
}

// HandleGetUser 取得單一使用者
func (s *Server) HandleGetUser(w http.ResponseWriter, r *http.Request) {
	// 從 URL 參數中取得 id
	id := chi.URLParam(r, "id")

	user, err := s.Store.Get(id)
	if err != nil {
		WriteError(w, http.StatusNotFound, "User not found")
		return
	}

	WriteJSON(w, http.StatusOK, user)
}

// HandleMe 回傳當前登入者的資訊
func (s *Server) HandleMe(w http.ResponseWriter, r *http.Request) {
	// 使用 Helper 安全地取出 ID
	userID, ok := GetUserIDFromContext(r.Context())

	if !ok {
		// 理論上經過 AuthMiddleware 不會發生這種事，但為了防禦性程式設計還是要寫
		WriteError(w, http.StatusInternalServerError, "User ID not found in context")
		return
	}

	// 回傳簡單的訊息
	response := map[string]string{
		"message": "You are authenticated!",
		"user_id": userID,
	}
	WriteJSON(w, http.StatusOK, response)
}
