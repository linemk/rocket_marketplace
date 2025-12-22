package config

import "time"

// LoggerConfig интерфейс конфигурации логгера
type LoggerConfig interface {
	Level() string
	AsJSON() bool
	OTLPEnabled() bool
	OTLPEndpoint() string
	ServiceName() string
}

// PostgresConfig интерфейс конфигурации PostgreSQL
type PostgresConfig interface {
	DSN() string
	MigrationsDir() string
}

// RedisConfig интерфейс конфигурации Redis
type RedisConfig interface {
	Addr() string
	Password() string
	DB() int
	DialTimeout() time.Duration
	ReadTimeout() time.Duration
	WriteTimeout() time.Duration
	PoolSize() int
}

// GRPCConfig интерфейс конфигурации gRPC сервера
type GRPCConfig interface {
	Address() string
}

// SessionConfig интерфейс конфигурации для сессий
type SessionConfig interface {
	TTL() time.Duration
}
