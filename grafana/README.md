# Grafana Dashboards for FGrapery

This directory contains Grafana dashboard configurations for monitoring all aspects of the FGrapery application.

## Dashboard Overview

Each functional area has its own independent dashboard:

### 1. HTTP Metrics Dashboard (`01-http-metrics.json`)
- HTTP request rate by method and status
- HTTP request duration (p50/p95/p99)
- Active HTTP requests
- Request/response size

### 2. Application Metrics Dashboard (`02-application-metrics.json`)
- Active requests
- Error rate by type
- Database query duration
- Cache hits/misses rate
- Cache hit rate percentage

### 3. Business Metrics Dashboard (`03-business-metrics.json`)
- User registration rate
- User login rate
- Story creation rate by genre
- AI generation rate by type and provider
- Total users, stories, storyboards
- Daily active users

### 4. Content Metrics Dashboard (`04-content-metrics.json`)
- Total storyboards, stories, users
- Character message rate
- Character token consumption
- Storyboard scene count distribution
- Storyboard token consumption
- User token consumption
- Group member count distribution
- Story participant count distribution

### 5. User Activity Metrics Dashboard (`05-user-activity-metrics.json`)
- Daily/Weekly/Monthly active users
- User growth rate (YoY/MoM)

### 6. Compliance Metrics Dashboard (`06-compliance-metrics.json`)
- Compliance checks rate
- Compliance check results
- Compliance violations by type
- Compliance check duration

### 7. Payment Metrics Dashboard (`07-payment-metrics.json`)
- Payment rate by provider and type
- Payment amount distribution
- Payment duration
- Payment refunds
- Active subscriptions
- Payment verification rate and duration

### 8. OAuth Metrics Dashboard (`08-oauth-metrics.json`)
- OAuth login rate
- OAuth login duration
- OAuth login errors
- OAuth token refresh rate
- OAuth link/unlink operations
- OAuth active providers

### 9. Notification Metrics Dashboard (`09-notification-metrics.json`)
- Notifications sent rate
- Notification delivery duration
- Notification errors
- Notifications queued
- Notification delivery rate
- Notifications by category
- Notification retries

### 10. Storyboard Generation Workflow Dashboard (`10-storyboard-generation-workflow.json`)
- Content generation rate and duration
- Scene generation rate and duration
- Image generation rate and duration (by scene type)
- Video generation rate and duration (by subdivision)

### 11. Image Generation Detailed Dashboard (`11-image-generation-detailed.json`)
- Image generation with characters
- Image generation with style
- Prompt details usage
- Character references count
- Token consumption
- Error types

### 12. Video Generation Detailed Dashboard (`12-video-generation-detailed.json`)
- Video generation subdivision
- Video segment count
- Token consumption
- Error types

### 13. Character Poster Generation Dashboard (`13-character-poster-generation.json`)
- Poster generation rate
- Concept generation duration
- Image generation duration
- Token consumption
- Error types

### 14. Story Style Configuration Dashboard (`14-story-style-configuration.json`)
- Total style configurations
- Style config usage rate
- Style config count by style name

### 15. AI Generation Quality Dashboard (`15-ai-generation-quality.json`)
- AI generation success rate
- Average token consumption
- Average generation duration
- Retry attempts

### 16. Storyboard Workflow Completion Dashboard (`16-storyboard-workflow-completion.json`)
- Workflow completion rate
- Workflow duration
- Workflow abandonment

## Installation

### Option 1: Import via Grafana UI

1. Open Grafana and navigate to Dashboards → Import
2. Click "Upload JSON file"
3. Select the dashboard JSON file you want to import
4. Configure the Prometheus datasource (set variable `datasource` to your Prometheus instance)
5. Click "Import"

### Option 2: Provision via Grafana Provisioning

1. Copy dashboard files to Grafana's provisioning directory:
   ```bash
   cp dashboards/*.json /etc/grafana/provisioning/dashboards/
   ```

2. Create a provisioning configuration file `/etc/grafana/provisioning/dashboards/dashboards.yml`:
   ```yaml
   apiVersion: 1
   providers:
     - name: 'FGrapery Dashboards'
       orgId: 1
       folder: 'FGrapery'
       type: file
       disableDeletion: false
       updateIntervalSeconds: 10
       allowUiUpdates: true
       options:
         path: /etc/grafana/provisioning/dashboards
         foldersFromFilesStructure: true
   ```

3. Restart Grafana

### Option 3: Use Grafana API

```bash
# Set your Grafana URL and API key
GRAFANA_URL="http://localhost:3000"
API_KEY="your-api-key"

# Import all dashboards
for file in dashboards/*.json; do
  curl -X POST \
    -H "Authorization: Bearer $API_KEY" \
    -H "Content-Type: application/json" \
    -d @"$file" \
    "$GRAFANA_URL/api/dashboards/db"
done
```

## Configuration

### Datasource Variable

All dashboards use a template variable `datasource` that should be set to your Prometheus datasource name. The default is "Prometheus".

To change this:
1. Open the dashboard in Grafana
2. Go to Dashboard Settings → Variables
3. Edit the `datasource` variable
4. Select your Prometheus datasource

### Time Range

Default time range is "Last 6 hours". You can change this:
- In the dashboard UI using the time picker
- In the JSON file by modifying the `time.from` field

### Refresh Interval

Default refresh interval is 30 seconds. You can change this:
- In the dashboard UI using the refresh dropdown
- In the JSON file by modifying the `refresh` field

## Metric Names

All metrics follow Prometheus naming conventions:
- Counters: `*_total`
- Histograms: `*_bucket`, `*_sum`, `*_count`
- Gauges: direct metric names

For the complete list of metrics and their labels, refer to `internal/telemetry/prometheus.go`.

## Regenerating Dashboards

If you need to regenerate dashboards after modifying the script:

```bash
cd grapery/grafana
python3 generate_dashboards.py
```

## Troubleshooting

### Metrics not showing

1. Verify Prometheus is scraping your application:
   ```bash
   curl http://localhost:9090/api/v1/targets
   ```

2. Check if metrics are being exposed:
   ```bash
   curl http://your-app:port/metrics | grep "metric_name"
   ```

3. Verify the datasource variable is correctly set in Grafana

### Panels showing "No data"

1. Check if the metric exists in Prometheus:
   ```bash
   curl 'http://localhost:9090/api/v1/query?query=metric_name'
   ```

2. Verify the PromQL query syntax in the panel
3. Check the time range - metrics might not exist for the selected time period

### Performance issues

- Reduce refresh interval for heavy dashboards
- Use recording rules in Prometheus for complex queries
- Limit the number of series returned by using more specific label filters

## Customization

Each dashboard JSON file can be customized:
- Add/remove panels
- Modify panel queries
- Change visualization types
- Adjust thresholds and alerts
- Add annotations

After customization, you can re-import the dashboard or update it via the Grafana UI.

## Related Documentation

- Prometheus Metrics Guide: `docs/PROMETHEUS_METRICS_GUIDE.md`
- Telemetry README: `internal/telemetry/README.md`

