package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	SyncSuccessTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "pihole_sync_success_total",
		Help: "The total number of successful synchronizations",
	})

	SyncFailureTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "pihole_sync_failure_total",
		Help: "The total number of failed synchronizations",
	})

	GravityEditTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "pihole_gravity_edit_total",
		Help: "The total number of gravity list edits",
	})

	APICallTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "pihole_api_call_total",
		Help: "The total number of API calls",
	})

	ErrorTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "pihole_error_total",
		Help: "The total number of errors",
	})
)

func IncrementSyncSuccess() {
	SyncSuccessTotal.Inc()
}

func IncrementSyncFailure() {
	SyncFailureTotal.Inc()
}

func IncrementGravityEdit() {
	GravityEditTotal.Inc()
}

func IncrementAPICall() {
	APICallTotal.Inc()
}

func IncrementError() {
	ErrorTotal.Inc()
}
