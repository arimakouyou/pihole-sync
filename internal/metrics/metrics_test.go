package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
)

func TestMetricsIncrement(t *testing.T) {
	prometheus.DefaultRegisterer = prometheus.NewRegistry()
	
	SyncSuccessTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "pihole_sync_success_total",
		Help: "The total number of successful synchronizations",
	})
	prometheus.MustRegister(SyncSuccessTotal)

	initialValue := testutil.ToFloat64(SyncSuccessTotal)
	
	IncrementSyncSuccess()
	
	newValue := testutil.ToFloat64(SyncSuccessTotal)
	assert.Equal(t, initialValue+1, newValue)
}

func TestAllMetricsIncrement(t *testing.T) {
	IncrementSyncSuccess()
	IncrementSyncFailure()
	IncrementGravityEdit()
	IncrementAPICall()
	IncrementError()
}
