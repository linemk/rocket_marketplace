package app

import (
	"context"
	"fmt"
	"net"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"

	"github.com/linemk/rocket-shop/payment/internal/config"
	"github.com/linemk/rocket-shop/platform/pkg/closer"
	"github.com/linemk/rocket-shop/platform/pkg/grpc/health"
	"github.com/linemk/rocket-shop/platform/pkg/logger"
	"github.com/linemk/rocket-shop/platform/pkg/tracing"
	payment_v1 "github.com/linemk/rocket-shop/shared/pkg/proto/payment/v1"
)

type App struct {
	diContainer    *diContainer
	grpcServer     *grpc.Server
	listener       net.Listener
	tracerProvider *sdktrace.TracerProvider
}

// New создает новое приложение
func New(ctx context.Context) (*App, error) {
	a := &App{}

	err := a.initDeps(ctx)
	if err != nil {
		return nil, err
	}

	return a, nil
}

// Run запускает приложение
func (a *App) Run(ctx context.Context) error {
	defer func() {
		_ = logger.Close(ctx) //nolint:gosec // best-effort shutdown
		_ = logger.Sync()     //nolint:gosec // best-effort shutdown
	}()

	return a.runGRPCServer(ctx)
}

func (a *App) initDeps(ctx context.Context) error {
	inits := []func(context.Context) error{
		a.initConfig,
		a.initLogger,
		a.initTracer,
		a.initCloser,
		a.initDI,
		a.initListener,
		a.initGRPCServer,
	}

	for _, f := range inits {
		err := f(ctx)
		if err != nil {
			return err
		}
	}

	return nil
}

func (a *App) initConfig(_ context.Context) error {
	return config.Load()
}

func (a *App) initLogger(ctx context.Context) error {
	return logger.Init(
		ctx,
		config.AppConfig().Logger.Level(),
		config.AppConfig().Logger.AsJSON(),
		config.AppConfig().Logger.OTLPEnabled(),
		config.AppConfig().Logger.OTLPEndpoint(),
		config.AppConfig().Logger.ServiceName(),
	)
}

func (a *App) initTracer(ctx context.Context) error {
	cfg := tracing.NewConfigFromEnv()
	if !cfg.Enabled {
		logger.Info(ctx, "Tracing disabled")
		return nil
	}

	tp, err := tracing.InitTracerProvider(ctx, cfg)
	if err != nil {
		return fmt.Errorf("init tracer: %w", err)
	}

	a.tracerProvider = tp
	otel.SetTracerProvider(tp)

	propagator := propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	)
	otel.SetTextMapPropagator(propagator)

	logger.Info(ctx, fmt.Sprintf("✅ Tracing initialized for service: %s", cfg.ServiceName))

	return nil
}

func (a *App) initCloser(_ context.Context) error {
	closer.SetLogger(logger.Logger())

	if a.tracerProvider != nil {
		closer.AddNamed("tracer provider", func(ctx context.Context) error {
			return tracing.ShutdownTracerProvider(ctx, a.tracerProvider)
		})
	}

	return nil
}

func (a *App) initDI(_ context.Context) error {
	a.diContainer = NewDiContainer()
	return nil
}

func (a *App) initListener(_ context.Context) error {
	listener, err := net.Listen("tcp", config.AppConfig().PaymentGRPC.Address())
	if err != nil {
		return err
	}

	closer.AddNamed("TCP listener", func(ctx context.Context) error {
		return listener.Close()
	})

	a.listener = listener

	return nil
}

func (a *App) initGRPCServer(ctx context.Context) error {
	opts := []grpc.ServerOption{
		grpc.Creds(insecure.NewCredentials()),
	}

	// Добавляем tracing interceptor если tracer инициализирован
	if a.tracerProvider != nil {
		opts = append(opts, grpc.UnaryInterceptor(tracing.UnaryServerInterceptor()))
		logger.Info(ctx, "✅ gRPC server tracing interceptor added")
	}

	a.grpcServer = grpc.NewServer(opts...)

	closer.AddNamed("gRPC server", func(ctx context.Context) error {
		a.grpcServer.GracefulStop()
		return nil
	})

	reflection.Register(a.grpcServer)

	// Регистрируем health service для проверки работоспособности
	health.RegisterService(a.grpcServer)

	// Регистрируем PaymentService
	payment_v1.RegisterPaymentServiceServer(a.grpcServer, a.diContainer.PaymentV1API(ctx))

	return nil
}

func (a *App) runGRPCServer(ctx context.Context) error {
	logger.Info(ctx, fmt.Sprintf("💳 PaymentService gRPC server listening on %s", config.AppConfig().PaymentGRPC.Address()))

	err := a.grpcServer.Serve(a.listener)
	if err != nil {
		return err
	}

	return nil
}
