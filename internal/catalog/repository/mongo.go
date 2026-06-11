package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/vladfc/event-driven-ecommerce-app/internal/catalog/domain"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type MongoRepository struct {
	collection *mongo.Collection
}

type productDocument struct {
	ID          string    `bson:"_id"`
	Name        string    `bson:"name"`
	Description string    `bson:"description"`
	PriceCents  int64     `bson:"price_cents"`
	Currency    int32     `bson:"currency"`
	CreatedAt   time.Time `bson:"created_at"`
	UpdatedAt   time.Time `bson:"updated_at"`
}

func NewMongoRepository(collection *mongo.Collection) *MongoRepository {
	return &MongoRepository{
		collection: collection,
	}
}

func (r *MongoRepository) GetProductByID(ctx context.Context, productID string) (domain.Product, error) {
	if strings.TrimSpace(productID) == "" {
		return domain.Product{}, domain.ErrInvalidProduct
	}

	var doc productDocument
	err := r.collection.FindOne(ctx, bson.M{"_id": productID}).Decode(&doc)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return domain.Product{}, domain.ErrProductNotFound
		}

		return domain.Product{}, fmt.Errorf("get catalog product by id from mongo: %w", err)
	}

	return mapMongoProduct(doc)
}

func (r *MongoRepository) ListProducts(ctx context.Context, page, pageSize int32) ([]domain.Product, int64, error) {
	if page < 0 || pageSize < 0 {
		return nil, 0, domain.ErrInvalidProduct
	}

	filter := bson.D{}
	total, err := r.collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("count catalog products from mongo: %w", err)
	}

	if total == 0 {
		return []domain.Product{}, 0, nil
	}

	if page <= 0 {
		page = 1
	}

	limit := int64(pageSize)
	if pageSize <= 0 {
		limit = total
	}

	offset := int64(page-1) * limit
	if offset >= total {
		return []domain.Product{}, total, nil
	}

	findOptions := options.Find().
		SetSort(bson.D{{Key: "_id", Value: 1}}).
		SetSkip(offset).
		SetLimit(limit)

	cursor, err := r.collection.Find(ctx, filter, findOptions)
	if err != nil {
		return nil, 0, fmt.Errorf("list catalog products from mongo: %w", err)
	}
	defer cursor.Close(ctx)

	var docs []productDocument
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, 0, fmt.Errorf("decode listed catalog products from mongo: %w", err)
	}

	products, err := mapMongoProducts(docs)
	if err != nil {
		return nil, 0, fmt.Errorf("map listed catalog products from mongo: %w", err)
	}

	return products, total, nil
}

func (r *MongoRepository) CreateProduct(ctx context.Context, product domain.Product) (domain.Product, error) {
	if err := validateProduct(product); err != nil {
		return domain.Product{}, err
	}

	now := time.Now().UTC()

	update := bson.M{
		"$set": bson.M{
			"name":        product.Name,
			"description": product.Description,
			"price_cents": product.PriceCents,
			"currency":    int32(product.Currency),
			"updated_at":  now,
		},
		"$setOnInsert": bson.M{
			"_id":        product.ID,
			"created_at": now,
		},
	}

	var doc productDocument
	err := r.collection.FindOneAndUpdate(
		ctx,
		bson.M{"_id": product.ID},
		update,
		options.FindOneAndUpdate().
			SetReturnDocument(options.After).
			SetUpsert(true),
	).Decode(&doc)
	if err != nil {
		return domain.Product{}, fmt.Errorf("upsert catalog product in mongo: %w", err)
	}

	return mapMongoProduct(doc)
}

func (r *MongoRepository) UpdateProduct(ctx context.Context, productID string, patch domain.ProductPatch) (domain.Product, error) {
	if strings.TrimSpace(productID) == "" || patch.Empty() {
		return domain.Product{}, domain.ErrInvalidProduct
	}

	if err := validateProductPatch(patch); err != nil {
		return domain.Product{}, err
	}

	set := bson.M{
		"updated_at": time.Now().UTC(),
	}
	if patch.Name != nil {
		set["name"] = strings.TrimSpace(*patch.Name)
	}
	if patch.Description != nil {
		set["description"] = strings.TrimSpace(*patch.Description)
	}
	if patch.PriceCents != nil {
		set["price_cents"] = *patch.PriceCents
	}
	if patch.Currency != nil {
		set["currency"] = int32(*patch.Currency)
	}

	var doc productDocument
	err := r.collection.FindOneAndUpdate(
		ctx,
		bson.M{"_id": productID},
		bson.M{"$set": set},
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	).Decode(&doc)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return domain.Product{}, domain.ErrProductNotFound
		}

		return domain.Product{}, fmt.Errorf("update catalog product in mongo: %w", err)
	}

	return mapMongoProduct(doc)
}

func (r *MongoRepository) DeleteProduct(ctx context.Context, productID string) error {
	if strings.TrimSpace(productID) == "" {
		return domain.ErrInvalidProduct
	}

	result, err := r.collection.DeleteOne(ctx, bson.M{"_id": productID})
	if err != nil {
		return fmt.Errorf("delete catalog product from mongo: %w", err)
	}
	if result.DeletedCount == 0 {
		return domain.ErrProductNotFound
	}

	return nil
}

func mapMongoProduct(doc productDocument) (domain.Product, error) {
	currency, err := mapDBCurrency(doc.Currency)
	if err != nil {
		return domain.Product{}, err
	}

	return domain.Product{
		ID:          doc.ID,
		Name:        doc.Name,
		Description: doc.Description,
		PriceCents:  doc.PriceCents,
		Currency:    currency,
	}, nil
}

func mapMongoProducts(docs []productDocument) ([]domain.Product, error) {
	products := make([]domain.Product, 0, len(docs))
	for _, doc := range docs {
		product, err := mapMongoProduct(doc)
		if err != nil {
			return nil, err
		}

		products = append(products, product)
	}

	return products, nil
}
