# Telemetry System Upgrade

This document summarizes the enhancements made to the Grapery API's logging and monitoring system.

## Overview

The telemetry system has been significantly enhanced with the following improvements:

1. **Enhanced Logging**
   - Structured logging with correlation IDs
   - Service and user context enrichment
   - Improved Alibaba Cloud SLS integration

2. **Advanced Metrics**
   - Comprehensive Prometheus metrics collection
   - HTTP request/response metrics
   - Database and cache performance metrics
   - Business metrics for key operations

3. **Distributed Tracing**
   - OpenTelemetry-based distributed tracing
   - Support for Jaeger and OTLP backends
   - Automatic HTTP request tracing
   - Custom span creation and management

4. **Middleware Integration**
   - Gin-compatible middleware for easy integration
   - Request ID and correlation ID handling
   - Automatic metrics collection
   - Distributed tracing propagation

## New Files

### Core Telemetry Components
- `internal/telemetry/correlation.go` - Correlation ID and context management
- `internal/telemetry/tracing.go` - OpenTelemetry distributed tracing
- `internal/telemetry/gin_middleware.go` - Gin-compatible middleware
- `internal/telemetry/middleware.go` - Standard HTTP middleware

### Documentation
- `internal/telemetry/README.md` - Comprehensive documentation
- `config.example.yaml` - Example configuration file
- `TELEMETRY_UPGRADE.md` - This summary document

## Modified Files

### Core System
- `internal/telemetry/logger.go` - Enhanced with tracing support
- `internal/telemetry/prometheus.go` - No changes (already comprehensive)
- `internal/telemetry/aliyun_sls.go` - No changes (already comprehensive)

### Configuration
- `internal/config/config.go` - Added tracing configuration options

### Application Integration
- `cmd/server/main.go` - Updated to use enhanced telemetry system

### Dependencies
- `go.mod` - Added OpenTelemetry dependencies

## Configuration Options

The telemetry system can be configured via:

1. **YAML Configuration File** - See `config.example.yaml`
2. **Environment Variables** - See documentation for full list

### New Configuration Options

#### Distributed Tracing
```yaml
telemetry:
  tracing:
    enabled: true
    service_name: grapery-api
    service_version: 1.0.0
    environment: production
    jaeger_endpoint: http://jaeger:14268/api/traces
    otlp_endpoint: http://otel-collector:4317
    sampling_ratio: 0.1
```

#### Enhanced Metrics
```yaml
telemetry:
  prometheus:
    enabled: true
    path: /metrics
    push_gateway: http://prometheus-pushgateway:9091
    push_interval: 15
    job_name: grapery
```

#### Enhanced Logging
```yaml
telemetry:
  sls:
    enabled: true
    endpoint: cn-shanghai.log.aliyuncs.com
    access_key_id: ${ALIYUN_ACCESS_KEY_ID}
    access_key_secret: ${ALIYUN_ACCESS_KEY_SECRET}
    project: grapery-prod
    logstore: apiservice
    topic: grapery
    source: api-server
```

## Usage Examples

### Adding Custom Metrics
```go
// Get metrics from telemetry manager
metrics := telemetryManager.Metrics

// Record custom business metrics
metrics.RecordUserRegistration("web")
metrics.RecordStoryCreation("interactive")
metrics.RecordAIGeneration("gemini", "image")
```

### Adding Custom Spans
```go
// Get tracer from telemetry manager
tracer := telemetryManager.Tracer

// Create custom span
ctx, span := tracer.Tracer("database").Start(ctx, "user.query")
defer span.End()

// Add attributes and events
span.SetAttributes(
    attribute.String("table", "users"),
    attribute.String("operation", "select"),
)
span.AddEvent("query.executed", 
    trace.WithAttributes(attribute.Int("rows", 10)),
)
```

### Adding Context to Logs
```go
// Get logger from context
logger := telemetry.LoggerFromContextWithCorrelation(ctx, defaultLogger)

// Add service context
serviceCtx := telemetry.ServiceContext{
    ServiceName:    "grapery-api",
    ServiceVersion: "1.0.0",
    Environment:    "production",
    InstanceID:     "api-server-1",
}
logger = telemetry.LoggerWithServiceContext(logger, serviceCtx)

// Add user context
userCtx := telemetry.UserContext{
    UserID:   "12345",
    Username: "john_doe",
    Role:     "user",
}
logger = telemetry.LoggerWithUserContext(logger, userCtx)

// Log with context
logger.Info("User action completed",
    zap.String("action", "create_story"),
    zap.String("story_id", "story-67890"),
)
```

## Deployment Considerations

1. **Sampling Ratio**: Use appropriate sampling ratios for tracing in production (e.g., 0.1 for 10% sampling)
2. **Resource Usage**: Monitor the resource usage of telemetry components
3. **Security**: Ensure sensitive data is not logged or traced
4. **Retention**: Configure appropriate retention policies for logs and traces
5. **Performance**: Monitor the performance impact of telemetry in production

## Migration Guide

To migrate from the previous telemetry system:

1. Update configuration to include new tracing options
2. Replace direct logger initialization with TelemetryManager
3. Add new middleware to the router
4. Update code to use correlation IDs and structured logging
5. Add custom metrics and spans as needed

## Benefits

1. **Improved Observability**: Better visibility into system behavior
2. **Faster Debugging**: Correlation IDs and distributed tracing
3. **Performance Insights**: Detailed metrics and performance data
4. **Business Intelligence**: Business metrics for key operations
5. **Production Readiness**: Enterprise-grade monitoring and logging
