package genapi

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"
)

var (
	loggerMu     sync.RWMutex
	globalLogger *zap.Logger
)

// SetLogger installs a zap logger used by GenAPI.
// If nil is passed, logging is disabled.
func SetLogger(logger *zap.Logger) {
	loggerMu.Lock()
	defer loggerMu.Unlock()
	globalLogger = logger
}

// GetLogger returns the current global logger.
func GetLogger() *zap.Logger {
	loggerMu.RLock()
	defer loggerMu.RUnlock()
	return globalLogger
}

// logOperation logs the start of an operation.
func logOperation(ctx context.Context, action string, fields LogFields) {
	logger := GetLogger()
	if logger == nil {
		return
	}
	logger.Info(action, fields.toZapFields()...)
}

// logOperationComplete logs the completion of an operation.
func logOperationComplete(ctx context.Context, action string, fields LogFields) {
	logger := GetLogger()
	if logger == nil {
		return
	}
	zapFields := fields.toZapFields()
	if fields.Error != nil {
		logger.Error(action+" failed", zapFields...)
	} else {
		logger.Info(action+" completed", zapFields...)
	}
}

// logDebug logs a debug message.
func logDebug(ctx context.Context, msg string, args ...any) {
	logger := GetLogger()
	if logger == nil {
		return
	}
	logger.Debug(msg, argsToZapFields(args)...)
}

// logInfo logs an info message.
func logInfo(ctx context.Context, msg string, args ...any) {
	logger := GetLogger()
	if logger == nil {
		return
	}
	logger.Info(msg, argsToZapFields(args)...)
}

// logWarn logs a warning message.
func logWarn(ctx context.Context, msg string, args ...any) {
	logger := GetLogger()
	if logger == nil {
		return
	}
	logger.Warn(msg, argsToZapFields(args)...)
}

// logError logs an error message.
func logError(ctx context.Context, msg string, args ...any) {
	logger := GetLogger()
	if logger == nil {
		return
	}
	logger.Error(msg, argsToZapFields(args)...)
}

// LogFields contains common fields for structured logging.
type LogFields struct {
	Provider    string
	Operation   OperationType
	MediaType   MediaType
	TaskID      string
	Duration    time.Duration
	Error       error
	RequestID   string
	UserID      int64
	ExtraFields map[string]any
}

// toZapFields converts LogFields to zap fields.
func (f LogFields) toZapFields() []zap.Field {
	fields := make([]zap.Field, 0, 10)
	if f.Provider != "" {
		fields = append(fields, zap.String("provider", f.Provider))
	}
	if f.Operation != "" {
		fields = append(fields, zap.String("operation", string(f.Operation)))
	}
	if f.MediaType != "" {
		fields = append(fields, zap.String("media_type", string(f.MediaType)))
	}
	if f.TaskID != "" {
		fields = append(fields, zap.String("task_id", f.TaskID))
	}
	if f.Duration > 0 {
		fields = append(fields, zap.Int64("duration_ms", f.Duration.Milliseconds()))
	}
	if f.Error != nil {
		fields = append(fields, zap.Error(f.Error))
	}
	if f.RequestID != "" {
		fields = append(fields, zap.String("request_id", f.RequestID))
	}
	if f.UserID != 0 {
		fields = append(fields, zap.Int64("user_id", f.UserID))
	}
	for k, v := range f.ExtraFields {
		fields = append(fields, zap.Any(k, v))
	}
	return fields
}

// argsToZapFields converts key-value pairs to zap fields.
func argsToZapFields(args []any) []zap.Field {
	if len(args) == 0 {
		return nil
	}
	fields := make([]zap.Field, 0, len(args)/2)
	for i := 0; i < len(args)-1; i += 2 {
		key, ok := args[i].(string)
		if !ok {
			continue
		}
		fields = append(fields, zap.Any(key, args[i+1]))
	}
	return fields
}
