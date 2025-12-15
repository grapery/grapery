# Telemetry System

This directory contains the telemetry system for the Grapery API, providing structured logging, metrics collection, and distributed tracing capabilities.

## Components

### 1. Logging (`logger.go`)

The logging system is built on top of [Zap](https://github.com/uber-go/zap) and provides:
- Structured JSON logging
- Multiple output destinations (console, Alibaba Cloud SLS)
- Configurable log levels
- Context-aware logging with correlation IDs

#### Usage

```go
// Initialize logger
logger, err := telemetry.NewLogger("info")
if err != nil {
    panic(err)
}
defer logger.Sync()

// Use logger
logger.Info("Application started",
    zap.String("version", "1.0.0"),
    zap.Int("port", 8080),
)
```

### 2. Metrics (`prometheus.go`)

The metrics system uses [Prometheus](https://prometheus.io/) to collect:
- HTTP request metrics (count, duration, size)
- Application metrics (active requests, errors)
- Database query metrics
- Cache hit/miss ratios
- Business metrics (user registrations, story creations, AI generations)

#### Usage

```go
// Initialize metrics
config := telemetry.PrometheusConfig{
    Enabled: true,
    Path:    "/metrics",
}
metrics := telemetry.NewMetrics(config)

// Record metrics
metrics.RecordHTTPRequest("GET", "/api/users", "200", time.Millisecond*100, 1024, 2048)
metrics.RecordDatabaseQuery("SELECT", "users", time.Millisecond*5)
metrics.RecordUserRegistration("web")
```

### 3. Distributed Tracing (`tracing.go`)

The tracing system uses [OpenTelemetry](https://opentelemetry.io/) to provide:
- Distributed trace propagation
- Span creation and management
- Integration with Jaeger and OTLP-compatible backends
- Automatic HTTP request tracing

#### Usage

```go
// Initialize tracer
config := telemetry.TracingConfig{
    Enabled:        true,
    ServiceName:    "grapery-api",
    ServiceVersion: "1.0.0",
    Environment:    "production",
    JaegerEndpoint: "http://jaeger:14268/api/traces",
}
tracer, err := telemetry.NewTracerProvider(config, logger)

// Create spans
ctx, span := tracer.Tracer("database").Start(ctx, "user.query")
defer span.End()

// Add attributes and events
span.SetAttributes(attribute.String("table", "users"))
span.AddEvent("query.executed", trace.WithAttributes(attribute.Int("rows", 10)))
```

### 4. Correlation (`correlation.go`)

The correlation system provides:
- Request correlation IDs for tracking requests across services
- Context propagation
- Service and user context enrichment

#### Usage

```go
// Add correlation ID to context
ctx := telemetry.ContextWithCorrelationID(ctx, "req-123456")

// Extract correlation ID
id := telemetry.CorrelationIDFromContext(ctx)

// Enrich logger with correlation ID
logger := telemetry.LoggerWithCorrelationID(logger, id)
```

### 5. Middleware

#### HTTP Middleware (`middleware.go`)

Standard HTTP middleware for metrics and logging:
- Request/response metrics collection
- Request logging with structured fields
- Request ID generation and propagation

#### Gin Middleware (`gin_middleware.go`)

Gin-specific middleware:
- Compatible with Gin framework
- Request/response metrics collection
- Request ID and correlation ID handling
- Distributed tracing integration

## Configuration

The telemetry system can be configured via:
1. YAML configuration file
2. Environment variables

### Environment Variables

#### Logging
- `GRAPERY_LOG_LEVEL`: Log level (debug, info, warn, error)

#### SLS (Alibaba Cloud Log Service)
- `TELEMETRY_SLS_ENABLED`: Enable SLS logging (true/false)
- `TELEMETRY_SLS_ENDPOINT`: SLS endpoint
- `TELEMETRY_SLS_ACCESS_KEY_ID`: SLS access key ID
- `TELEMETRY_SLS_ACCESS_KEY_SECRET`: SLS access key secret
- `TELEMETRY_SLS_PROJECT`: SLS project name
- `TELEMETRY_SLS_LOGSTORE`: SLS logstore name
- `TELEMETRY_SLS_TOPIC`: SLS topic
- `TELEMETRY_SLS_SOURCE`: SLS source

#### Prometheus Metrics
- `TELEMETRY_PROMETHEUS_ENABLED`: Enable Prometheus metrics (true/false)
- `TELEMETRY_PROMETHEUS_PATH`: Metrics endpoint path (default: /metrics)
- `TELEMETRY_PROMETHEUS_PUSH_GATEWAY`: Prometheus push gateway URL
- `TELEMETRY_PROMETHEUS_PUSH_INTERVAL`: Push interval in seconds
- `TELEMETRY_PROMETHEUS_JOB_NAME`: Job name for push gateway

#### Distributed Tracing
- `TELEMETRY_TRACING_ENABLED`: Enable distributed tracing (true/false)
- `TELEMETRY_TRACING_SERVICE_NAME`: Service name
- `TELEMETRY_TRACING_SERVICE_VERSION`: Service version
- `TELEMETRY_TRACING_ENVIRONMENT`: Environment (development, staging, production)
- `TELEMETRY_TRACING_JAEGER_ENDPOINT`: Jaeger collector endpoint
- `TELEMETRY_TRACING_OTLP_ENDPOINT`: OTLP endpoint
- `TELEMETRY_TRACING_SAMPLING_RATIO`: Sampling ratio (0.0 to 1.0)

## Integration with Main Application

The telemetry system is integrated into the main application via the `TelemetryManager`:

```go
// Initialize telemetry manager
telemetryConfig := telemetry.TelemetryManagerConfig{
    LogLevel: cfg.LogLevel,
    SLS: &telemetry.SLSConfig{...},
    Prometheus: &telemetry.PrometheusConfig{...},
    Tracing: &telemetry.TracingConfig{...},
}

telemetryManager, err := telemetry.NewTelemetryManager(telemetryConfig)
if err != nil {
    panic(err)
}
defer telemetryManager.Close()

// Use logger
logger := telemetryManager.Logger

// Add middleware
router.Use(telemetry.GinCorrelationMiddleware(logger))
router.Use(telemetry.GinRequestIDMiddleware(logger))
if telemetryManager.Tracer != nil {
    router.Use(telemetryManager.Tracer.GinTraceMiddleware())
}
if telemetryManager.Metrics != nil {
    router.Use(telemetry.GinHTTPMiddleware(logger, telemetryManager.Metrics))
}

// Add metrics endpoint
if telemetryManager.Metrics != nil && cfg.Telemetry.Prometheus.Enabled {
    router.GET(cfg.Telemetry.Prometheus.Path, gin.WrapH(telemetryManager.Metrics.Handler()))
}
```

## Best Practices

1. **Structured Logging**: Always use structured fields instead of formatted strings
2. **Correlation IDs**: Use correlation IDs to track requests across services
3. **Sampling**: Use appropriate sampling ratios for tracing in production
4. **Context Propagation**: Always propagate context through the call chain
5. **Error Handling**: Record errors with appropriate context and stack traces
6. **Performance**: Monitor the performance impact of telemetry in production

## Example Configuration

See `config.example.yaml` for a complete example configuration file.
