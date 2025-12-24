package tracing

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

const (
	defaultTimeout              = 10 * time.Second
	defaultRetryInitialInterval = 1 * time.Second
	defaultRetryMaxInterval     = 30 * time.Second
	defaultRetryMaxElapsedTime  = 1 * time.Minute
	defaultCompressor           = "gzip"
)

// InitTracerProvider инициализирует и возвращает TracerProvider с OTLP exporter
func InitTracerProvider(ctx context.Context, cfg Config) (*sdktrace.TracerProvider, error) {
	// Создаём OTLP exporter для отправки трейсов
	exporter, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint(cfg.Endpoint),
		otlptracegrpc.WithInsecure(),
		otlptracegrpc.WithTimeout(defaultTimeout),
		otlptracegrpc.WithCompressor(defaultCompressor),
		otlptracegrpc.WithRetry(otlptracegrpc.RetryConfig{
			Enabled:         true,
			InitialInterval: defaultRetryInitialInterval,
			MaxInterval:     defaultRetryMaxInterval,
			MaxElapsedTime:  defaultRetryMaxElapsedTime,
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP trace exporter: %w", err)
	}

	// Создаём ресурс с метаданными сервиса
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(cfg.ServiceName),
			semconv.ServiceVersion("1.0.0"),
			attribute.String("environment", cfg.Environment),
		),
		resource.WithHost(),
		resource.WithOS(),
		resource.WithProcess(),
		resource.WithContainer(),
		resource.WithTelemetrySDK(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Создаём TracerProvider с батчингом и семплированием
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.ParentBased(sdktrace.TraceIDRatioBased(cfg.SampleRate))),
	)

	return tp, nil
}

// ShutdownTracerProvider корректно завершает работу TracerProvider
func ShutdownTracerProvider(ctx context.Context, tp *sdktrace.TracerProvider) error {
	if tp == nil {
		return nil
	}

	shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := tp.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("failed to shutdown tracer provider: %w", err)
	}

	return nil
}
