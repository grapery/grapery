package telemetry

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/client_golang/prometheus/push"
	"github.com/prometheus/common/expfmt"
)

// PrometheusConfig holds Prometheus configuration
type PrometheusConfig struct {
	Enabled      bool              `yaml:"enabled"`
	Path         string            `yaml:"path"`
	PushGateway  string            `yaml:"push_gateway"`
	PushInterval int               `yaml:"push_interval"`
	JobName      string            `yaml:"job_name"`
	AccessKey    string            `yaml:"access_key"`
	SecretKey    string            `yaml:"secret_key"`
	Grouping     map[string]string `yaml:"grouping"`
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

	// Content metrics
	StoryboardCount         prometheus.Gauge
	StoryCount              prometheus.Gauge
	UserCount               prometheus.Gauge
	GroupMemberCount        *prometheus.HistogramVec
	StoryParticipantCount   *prometheus.HistogramVec
	CharacterMessageCount   *prometheus.CounterVec
	CharacterTokenConsumed  *prometheus.HistogramVec
	StoryboardSceneCount    *prometheus.HistogramVec
	StoryboardChildCount    *prometheus.HistogramVec
	StoryboardTokenConsumed *prometheus.HistogramVec
	UserTokenConsumed       *prometheus.HistogramVec

	// User activity metrics
	DailyActiveUsers   prometheus.Gauge
	WeeklyActiveUsers  prometheus.Gauge
	MonthlyActiveUsers prometheus.Gauge
	UserGrowthRate     *prometheus.GaugeVec // Year-over-year and month-over-month growth

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

		// Content metrics
		StoryboardCount: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "storyboard_count",
				Help: "Total number of storyboards",
			},
		),
		StoryCount: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "story_count",
				Help: "Total number of stories",
			},
		),
		UserCount: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "user_count",
				Help: "Total number of users",
			},
		),
		DailyActiveUsers: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "daily_active_users",
				Help: "Number of daily active users",
			},
		),
		WeeklyActiveUsers: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "weekly_active_users",
				Help: "Number of weekly active users",
			},
		),
		MonthlyActiveUsers: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "monthly_active_users",
				Help: "Number of monthly active users",
			},
		),
		UserGrowthRate: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "user_growth_rate",
				Help: "User growth rate (YoY or MoM)",
			},
			[]string{"type"}, // "yoy" or "mom"
		),
		GroupMemberCount: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "group_member_count",
				Help:    "Number of members in each group",
				Buckets: []float64{1, 5, 10, 20, 50, 100, 200, 500, 1000},
			},
			[]string{"group_id"},
		),
		StoryParticipantCount: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "story_participant_count",
				Help:    "Number of participants in each story",
				Buckets: []float64{1, 2, 5, 10, 20, 50, 100},
			},
			[]string{"story_id"},
		),
		CharacterMessageCount: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "character_message_count_total",
				Help: "Total number of messages sent by each character",
			},
			[]string{"character_id"},
		),
		CharacterTokenConsumed: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "character_token_consumed",
				Help:    "Token consumption by each character",
				Buckets: []float64{10, 50, 100, 500, 1000, 5000, 10000, 50000, 100000},
			},
			[]string{"character_id"},
		),
		StoryboardSceneCount: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "storyboard_scene_count",
				Help:    "Number of scenes in each storyboard",
				Buckets: []float64{1, 2, 3, 5, 8, 10, 15, 20, 30},
			},
			[]string{"storyboard_id"},
		),
		StoryboardChildCount: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "storyboard_child_count",
				Help:    "Number of child storyboards for each storyboard",
				Buckets: []float64{0, 1, 2, 5, 10, 20, 50, 100},
			},
			[]string{"storyboard_id"},
		),
		StoryboardTokenConsumed: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "storyboard_token_consumed",
				Help:    "Token consumption for creating each storyboard",
				Buckets: []float64{100, 500, 1000, 5000, 10000, 50000, 100000, 500000},
			},
			[]string{"storyboard_id"},
		),
		UserTokenConsumed: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "user_token_consumed",
				Help:    "Token consumption by each user",
				Buckets: []float64{100, 500, 1000, 5000, 10000, 50000, 100000, 500000, 1000000},
			},
			[]string{"user_id"},
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
		m.StoryboardCount,
		m.StoryCount,
		m.UserCount,
		m.DailyActiveUsers,
		m.WeeklyActiveUsers,
		m.MonthlyActiveUsers,
		m.UserGrowthRate,
		m.GroupMemberCount,
		m.StoryParticipantCount,
		m.CharacterMessageCount,
		m.CharacterTokenConsumed,
		m.StoryboardSceneCount,
		m.StoryboardChildCount,
		m.StoryboardTokenConsumed,
		m.UserTokenConsumed,
	)

	// Note: Go collectors (NewGoCollector, NewProcessCollector) are not registered
	// to avoid pushing Go runtime metrics. Only application-specific metrics are pushed.

	// Setup push gateway if configured
	// Reference: https://prometheus.io/docs/instrumenting/pushing/
	// Example: pusher := push.New(url, "job").Collector(metric).BasicAuth("ak", "sk").Format(expfmt.FmtProtoDelim)
	if config.PushGateway != "" {
		// Create pusher with registry (contains all metrics)
		// Using Gatherer instead of Collector because we have multiple metrics
		pusher := push.New(config.PushGateway, config.JobName).
			Gatherer(registry).
			Client(http.DefaultClient).
			Format(expfmt.FmtProtoDelim)

		// Add BasicAuth if credentials are provided (required for Aliyun Log Service)
		// Format: .BasicAuth("ak", "sk")
		if config.AccessKey != "" && config.SecretKey != "" {
			pusher = pusher.BasicAuth(config.AccessKey, config.SecretKey)
		}

		// Add grouping labels if provided
		// Format: .Grouping("key1", "value1").Grouping("key2", "value2")
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
	// Start test data updater (for testing push gateway)
	// This simulates some metrics updates to verify push functionality
	m.wg.Add(1)
	go func() {
		defer m.wg.Done()

		ticker := time.NewTicker(10 * time.Second) // Update test data every 10 seconds
		defer ticker.Stop()

		// Initial update
		m.updateTestMetrics()

		for {
			select {
			case <-ticker.C:
				// Update test metrics to verify push gateway is working
				m.updateTestMetrics()
			case <-ctx.Done():
				// Final update before shutdown
				m.updateTestMetrics()
				return
			case <-m.stopChan:
				// Final update before shutdown
				m.updateTestMetrics()
				return
			}
		}
	}()

	if m.pusher == nil || m.config.PushInterval <= 0 {
		return
	}

	// Start push gateway pusher
	// Push metrics to PushGateway at configured interval
	m.wg.Add(1)
	go func() {
		defer m.wg.Done()

		ticker := time.NewTicker(time.Duration(m.config.PushInterval) * time.Second)
		defer ticker.Stop()

		// Push immediately on start
		if err := m.pusher.Push(); err != nil {
			// Log error but don't fail - errors will be handled by application logger
			fmt.Printf("Failed to push metrics to PushGateway on start: %v\n", err)
		} else {
			fmt.Printf("Successfully pushed metrics to PushGateway: %s\n", m.config.PushGateway)
		}

		for {
			select {
			case <-ticker.C:
				// Push metrics at configured interval
				if err := m.pusher.Push(); err != nil {
					fmt.Printf("Failed to push metrics to PushGateway: %v\n", err)
				} else {
					fmt.Printf("Successfully pushed metrics to PushGateway (interval: %ds)\n", m.config.PushInterval)
				}
			case <-ctx.Done():
				// Push final metrics before shutdown
				if err := m.pusher.Push(); err != nil {
					fmt.Printf("Failed to push final metrics to PushGateway: %v\n", err)
				} else {
					fmt.Printf("Successfully pushed final metrics to PushGateway\n")
				}
				return
			case <-m.stopChan:
				// Push final metrics before shutdown
				if err := m.pusher.Push(); err != nil {
					fmt.Printf("Failed to push final metrics to PushGateway: %v\n", err)
				} else {
					fmt.Printf("Successfully pushed final metrics to PushGateway\n")
				}
				return
			}
		}
	}()
}

