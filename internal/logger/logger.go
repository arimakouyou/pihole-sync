package logger

import (
	"github.com/arimakouyou/pihole-sync/internal/config"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var Logger *zap.Logger

// InitLogger initializes the global logger based on configuration
func InitLogger(cfg *config.Config) error {
	var zapConfig zap.Config

	// Determine log level from config
	level := getLogLevel(cfg.Logging.Level, cfg.Logging.Debug)

	if cfg.Logging.Debug {
		// Development configuration for debug mode
		zapConfig = zap.NewDevelopmentConfig()
		zapConfig.Level = zap.NewAtomicLevelAt(zapcore.DebugLevel)
		zapConfig.Development = true
	} else {
		// Production configuration
		zapConfig = zap.NewProductionConfig()
		zapConfig.Level = zap.NewAtomicLevelAt(level)
		zapConfig.Development = false

		// Use console encoder for better readability
		zapConfig.Encoding = "console"
		zapConfig.EncoderConfig.TimeKey = "time"
		zapConfig.EncoderConfig.LevelKey = "level"
		zapConfig.EncoderConfig.MessageKey = "msg"
		zapConfig.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
		zapConfig.EncoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
		zapConfig.EncoderConfig.EncodeCaller = zapcore.ShortCallerEncoder
	}

	var err error
	Logger, err = zapConfig.Build()
	if err != nil {
		return err
	}

	return nil
}

// getLogLevel converts string level to zapcore.Level
func getLogLevel(levelStr string, debug bool) zapcore.Level {
	if debug {
		return zapcore.DebugLevel
	}

	switch levelStr {
	case "DEBUG":
		return zapcore.DebugLevel
	case "INFO":
		return zapcore.InfoLevel
	case "WARN", "WARNING":
		return zapcore.WarnLevel
	case "ERROR":
		return zapcore.ErrorLevel
	case "FATAL":
		return zapcore.FatalLevel
	default:
		return zapcore.InfoLevel
	}
}

// Cleanup flushes the logger
func Cleanup() {
	if Logger != nil {
		Logger.Sync()
	}
}
