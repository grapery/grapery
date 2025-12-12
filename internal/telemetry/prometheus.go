package telemetry

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/client_golang/prometheus/push"
	"github.com/prometheus/common/expfmt"
)

// PrometheusConfig holds Prometheus configuration
type PrometheusConfig struct {
	Enabled      bool
	Path         string
	PushGateway  string
	PushInterval int
	JobName      string
	// BasicAuth for PushGateway (required for Aliyun Log Service)
	AccessKey string
	SecretKey string
	// Grouping labels for PushGateway (optional)
	Grouping map[string]string
}

// Metrics holds all Prometheus metrics
type Metrics struct {
	registry *prometheus.Registry

	// HTTP metrics
	HTTPRequestsTotal   *prometheus.CounterVec
	HTTPRequestDuration *prometheus.HistogramVec
	HTTPRequestSize     *prometheus.SummaryVec
	HTTPResponseSize    *prometheus.SummaryVec

	// Application metrics
	ActiveRequests    prometheus.Gauge
	ErrorsTotal       *prometheus.CounterVec
	DatabaseQueryTime *prometheus.HistogramVec
	CacheHits         *prometheus.CounterVec
	CacheMisses       *prometheus.CounterVec

	// Business metrics
	UserRegistrations *prometheus.CounterVec
	UserLogins        *prometheus.CounterVec
	StoryCreations    *prometheus.CounterVec
	AIGenerations     *prometheus.CounterVec

	// System metrics (collected by default collectors)
	config   PrometheusConfig
	pusher   *push.Pusher
	stopChan chan struct{}
	wg       sync.WaitGroup
}

// NewMetrics creates a new Metrics instance
func NewMetrics(config PrometheusConfig) *Metrics {
	registry := prometheus.NewRegistry()

	m := &Metrics{
		registry: registry,
		config:   config,
		stopChan: make(chan struct{}),

		// HTTP metrics
		HTTPRequestsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "http_requests_total",
				Help: "Total number of HTTP requests",
			},
			[]string{"method", "path", "status"},
		),
		HTTPRequestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "http_request_duration_seconds",
				Help:    "HTTP request duration in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"method", "path", "status"},
		),
		HTTPRequestSize: prometheus.NewSummaryVec(
			prometheus.SummaryOpts{
				Name:       "http_request_size_bytes",
				Help:       "HTTP request size in bytes",
				Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
			},
			[]string{"method", "path"},
		),
		HTTPResponseSize: prometheus.NewSummaryVec(
			prometheus.SummaryOpts{
				Name:       "http_response_size_bytes",
				Help:       "HTTP response size in bytes",
				Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
			},
			[]string{"method", "path"},
		),

		// Application metrics
		ActiveRequests: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "http_active_requests",
				Help: "Number of active HTTP requests",
			},
		),
		ErrorsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "errors_total",
				Help: "Total number of errors",
			},
			[]string{"type", "code"},
		),
		DatabaseQueryTime: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "database_query_duration_seconds",
				Help:    "Database query duration in seconds",
				Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1},
			},
			[]string{"operation", "table"},
		),
		CacheHits: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "cache_hits_total",
				Help: "Total number of cache hits",
			},
			[]string{"cache"},
		),
		CacheMisses: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "cache_misses_total",
				Help: "Total number of cache misses",
			},
			[]string{"cache"},
		),

		// Business metrics
		UserRegistrations: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "user_registrations_total",
				Help: "Total number of user registrations",
			},
			[]string{"source"},
		),
		UserLogins: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "user_logins_total",
				Help: "Total number of user logins",
			},
			[]string{"method"},
		),
		StoryCreations: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "story_creations_total",
				Help: "Total number of story creations",
			},
			[]string{"type"},
		),
		AIGenerations: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "ai_generations_total",
				Help: "Total number of AI generations",
			},
			[]string{"provider", "type"},
		),
	}

	// Register all metrics
	registry.MustRegister(
		m.HTTPRequestsTotal,
		m.HTTPRequestDuration,
		m.HTTPRequestSize,
		m.HTTPResponseSize,
		m.ActiveRequests,
		m.ErrorsTotal,
		m.DatabaseQueryTime,
		m.CacheHits,
		m.CacheMisses,
		m.UserRegistrations,
		m.UserLogins,
		m.StoryCreations,
		m.AIGenerations,
	)

	// Register default Go collectors
	registry.MustRegister(collectors.NewGoCollector())
	registry.MustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))

	// Setup push gateway if configured
	if config.PushGateway != "" {
		pusher := push.New(config.PushGateway, config.JobName).
			Gatherer(registry).
			Client(http.DefaultClient).
			Format(expfmt.FmtProtoDelim)

		// Add BasicAuth if credentials are provided (required for Aliyun Log Service)
		if config.AccessKey != "" && config.SecretKey != "" {
			pusher = pusher.BasicAuth(config.AccessKey, config.SecretKey)
		}

		// Add grouping labels if provided
		if config.Grouping != nil {
			for key, value := range config.Grouping {
				pusher = pusher.Grouping(key, value)
			}
		}

		m.pusher = pusher
	}

	return m
}

