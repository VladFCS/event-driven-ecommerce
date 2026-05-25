package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vladfc/event-driven-ecommerce-app/internal/inventory/domain"
	inventorydb "github.com/vladfc/event-driven-ecommerce-app/internal/inventory/repository/sqlc"
)

type PostgresRepository struct {
	pool    *pgxpool.Pool
	queries *inventorydb.Queries
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{
		pool:    pool,
		queries: inventorydb.New(pool),
	}
}

func (r *PostgresRepository) GetStockByProductID(ctx context.Context, productID string) (domain.Stock, error) {
	if strings.TrimSpace(productID) == "" {
		return domain.Stock{}, domain.ErrInvalidProductID
	}
	row, err := r.queries.GetStockByProductID(ctx, productID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return domain.Stock{}, domain.ErrStockNotFound
		}

		return domain.Stock{}, fmt.Errorf("get stock by id from postgres: %w", err)
	}

	return mapDBStock(row)
}

func mapDBStock(row inventorydb.InventoryStock) (domain.Stock, error) {
	return domain.Stock{
		ProductID:         row.ProductID,
		AvailableQuantity: row.AvailableQuantity,
		ReservedQuantity:  row.ReservedQuantity,
		TotalQuantity:     row.TotalQuantity,
	}, nil
}
