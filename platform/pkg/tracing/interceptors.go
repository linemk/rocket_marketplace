package tracing

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	grpccodes "google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// UnaryServerInterceptor возвращает gRPC server interceptor для трейсинга
func UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		tracer := otel.GetTracerProvider().Tracer("grpc-server")
		propagator := otel.GetTextMapPropagator()

		// Извлекаем metadata из входящего запроса
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			md = metadata.New(nil)
		}

		// Извлекаем trace context из metadata (propagation)
		carrier := MetadataCarrier{MD: md}
		ctx = propagator.Extract(ctx, carrier)

		// Создаём server span
		ctx, span := tracer.Start(
			ctx,
			info.FullMethod,
			trace.WithSpanKind(trace.SpanKindServer),
		)
		defer span.End()

		// Вызываем обработчик
		resp, err := handler(ctx, req)

		// Записываем ошибку в span если есть
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())

			// Добавляем gRPC status code
			if st, ok := status.FromError(err); ok {
				span.SetAttributes(statusCodeAttr(st.Code()))
			}
		} else {
			span.SetStatus(codes.Ok, "")
		}

		return resp, err
	}
}

// UnaryClientInterceptor возвращает gRPC client interceptor для трейсинга
func UnaryClientInterceptor() grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req, reply interface{},
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		tracer := otel.GetTracerProvider().Tracer("grpc-client")
		propagator := otel.GetTextMapPropagator()

		// Создаём client span
		ctx, span := tracer.Start(
			ctx,
			method,
			trace.WithSpanKind(trace.SpanKindClient),
		)
		defer span.End()

		// Получаем существующие metadata или создаём новые
		md, ok := metadata.FromOutgoingContext(ctx)
		if !ok {
			md = metadata.New(nil)
		} else {
			// Копируем metadata чтобы не изменять оригинал
			md = md.Copy()
		}

		// Внедряем trace context в metadata (propagation)
		carrier := MetadataCarrier{MD: md}
		propagator.Inject(ctx, carrier)

		// Обновляем context с новыми metadata
		ctx = metadata.NewOutgoingContext(ctx, carrier.MD)

		// Вызываем метод
		err := invoker(ctx, method, req, reply, cc, opts...)

		// Записываем ошибку в span если есть
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())

			// Добавляем gRPC status code
			if st, ok := status.FromError(err); ok {
				span.SetAttributes(statusCodeAttr(st.Code()))
			}
		} else {
			span.SetStatus(codes.Ok, "")
		}

		return err
	}
}

// statusCodeAttr возвращает attribute для gRPC status code
func statusCodeAttr(c grpccodes.Code) attribute.KeyValue {
	return attribute.Int("rpc.grpc.status_code", int(c))
}
