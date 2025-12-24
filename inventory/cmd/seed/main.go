package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"

	"github.com/linemk/rocket-shop/inventory/internal/entyties/models"
	"github.com/linemk/rocket-shop/platform/pkg/logger"
	inventory_v1 "github.com/linemk/rocket-shop/shared/pkg/proto/inventory/v1"
)

func main() {
	ctx := context.Background()

	// Инициализируем логгер
	if err := logger.Init(ctx, "info", false, false, "", "inventory-seed"); err != nil {
		panic(err)
	}
	defer func() {
		_ = logger.Close(ctx) //nolint:gosec // best-effort shutdown
		_ = logger.Sync()     //nolint:gosec // best-effort shutdown
	}()

	if err := run(ctx); err != nil {
		logger.Fatal(ctx, "Ошибка выполнения", zap.Error(err))
	}
}

func run(ctx context.Context) error {
	// Загружаем .env файл (игнорируем ошибку, т.к. файл может отсутствовать в CI)
	//nolint:gosec,errcheck
	_ = godotenv.Load("deploy/compose/inventory/.env")

	// Получаем параметры подключения из окружения
	mongoUser := os.Getenv("INVENTORY_MONGO_USER")
	if mongoUser == "" {
		mongoUser = "inventory_user"
	}

	mongoPassword := os.Getenv("INVENTORY_MONGO_PASSWORD")
	if mongoPassword == "" {
		mongoPassword = "inventory_password"
	}

	mongoPort := os.Getenv("INVENTORY_MONGO_PORT")
	if mongoPort == "" {
		mongoPort = "27017"
	}

	mongoDatabase := os.Getenv("INVENTORY_MONGO_DB")
	if mongoDatabase == "" {
		mongoDatabase = "inventory_db"
	}

	// Формируем URI для подключения к localhost (для dev окружения)
	mongoURI := fmt.Sprintf("mongodb://%s:%s@localhost:%s/%s?authSource=admin",
		mongoUser, mongoPassword, mongoPort, mongoDatabase)

	opCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Подключаемся к MongoDB
	client, err := mongo.Connect(opCtx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		return fmt.Errorf("не удалось подключиться к MongoDB: %w", err)
	}
	defer func() {
		disconnectCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		if err := client.Disconnect(disconnectCtx); err != nil {
			logger.Error(ctx, "Ошибка при отключении от MongoDB", zap.Error(err))
		}
	}()

	// Проверяем подключение
	if err := client.Ping(opCtx, nil); err != nil {
		return fmt.Errorf("не удалось проверить подключение к MongoDB: %w", err)
	}

	collection := client.Database(mongoDatabase).Collection("parts")

	// Создаем тестовые детали
	parts := generateParts(10)

	logger.Info(ctx, "🌱 Заполняем базу данных тестовыми деталями", zap.Int("count", len(parts)))

	for i, part := range parts {
		_, err := collection.InsertOne(opCtx, part)
		if err != nil {
			logger.Error(ctx, "⚠️  Ошибка при вставке детали", zap.Int("index", i+1), zap.Error(err))
			continue
		}
		logger.Info(ctx, "✅ Создана деталь", zap.Int("index", i+1), zap.Int("total", len(parts)), zap.String("name", part.Name), zap.String("uuid", part.UUID))
	}

	logger.Info(ctx, "🎉 База данных успешно заполнена!")
	return nil
}

func generateParts(count int) []models.Part {
	parts := make([]models.Part, 0, count)
	now := time.Now()

	categories := []inventory_v1.Category{
		inventory_v1.Category_CATEGORY_ENGINE,
		inventory_v1.Category_CATEGORY_FUEL,
		inventory_v1.Category_CATEGORY_PORTHOLE,
		inventory_v1.Category_CATEGORY_WING,
	}

	partNames := []string{
		"Quantum Engine X-3000",
		"Fusion Reactor Core",
		"Titanium Wing Panel",
		"Reinforced Porthole",
		"Plasma Fuel Tank",
		"Ion Thruster Assembly",
		"Carbon Fiber Wing",
		"Armored Viewport",
		"Hyperdrive Fuel Cell",
		"Warp Drive Engine",
	}

	for i := 0; i < count; i++ {
		category := categories[i%len(categories)]
		name := partNames[i%len(partNames)]
		if i >= len(partNames) {
			name = fmt.Sprintf("%s Mark-%d", name, i/len(partNames)+1)
		}

		parts = append(parts, models.Part{
			UUID:          uuid.New().String(),
			Name:          name,
			Description:   gofakeit.Sentence(15),
			Price:         gofakeit.Float64Range(100, 50000),
			StockQuantity: int64(gofakeit.IntRange(10, 500)),
			Category:      category,
			Dimensions: &models.Dimensions{
				Length: gofakeit.Float64Range(10, 300),
				Width:  gofakeit.Float64Range(10, 200),
				Height: gofakeit.Float64Range(10, 150),
				Weight: gofakeit.Float64Range(5, 1000),
			},
			Manufacturer: &models.Manufacturer{
				Name:    gofakeit.Company(),
				Country: gofakeit.Country(),
				Website: gofakeit.URL(),
			},
			Tags: []string{
				gofakeit.Word(),
				gofakeit.Word(),
				"spaceship",
			},
			Metadata:  map[string]interface{}{"generated": true, "version": "1.0"},
			CreatedAt: now,
			UpdatedAt: now,
		})
	}

	return parts
}
