package api

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// JWTSecret 用於簽署 JWT 的密鑰
// ⚠️ 在生產環境中，這應該從環境變數或配置檔讀取，而不是硬編碼
const JWTSecret = "your-secret-key-change-this-in-production"

// Claims 定義 JWT 的 payload 結構
type Claims struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	jwt.RegisteredClaims
}

// GenerateJWT 生成一個新的 JWT token
// 參數：
//   - userID: 用戶 ID
//   - username: 用戶名
//   - email: 用戶郵箱
//   - expirationMinutes: token 過期時間（分鐘）
//
// 返回：JWT token 字符串
func GenerateJWT(userID, username, email string, expirationMinutes int) (string, error) {
	// 設定過期時間
	expirationTime := time.Now().Add(time.Duration(expirationMinutes) * time.Minute)

	// 創建 Claims
	claims := &Claims{
		UserID:   userID,
		Username: username,
		Email:    email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "my-api",                // 簽發者
			Subject:   userID,                  // 主題（通常是用戶 ID）
			ID:        generateTokenID(userID), // JWT ID（可選，用於撤銷）
		},
	}

	// 使用 HS256 算法創建 token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// 簽署 token
	tokenString, err := token.SignedString([]byte(JWTSecret))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

// ValidateJWT 驗證並解析 JWT token
// 參數：tokenString - JWT token 字符串
// 返回：Claims 和錯誤
func ValidateJWT(tokenString string) (*Claims, error) {
	// 解析 token
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// 驗證簽名算法是否正確
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(JWTSecret), nil
	})

	if err != nil {
		return nil, err
	}

	// 檢查 token 是否有效
	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}

// RefreshJWT 刷新 JWT token（生成新的 token）
// 參數：oldTokenString - 舊的 JWT token
// 返回：新的 JWT token
func RefreshJWT(oldTokenString string) (string, error) {
	// 首先驗證舊 token
	claims, err := ValidateJWT(oldTokenString)
	if err != nil {
		return "", err
	}

	// 生成新的 token（延長過期時間為 5 分鐘）
	return GenerateJWT(claims.UserID, claims.Username, claims.Email, 5)
}

// generateTokenID 生成一個唯一的 token ID
// 這可以用於 token 撤銷機制（存儲在 Redis 中）
func generateTokenID(userID string) string {
	return userID + "-" + time.Now().Format("20060102150405")
}
