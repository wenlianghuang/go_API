package service

import (
	"context"
	"fmt"
	"my-api/errors"
	"my-api/model"
	"my-api/store"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// AuthService 處理認證相關的業務邏輯
type AuthService struct {
	Store        store.Storage
	JWTGenerator JWTGenerator // JWT 生成器接口（改為公開，以便 middleware 使用）
}

// JWTGenerator 定義 JWT 生成的接口，避免循環依賴
type JWTGenerator interface {
	GenerateJWT(userID, username, email string, expirationHours int) (string, error)
	RefreshJWT(oldTokenString string) (string, error)
	ValidateJWT(tokenString string) (JWTClaims, error)
}

// JWTClaims 定義 JWT Claims 的接口
type JWTClaims interface {
	GetUserID() string
	GetUsername() string
	GetEmail() string
}

// NewAuthService 創建一個新的 AuthService 實例
func NewAuthService(store store.Storage, jwtGenerator JWTGenerator) *AuthService {
	return &AuthService{
		Store:        store,
		JWTGenerator: jwtGenerator,
	}
}

// RegisterInput 註冊輸入參數
type RegisterInput struct {
	Username string
	Email    string
	Password string
}

// RegisterResult 註冊結果
type RegisterResult struct {
	User      model.User
	Token     string
	ExpiresAt time.Time
}

// Register 處理用戶註冊業務邏輯
// 返回 RegisterResult 和錯誤
func (s *AuthService) Register(ctx context.Context, input RegisterInput) (*RegisterResult, error) {
	// 檢查用戶是否已存在
	if _, err := s.Store.GetUserByEmail(ctx, input.Email); err == nil {
		return nil, errors.NewUserExistsError(input.Email)
	}

	// 對密碼進行哈希加密
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, errors.NewInternalError("hash password", err)
	}

	// 創建用戶
	user := model.User{
		ID:        fmt.Sprintf("usr_%d", time.Now().UnixNano()),
		Username:  input.Username,
		Email:     input.Email,
		Password:  string(hashedPassword),
		CreatedAt: time.Now(),
	}

	// 保存到數據庫
	if err := s.Store.Create(ctx, user); err != nil {
		return nil, errors.NewInternalError("create user", err)
	}

	// 生成 JWT token
	token, err := s.JWTGenerator.GenerateJWT(user.ID, user.Username, user.Email, 5) // 5分鐘過期
	if err != nil {
		return nil, errors.NewInternalError("generate token", err)
	}

	// 構建結果
	result := &RegisterResult{
		User:      user,
		Token:     token,
		ExpiresAt: time.Now().Add(30 * time.Minute),
	}

	return result, nil
}

// LoginInput 登入輸入參數
type LoginInput struct {
	Email    string
	Password string
}

// LoginResult 登入結果（與 RegisterResult 結構相同，可重用）
type LoginResult = RegisterResult

// Login 處理用戶登入業務邏輯
// 返回 LoginResult 和錯誤
func (s *AuthService) Login(ctx context.Context, input LoginInput) (*LoginResult, error) {
	// 根據 email 查找用戶
	user, err := s.Store.GetUserByEmail(ctx, input.Email)
	if err != nil {
		return nil, errors.NewInvalidCredentialsError()
	}

	// 驗證密碼
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(input.Password)); err != nil {
		return nil, errors.NewInvalidCredentialsError()
	}

	// 生成 JWT token
	token, err := s.JWTGenerator.GenerateJWT(user.ID, user.Username, user.Email, 5) // 5分鐘過期
	if err != nil {
		return nil, errors.NewInternalError("generate token", err)
	}

	// 構建結果
	result := &LoginResult{
		User:      user,
		Token:     token,
		ExpiresAt: time.Now().Add(30 * time.Minute),
	}

	return result, nil
}

// RefreshTokenInput 刷新 token 輸入參數
type RefreshTokenInput struct {
	TokenString string
}

// RefreshTokenResult 刷新 token 結果（與 RegisterResult 結構相同，可重用）
type RefreshTokenResult = RegisterResult

// RefreshToken 處理刷新 token 業務邏輯
// 返回 RefreshTokenResult 和錯誤
func (s *AuthService) RefreshToken(ctx context.Context, input RefreshTokenInput) (*RefreshTokenResult, error) {
	// 刷新 token
	newToken, err := s.JWTGenerator.RefreshJWT(input.TokenString)
	if err != nil {
		return nil, errors.NewTokenInvalidError(err.Error())
	}

	// 解析新 token 以獲取用戶信息
	claims, err := s.JWTGenerator.ValidateJWT(newToken)
	if err != nil {
		return nil, errors.NewInternalError("parse new token", err)
	}

	// 從數據庫獲取用戶最新信息
	user, err := s.Store.Get(ctx, claims.GetUserID())
	if err != nil {
		return nil, errors.NewInternalError("fetch user", err)
	}

	// 構建結果
	result := &RefreshTokenResult{
		User:      user,
		Token:     newToken,
		ExpiresAt: time.Now().Add(30 * time.Minute),
	}

	return result, nil
}
