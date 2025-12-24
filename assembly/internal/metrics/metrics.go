package metrics

import "github.com/prometheus/client_golang/prometheus"

// AssemblyMetrics holds Assembly service metrics
type AssemblyMetrics struct {
	Duration *prometheus.HistogramVec
}
