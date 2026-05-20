#!/usr/bin/env python3
"""
Generate Grafana dashboard JSON files aligned with current Prometheus metrics.

Tier split:
  - System: HTTP traffic, errors, cache, active requests (dashboards 01-02)
  - Business: users, content, AI workflows, payments, etc. (dashboards 04-16)
"""

import json
import os

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
                "type": "dashboard",
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
                "current": {"selected": False, "text": "Prometheus", "value": "Prometheus"},
                "hide": 0,
                "label": "Datasource",
                "name": "datasource",
                "options": [],
                "query": "prometheus",
                "refresh": 1,
                "regex": "",
                "type": "datasource",
            }
        ]
    },
    "time": {"from": "now-6h", "to": "now"},
    "timepicker": {},
    "timezone": "",
    "title": "",
    "uid": "",
    "version": 1,
}


def create_timeseries_panel(id, title, expr, legend_format, grid_pos, unit="ops", thresholds=None):
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
                    "thresholdsStyle": {"mode": "off"},
                },
                "mappings": [],
                "thresholds": {"mode": "absolute", "steps": [{"color": "green", "value": None}]},
                "unit": unit,
            },
            "overrides": [],
        },
        "gridPos": grid_pos,
        "id": id,
        "options": {
            "legend": {"calcs": ["sum"], "displayMode": "table", "placement": "bottom", "showLegend": True},
            "tooltip": {"mode": "multi", "sort": "desc"},
        },
        "targets": [
            {"datasource": "${datasource}", "expr": expr, "legendFormat": legend_format, "refId": "A"}
        ],
        "title": title,
        "type": "timeseries",
    }
    if thresholds:
        panel["fieldConfig"]["defaults"]["thresholds"]["steps"] = thresholds
    return panel


def create_stat_panel(id, title, expr, grid_pos, thresholds=None):
    panel = {
        "datasource": "${datasource}",
        "fieldConfig": {
            "defaults": {
                "color": {"mode": "thresholds"},
                "mappings": [],
                "thresholds": {"mode": "absolute", "steps": [{"color": "green", "value": None}]},
                "unit": "short",
            },
            "overrides": [],
        },
        "gridPos": grid_pos,
        "id": id,
        "options": {
            "colorMode": "value",
            "graphMode": "area",
            "justifyMode": "auto",
            "orientation": "auto",
            "reduceOptions": {"calcs": ["lastNotNull"], "fields": "", "values": False},
            "textMode": "auto",
        },
        "pluginVersion": "9.0.0",
        "targets": [{"datasource": "${datasource}", "expr": expr, "refId": "A"}],
        "title": title,
        "type": "stat",
    }
    if thresholds:
        panel["fieldConfig"]["defaults"]["thresholds"]["steps"] = thresholds
    return panel


def create_histogram_panel(id, title, metric_name, grid_pos, unit="s", quantiles=None, by_labels=None):
    if quantiles is None:
        quantiles = ["0.50", "0.95", "0.99"]
    ref_ids = ["A", "B", "C"]
    by_clause = ""
    if by_labels:
        by_clause = ", " + ", ".join(by_labels)
    targets = []
    for i, q in enumerate(quantiles[: len(ref_ids)]):
        targets.append(
            {
                "datasource": "${datasource}",
                "expr": f"histogram_quantile({q}, sum(rate({metric_name}_bucket[$__rate_interval])) by (le{by_clause}))",
                "legendFormat": f"p{int(float(q) * 100)}",
                "refId": ref_ids[i],
            }
        )
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
                    "thresholdsStyle": {"mode": "off"},
                },
                "mappings": [],
                "thresholds": {
                    "mode": "absolute",
                    "steps": [
                        {"color": "green", "value": None},
                        {"color": "yellow", "value": 1},
                        {"color": "red", "value": 5},
                    ],
                },
                "unit": unit,
            },
            "overrides": [],
        },
        "gridPos": grid_pos,
        "id": id,
        "options": {
            "legend": {"calcs": ["mean", "p95", "p99"], "displayMode": "table", "placement": "bottom", "showLegend": True},
            "tooltip": {"mode": "multi", "sort": "desc"},
        },
        "targets": targets,
        "title": title,
        "type": "timeseries",
    }


