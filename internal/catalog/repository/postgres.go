package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
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

func mapDBCurrency(value int32) (catalogv1.Currency, error) {
	currency := catalogv1.Currency(value)
	switch currency {
	case catalogv1.Currency_CURRENCY_USD, catalogv1.Currency_CURRENCY_EUR:
		return currency, nil
	default:
		return catalogv1.Currency_CURRENCY_UNSPECIFIED, fmt.Errorf("unknown catalog currency value: %d", value)
	}
}
