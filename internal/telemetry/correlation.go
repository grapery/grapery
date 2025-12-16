package telemetry

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"go.uber.org/zap"
)

// CorrelationIDKey is the context key for correlation ID
type CorrelationIDKey struct{}

// CorrelationIDHeader is the HTTP header for correlation ID
const CorrelationIDHeader = "X-Correlation-ID"

// NewCorrelationID generates a new correlation ID
func NewCorrelationID() string {
	bytes := make([]byte, 8)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to timestamp-based ID if random generation fails
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(bytes)
}

// ContextWithCorrelationID adds a correlation ID to the context
func ContextWithCorrelationID(ctx context.Context, correlationID string) context.Context {
	return context.WithValue(ctx, CorrelationIDKey{}, correlationID)
}

// CorrelationIDFromContext extracts the correlation ID from the context
func CorrelationIDFromContext(ctx context.Context) string {
	if id, ok := ctx.Value(CorrelationIDKey{}).(string); ok {
		return id
	}
	return ""
}

// LoggerWithCorrelationID adds correlation ID to the logger
func LoggerWithCorrelationID(logger *zap.Logger, correlationID string) *zap.Logger {
	if correlationID != "" {
		return logger.With(zap.String("correlation_id", correlationID))
	}
	return logger
}

// LoggerFromContextWithCorrelation extracts the logger from context and adds correlation ID
func LoggerFromContextWithCorrelation(ctx context.Context, defaultLogger *zap.Logger) *zap.Logger {
	logger := defaultLogger

	// Try to get logger from context first
	if ctxLogger := LoggerFromContext(ctx); ctxLogger != nil {
		logger = ctxLogger
	}

	// Add correlation ID if available
	if correlationID := CorrelationIDFromContext(ctx); correlationID != "" {
		logger = logger.With(zap.String("correlation_id", correlationID))
	}

	return logger
}

// ServiceContext represents service context information for logging
type ServiceContext struct {
	ServiceName    string
	ServiceVersion string
	Environment    string
	InstanceID     string
}

// LoggerWithServiceContext adds service context to the logger
func LoggerWithServiceContext(logger *zap.Logger, ctx ServiceContext) *zap.Logger {
	fields := []zap.Field{}

	if ctx.ServiceName != "" {
		fields = append(fields, zap.String("service_name", ctx.ServiceName))
	}

	if ctx.ServiceVersion != "" {
		fields = append(fields, zap.String("service_version", ctx.ServiceVersion))
	}

	if ctx.Environment != "" {
		fields = append(fields, zap.String("environment", ctx.Environment))
	}

	if ctx.InstanceID != "" {
		fields = append(fields, zap.String("instance_id", ctx.InstanceID))
	}

	return logger.With(fields...)
}

// UserContext represents user context information for logging
type UserContext struct {
	UserID   string
	Username string
	Role     string
}

// LoggerWithUserContext adds user context to the logger
func LoggerWithUserContext(logger *zap.Logger, ctx UserContext) *zap.Logger {
	fields := []zap.Field{}

	if ctx.UserID != "" {
		fields = append(fields, zap.String("user_id", ctx.UserID))
	}

	if ctx.Username != "" {
		fields = append(fields, zap.String("username", ctx.Username))
	}

	if ctx.Role != "" {
		fields = append(fields, zap.String("user_role", ctx.Role))
	}

	return logger.With(fields...)
}
