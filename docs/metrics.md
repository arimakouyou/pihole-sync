# Pi-hole Metrics Integration

This document describes the Pi-hole metrics integration feature that collects comprehensive statistics from Pi-hole FTL API and exposes them as Prometheus metrics.

## Overview

The Pi-hole metrics integration extends the basic `pihole-sync` application with detailed monitoring capabilities by collecting statistics from Pi-hole's FTL (Fast-Lightweight Teleporter) API and exposing them in Prometheus format.

## Configuration

Add the following configuration to your `config.yaml`:

```yaml
metrics:
  enabled: true                    # Enable/disable metrics collection
  collection_interval: "30s"      # How often to collect metrics
  enable_top_domains: true        # Collect top domains statistics
  enable_top_clients: true        # Collect top clients statistics  
  enable_upstreams: true          # Collect upstream server statistics
  enable_cache_metrics: true      # Collect DNS cache metrics
  top_items_limit: 10             # Limit for top domains/clients
```

### Configuration Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `enabled` | bool | `true` | Enable or disable metrics collection |
| `collection_interval` | duration | `30s` | Interval between metric collections |
| `enable_top_domains` | bool | `true` | Collect top permitted/blocked domains |
| `enable_top_clients` | bool | `true` | Collect top client statistics |
| `enable_upstreams` | bool | `true` | Collect upstream server statistics |
| `enable_cache_metrics` | bool | `true` | Collect DNS cache metrics |
| `top_items_limit` | int | `10` | Maximum number of top items to track |

## Available Metrics

### Core Statistics

These metrics are collected from the `/stats/summary` endpoint:

| Metric Name | Type | Description |
|-------------|------|-------------|
| `pihole_domains_blocked_total` | Gauge | Number of domains being blocked |
| `pihole_dns_queries_today_total` | Gauge | Number of DNS queries today |
| `pihole_ads_blocked_today_total` | Gauge | Number of ads blocked today |
| `pihole_ads_percentage_today` | Gauge | Percentage of ads blocked today |
| `pihole_unique_domains_total` | Gauge | Number of unique domains queried |
| `pihole_queries_forwarded_total` | Gauge | Number of queries forwarded to upstream |
| `pihole_queries_cached_total` | Gauge | Number of queries answered from cache |
| `pihole_clients_ever_seen_total` | Gauge | Total number of clients ever seen |
| `pihole_unique_clients_total` | Gauge | Number of unique clients today |
| `pihole_dns_queries_all_types_total` | Gauge | Total DNS queries of all types |
| `pihole_reply_unknown_total` | Gauge | Number of unknown reply types |
| `pihole_reply_nodata_total` | Gauge | Number of NODATA replies |
| `pihole_reply_nxdomain_total` | Gauge | Number of NXDOMAIN replies |
| `pihole_reply_cname_total` | Gauge | Number of CNAME replies |
| `pihole_reply_ip_total` | Gauge | Number of IP address replies |
| `pihole_privacy_level` | Gauge | Current privacy level setting |
| `pihole_status_enabled` | Gauge | Pi-hole status (1=enabled, 0=disabled) |

### Query Types

Collected from `/stats/query_types` endpoint:

| Metric Name | Type | Labels | Description |
|-------------|------|--------|-------------|
| `pihole_query_types_total` | Gauge | `type` | Query count by DNS record type |

Common query types include: `A`, `AAAA`, `ANY`, `SRV`, `SOA`, `PTR`, `TXT`, `NAPTR`, `MX`, `DS`, `RRSIG`, `DNSKEY`, `NS`, `OTHER`.

### Upstream Servers

Collected from `/stats/upstreams` endpoint:

| Metric Name | Type | Labels | Description |
|-------------|------|--------|-------------|
| `pihole_upstream_queries_total` | Gauge | `upstream` | Query percentage by upstream server |

### Top Domains

Collected from `/stats/top_domains` endpoint:

