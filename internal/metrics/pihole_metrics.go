package metrics

import (
	"time"

	"github.com/arimakouyou/pihole-sync/internal/types"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Prometheus metrics
var (
	// Core Statistics from /stats/summary with instance labels
	PiholeDomainsBlocked = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pihole_domains_blocked_total",
		Help: "Number of domains being blocked by Pi-hole",
	}, []string{"instance", "role"})

	PiholeDNSQueriesToday = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pihole_dns_queries_today_total",
		Help: "Number of DNS queries today",
	}, []string{"instance", "role"})

	PiholeAdsBlockedToday = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pihole_ads_blocked_today_total",
		Help: "Number of ads blocked today",
	}, []string{"instance", "role"})

	PiholeAdsPercentageToday = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pihole_ads_percentage_today",
		Help: "Percentage of ads blocked today",
	}, []string{"instance", "role"})

	PiholeUniqueDomains = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pihole_unique_domains_total",
		Help: "Number of unique domains",
	}, []string{"instance", "role"})

	PiholeQueriesForwarded = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pihole_queries_forwarded_total",
		Help: "Number of queries forwarded to upstream servers",
	}, []string{"instance", "role"})

	PiholeQueriesCached = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pihole_queries_cached_total",
		Help: "Number of queries answered from cache",
	}, []string{"instance", "role"})

	PiholeClientsEverSeen = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pihole_clients_ever_seen_total",
		Help: "Number of clients ever seen",
	}, []string{"instance", "role"})

	PiholeUniqueClients = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pihole_unique_clients_total",
		Help: "Number of unique clients",
	}, []string{"instance", "role"})

	PiholeDNSQueriesAllTypes = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pihole_dns_queries_all_types_total",
		Help: "Total number of DNS queries of all types",
	}, []string{"instance", "role"})

	PiholeReplyUnknown = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pihole_reply_unknown_total",
		Help: "Number of unknown reply types",
	}, []string{"instance", "role"})

	PiholeReplyNodata = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pihole_reply_nodata_total",
		Help: "Number of NODATA replies",
	}, []string{"instance", "role"})

	PiholeReplyNxdomain = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pihole_reply_nxdomain_total",
		Help: "Number of NXDOMAIN replies",
	}, []string{"instance", "role"})

	PiholeReplyCname = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pihole_reply_cname_total",
		Help: "Number of CNAME replies",
	}, []string{"instance", "role"})

	PiholeReplyIP = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pihole_reply_ip_total",
		Help: "Number of IP address replies",
	}, []string{"instance", "role"})

	PiholePrivacyLevel = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pihole_privacy_level",
		Help: "Current privacy level setting",
	}, []string{"instance", "role"})

	PiholeStatusEnabled = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pihole_status_enabled",
		Help: "Pi-hole status (1=enabled, 0=disabled)",
	}, []string{"instance", "role"})

	// Query Types Metrics
	PiholeQueryTypes = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pihole_query_types_total",
		Help: "Number of queries by DNS record type",
	}, []string{"instance", "role", "type"})

	// Upstream Servers Metrics
	PiholeUpstreamQueries = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pihole_upstream_queries_total",
		Help: "Number of queries sent to upstream servers",
	}, []string{"instance", "role", "upstream"})

	// Top Domains Metrics
	PiholeTopPermittedDomains = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pihole_top_permitted_domains_total",
		Help: "Top permitted domains by query count",
	}, []string{"instance", "role", "domain"})

	PiholeTopBlockedDomains = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pihole_top_blocked_domains_total",
		Help: "Top blocked domains by query count",
	}, []string{"instance", "role", "domain"})

	// Top Clients Metrics
	PiholeTopClients = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pihole_top_clients_total",
		Help: "Top clients by query count",
	}, []string{"instance", "role", "client"})

	// API Error Metrics
	PiholeAPIErrors = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "pihole_api_errors_total",
		Help: "Number of API errors by endpoint",
	}, []string{"instance", "role", "endpoint"})

	PiholeAPIResponseTime = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "pihole_api_response_time_seconds",
		Help:    "API response time by endpoint",
		Buckets: prometheus.DefBuckets,
	}, []string{"instance", "role", "endpoint"})

	PiholeLastSuccessfulCollection = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pihole_last_successful_collection_timestamp",
		Help: "Timestamp of the last successful metrics collection",
	}, []string{"instance", "role"})
)

