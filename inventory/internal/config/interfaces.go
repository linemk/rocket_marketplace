package config

// LoggerConfig интерфейс конфигурации логгера
type LoggerConfig interface {
	Level() string
	AsJSON() bool
	OTLPEnabled() bool
	OTLPEndpoint() string
	ServiceName() string
}

// InventoryGRPCConfig интерфейс конфигурации gRPC сервера Inventory
type InventoryGRPCConfig interface {
	Address() string
}

// IAMGRPCConfig интерфейс конфигурации gRPC клиента IAM
type IAMGRPCConfig interface {
	Address() string
}

// MongoConfig интерфейс конфигурации MongoDB
type MongoConfig interface {
	URI() string
	DatabaseName() string
}
