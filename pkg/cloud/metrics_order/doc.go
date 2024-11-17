package metricsorder

// pkg/cloud/metrics_order/doc.go
type MetricsOrder struct {
	AppKey    string
	AppSecret string
	Address   string
}

func NewMetricsOrder(appKey, appSecret, address string) *MetricsOrder {
	return &MetricsOrder{
		AppKey:    appKey,
		AppSecret: appSecret,
		Address:   address,
	}
}
