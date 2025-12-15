package telemetry

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.uber.org/zap"
)

// GinHTTPMiddleware returns a Gin middleware that records metrics and logs requests
func GinHTTPMiddleware(logger *zap.Logger, metrics *Metrics) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// Increment active requests counter
		if metrics != nil {
			metrics.IncActiveRequests()
		}

		// Process request
		c.Next()

		// Calculate duration
		duration := time.Since(start)

		// Get request size
		requestSize := float64(c.Request.ContentLength)
		if requestSize < 0 {
			requestSize = 0
		}

		// Get response size
		responseSize := float64(c.Writer.Size())

		// Get status code
		status := strconv.Itoa(c.Writer.Status())

		// Get route pattern if available
		route := c.Request.URL.Path
		if c.FullPath() != "" {
			route = c.FullPath()
		}

		// Record metrics
		if metrics != nil {
			metrics.RecordHTTPRequest(
				c.Request.Method,
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
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.String("route", route),
			zap.Int("status", c.Writer.Status()),
			zap.Duration("duration", duration),
			zap.Float64("request_size", requestSize),
			zap.Float64("response_size", responseSize),
			zap.String("user_agent", c.Request.UserAgent()),
			zap.String("remote_addr", c.Request.RemoteAddr),
		)
	}
}

// GinRequestIDMiddleware adds a unique request ID to the context and logs
func GinRequestIDMiddleware(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = c.Request.Header.Get("X-Request-ID")
		}
		
		// Add request ID to logger
		log := logger.With(zap.String("request_id", requestID))
		
		// Store logger in context
		c.Set("logger", log)
		
		// Continue
		c.Next()
	}
}

// GinTraceMiddleware creates a Gin middleware for tracing
func (tp *TracerProvider) GinTraceMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !tp.config.Enabled {
			c.Next()
			return
		}

		// Extract context from incoming headers
		ctx := otel.GetTextMapPropagator().Extract(c.Request.Context(), propagation.HeaderCarrier(c.Request.Header))
		
		// Get tracer
		tracer := tp.Tracer("http-server")
		
		// Start span
		spanName := c.Request.Method + " " + c.FullPath()
		if c.FullPath() == "" {
			spanName = c.Request.Method + " " + c.Request.URL.Path
		}
		ctx, span := tracer.Start(ctx, spanName)
		defer span.End()
		
		// Add attributes
		span.SetAttributes(
			attribute.String("http.method", c.Request.Method),
			attribute.String("http.url", c.Request.URL.String()),
			attribute.String("http.host", c.Request.Host),
			attribute.String("http.scheme", c.Request.URL.Scheme),
			attribute.String("http.user_agent", c.Request.UserAgent()),
			attribute.String("http.remote_addr", c.Request.RemoteAddr),
		)
		
		// Update request context
		c.Request = c.Request.WithContext(ctx)
		
		// Process request
		c.Next()
		
		// Add status code attribute
		span.SetAttributes(
			attribute.Int("http.status_code", c.Writer.Status()),
		)
		
		// Set status based on status code
		if c.Writer.Status() >= 400 {
			span.SetStatus(codes.Error, http.StatusText(c.Writer.Status()))
		}
	}
}

// GinLoggerFromContext returns the logger from the Gin context
func GinLoggerFromContext(c *gin.Context) *zap.Logger {
	if logger, exists := c.Get("logger"); exists {
		if l, ok := logger.(*zap.Logger); ok {
			return l
		}
	}
	return nil
}

// GinCorrelationMiddleware adds correlation ID handling
func GinCorrelationMiddleware(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get correlation ID from header or generate a new one
		correlationID := c.GetHeader(CorrelationIDHeader)
		if correlationID == "" {
			correlationID = NewCorrelationID()
		}
		
		// Add to response header
		c.Header(CorrelationIDHeader, correlationID)
		
		// Add to context
		ctx := ContextWithCorrelationID(c.Request.Context(), correlationID)
		c.Request = c.Request.WithContext(ctx)
		
		// Add to logger
		log := logger.With(zap.String("correlation_id", correlationID))
		c.Set("logger", log)
		
		c.Next()
	}
}