| Metric Name | Type | Labels | Description |
|-------------|------|--------|-------------|
| `pihole_top_permitted_domains_total` | Gauge | `domain` | Top permitted domains by query count |
| `pihole_top_blocked_domains_total` | Gauge | `domain` | Top blocked domains by query count |

### Top Clients

Collected from `/stats/top_clients` endpoint:

| Metric Name | Type | Labels | Description |
|-------------|------|--------|-------------|
| `pihole_top_clients_total` | Gauge | `client` | Top clients by query count |

### System Metrics

Additional monitoring metrics:

| Metric Name | Type | Labels | Description |
|-------------|------|--------|-------------|
| `pihole_api_errors_total` | Counter | `endpoint` | API errors by endpoint |
| `pihole_api_response_time_seconds` | Histogram | `endpoint` | API response time by endpoint |
| `pihole_last_successful_collection_timestamp` | Gauge | | Timestamp of last successful collection |

## Prometheus Queries

### Example PromQL Queries

**Current blocking percentage:**
```promql
pihole_ads_percentage_today
```

**Queries per hour:**
```promql
rate(pihole_dns_queries_today_total[1h]) * 3600
```

**Top 5 blocked domains:**
```promql
topk(5, pihole_top_blocked_domains_total)
```

**Query type distribution:**
```promql
sum by (type) (pihole_query_types_total)
```

**Upstream server usage:**
```promql
pihole_upstream_queries_total
```

**API health:**
```promql
rate(pihole_api_errors_total[5m])
```

## Grafana Dashboard

### Recommended Panels

1. **Overview Panel**
   - Current blocking percentage
   - Total queries today
   - Ads blocked today
   - Unique clients

2. **Query Analysis**
   - Query types distribution (pie chart)
   - Queries over time (time series)
   - Cache hit ratio

3. **Top Lists**
   - Top blocked domains (table)
   - Top permitted domains (table)
   - Top clients (table)

4. **Upstream Monitoring**
   - Upstream server distribution
   - Upstream response times

5. **System Health**
   - API response times
   - Error rates
   - Last successful collection

### Sample Dashboard JSON

A complete Grafana dashboard configuration will be provided in `docs/grafana-dashboard.json`.

## Monitoring and Alerting

### Recommended Alerts

**Pi-hole Down:**
```promql
pihole_status_enabled == 0
```

**High Block Rate:**
```promql
pihole_ads_percentage_today > 50
```

**API Errors:**
```promql
rate(pihole_api_errors_total[5m]) > 0.1
```

**Stale Metrics:**
```promql
time() - pihole_last_successful_collection_timestamp > 300
```

## Troubleshooting

### Common Issues

1. **No metrics appearing:**
   - Check that `metrics.enabled` is `true` in config
   - Verify Pi-hole API is accessible
   - Check application logs for authentication errors

2. **Stale metrics:**
   - Check `pihole_last_successful_collection_timestamp`
   - Verify network connectivity to Pi-hole
   - Check for API authentication issues

3. **High memory usage:**
   - Reduce `top_items_limit` in configuration
   - Disable unnecessary metric collections

### Debug Commands

Check metrics endpoint:
```bash
curl http://localhost:8080/metrics | grep pihole_
```

View collection logs:
```bash
# Logs will show [METRICS] prefix for metrics-related messages
```

## Performance Considerations

- Default collection interval is 30 seconds
- Each collection makes 3-5 API calls to Pi-hole
- Memory usage scales with `top_items_limit`
- Consider Pi-hole load when setting collection interval

## API Compatibility

This implementation is compatible with:
- Pi-hole FTL v5.0+
- Pi-hole Web Interface v5.0+

The metrics collection uses the following Pi-hole API endpoints:
- `/stats/summary`
- `/stats/query_types`
- `/stats/upstreams`
- `/stats/top_domains`
- `/stats/top_clients`
