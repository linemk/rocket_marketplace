//go:build integration

package integration

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/linemk/rocket-shop/platform/pkg/logger"
)

const testsTimeout = 5 * time.Minute

var (
	env *TestEnvironment

	suiteCtx    context.Context
	suiteCancel context.CancelFunc
)

func TestIntegration(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Inventory Service Integration Test Suite")
}

var _ = BeforeSuite(func() {
	suiteCtx, suiteCancel = context.WithTimeout(context.Background(), testsTimeout)

	err := logger.Init(suiteCtx, "debug", true, false, "", "inventory-integration")
	if err != nil {
		panic(fmt.Sprintf("не удалось инициализировать логгер: %v", err))
	}

	// Устанавливаем переменные окружения для MongoDB
	_ = os.Setenv("MONGO_INITDB_ROOT_USERNAME", "root")
	_ = os.Setenv("MONGO_INITDB_ROOT_PASSWORD", "root")
	_ = os.Setenv("MONGO_IMAGE_NAME", "mongo:8.0")
	_ = os.Setenv("MONGO_DATABASE", "inventory-test")
	_ = os.Setenv("GRPC_PORT", "50051")

	logger.Info(suiteCtx, "Запуск тестового окружения...")
	env = setupTestEnvironment(suiteCtx)
})

var _ = AfterSuite(func() {
	logger.Info(suiteCtx, "Завершение набора тестов")
	if env != nil {
		teardownTestEnvironment(suiteCtx, env)
	}
	suiteCancel()
})
