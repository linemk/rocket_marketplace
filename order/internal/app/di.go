package app

import (
	"context"
	"fmt"

	"github.com/IBM/sarama"
	"github.com/jackc/pgx/v5/pgxpool"

	inventoryClient "github.com/linemk/rocket-shop/order/internal/client/grpc/inventory/v1"
	paymentClient "github.com/linemk/rocket-shop/order/internal/client/grpc/payment/v1"
	"github.com/linemk/rocket-shop/order/internal/config"
	v1 "github.com/linemk/rocket-shop/order/internal/delivery/v1"
	ordermetrics "github.com/linemk/rocket-shop/order/internal/metrics"
	"github.com/linemk/rocket-shop/order/internal/repository"
	"github.com/linemk/rocket-shop/order/internal/service"
	"github.com/linemk/rocket-shop/order/internal/service/consumer/order_consumer"
	"github.com/linemk/rocket-shop/order/internal/service/producer/order_producer"
	"github.com/linemk/rocket-shop/order/internal/usecase"
	"github.com/linemk/rocket-shop/platform/pkg/closer"
	"github.com/linemk/rocket-shop/platform/pkg/kafka/consumer"
	"github.com/linemk/rocket-shop/platform/pkg/kafka/producer"
	"github.com/linemk/rocket-shop/platform/pkg/logger"
	kafkaMiddleware "github.com/linemk/rocket-shop/platform/pkg/middleware/kafka"
	prommetrics "github.com/linemk/rocket-shop/platform/pkg/prometheus"
	iamclient "github.com/linemk/rocket-shop/shared/pkg/iamclient"
	order_v1 "github.com/linemk/rocket-shop/shared/pkg/openapi/order/v1"
)

type diContainer struct {
	orderV1API order_v1.Handler

	orderUseCase usecase.OrderUseCase

	orderRepository repository.OrderRepository

	inventoryClient inventoryClient.InventoryClient
	paymentClient   paymentClient.PaymentClient
	iamClient       *iamclient.Client

	consumerService      service.ConsumerService
	orderProducerService service.OrderProducerService

	prometheusMetrics *prommetrics.Metrics
	orderMetrics      *ordermetrics.OrderMetrics

	dbPool *pgxpool.Pool
}

func NewDiContainer() *diContainer {
	return &diContainer{}
}

func (d *diContainer) SetDBPool(pool *pgxpool.Pool) {
	d.dbPool = pool
}

func (d *diContainer) OrderV1API(ctx context.Context) order_v1.Handler {
	if d.orderV1API == nil {
		d.orderV1API = v1.NewAPI(d.OrderUseCase(ctx))
	}

	return d.orderV1API
}

func (d *diContainer) OrderUseCase(ctx context.Context) usecase.OrderUseCase {
	if d.orderUseCase == nil {
		d.orderUseCase = usecase.NewUseCase(
			d.OrderRepository(ctx),
			d.InventoryClient(ctx),
			d.PaymentClient(ctx),
			d.OrderProducerService(ctx),
			d.OrderMetrics(),
		)
	}

	return d.orderUseCase
}

func (d *diContainer) OrderRepository(ctx context.Context) repository.OrderRepository {
	if d.orderRepository == nil {
		d.orderRepository = repository.NewRepository(d.dbPool)
	}

	return d.orderRepository
}

func (d *diContainer) InventoryClient(ctx context.Context) inventoryClient.InventoryClient {
	if d.inventoryClient == nil {
		client, err := inventoryClient.NewClient(config.AppConfig().InventoryGRPC.Address())
		if err != nil {
			panic(fmt.Sprintf("failed to create inventory client: %s\n", err.Error()))
		}

		closer.AddNamed("Inventory gRPC client", func(ctx context.Context) error {
			return client.Close()
		})

		d.inventoryClient = client
	}

	return d.inventoryClient
}

func (d *diContainer) PaymentClient(ctx context.Context) paymentClient.PaymentClient {
	if d.paymentClient == nil {
		client, err := paymentClient.NewClient(config.AppConfig().PaymentGRPC.Address())
		if err != nil {
			panic(fmt.Sprintf("failed to create payment client: %s\n", err.Error()))
		}

		closer.AddNamed("Payment gRPC client", func(ctx context.Context) error {
			return client.Close()
		})

		d.paymentClient = client
	}

	return d.paymentClient
}

func (d *diContainer) ConsumerService(ctx context.Context) service.ConsumerService {
	if d.consumerService == nil {
		// Создаем Kafka consumer group
		saramaConfig := sarama.NewConfig()
		saramaConfig.Version = sarama.V2_6_0_0
		saramaConfig.Consumer.Group.Rebalance.Strategy = sarama.NewBalanceStrategyRoundRobin()
		saramaConfig.Consumer.Offsets.Initial = sarama.OffsetOldest

		consumerGroup, err := sarama.NewConsumerGroup(
			config.AppConfig().Kafka.Brokers(),
			config.AppConfig().OrderAssembledConsumer.GroupID(),
			saramaConfig,
		)
		if err != nil {
			panic(fmt.Sprintf("failed to create Kafka consumer group: %s\n", err.Error()))
		}

		closer.AddNamed("Kafka consumer group", func(ctx context.Context) error {
			return consumerGroup.Close()
		})

		// Создаем Kafka consumer с middleware
		kafkaConsumer := consumer.NewConsumer(
			consumerGroup,
			[]string{config.AppConfig().OrderAssembledConsumer.Topic()},
			logger.Logger(),
			kafkaMiddleware.Logging(logger.Logger()),
		)

		// Создаем handler
		handler := order_consumer.NewHandler(d.OrderRepository(ctx), logger.Logger())

		d.consumerService = order_consumer.NewConsumer(kafkaConsumer, handler, logger.Logger())
	}

	return d.consumerService
}

func (d *diContainer) OrderProducerService(ctx context.Context) service.OrderProducerService {
	if d.orderProducerService == nil {
		// Создаем Kafka sync producer
		saramaConfig := sarama.NewConfig()
		saramaConfig.Version = sarama.V2_6_0_0
		saramaConfig.Producer.Return.Successes = true
		saramaConfig.Producer.RequiredAcks = sarama.WaitForAll
		saramaConfig.Producer.Retry.Max = 5

		syncProducer, err := sarama.NewSyncProducer(config.AppConfig().Kafka.Brokers(), saramaConfig)
		if err != nil {
			panic(fmt.Sprintf("failed to create Kafka sync producer: %s\n", err.Error()))
		}

		closer.AddNamed("Kafka sync producer", func(ctx context.Context) error {
			return syncProducer.Close()
		})

		kafkaProducer := producer.NewProducer(
			syncProducer,
			config.AppConfig().OrderPaidProducer.Topic(),
			logger.Logger(),
		)

		d.orderProducerService = order_producer.NewProducer(kafkaProducer, logger.Logger())
	}

	return d.orderProducerService
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

func (d *diContainer) OrderMetrics() *ordermetrics.OrderMetrics {
	if d.orderMetrics == nil {
		pm := d.PrometheusMetrics()
		d.orderMetrics = &ordermetrics.OrderMetrics{
			OrdersTotal: pm.NewCounter(
				"orders_total",
				"Total number of orders created",
				[]string{"status"},
			),
			RevenueTotal: pm.NewCounter(
				"orders_revenue_total",
				"Total revenue from orders",
				[]string{"payment_method"},
			),
		}
	}

	return d.orderMetrics
}
