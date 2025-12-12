package telemetry

import (
	"bytes"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// PrometheusMiddleware returns a Gin middleware for Prometheus metrics collection
func PrometheusMiddleware(metrics *Metrics) gin.HandlerFunc {
	return func(c *gin.Context) {
		if metrics == nil {
			c.Next()
			return
		}

		// Skip metrics endpoint itself
		if c.Request.URL.Path == "/metrics" {
			c.Next()
			return
		}

		start := time.Now()
		requestSize := computeApproximateRequestSize(c.Request)

		metrics.IncActiveRequests()

		c.Next()

		metrics.DecActiveRequests()

		duration := time.Since(start)
		status := strconv.Itoa(c.Writer.Status())
		responseSize := float64(c.Writer.Size())
		if responseSize < 0 {
			responseSize = 0
		}

		// Use path template if available, otherwise use path
		path := c.FullPath()
		if path == "" {
			path = c.Request.URL.Path
		}

		metrics.RecordHTTPRequest(
			c.Request.Method,
			path,
			status,
			duration,
			float64(requestSize),
			responseSize,
		)

		// Record errors
		if len(c.Errors) > 0 {
			for _, err := range c.Errors {
				errType := "unknown"
				if err.IsType(gin.ErrorTypeBind) {
					errType = "bind"
				} else if err.IsType(gin.ErrorTypeRender) {
					errType = "render"
				} else if err.IsType(gin.ErrorTypePrivate) {
					errType = "private"
				} else if err.IsType(gin.ErrorTypePublic) {
					errType = "public"
				}
				metrics.RecordError(errType, status)
			}
		}
	}
}

// computeApproximateRequestSize computes the approximate size of the request
func computeApproximateRequestSize(r *http.Request) int {
	size := 0
	if r.URL != nil {
		size += len(r.URL.String())
	}
	size += len(r.Method)
	size += len(r.Proto)

	for name, values := range r.Header {
		size += len(name)
		for _, value := range values {
			size += len(value)
		}
	}
	size += len(r.Host)

	if r.ContentLength != -1 {
		size += int(r.ContentLength)
	}
	return size
}

// responseWriter wraps gin.ResponseWriter to capture response body
type responseWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w *responseWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

func (w *responseWriter) WriteString(s string) (int, error) {
	w.body.WriteString(s)
	return w.ResponseWriter.WriteString(s)
}

// getOrGenerateTraceID gets trace_id from request header or generates a new one
func getOrGenerateTraceID(c *gin.Context) string {
	// Try to get trace_id from header (common headers: X-Trace-Id, X-Request-Id, Trace-Id)
	traceID := c.GetHeader("X-Trace-Id")
	if traceID == "" {
		traceID = c.GetHeader("X-Request-Id")
	}
	if traceID == "" {
		traceID = c.GetHeader("Trace-Id")
	}
	if traceID == "" {
		// Generate a new trace_id
		traceID = uuid.New().String()
	}
	// Store trace_id in context for downstream use
	c.Set("trace_id", traceID)
	return traceID
}

// readRequestBody reads and returns the request body, restoring it for the handler
func readRequestBody(c *gin.Context) ([]byte, error) {
	if c.Request.Body == nil {
		return nil, nil
	}

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return nil, err
	}

	// Restore the request body so handlers can read it
	c.Request.Body = io.NopCloser(bytes.NewBuffer(body))
	return body, nil
}

// LoggerMiddleware returns a Gin middleware for structured logging
func LoggerMiddleware(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		// Get or generate trace_id
		traceID := getOrGenerateTraceID(c)

		// Read request body
		var requestBodyStr string
		if c.Request.Method == http.MethodPost || c.Request.Method == http.MethodPut || c.Request.Method == http.MethodPatch {
			body, err := readRequestBody(c)
			if err == nil {
				// Limit request body size for logging (max 10KB)
				if len(body) > 10240 {
					requestBodyStr = string(body[:10240]) + "... (truncated)"
				} else {
					requestBodyStr = string(body)
				}
			}
		}

		// Wrap response writer to capture response body
		responseBody := &bytes.Buffer{}
		writer := &responseWriter{
			ResponseWriter: c.Writer,
			body:           responseBody,
		}
		c.Writer = writer

		c.Next()

		// Don't log metrics endpoint
		if path == "/metrics" {
			return
		}

		latency := time.Since(start)
		clientIP := c.ClientIP()
		method := c.Request.Method
		statusCode := c.Writer.Status()
		errorMessage := c.Errors.ByType(gin.ErrorTypePrivate).String()

		if raw != "" {
			path = path + "?" + raw
		}

		// Get response body
		var responseBodyStr string
		if responseBody.Len() > 0 {
			// Limit response body size for logging (max 10KB)
			if responseBody.Len() > 10240 {
				responseBodyStr = responseBody.String()[:10240] + "... (truncated)"
			} else {
				responseBodyStr = responseBody.String()
			}
		}

		// Build log fields
		fields := []zap.Field{
			zap.String("trace_id", traceID),
			zap.String("client_ip", clientIP),
			zap.String("method", method),
			zap.String("path", path),
			zap.Int("status", statusCode),
			zap.Duration("latency", latency),
			zap.Int("body_size", c.Writer.Size()),
		}

		// Add request body if available
		if requestBodyStr != "" {
			fields = append(fields, zap.String("request_body", requestBodyStr))
		}

		// Add query parameters if available
		if raw != "" {
			fields = append(fields, zap.String("query_params", raw))
		}

		// Add response body if available
		if responseBodyStr != "" {
			fields = append(fields, zap.String("response_body", responseBodyStr))
		}

		// Add error if any
		if errorMessage != "" {
			fields = append(fields, zap.String("error", errorMessage))
		}

		logger.Info("HTTP request", fields...)
	}
}
