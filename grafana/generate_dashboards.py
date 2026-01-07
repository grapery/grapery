#!/usr/bin/env python3
"""
Generate Grafana dashboard JSON files for all monitoring metrics.
Each functional area gets its own independent dashboard.
"""

import json
import os

# Base dashboard template
BASE_TEMPLATE = {
    "annotations": {
        "list": [
            {
                "builtIn": 1,
                "datasource": "-- Grafana --",
                "enable": True,
                "hide": True,
                "iconColor": "rgba(0, 211, 255, 1)",
                "name": "Annotations & Alerts",
                "type": "dashboard"
            }
        ]
    },
    "editable": True,
    "fiscalYearStartMonth": 0,
    "graphTooltip": 1,
    "id": None,
    "links": [],
    "liveNow": False,
    "panels": [],
    "refresh": "30s",
    "schemaVersion": 38,
    "style": "dark",
    "tags": [],
    "templating": {
        "list": [
            {
                "current": {
                    "selected": False,
                    "text": "Prometheus",
                    "value": "Prometheus"
                },
                "hide": 0,
                "label": "Datasource",
                "name": "datasource",
                "options": [],
                "query": "prometheus",
                "refresh": 1,
                "regex": "",
                "type": "datasource"
            }
        ]
    },
    "time": {
        "from": "now-6h",
        "to": "now"
    },
    "timepicker": {},
    "timezone": "",
    "title": "",
    "uid": "",
    "version": 1
}

def create_timeseries_panel(id, title, expr, legend_format, grid_pos, unit="ops", thresholds=None):
    """Create a timeseries panel"""
    panel = {
        "datasource": "${datasource}",
        "fieldConfig": {
            "defaults": {
                "color": {"mode": "palette-classic"},
                "custom": {
                    "axisCenteredZero": False,
                    "axisColorMode": "text",
                    "axisLabel": "",
                    "axisPlacement": "auto",
                    "barAlignment": 0,
                    "drawStyle": "line",
                    "fillOpacity": 10,
                    "gradientMode": "none",
                    "hideFrom": {"legend": False, "tooltip": False, "viz": False},
                    "lineInterpolation": "smooth",
                    "lineWidth": 2,
                    "pointSize": 5,
                    "scaleDistribution": {"type": "linear"},
                    "showPoints": "never",
                    "spanNulls": False,
                    "stacking": {"group": "A", "mode": "none"},
                    "thresholdsStyle": {"mode": "off"}
                },
                "mappings": [],
                "thresholds": {"mode": "absolute", "steps": [{"color": "green", "value": None}]},
                "unit": unit
            },
            "overrides": []
        },
        "gridPos": grid_pos,
        "id": id,
        "options": {
            "legend": {"calcs": ["sum"], "displayMode": "table", "placement": "bottom", "showLegend": True},
            "tooltip": {"mode": "multi", "sort": "desc"}
        },
        "targets": [
            {
                "datasource": "${datasource}",
                "expr": expr,
                "legendFormat": legend_format,
                "refId": "A"
            }
        ],
        "title": title,
        "type": "timeseries"
    }
    if thresholds:
        panel["fieldConfig"]["defaults"]["thresholds"]["steps"] = thresholds
    return panel

def create_stat_panel(id, title, expr, grid_pos, thresholds=None):
    """Create a stat panel"""
    panel = {
        "datasource": "${datasource}",
        "fieldConfig": {
            "defaults": {
                "color": {"mode": "thresholds"},
                "mappings": [],
                "thresholds": {"mode": "absolute", "steps": [{"color": "green", "value": None}]},
                "unit": "short"
            },
            "overrides": []
        },
        "gridPos": grid_pos,
        "id": id,
        "options": {
            "colorMode": "value",
            "graphMode": "area",
            "justifyMode": "auto",
            "orientation": "auto",
            "reduceOptions": {
                "calcs": ["lastNotNull"],
                "fields": "",
                "values": False
            },
            "textMode": "auto"
        },
        "pluginVersion": "9.0.0",
        "targets": [
            {
                "datasource": "${datasource}",
                "expr": expr,
                "refId": "A"
            }
        ],
        "title": title,
        "type": "stat"
    }
    if thresholds:
        panel["fieldConfig"]["defaults"]["thresholds"]["steps"] = thresholds
    return panel

