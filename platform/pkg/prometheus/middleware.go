package prometheus

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// HTTPMetrics holds HTTP-related metrics
type HTTPMetrics struct {
	requestsTotal   *prometheus.CounterVec
	requestDuration *prometheus.HistogramVec
}

// NewHTTPMetrics creates HTTP metrics
func NewHTTPMetrics(m *Metrics) *HTTPMetrics {
	return &HTTPMetrics{
		requestsTotal: m.NewCounter(
			"http_requests_total",
			"Total HTTP requests",
			[]string{"method", "path", "status"},
		),
		requestDuration: m.NewHistogram(
			"http_request_duration_seconds",
			"HTTP request duration in seconds",
			[]string{"method", "path"},
			[]float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1, 2, 5},
		),
	}
}

// Middleware returns chi-compatible HTTP metrics middleware
func (h *HTTPMetrics) Middleware() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Wrap response writer to capture status code
			wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			next.ServeHTTP(wrapped, r)

			duration := time.Since(start).Seconds()
			path := r.URL.Path
			method := r.Method
			status := wrapped.statusCode

			h.requestsTotal.WithLabelValues(method, path, http.StatusText(status)).Inc()
			h.requestDuration.WithLabelValues(method, path).Observe(duration)
		})
	}
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

// WriteHeader captures the status code
func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}
