# Grafana Dashboard Index

Complete list of all monitoring dashboards organized by functional area.

## Quick Reference

| # | Dashboard Name | File | Key Metrics |
|---|---------------|------|-------------|
| 01 | HTTP Metrics | `01-http-metrics.json` | Request rate, duration, size |
| 02 | Application Metrics | `02-application-metrics.json` | Errors, DB queries, cache |
| 03 | Business Metrics | `03-business-metrics.json` | Users, stories, AI generations |
| 04 | Content Metrics | `04-content-metrics.json` | Stories, storyboards, characters |
| 05 | User Activity | `05-user-activity-metrics.json` | DAU, WAU, MAU, growth |
| 06 | Compliance | `06-compliance-metrics.json` | Compliance checks, violations |
| 07 | Payment | `07-payment-metrics.json` | Payments, refunds, subscriptions |
| 08 | OAuth | `08-oauth-metrics.json` | OAuth logins, errors |
| 09 | Notification | `09-notification-metrics.json` | Notifications sent, delivery |
| 10 | Storyboard Workflow | `10-storyboard-generation-workflow.json` | Content/scene/image/video generation |
| 11 | Image Generation | `11-image-generation-detailed.json` | Image gen details, characters, style |
| 12 | Video Generation | `12-video-generation-detailed.json` | Video gen details, subdivision |
| 13 | Poster Generation | `13-character-poster-generation.json` | Poster concept and image gen |
| 14 | Style Config | `14-story-style-configuration.json` | Style config usage and count |
| 15 | AI Quality | `15-ai-generation-quality.json` | Success rate, tokens, duration |
| 16 | Workflow Completion | `16-storyboard-workflow-completion.json` | Workflow completion and abandonment |

## Dashboard Details

### Infrastructure & Application

#### 01. HTTP Metrics Dashboard
**File**: `01-http-metrics.json`  
**Tags**: `http`, `metrics`  
**Panels**:
- HTTP Request Rate by Method
- HTTP Request Rate by Status
- HTTP Request Duration (p50/p95/p99)
- Active HTTP Requests
- HTTP Request/Response Size
- Total HTTP Request Rate

#### 02. Application Metrics Dashboard
**File**: `02-application-metrics.json`  
**Tags**: `application`, `metrics`  
**Panels**:
- Active Requests
- Error Rate by Type
- Database Query Duration (p50/p95/p99)
- Cache Hits/Misses Rate
- Cache Hit Rate (%)

### Business & Content

#### 03. Business Metrics Dashboard
**File**: `03-business-metrics.json`  
**Tags**: `business`, `metrics`  
**Panels**:
- User Registration Rate
- User Login Rate
- Story Creation Rate by Genre
- AI Generation Rate by Type and Provider
- Total Users/Stories/Storyboards
- Daily Active Users

#### 04. Content Metrics Dashboard
**File**: `04-content-metrics.json`  
**Tags**: `content`, `metrics`  
**Panels**:
- Total Storyboards/Stories/Users
- Character Message Rate
- Character Token Consumption
- Storyboard Scene Count Distribution
- Storyboard Token Consumption
- User Token Consumption
- Group Member Count Distribution
- Story Participant Count Distribution

#### 05. User Activity Metrics Dashboard
**File**: `05-user-activity-metrics.json`  
**Tags**: `user`, `activity`  
**Panels**:
- Daily/Weekly/Monthly Active Users
- User Growth Rate (YoY/MoM)

### Security & Compliance

#### 06. Compliance Metrics Dashboard
**File**: `06-compliance-metrics.json`  
**Tags**: `compliance`  
**Panels**:
- Compliance Checks Rate
- Compliance Check Results
- Compliance Violations by Type
- Compliance Check Duration

### Payment & Authentication

#### 07. Payment Metrics Dashboard
**File**: `07-payment-metrics.json`  
**Tags**: `payment`  
**Panels**:
- Payment Rate by Provider and Type
- Payment Amount Distribution
- Payment Duration
- Payment Refunds
- Active Subscriptions
- Payment Verification Rate and Duration