def create_gauge_panel(id, title, expr, grid_pos, unit="percent"):
    return {
        "datasource": "${datasource}",
        "fieldConfig": {
            "defaults": {
                "color": {"mode": "thresholds"},
                "mappings": [],
                "max": 100 if unit == "percent" else None,
                "min": 0 if unit == "percent" else None,
                "thresholds": {
                    "mode": "absolute",
                    "steps": [
                        {"color": "red", "value": None},
                        {"color": "yellow", "value": 70},
                        {"color": "green", "value": 90},
                    ],
                },
                "unit": unit,
            },
            "overrides": [],
        },
        "gridPos": grid_pos,
        "id": id,
        "options": {
            "orientation": "auto",
            "reduceOptions": {"calcs": ["lastNotNull"], "fields": "", "values": False},
            "showThresholdLabels": False,
            "showThresholdMarkers": True,
        },
        "pluginVersion": "9.0.0",
        "targets": [{"datasource": "${datasource}", "expr": expr, "refId": "A"}],
        "title": title,
        "type": "gauge",
    }


def create_row_panel(id, title, grid_pos):
    return {
        "collapsed": False,
        "gridPos": grid_pos,
        "id": id,
        "panels": [],
        "title": title,
        "type": "row",
    }


DASHBOARDS = {
    "01-http-metrics": {
        "title": "HTTP Metrics Dashboard",
        "uid": "http-metrics",
        "tags": ["http", "system"],
        "refresh": "10s",
        "panels": [
            create_timeseries_panel(
                1,
                "HTTP Request Rate by Method",
                "sum(rate(http_requests_total[$__rate_interval])) by (method)",
                "{{ method }}",
                {"h": 8, "w": 12, "x": 0, "y": 0},
                unit="reqps",
            ),
            create_timeseries_panel(
                2,
                "HTTP Request Rate by Status",
                "sum(rate(http_requests_total[$__rate_interval])) by (status)",
                "{{ status }}",
                {"h": 8, "w": 12, "x": 12, "y": 0},
                unit="reqps",
            ),
            create_histogram_panel(
                3, "HTTP Request Duration", "http_request_duration_seconds", {"h": 8, "w": 12, "x": 0, "y": 8}
            ),
            create_stat_panel(4, "Active Requests", "http_active_requests", {"h": 4, "w": 6, "x": 12, "y": 8}),
            create_stat_panel(
                5,
                "Success Rate (2xx)",
                '100 * sum(rate(http_requests_total{status=~"2.."}[$__rate_interval])) / sum(rate(http_requests_total[$__rate_interval]))',
                {"h": 4, "w": 6, "x": 18, "y": 8},
                thresholds=[
                    {"color": "red", "value": None},
                    {"color": "yellow", "value": 90},
                    {"color": "green", "value": 99},
                ],
            ),
            create_timeseries_panel(
                6,
                "Top Paths by Request Rate",
                "topk(10, sum(rate(http_requests_total[$__rate_interval])) by (path))",
                "{{ path }}",
                {"h": 8, "w": 12, "x": 0, "y": 12},
                unit="reqps",
            ),
            create_timeseries_panel(
                7,
                "4xx / 5xx Error Rate",
                'sum(rate(http_requests_total{status=~"[45].."}[$__rate_interval])) by (status)',
                "{{ status }}",
                {"h": 8, "w": 12, "x": 12, "y": 12},
                unit="reqps",
            ),
            create_timeseries_panel(
                8,
                "API Error Codes (http_error_total)",
                "sum(rate(http_error_total[$__rate_interval])) by (error_code, method)",
                "{{ error_code }}/{{ method }}",
                {"h": 8, "w": 12, "x": 0, "y": 20},
            ),
            create_histogram_panel(
                9, "API Error Duration", "http_error_duration_seconds", {"h": 8, "w": 12, "x": 12, "y": 20}
            ),
            create_timeseries_panel(
                10,
                "Request Size (bytes/s)",
                "sum(rate(http_request_size_bytes_sum[$__rate_interval]))",
                "request",
                {"h": 8, "w": 12, "x": 0, "y": 28},
                unit="Bps",
            ),
            create_timeseries_panel(
                11,
                "Response Size (bytes/s)",
                "sum(rate(http_response_size_bytes_sum[$__rate_interval]))",
                "response",
                {"h": 8, "w": 12, "x": 12, "y": 28},
                unit="Bps",
            ),
        ],
    },
    "02-application-metrics": {
        "title": "Application Metrics Dashboard",
        "uid": "application-metrics",
        "tags": ["application", "system"],
        "refresh": "10s",
        "panels": [
            create_stat_panel(1, "Active Requests", "http_active_requests", {"h": 4, "w": 6, "x": 0, "y": 0}),
            create_timeseries_panel(
                2,
                "HTTP API Errors by Code",
                "sum(rate(http_error_total[$__rate_interval])) by (error_code)",
                "{{ error_code }}",
                {"h": 8, "w": 12, "x": 0, "y": 4},
            ),
            create_histogram_panel(
                3, "HTTP API Error Duration", "http_error_duration_seconds", {"h": 8, "w": 12, "x": 12, "y": 4}
            ),
            create_timeseries_panel(
                4,
                "Cache Hits / Misses",
                "sum(rate(cache_hits_total[$__rate_interval])) by (cache)",
                "hit {{ cache }}",
                {"h": 8, "w": 12, "x": 0, "y": 12},
            ),
            create_timeseries_panel(
                5,
                "Cache Misses",
                "sum(rate(cache_misses_total[$__rate_interval])) by (cache)",
                "miss {{ cache }}",
                {"h": 8, "w": 12, "x": 12, "y": 12},
            ),
            create_gauge_panel(
                6,
                "Cache Hit Rate",
                "100 * sum(rate(cache_hits_total[$__rate_interval])) / (sum(rate(cache_hits_total[$__rate_interval])) + sum(rate(cache_misses_total[$__rate_interval])))",
                {"h": 8, "w": 12, "x": 0, "y": 20},
            ),
        ],
    },
    "03-business-metrics": {
        "title": "Business Metrics Overview",
        "uid": "business-metrics",
        "tags": ["business", "overview"],
        "panels": [
            create_stat_panel(1, "Total Users", "user_count", {"h": 4, "w": 6, "x": 0, "y": 0}),
            create_stat_panel(2, "Total Stories", "story_count", {"h": 4, "w": 6, "x": 6, "y": 0}),
            create_stat_panel(3, "Total Storyboards", "storyboard_count", {"h": 4, "w": 6, "x": 12, "y": 0}),
            create_stat_panel(4, "Daily Active Users", "daily_active_users", {"h": 4, "w": 6, "x": 18, "y": 0}),
            create_timeseries_panel(
                5,
                "User Registrations",
                "sum(rate(user_registrations_total[$__rate_interval])) by (source)",
                "{{ source }}",
                {"h": 8, "w": 12, "x": 0, "y": 4},
            ),
            create_timeseries_panel(
                6,
                "User Logins",
                "sum(rate(user_logins_total[$__rate_interval])) by (method)",
                "{{ method }}",
                {"h": 8, "w": 12, "x": 12, "y": 4},
            ),
            create_timeseries_panel(
                7,
                "Story Creations",
                "sum(rate(story_creations_total[$__rate_interval])) by (type)",
                "{{ type }}",
                {"h": 8, "w": 12, "x": 0, "y": 12},
            ),
            create_timeseries_panel(
                8,
                "AI Generations",
                "sum(rate(ai_generations_total[$__rate_interval])) by (provider, type)",
                "{{ provider }}/{{ type }}",
                {"h": 8, "w": 12, "x": 12, "y": 12},
            ),
            create_stat_panel(9, "Weekly Active Users", "weekly_active_users", {"h": 4, "w": 6, "x": 0, "y": 20}),
            create_stat_panel(10, "Monthly Active Users", "monthly_active_users", {"h": 4, "w": 6, "x": 6, "y": 20}),
            create_timeseries_panel(
                11,
                "User Growth Rate",
                "user_growth_rate",
                "{{ type }}",
                {"h": 8, "w": 12, "x": 12, "y": 20},
                unit="percent",
            ),
        ],
    },
    "10-storyboard-generation-workflow": {
        "title": "Storyboard Generation Workflow Dashboard",
        "uid": "storyboard-generation-workflow",
        "tags": ["storyboard", "business", "workflow"],
        "panels": [
            create_row_panel(1, "Content Generation", {"h": 1, "w": 24, "x": 0, "y": 0}),
            create_timeseries_panel(
                2,
                "Content Generation Rate",
                "sum(rate(storyboard_content_generations_total[$__rate_interval])) by (status)",
                "{{ status }}",
                {"h": 8, "w": 12, "x": 0, "y": 1},
            ),
            create_histogram_panel(
                3,
                "Content Generation Duration",
                "storyboard_content_generation_duration_seconds",
                {"h": 8, "w": 12, "x": 12, "y": 1},
            ),
            create_row_panel(4, "Scene Generation", {"h": 1, "w": 24, "x": 0, "y": 9}),
            create_timeseries_panel(
                5,
                "Scene Generation Rate",
                "sum(rate(storyboard_scene_generations_total[$__rate_interval])) by (status)",
                "{{ status }}",
                {"h": 8, "w": 12, "x": 0, "y": 10},
            ),
            create_histogram_panel(
                6,
                "Scene Generation Duration",
                "storyboard_scene_generation_duration_seconds",
                {"h": 8, "w": 12, "x": 12, "y": 10},
            ),
            create_row_panel(7, "Image / Video Generation", {"h": 1, "w": 24, "x": 0, "y": 18}),
            create_timeseries_panel(
                8,
                "Image Generation Rate",
                "sum(rate(storyboard_image_generations_total[$__rate_interval])) by (status, scene_type)",
                "{{ status }}/{{ scene_type }}",
                {"h": 8, "w": 12, "x": 0, "y": 19},
            ),
            create_histogram_panel(
                9,
                "Image Generation Duration",
                "storyboard_image_generation_duration_seconds",
                {"h": 8, "w": 12, "x": 12, "y": 19},
                by_labels=["scene_type"],
            ),
            create_timeseries_panel(
                10,
                "Video Generation Rate",
                "sum(rate(storyboard_video_generations_total[$__rate_interval])) by (status, is_subdivided)",
                "{{ status }}/{{ is_subdivided }}",
                {"h": 8, "w": 12, "x": 0, "y": 27},
            ),
            create_histogram_panel(
                11,
                "Video Generation Duration",
                "storyboard_video_generation_duration_seconds",
                {"h": 8, "w": 12, "x": 12, "y": 27},
                by_labels=["is_subdivided"],
            ),
        ],
    },
    "04-content-metrics": {
        "title": "Content Metrics Dashboard",
        "uid": "content-metrics",
        "tags": ["content", "business"],
        "panels": [
            create_stat_panel(1, "Total Storyboards", "storyboard_count", {"h": 4, "w": 8, "x": 0, "y": 0}),
            create_stat_panel(2, "Total Stories", "story_count", {"h": 4, "w": 8, "x": 8, "y": 0}),
            create_stat_panel(3, "Total Users", "user_count", {"h": 4, "w": 8, "x": 16, "y": 0}),
            create_timeseries_panel(
                4,
                "Story Creations Rate",
                "sum(rate(story_creations_total[$__rate_interval])) by (type)",
                "{{ type }}",
                {"h": 8, "w": 12, "x": 0, "y": 4},
            ),
            create_histogram_panel(
                5, "Storyboard Scene Count", "storyboard_scene_count", {"h": 8, "w": 12, "x": 12, "y": 4}, unit="short"
            ),
            create_histogram_panel(
                6, "Storyboard Token Consumption", "storyboard_token_consumed", {"h": 8, "w": 12, "x": 0, "y": 12}, unit="short"
            ),
            create_histogram_panel(
                7, "User Token Consumption", "user_token_consumed", {"h": 8, "w": 12, "x": 12, "y": 12}, unit="short"
            ),
            create_histogram_panel(
                8, "Story Participant Count", "story_participant_count", {"h": 8, "w": 12, "x": 0, "y": 20}, unit="short"
            ),
            create_histogram_panel(
                9, "Storyboard Fork Count", "storyboard_child_count", {"h": 8, "w": 12, "x": 12, "y": 20}, unit="short"
            ),
        ],
    },
    "05-user-activity-metrics": {
        "title": "User Activity Metrics Dashboard",
        "uid": "user-activity-metrics",
        "tags": ["user", "business"],
        "panels": [
            create_stat_panel(1, "Daily Active Users", "daily_active_users", {"h": 4, "w": 8, "x": 0, "y": 0}),
            create_stat_panel(2, "Weekly Active Users", "weekly_active_users", {"h": 4, "w": 8, "x": 8, "y": 0}),
            create_stat_panel(3, "Monthly Active Users", "monthly_active_users", {"h": 4, "w": 8, "x": 16, "y": 0}),
            create_timeseries_panel(
                4,
                "User Registrations",
                "sum(rate(user_registrations_total[$__rate_interval])) by (source)",
                "{{ source }}",
                {"h": 8, "w": 12, "x": 0, "y": 4},
            ),
            create_timeseries_panel(
                5,
                "User Logins",
                "sum(rate(user_logins_total[$__rate_interval])) by (method)",
                "{{ method }}",
                {"h": 8, "w": 12, "x": 12, "y": 4},
            ),
            create_timeseries_panel(
                6,
                "User Growth Rate",
                "user_growth_rate",
                "{{ type }}",
                {"h": 8, "w": 24, "x": 0, "y": 12},
                unit="percent",
            ),
        ],
    },
    "06-compliance-metrics": {
        "title": "Compliance Metrics Dashboard",
        "uid": "compliance-metrics",
        "tags": ["compliance", "business"],
        "panels": [
            create_timeseries_panel(
                1,
                "Compliance Checks Rate",
                "sum(rate(compliance_checks_total[$__rate_interval]))",
                "Total",
                {"h": 8, "w": 12, "x": 0, "y": 0},
            ),
            create_timeseries_panel(
                2,
                "Compliance Check Results",
                "sum(rate(compliance_check_results_total[$__rate_interval])) by (status)",
                "{{ status }}",
                {"h": 8, "w": 12, "x": 12, "y": 0},
            ),
            create_timeseries_panel(
                3,
                "Compliance Violations by Type",
                "sum(rate(compliance_violations_by_type_total[$__rate_interval])) by (violation_type)",
                "{{ violation_type }}",
                {"h": 8, "w": 12, "x": 0, "y": 8},
            ),
            create_histogram_panel(
                4, "Compliance Check Duration", "compliance_check_duration_seconds", {"h": 8, "w": 12, "x": 12, "y": 8}
            ),
        ],
    },
    "07-payment-metrics": {
        "title": "Payment Metrics Dashboard",
        "uid": "payment-metrics",
        "tags": ["payment", "business"],
        "panels": [
            create_timeseries_panel(
                1,
                "Payment Rate by Provider",
                "sum(rate(payment_total[$__rate_interval])) by (provider, type, status)",
                "{{ provider }}/{{ type }}/{{ status }}",
                {"h": 8, "w": 12, "x": 0, "y": 0},
            ),
            create_timeseries_panel(
                2,
                "Payment Refunds",
                "sum(rate(payment_refunds_total[$__rate_interval])) by (provider, reason)",
                "{{ provider }}/{{ reason }}",
                {"h": 8, "w": 12, "x": 12, "y": 0},
            ),
            create_stat_panel(3, "Active Subscriptions", "sum(payment_subscriptions_active)", {"h": 4, "w": 8, "x": 0, "y": 8}),
            create_timeseries_panel(
                4,
                "Payment Verification Rate",
                "sum(rate(payment_verify_total[$__rate_interval])) by (provider, status)",
                "{{ provider }}/{{ status }}",
                {"h": 8, "w": 12, "x": 0, "y": 12},
            ),
            create_histogram_panel(
                5, "Payment Verification Duration", "payment_verify_duration_seconds", {"h": 8, "w": 12, "x": 12, "y": 12}
            ),
        ],
    },
    "08-oauth-metrics": {
        "title": "OAuth Metrics Dashboard",
        "uid": "oauth-metrics",
        "tags": ["oauth", "business"],
        "panels": [
            create_timeseries_panel(
                1,
                "OAuth Login Rate",
                "sum(rate(oauth_login_total[$__rate_interval])) by (provider, status)",
                "{{ provider }}/{{ status }}",
                {"h": 8, "w": 12, "x": 0, "y": 0},
            ),
            create_histogram_panel(
                2, "OAuth Login Duration", "oauth_login_duration_seconds", {"h": 8, "w": 12, "x": 12, "y": 0}
            ),
            create_timeseries_panel(
                3,
                "OAuth Login Errors",
                "sum(rate(oauth_login_errors_total[$__rate_interval])) by (provider, error_type)",
                "{{ provider }}/{{ error_type }}",
                {"h": 8, "w": 24, "x": 0, "y": 8},
            ),
        ],
    },
    "09-notification-metrics": {
        "title": "Notification Metrics Dashboard",
        "uid": "notification-metrics",
        "tags": ["notification", "business"],
        "panels": [
            create_timeseries_panel(
                1,
                "Notifications Sent Rate",
                "sum(rate(notifications_sent_total[$__rate_interval])) by (type, channel, status)",
                "{{ type }}/{{ channel }}/{{ status }}",
                {"h": 8, "w": 12, "x": 0, "y": 0},
            ),
            create_histogram_panel(
                2, "Notification Delivery Duration", "notification_delivery_duration_seconds", {"h": 8, "w": 12, "x": 12, "y": 0}
            ),
            create_timeseries_panel(
                3,
                "Notification Errors",
                "sum(rate(notification_errors_total[$__rate_interval])) by (type, channel, error_type)",
                "{{ type }}/{{ channel }}/{{ error_type }}",
                {"h": 8, "w": 12, "x": 0, "y": 8},
            ),
            create_timeseries_panel(
                4,
                "Notifications by Category",
                "sum(rate(notifications_by_category_total[$__rate_interval])) by (category)",
                "{{ category }}",
                {"h": 8, "w": 12, "x": 12, "y": 8},
            ),
        ],
    },
    "11-image-generation-detailed": {
        "title": "Image Generation Detailed Dashboard",
        "uid": "image-generation-detailed",
        "tags": ["image", "business"],
        "panels": [
            create_timeseries_panel(
                1,
                "Image Generation with Characters",
                "sum(rate(image_generation_with_characters_total[$__rate_interval])) by (status, character_count)",
                "{{ status }}/{{ character_count }}",
                {"h": 8, "w": 12, "x": 0, "y": 0},
            ),
            create_timeseries_panel(
                2,
                "Image Generation with Style",
                "sum(rate(image_generation_with_style_total[$__rate_interval])) by (status, has_style)",
                "{{ status }}/{{ has_style }}",
                {"h": 8, "w": 12, "x": 12, "y": 0},
            ),
            create_histogram_panel(
                3, "Image Generation Token Consumption", "image_generation_token_consumed", {"h": 8, "w": 12, "x": 0, "y": 8}, unit="short"
            ),
            create_timeseries_panel(
                4,
                "Prompt Details Usage",
                "sum(rate(image_generation_prompt_details_used_total[$__rate_interval])) by (has_prompt_details)",
                "{{ has_prompt_details }}",
                {"h": 8, "w": 12, "x": 12, "y": 8},
            ),
            create_timeseries_panel(
                5,
                "Image Generation Errors",
                "sum(rate(image_generation_errors_total[$__rate_interval])) by (error_type)",
                "{{ error_type }}",
                {"h": 8, "w": 12, "x": 0, "y": 16},
            ),
        ],
    },
    "12-video-generation-detailed": {
        "title": "Video Generation Detailed Dashboard",
        "uid": "video-generation-detailed",
        "tags": ["video", "business"],
        "panels": [
            create_timeseries_panel(
                1,
                "Video Generation Subdivision",
                "sum(rate(video_generation_subdivided_total[$__rate_interval])) by (is_subdivided, status)",
                "{{ is_subdivided }}/{{ status }}",
                {"h": 8, "w": 12, "x": 0, "y": 0},
            ),
            create_histogram_panel(
                2, "Video Segment Count", "video_generation_segment_count", {"h": 8, "w": 12, "x": 12, "y": 0}, unit="short"
            ),
            create_histogram_panel(
                3, "Video Token Consumption", "video_generation_token_consumed", {"h": 8, "w": 12, "x": 0, "y": 8}, unit="short"
            ),
            create_timeseries_panel(
                4,
                "Video Generation Errors",
                "sum(rate(video_generation_errors_total[$__rate_interval])) by (error_type)",
                "{{ error_type }}",
                {"h": 8, "w": 12, "x": 12, "y": 8},
            ),
        ],
    },
    "14-story-style-configuration": {
        "title": "Story Style Configuration Dashboard",
        "uid": "story-style-configuration",
        "tags": ["style", "business"],
        "panels": [
            create_stat_panel(1, "Total Style Configurations", "story_style_config_count", {"h": 4, "w": 8, "x": 0, "y": 0}),
            create_timeseries_panel(
                2,
                "Style Config Usage Rate",
                "sum(rate(story_style_config_usage_total[$__rate_interval])) by (usage_type)",
                "{{ usage_type }}",
                {"h": 8, "w": 12, "x": 0, "y": 4},
            ),
            create_timeseries_panel(
                3,
                "Style Config Count by Style Name",
                "story_style_config_by_style",
                "{{ style }}",
                {"h": 8, "w": 12, "x": 12, "y": 4},
                unit="short",
            ),
        ],
    },
    "15-ai-generation-retries": {
        "title": "AI Generation Retries Dashboard",
        "uid": "ai-generation-retries",
        "tags": ["ai", "business"],
        "panels": [
            create_timeseries_panel(
                1,
                "AI Generations Rate",
                "sum(rate(ai_generations_total[$__rate_interval])) by (provider, type)",
                "{{ provider }}/{{ type }}",
                {"h": 8, "w": 12, "x": 0, "y": 0},
            ),
            create_timeseries_panel(
                2,
                "AI Generation Retries",
                "sum(rate(ai_generation_retries_total[$__rate_interval])) by (type, provider, retry_count)",
                "{{ type }}/{{ provider }}/{{ retry_count }}",
                {"h": 8, "w": 12, "x": 12, "y": 0},
            ),
        ],
    },
    "16-storyboard-workflow-completion": {
        "title": "Storyboard Workflow Completion Dashboard",
        "uid": "storyboard-workflow-completion",
        "tags": ["storyboard", "business"],
        "panels": [
            create_timeseries_panel(
                1,
                "Workflow Completion Rate",
                "sum(rate(storyboard_workflow_completed_total[$__rate_interval])) by (workflow_status)",
                "{{ workflow_status }}",
                {"h": 8, "w": 12, "x": 0, "y": 0},
            ),
            create_histogram_panel(
                2, "Workflow Duration (published only)", "storyboard_workflow_duration_seconds", {"h": 8, "w": 12, "x": 12, "y": 0}
            ),
            create_timeseries_panel(
                3,
                "Content Generation",
                "sum(rate(storyboard_content_generations_total[$__rate_interval])) by (status)",
                "{{ status }}",
                {"h": 8, "w": 8, "x": 0, "y": 8},
            ),
            create_timeseries_panel(
                4,
                "Image Generation",
                "sum(rate(storyboard_image_generations_total[$__rate_interval])) by (status, scene_type)",
                "{{ status }}/{{ scene_type }}",
                {"h": 8, "w": 8, "x": 8, "y": 8},
            ),
            create_timeseries_panel(
                5,
                "Video Generation",
                "sum(rate(storyboard_video_generations_total[$__rate_interval])) by (status, is_subdivided)",
                "{{ status }}/{{ is_subdivided }}",
                {"h": 8, "w": 8, "x": 16, "y": 8},
            ),
        ],
    },
}


def generate_dashboard(filename, config):
    dashboard = BASE_TEMPLATE.copy()
    dashboard["title"] = config["title"]
    dashboard["uid"] = config["uid"]
    dashboard["tags"] = config["tags"]
    dashboard["panels"] = config["panels"]
    if "refresh" in config:
        dashboard["refresh"] = config["refresh"]
    output_path = os.path.join("dashboards", filename)
    with open(output_path, "w", encoding="utf-8") as f:
        json.dump(dashboard, f, indent=2, ensure_ascii=False)
    print(f"Generated: {output_path}")


def main():
    os.makedirs("dashboards", exist_ok=True)
    for filename, config in DASHBOARDS.items():
        generate_dashboard(f"{filename}.json", config)
    # Remove dashboards for deleted metric families
    for obsolete in (
        "13-character-poster-generation.json",
        "15-ai-generation-quality.json",
    ):
        path = os.path.join("dashboards", obsolete)
        if os.path.exists(path):
            os.remove(path)
            print(f"Removed obsolete: {path}")
    print(f"\nGenerated {len(DASHBOARDS)} dashboard files in dashboards/")


if __name__ == "__main__":
    main()
