package config

//go:generate ../../bin/mockery --name LoggerConfig --output ./mocks --outpkg mocks --filename mock_logger_config.go
type LoggerConfig interface {
	Level() string
	AsJSON() bool
	OTLPEnabled() bool
	OTLPEndpoint() string
	ServiceName() string
}

//go:generate ../../bin/mockery --name KafkaConfig --output ./mocks --outpkg mocks --filename mock_kafka_config.go
type KafkaConfig interface {
	Brokers() []string
}

//go:generate ../../bin/mockery --name TelegramBotConfig --output ./mocks --outpkg mocks --filename mock_telegram_bot_config.go
type TelegramBotConfig interface {
	Token() string
	ChatID() string
}

//go:generate ../../bin/mockery --name OrderPaidConsumerConfig --output ./mocks --outpkg mocks --filename mock_order_paid_consumer_config.go
type OrderPaidConsumerConfig interface {
	Topic() string
	GroupID() string
}

//go:generate ../../bin/mockery --name OrderAssembledConsumerConfig --output ./mocks --outpkg mocks --filename mock_order_assembled_consumer_config.go
type OrderAssembledConsumerConfig interface {
	Topic() string
	GroupID() string
}
