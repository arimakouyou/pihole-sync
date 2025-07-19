# Prometheus & Grafana Setup for Pi-hole Monitoring

This guide explains how to set up Prometheus and Grafana to monitor Pi-hole using the pihole-sync metrics.

## Prometheus Configuration

Add the following job to your `prometheus.yml`:

```yaml
global:
  scrape_interval: 30s

scrape_configs:
  - job_name: 'pihole-sync'
    static_configs:
      - targets: ['localhost:8080']
    scrape_interval: 30s
    metrics_path: /metrics
```

## Grafana Dashboard

### Import Dashboard

1. Open Grafana web interface
2. Go to "+" → "Import"
3. Upload the dashboard JSON or copy the configuration below

### Dashboard Configuration

```json
{
  "dashboard": {
    "id": null,
    "title": "Pi-hole Monitoring",
    "tags": ["pihole", "dns", "monitoring"],
    "timezone": "browser",
    "refresh": "30s",
    "panels": [
      {
        "id": 1,
        "title": "Pi-hole Status",
        "type": "stat",
        "targets": [
          {
            "expr": "pihole_status_enabled",
            "legendFormat": "Status"
          }
        ],
        "fieldConfig": {
          "defaults": {
            "mappings": [
              {
                "options": {
                  "0": {"text": "Disabled", "color": "red"},
                  "1": {"text": "Enabled", "color": "green"}
                },
                "type": "value"
              }
            ]
          }
        }
      },
      {
        "id": 2,
        "title": "Blocking Overview",
        "type": "stat",
        "targets": [
          {
            "expr": "pihole_ads_blocked_today_total",
            "legendFormat": "Ads Blocked Today"
          },
          {
            "expr": "pihole_ads_percentage_today",
            "legendFormat": "Block Percentage"
          },
          {
            "expr": "pihole_dns_queries_today_total",
            "legendFormat": "Total Queries"
          }
        ]
      },
      {
        "id": 3,
        "title": "DNS Queries Over Time",
        "type": "graph",
        "targets": [
          {
            "expr": "rate(pihole_dns_queries_today_total[5m]) * 300",
            "legendFormat": "Queries per 5min"
          },
          {
            "expr": "rate(pihole_ads_blocked_today_total[5m]) * 300",
            "legendFormat": "Blocked per 5min"
          }
        ],
        "yAxes": [
          {
            "label": "Queries",
            "min": 0
          }
        ]
      },
      {
        "id": 4,
        "title": "Query Types Distribution",
        "type": "piechart",
        "targets": [
          {
            "expr": "pihole_query_types_total",
            "legendFormat": "{{type}}"
          }
        ]
      },
      {
        "id": 5,
        "title": "Top Blocked Domains",
        "type": "table",
        "targets": [
          {
            "expr": "topk(10, pihole_top_blocked_domains_total)",
            "legendFormat": "{{domain}}"
          }
        ]
      },
      {
        "id": 6,
        "title": "Top Clients",
        "type": "table",
        "targets": [
          {
            "expr": "topk(10, pihole_top_clients_total)",
            "legendFormat": "{{client}}"
          }
        ]
      },
      {
        "id": 7,
        "title": "Upstream Servers",
        "type": "bargauge",
        "targets": [
          {
            "expr": "pihole_upstream_queries_total",
            "legendFormat": "{{upstream}}"
          }
        ]
      },
      {
        "id": 8,
        "title": "API Response Times",
        "type": "graph",
        "targets": [
          {
            "expr": "histogram_quantile(0.95, rate(pihole_api_response_time_seconds_bucket[5m]))",
            "legendFormat": "95th percentile"
          },
          {
            "expr": "histogram_quantile(0.50, rate(pihole_api_response_time_seconds_bucket[5m]))",
            "legendFormat": "50th percentile"
          }
        ]
      },
      {
        "id": 9,
        "title": "Cache Performance",
        "type": "stat",
        "targets": [
          {
            "expr": "pihole_queries_cached_total / (pihole_queries_cached_total + pihole_queries_forwarded_total) * 100",
            "legendFormat": "Cache Hit Rate %"
          }
        ]
      },
      {
        "id": 10,
        "title": "System Health",
        "type": "stat",
        "targets": [
          {
            "expr": "rate(pihole_api_errors_total[5m])",
            "legendFormat": "API Errors/sec"
          },
          {
            "expr": "time() - pihole_last_successful_collection_timestamp",
            "legendFormat": "Seconds since last collection"
          }
        ]
      }
    ]
  }
}
```