// updateTestMetrics updates test metrics to verify push gateway functionality
func (m *Metrics) updateTestMetrics() {
	// Simulate some test data updates
	// These metrics will change over time, making it easy to verify push gateway is working

	// Update active requests (simulate some activity)
	m.ActiveRequests.Set(float64(time.Now().Unix() % 100))

	// Increment some test counters periodically
	m.UserRegistrations.WithLabelValues("test").Inc()
	m.UserLogins.WithLabelValues("test").Inc()
	m.StoryCreations.WithLabelValues("test").Inc()
	m.AIGenerations.WithLabelValues("test", "text").Inc()

	// Simulate cache activity
	m.CacheHits.WithLabelValues("test").Inc()
	m.CacheMisses.WithLabelValues("test").Add(0.5) // Can use Add for fractional increments

	// Simulate database query
	m.DatabaseQueryTime.WithLabelValues("select", "test_table").Observe(0.01)
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

// RecordStoryboardCount records the total number of storyboards
func (m *Metrics) RecordStoryboardCount(count float64) {
	m.StoryboardCount.Set(count)
}

// RecordStoryCount records the total number of stories
func (m *Metrics) RecordStoryCount(count float64) {
	m.StoryCount.Set(count)
}

// RecordGroupMemberCount records the number of members in a group
func (m *Metrics) RecordGroupMemberCount(groupID string, count float64) {
	m.GroupMemberCount.WithLabelValues(groupID).Observe(count)
}

// RecordStoryParticipantCount records the number of participants in a story
func (m *Metrics) RecordStoryParticipantCount(storyID string, count float64) {
	m.StoryParticipantCount.WithLabelValues(storyID).Observe(count)
}

// RecordCharacterMessage records a message sent by a character
func (m *Metrics) RecordCharacterMessage(characterID string) {
	m.CharacterMessageCount.WithLabelValues(characterID).Inc()
}

// RecordCharacterTokenConsumed records token consumption by a character
func (m *Metrics) RecordCharacterTokenConsumed(characterID string, tokens float64) {
	m.CharacterTokenConsumed.WithLabelValues(characterID).Observe(tokens)
}

// RecordStoryboardSceneCount records the number of scenes in a storyboard
func (m *Metrics) RecordStoryboardSceneCount(storyboardID string, count float64) {
	m.StoryboardSceneCount.WithLabelValues(storyboardID).Observe(count)
}

// RecordStoryboardChildCount records the number of child storyboards
func (m *Metrics) RecordStoryboardChildCount(storyboardID string, count float64) {
	m.StoryboardChildCount.WithLabelValues(storyboardID).Observe(count)
}

// RecordStoryboardTokenConsumed records token consumption for creating a storyboard
func (m *Metrics) RecordStoryboardTokenConsumed(storyboardID string, tokens float64) {
	m.StoryboardTokenConsumed.WithLabelValues(storyboardID).Observe(tokens)
}

// RecordUserTokenConsumed records token consumption by a user
func (m *Metrics) RecordUserTokenConsumed(userID string, tokens float64) {
	m.UserTokenConsumed.WithLabelValues(userID).Observe(tokens)
}

// RecordUserCount records the total number of users
func (m *Metrics) RecordUserCount(count float64) {
	m.UserCount.Set(count)
}

// RecordDailyActiveUsers records the number of daily active users
func (m *Metrics) RecordDailyActiveUsers(count float64) {
	m.DailyActiveUsers.Set(count)
}

// RecordWeeklyActiveUsers records the number of weekly active users
func (m *Metrics) RecordWeeklyActiveUsers(count float64) {
	m.WeeklyActiveUsers.Set(count)
}

// RecordMonthlyActiveUsers records the number of monthly active users
func (m *Metrics) RecordMonthlyActiveUsers(count float64) {
	m.MonthlyActiveUsers.Set(count)
}

// RecordUserGrowthRate records user growth rate (YoY or MoM)
func (m *Metrics) RecordUserGrowthRate(growthType string, rate float64) {
	m.UserGrowthRate.WithLabelValues(growthType).Set(rate)
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
