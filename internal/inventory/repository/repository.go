package repository

import (
	"context"

	"github.com/vladfc/event-driven-ecommerce-app/internal/inventory/domain"
)

type InventoryRepository interface {
	GetStockByProductID(ctx context.Context, productID string) (domain.Stock, error)
	ReserveStock(ctx context.Context, reservation domain.StockReservation) (domain.Stock, error)
	ReleaseStock(ctx context.Context, reservation domain.StockReservation) (domain.Stock, error)
}
