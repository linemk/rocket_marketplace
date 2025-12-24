package config

// LoggerConfig интерфейс конфигурации логгера
type LoggerConfig interface {
	Level() string
	AsJSON() bool
	OTLPEnabled() bool
	OTLPEndpoint() string
	ServiceName() string
}

// OrderHTTPConfig интерфейс конфигурации HTTP сервера Order
type OrderHTTPConfig interface {
	Address() string
}

// MetricsConfig интерфейс конфигурации Prometheus метрик
type MetricsConfig interface {
	Port() int
}

// PostgresConfig интерфейс конфигурации PostgreSQL
type PostgresConfig interface {
	DSN() string
	MigrationsDir() string
}

// InventoryGRPCConfig интерфейс конфигурации gRPC клиента для Inventory
type InventoryGRPCConfig interface {
	Address() string
}

// PaymentGRPCConfig интерфейс конфигурации gRPC клиента для Payment
type PaymentGRPCConfig interface {
	Address() string
}

// KafkaConfig интерфейс конфигурации Kafka
type KafkaConfig interface {
	Brokers() []string
}

// OrderPaidProducerConfig интерфейс конфигурации Kafka producer для OrderPaid
type OrderPaidProducerConfig interface {
	Topic() string
}

// OrderAssembledConsumerConfig интерфейс конфигурации Kafka consumer для OrderAssembled
type OrderAssembledConsumerConfig interface {
	Topic() string
	GroupID() string
}

// IAMGRPCConfig интерфейс конфигурации gRPC клиента для IAM
type IAMGRPCConfig interface {
	Address() string
}
