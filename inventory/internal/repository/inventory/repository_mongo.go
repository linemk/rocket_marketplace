package inventory

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"

	"github.com/linemk/rocket-shop/inventory/internal/entyties/models"
	"github.com/linemk/rocket-shop/platform/pkg/logger"
)

// MongoRepository представляет MongoDB репозиторий для деталей
type MongoRepository struct {
	collection *mongo.Collection
}

// NewMongoRepository создает новый MongoDB репозиторий
func NewMongoRepository(ctx context.Context, db *mongo.Database) *MongoRepository {
	collection := db.Collection("parts")

	// Создаем индексы при инициализации
	indexCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	indexModels := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "uuid", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{{Key: "name", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "category", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "tags", Value: 1}},
		},
	}

	_, err := collection.Indexes().CreateMany(indexCtx, indexModels)
	if err != nil {
		logger.Error(ctx, "Failed to create index", zap.Error(err))
		panic(err)
	}

	return &MongoRepository{
		collection: collection,
	}
}

// GetPart получает деталь по UUID
func (r *MongoRepository) GetPart(ctx context.Context, uuid string) (models.Part, error) {
	var part models.Part
	err := r.collection.FindOne(ctx, bson.M{"uuid": uuid}).Decode(&part)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return models.Part{}, errors.New("part not found")
		}
		return models.Part{}, err
	}

	return part, nil
}

// ListParts возвращает список деталей с применением фильтров
func (r *MongoRepository) ListParts(ctx context.Context, filter models.PartFilter) ([]models.Part, error) {
	// Формируем фильтр для MongoDB
	mongoFilter := bson.M{}

	if len(filter.UUIDs) > 0 {
		mongoFilter["uuid"] = bson.M{"$in": filter.UUIDs}
	}

	if len(filter.Names) > 0 {
		mongoFilter["name"] = bson.M{"$in": filter.Names}
	}

	if len(filter.Categories) > 0 {
		// Конвертируем enum в int32 для MongoDB
		categories := make([]int32, len(filter.Categories))
		for i, cat := range filter.Categories {
			categories[i] = int32(cat)
		}
		mongoFilter["category"] = bson.M{"$in": categories}
	}

	if len(filter.ManufacturerCountries) > 0 {
		mongoFilter["manufacturer.country"] = bson.M{"$in": filter.ManufacturerCountries}
	}

	if len(filter.Tags) > 0 {
		mongoFilter["tags"] = bson.M{"$in": filter.Tags}
	}

	cursor, err := r.collection.Find(ctx, mongoFilter)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := cursor.Close(ctx); err != nil {
			logger.Warn(ctx, "cursor close error", zap.Error(err))
		}
	}()

	var parts []models.Part
	if err := cursor.All(ctx, &parts); err != nil {
		return nil, err
	}

	return parts, nil
}

// CreatePart создает новую деталь
func (r *MongoRepository) CreatePart(ctx context.Context, part models.Part) error {
	part.CreatedAt = time.Now()
	part.UpdatedAt = time.Now()

	_, err := r.collection.InsertOne(ctx, part)
	if err != nil {
		return err
	}

	return nil
}

// UpdatePart обновляет существующую деталь
func (r *MongoRepository) UpdatePart(ctx context.Context, uuid string, part models.Part) error {
	part.UpdatedAt = time.Now()

	result, err := r.collection.UpdateOne(
		ctx,
		bson.M{"uuid": uuid},
		bson.M{"$set": part},
	)
	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return errors.New("part not found")
	}

	return nil
}

// DeletePart удаляет деталь по UUID
func (r *MongoRepository) DeletePart(ctx context.Context, uuid string) error {
	result, err := r.collection.DeleteOne(ctx, bson.M{"uuid": uuid})
	if err != nil {
		return err
	}

	if result.DeletedCount == 0 {
		return errors.New("part not found")
	}

	return nil
}
