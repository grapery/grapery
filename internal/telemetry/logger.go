package telemetry

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// LoggerConfig holds logger configuration
type LoggerConfig struct {
	Level string
	SLS   *SLSConfig
}

// NewLogger returns a configured zap.Logger instance
func NewLogger(level string) (*zap.Logger, error) {
	return NewLoggerWithConfig(LoggerConfig{
		Level: level,
		SLS: &SLSConfig{
			Endpoint:        "cn-hangzhou.log.aliyuncs.com",
			AccessKeyID:     os.Getenv("ALIYUN_ACCESS_KEY_ID"),
			AccessKeySecret: os.Getenv("ALIYUN_ACCESS_KEY_SECRET"),
			Project:         "grapery-dev",
			Logstore:        "apiservice",
			Topic:           "api-backend",
			Source:          "backend",
		},
	})
}

// NewLoggerWithConfig returns a configured zap.Logger with optional SLS integration
func NewLoggerWithConfig(config LoggerConfig) (*zap.Logger, error) {
	lvl := parseLogLevel(config.Level)

	// Create encoder config
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "ts",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		FunctionKey:    zapcore.OmitKey,
		MessageKey:     "msg",
		StacktraceKey:  "stack",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	// Create console core (always enabled)
	consoleEncoder := zapcore.NewJSONEncoder(encoderConfig)
	consoleCore := zapcore.NewCore(
		consoleEncoder,
		zapcore.AddSync(os.Stdout),
		lvl,
	)

	cores := []zapcore.Core{consoleCore}
	// Add SLS core if configured
	if config.SLS != nil && config.SLS.Endpoint != "" {
		slsCore, err := NewSLSCore(*config.SLS, lvl)
		if err != nil {
			// Log warning but don't fail - continue without SLS
			consoleCore.Write(zapcore.Entry{
				Level:   zapcore.WarnLevel,
				Message: "Failed to initialize SLS logger: " + err.Error(),
			}, nil)
		} else {
			cores = append(cores, slsCore)
		}
	}

	// Combine cores
	core := zapcore.NewTee(cores...)

	// Build logger
	logger := zap.New(core,
		zap.AddCaller(),
		zap.AddStacktrace(zapcore.ErrorLevel),
	)

	return logger, nil
}

// parseLogLevel parses a string log level to zapcore.Level
func parseLogLevel(level string) zapcore.Level {
	switch strings.ToLower(level) {
	case "debug":
		return zapcore.DebugLevel
	case "info":
		return zapcore.InfoLevel
	case "warn", "warning":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	case "dpanic":
		return zapcore.DPanicLevel
	case "panic":
		return zapcore.PanicLevel
	case "fatal":
		return zapcore.FatalLevel
	default:
		return zapcore.InfoLevel
	}
}

// TelemetryManager manages all telemetry components
type TelemetryManager struct {
	Logger  *zap.Logger
	Metrics *Metrics
	SLSCore *SLSCore
	Tracer  *TracerProvider
	config  TelemetryManagerConfig
}

// TelemetryManagerConfig holds telemetry manager configuration
type TelemetryManagerConfig struct {
	LogLevel   string
	SLS        *SLSConfig
	Prometheus *PrometheusConfig
	Tracing    *TracingConfig
}

// NewTelemetryManager creates a new telemetry manager
func NewTelemetryManager(config TelemetryManagerConfig) (*TelemetryManager, error) {
	tm := &TelemetryManager{
		config: config,
	}

	// Initialize logger with SLS if enabled
	loggerConfig := LoggerConfig{
		Level: config.LogLevel,
	}
	if config.SLS != nil && config.SLS.Endpoint != "" {
		loggerConfig.SLS = config.SLS
	}

	logger, err := NewLoggerWithConfig(loggerConfig)
	if err != nil {
		return nil, err
	}
	tm.Logger = logger

	// Initialize Prometheus metrics if enabled
	if config.Prometheus != nil && config.Prometheus.Enabled {
		tm.Metrics = NewMetrics(*config.Prometheus)
	}

	// Initialize tracing if enabled
	if config.Tracing != nil {
		tracer, err := NewTracerProvider(*config.Tracing, logger)
		if err != nil {
			logger.Error("Failed to initialize tracer provider", zap.Error(err))
			return nil, err
		}
		tm.Tracer = tracer
	}

	return tm, nil
}

// Close closes all telemetry components
func (tm *TelemetryManager) Close() {
	if tm.Logger != nil {
		_ = tm.Logger.Sync()
	}
	if tm.SLSCore != nil {
		tm.SLSCore.Close()
	}
	if tm.Metrics != nil {
		tm.Metrics.Stop()
	}
	if tm.Tracer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = tm.Tracer.Shutdown(ctx)
	}
}
