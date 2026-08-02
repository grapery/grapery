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

// Metrics holds all Prometheus metrics, grouped by tier:
//   - System: HTTP traffic, errors, cache, active requests
//   - Business: users, content, AI workflows, payments, notifications, compliance
type Metrics struct {
	registry *prometheus.Registry

	// --- System tier ---
	HTTPRequestsTotal   *prometheus.CounterVec
	HTTPRequestDuration *prometheus.HistogramVec
	HTTPRequestSize     *prometheus.SummaryVec
	HTTPResponseSize    *prometheus.SummaryVec
	HTTPErrorTotal        *prometheus.CounterVec
	HTTPErrorDuration     *prometheus.HistogramVec
	ActiveRequests    prometheus.Gauge
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
	StoryParticipantCount   prometheus.Histogram
	StoryboardSceneCount    prometheus.Histogram
	StoryboardChildCount    prometheus.Histogram
	StoryboardTokenConsumed prometheus.Histogram
	UserTokenConsumed       prometheus.Histogram

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
	PaymentTotal          *prometheus.CounterVec
	PaymentRefundsTotal   *prometheus.CounterVec
	PaymentSubscriptions  *prometheus.GaugeVec
	PaymentVerifyTotal    *prometheus.CounterVec
	PaymentVerifyDuration *prometheus.HistogramVec

	// OAuth/Third-party login metrics
	OAuthLoginTotal    *prometheus.CounterVec
	OAuthLoginDuration *prometheus.HistogramVec
	OAuthLoginErrors   *prometheus.CounterVec

	// Notification metrics
	NotificationsSentTotal       *prometheus.CounterVec
	NotificationDeliveryDuration *prometheus.HistogramVec
	NotificationErrors           *prometheus.CounterVec
	NotificationByCategory       *prometheus.CounterVec

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

	// Story Style Configuration metrics
	StoryStyleConfigUsage   *prometheus.CounterVec // usage_type only (no style_id — avoids cardinality)
	StoryStyleConfigCount   prometheus.Gauge
	StoryStyleConfigByStyle *prometheus.GaugeVec

	// AI generation retries (success/failure tracked via workflow counters)
	AIGenerationRetries *prometheus.CounterVec

	// Storyboard Workflow Completion metrics
	StoryboardWorkflowCompleted *prometheus.CounterVec
	StoryboardWorkflowDuration  *prometheus.HistogramVec

	// Share funnel metrics (issue / open)
	ShareEventsTotal *prometheus.CounterVec

	config    PrometheusConfig
	pusher    *push.Pusher
	stopChan  chan struct{}
	wg        sync.WaitGroup
	startOnce sync.Once
	stopOnce  sync.Once
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
				Help:    "[system] HTTP error occurrence time distribution in seconds",
				Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2, 5, 10},
			},
			[]string{"error_code", "method", "path"},
		),

		ActiveRequests: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "http_active_requests",
				Help: "[system] Number of active HTTP requests",
			},
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
		StoryParticipantCount: prometheus.NewHistogram(
			prometheus.HistogramOpts{
				Name:    "story_participant_count",
				Help:    "[business] Number of participants per story (aggregated distribution)",
				Buckets: []float64{1, 2, 5, 10, 20, 50, 100},
			},
		),
		StoryboardSceneCount: prometheus.NewHistogram(
			prometheus.HistogramOpts{
				Name:    "storyboard_scene_count",
				Help:    "[business] Number of scenes per storyboard (aggregated distribution)",
				Buckets: []float64{1, 2, 3, 5, 8, 10, 15, 20, 30},
			},
		),
		StoryboardChildCount: prometheus.NewHistogram(
			prometheus.HistogramOpts{
				Name:    "storyboard_child_count",
				Help:    "[business] Number of child storyboards (aggregated distribution)",
				Buckets: []float64{0, 1, 2, 5, 10, 20, 50, 100},
			},
		),
		StoryboardTokenConsumed: prometheus.NewHistogram(
			prometheus.HistogramOpts{
				Name:    "storyboard_token_consumed",
				Help:    "[business] Token consumption per storyboard (aggregated distribution)",
				Buckets: []float64{100, 500, 1000, 5000, 10000, 50000, 100000, 500000},
			},
		),
		UserTokenConsumed: prometheus.NewHistogram(
			prometheus.HistogramOpts{
				Name:    "user_token_consumed",
				Help:    "[business] Token consumption per user action (aggregated distribution)",
				Buckets: []float64{100, 500, 1000, 5000, 10000, 50000, 100000, 500000, 1000000},
			},
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
				Help: "[business] Total number of payment transactions",
			},
			[]string{"provider", "type", "status"},
		),
		PaymentRefundsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "payment_refunds_total",
				Help: "[business] Total number of refund transactions",
			},
			[]string{"provider", "reason"},
		),
		PaymentSubscriptions: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "payment_subscriptions_active",
				Help: "[business] Active subscriptions by provider and plan",
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
				Help: "[business] Total number of OAuth login errors by type",
			},
			[]string{"provider", "error_type"},
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
				Help: "[business] Total number of notification errors",
			},
			[]string{"type", "channel", "error_type"},
		),
		NotificationByCategory: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "notifications_by_category_total",
				Help: "[business] Total number of notifications by category",
			},
			[]string{"category"},
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

		// Story Style Configuration metrics
		StoryStyleConfigUsage: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "story_style_config_usage_total",
				Help: "[business] Total number of story style configuration usages by type",
			},
			[]string{"usage_type"},
		),
		StoryStyleConfigCount: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "story_style_config_count",
				Help: "[business] Total number of story style configurations",
			},
		),
		StoryStyleConfigByStyle: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "story_style_config_by_style",
				Help: "[business] Number of style configurations by style name",
			},
			[]string{"style"},
		),

		AIGenerationRetries: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "ai_generation_retries_total",
				Help: "[business] Total number of AI generation retries",
			},
			[]string{"type", "provider", "retry_count"},
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
				Help:    "[business] Total workflow duration from start to completion",
				Buckets: []float64{60, 300, 600, 1800, 3600, 7200, 14400, 28800, 86400},
			},
			[]string{"workflow_status"},
		),

		ShareEventsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "share_events_total",
				Help: "[business] Share funnel events (issue / open) by kind, platform, and source",
			},
			[]string{"event_type", "kind", "platform", "source"},
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
		m.ActiveRequests,
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
		m.StoryParticipantCount,
		m.StoryboardSceneCount,
		m.StoryboardChildCount,
		m.StoryboardTokenConsumed,
		m.UserTokenConsumed,
		m.ComplianceChecksTotal,
		m.ComplianceCheckResults,
		m.ComplianceViolationsByType,
		m.ComplianceCheckDuration,
		m.PaymentTotal,
		m.PaymentRefundsTotal,
		m.PaymentSubscriptions,
		m.PaymentVerifyTotal,
		m.PaymentVerifyDuration,
		m.OAuthLoginTotal,
		m.OAuthLoginDuration,
		m.OAuthLoginErrors,
		m.NotificationsSentTotal,
		m.NotificationDeliveryDuration,
		m.NotificationErrors,
		m.NotificationByCategory,
		m.StoryboardContentGenerations,
		m.StoryboardContentGenerationTime,
		m.StoryboardSceneGenerations,
		m.StoryboardSceneGenerationTime,
		m.StoryboardImageGenerations,
		m.StoryboardImageGenerationTime,
		m.StoryboardVideoGenerations,
		m.StoryboardVideoGenerationTime,
		m.ImageGenerationWithCharacters,
		m.ImageGenerationWithStyle,
		m.ImageGenerationPromptDetailsUsed,
		m.ImageGenerationCharacterRefs,
		m.ImageGenerationTokenConsumed,
		m.ImageGenerationErrors,
		m.VideoGenerationSubdivided,
		m.VideoGenerationSegmentCount,
		m.VideoGenerationTokenConsumed,
		m.VideoGenerationErrors,
		m.StoryStyleConfigUsage,
		m.StoryStyleConfigCount,
		m.StoryStyleConfigByStyle,
		m.AIGenerationRetries,
		m.StoryboardWorkflowCompleted,
		m.StoryboardWorkflowDuration,
		m.ShareEventsTotal,
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

