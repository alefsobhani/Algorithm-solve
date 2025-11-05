package matching

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	matchingDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "matching_time_seconds",
		Help:    "Time spent attempting to match a trip to a driver.",
		Buckets: prometheus.DefBuckets,
	}, []string{"result"})

	assignmentAttempts = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "assignment_attempts_total",
		Help: "Total assignment attempts grouped by outcome.",
	}, []string{"result"})
)
