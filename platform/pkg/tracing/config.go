package tracing

import (
	"os"
	"strconv"
)

const (
	defaultOTLPEndpoint = "localhost:4317"
	defaultEnvironment  = "development"
	defaultSampleRate   = 1.0
)

// Config содержит конфигурацию для OpenTelemetry трейсинга
type Config struct {
	Enabled     bool
	Endpoint    string
	ServiceName string
	Environment string
	SampleRate  float64
}

// NewConfigFromEnv создаёт конфигурацию из переменных окружения
func NewConfigFromEnv() Config {
	enabled := os.Getenv("OTLP_ENABLED") == "true"

	endpoint := os.Getenv("OTLP_ENDPOINT")
	if endpoint == "" {
		endpoint = defaultOTLPEndpoint
	}

	serviceName := os.Getenv("SERVICE_NAME")

	environment := os.Getenv("ENVIRONMENT")
	if environment == "" {
		environment = defaultEnvironment
	}

	sampleRate := defaultSampleRate
	if rate := os.Getenv("OTEL_SAMPLE_RATE"); rate != "" {
		if parsed, err := strconv.ParseFloat(rate, 64); err == nil {
			sampleRate = parsed
		}
	}

	return Config{
		Enabled:     enabled,
		Endpoint:    endpoint,
		ServiceName: serviceName,
		Environment: environment,
		SampleRate:  sampleRate,
	}
}
