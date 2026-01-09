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

	// HTTP Error metrics
	HTTPErrorTotal        *prometheus.CounterVec   // error_code, method, path: total count of errors by type
	HTTPErrorDuration     *prometheus.HistogramVec // error_code, method, path: error occurrence time distribution
	HTTPErrorDistribution *prometheus.SummaryVec   // error_code, method, path: error distribution with percentiles
	HTTPErrorRate         *prometheus.GaugeVec     // error_code, method, path: error rate percentage (0-100)
	HTTPErrorPercentiles  *prometheus.SummaryVec   // error_code: error percentiles (p50, p90, p95, p99)

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

	// Compliance metrics
	ComplianceChecksTotal      prometheus.Counter
	ComplianceCheckResults     *prometheus.CounterVec // status: "compliant" | "non_compliant"
	ComplianceViolationsByType *prometheus.CounterVec // violation_type: "porn" | "politics" | "abuse" | etc.
	ComplianceCheckDuration    *prometheus.HistogramVec

	// Payment metrics
	PaymentTotal          *prometheus.CounterVec   // provider: "apple" | "google" | "stripe", type: "subscription" | "one_time", status: "success" | "failed" | "pending"
	PaymentAmount         *prometheus.HistogramVec // provider, currency
	PaymentDuration       *prometheus.HistogramVec // provider, status
	PaymentRefundsTotal   *prometheus.CounterVec   // provider, reason
	PaymentRefundAmount   *prometheus.HistogramVec // provider, currency
	PaymentSubscriptions  *prometheus.GaugeVec     // provider, plan: active subscription count by plan
	PaymentVerifyTotal    *prometheus.CounterVec   // provider, status: receipt/purchase verification count
	PaymentVerifyDuration *prometheus.HistogramVec // provider: verification duration

	// OAuth/Third-party login metrics
	OAuthLoginTotal        *prometheus.CounterVec   // provider: "apple" | "google" | "facebook" | "twitter", status: "success" | "failed"
	OAuthLoginDuration     *prometheus.HistogramVec // provider
	OAuthLoginErrors       *prometheus.CounterVec   // provider, error_type: "invalid_token" | "expired" | "network" | "unknown"
	OAuthTokenRefreshTotal *prometheus.CounterVec   // provider, status
	OAuthLinkTotal         *prometheus.CounterVec   // provider, action: "link" | "unlink", status
	OAuthActiveProviders   *prometheus.GaugeVec     // provider: count of users using each provider

	// Notification metrics
	NotificationsSentTotal       *prometheus.CounterVec   // type: "push" | "email" | "sms" | "in_app", channel: "apns" | "fcm" | "smtp" | etc., status
	NotificationDeliveryDuration *prometheus.HistogramVec // type, channel
	NotificationErrors           *prometheus.CounterVec   // type, channel, error_type
	NotificationsQueued          prometheus.Gauge         // current queue size
	NotificationDeliveryRate     *prometheus.GaugeVec     // type, channel: success rate
	NotificationByCategory       *prometheus.CounterVec   // category: "marketing" | "transactional" | "system" | "social"
	NotificationRetries          *prometheus.CounterVec   // type, channel: retry attempts

	// Storyboard Generation Workflow metrics
	StoryboardContentGenerations    *prometheus.CounterVec   // step: "content", status: "pending" | "processing" | "completed" | "failed"
	StoryboardContentGenerationTime *prometheus.HistogramVec // step: "content"
	StoryboardSceneGenerations      *prometheus.CounterVec   // step: "scene_details", status
	StoryboardSceneGenerationTime   *prometheus.HistogramVec // step: "scene_details"
	StoryboardImageGenerations      *prometheus.CounterVec   // step: "image", status, scene_type: "transition" | "with_characters"
	StoryboardImageGenerationTime   *prometheus.HistogramVec // step: "image", scene_type
	StoryboardVideoGenerations      *prometheus.CounterVec   // step: "video", status, is_subdivided: "true" | "false"
	StoryboardVideoGenerationTime   *prometheus.HistogramVec // step: "video", is_subdivided

	// Image Generation Detailed metrics
	ImageGenerationWithCharacters    *prometheus.CounterVec   // status: "completed" | "failed", character_count: "0" | "1" | "2+"
	ImageGenerationWithStyle         *prometheus.CounterVec   // status, has_style: "true" | "false"
	ImageGenerationPromptDetailsUsed *prometheus.CounterVec   // has_prompt_details: "true" | "false"
	ImageGenerationCharacterRefs     *prometheus.HistogramVec // character_count: number of character references used
	ImageGenerationTokenConsumed     *prometheus.HistogramVec // step: "prompt" | "image", scene_type
	ImageGenerationErrors            *prometheus.CounterVec   // error_type: "ai_error" | "image_api_error" | "parsing_error" | "timeout" | "unknown"

	// Video Generation Detailed metrics
	VideoGenerationSubdivided    *prometheus.CounterVec   // is_subdivided: "true" | "false", status
	VideoGenerationSegmentCount  *prometheus.HistogramVec // segment_count: number of video segments
	VideoGenerationTokenConsumed *prometheus.HistogramVec // step: "prompt" | "video"
	VideoGenerationErrors        *prometheus.CounterVec   // error_type: "ai_error" | "video_api_error" | "timeout" | "unknown"

	// Character Poster Generation metrics
	CharacterPosterGenerations    *prometheus.CounterVec   // status, has_story_reference: "true" | "false"
	CharacterPosterGenerationTime *prometheus.HistogramVec // step: "concept" | "image"
	CharacterPosterConceptTime    *prometheus.HistogramVec // concept generation duration
	CharacterPosterImageTime      *prometheus.HistogramVec // image generation duration
	CharacterPosterTokenConsumed  *prometheus.HistogramVec // step: "concept" | "image"
	CharacterPosterErrors         *prometheus.CounterVec   // error_type: "concept_error" | "image_error" | "parsing_error" | "unknown"

	// Story Style Configuration metrics
	StoryStyleConfigUsage   *prometheus.CounterVec // style_id, usage_type: "image_generation" | "video_generation" | "poster_generation"
	StoryStyleConfigCount   prometheus.Gauge       // total number of style configurations
	StoryStyleConfigByStyle *prometheus.GaugeVec   // style: style name, count

	// AI Generation Quality metrics
	AIGenerationSuccessRate     *prometheus.GaugeVec   // type: "image" | "video" | "poster" | "content" | "scene", provider
	AIGenerationAverageTokens   *prometheus.GaugeVec   // type, provider
	AIGenerationAverageDuration *prometheus.GaugeVec   // type, provider
	AIGenerationRetries         *prometheus.CounterVec // type, provider, retry_count

	// Storyboard Workflow Completion metrics
	StoryboardWorkflowCompleted *prometheus.CounterVec   // workflow_status: "content_ready" | "images_ready" | "video_ready" | "published"
	StoryboardWorkflowDuration  *prometheus.HistogramVec // total workflow duration from start to completion
	StoryboardWorkflowAbandoned *prometheus.CounterVec   // abandoned_at_step: "content" | "images" | "video"

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

		// HTTP Error metrics
		HTTPErrorTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "http_error_total",
				Help: "Total number of HTTP errors by error code, method, and path",
			},
			[]string{"error_code", "method", "path"}, // error_code: "-1" (InvalidParams), "-2" (Unauthorized), "-3" (Forbidden), "-4" (NotFound), "-5" (InternalError), "-6" (DuplicateEntry), "-7" (RateLimitExceed), "-8" (TokenExpired), "-9" (InvalidToken), "0" (GenericError)
		),
		HTTPErrorDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "http_error_duration_seconds",
				Help:    "HTTP error occurrence time distribution in seconds",
				Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2, 5, 10},
			},
			[]string{"error_code", "method", "path"},
		),
		HTTPErrorDistribution: prometheus.NewSummaryVec(
			prometheus.SummaryOpts{
				Name:       "http_error_distribution_seconds",
				Help:       "HTTP error distribution with percentiles (p50, p90, p95, p99)",
				Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.95: 0.005, 0.99: 0.001},
			},
			[]string{"error_code", "method", "path"},
		),
		HTTPErrorRate: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "http_error_rate_percent",
				Help: "HTTP error rate percentage (0-100) by error code, method, and path",
			},
			[]string{"error_code", "method", "path"},
		),
		HTTPErrorPercentiles: prometheus.NewSummaryVec(
			prometheus.SummaryOpts{
				Name:       "http_error_percentiles_seconds",
				Help:       "HTTP error percentiles (p50, p90, p95, p99) aggregated by error code",
				Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.95: 0.005, 0.99: 0.001},
			},
			[]string{"error_code"},
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

		// Compliance metrics
		ComplianceChecksTotal: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "compliance_checks_total",
				Help: "Total number of compliance checks performed",
			},
		),
		ComplianceCheckResults: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "compliance_check_results_total",
				Help: "Total number of compliance check results by status",
			},
			[]string{"status"}, // "compliant" | "non_compliant"
		),
		ComplianceViolationsByType: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "compliance_violations_by_type_total",
				Help: "Total number of compliance violations by violation type",
			},
			[]string{"violation_type"}, // "porn" | "politics" | "abuse" | "terrorism" | "spam" | "flood" | "contraband" | "ad" | "review" | "unknown"
		),
		ComplianceCheckDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "compliance_check_duration_seconds",
				Help:    "Compliance check duration in seconds",
				Buckets: []float64{0.1, 0.25, 0.5, 1, 2, 5, 10, 30},
			},
			[]string{"status"}, // "compliant" | "non_compliant" | "error"
		),

		// Payment metrics
		PaymentTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "payment_total",
				Help: "Total number of payment transactions",
			},
			[]string{"provider", "type", "status"}, // provider: "apple" | "google" | "stripe", type: "subscription" | "one_time", status: "success" | "failed" | "pending"
		),
		PaymentAmount: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "payment_amount",
				Help:    "Payment amount distribution",
				Buckets: []float64{0.99, 2.99, 4.99, 9.99, 19.99, 49.99, 99.99, 199.99, 499.99},
			},
			[]string{"provider", "currency"},
		),
		PaymentDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "payment_duration_seconds",
				Help:    "Payment processing duration in seconds",
				Buckets: []float64{0.1, 0.5, 1, 2, 5, 10, 30, 60},
			},
			[]string{"provider", "status"},
		),
		PaymentRefundsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "payment_refunds_total",
				Help: "Total number of refund transactions",
			},
			[]string{"provider", "reason"},
		),
		PaymentRefundAmount: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "payment_refund_amount",
				Help:    "Refund amount distribution",
				Buckets: []float64{0.99, 2.99, 4.99, 9.99, 19.99, 49.99, 99.99, 199.99, 499.99},
			},
			[]string{"provider", "currency"},
		),
		PaymentSubscriptions: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "payment_subscriptions_active",
				Help: "Number of active subscriptions by provider and plan",
			},
			[]string{"provider", "plan"},
		),
		PaymentVerifyTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "payment_verify_total",
				Help: "Total number of payment verification requests",
			},
			[]string{"provider", "status"},
		),
		PaymentVerifyDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "payment_verify_duration_seconds",
				Help:    "Payment verification duration in seconds",
				Buckets: []float64{0.1, 0.25, 0.5, 1, 2, 5, 10},
			},
			[]string{"provider"},
		),

		// OAuth/Third-party login metrics
		OAuthLoginTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "oauth_login_total",
				Help: "Total number of OAuth login attempts",
			},
			[]string{"provider", "status"}, // provider: "apple" | "google" | "facebook" | "twitter", status: "success" | "failed"
		),
		OAuthLoginDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "oauth_login_duration_seconds",
				Help:    "OAuth login processing duration in seconds",
				Buckets: []float64{0.1, 0.25, 0.5, 1, 2, 5, 10},
			},
			[]string{"provider"},
		),
		OAuthLoginErrors: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "oauth_login_errors_total",
				Help: "Total number of OAuth login errors by type",
			},
			[]string{"provider", "error_type"}, // error_type: "invalid_token" | "expired" | "network" | "unknown"
		),
		OAuthTokenRefreshTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "oauth_token_refresh_total",
				Help: "Total number of OAuth token refresh attempts",
			},
			[]string{"provider", "status"},
		),
		OAuthLinkTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "oauth_link_total",
				Help: "Total number of OAuth account link/unlink operations",
			},
			[]string{"provider", "action", "status"}, // action: "link" | "unlink"
		),
		OAuthActiveProviders: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "oauth_active_providers",
				Help: "Number of users using each OAuth provider",
			},
			[]string{"provider"},
		),

		// Notification metrics
		NotificationsSentTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "notifications_sent_total",
				Help: "Total number of notifications sent",
			},
			[]string{"type", "channel", "status"}, // type: "push" | "email" | "sms" | "in_app", channel: "apns" | "fcm" | "smtp", status: "success" | "failed"
		),
		NotificationDeliveryDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "notification_delivery_duration_seconds",
				Help:    "Notification delivery duration in seconds",
				Buckets: []float64{0.01, 0.05, 0.1, 0.25, 0.5, 1, 2, 5},
			},
			[]string{"type", "channel"},
		),
		NotificationErrors: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "notification_errors_total",
				Help: "Total number of notification errors",
			},
			[]string{"type", "channel", "error_type"}, // error_type: "invalid_token" | "rate_limit" | "network" | "unknown"
		),
		NotificationsQueued: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "notifications_queued",
				Help: "Current number of notifications in queue",
			},
		),
		NotificationDeliveryRate: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "notification_delivery_rate",
				Help: "Notification delivery success rate (0-1)",
			},
			[]string{"type", "channel"},
		),
		NotificationByCategory: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "notifications_by_category_total",
				Help: "Total number of notifications by category",
			},
			[]string{"category"}, // "marketing" | "transactional" | "system" | "social"
		),
		NotificationRetries: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "notification_retries_total",
				Help: "Total number of notification retry attempts",
			},
			[]string{"type", "channel"},
		),

		// Storyboard Generation Workflow metrics
		StoryboardContentGenerations: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "storyboard_content_generations_total",
				Help: "Total number of storyboard content generations",
			},
			[]string{"status"}, // "pending" | "processing" | "completed" | "failed"
		),
		StoryboardContentGenerationTime: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "storyboard_content_generation_duration_seconds",
				Help:    "Storyboard content generation duration in seconds",
				Buckets: []float64{1, 2, 5, 10, 20, 30, 60, 120, 300},
			},
			[]string{"status"},
		),
		StoryboardSceneGenerations: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "storyboard_scene_generations_total",
				Help: "Total number of storyboard scene detail generations",
			},
			[]string{"status"},
		),
		StoryboardSceneGenerationTime: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "storyboard_scene_generation_duration_seconds",
				Help:    "Storyboard scene detail generation duration in seconds",
				Buckets: []float64{0.5, 1, 2, 5, 10, 20, 30, 60},
			},
			[]string{"status"},
		),
		StoryboardImageGenerations: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "storyboard_image_generations_total",
				Help: "Total number of storyboard image generations",
			},
			[]string{"status", "scene_type"}, // scene_type: "transition" | "with_characters"
		),
		StoryboardImageGenerationTime: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "storyboard_image_generation_duration_seconds",
				Help:    "Storyboard image generation duration in seconds",
				Buckets: []float64{5, 10, 15, 20, 30, 45, 60, 90, 120, 180, 300},
			},
			[]string{"scene_type"},
		),
		StoryboardVideoGenerations: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "storyboard_video_generations_total",
				Help: "Total number of storyboard video generations",
			},
			[]string{"status", "is_subdivided"}, // is_subdivided: "true" | "false"
		),
		StoryboardVideoGenerationTime: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "storyboard_video_generation_duration_seconds",
				Help:    "Storyboard video generation duration in seconds",
				Buckets: []float64{10, 30, 60, 120, 300, 600, 900, 1800, 3600},
			},
			[]string{"is_subdivided"},
		),

		// Image Generation Detailed metrics
		ImageGenerationWithCharacters: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "image_generation_with_characters_total",
				Help: "Total number of image generations with character references",
			},
			[]string{"status", "character_count"}, // character_count: "0" | "1" | "2+"
		),
		ImageGenerationWithStyle: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "image_generation_with_style_total",
				Help: "Total number of image generations with story style configuration",
			},
			[]string{"status", "has_style"}, // has_style: "true" | "false"
		),
		ImageGenerationPromptDetailsUsed: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "image_generation_prompt_details_used_total",
				Help: "Total number of image generations using structured prompt details",
			},
			[]string{"has_prompt_details"}, // has_prompt_details: "true" | "false"
		),
		ImageGenerationCharacterRefs: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "image_generation_character_refs_count",
				Help:    "Number of character references used in image generation",
				Buckets: []float64{0, 1, 2, 3, 4, 5, 10},
			},
			[]string{"scene_type"},
		),
		ImageGenerationTokenConsumed: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "image_generation_token_consumed",
				Help:    "Token consumption for image generation by step",
				Buckets: []float64{50, 100, 200, 500, 1000, 2000, 5000, 10000},
			},
			[]string{"step", "scene_type"}, // step: "prompt" | "image"
		),
		ImageGenerationErrors: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "image_generation_errors_total",
				Help: "Total number of image generation errors by type",
			},
			[]string{"error_type"}, // "ai_error" | "image_api_error" | "parsing_error" | "timeout" | "unknown"
		),

		// Video Generation Detailed metrics
		VideoGenerationSubdivided: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "video_generation_subdivided_total",
				Help: "Total number of video generations with subdivision",
			},
			[]string{"is_subdivided", "status"},
		),
		VideoGenerationSegmentCount: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "video_generation_segment_count",
				Help:    "Number of video segments when subdivision is applied",
				Buckets: []float64{1, 2, 3, 4, 5, 6, 8, 10, 15, 20},
			},
			[]string{"is_subdivided"},
		),
		VideoGenerationTokenConsumed: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "video_generation_token_consumed",
				Help:    "Token consumption for video generation by step",
				Buckets: []float64{100, 200, 500, 1000, 2000, 5000, 10000, 20000},
			},
			[]string{"step"}, // step: "prompt" | "video"
		),
		VideoGenerationErrors: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "video_generation_errors_total",
				Help: "Total number of video generation errors by type",
			},
			[]string{"error_type"}, // "ai_error" | "video_api_error" | "timeout" | "unknown"
		),

		// Character Poster Generation metrics
		CharacterPosterGenerations: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "character_poster_generations_total",
				Help: "Total number of character poster generations",
			},
			[]string{"status", "has_story_reference"}, // has_story_reference: "true" | "false"
		),
		CharacterPosterGenerationTime: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "character_poster_generation_duration_seconds",
				Help:    "Character poster generation duration in seconds",
				Buckets: []float64{5, 10, 15, 20, 30, 45, 60, 90, 120, 180, 300},
			},
			[]string{"step"}, // step: "concept" | "image"
		),
		CharacterPosterConceptTime: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "character_poster_concept_duration_seconds",
				Help:    "Character poster concept generation duration in seconds",
				Buckets: []float64{1, 2, 5, 10, 20, 30, 60},
			},
			[]string{"status"},
		),
		CharacterPosterImageTime: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "character_poster_image_duration_seconds",
				Help:    "Character poster image generation duration in seconds",
				Buckets: []float64{5, 10, 15, 20, 30, 45, 60, 90, 120},
			},
			[]string{"status"},
		),
		CharacterPosterTokenConsumed: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "character_poster_token_consumed",
				Help:    "Token consumption for character poster generation by step",
				Buckets: []float64{50, 100, 200, 500, 1000, 2000, 5000},
			},
			[]string{"step"}, // step: "concept" | "image"
		),
		CharacterPosterErrors: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "character_poster_errors_total",
				Help: "Total number of character poster generation errors by type",
			},
			[]string{"error_type"}, // "concept_error" | "image_error" | "parsing_error" | "unknown"
		),

		// Story Style Configuration metrics
		StoryStyleConfigUsage: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "story_style_config_usage_total",
				Help: "Total number of story style configuration usages",
			},
			[]string{"style_id", "usage_type"}, // usage_type: "image_generation" | "video_generation" | "poster_generation"
		),
		StoryStyleConfigCount: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "story_style_config_count",
				Help: "Total number of story style configurations",
			},
		),
		StoryStyleConfigByStyle: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "story_style_config_by_style",
				Help: "Number of style configurations by style name",
			},
			[]string{"style"},
		),

		// AI Generation Quality metrics
		AIGenerationSuccessRate: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ai_generation_success_rate",
				Help: "AI generation success rate (0-1)",
			},
			[]string{"type", "provider"}, // type: "image" | "video" | "poster" | "content" | "scene"
		),
		AIGenerationAverageTokens: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ai_generation_average_tokens",
				Help: "Average token consumption for AI generation",
			},
			[]string{"type", "provider"},
		),
		AIGenerationAverageDuration: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ai_generation_average_duration_seconds",
				Help: "Average duration for AI generation in seconds",
			},
			[]string{"type", "provider"},
		),
		AIGenerationRetries: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "ai_generation_retries_total",
				Help: "Total number of AI generation retries",
			},
			[]string{"type", "provider", "retry_count"}, // retry_count: "1" | "2" | "3+"
		),

		// Storyboard Workflow Completion metrics
		StoryboardWorkflowCompleted: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "storyboard_workflow_completed_total",
				Help: "Total number of completed storyboard workflows",
			},
			[]string{"workflow_status"}, // "content_ready" | "images_ready" | "video_ready" | "published"
		),
		StoryboardWorkflowDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "storyboard_workflow_duration_seconds",
				Help:    "Total workflow duration from start to completion",
				Buckets: []float64{60, 300, 600, 1800, 3600, 7200, 14400, 28800, 86400},
			},
			[]string{"workflow_status"},
		),
		StoryboardWorkflowAbandoned: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "storyboard_workflow_abandoned_total",
				Help: "Total number of abandoned storyboard workflows",
			},
			[]string{"abandoned_at_step"}, // "content" | "images" | "video"
		),
	}

	// Register all metrics
	registry.MustRegister(
		m.HTTPRequestsTotal,
		m.HTTPRequestDuration,
		m.HTTPRequestSize,
		m.HTTPResponseSize,
		m.HTTPErrorTotal,
		m.HTTPErrorDuration,
		m.HTTPErrorDistribution,
		m.HTTPErrorRate,
		m.HTTPErrorPercentiles,
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
		m.ComplianceChecksTotal,
		m.ComplianceCheckResults,
		m.ComplianceViolationsByType,
		m.ComplianceCheckDuration,
		// Payment metrics
		m.PaymentTotal,
		m.PaymentAmount,
		m.PaymentDuration,
		m.PaymentRefundsTotal,
		m.PaymentRefundAmount,
		m.PaymentSubscriptions,
		m.PaymentVerifyTotal,
		m.PaymentVerifyDuration,
		// OAuth metrics
		m.OAuthLoginTotal,
		m.OAuthLoginDuration,
		m.OAuthLoginErrors,
		m.OAuthTokenRefreshTotal,
		m.OAuthLinkTotal,
		m.OAuthActiveProviders,
		// Notification metrics
		m.NotificationsSentTotal,
		m.NotificationDeliveryDuration,
		m.NotificationErrors,
		m.NotificationsQueued,
		m.NotificationDeliveryRate,
		m.NotificationByCategory,
		m.NotificationRetries,
		// Storyboard Generation Workflow metrics
		m.StoryboardContentGenerations,
		m.StoryboardContentGenerationTime,
		m.StoryboardSceneGenerations,
		m.StoryboardSceneGenerationTime,
		m.StoryboardImageGenerations,
		m.StoryboardImageGenerationTime,
		m.StoryboardVideoGenerations,
		m.StoryboardVideoGenerationTime,
		// Image Generation Detailed metrics
		m.ImageGenerationWithCharacters,
		m.ImageGenerationWithStyle,
		m.ImageGenerationPromptDetailsUsed,
		m.ImageGenerationCharacterRefs,
		m.ImageGenerationTokenConsumed,
		m.ImageGenerationErrors,
		// Video Generation Detailed metrics
		m.VideoGenerationSubdivided,
		m.VideoGenerationSegmentCount,
		m.VideoGenerationTokenConsumed,
		m.VideoGenerationErrors,
		// Character Poster Generation metrics
		m.CharacterPosterGenerations,
		m.CharacterPosterGenerationTime,
		m.CharacterPosterConceptTime,
		m.CharacterPosterImageTime,
		m.CharacterPosterTokenConsumed,
		m.CharacterPosterErrors,
		// Story Style Configuration metrics
		m.StoryStyleConfigUsage,
		m.StoryStyleConfigCount,
		m.StoryStyleConfigByStyle,
		// AI Generation Quality metrics
		m.AIGenerationSuccessRate,
		m.AIGenerationAverageTokens,
		m.AIGenerationAverageDuration,
		m.AIGenerationRetries,
		// Storyboard Workflow Completion metrics
		m.StoryboardWorkflowCompleted,
		m.StoryboardWorkflowDuration,
		m.StoryboardWorkflowAbandoned,
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

// RecordHTTPError records an HTTP error
// errorCode: error code as string ("-1", "-2", "-3", "-4", "-5", "-6", "-7", "-8", "-9", "0")
// method: HTTP method ("GET", "POST", "PUT", "DELETE", etc.)
// path: HTTP request path
// duration: time when error occurred (can be request duration or error occurrence time)
func (m *Metrics) RecordHTTPError(errorCode, method, path string, duration time.Duration) {
	m.HTTPErrorTotal.WithLabelValues(errorCode, method, path).Inc()
	m.HTTPErrorDuration.WithLabelValues(errorCode, method, path).Observe(duration.Seconds())
	m.HTTPErrorDistribution.WithLabelValues(errorCode, method, path).Observe(duration.Seconds())
	m.HTTPErrorPercentiles.WithLabelValues(errorCode).Observe(duration.Seconds())
}

// RecordHTTPErrorRate records HTTP error rate percentage
// errorCode: error code as string
// method: HTTP method
// path: HTTP request path
// rate: error rate percentage (0-100)
func (m *Metrics) RecordHTTPErrorRate(errorCode, method, path string, rate float64) {
	m.HTTPErrorRate.WithLabelValues(errorCode, method, path).Set(rate)
}

// RecordHTTPErrorSimple records a simple HTTP error without duration (for counting only)
func (m *Metrics) RecordHTTPErrorSimple(errorCode, method, path string) {
	m.HTTPErrorTotal.WithLabelValues(errorCode, method, path).Inc()
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

// RecordComplianceCheck records a compliance check with duration and result
func (m *Metrics) RecordComplianceCheck(status string, duration time.Duration, violationTypes []string) {
	m.ComplianceChecksTotal.Inc()
	m.ComplianceCheckResults.WithLabelValues(status).Inc()
	m.ComplianceCheckDuration.WithLabelValues(status).Observe(duration.Seconds())

	// Record violations by type
	for _, violationType := range violationTypes {
		if violationType != "" {
			m.ComplianceViolationsByType.WithLabelValues(violationType).Inc()
		}
	}
}

// ============================================
// Payment Metrics Recording Methods
// ============================================

// RecordPayment records a payment transaction
// provider: "apple" | "google" | "stripe"
// paymentType: "subscription" | "one_time"
// status: "success" | "failed" | "pending"
func (m *Metrics) RecordPayment(provider, paymentType, status string, amount float64, currency string, duration time.Duration) {
	m.PaymentTotal.WithLabelValues(provider, paymentType, status).Inc()
	m.PaymentAmount.WithLabelValues(provider, currency).Observe(amount)
	m.PaymentDuration.WithLabelValues(provider, status).Observe(duration.Seconds())
}

// RecordPaymentSimple records a simple payment transaction without duration
func (m *Metrics) RecordPaymentSimple(provider, paymentType, status string) {
	m.PaymentTotal.WithLabelValues(provider, paymentType, status).Inc()
}

// RecordPaymentAmount records payment amount
func (m *Metrics) RecordPaymentAmount(provider, currency string, amount float64) {
	m.PaymentAmount.WithLabelValues(provider, currency).Observe(amount)
}

// RecordPaymentRefund records a refund transaction
func (m *Metrics) RecordPaymentRefund(provider, reason, currency string, amount float64) {
	m.PaymentRefundsTotal.WithLabelValues(provider, reason).Inc()
	m.PaymentRefundAmount.WithLabelValues(provider, currency).Observe(amount)
}

// RecordPaymentSubscription updates active subscription count
func (m *Metrics) RecordPaymentSubscription(provider, plan string, count float64) {
	m.PaymentSubscriptions.WithLabelValues(provider, plan).Set(count)
}

// IncPaymentSubscription increments subscription count
func (m *Metrics) IncPaymentSubscription(provider, plan string) {
	m.PaymentSubscriptions.WithLabelValues(provider, plan).Inc()
}

// DecPaymentSubscription decrements subscription count
func (m *Metrics) DecPaymentSubscription(provider, plan string) {
	m.PaymentSubscriptions.WithLabelValues(provider, plan).Dec()
}

// RecordPaymentVerify records a payment verification request
func (m *Metrics) RecordPaymentVerify(provider, status string, duration time.Duration) {
	m.PaymentVerifyTotal.WithLabelValues(provider, status).Inc()
	m.PaymentVerifyDuration.WithLabelValues(provider).Observe(duration.Seconds())
}

// ============================================
// OAuth/Third-party Login Metrics Recording Methods
// ============================================

// RecordOAuthLogin records an OAuth login attempt
// provider: "apple" | "google" | "facebook" | "twitter" | "wechat"
// status: "success" | "failed"
func (m *Metrics) RecordOAuthLogin(provider, status string, duration time.Duration) {
	m.OAuthLoginTotal.WithLabelValues(provider, status).Inc()
	m.OAuthLoginDuration.WithLabelValues(provider).Observe(duration.Seconds())
}

// RecordOAuthLoginSimple records a simple OAuth login attempt without duration
func (m *Metrics) RecordOAuthLoginSimple(provider, status string) {
	m.OAuthLoginTotal.WithLabelValues(provider, status).Inc()
}

// RecordOAuthLoginError records an OAuth login error
// errorType: "invalid_token" | "expired" | "network" | "user_cancelled" | "unknown"
func (m *Metrics) RecordOAuthLoginError(provider, errorType string) {
	m.OAuthLoginErrors.WithLabelValues(provider, errorType).Inc()
}

// RecordOAuthTokenRefresh records an OAuth token refresh attempt
func (m *Metrics) RecordOAuthTokenRefresh(provider, status string) {
	m.OAuthTokenRefreshTotal.WithLabelValues(provider, status).Inc()
}

// RecordOAuthLink records an OAuth account link/unlink operation
// action: "link" | "unlink"
// status: "success" | "failed"
func (m *Metrics) RecordOAuthLink(provider, action, status string) {
	m.OAuthLinkTotal.WithLabelValues(provider, action, status).Inc()
}

// RecordOAuthActiveProviders updates the count of users using a specific OAuth provider
func (m *Metrics) RecordOAuthActiveProviders(provider string, count float64) {
	m.OAuthActiveProviders.WithLabelValues(provider).Set(count)
}

// ============================================
// Notification Metrics Recording Methods
// ============================================

// RecordNotificationSent records a notification sent
// notificationType: "push" | "email" | "sms" | "in_app"
// channel: "apns" | "fcm" | "smtp" | "sms_provider"
// status: "success" | "failed"
func (m *Metrics) RecordNotificationSent(notificationType, channel, status string, duration time.Duration) {
	m.NotificationsSentTotal.WithLabelValues(notificationType, channel, status).Inc()
	m.NotificationDeliveryDuration.WithLabelValues(notificationType, channel).Observe(duration.Seconds())
}

// RecordNotificationSentSimple records a simple notification sent without duration
func (m *Metrics) RecordNotificationSentSimple(notificationType, channel, status string) {
	m.NotificationsSentTotal.WithLabelValues(notificationType, channel, status).Inc()
}

// RecordNotificationError records a notification error
// errorType: "invalid_token" | "rate_limit" | "network" | "payload_too_large" | "unknown"
func (m *Metrics) RecordNotificationError(notificationType, channel, errorType string) {
	m.NotificationErrors.WithLabelValues(notificationType, channel, errorType).Inc()
}

// SetNotificationsQueued updates the current queue size
func (m *Metrics) SetNotificationsQueued(count float64) {
	m.NotificationsQueued.Set(count)
}

// IncNotificationsQueued increments the queue size
func (m *Metrics) IncNotificationsQueued() {
	m.NotificationsQueued.Inc()
}

// DecNotificationsQueued decrements the queue size
func (m *Metrics) DecNotificationsQueued() {
	m.NotificationsQueued.Dec()
}

// RecordNotificationDeliveryRate updates the delivery success rate
func (m *Metrics) RecordNotificationDeliveryRate(notificationType, channel string, rate float64) {
	m.NotificationDeliveryRate.WithLabelValues(notificationType, channel).Set(rate)
}

// RecordNotificationByCategory records a notification by category
// category: "marketing" | "transactional" | "system" | "social"
func (m *Metrics) RecordNotificationByCategory(category string) {
	m.NotificationByCategory.WithLabelValues(category).Inc()
}

// RecordNotificationRetry records a notification retry attempt
func (m *Metrics) RecordNotificationRetry(notificationType, channel string) {
	m.NotificationRetries.WithLabelValues(notificationType, channel).Inc()
}

// ============================================
// Storyboard Generation Workflow Metrics Recording Methods
// ============================================

// RecordStoryboardContentGeneration records a storyboard content generation
func (m *Metrics) RecordStoryboardContentGeneration(status string, duration time.Duration) {
	m.StoryboardContentGenerations.WithLabelValues(status).Inc()
	m.StoryboardContentGenerationTime.WithLabelValues(status).Observe(duration.Seconds())
}

// RecordStoryboardSceneGeneration records a storyboard scene detail generation
func (m *Metrics) RecordStoryboardSceneGeneration(status string, duration time.Duration) {
	m.StoryboardSceneGenerations.WithLabelValues(status).Inc()
	m.StoryboardSceneGenerationTime.WithLabelValues(status).Observe(duration.Seconds())
}

// RecordStoryboardImageGeneration records a storyboard image generation
// sceneType: "transition" | "with_characters"
func (m *Metrics) RecordStoryboardImageGeneration(status, sceneType string, duration time.Duration) {
	m.StoryboardImageGenerations.WithLabelValues(status, sceneType).Inc()
	m.StoryboardImageGenerationTime.WithLabelValues(sceneType).Observe(duration.Seconds())
}

// RecordStoryboardVideoGeneration records a storyboard video generation
func (m *Metrics) RecordStoryboardVideoGeneration(status string, isSubdivided bool, duration time.Duration) {
	isSubdividedStr := "false"
	if isSubdivided {
		isSubdividedStr = "true"
	}
	m.StoryboardVideoGenerations.WithLabelValues(status, isSubdividedStr).Inc()
	m.StoryboardVideoGenerationTime.WithLabelValues(isSubdividedStr).Observe(duration.Seconds())
}

// ============================================
// Image Generation Detailed Metrics Recording Methods
// ============================================

// RecordImageGenerationWithCharacters records image generation with character references
// characterCount: number of characters (0, 1, 2+)
func (m *Metrics) RecordImageGenerationWithCharacters(status string, characterCount int) {
	charCountLabel := "0"
	if characterCount == 1 {
		charCountLabel = "1"
	} else if characterCount >= 2 {
		charCountLabel = "2+"
	}
	m.ImageGenerationWithCharacters.WithLabelValues(status, charCountLabel).Inc()
}

// RecordImageGenerationWithStyle records image generation with story style configuration
func (m *Metrics) RecordImageGenerationWithStyle(status string, hasStyle bool) {
	hasStyleStr := "false"
	if hasStyle {
		hasStyleStr = "true"
	}
	m.ImageGenerationWithStyle.WithLabelValues(status, hasStyleStr).Inc()
}

// RecordImageGenerationPromptDetails records usage of structured prompt details
func (m *Metrics) RecordImageGenerationPromptDetails(hasPromptDetails bool) {
	hasPromptDetailsStr := "false"
	if hasPromptDetails {
		hasPromptDetailsStr = "true"
	}
	m.ImageGenerationPromptDetailsUsed.WithLabelValues(hasPromptDetailsStr).Inc()
}

// RecordImageGenerationCharacterRefs records number of character references used
func (m *Metrics) RecordImageGenerationCharacterRefs(sceneType string, characterCount float64) {
	m.ImageGenerationCharacterRefs.WithLabelValues(sceneType).Observe(characterCount)
}

// RecordImageGenerationTokenConsumed records token consumption for image generation
// step: "prompt" | "image"
func (m *Metrics) RecordImageGenerationTokenConsumed(step, sceneType string, tokens float64) {
	m.ImageGenerationTokenConsumed.WithLabelValues(step, sceneType).Observe(tokens)
}

// RecordImageGenerationError records an image generation error
// errorType: "ai_error" | "image_api_error" | "parsing_error" | "timeout" | "unknown"
func (m *Metrics) RecordImageGenerationError(errorType string) {
	m.ImageGenerationErrors.WithLabelValues(errorType).Inc()
}

// ============================================
// Video Generation Detailed Metrics Recording Methods
// ============================================

// RecordVideoGenerationSubdivided records video generation with subdivision
func (m *Metrics) RecordVideoGenerationSubdivided(isSubdivided bool, status string) {
	isSubdividedStr := "false"
	if isSubdivided {
		isSubdividedStr = "true"
	}
	m.VideoGenerationSubdivided.WithLabelValues(isSubdividedStr, status).Inc()
}

// RecordVideoGenerationSegmentCount records number of video segments
func (m *Metrics) RecordVideoGenerationSegmentCount(isSubdivided bool, segmentCount float64) {
	isSubdividedStr := "false"
	if isSubdivided {
		isSubdividedStr = "true"
	}
	m.VideoGenerationSegmentCount.WithLabelValues(isSubdividedStr).Observe(segmentCount)
}

// RecordVideoGenerationTokenConsumed records token consumption for video generation
// step: "prompt" | "video"
func (m *Metrics) RecordVideoGenerationTokenConsumed(step string, tokens float64) {
	m.VideoGenerationTokenConsumed.WithLabelValues(step).Observe(tokens)
}

// RecordVideoGenerationError records a video generation error
// errorType: "ai_error" | "video_api_error" | "timeout" | "unknown"
func (m *Metrics) RecordVideoGenerationError(errorType string) {
	m.VideoGenerationErrors.WithLabelValues(errorType).Inc()
}

// ============================================
// Character Poster Generation Metrics Recording Methods
// ============================================

// RecordCharacterPosterGeneration records a character poster generation
func (m *Metrics) RecordCharacterPosterGeneration(status string, hasStoryReference bool) {
	hasStoryRefStr := "false"
	if hasStoryReference {
		hasStoryRefStr = "true"
	}
	m.CharacterPosterGenerations.WithLabelValues(status, hasStoryRefStr).Inc()
}

// RecordCharacterPosterGenerationTime records character poster generation duration
// step: "concept" | "image"
func (m *Metrics) RecordCharacterPosterGenerationTime(step string, duration time.Duration) {
	m.CharacterPosterGenerationTime.WithLabelValues(step).Observe(duration.Seconds())
}

// RecordCharacterPosterConceptTime records character poster concept generation duration
func (m *Metrics) RecordCharacterPosterConceptTime(status string, duration time.Duration) {
	m.CharacterPosterConceptTime.WithLabelValues(status).Observe(duration.Seconds())
}

// RecordCharacterPosterImageTime records character poster image generation duration
func (m *Metrics) RecordCharacterPosterImageTime(status string, duration time.Duration) {
	m.CharacterPosterImageTime.WithLabelValues(status).Observe(duration.Seconds())
}

// RecordCharacterPosterTokenConsumed records token consumption for character poster generation
// step: "concept" | "image"
func (m *Metrics) RecordCharacterPosterTokenConsumed(step string, tokens float64) {
	m.CharacterPosterTokenConsumed.WithLabelValues(step).Observe(tokens)
}

// RecordCharacterPosterError records a character poster generation error
// errorType: "concept_error" | "image_error" | "parsing_error" | "unknown"
func (m *Metrics) RecordCharacterPosterError(errorType string) {
	m.CharacterPosterErrors.WithLabelValues(errorType).Inc()
}

// ============================================
// Story Style Configuration Metrics Recording Methods
// ============================================

// RecordStoryStyleConfigUsage records usage of a story style configuration
// usageType: "image_generation" | "video_generation" | "poster_generation"
func (m *Metrics) RecordStoryStyleConfigUsage(styleID, usageType string) {
	m.StoryStyleConfigUsage.WithLabelValues(styleID, usageType).Inc()
}

// RecordStoryStyleConfigCount records the total number of style configurations
func (m *Metrics) RecordStoryStyleConfigCount(count float64) {
	m.StoryStyleConfigCount.Set(count)
}

// RecordStoryStyleConfigByStyle records style configuration count by style name
func (m *Metrics) RecordStoryStyleConfigByStyle(style string, count float64) {
	m.StoryStyleConfigByStyle.WithLabelValues(style).Set(count)
}

// ============================================
// AI Generation Quality Metrics Recording Methods
// ============================================

// RecordAIGenerationSuccessRate records AI generation success rate
// generationType: "image" | "video" | "poster" | "content" | "scene"
func (m *Metrics) RecordAIGenerationSuccessRate(generationType, provider string, rate float64) {
	m.AIGenerationSuccessRate.WithLabelValues(generationType, provider).Set(rate)
}

// RecordAIGenerationAverageTokens records average token consumption
func (m *Metrics) RecordAIGenerationAverageTokens(generationType, provider string, avgTokens float64) {
	m.AIGenerationAverageTokens.WithLabelValues(generationType, provider).Set(avgTokens)
}

// RecordAIGenerationAverageDuration records average generation duration
func (m *Metrics) RecordAIGenerationAverageDuration(generationType, provider string, avgDuration float64) {
	m.AIGenerationAverageDuration.WithLabelValues(generationType, provider).Set(avgDuration)
}

// RecordAIGenerationRetry records an AI generation retry
// retryCount: number of retries (1, 2, 3+)
func (m *Metrics) RecordAIGenerationRetry(generationType, provider string, retryCount int) {
	retryCountLabel := "1"
	if retryCount == 2 {
		retryCountLabel = "2"
	} else if retryCount >= 3 {
		retryCountLabel = "3+"
	}
	m.AIGenerationRetries.WithLabelValues(generationType, provider, retryCountLabel).Inc()
}

// ============================================
// Storyboard Workflow Completion Metrics Recording Methods
// ============================================

// RecordStoryboardWorkflowCompleted records a completed storyboard workflow
// workflowStatus: "content_ready" | "images_ready" | "video_ready" | "published"
func (m *Metrics) RecordStoryboardWorkflowCompleted(workflowStatus string, duration time.Duration) {
	m.StoryboardWorkflowCompleted.WithLabelValues(workflowStatus).Inc()
	m.StoryboardWorkflowDuration.WithLabelValues(workflowStatus).Observe(duration.Seconds())
}

// RecordStoryboardWorkflowAbandoned records an abandoned storyboard workflow
// abandonedAtStep: "content" | "images" | "video"
func (m *Metrics) RecordStoryboardWorkflowAbandoned(abandonedAtStep string) {
	m.StoryboardWorkflowAbandoned.WithLabelValues(abandonedAtStep).Inc()
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