def create_histogram_panel(id, title, metric_name, grid_pos, unit="s", quantiles=["0.50", "0.95", "0.99"]):
    """Create a histogram quantile panel"""
    targets = []
    ref_ids = ["A", "B", "C"]
    for i, q in enumerate(quantiles[:len(ref_ids)]):
        targets.append({
            "datasource": "${datasource}",
            "expr": f"histogram_quantile({q}, sum(rate({metric_name}_bucket[$__rate_interval])) by (le))",
            "legendFormat": f"p{int(float(q)*100)}",
            "refId": ref_ids[i]
        })
    
    return {
        "datasource": "${datasource}",
        "fieldConfig": {
            "defaults": {
                "color": {"mode": "palette-classic"},
                "custom": {
                    "axisCenteredZero": False,
                    "axisColorMode": "text",
                    "axisLabel": "",
                    "axisPlacement": "auto",
                    "barAlignment": 0,
                    "drawStyle": "line",
                    "fillOpacity": 10,
                    "gradientMode": "none",
                    "hideFrom": {"legend": False, "tooltip": False, "viz": False},
                    "lineInterpolation": "smooth",
                    "lineWidth": 2,
                    "pointSize": 5,
                    "scaleDistribution": {"type": "linear"},
                    "showPoints": "never",
                    "spanNulls": False,
                    "stacking": {"group": "A", "mode": "none"},
                    "thresholdsStyle": {"mode": "off"}
                },
                "mappings": [],
                "thresholds": {
                    "mode": "absolute",
                    "steps": [
                        {"color": "green", "value": None},
                        {"color": "yellow", "value": 1},
                        {"color": "red", "value": 5}
                    ]
                },
                "unit": unit
            },
            "overrides": []
        },
        "gridPos": grid_pos,
        "id": id,
        "options": {
            "legend": {"calcs": ["mean", "p95", "p99"], "displayMode": "table", "placement": "bottom", "showLegend": True},
            "tooltip": {"mode": "multi", "sort": "desc"}
        },
        "targets": targets,
        "title": title,
        "type": "timeseries"
    }

def create_row_panel(id, title, grid_pos, collapsed=False):
    """Create a row panel"""
    return {
        "collapsed": collapsed,
        "gridPos": grid_pos,
        "id": id,
        "panels": [],
        "title": title,
        "type": "row"
    }

