package telemetry

// PrometheusConfig holds Prometheus configuration.
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

// MetricTier distinguishes system-level from business-level instrumentation.
type MetricTier string

const (
	TierSystem   MetricTier = "system"
	TierBusiness MetricTier = "business"
)
