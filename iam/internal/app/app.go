package app

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"github.com/linemk/rocket-shop/iam/internal/config"
	"github.com/linemk/rocket-shop/iam/internal/di"
	authv1 "github.com/linemk/rocket-shop/shared/pkg/proto/auth/v1"
	userv1 "github.com/linemk/rocket-shop/shared/pkg/proto/user/v1"

	"github.com/linemk/rocket-shop/platform/pkg/cache"
	rediscache "github.com/linemk/rocket-shop/platform/pkg/cache/redis"
	"github.com/linemk/rocket-shop/platform/pkg/closer"
	"github.com/linemk/rocket-shop/platform/pkg/grpcserver"
	"github.com/linemk/rocket-shop/platform/pkg/logger"
	"github.com/linemk/rocket-shop/platform/pkg/migrator/pg"
)

type App struct {
	grpcServer  *grpc.Server
	diContainer *di.Container
	db          *pgxpool.Pool
	cache       cache.Client
}

func New(ctx context.Context) (*App, error) {
	a := &App{}

	err := a.initDeps(ctx)
	if err != nil {
		return nil, err
	}

	return a, nil
}

func (a *App) Run(ctx context.Context) error {
	defer func() {
		_ = logger.Close()
		_ = logger.Sync()
	}()

	return a.runGRPCServer(ctx)
}

func (a *App) initDeps(ctx context.Context) error {
	inits := []func(context.Context) error{
		a.initConfig,
		a.initLogger,
		a.initCloser,
		a.initCache,
		a.initDatabase,
		a.initMigrations,
		a.initDI,
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

func (a *App) initCache(ctx context.Context) error {
	redisConfig := config.AppConfig().Redis
	cacheConfig := cache.Config{
		Addr:         redisConfig.Addr(),
		Password:     redisConfig.Password(),
		DB:           redisConfig.DB(),
		DialTimeout:  redisConfig.DialTimeout(),
		ReadTimeout:  redisConfig.ReadTimeout(),
		WriteTimeout: redisConfig.WriteTimeout(),
		PoolSize:     redisConfig.PoolSize(),
	}

	var err error
	a.cache, err = rediscache.NewClient(cacheConfig)
	if err != nil {
		return fmt.Errorf("failed to create cache client: %w", err)
	}

	if err := a.cache.Ping(ctx); err != nil {
		return fmt.Errorf("failed to ping Redis: %w", err)
	}

	logger.Info(ctx, "Successfully connected to Redis")

	closer.AddNamed("Redis cache", func(ctx context.Context) error {
		return a.cache.Close()
	})

	return nil
}

func (a *App) initDatabase(ctx context.Context) error {
	pool, err := pgxpool.New(ctx, config.AppConfig().Postgres.DSN())
	if err != nil {
		return fmt.Errorf("failed to create pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	logger.Info(ctx, "Successfully connected to PostgreSQL")

	a.db = pool

	closer.AddNamed("PostgreSQL pool", func(ctx context.Context) error {
		pool.Close()
		return nil
	})

	return nil
}

func (a *App) initMigrations(ctx context.Context) error {
	sqlDB := stdlib.OpenDBFromPool(a.db)

	migrator := pg.NewMigrator(sqlDB, config.AppConfig().Postgres.MigrationsDir())
	if err := migrator.Up(); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	logger.Info(ctx, "Migrations applied successfully")

	if err := sqlDB.Close(); err != nil {
		logger.Error(ctx, "Failed to close sqlDB")
	}

	return nil
}

func (a *App) initDI(_ context.Context) error {
	a.diContainer = di.New(a.db, a.cache)
	return nil
}

func (a *App) initGRPCServer(ctx context.Context) error {
	opts := []grpc.ServerOption{
		grpc.ConnectionTimeout(5 * time.Second),
	}

	a.grpcServer = grpc.NewServer(opts...)

	authv1.RegisterAuthServiceServer(a.grpcServer, a.diContainer.AuthHandler)
	userv1.RegisterUserServiceServer(a.grpcServer, a.diContainer.UserHandler)

	reflection.Register(a.grpcServer)

	closer.AddNamed("gRPC server", func(ctx context.Context) error {
		a.grpcServer.GracefulStop()
		return nil
	})

	return nil
}

func (a *App) runGRPCServer(ctx context.Context) error {
	listener, err := grpcserver.NewListener(config.AppConfig().GRPC.Address())
	if err != nil {
		return fmt.Errorf("failed to create listener: %w", err)
	}

	logger.Info(ctx, fmt.Sprintf("🚀 IAM gRPC server listening on %s", config.AppConfig().GRPC.Address()))

	if err := a.grpcServer.Serve(listener); err != nil {
		return fmt.Errorf("failed to serve gRPC: %w", err)
	}

	return nil
}