// Start starts the metrics push goroutine if push gateway is configured.
// Safe to call multiple times; only the first call starts the pusher.
func (m *Metrics) Start(ctx context.Context) {
	m.startOnce.Do(func() {
		if m.pusher == nil || m.config.PushInterval <= 0 {
			return
		}

		m.wg.Add(1)
		go func() {
			defer m.wg.Done()

			ticker := time.NewTicker(time.Duration(m.config.PushInterval) * time.Second)
			defer ticker.Stop()

			if err := m.pusher.Push(); err != nil {
				fmt.Printf("Failed to push metrics to PushGateway on start: %v\n", err)
			}

			for {
				select {
				case <-ticker.C:
					if err := m.pusher.Push(); err != nil {
						fmt.Printf("Failed to push metrics to PushGateway: %v\n", err)
					}
				case <-ctx.Done():
					_ = m.pusher.Push()
					return
				case <-m.stopChan:
					_ = m.pusher.Push()
					return
				}
			}
		}()
	})
}

// Stop stops the metrics push goroutine.
func (m *Metrics) Stop() {
	m.stopOnce.Do(func() {
		close(m.stopChan)
	})
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

// RecordHTTPRequest records an HTTP request (system tier).
func (m *Metrics) RecordHTTPRequest(method, path, status string, duration time.Duration, requestSize, responseSize float64) {
	path = NormalizeMetricPath(path)
	m.HTTPRequestsTotal.WithLabelValues(method, path, status).Inc()
	m.HTTPRequestDuration.WithLabelValues(method, path, status).Observe(duration.Seconds())
	m.HTTPRequestSize.WithLabelValues(method, path).Observe(requestSize)
	m.HTTPResponseSize.WithLabelValues(method, path).Observe(responseSize)
}

// RecordHTTPError records an HTTP error (system tier).
func (m *Metrics) RecordHTTPError(errorCode, method, path string, duration time.Duration) {
	path = NormalizeMetricPath(path)
	m.HTTPErrorTotal.WithLabelValues(errorCode, method, path).Inc()
	m.HTTPErrorDuration.WithLabelValues(errorCode, method, path).Observe(duration.Seconds())
}

// RecordHTTPErrorSimple records a simple HTTP error without duration.
func (m *Metrics) RecordHTTPErrorSimple(errorCode, method, path string) {
	path = NormalizeMetricPath(path)
	m.HTTPErrorTotal.WithLabelValues(errorCode, method, path).Inc()
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

// RecordStoryParticipantCount records the number of participants in a story.
func (m *Metrics) RecordStoryParticipantCount(_ string, count float64) {
	m.StoryParticipantCount.Observe(count)
}

// RecordStoryboardSceneCount records the number of scenes in a storyboard.
func (m *Metrics) RecordStoryboardSceneCount(_ string, count float64) {
	m.StoryboardSceneCount.Observe(count)
}

// RecordStoryboardChildCount records the number of child storyboards.
func (m *Metrics) RecordStoryboardChildCount(_ string, count float64) {
	m.StoryboardChildCount.Observe(count)
}

// RecordStoryboardTokenConsumed records token consumption for a storyboard.
func (m *Metrics) RecordStoryboardTokenConsumed(_ string, tokens float64) {
	m.StoryboardTokenConsumed.Observe(tokens)
}

// RecordUserTokenConsumed records token consumption by a user.
func (m *Metrics) RecordUserTokenConsumed(_ string, tokens float64) {
	m.UserTokenConsumed.Observe(tokens)
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

// RecordPaymentSimple records a payment transaction counter.
func (m *Metrics) RecordPaymentSimple(provider, paymentType, status string) {
	m.PaymentTotal.WithLabelValues(provider, paymentType, status).Inc()
}

// IncPaymentSubscription increments active subscription count.
func (m *Metrics) IncPaymentSubscription(provider, plan string) {
	m.PaymentSubscriptions.WithLabelValues(provider, plan).Inc()
}

// DecPaymentSubscription decrements active subscription count.
func (m *Metrics) DecPaymentSubscription(provider, plan string) {
	m.PaymentSubscriptions.WithLabelValues(provider, plan).Dec()
}

// RecordPaymentVerify records a payment verification request.
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

// RecordOAuthLoginError records an OAuth login error.
func (m *Metrics) RecordOAuthLoginError(provider, errorType string) {
	m.OAuthLoginErrors.WithLabelValues(provider, errorType).Inc()
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

// RecordNotificationByCategory records a notification by category.
func (m *Metrics) RecordNotificationByCategory(category string) {
	m.NotificationByCategory.WithLabelValues(category).Inc()
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
// Story Style Configuration Metrics Recording Methods
// ============================================

// RecordStoryStyleConfigUsage records usage of a story style configuration by type.
func (m *Metrics) RecordStoryStyleConfigUsage(_ string, usageType string) {
	m.StoryStyleConfigUsage.WithLabelValues(usageType).Inc()
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

// RecordAIGenerationRetry records an AI generation retry.
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

// RecordStoryboardWorkflowCompleted records a completed storyboard workflow.
func (m *Metrics) RecordStoryboardWorkflowCompleted(workflowStatus string, duration time.Duration) {
	m.StoryboardWorkflowCompleted.WithLabelValues(workflowStatus).Inc()
	if duration > 0 {
		m.StoryboardWorkflowDuration.WithLabelValues(workflowStatus).Observe(duration.Seconds())
	}
}

// RecordShareEvent records a share funnel event (issue / open).
func (m *Metrics) RecordShareEvent(eventType, kind, platform, source string) {
	if m == nil || m.ShareEventsTotal == nil {
		return
	}
	if eventType == "" {
		eventType = "unknown"
	}
	if kind == "" {
		kind = "unknown"
	}
	if platform == "" {
		platform = "unknown"
	}
	if source == "" {
		source = "unknown"
	}
	m.ShareEventsTotal.WithLabelValues(eventType, kind, platform, source).Inc()
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
