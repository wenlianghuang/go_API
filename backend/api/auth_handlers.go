package api

import (
	"fmt"
	"net/http"

	"my-api/errors"
	"my-api/service"
)

// ==========================================
// 認證相關的 Handlers
// ==========================================

// HandleRegister 處理用戶註冊請求
// @Summary      用戶註冊
// @Description  註冊一個新的用戶帳號並返回 JWT token
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        user  body      RegisterRequest  true  "註冊資訊"
// @Success      201   {object}  AuthResponse
// @Failure      400   {object}  ErrorResponse
// @Failure      409   {object}  ErrorResponse  "用戶已存在"
// @Failure      500   {object}  ErrorResponse
// @Router       /auth/register [post]
func (s *Server) HandleRegister(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest

	// 使用新的驗證工具：自動解碼 + 驗證
	if err := ValidateAndDecode(r, &req); err != nil {
		HandleValidationError(w, err)
		return
	}

	// 調用 service 執行業務邏輯
	result, err := s.AuthService.Register(r.Context(), service.RegisterInput{
		Username: req.Username,
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		errors.HandleError(w, err, s.ErrorLogger)
		return
	}

	// 構建響應
	response := AuthResponse{
		Token:     result.Token,
		ExpiresAt: result.ExpiresAt,
		User:      ToUserResponse(result.User),
	}

	// 返回響應
	WriteJSON(w, http.StatusCreated, response)
}

// HandleLogin 處理用戶登入請求
// @Summary      用戶登入
// @Description  使用郵箱和密碼登入，返回 JWT token
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        credentials  body      LoginRequest  true  "登入憑證"
// @Success      200          {object}  AuthResponse
// @Failure      400          {object}  ErrorResponse
// @Failure      401          {object}  ErrorResponse  "郵箱或密碼錯誤"
// @Failure      500          {object}  ErrorResponse
// @Router       /auth/login [post]
func (s *Server) HandleLogin(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest

	// 使用新的驗證工具：自動解碼 + 驗證
	if err := ValidateAndDecode(r, &req); err != nil {
		HandleValidationError(w, err)
		return
	}

	// 調用 service 執行業務邏輯
	result, err := s.AuthService.Login(r.Context(), service.LoginInput{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		errors.HandleError(w, err, s.ErrorLogger)
		return
	}

	// 構建響應
	response := AuthResponse{
		Token:     result.Token,
		ExpiresAt: result.ExpiresAt,
		User:      ToUserResponse(result.User),
	}

	// 返回響應
	WriteJSON(w, http.StatusOK, response)
}

// HandleRefreshToken 處理刷新 token 請求
// @Summary      刷新 JWT token
// @Description  使用舊的 token 刷新獲取新的 token（延長過期時間）
// @Tags         auth
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  AuthResponse
// @Failure      401  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Router       /auth/refresh [post]
func (s *Server) HandleRefreshToken(w http.ResponseWriter, r *http.Request) {
	// 從 header 中獲取 token
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		errors.HandleError(w, errors.NewUnauthorizedError("Missing Authorization header"), s.ErrorLogger)
		return
	}

	// 解析 Bearer token
	var tokenString string
	if _, err := fmt.Sscanf(authHeader, "Bearer %s", &tokenString); err != nil {
		errors.HandleError(w, errors.NewUnauthorizedError("Invalid token format"), s.ErrorLogger)
		return
	}

	// 調用 service 執行業務邏輯
	result, err := s.AuthService.RefreshToken(r.Context(), service.RefreshTokenInput{
		TokenString: tokenString,
	})
	if err != nil {
		errors.HandleError(w, err, s.ErrorLogger)
		return
	}

	// 構建響應
	response := AuthResponse{
		Token:     result.Token,
		ExpiresAt: result.ExpiresAt,
		User:      ToUserResponse(result.User),
	}

	// 返回響應
	WriteJSON(w, http.StatusOK, response)
}
