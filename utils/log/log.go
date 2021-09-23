package log

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var logger *zap.Logger

func init() {
	logCfg := zap.NewProductionConfig()

	logCfg.EncoderConfig.TimeKey = "time"
	logCfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	logCfg.EncoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder

	logger, _ = logCfg.Build()
	fields := zap.Fields(zap.Any("env", os.Environ()))

	logger = logger.WithOptions(fields)
}

func Log() *zap.Logger {
	return logger
}