# Dashboard configurations
DASHBOARDS = {
    "04-content-metrics": {
        "title": "Content Metrics Dashboard",
        "uid": "content-metrics",
        "tags": ["content", "metrics"],
        "panels": [
            create_stat_panel(1, "Total Storyboards", "storyboard_count", {"h": 4, "w": 6, "x": 0, "y": 0}),
            create_stat_panel(2, "Total Stories", "story_count", {"h": 4, "w": 6, "x": 6, "y": 0}),
            create_stat_panel(3, "Total Users", "user_count", {"h": 4, "w": 6, "x": 12, "y": 0}),
            create_timeseries_panel(4, "Character Message Rate", 
                "sum(rate(character_message_count_total[$__rate_interval])) by (character_id)",
                "{{ character_id }}", {"h": 8, "w": 12, "x": 0, "y": 4}),
            create_timeseries_panel(5, "Character Token Consumption",
                "sum(rate(character_token_consumed_sum[$__rate_interval])) by (character_id)",
                "{{ character_id }}", {"h": 8, "w": 12, "x": 12, "y": 4}, unit="tokens"),
            create_histogram_panel(6, "Storyboard Scene Count Distribution",
                "storyboard_scene_count", {"h": 8, "w": 12, "x": 0, "y": 12}, unit="short"),
            create_histogram_panel(7, "Storyboard Token Consumption",
                "storyboard_token_consumed", {"h": 8, "w": 12, "x": 12, "y": 12}, unit="tokens"),
            create_histogram_panel(8, "User Token Consumption",
                "user_token_consumed", {"h": 8, "w": 12, "x": 0, "y": 20}, unit="tokens"),
            create_histogram_panel(9, "Group Member Count Distribution",
                "group_member_count", {"h": 8, "w": 12, "x": 12, "y": 20}, unit="short"),
            create_histogram_panel(10, "Story Participant Count Distribution",
                "story_participant_count", {"h": 8, "w": 12, "x": 0, "y": 28}, unit="short")
        ]
    },
    "05-user-activity-metrics": {
        "title": "User Activity Metrics Dashboard",
        "uid": "user-activity-metrics",
        "tags": ["user", "activity"],
        "panels": [
            create_stat_panel(1, "Daily Active Users", "daily_active_users", {"h": 4, "w": 8, "x": 0, "y": 0}),
            create_stat_panel(2, "Weekly Active Users", "weekly_active_users", {"h": 4, "w": 8, "x": 8, "y": 0}),
            create_stat_panel(3, "Monthly Active Users", "monthly_active_users", {"h": 4, "w": 8, "x": 16, "y": 0}),
            create_timeseries_panel(4, "User Growth Rate",
                "user_growth_rate", "{{ type }}", {"h": 8, "w": 24, "x": 0, "y": 4}, unit="percent")
        ]
    },
    "06-compliance-metrics": {
        "title": "Compliance Metrics Dashboard",
        "uid": "compliance-metrics",
        "tags": ["compliance"],
        "panels": [
            create_timeseries_panel(1, "Compliance Checks Rate",
                "sum(rate(compliance_checks_total[$__rate_interval]))",
                "Total", {"h": 8, "w": 12, "x": 0, "y": 0}),
            create_timeseries_panel(2, "Compliance Check Results",
                "sum(rate(compliance_check_results_total[$__rate_interval])) by (status)",
                "{{ status }}", {"h": 8, "w": 12, "x": 12, "y": 0}),
            create_timeseries_panel(3, "Compliance Violations by Type",
                "sum(rate(compliance_violations_by_type_total[$__rate_interval])) by (violation_type)",
                "{{ violation_type }}", {"h": 8, "w": 12, "x": 0, "y": 8}),
            create_histogram_panel(4, "Compliance Check Duration",
                "compliance_check_duration_seconds", {"h": 8, "w": 12, "x": 12, "y": 8})
        ]
    },
    "07-payment-metrics": {
        "title": "Payment Metrics Dashboard",
        "uid": "payment-metrics",
        "tags": ["payment"],
        "panels": [
            create_timeseries_panel(1, "Payment Rate by Provider and Type",
                "sum(rate(payment_total[$__rate_interval])) by (provider, type, status)",
                "{{ provider }}/{{ type }}/{{ status }}", {"h": 8, "w": 12, "x": 0, "y": 0}),
            create_histogram_panel(2, "Payment Amount Distribution",
                "payment_amount", {"h": 8, "w": 12, "x": 12, "y": 0}, unit="currencyUSD"),
            create_histogram_panel(3, "Payment Duration",
                "payment_duration_seconds", {"h": 8, "w": 12, "x": 0, "y": 8}),
            create_timeseries_panel(4, "Payment Refunds",
                "sum(rate(payment_refunds_total[$__rate_interval])) by (provider, reason)",
                "{{ provider }}/{{ reason }}", {"h": 8, "w": 12, "x": 12, "y": 8}),
            create_stat_panel(5, "Active Subscriptions",
                "sum(payment_subscriptions_active)", {"h": 4, "w": 8, "x": 0, "y": 16}),
            create_timeseries_panel(6, "Payment Verification Rate",
                "sum(rate(payment_verify_total[$__rate_interval])) by (provider, status)",
                "{{ provider }}/{{ status }}", {"h": 8, "w": 12, "x": 0, "y": 20}),
            create_histogram_panel(7, "Payment Verification Duration",
                "payment_verify_duration_seconds", {"h": 8, "w": 12, "x": 12, "y": 20})
        ]
    },
    "08-oauth-metrics": {
        "title": "OAuth Metrics Dashboard",
        "uid": "oauth-metrics",
        "tags": ["oauth", "authentication"],
        "panels": [
            create_timeseries_panel(1, "OAuth Login Rate",
                "sum(rate(oauth_login_total[$__rate_interval])) by (provider, status)",
                "{{ provider }}/{{ status }}", {"h": 8, "w": 12, "x": 0, "y": 0}),
            create_histogram_panel(2, "OAuth Login Duration",
                "oauth_login_duration_seconds", {"h": 8, "w": 12, "x": 12, "y": 0}),
            create_timeseries_panel(3, "OAuth Login Errors",
                "sum(rate(oauth_login_errors_total[$__rate_interval])) by (provider, error_type)",
                "{{ provider }}/{{ error_type }}", {"h": 8, "w": 12, "x": 0, "y": 8}),
            create_timeseries_panel(4, "OAuth Token Refresh Rate",
                "sum(rate(oauth_token_refresh_total[$__rate_interval])) by (provider, status)",
                "{{ provider }}/{{ status }}", {"h": 8, "w": 12, "x": 12, "y": 8}),
            create_timeseries_panel(5, "OAuth Link/Unlink Operations",
                "sum(rate(oauth_link_total[$__rate_interval])) by (provider, action, status)",
                "{{ provider }}/{{ action }}/{{ status }}", {"h": 8, "w": 12, "x": 0, "y": 16}),
            create_stat_panel(6, "OAuth Active Providers",
                "sum(oauth_active_providers)", {"h": 4, "w": 12, "x": 12, "y": 16})
        ]
    },
    "09-notification-metrics": {
        "title": "Notification Metrics Dashboard",
        "uid": "notification-metrics",
        "tags": ["notification"],
        "panels": [
            create_timeseries_panel(1, "Notifications Sent Rate",
                "sum(rate(notifications_sent_total[$__rate_interval])) by (type, channel, status)",
                "{{ type }}/{{ channel }}/{{ status }}", {"h": 8, "w": 12, "x": 0, "y": 0}),
            create_histogram_panel(2, "Notification Delivery Duration",
                "notification_delivery_duration_seconds", {"h": 8, "w": 12, "x": 12, "y": 0}),
            create_timeseries_panel(3, "Notification Errors",
                "sum(rate(notification_errors_total[$__rate_interval])) by (type, channel, error_type)",
                "{{ type }}/{{ channel }}/{{ error_type }}", {"h": 8, "w": 12, "x": 0, "y": 8}),
            create_stat_panel(4, "Notifications Queued",
                "notifications_queued", {"h": 4, "w": 6, "x": 12, "y": 8}),
            create_stat_panel(5, "Notification Delivery Rate",
                "notification_delivery_rate", {"h": 4, "w": 6, "x": 18, "y": 8}, thresholds=[
                    {"color": "red", "value": None},
                    {"color": "yellow", "value": 0.7},
                    {"color": "green", "value": 0.9}
                ]),
            create_timeseries_panel(6, "Notifications by Category",
                "sum(rate(notifications_by_category_total[$__rate_interval])) by (category)",
                "{{ category }}", {"h": 8, "w": 12, "x": 0, "y": 12}),
            create_timeseries_panel(7, "Notification Retries",
                "sum(rate(notification_retries_total[$__rate_interval])) by (type, channel)",
                "{{ type }}/{{ channel }}", {"h": 8, "w": 12, "x": 12, "y": 12})
        ]
    },
    "11-image-generation-detailed": {
        "title": "Image Generation Detailed Dashboard",
        "uid": "image-generation-detailed",
        "tags": ["image", "generation", "detailed"],
        "panels": [
            create_row_panel(1, "📊 Image Generation Overview", {"h": 1, "w": 24, "x": 0, "y": 0}),
            create_timeseries_panel(2, "Image Generation with Characters",
                "sum(rate(image_generation_with_characters_total[$__rate_interval])) by (status, character_count)",
                "{{ status }}/{{ character_count }}", {"h": 8, "w": 12, "x": 0, "y": 1}),
            create_timeseries_panel(3, "Image Generation with Style",
                "sum(rate(image_generation_with_style_total[$__rate_interval])) by (status, has_style)",
                "{{ status }}/{{ has_style }}", {"h": 8, "w": 12, "x": 12, "y": 1}),
            create_timeseries_panel(4, "Prompt Details Usage",
                "sum(rate(image_generation_prompt_details_used_total[$__rate_interval])) by (has_prompt_details)",
                "{{ has_prompt_details }}", {"h": 8, "w": 12, "x": 0, "y": 9}),
            create_histogram_panel(5, "Character References Count",
                "image_generation_character_refs_count", {"h": 8, "w": 12, "x": 12, "y": 9}, unit="short"),
            create_histogram_panel(6, "Image Generation Token Consumption",
                "image_generation_token_consumed", {"h": 8, "w": 12, "x": 0, "y": 17}, unit="tokens"),
            create_timeseries_panel(7, "Image Generation Errors",
                "sum(rate(image_generation_errors_total[$__rate_interval])) by (error_type)",
                "{{ error_type }}", {"h": 8, "w": 12, "x": 12, "y": 17})
        ]
    },
    "12-video-generation-detailed": {
        "title": "Video Generation Detailed Dashboard",
        "uid": "video-generation-detailed",
        "tags": ["video", "generation", "detailed"],
        "panels": [
            create_timeseries_panel(1, "Video Generation Subdivision",
                "sum(rate(video_generation_subdivided_total[$__rate_interval])) by (is_subdivided, status)",
                "{{ is_subdivided }}/{{ status }}", {"h": 8, "w": 12, "x": 0, "y": 0}),
            create_histogram_panel(2, "Video Segment Count",
                "video_generation_segment_count", {"h": 8, "w": 12, "x": 12, "y": 0}, unit="short"),
            create_histogram_panel(3, "Video Generation Token Consumption",
                "video_generation_token_consumed", {"h": 8, "w": 12, "x": 0, "y": 8}, unit="tokens"),
            create_timeseries_panel(4, "Video Generation Errors",
                "sum(rate(video_generation_errors_total[$__rate_interval])) by (error_type)",
                "{{ error_type }}", {"h": 8, "w": 12, "x": 12, "y": 8})
        ]
    },
    "13-character-poster-generation": {
        "title": "Character Poster Generation Dashboard",
        "uid": "character-poster-generation",
        "tags": ["character", "poster", "generation"],
        "panels": [
            create_row_panel(1, "🎨 Concept Generation", {"h": 1, "w": 24, "x": 0, "y": 0}),
            create_timeseries_panel(2, "Poster Generation Rate",
                "sum(rate(character_poster_generations_total[$__rate_interval])) by (status, has_story_reference)",
                "{{ status }}/{{ has_story_reference }}", {"h": 8, "w": 12, "x": 0, "y": 1}),
            create_histogram_panel(3, "Concept Generation Duration",
                "character_poster_concept_duration_seconds", {"h": 8, "w": 12, "x": 12, "y": 1}),
            create_row_panel(4, "🖼️ Image Generation", {"h": 1, "w": 24, "x": 0, "y": 9}),
            create_histogram_panel(5, "Image Generation Duration",
                "character_poster_image_duration_seconds", {"h": 8, "w": 12, "x": 0, "y": 10}),
            create_histogram_panel(6, "Poster Generation Duration",
                "character_poster_generation_duration_seconds", {"h": 8, "w": 12, "x": 12, "y": 10}),
            create_histogram_panel(7, "Poster Token Consumption",
                "character_poster_token_consumed", {"h": 8, "w": 12, "x": 0, "y": 18}, unit="tokens"),
            create_timeseries_panel(8, "Poster Generation Errors",
                "sum(rate(character_poster_errors_total[$__rate_interval])) by (error_type)",
                "{{ error_type }}", {"h": 8, "w": 12, "x": 12, "y": 18})
        ]
    },
    "14-story-style-configuration": {
        "title": "Story Style Configuration Dashboard",
        "uid": "story-style-configuration",
        "tags": ["style", "configuration"],
        "panels": [
            create_stat_panel(1, "Total Style Configurations",
                "story_style_config_count", {"h": 4, "w": 8, "x": 0, "y": 0}),
            create_timeseries_panel(2, "Style Config Usage Rate",
                "sum(rate(story_style_config_usage_total[$__rate_interval])) by (style_id, usage_type)",
                "{{ style_id }}/{{ usage_type }}", {"h": 8, "w": 12, "x": 0, "y": 4}),
            create_timeseries_panel(3, "Style Config Count by Style Name",
                "story_style_config_by_style", "{{ style }}", {"h": 8, "w": 12, "x": 12, "y": 4}, unit="short")
        ]
    },
    "15-ai-generation-quality": {
        "title": "AI Generation Quality Dashboard",
        "uid": "ai-generation-quality",
        "tags": ["ai", "quality"],
        "panels": [
            create_stat_panel(1, "AI Generation Success Rate",
                "ai_generation_success_rate", {"h": 4, "w": 8, "x": 0, "y": 0}, thresholds=[
                    {"color": "red", "value": None},
                    {"color": "yellow", "value": 0.7},
                    {"color": "green", "value": 0.9}
                ]),
            create_timeseries_panel(2, "Average Token Consumption",
                "ai_generation_average_tokens", "{{ type }}/{{ provider }}", {"h": 8, "w": 12, "x": 0, "y": 4}, unit="tokens"),
            create_timeseries_panel(3, "Average Generation Duration",
                "ai_generation_average_duration_seconds", "{{ type }}/{{ provider }}", {"h": 8, "w": 12, "x": 12, "y": 4}),
            create_timeseries_panel(4, "AI Generation Retries",
                "sum(rate(ai_generation_retries_total[$__rate_interval])) by (type, provider, retry_count)",
                "{{ type }}/{{ provider }}/{{ retry_count }}", {"h": 8, "w": 24, "x": 0, "y": 12})
        ]
    },
    "16-storyboard-workflow-completion": {
        "title": "Storyboard Workflow Completion Dashboard",
        "uid": "storyboard-workflow-completion",
        "tags": ["storyboard", "workflow", "completion"],
        "panels": [
            create_timeseries_panel(1, "Workflow Completion Rate",
                "sum(rate(storyboard_workflow_completed_total[$__rate_interval])) by (workflow_status)",
                "{{ workflow_status }}", {"h": 8, "w": 12, "x": 0, "y": 0}),
            create_histogram_panel(2, "Workflow Duration",
                "storyboard_workflow_duration_seconds", {"h": 8, "w": 12, "x": 12, "y": 0}),
            create_timeseries_panel(3, "Workflow Abandoned",
                "sum(rate(storyboard_workflow_abandoned_total[$__rate_interval])) by (abandoned_at_step)",
                "{{ abandoned_at_step }}", {"h": 8, "w": 24, "x": 0, "y": 8})
        ]
    }
}

def generate_dashboard(filename, config):
    """Generate a dashboard JSON file"""
    dashboard = BASE_TEMPLATE.copy()
    dashboard["title"] = config["title"]
    dashboard["uid"] = config["uid"]
    dashboard["tags"] = config["tags"]
    dashboard["panels"] = config["panels"]
    
    # Write to file
    output_path = os.path.join("dashboards", filename)
    with open(output_path, "w", encoding="utf-8") as f:
        json.dump(dashboard, f, indent=2, ensure_ascii=False)
    print(f"Generated: {output_path}")

def main():
    """Generate all dashboards"""
    os.makedirs("dashboards", exist_ok=True)
    
    for filename, config in DASHBOARDS.items():
        generate_dashboard(f"{filename}.json", config)
    
    print(f"\nGenerated {len(DASHBOARDS)} dashboard files in dashboards/ directory")

if __name__ == "__main__":
    main()

