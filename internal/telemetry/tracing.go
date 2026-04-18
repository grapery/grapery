package telemetry

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// TracingConfig holds tracing configuration
type TracingConfig struct {
	Enabled        bool              `yaml:"enabled"`
	ServiceName    string            `yaml:"service_name"`
	ServiceVersion string            `yaml:"service_version"`
	Environment    string            `yaml:"environment"`
	JaegerEndpoint string            `yaml:"jaeger_endpoint"`
	OTLPEndpoint   string            `yaml:"otlp_endpoint"`
	SamplingRatio  float64           `yaml:"sampling_ratio"`
	Headers        map[string]string `yaml:"headers"`
}

// TracerProvider wraps the OpenTelemetry TracerProvider
type TracerProvider struct {
	provider trace.TracerProvider
	logger   *zap.Logger
	config   TracingConfig
}

// NewTracerProvider initializes a new tracer provider
func NewTracerProvider(config TracingConfig, logger *zap.Logger) (*TracerProvider, error) {
	if !config.Enabled {
		logger.Info("Tracing is disabled")
		return &TracerProvider{
			provider: trace.NewNoopTracerProvider(),
			logger:   logger,
			config:   config,
		}, nil
	}

	// Set default values
	if config.ServiceName == "" {
		config.ServiceName = "grapery-api"
	}
	if config.ServiceVersion == "" {
		config.ServiceVersion = "1.0.0"
	}
	if config.Environment == "" {
		config.Environment = "development"
	}
	if config.SamplingRatio == 0 {
		config.SamplingRatio = 1.0 // Sample all traces in development
	}

	// Create resource with service information
	res, err := resource.New(context.Background(),
		resource.WithAttributes(
			attribute.String("service.name", config.ServiceName),
			attribute.String("service.version", config.ServiceVersion),
			attribute.String("service.environment", config.Environment),
			attribute.String("service.instance.id", getInstanceID()),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Create trace exporter
	var exporter sdktrace.SpanExporter
	if config.OTLPEndpoint != "" {
		// Use OTLP exporter
		client := otlptracehttp.NewClient(
			otlptracehttp.WithEndpoint(config.OTLPEndpoint),
		)
		exporter, err = otlptrace.New(context.Background(), client)
		if err != nil {
			return nil, fmt.Errorf("failed to create OTLP exporter: %w", err)
		}
		logger.Info("Using OTLP exporter", zap.String("endpoint", config.OTLPEndpoint))
	} else if config.JaegerEndpoint != "" {
		// Use Jaeger exporter
		exporter, err = jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint(config.JaegerEndpoint)))
		if err != nil {
			return nil, fmt.Errorf("failed to create Jaeger exporter: %w", err)
		}
		logger.Info("Using Jaeger exporter", zap.String("endpoint", config.JaegerEndpoint))
	} else {
		logger.Info("No tracing endpoint configured, using console exporter")
		// Fallback to console exporter for development
		exporter, err = jaeger.New(jaeger.WithAgentEndpoint(jaeger.WithAgentHost("localhost")))
		if err != nil {
			return nil, fmt.Errorf("failed to create Jaeger agent exporter: %w", err)
		}
	}

	// Create tracer provider with sampler
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.TraceIDRatioBased(config.SamplingRatio)),
	)

	// Register globally
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	logger.Info("Tracing initialized",
		zap.String("service", config.ServiceName),
		zap.String("version", config.ServiceVersion),
		zap.String("environment", config.Environment),
		zap.Float64("sampling_ratio", config.SamplingRatio),
	)

	return &TracerProvider{
		provider: tp,
		logger:   logger,
		config:   config,
	}, nil
}

// Tracer returns a tracer from the provider
func (tp *TracerProvider) Tracer(name string) trace.Tracer {
	return tp.provider.Tracer(name)
}

// Shutdown shuts down the tracer provider
func (tp *TracerProvider) Shutdown(ctx context.Context) error {
	if prov, ok := tp.provider.(*sdktrace.TracerProvider); ok {
		tp.logger.Info("Shutting down tracer provider")
		return prov.Shutdown(ctx)
	}
	return nil
}

// TraceMiddleware creates an HTTP middleware for tracing
func (tp *TracerProvider) TraceMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !tp.config.Enabled {
				next.ServeHTTP(w, r)
				return
			}

			// Extract context from incoming headers
			ctx := otel.GetTextMapPropagator().Extract(r.Context(), propagation.HeaderCarrier(r.Header))

			// Get tracer
			tracer := tp.Tracer("http-server")

			// Start span
			spanName := fmt.Sprintf("%s %s", r.Method, r.URL.Path)
			ctx, span := tracer.Start(ctx, spanName)
			defer span.End()

			// Add attributes
			span.SetAttributes(
				attribute.String("http.method", r.Method),
				attribute.String("http.url", r.URL.String()),
				attribute.String("http.host", r.Host),
				attribute.String("http.scheme", r.URL.Scheme),
				attribute.String("http.user_agent", r.UserAgent()),
				attribute.String("http.remote_addr", r.RemoteAddr),
			)

			// Wrap response writer to capture status code
			wrapped := &tracingResponseWriter{ResponseWriter: w, statusCode: 200}

			// Continue with traced context
			next.ServeHTTP(wrapped, r.WithContext(ctx))

			// Add status code attribute
			span.SetAttributes(
				attribute.Int("http.status_code", wrapped.statusCode),
			)

			// Set status based on status code
			if wrapped.statusCode >= 400 {
				span.SetStatus(codes.Error, fmt.Sprintf("HTTP %d", wrapped.statusCode))
			}
		})
	}
}

// tracingResponseWriter wraps http.ResponseWriter to capture status code
type tracingResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *tracingResponseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// getInstanceID returns a unique instance ID
func getInstanceID() string {
	hostname, _ := os.Hostname()
	if hostname == "" {
		hostname = "unknown"
	}
	return fmt.Sprintf("%s-%d", hostname, time.Now().Unix())
}

// StartSpan starts a new span with the given name
func StartSpan(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return otel.Tracer("grapery").Start(ctx, name, opts...)
}

// AddSpanEvent adds an event to the current span
func AddSpanEvent(ctx context.Context, name string, attributes ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	if span != nil {
		span.AddEvent(name, trace.WithAttributes(attributes...))
	}
}

// SetSpanAttribute sets an attribute on the current span
func SetSpanAttribute(ctx context.Context, key string, value interface{}) {
	span := trace.SpanFromContext(ctx)
	if span != nil {
		switch v := value.(type) {
		case string:
			span.SetAttributes(attribute.String(key, v))
		case int:
			span.SetAttributes(attribute.Int(key, v))
		case int64:
			span.SetAttributes(attribute.Int64(key, v))
		case float64:
			span.SetAttributes(attribute.Float64(key, v))
		case bool:
			span.SetAttributes(attribute.Bool(key, v))
		default:
			span.SetAttributes(attribute.String(key, fmt.Sprintf("%v", v)))
		}
	}
}

// SetSpanError marks the current span as having an error
func SetSpanError(ctx context.Context, err error) {
	span := trace.SpanFromContext(ctx)
	if span != nil && err != nil {
		span.SetStatus(codes.Error, err.Error())
		span.SetAttributes(attribute.String("error.message", err.Error()))
	}
}
