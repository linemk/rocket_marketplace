package config

import (
	"os"

	"github.com/joho/godotenv"

	"github.com/linemk/rocket-shop/order/internal/config/env"
)

var appConfig *config

type config struct {
	Logger                 LoggerConfig
	OrderHTTP              OrderHTTPConfig
	Metrics                MetricsConfig
	Postgres               PostgresConfig
	InventoryGRPC          InventoryGRPCConfig
	PaymentGRPC            PaymentGRPCConfig
	IAMGRPC                IAMGRPCConfig
	Kafka                  KafkaConfig
	OrderPaidProducer      OrderPaidProducerConfig
	OrderAssembledConsumer OrderAssembledConsumerConfig
}

// Load загружает конфигурацию из переменных окружения
func Load(path ...string) error {
	err := godotenv.Load(path...)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	loggerCfg, err := env.NewLoggerConfig()
	if err != nil {
		return err
	}

	orderHTTPCfg, err := env.NewOrderHTTPConfig()
	if err != nil {
		return err
	}

	metricsCfg, err := env.NewMetricsConfig()
	if err != nil {
		return err
	}

	postgresCfg, err := env.NewPostgresConfig()
	if err != nil {
		return err
	}

	inventoryGRPCCfg, err := env.NewInventoryGRPCConfig()
	if err != nil {
		return err
	}

	paymentGRPCCfg, err := env.NewPaymentGRPCConfig()
	if err != nil {
		return err
	}

	iamGRPCCfg, err := env.NewIAMGRPCConfig()
	if err != nil {
		return err
	}

	kafkaCfg, err := env.NewKafkaConfig()
	if err != nil {
		return err
	}

	orderPaidProducerCfg, err := env.NewOrderPaidProducerConfig()
	if err != nil {
		return err
	}

	orderAssembledConsumerCfg, err := env.NewOrderAssembledConsumerConfig()
	if err != nil {
		return err
	}

	appConfig = &config{
		Logger:                 loggerCfg,
		OrderHTTP:              orderHTTPCfg,
		Metrics:                metricsCfg,
		Postgres:               postgresCfg,
		InventoryGRPC:          inventoryGRPCCfg,
		PaymentGRPC:            paymentGRPCCfg,
		IAMGRPC:                iamGRPCCfg,
		Kafka:                  kafkaCfg,
		OrderPaidProducer:      orderPaidProducerCfg,
		OrderAssembledConsumer: orderAssembledConsumerCfg,
	}

	return nil
}

// AppConfig возвращает глобальную конфигурацию приложения
func AppConfig() *config {
	return appConfig
}
