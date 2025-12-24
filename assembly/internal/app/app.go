package app

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"github.com/linemk/rocket-shop/assembly/internal/config"
	"github.com/linemk/rocket-shop/platform/pkg/closer"
	"github.com/linemk/rocket-shop/platform/pkg/logger"
	prommetrics "github.com/linemk/rocket-shop/platform/pkg/prometheus"
)

type App struct {
	diContainer *diContainer
}

func NewApp(ctx context.Context) (*App, error) {
	app := &App{}

	err := app.initDeps(ctx)
	if err != nil {
		return nil, err
	}

	return app, nil
}

func (a *App) Run(ctx context.Context) error {
	defer func() {
		_ = logger.Close(ctx) //nolint:gosec // best-effort shutdown
		_ = logger.Sync()     //nolint:gosec // best-effort shutdown
		if err := closer.CloseAll(ctx); err != nil {
			logger.Error(ctx, "failed to close all resources", zap.Error(err))
		}
		closer.Wait()
	}()

	// Запускаем metrics HTTP server в отдельной горутине
	go func() {
		metricsPort := fmt.Sprintf(":%d", config.AppConfig().Metrics.Port())
		if err := prommetrics.StartMetricsServer(ctx, metricsPort, a.diContainer.PrometheusMetrics()); err != nil {
			logger.Error(ctx, fmt.Sprintf("Metrics server error: %v", err))
		}
	}()

	// Запускаем Kafka consumers
	return a.runConsumers(ctx)
}

func (a *App) initDeps(ctx context.Context) error {
	inits := []func(context.Context) error{
		a.initConfig,
		a.initLogger,
		a.initCloser,
		a.initDiContainer,
	}

	for _, f := range inits {
		err := f(ctx)
		if err != nil {
			return err
		}
	}

	return nil
}

func (a *App) initConfig(ctx context.Context) error {
	err := config.Load(".env")
	if err != nil {
		logger.Warn(ctx, "failed to load .env file", zap.Error(err))
	}

	return nil
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

func (a *App) initCloser(_ context.Context) error {
	closer.SetLogger(logger.Logger())
	return nil
}

func (a *App) initDiContainer(_ context.Context) error {
	a.diContainer = NewDiContainer()
	return nil
}

func (a *App) runConsumers(ctx context.Context) error {
	return a.diContainer.ConsumerService(ctx).RunConsumers(ctx)
}
