package types

// SummaryStats represents the statistics from /stats/summary endpoint
type SummaryStats struct {
	DomainsBlocked     int     `json:"domains_being_blocked"`
	DNSQueriesToday    int     `json:"dns_queries_today"`
	AdsBlockedToday    int     `json:"ads_blocked_today"`
	AdsPercentageToday float64 `json:"ads_percentage_today"`
	UniqueDomains      int     `json:"unique_domains"`
	QueriesForwarded   int     `json:"queries_forwarded"`
	QueriesCached      int     `json:"queries_cached"`
	ClientsEverSeen    int     `json:"clients_ever_seen"`
	UniqueClients      int     `json:"unique_clients"`
	DNSQueriesAllTypes int     `json:"dns_queries_all_types"`
	ReplyUnknown       int     `json:"reply_UNKNOWN"`
	ReplyNodata        int     `json:"reply_NODATA"`
	ReplyNxdomain      int     `json:"reply_NXDOMAIN"`
	ReplyCname         int     `json:"reply_CNAME"`
	ReplyIP            int     `json:"reply_IP"`
	PrivacyLevel       int     `json:"privacy_level"`
	Status             string  `json:"status"`
}

// QueryTypes represents the statistics from /stats/query_types endpoint
type QueryTypes struct {
	Querytypes map[string]float64 `json:"querytypes"`
}

// UpstreamServer represents a single upstream server
type UpstreamServer struct {
	IP         string `json:"ip"`
	Name       string `json:"name"`
	Port       int    `json:"port"`
	Count      int    `json:"count"`
	Statistics struct {
		Response float64 `json:"response"`
		Variance float64 `json:"variance"`
	} `json:"statistics"`
}

// Upstreams represents the statistics from /stats/upstreams endpoint
type Upstreams struct {
	Upstreams        []UpstreamServer `json:"upstreams"`
	TotalQueries     int              `json:"total_queries"`
	ForwardedQueries int              `json:"forwarded_queries"`
	Took             float64          `json:"took"`
}

// TopDomains represents the statistics from /stats/top_domains endpoint
type TopDomains struct {
	TopQueries map[string]int `json:"top_queries"`
	TopAds     map[string]int `json:"top_ads"`
}

// TopClients represents the statistics from /stats/top_clients endpoint
type TopClients struct {
	TopSources map[string]int `json:"top_sources"`
}

// RecentBlocked represents the statistics from /stats/recent_blocked endpoint
type RecentBlocked []string

// CacheInfo represents DNS cache information
type CacheInfo struct {
	CacheSize      int `json:"cache-size"`
	CacheInserted  int `json:"cache-inserted"`
	CacheEvictions int `json:"cache-evictions"`
	CacheLiveFreed int `json:"cache-live-freed"`
}
