package telemetry

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"go.uber.org/zap"
)

// HTTPMiddleware returns an HTTP middleware that records metrics and logs requests
func HTTPMiddleware(logger *zap.Logger, metrics *Metrics) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := &metricsResponseWriter{ResponseWriter: w, statusCode: 200}

			// Increment active requests counter
			if metrics != nil {
				metrics.IncActiveRequests()
			}

			// Process request
			next.ServeHTTP(ww, r)

			// Calculate duration
			duration := time.Since(start)

			// Get request size
			requestSize := float64(r.ContentLength)
			if requestSize < 0 {
				requestSize = 0
			}

			// Get response size
			responseSize := float64(ww.bytesWritten)

			// Get status code
			status := strconv.Itoa(ww.statusCode)

			// Get route pattern
			route := r.URL.Path

			// Record metrics
			if metrics != nil {
				metrics.RecordHTTPRequest(
					r.Method,
					route,
					status,
					duration,
					requestSize,
					responseSize,
				)
				metrics.DecActiveRequests()
			}

			// Log request
			logger.Info("HTTP request",
				zap.String("method", r.Method),
				zap.String("path", r.URL.Path),
				zap.String("route", route),
				zap.Int("status", ww.statusCode),
				zap.Duration("duration", duration),
				zap.Float64("request_size", requestSize),
				zap.Float64("response_size", responseSize),
				zap.String("user_agent", r.UserAgent()),
				zap.String("remote_addr", r.RemoteAddr),
			)
		})
	}
}

// metricsResponseWriter wraps http.ResponseWriter to capture status code and bytes written
type metricsResponseWriter struct {
	http.ResponseWriter
	statusCode   int
	bytesWritten int
}

func (rw *metricsResponseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *metricsResponseWriter) Write(b []byte) (int, error) {
	n, err := rw.ResponseWriter.Write(b)
	rw.bytesWritten += n
	return n, err
}

// RequestIDMiddleware adds a unique request ID to the context and logs
func RequestIDMiddleware(logger *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestID := r.Header.Get("X-Request-ID")
			if requestID == "" {
				requestID = generateRequestID()
			}
			
			// Add request ID to logger
			log := logger.With(zap.String("request_id", requestID))
			
			// Store logger in context
			ctx := context.WithValue(r.Context(), "logger", log)
			
			// Continue with new context
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// generateRequestID generates a simple request ID
func generateRequestID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

// LoggerFromContext returns the logger from the context
func LoggerFromContext(ctx context.Context) *zap.Logger {
	if logger, ok := ctx.Value("logger").(*zap.Logger); ok {
		return logger
	}
	return nil
}