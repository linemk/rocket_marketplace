package app

import (
	"context"

	"go.uber.org/zap"

	"github.com/linemk/rocket-shop/notification/internal/config"
	"github.com/linemk/rocket-shop/platform/pkg/closer"
	"github.com/linemk/rocket-shop/platform/pkg/logger"
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
		_ = logger.Close()
		_ = logger.Sync()
		if err := closer.CloseAll(ctx); err != nil {
			logger.Error(ctx, "failed to close all resources", zap.Error(err))
		}
		closer.Wait()
	}()

	// Запускаем оба Kafka consumers параллельно
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

func (a *App) initLogger(_ context.Context) error {
	return logger.Init(
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
	// Запускаем оба consumer'а одновременно
	errChan := make(chan error, 2)

	go func() {
		errChan <- a.diContainer.OrderPaidConsumer(ctx).RunConsumer(ctx)
	}()

	go func() {
		errChan <- a.diContainer.OrderAssembledConsumer(ctx).RunConsumer(ctx)
	}()

	// Ждем первой ошибки
	return <-errChan
}
