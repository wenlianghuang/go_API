package logger

import (
	"fmt"
	"strings"

	"my-api/config"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func parseLevel(level string) (zapcore.Level, error) {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "", "info":
		return zapcore.InfoLevel, nil
	case "debug":
		return zapcore.DebugLevel, nil
	case "warn", "warning":
		return zapcore.WarnLevel, nil
	case "error":
		return zapcore.ErrorLevel, nil
	default:
		return zapcore.InfoLevel, fmt.Errorf("unknown LOG_LEVEL %q", level)
	}
}

// New returns a zap logger configured by environment.
//
// - development: console encoder (human-readable)
// - production: json encoder (machine-friendly)
func New(cfg *config.Config) (*zap.Logger, error) {
	env := strings.ToLower(strings.TrimSpace(cfg.App.Env))
	level, err := parseLevel(cfg.App.LogLevel)
	if err != nil {
		return nil, err
	}

	if env == "" {
		env = "development"
	}

	if env == "production" {
		zcfg := zap.NewProductionConfig()
		zcfg.Level = zap.NewAtomicLevelAt(level)
		// Use ISO8601 for easier reading in json logs
		zcfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
		return zcfg.Build()
	}

	zcfg := zap.NewDevelopmentConfig()
	zcfg.Level = zap.NewAtomicLevelAt(level)
	// Keep console logs compact & readable
	zcfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	return zcfg.Build()
}
