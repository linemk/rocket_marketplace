package app

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"github.com/linemk/rocket-shop/order/internal/config"
	"github.com/linemk/rocket-shop/platform/pkg/closer"
	"github.com/linemk/rocket-shop/platform/pkg/logger"
	httpmiddleware "github.com/linemk/rocket-shop/platform/pkg/middleware/http"
	"github.com/linemk/rocket-shop/platform/pkg/migrator/pg"
	prommetrics "github.com/linemk/rocket-shop/platform/pkg/prometheus"
	"github.com/linemk/rocket-shop/platform/pkg/tracing"
	order_v1 "github.com/linemk/rocket-shop/shared/pkg/openapi/order/v1"
)

const (
	readHeaderTimeout = 5 * time.Second
)

type App struct {
	diContainer    *diContainer
	httpServer     *http.Server
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

	// Запускаем metrics HTTP server в отдельной горутине
	go func() {
		metricsPort := fmt.Sprintf(":%d", config.AppConfig().Metrics.Port())
		if err := prommetrics.StartMetricsServer(ctx, metricsPort, a.diContainer.PrometheusMetrics()); err != nil {
			logger.Error(ctx, fmt.Sprintf("Metrics server error: %v", err))
		}
	}()

	// Запускаем Kafka consumer в отдельной горутине
	go func() {
		if err := a.diContainer.ConsumerService(ctx).RunConsumers(ctx); err != nil {
			logger.Error(ctx, fmt.Sprintf("Kafka consumer error: %v", err))
		}
	}()

	// Запускаем HTTP сервер
	return a.runHTTPServer(ctx)
}

func (a *App) initDeps(ctx context.Context) error {
	inits := []func(context.Context) error{
		a.initConfig,
		a.initLogger,
		a.initTracer,
		a.initCloser,
		a.initDI,
		a.initMigrations,
		a.initHTTPServer,
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

func (a *App) initMigrations(ctx context.Context) error {
	// Создаем пул соединений
	pool, err := pgxpool.New(ctx, config.AppConfig().Postgres.DSN())
	if err != nil {
		return fmt.Errorf("failed to create pool: %w", err)
	}

	// Проверяем соединение
	if err := pool.Ping(ctx); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	logger.Info(ctx, "Successfully connected to PostgreSQL")

	// Сохраняем pool в DI контейнер перед миграциями
	a.diContainer.SetDBPool(pool)

	// Получаем *sql.DB для миграций
	sqlDB := stdlib.OpenDBFromPool(pool)

	// Выполняем миграции
	migrator := pg.NewMigrator(sqlDB, config.AppConfig().Postgres.MigrationsDir())
	if err := migrator.Up(); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	logger.Info(ctx, "Migrations applied successfully")

	// Закрываем sqlDB, так как мы будем использовать pool
	if err := sqlDB.Close(); err != nil {
		logger.Error(ctx, "Failed to close sqlDB")
	}

	// Регистрируем закрытие pool
	closer.AddNamed("PostgreSQL pool", func(ctx context.Context) error {
		pool.Close()
		return nil
	})

	return nil
}

func (a *App) initHTTPServer(ctx context.Context) error {
	// Создаем OpenAPI сервер
	orderServer, err := order_v1.NewServer(a.diContainer.OrderV1API(ctx))
	if err != nil {
		return fmt.Errorf("failed to create order server: %w", err)
	}

	// Создаем роутер
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(10 * time.Second))
	r.Use(render.SetContentType(render.ContentTypeJSON))
	r.Use(httpmiddleware.OptionalAuthMiddleware)

	// Добавляем tracing middleware если tracer инициализирован
	if a.tracerProvider != nil {
		r.Use(tracing.HTTPMiddleware(config.AppConfig().Logger.ServiceName()))
		logger.Info(ctx, "✅ HTTP tracing middleware added")
	}

	// Добавляем prometheus metrics middleware
	httpMetrics := prommetrics.NewHTTPMetrics(a.diContainer.PrometheusMetrics())
	r.Use(httpMetrics.Middleware())

	r.Mount("/", orderServer)

	// Создаем HTTP сервер
	a.httpServer = &http.Server{
		Addr:              config.AppConfig().OrderHTTP.Address(),
		Handler:           r,
		ReadHeaderTimeout: readHeaderTimeout,
	}

	closer.AddNamed("HTTP server", func(ctx context.Context) error {
		return a.httpServer.Shutdown(ctx)
	})

	return nil
}

func (a *App) runHTTPServer(ctx context.Context) error {
	logger.Info(ctx, fmt.Sprintf("🚀 OrderService HTTP server listening on %s", config.AppConfig().OrderHTTP.Address()))

	err := a.httpServer.ListenAndServe()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	return nil
}
