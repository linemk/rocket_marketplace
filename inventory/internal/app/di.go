package app

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"

	"github.com/linemk/rocket-shop/inventory/internal/config"
	v1 "github.com/linemk/rocket-shop/inventory/internal/delivery/v1"
	"github.com/linemk/rocket-shop/inventory/internal/repository"
	inventoryRepository "github.com/linemk/rocket-shop/inventory/internal/repository/inventory"
	"github.com/linemk/rocket-shop/inventory/internal/usecase"
	"github.com/linemk/rocket-shop/platform/pkg/closer"
	prommetrics "github.com/linemk/rocket-shop/platform/pkg/prometheus"
	iamclient "github.com/linemk/rocket-shop/shared/pkg/iamclient"
	inventory_v1 "github.com/linemk/rocket-shop/shared/pkg/proto/inventory/v1"
)

type diContainer struct {
	inventoryV1API inventory_v1.InventoryServiceServer

	inventoryUseCase usecase.InventoryUseCase

	inventoryRepository repository.InventoryRepository

	mongoDBClient     *mongo.Client
	mongoDBHandle     *mongo.Database
	iamClient         *iamclient.Client
	prometheusMetrics *prommetrics.Metrics
}

func NewDiContainer() *diContainer {
	return &diContainer{}
}

func (d *diContainer) InventoryV1API(ctx context.Context) inventory_v1.InventoryServiceServer {
	if d.inventoryV1API == nil {
		d.inventoryV1API = v1.NewAPI(d.InventoryUseCase(ctx))
	}

	return d.inventoryV1API
}

func (d *diContainer) InventoryUseCase(ctx context.Context) usecase.InventoryUseCase {
	if d.inventoryUseCase == nil {
		d.inventoryUseCase = usecase.NewUseCase(d.InventoryRepository(ctx))
	}

	return d.inventoryUseCase
}

func (d *diContainer) InventoryRepository(ctx context.Context) repository.InventoryRepository {
	if d.inventoryRepository == nil {
		d.inventoryRepository = inventoryRepository.NewMongoRepository(ctx, d.MongoDBHandle(ctx))
	}

	return d.inventoryRepository
}

func (d *diContainer) MongoDBClient(ctx context.Context) *mongo.Client {
	if d.mongoDBClient == nil {
		client, err := mongo.Connect(ctx, options.Client().ApplyURI(config.AppConfig().Mongo.URI()))
		if err != nil {
			panic(fmt.Sprintf("failed to connect to MongoDB: %s\n", err.Error()))
		}

		err = client.Ping(ctx, readpref.Primary())
		if err != nil {
			panic(fmt.Sprintf("failed to ping MongoDB: %v\n", err))
		}

		closer.AddNamed("MongoDB client", func(ctx context.Context) error {
			return client.Disconnect(ctx)
		})

		d.mongoDBClient = client
	}

	return d.mongoDBClient
}

func (d *diContainer) MongoDBHandle(ctx context.Context) *mongo.Database {
	if d.mongoDBHandle == nil {
		d.mongoDBHandle = d.MongoDBClient(ctx).Database(config.AppConfig().Mongo.DatabaseName())
	}

	return d.mongoDBHandle
}

func (d *diContainer) IAMClient(ctx context.Context) *iamclient.Client {
	if d.iamClient == nil {
		client, err := iamclient.New(ctx, config.AppConfig().IAMGRPC.Address())
		if err != nil {
			panic(fmt.Sprintf("failed to create IAM client: %s\n", err.Error()))
		}
		closer.AddNamed("IAM gRPC client", func(ctx context.Context) error {
			return client.Close()
		})
		d.iamClient = client
	}
	return d.iamClient
}

func (d *diContainer) PrometheusMetrics() *prommetrics.Metrics {
	if d.prometheusMetrics == nil {
		d.prometheusMetrics = prommetrics.New()
	}

	return d.prometheusMetrics
}
