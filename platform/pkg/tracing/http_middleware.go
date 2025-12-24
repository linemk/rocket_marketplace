package tracing

import (
	"net/http"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

// HTTPMiddleware возвращает middleware для трейсинга HTTP запросов
func HTTPMiddleware(serviceName string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tracer := otel.GetTracerProvider().Tracer(serviceName)
			propagator := otel.GetTextMapPropagator()

			// Извлекаем trace context из HTTP headers
			ctx := propagator.Extract(r.Context(), propagation.HeaderCarrier(r.Header))

			// Создаём span для HTTP запроса
			ctx, span := tracer.Start(
				ctx,
				r.Method+" "+r.URL.Path,
				trace.WithSpanKind(trace.SpanKindServer),
				trace.WithAttributes(
					attribute.String("http.method", r.Method),
					attribute.String("http.url", r.URL.String()),
					attribute.String("http.target", r.URL.Path),
					attribute.String("http.scheme", r.URL.Scheme),
					attribute.String("http.host", r.Host),
				),
			)
			defer span.End()

			// Создаём response writer wrapper для захвата status code
			rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			// Добавляем trace_id в response header
			if span.SpanContext().HasTraceID() {
				rw.Header().Set("X-Trace-Id", span.SpanContext().TraceID().String())
			}

			// Обрабатываем запрос
			next.ServeHTTP(rw, r.WithContext(ctx))

			// Добавляем status code в span
			span.SetAttributes(attribute.Int("http.status_code", rw.statusCode))

			// Устанавливаем статус span в зависимости от HTTP status
			if rw.statusCode >= 400 {
				span.SetStatus(codes.Error, http.StatusText(rw.statusCode))
			} else {
				span.SetStatus(codes.Ok, "")
			}
		})
	}
}

// responseWriter оборачивает http.ResponseWriter для захвата status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(statusCode int) {
	rw.statusCode = statusCode
	rw.ResponseWriter.WriteHeader(statusCode)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	return rw.ResponseWriter.Write(b)
}