#### 08. OAuth Metrics Dashboard
**File**: `08-oauth-metrics.json`  
**Tags**: `oauth`, `authentication`  
**Panels**:
- OAuth Login Rate
- OAuth Login Duration
- OAuth Login Errors
- OAuth Token Refresh Rate
- OAuth Link/Unlink Operations
- OAuth Active Providers

### Notifications

#### 09. Notification Metrics Dashboard
**File**: `09-notification-metrics.json`  
**Tags**: `notification`  
**Panels**:
- Notifications Sent Rate
- Notification Delivery Duration
- Notification Errors
- Notifications Queued
- Notification Delivery Rate
- Notifications by Category
- Notification Retries

### AI Generation Workflows

#### 10. Storyboard Generation Workflow Dashboard
**File**: `10-storyboard-generation-workflow.json`  
**Tags**: `storyboard`, `generation`, `workflow`  
**Panels**:
- Content Generation Rate and Duration
- Scene Generation Rate and Duration
- Image Generation Rate and Duration (by scene type)
- Video Generation Rate and Duration (by subdivision)

#### 11. Image Generation Detailed Dashboard
**File**: `11-image-generation-detailed.json`  
**Tags**: `image`, `generation`, `detailed`  
**Panels**:
- Image Generation with Characters
- Image Generation with Style
- Prompt Details Usage
- Character References Count
- Token Consumption
- Error Types

#### 12. Video Generation Detailed Dashboard
**File**: `12-video-generation-detailed.json`  
**Tags**: `video`, `generation`, `detailed`  
**Panels**:
- Video Generation Subdivision
- Video Segment Count
- Token Consumption
- Error Types

#### 13. Character Poster Generation Dashboard
**File**: `13-character-poster-generation.json`  
**Tags**: `character`, `poster`, `generation`  
**Panels**:
- Poster Generation Rate
- Concept Generation Duration
- Image Generation Duration
- Token Consumption
- Error Types

### Configuration & Quality

#### 14. Story Style Configuration Dashboard
**File**: `14-story-style-configuration.json`  
**Tags**: `style`, `configuration`  
**Panels**:
- Total Style Configurations
- Style Config Usage Rate
- Style Config Count by Style Name

#### 15. AI Generation Quality Dashboard
**File**: `15-ai-generation-quality.json`  
**Tags**: `ai`, `quality`  
**Panels**:
- AI Generation Success Rate
- Average Token Consumption
- Average Generation Duration
- Retry Attempts

#### 16. Storyboard Workflow Completion Dashboard
**File**: `16-storyboard-workflow-completion.json`  
**Tags**: `storyboard`, `workflow`, `completion`  
**Panels**:
- Workflow Completion Rate
- Workflow Duration
- Workflow Abandonment

## Usage Tips

1. **Start with Overview Dashboards**: Begin with Business Metrics (03) and Application Metrics (02) for a high-level view
2. **Deep Dive**: Use detailed dashboards (11-13) for troubleshooting specific generation issues
3. **Monitor Workflows**: Use Workflow dashboards (10, 16) to track end-to-end process health
4. **Set Alerts**: Configure Grafana alerts based on thresholds in these dashboards
5. **Customize**: Modify panels to match your specific monitoring needs

## Metric Naming Convention

All metrics follow Prometheus standards:
- **Counters**: `*_total` - use `rate()` or `increase()` in queries
- **Histograms**: `*_bucket`, `*_sum`, `*_count` - use `histogram_quantile()` for percentiles
- **Gauges**: Direct metric names - use as-is or with `avg()`, `max()`, etc.

## Related Files

- `generate_dashboards.py` - Script to regenerate all dashboards
- `README.md` - Installation and usage instructions
- `../internal/telemetry/prometheus.go` - Metric definitions

