package config

import (
	"os"

	"github.com/joho/godotenv"

	"github.com/linemk/rocket-shop/assembly/internal/config/env"
)

var appConfig *config

type config struct {
	Logger                 LoggerConfig
	Metrics                MetricsConfig
	Kafka                  KafkaConfig
	OrderPaidConsumer      OrderPaidConsumerConfig
	OrderAssembledProducer OrderAssembledProducerConfig
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

	metricsCfg, err := env.NewMetricsConfig()
	if err != nil {
		return err
	}

	kafkaCfg, err := env.NewKafkaConfig()
	if err != nil {
		return err
	}

	orderPaidConsumerCfg, err := env.NewOrderPaidConsumerConfig()
	if err != nil {
		return err
	}

	orderAssembledProducerCfg, err := env.NewOrderAssembledProducerConfig()
	if err != nil {
		return err
	}

	appConfig = &config{
		Logger:                 loggerCfg,
		Metrics:                metricsCfg,
		Kafka:                  kafkaCfg,
		OrderPaidConsumer:      orderPaidConsumerCfg,
		OrderAssembledProducer: orderAssembledProducerCfg,
	}

	return nil
}

// AppConfig возвращает глобальную конфигурацию приложения
func AppConfig() *config {
	return appConfig
}
