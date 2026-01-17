package api

import (
	"fmt"
	"net/http"
	"time"

	"my-api/model"

	"github.com/go-chi/chi/v5"
)

// ==========================================
// 用戶管理相關的 Handlers (保留舊的，用於兼容)
// ==========================================

// HandleCreateUser 處理建立使用者的請求（已廢棄，請使用 /auth/register）
// @Summary      創建新用戶（已廢棄）
// @Description  創建一個新的用戶帳號（已廢棄，請使用 /auth/register）
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        user  body      CreateUserRequest  true  "用戶資訊"
// @Success      201   {object}  UserResponse
// @Failure      400   {object}  ErrorResponse
// @Failure      500   {object}  ErrorResponse
// @Router       /users [post]
// @Deprecated
func (s *Server) HandleCreateUser(w http.ResponseWriter, r *http.Request) {
	var req CreateUserRequest

	// 使用新的驗證工具
	if err := ValidateAndDecode(r, &req); err != nil {
		HandleValidationError(w, err)
		return
	}

	// 轉換成 Domain Model
	user := model.User{
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
// @Summary      獲取所有用戶列表
// @Description  獲取系統中所有用戶的列表
// @Tags         users
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Success      200  {array}   UserResponse
// @Failure      401  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Router       /users [get]
func (s *Server) HandleListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := s.Store.List()
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to fetch users")
		return
	}
	WriteJSON(w, http.StatusOK, users)
}

// HandleGetUser 取得單一使用者
// @Summary      獲取單個用戶
// @Description  根據用戶 ID 獲取用戶詳細資訊
// @Tags         users
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string  true  "用戶 ID"
// @Success      200  {object}  UserResponse
// @Failure      401  {object}  ErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Router       /users/{id} [get]
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
// @Summary      獲取當前用戶資訊
// @Description  獲取當前已認證用戶的資訊
// @Tags         users
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  map[string]string
// @Failure      401  {object}  ErrorResponse
// @Router       /me [get]
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
