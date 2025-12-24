package prometheus

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metrics represents Prometheus metrics registry and helpers
type Metrics struct {
	registry *prometheus.Registry
	factory  promauto.Factory
}

// New creates new Prometheus Metrics instance
func New() *Metrics {
	registry := prometheus.NewRegistry()
	factory := promauto.With(registry)

	return &Metrics{
		registry: registry,
		factory:  factory,
	}
}

// NewCounter creates new counter metric with labels
func (m *Metrics) NewCounter(name, help string, labels []string) *prometheus.CounterVec {
	return m.factory.NewCounterVec(prometheus.CounterOpts{
		Name: name,
		Help: help,
	}, labels)
}

// NewHistogram creates new histogram metric with labels
func (m *Metrics) NewHistogram(name, help string, labels []string, buckets []float64) *prometheus.HistogramVec {
	return m.factory.NewHistogramVec(prometheus.HistogramOpts{
		Name:    name,
		Help:    help,
		Buckets: buckets,
	}, labels)
}

// Handler returns HTTP handler for /metrics endpoint
func (m *Metrics) Handler() http.Handler {
	return promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{})
}

// Registry returns underlying Prometheus registry
func (m *Metrics) Registry() *prometheus.Registry {
	return m.registry
}
