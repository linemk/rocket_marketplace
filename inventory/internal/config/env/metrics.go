package env

import (
	"os"
	"strconv"
)

const metricsPortEnv = "INVENTORY_METRICS_PORT"

type metricsConfig struct {
	port int
}

// NewMetricsConfig creates metrics configuration from environment variables
func NewMetricsConfig() (*metricsConfig, error) {
	portStr := os.Getenv(metricsPortEnv)
	if portStr == "" {
		portStr = "9092"
	}

	port, err := strconv.Atoi(portStr)
	if err != nil {
		return nil, err
	}

	return &metricsConfig{
		port: port,
	}, nil
}

func (c *metricsConfig) Port() int {
	return c.port
}
