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

func (r *MemoryRepository) ReserveStock(ctx context.Context, reservation domain.StockReservation) (domain.Stock, error) {
	_ = ctx

	r.mu.Lock()
	defer r.mu.Unlock()

	stock, ok := r.inventory[reservation.ProductID]
	if !ok {
		return domain.Stock{}, domain.ErrStockNotFound
	}

	if reservation.Quantity > stock.AvailableQuantity {
		return domain.Stock{}, domain.ErrInsufficientStock
	}

	if _, ok := r.reservations[reservation.ProductID]; !ok {
		r.reservations[reservation.ProductID] = make(map[string]int64)
	}

	stock.AvailableQuantity -= reservation.Quantity
	stock.ReservedQuantity += reservation.Quantity
	r.inventory[reservation.ProductID] = stock
	r.reservations[reservation.ProductID][reservation.OrderID] += reservation.Quantity

	return cloneStock(stock), nil
}

func (r *MemoryRepository) ReleaseStock(ctx context.Context, reservation domain.StockReservation) (domain.Stock, error) {
	_ = ctx

	r.mu.Lock()
	defer r.mu.Unlock()

	stock, ok := r.inventory[reservation.ProductID]
	if !ok {
		return domain.Stock{}, domain.ErrStockNotFound
	}

	productReservations, ok := r.reservations[reservation.ProductID]
	if !ok {
		return domain.Stock{}, domain.ErrReservationNotFound
	}

	reservedForOrder, ok := productReservations[reservation.OrderID]
	if !ok || reservedForOrder < reservation.Quantity {
		return domain.Stock{}, domain.ErrReservationNotFound
	}

	stock.AvailableQuantity += reservation.Quantity
	stock.ReservedQuantity -= reservation.Quantity
	r.inventory[reservation.ProductID] = stock

	if reservedForOrder == reservation.Quantity {
		delete(productReservations, reservation.OrderID)
	} else {
		productReservations[reservation.OrderID] = reservedForOrder - reservation.Quantity
	}

	if len(productReservations) == 0 {
		delete(r.reservations, reservation.ProductID)
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
