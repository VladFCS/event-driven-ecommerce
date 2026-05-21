package repository

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	catalogv1 "github.com/vladfc/event-driven-ecommerce-app/gen/catalog/v1"
	"github.com/vladfc/event-driven-ecommerce-app/internal/catalog/domain"
	catalogdb "github.com/vladfc/event-driven-ecommerce-app/internal/catalog/repository/sqlc"
)

type PostgresRepository struct {
	queries *catalogdb.Queries
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{
		queries: catalogdb.New(pool),
	}
}

func (r *PostgresRepository) GetProductByID(ctx context.Context, productID string) (domain.Product, error) {
	if strings.TrimSpace(productID) == "" {
		return domain.Product{}, domain.ErrInvalidProduct
	}

	row, err := r.queries.GetCatalogProductByID(ctx, productID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return domain.Product{}, domain.ErrProductNotFound
		}

		return domain.Product{}, fmt.Errorf("get catalog product by id from postgres: %w", err)
	}

	return mapDBProduct(row)
}

func (r *PostgresRepository) ListProducts(ctx context.Context, page, pageSize int32) ([]domain.Product, int64, error) {
	if page < 0 || pageSize < 0 {
		return nil, 0, domain.ErrInvalidProduct
	}

	total, err := r.queries.CountCatalogProducts(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("count catalog products from postgres: %w", err)
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

	rows, err := r.queries.ListCatalogProducts(ctx, catalogdb.ListCatalogProductsParams{
		Limit:  clampInt64ToInt32(limit),
		Offset: clampInt64ToInt32(offset),
	})
	if err != nil {
		return nil, 0, fmt.Errorf("list catalog products from postgres: %w", err)
	}

	products, err := mapDBProducts(rows)
	if err != nil {
		return nil, 0, fmt.Errorf("map listed catalog products from postgres: %w", err)
	}

	return products, total, nil
}

func (r *PostgresRepository) CreateProduct(ctx context.Context, product domain.Product) (domain.Product, error) {
	if err := validateProduct(product); err != nil {
		return domain.Product{}, err
	}

	row, err := r.queries.CreateCatalogProduct(ctx, toCreateCatalogProductParams(product))
	if err != nil {
		return domain.Product{}, fmt.Errorf("create catalog product in postgres: %w", err)
	}

	mapped, err := mapDBProduct(row)
	if err != nil {
		return domain.Product{}, fmt.Errorf("map created catalog product from postgres: %w", err)
	}

	return mapped, nil
}

func mapDBProduct(row catalogdb.CatalogProduct) (domain.Product, error) {
	currency, err := mapDBCurrency(row.Currency)
	if err != nil {
		return domain.Product{}, err
	}

	return domain.Product{
		ID:          row.ID,
		Name:        row.Name,
		Description: row.Description,
		PriceCents:  row.PriceCents,
		Currency:    currency,
	}, nil
}

func mapDBProducts(rows []catalogdb.CatalogProduct) ([]domain.Product, error) {
	products := make([]domain.Product, 0, len(rows))
	for _, row := range rows {
		product, err := mapDBProduct(row)
		if err != nil {
			return nil, err
		}

		products = append(products, product)
	}

	return products, nil
}

func mapDBCurrency(value int32) (catalogv1.Currency, error) {
	currency := catalogv1.Currency(value)
	switch currency {
	case catalogv1.Currency_CURRENCY_USD, catalogv1.Currency_CURRENCY_EUR:
		return currency, nil
	default:
		return catalogv1.Currency_CURRENCY_UNSPECIFIED, fmt.Errorf("unknown catalog currency value: %d", value)
	}
}

func clampInt64ToInt32(value int64) int32 {
	if value > math.MaxInt32 {
		return math.MaxInt32
	}
	if value < math.MinInt32 {
		return math.MinInt32
	}

	return int32(value)
}

func validateProduct(product domain.Product) error {
	if strings.TrimSpace(product.ID) == "" ||
		strings.TrimSpace(product.Name) == "" ||
		product.PriceCents <= 0 ||
		product.Currency == catalogv1.Currency_CURRENCY_UNSPECIFIED {
		return domain.ErrInvalidProduct
	}

	return nil
}

func toCreateCatalogProductParams(product domain.Product) catalogdb.CreateCatalogProductParams {
	now := currentTimestamp()

	return catalogdb.CreateCatalogProductParams{
		ID:          product.ID,
		Name:        product.Name,
		Description: product.Description,
		PriceCents:  product.PriceCents,
		Currency:    int32(product.Currency),
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

func currentTimestamp() pgtype.Timestamptz {
	return pgtype.Timestamptz{
		Time:  time.Now().UTC(),
		Valid: true,
	}
}
