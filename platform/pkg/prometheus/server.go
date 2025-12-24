package prometheus

import (
	"context"
	"net/http"
	"time"

	"go.uber.org/zap"

	"github.com/linemk/rocket-shop/platform/pkg/logger"
)

// StartMetricsServer starts HTTP server for metrics endpoint
func StartMetricsServer(ctx context.Context, addr string, metrics *Metrics) error {
	mux := http.NewServeMux()
	// Prometheus по умолчанию ходит на /metrics.
	// Держим также "/" для обратной совместимости и ручной проверки.
	mux.Handle("/metrics", metrics.Handler())
	mux.Handle("/", metrics.Handler())

	server := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error(ctx, "metrics server error", zap.Error(err))
		}
	}()

	// Graceful shutdown on context cancellation
	go func() {
		<-ctx.Done()
		baseCtx := context.WithoutCancel(ctx)
		shutdownCtx, cancel := context.WithTimeout(baseCtx, 5*time.Second)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			logger.Error(baseCtx, "metrics server shutdown error", zap.Error(err))
		}
	}()

	return nil
}
