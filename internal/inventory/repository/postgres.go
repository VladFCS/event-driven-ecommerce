package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
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

func (r *PostgresRepository) ReserveStock(ctx context.Context, reservation domain.StockReservation) (domain.Stock, error) {
	if err := validateStockReservation(reservation); err != nil {
		return domain.Stock{}, err
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.Stock{}, fmt.Errorf("begin reserve stock transaction: %w", err)
	}

	defer func() {
		_ = tx.Rollback(ctx)
	}()

	qtx := r.queries.WithTx(tx)
	stock, err := qtx.GetStockByProductIDForUpdate(ctx, reservation.ProductID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return domain.Stock{}, domain.ErrStockNotFound
		}

		return domain.Stock{}, fmt.Errorf("get stock by id from postgres for update: %w", err)
	}

	if reservation.Quantity > stock.AvailableQuantity {
		return domain.Stock{}, domain.ErrInsufficientStock
	}

	now := time.Now().UTC()
	nowTz := pgtype.Timestamptz{
		Time:  now,
		Valid: true,
	}

	row, err := qtx.ReserveInventoryStock(ctx, inventorydb.ReserveInventoryStockParams{
		ProductID:         reservation.ProductID,
		AvailableQuantity: reservation.Quantity,
		UpdatedAt:         nowTz,
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return domain.Stock{}, domain.ErrInsufficientStock
		}

		return domain.Stock{}, fmt.Errorf("reserve stock in postgres: %w", err)
	}

	if err := qtx.UpsertInventoryReservation(ctx, inventorydb.UpsertInventoryReservationParams{
		ProductID: reservation.ProductID,
		OrderID:   reservation.OrderID,
		Quantity:  reservation.Quantity,
		CreatedAt: nowTz,
		UpdatedAt: nowTz,
	}); err != nil {
		return domain.Stock{}, fmt.Errorf("upsert inventory reservation in postgres: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.Stock{}, fmt.Errorf("commit reserve stock transaction: %w", err)
	}

	return mapDBStock(row)
}

func (r *PostgresRepository) ReleaseStock(ctx context.Context, reservation domain.StockReservation) (domain.Stock, error) {
	if err := validateStockReservation(reservation); err != nil {
		return domain.Stock{}, err
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.Stock{}, fmt.Errorf("begin release stock transaction: %w", err)
	}

	defer func() {
		_ = tx.Rollback(ctx)
	}()

	qtx := r.queries.WithTx(tx)
	if _, err := qtx.GetStockByProductIDForUpdate(ctx, reservation.ProductID); err != nil {
		if err == pgx.ErrNoRows {
			return domain.Stock{}, domain.ErrStockNotFound
		}

		return domain.Stock{}, fmt.Errorf("get stock by id from postgres for update: %w", err)
	}

	reserved, err := qtx.GetReservationByProductIDAndOrderIDForUpdate(ctx, inventorydb.GetReservationByProductIDAndOrderIDForUpdateParams{
		ProductID: reservation.ProductID,
		OrderID:   reservation.OrderID,
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return domain.Stock{}, domain.ErrReservationNotFound
		}

		return domain.Stock{}, fmt.Errorf("get inventory reservation from postgres for update: %w", err)
	}

	if reserved.Quantity < reservation.Quantity {
		return domain.Stock{}, domain.ErrReservationNotFound
	}

	now := time.Now().UTC()
	nowTz := pgtype.Timestamptz{
		Time:  now,
		Valid: true,
	}

	row, err := qtx.ReleaseInventoryStock(ctx, inventorydb.ReleaseInventoryStockParams{
		ProductID:         reservation.ProductID,
		AvailableQuantity: reservation.Quantity,
		UpdatedAt:         nowTz,
	})
	if err != nil {
		return domain.Stock{}, fmt.Errorf("release stock in postgres: %w", err)
	}

	if reserved.Quantity == reservation.Quantity {
		if err := qtx.DeleteInventoryReservation(ctx, inventorydb.DeleteInventoryReservationParams{
			ProductID: reservation.ProductID,
			OrderID:   reservation.OrderID,
		}); err != nil {
			return domain.Stock{}, fmt.Errorf("delete inventory reservation in postgres: %w", err)
		}
	} else {
		if err := qtx.UpdateInventoryReservationQuantity(ctx, inventorydb.UpdateInventoryReservationQuantityParams{
			ProductID: reservation.ProductID,
			OrderID:   reservation.OrderID,
			Quantity:  reserved.Quantity - reservation.Quantity,
			UpdatedAt: nowTz,
		}); err != nil {
			return domain.Stock{}, fmt.Errorf("update inventory reservation quantity in postgres: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.Stock{}, fmt.Errorf("commit release stock transaction: %w", err)
	}

	return mapDBStock(row)
}

func validateStockReservation(reservation domain.StockReservation) error {
	if strings.TrimSpace(reservation.OrderID) == "" ||
		strings.TrimSpace(reservation.ProductID) == "" ||
		reservation.Quantity <= 0 {
		return domain.ErrInvalidStock
	}

	return nil
}

func mapDBStock(row inventorydb.InventoryStock) (domain.Stock, error) {
	return domain.Stock{
		ProductID:         row.ProductID,
		AvailableQuantity: row.AvailableQuantity,
		ReservedQuantity:  row.ReservedQuantity,
		TotalQuantity:     row.TotalQuantity,
	}, nil
}
