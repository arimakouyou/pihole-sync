package metrics

import (
	"testing"

	"github.com/arimakouyou/pihole-sync/internal/types"
)

func TestUpdateSummaryStats(t *testing.T) {
	stats := &types.SummaryStats{
		DomainsBlocked:     1234,
		DNSQueriesToday:    5678,
		AdsBlockedToday:    567,
		AdsPercentageToday: 10.5,
		UniqueDomains:      890,
		QueriesForwarded:   3456,
		QueriesCached:      2222,
		ClientsEverSeen:    15,
		UniqueClients:      12,
		DNSQueriesAllTypes: 5678,
		ReplyUnknown:       0,
		ReplyNodata:        10,
		ReplyNxdomain:      20,
		ReplyCname:         30,
		ReplyIP:            5618,
		PrivacyLevel:       0,
		Status:             "enabled",
	}

	// This should not panic
	UpdateSummaryStats(stats, "test.localhost", "master")

	// Test with disabled status
	stats.Status = "disabled"
	UpdateSummaryStats(stats, "test.localhost", "master")
}

func TestUpdateQueryTypes(t *testing.T) {
	queryTypes := &types.QueryTypes{
		Querytypes: map[string]float64{
			"A":     75.5,
			"AAAA":  15.2,
			"PTR":   5.1,
			"OTHER": 4.2,
		},
	}

	// This should not panic
	UpdateQueryTypes(queryTypes, "test.localhost", "master")
}

func TestUpdateUpstreams(t *testing.T) {
	upstreams := &types.Upstreams{
		Upstreams: []types.UpstreamServer{
			{
				IP:    "8.8.8.8",
				Name:  "8.8.8.8",
				Port:  53,
				Count: 1500,
				Statistics: struct {
					Response float64 `json:"response"`
					Variance float64 `json:"variance"`
				}{
					Response: 45.2,
					Variance: 1.2,
				},
			},
			{
				IP:    "cache",
				Name:  "cache",
				Port:  -1,
				Count: 800,
				Statistics: struct {
					Response float64 `json:"response"`
					Variance float64 `json:"variance"`
				}{
					Response: 30.1,
					Variance: 0.8,
				},
			},
		},
		TotalQueries:     2300,
		ForwardedQueries: 1500,
		Took:             0.001,
	}

	// This should not panic
	UpdateUpstreams(upstreams, "test.localhost", "master")
}

func TestUpdateTopDomains(t *testing.T) {
	topDomains := &types.TopDomains{
		TopQueries: map[string]int{
			"example.com":       100,
			"google.com":        80,
			"facebook.com":      60,
			"github.com":        40,
			"stackoverflow.com": 20,
		},
		TopAds: map[string]int{
			"ads.example.com":    50,
			"tracker.google.com": 30,
			"ads.facebook.com":   25,
		},
	}

	// Test with limit
	UpdateTopDomains(topDomains, "test.localhost", "master", 3)

	// Test with higher limit than available domains
	UpdateTopDomains(topDomains, "test.localhost", "master", 10)
}

func TestUpdateTopClients(t *testing.T) {
	topClients := &types.TopClients{
		TopSources: map[string]int{
			"192.168.1.10": 200,
			"192.168.1.11": 150,
			"192.168.1.12": 100,
			"192.168.1.13": 80,
			"192.168.1.14": 60,
		},
	}

	// Test with limit
	UpdateTopClients(topClients, "test.localhost", "master", 3)

	// Test with higher limit than available clients
	UpdateTopClients(topClients, "test.localhost", "master", 10)
}

func TestRecordAPIError(t *testing.T) {
	// This should not panic
	RecordAPIError("test.localhost", "master", "stats/summary")
	RecordAPIError("test.localhost", "slave", "stats/query_types")
}

func TestRecordAPIResponseTime(t *testing.T) {
	// This should not panic
	RecordAPIResponseTime("test.localhost", "master", "stats/summary", 0.5)
	RecordAPIResponseTime("test.localhost", "slave", "stats/query_types", 1.2)
}

func TestRecordSuccessfulCollection(t *testing.T) {
	// This should not panic
	RecordSuccessfulCollection("test.localhost", "master")
}
