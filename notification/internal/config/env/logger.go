package env

import (
	"os"
)

const (
	logLevelEnv     = "LOG_LEVEL"
	logAsJSONEnv    = "LOG_AS_JSON"
	otlpEnabledEnv  = "OTLP_ENABLED"
	otlpEndpointEnv = "OTLP_ENDPOINT"
	serviceNameEnv  = "SERVICE_NAME"
)

type loggerConfig struct {
	level        string
	asJSON       bool
	otlpEnabled  bool
	otlpEndpoint string
	serviceName  string
}

// NewLoggerConfig создает конфигурацию логгера из переменных окружения
func NewLoggerConfig() (*loggerConfig, error) {
	level := os.Getenv(logLevelEnv)
	if level == "" {
		level = "info"
	}

	asJSON := os.Getenv(logAsJSONEnv) == "true"

	otlpEnabled := os.Getenv(otlpEnabledEnv) == "true"

	otlpEndpoint := os.Getenv(otlpEndpointEnv)
	if otlpEndpoint == "" {
		otlpEndpoint = "localhost:4317"
	}

	serviceName := os.Getenv(serviceNameEnv)
	if serviceName == "" {
		serviceName = "notification"
	}

	return &loggerConfig{
		level:        level,
		asJSON:       asJSON,
		otlpEnabled:  otlpEnabled,
		otlpEndpoint: otlpEndpoint,
		serviceName:  serviceName,
	}, nil
}

func (c *loggerConfig) Level() string {
	return c.level
}

func (c *loggerConfig) AsJSON() bool {
	return c.asJSON
}

func (c *loggerConfig) OTLPEnabled() bool {
	return c.otlpEnabled
}

func (c *loggerConfig) OTLPEndpoint() string {
	return c.otlpEndpoint
}

func (c *loggerConfig) ServiceName() string {
	return c.serviceName
}