// UpdateSummaryStats updates Prometheus metrics with summary statistics
func UpdateSummaryStats(stats *types.SummaryStats, instance, role string) {
	PiholeDomainsBlocked.WithLabelValues(instance, role).Set(float64(stats.DomainsBlocked))
	PiholeDNSQueriesToday.WithLabelValues(instance, role).Set(float64(stats.DNSQueriesToday))
	PiholeAdsBlockedToday.WithLabelValues(instance, role).Set(float64(stats.AdsBlockedToday))
	PiholeAdsPercentageToday.WithLabelValues(instance, role).Set(stats.AdsPercentageToday)
	PiholeUniqueDomains.WithLabelValues(instance, role).Set(float64(stats.UniqueDomains))
	PiholeQueriesForwarded.WithLabelValues(instance, role).Set(float64(stats.QueriesForwarded))
	PiholeQueriesCached.WithLabelValues(instance, role).Set(float64(stats.QueriesCached))
	PiholeClientsEverSeen.WithLabelValues(instance, role).Set(float64(stats.ClientsEverSeen))
	PiholeUniqueClients.WithLabelValues(instance, role).Set(float64(stats.UniqueClients))
	PiholeDNSQueriesAllTypes.WithLabelValues(instance, role).Set(float64(stats.DNSQueriesAllTypes))
	PiholeReplyUnknown.WithLabelValues(instance, role).Set(float64(stats.ReplyUnknown))
	PiholeReplyNodata.WithLabelValues(instance, role).Set(float64(stats.ReplyNodata))
	PiholeReplyNxdomain.WithLabelValues(instance, role).Set(float64(stats.ReplyNxdomain))
	PiholeReplyCname.WithLabelValues(instance, role).Set(float64(stats.ReplyCname))
	PiholeReplyIP.WithLabelValues(instance, role).Set(float64(stats.ReplyIP))
	PiholePrivacyLevel.WithLabelValues(instance, role).Set(float64(stats.PrivacyLevel))

	// Convert status string to numeric value
	if stats.Status == "enabled" {
		PiholeStatusEnabled.WithLabelValues(instance, role).Set(1)
	} else {
		PiholeStatusEnabled.WithLabelValues(instance, role).Set(0)
	}
}

// UpdateQueryTypes updates Prometheus metrics with query type statistics
func UpdateQueryTypes(queryTypes *types.QueryTypes, instance, role string) {
	for queryType, percentage := range queryTypes.Querytypes {
		PiholeQueryTypes.WithLabelValues(instance, role, queryType).Set(percentage)
	}
}

// UpdateUpstreams updates Prometheus metrics with upstream server statistics
func UpdateUpstreams(upstreams *types.Upstreams, instance, role string) {
	for _, upstream := range upstreams.Upstreams {
		// Use the upstream name (like "8.8.8.8" or "cache", "blocklist") as the label
		upstreamLabel := upstream.Name
		if upstreamLabel == "" {
			upstreamLabel = upstream.IP
		}

		// Set the count of queries sent to this upstream
		PiholeUpstreamQueries.WithLabelValues(instance, role, upstreamLabel).Set(float64(upstream.Count))
	}
}

// UpdateTopDomains updates Prometheus metrics with top domains statistics
func UpdateTopDomains(topDomains *types.TopDomains, instance, role string, limit int) {
	// Update permitted domains (limited by configuration)
	count := 0
	for domain, queries := range topDomains.TopQueries {
		if count >= limit {
			break
		}
		PiholeTopPermittedDomains.WithLabelValues(instance, role, domain).Set(float64(queries))
		count++
	}

	// Update blocked domains (limited by configuration)
	count = 0
	for domain, queries := range topDomains.TopAds {
		if count >= limit {
			break
		}
		PiholeTopBlockedDomains.WithLabelValues(instance, role, domain).Set(float64(queries))
		count++
	}
}

// UpdateTopClients updates Prometheus metrics with top clients statistics
func UpdateTopClients(topClients *types.TopClients, instance, role string, limit int) {
	// Update top clients (limited by configuration)
	count := 0
	for client, queries := range topClients.TopSources {
		if count >= limit {
			break
		}
		PiholeTopClients.WithLabelValues(instance, role, client).Set(float64(queries))
		count++
	}
}

// RecordAPIError records an API error for a specific endpoint
func RecordAPIError(instance, role, endpoint string) {
	PiholeAPIErrors.WithLabelValues(instance, role, endpoint).Inc()
}

// RecordAPIResponseTime records the response time for an API endpoint
func RecordAPIResponseTime(instance, role, endpoint string, duration float64) {
	PiholeAPIResponseTime.WithLabelValues(instance, role, endpoint).Observe(duration)
}

// RecordSuccessfulCollection updates the timestamp of the last successful collection
func RecordSuccessfulCollection(instance, role string) {
	PiholeLastSuccessfulCollection.WithLabelValues(instance, role).Set(float64(time.Now().Unix()))
}
