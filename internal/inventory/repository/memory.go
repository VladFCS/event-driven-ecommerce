package repository

import (
	"context"
	"strings"
	"sync"

	"github.com/vladfc/event-driven-ecommerce-app/internal/inventory/domain"
)

type MemoryRepository struct {
	mu           sync.RWMutex
	inventory    map[string]domain.Stock
	reservations map[string]map[string]int64
}

func NewMemoryRepository(seed []domain.Stock) *MemoryRepository {
	inventory := make(map[string]domain.Stock, len(seed))
	for _, stock := range seed {
		inventory[stock.ProductID] = cloneStock(stock)
	}
	return &MemoryRepository{
		inventory:    inventory,
		reservations: make(map[string]map[string]int64),
	}
}

func (r *MemoryRepository) GetStockByProductID(ctx context.Context, productID string) (domain.Stock, error) {
	_ = ctx

	r.mu.RLock()
	defer r.mu.RUnlock()

	stock, ok := r.inventory[productID]
	if !ok {
		return domain.Stock{}, domain.ErrStockNotFound
	}

	return cloneStock(stock), nil
}

func (r *MemoryRepository) ReserveStock(ctx context.Context, productID string, quantity int64, orderID string) (domain.Stock, error) {
	_ = ctx

	r.mu.Lock()
	defer r.mu.Unlock()

	stock, ok := r.inventory[productID]
	if !ok {
		return domain.Stock{}, domain.ErrStockNotFound
	}

	if quantity > stock.AvailableQuantity {
		return domain.Stock{}, domain.ErrInsufficientStock
	}

	if _, ok := r.reservations[productID]; !ok {
		r.reservations[productID] = make(map[string]int64)
	}

	stock.AvailableQuantity -= quantity
	stock.ReservedQuantity += quantity
	r.inventory[productID] = stock
	r.reservations[productID][orderID] += quantity

	return cloneStock(stock), nil
}

func (r *MemoryRepository) ReleaseStock(ctx context.Context, productID string, quantity int64, orderID string) (domain.Stock, error) {
	_ = ctx

	r.mu.Lock()
	defer r.mu.Unlock()

	stock, ok := r.inventory[productID]
	if !ok {
		return domain.Stock{}, domain.ErrStockNotFound
	}

	productReservations, ok := r.reservations[productID]
	if !ok {
		return domain.Stock{}, domain.ErrReservationNotFound
	}

	reservedForOrder, ok := productReservations[orderID]
	if !ok || reservedForOrder < quantity {
		return domain.Stock{}, domain.ErrReservationNotFound
	}

	stock.AvailableQuantity += quantity
	stock.ReservedQuantity -= quantity
	r.inventory[productID] = stock

	if reservedForOrder == quantity {
		delete(productReservations, orderID)
	} else {
		productReservations[orderID] = reservedForOrder - quantity
	}

	if len(productReservations) == 0 {
		delete(r.reservations, productID)
	}

	return cloneStock(stock), nil
}

func cloneStock(stock domain.Stock) domain.Stock {
	return domain.Stock{
		ProductID:         strings.Clone(stock.ProductID),
		AvailableQuantity: stock.AvailableQuantity,
		ReservedQuantity:  stock.ReservedQuantity,
		TotalQuantity:     stock.TotalQuantity,
	}
}
