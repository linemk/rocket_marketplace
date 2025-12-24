package app

import (
	"context"
	"fmt"
	"sync"

	"github.com/IBM/sarama"
	"go.uber.org/zap"

	"github.com/linemk/rocket-shop/notification/internal/client/http/telegram"
	"github.com/linemk/rocket-shop/notification/internal/config"
	"github.com/linemk/rocket-shop/notification/internal/service"
	"github.com/linemk/rocket-shop/notification/internal/service/consumer/order_assembled_consumer"
	"github.com/linemk/rocket-shop/notification/internal/service/consumer/order_paid_consumer"
	telegramService "github.com/linemk/rocket-shop/notification/internal/service/telegram"
	"github.com/linemk/rocket-shop/platform/pkg/closer"
	"github.com/linemk/rocket-shop/platform/pkg/kafka/consumer"
	"github.com/linemk/rocket-shop/platform/pkg/logger"
	kafkaMiddleware "github.com/linemk/rocket-shop/platform/pkg/middleware/kafka"
)

type LoggerInterface interface {
	Info(ctx context.Context, msg string, fields ...zap.Field)
	Error(ctx context.Context, msg string, fields ...zap.Field)
}

type diContainer struct {
	orderPaidConsumerOnce      sync.Once
	orderAssembledConsumerOnce sync.Once
	telegramServiceOnce        sync.Once

	orderPaidConsumer      *order_paid_consumer.Consumer
	orderAssembledConsumer *order_assembled_consumer.Consumer
	telegramService        service.TelegramService
}

func NewDiContainer() *diContainer {
	return &diContainer{}
}

//nolint:dupl // Дублирование с OrderAssembledConsumer неизбежно из-за разных типов
func (d *diContainer) OrderPaidConsumer(ctx context.Context) *order_paid_consumer.Consumer {
	d.orderPaidConsumerOnce.Do(func() {
		// Создаем Kafka consumer group
		saramaConfig := sarama.NewConfig()
		saramaConfig.Version = sarama.V2_6_0_0
		saramaConfig.Consumer.Group.Rebalance.Strategy = sarama.NewBalanceStrategyRoundRobin()
		saramaConfig.Consumer.Offsets.Initial = sarama.OffsetOldest

		consumerGroup, err := sarama.NewConsumerGroup(
			config.AppConfig().Kafka.Brokers(),
			config.AppConfig().OrderPaidConsumer.GroupID(),
			saramaConfig,
		)
		if err != nil {
			panic(fmt.Sprintf("failed to create Kafka consumer group for OrderPaid: %s\n", err.Error()))
		}

		closer.AddNamed("Kafka consumer group for OrderPaid", func(ctx context.Context) error {
			return consumerGroup.Close()
		})

		// Создаем Kafka consumer с middleware
		kafkaConsumer := consumer.NewConsumer(
			consumerGroup,
			[]string{config.AppConfig().OrderPaidConsumer.Topic()},
			d.Logger(ctx),
			kafkaMiddleware.Logging(d.Logger(ctx)),
		)

		// Создаем handler
		handler := order_paid_consumer.NewHandler(d.TelegramService(ctx), d.Logger(ctx))

		d.orderPaidConsumer = order_paid_consumer.NewConsumer(kafkaConsumer, handler, d.Logger(ctx))
	})

	return d.orderPaidConsumer
}

//nolint:dupl // Дублирование с OrderPaidConsumer неизбежно из-за разных типов
func (d *diContainer) OrderAssembledConsumer(ctx context.Context) *order_assembled_consumer.Consumer {
	d.orderAssembledConsumerOnce.Do(func() {
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
			panic(fmt.Sprintf("failed to create Kafka consumer group for OrderAssembled: %s\n", err.Error()))
		}

		closer.AddNamed("Kafka consumer group for OrderAssembled", func(ctx context.Context) error {
			return consumerGroup.Close()
		})

		// Создаем Kafka consumer с middleware
		kafkaConsumer := consumer.NewConsumer(
			consumerGroup,
			[]string{config.AppConfig().OrderAssembledConsumer.Topic()},
			d.Logger(ctx),
			kafkaMiddleware.Logging(d.Logger(ctx)),
		)

		// Создаем handler
		handler := order_assembled_consumer.NewHandler(d.TelegramService(ctx), d.Logger(ctx))

		d.orderAssembledConsumer = order_assembled_consumer.NewConsumer(kafkaConsumer, handler, d.Logger(ctx))
	})

	return d.orderAssembledConsumer
}

func (d *diContainer) TelegramService(ctx context.Context) service.TelegramService {
	d.telegramServiceOnce.Do(func() {
		telegramClient := telegram.NewClient(
			config.AppConfig().TelegramBot.Token(),
			config.AppConfig().TelegramBot.ChatID(),
			d.Logger(ctx),
		)
		d.telegramService = telegramService.NewService(telegramClient, d.Logger(ctx))
	})

	return d.telegramService
}

func (d *diContainer) Logger(ctx context.Context) LoggerInterface {
	// Инициализируем глобальный логгер если еще не инициализирован
	if err := logger.Init(
		ctx,
		config.AppConfig().Logger.Level(),
		config.AppConfig().Logger.AsJSON(),
		config.AppConfig().Logger.OTLPEnabled(),
		config.AppConfig().Logger.OTLPEndpoint(),
		config.AppConfig().Logger.ServiceName(),
	); err != nil {
		panic(fmt.Sprintf("failed to init logger: %s\n", err.Error()))
	}
	return logger.Logger()
}
