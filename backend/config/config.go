package config

import (
	"fmt"
	"os"
)

// Config 應用程式配置結構
type Config struct {
	Database DatabaseConfig
	Redis    RedisConfig
	App      AppConfig
}

// DatabaseConfig 資料庫配置
type DatabaseConfig struct {
	DSN string
}

// RedisConfig Redis 配置
type RedisConfig struct {
	Addr string
}

// AppConfig 應用程式配置
type AppConfig struct {
	APIPort     string
	MetricsPort string
	JWTSecret   string
}

// LoadConfig 載入配置並驗證（快速失敗）
func LoadConfig() (*Config, error) {
	cfg := &Config{}

	// 載入 Database 配置
	cfg.Database.DSN = os.Getenv("DB_DSN")
	if cfg.Database.DSN == "" {
		return nil, fmt.Errorf("DB_DSN is required but not set")
	}

	// 載入 Redis 配置
	cfg.Redis.Addr = os.Getenv("REDIS_ADDR")
	if cfg.Redis.Addr == "" {
		cfg.Redis.Addr = "localhost:6379"
	}

	// 載入 App 配置
	cfg.App.APIPort = os.Getenv("API_PORT")
	if cfg.App.APIPort == "" {
		cfg.App.APIPort = "8080"
	}

	cfg.App.MetricsPort = os.Getenv("METRICS_PORT")
	if cfg.App.MetricsPort == "" {
		cfg.App.MetricsPort = "9090"
	}

	cfg.App.JWTSecret = os.Getenv("JWT_SECRET")
	if cfg.App.JWTSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET is required but not set")
	}

	return cfg, nil
}
