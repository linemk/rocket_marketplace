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

//go:generate ../../bin/mockery --name OrderPaidConsumerConfig --output ./mocks --outpkg mocks --filename mock_order_paid_consumer_config.go
type OrderPaidConsumerConfig interface {
	Topic() string
	GroupID() string
}

//go:generate ../../bin/mockery --name OrderAssembledProducerConfig --output ./mocks --outpkg mocks --filename mock_order_assembled_producer_config.go
type OrderAssembledProducerConfig interface {
	Topic() string
}
