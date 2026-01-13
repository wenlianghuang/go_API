package api

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGenerateJWT tests JWT token generation with various inputs
func TestGenerateJWT(t *testing.T) {
	// Setup: use a test secret
	oldSecret := JWTSecret
	JWTSecret = "test-secret-key"
	defer func() { JWTSecret = oldSecret }()

	tests := []struct {
		name            string
		userID          string
		username        string
		email           string
		expirationHours int
		wantErr         bool
		checkToken      bool
	}{
		{
			name:            "Valid token generation",
			userID:          "user123",
			username:        "testuser",
			email:           "test@example.com",
			expirationHours: 24,
			wantErr:         false,
			checkToken:      true,
		},
		{
			name:            "Valid token with short expiration",
			userID:          "user456",
			username:        "shortuser",
			email:           "short@example.com",
			expirationHours: 1,
			wantErr:         false,
			checkToken:      true,
		},
		{
			name:            "Valid token with long expiration",
			userID:          "user789",
			username:        "longuser",
			email:           "long@example.com",
			expirationHours: 168, // 7 days
			wantErr:         false,
			checkToken:      true,
		},
		{
			name:            "Empty user fields",
			userID:          "",
			username:        "",
			email:           "",
			expirationHours: 24,
			wantErr:         false,
			checkToken:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := GenerateJWT(tt.userID, tt.username, tt.email, tt.expirationHours)

			// 使用 testify 断言
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)

			if tt.checkToken {
				assert.NotEmpty(t, token, "Token should not be empty")

				// Verify the token can be validated
				claims, err := ValidateJWT(token)
				require.NoError(t, err, "Should be able to validate generated token")
				require.NotNil(t, claims, "Claims should not be nil")

				// Verify claims match input - 更清晰的断言
				assert.Equal(t, tt.userID, claims.UserID, "UserID should match")
				assert.Equal(t, tt.username, claims.Username, "Username should match")
				assert.Equal(t, tt.email, claims.Email, "Email should match")
			}
		})
	}
}

// TestValidateJWT tests JWT token validation with various scenarios
func TestValidateJWT(t *testing.T) {
	// Setup: use a test secret
	oldSecret := JWTSecret
	JWTSecret = "test-secret-key"
	defer func() { JWTSecret = oldSecret }()

	// Generate valid tokens for testing - 使用 require 确保测试数据生成成功
	validToken, err := GenerateJWT("user123", "testuser", "test@example.com", 24)
	require.NoError(t, err)

	expiredToken, err := GenerateJWT("user456", "expired", "expired@example.com", -1)
	require.NoError(t, err)

	// Generate token with different secret
	JWTSecret = "different-secret"
	tokenWithDifferentSecret, err := GenerateJWT("user789", "diffuser", "diff@example.com", 24)
	require.NoError(t, err)
	JWTSecret = "test-secret-key" // Restore

	tests := []struct {
		name          string
		token         string
		wantErr       bool
		errorContains string
		expectedUser  string
	}{
		{
			name:         "Valid token",
			token:        validToken,
			wantErr:      false,
			expectedUser: "user123",
		},
		{
			name:          "Expired token",
			token:         expiredToken,
			wantErr:       true,
			errorContains: "token",
		},
		{
			name:          "Invalid signature",
			token:         tokenWithDifferentSecret,
			wantErr:       true,
			errorContains: "signature",
		},
		{
			name:          "Empty token",
			token:         "",
			wantErr:       true,
			errorContains: "token",
		},
		{
			name:          "Malformed token",
			token:         "invalid.token.string",
			wantErr:       true,
			errorContains: "token",
		},
		{
			name:          "Random string",
			token:         "this-is-not-a-jwt-token",
			wantErr:       true,
			errorContains: "token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims, err := ValidateJWT(tt.token)

			if tt.wantErr {
				assert.Error(t, err, "Should return error for invalid token")
				if tt.errorContains != "" {
					assert.Contains(t, strings.ToLower(err.Error()),
						strings.ToLower(tt.errorContains),
						"Error message should contain expected text")
				}
				return
			}

			// 成功案例的断言
			assert.NoError(t, err)
			assert.NotNil(t, claims, "Claims should not be nil for valid token")
			assert.Equal(t, tt.expectedUser, claims.UserID, "UserID should match")
		})
	}
}

