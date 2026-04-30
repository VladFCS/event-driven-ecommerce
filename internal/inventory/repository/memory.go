package repository

import (
	"context"
	"fmt"
	"sync"

	"github.com/vladfc/event-driven-ecommerce-app/internal/inventory/domain"
)

type MemoryRepository struct {
	mu 			sync.RWMutex
	inventory map[string]domain.Stock
}

func NewMemoryRepository(seed []domain.Stock) *MemoryRepository {
	inventory := make(map[string]domain.Stock, len(seed))
	for _, stock := range seed {
		inventory[stock.ProductID] = stock
	}
	return &MemoryRepository{
		inventory: inventory,
	}
}

func (r *MemoryRepository) GetStockByProductID(ctx context.Context, productID string) (domain.Stock, error) {
	_ = ctx

	r.mu.RLock()
	defer r.mu.RUnlock()

	stock, ok := r.inventory[productID]
	if !ok {
		return domain.Stock{}, fmt.Errorf("stock not found for product ID: %s", productID)
	}
	return stock, nil
}