## Alert Rules

### Prometheus Alert Rules

Add to your `alert.rules.yml`:

```yaml
groups:
  - name: pihole
    rules:
      - alert: PiholeDown
        expr: pihole_status_enabled == 0
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: "Pi-hole is disabled"
          description: "Pi-hole blocking is currently disabled"

      - alert: PiholeHighBlockRate
        expr: pihole_ads_percentage_today > 80
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High Pi-hole block rate"
          description: "Pi-hole is blocking {{ $value }}% of queries"

      - alert: PiholeAPIErrors
        expr: rate(pihole_api_errors_total[5m]) > 0.1
        for: 2m
        labels:
          severity: warning
        annotations:
          summary: "Pi-hole API errors detected"
          description: "Pi-hole API is experiencing errors at {{ $value }} errors/sec"

      - alert: PiholeStaleMetrics
        expr: time() - pihole_last_successful_collection_timestamp > 300
        for: 1m
        labels:
          severity: warning
        annotations:
          summary: "Pi-hole metrics are stale"
          description: "Last successful metrics collection was {{ $value }} seconds ago"

      - alert: PiholeLowCacheHitRate
        expr: pihole_queries_cached_total / (pihole_queries_cached_total + pihole_queries_forwarded_total) * 100 < 30
        for: 10m
        labels:
          severity: warning
        annotations:
          summary: "Low Pi-hole cache hit rate"
          description: "Pi-hole cache hit rate is {{ $value }}%"
```

## Grafana Alerting

### Configure Notification Channels

1. Go to Alerting → Notification channels
2. Add your preferred notification method (Slack, email, etc.)

### Example Slack Notification

```json
{
  "channel": "#pihole-alerts",
  "title": "Pi-hole Alert",
  "text": "{{ range .Alerts }}{{ .Annotations.summary }}\n{{ .Annotations.description }}{{ end }}"
}
```

## Sample PromQL Queries

### Performance Queries

```promql
# Current blocking percentage
pihole_ads_percentage_today

# Queries per minute over last hour
rate(pihole_dns_queries_today_total[1h]) * 60

# Cache hit ratio
pihole_queries_cached_total / (pihole_queries_cached_total + pihole_queries_forwarded_total) * 100

# Top 5 blocked domains
topk(5, pihole_top_blocked_domains_total)

# Top 5 query types
topk(5, pihole_query_types_total)

# Upstream server distribution
pihole_upstream_queries_total

# API health over time
rate(pihole_api_errors_total[5m])

# Response time 95th percentile
histogram_quantile(0.95, rate(pihole_api_response_time_seconds_bucket[5m]))
```

### Capacity Planning

```promql
# Daily query growth rate
increase(pihole_dns_queries_today_total[24h])

# Client growth
pihole_unique_clients_total

# Domain blocklist size
pihole_domains_blocked_total
```

## Dashboard Variables

Add these variables for dynamic filtering:

1. **Upstream Server**: `label_values(pihole_upstream_queries_total, upstream)`
2. **Query Type**: `label_values(pihole_query_types_total, type)`
3. **Time Range**: Standard Grafana time picker

## Troubleshooting

### Common Issues

1. **No data in Grafana**:
   - Check Prometheus is scraping: `http://prometheus:9090/targets`
   - Verify metrics endpoint: `curl http://pihole-sync:8080/metrics`

2. **Missing metrics**:
   - Check configuration: `metrics.enabled: true`
   - Verify Pi-hole API accessibility
   - Check application logs

3. **High cardinality warnings**:
   - Reduce `top_items_limit` in configuration
   - Disable unnecessary metric collections

### Debug Commands

```bash
# Check if metrics are being collected
curl http://localhost:8080/metrics | grep pihole_

# Verify Prometheus can scrape
curl http://prometheus:9090/api/v1/targets

# Check specific metric
curl http://localhost:8080/metrics | grep pihole_ads_percentage_today
```