// TestRefreshJWT tests JWT token refresh functionality
func TestRefreshJWT(t *testing.T) {
	// Setup: use a test secret
	oldSecret := JWTSecret
	JWTSecret = "test-secret-key"
	defer func() { JWTSecret = oldSecret }()

	tests := []struct {
		name          string
		setupToken    func() string
		wantErr       bool
		errorContains string
	}{
		{
			name: "Refresh valid token",
			setupToken: func() string {
				token, _ := GenerateJWT("user123", "testuser", "test@example.com", 24)
				return token
			},
			wantErr: false,
		},
		{
			name: "Refresh expired token",
			setupToken: func() string {
				token, _ := GenerateJWT("user456", "expired", "expired@example.com", -1)
				return token
			},
			wantErr:       true,
			errorContains: "token",
		},
		{
			name: "Refresh invalid token",
			setupToken: func() string {
				return "invalid.token.string"
			},
			wantErr:       true,
			errorContains: "token",
		},
		{
			name: "Refresh token with invalid signature",
			setupToken: func() string {
				oldSecret := JWTSecret
				JWTSecret = "different-secret"
				token, _ := GenerateJWT("user789", "diffuser", "diff@example.com", 24)
				JWTSecret = oldSecret
				return token
			},
			wantErr:       true,
			errorContains: "signature",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldToken := tt.setupToken()

			// Wait a moment to ensure IssuedAt is different
			if !tt.wantErr {
				time.Sleep(10 * time.Millisecond)
			}

			newToken, err := RefreshJWT(oldToken)

			if tt.wantErr {
				assert.Error(t, err, "Should return error for invalid token refresh")
				if tt.errorContains != "" {
					assert.Contains(t, strings.ToLower(err.Error()),
						strings.ToLower(tt.errorContains),
						"Error message should contain expected text")
				}
				return
			}

			// 成功案例的断言 - 更清晰易读
			assert.NoError(t, err)
			assert.NotEmpty(t, newToken, "Refreshed token should not be empty")
			assert.NotEqual(t, oldToken, newToken, "Refreshed token should be different from original")

			// Validate the new token
			newClaims, err := ValidateJWT(newToken)
			require.NoError(t, err, "Refreshed token should be valid")
			require.NotNil(t, newClaims)

			// Verify original claims are preserved
			oldClaims, _ := ValidateJWT(oldToken)
			assert.Equal(t, oldClaims.UserID, newClaims.UserID, "UserID should be preserved")
			assert.Equal(t, oldClaims.Username, newClaims.Username, "Username should be preserved")
			assert.Equal(t, oldClaims.Email, newClaims.Email, "Email should be preserved")
		})
	}
}

// TestJWTExpirationTime tests that tokens expire at the correct time
func TestJWTExpirationTime(t *testing.T) {
	oldSecret := JWTSecret
	JWTSecret = "test-secret-key"
	defer func() { JWTSecret = oldSecret }()

	tests := []struct {
		name            string
		expirationHours int
		checkAfter      time.Duration
		shouldBeValid   bool
	}{
		{
			name:            "Token valid within expiration",
			expirationHours: 1,
			checkAfter:      100 * time.Millisecond,
			shouldBeValid:   true,
		},
		{
			name:            "Token expired after negative hours",
			expirationHours: -1,
			checkAfter:      0,
			shouldBeValid:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := GenerateJWT("user123", "testuser", "test@example.com", tt.expirationHours)
			require.NoError(t, err)

			// Wait if needed
			if tt.checkAfter > 0 {
				time.Sleep(tt.checkAfter)
			}

			_, err = ValidateJWT(token)

			if tt.shouldBeValid {
				assert.NoError(t, err, "Token should still be valid")
			} else {
				assert.Error(t, err, "Token should be expired")
			}
		})
	}
}

// TestJWTClaimsContent tests that JWT claims contain correct information
func TestJWTClaimsContent(t *testing.T) {
	oldSecret := JWTSecret
	JWTSecret = "test-secret-key"
	defer func() { JWTSecret = oldSecret }()

	tests := []struct {
		name     string
		userID   string
		username string
		email    string
	}{
		{
			name:     "Standard user",
			userID:   "user123",
			username: "testuser",
			email:    "test@example.com",
		},
		{
			name:     "User with special characters",
			userID:   "user-456_test",
			username: "test.user+special",
			email:    "test+special@example.com",
		},
		{
			name:     "User with Chinese characters",
			userID:   "用戶123",
			username: "測試用戶",
			email:    "test@example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := GenerateJWT(tt.userID, tt.username, tt.email, 24)
			require.NoError(t, err)

			claims, err := ValidateJWT(token)
			require.NoError(t, err)
			require.NotNil(t, claims)

			// Verify all claims using assert - 清晰易读的断言
			assert.Equal(t, tt.userID, claims.UserID, "UserID should match")
			assert.Equal(t, tt.username, claims.Username, "Username should match")
			assert.Equal(t, tt.email, claims.Email, "Email should match")

			// Verify standard JWT claims
			assert.Equal(t, "my-api", claims.Issuer, "Issuer should be 'my-api'")
			assert.Equal(t, tt.userID, claims.Subject, "Subject should match UserID")
			assert.NotNil(t, claims.ExpiresAt, "ExpiresAt should not be nil")
			assert.NotNil(t, claims.IssuedAt, "IssuedAt should not be nil")
			assert.NotNil(t, claims.NotBefore, "NotBefore should not be nil")

			// Verify expiration is in the future
			assert.True(t, claims.ExpiresAt.After(time.Now()),
				"ExpiresAt should be in the future")
		})
	}
}

// BenchmarkGenerateJWT benchmarks JWT generation performance
func BenchmarkGenerateJWT(b *testing.B) {
	oldSecret := JWTSecret
	JWTSecret = "test-secret-key"
	defer func() { JWTSecret = oldSecret }()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = GenerateJWT("user123", "testuser", "test@example.com", 24)
	}
}

// BenchmarkValidateJWT benchmarks JWT validation performance
func BenchmarkValidateJWT(b *testing.B) {
	oldSecret := JWTSecret
	JWTSecret = "test-secret-key"
	defer func() { JWTSecret = oldSecret }()

	token, _ := GenerateJWT("user123", "testuser", "test@example.com", 24)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ValidateJWT(token)
	}
}