// Start starts the metrics push goroutine if push gateway is configured
func (m *Metrics) Start(ctx context.Context) {
	if m.pusher == nil || m.config.PushInterval <= 0 {
		return
	}

	m.wg.Add(1)
	go func() {
		defer m.wg.Done()

		ticker := time.NewTicker(time.Duration(m.config.PushInterval) * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if err := m.pusher.Push(); err != nil {
					// Log error but don't fail
					// The error will be logged by the application logger
				}
			case <-ctx.Done():
				// Push final metrics before shutdown
				_ = m.pusher.Push()
				return
			case <-m.stopChan:
				// Push final metrics before shutdown
				_ = m.pusher.Push()
				return
			}
		}
	}()
}

// Stop stops the metrics push goroutine
func (m *Metrics) Stop() {
	close(m.stopChan)
	m.wg.Wait()
}

// Handler returns an HTTP handler for the metrics endpoint
func (m *Metrics) Handler() http.Handler {
	return promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{
		EnableOpenMetrics: true,
	})
}

// Registry returns the Prometheus registry
func (m *Metrics) Registry() *prometheus.Registry {
	return m.registry
}

// RecordHTTPRequest records an HTTP request
func (m *Metrics) RecordHTTPRequest(method, path, status string, duration time.Duration, requestSize, responseSize float64) {
	m.HTTPRequestsTotal.WithLabelValues(method, path, status).Inc()
	m.HTTPRequestDuration.WithLabelValues(method, path, status).Observe(duration.Seconds())
	m.HTTPRequestSize.WithLabelValues(method, path).Observe(requestSize)
	m.HTTPResponseSize.WithLabelValues(method, path).Observe(responseSize)
}

// RecordError records an error
func (m *Metrics) RecordError(errorType, code string) {
	m.ErrorsTotal.WithLabelValues(errorType, code).Inc()
}

// RecordDatabaseQuery records a database query duration
func (m *Metrics) RecordDatabaseQuery(operation, table string, duration time.Duration) {
	m.DatabaseQueryTime.WithLabelValues(operation, table).Observe(duration.Seconds())
}

// RecordCacheHit records a cache hit
func (m *Metrics) RecordCacheHit(cache string) {
	m.CacheHits.WithLabelValues(cache).Inc()
}

// RecordCacheMiss records a cache miss
func (m *Metrics) RecordCacheMiss(cache string) {
	m.CacheMisses.WithLabelValues(cache).Inc()
}

// RecordUserRegistration records a user registration
func (m *Metrics) RecordUserRegistration(source string) {
	m.UserRegistrations.WithLabelValues(source).Inc()
}

// RecordUserLogin records a user login
func (m *Metrics) RecordUserLogin(method string) {
	m.UserLogins.WithLabelValues(method).Inc()
}

// RecordStoryCreation records a story creation
func (m *Metrics) RecordStoryCreation(storyType string) {
	m.StoryCreations.WithLabelValues(storyType).Inc()
}

// RecordAIGeneration records an AI generation
func (m *Metrics) RecordAIGeneration(provider, generationType string) {
	m.AIGenerations.WithLabelValues(provider, generationType).Inc()
}

// IncActiveRequests increments the active requests counter
func (m *Metrics) IncActiveRequests() {
	m.ActiveRequests.Inc()
}

// DecActiveRequests decrements the active requests counter
func (m *Metrics) DecActiveRequests() {
	m.ActiveRequests.Dec()
}

// DefaultMetrics is the global metrics instance
var (
	defaultMetrics     *Metrics
	defaultMetricsOnce sync.Once
)

// GetDefaultMetrics returns the default metrics instance
func GetDefaultMetrics() *Metrics {
	return defaultMetrics
}

// InitDefaultMetrics initializes the default metrics instance
func InitDefaultMetrics(config PrometheusConfig) *Metrics {
	defaultMetricsOnce.Do(func() {
		defaultMetrics = NewMetrics(config)
	})
	return defaultMetrics
}
