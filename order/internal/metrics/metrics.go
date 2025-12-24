package metrics

import "github.com/prometheus/client_golang/prometheus"

// OrderMetrics holds Order service business metrics
type OrderMetrics struct {
	OrdersTotal  *prometheus.CounterVec
	RevenueTotal *prometheus.CounterVec
}
