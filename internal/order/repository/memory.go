package repository

import (
	"context"
	"sort"
	"strings"
	"sync"

	"github.com/vladfc/event-driven-ecommerce-app/internal/order/domain"
)

type MemoryRepository struct {
	mu     sync.RWMutex
	orders map[string]domain.Order
}

func NewMemoryRepository(seed []domain.Order) *MemoryRepository {
	orders := make(map[string]domain.Order, len(seed))
	for _, order := range seed {
		orders[order.ID] = cloneOrder(order)
	}

	return &MemoryRepository{
		orders: orders,
	}
}

func (r *MemoryRepository) CreateOrder(ctx context.Context, order domain.Order) (domain.Order, error) {
	_ = ctx

	r.mu.Lock()
	defer r.mu.Unlock()

	if strings.TrimSpace(order.ID) == "" || strings.TrimSpace(order.CustomerID) == "" {
		return domain.Order{}, domain.ErrInvalidOrder
	}

	if _, exists := r.orders[order.ID]; exists {
		return domain.Order{}, domain.ErrOrderAlreadyExists
	}

	cloned := cloneOrder(order)
	r.orders[order.ID] = cloned

	return cloneOrder(cloned), nil
}

func (r *MemoryRepository) GetOrderByID(ctx context.Context, id string) (domain.Order, error) {
	_ = ctx

	r.mu.RLock()
	defer r.mu.RUnlock()

	order, ok := r.orders[id]
	if !ok {
		return domain.Order{}, domain.ErrOrderNotFound
	}

	return cloneOrder(order), nil
}

func (r *MemoryRepository) ListOrdersByCustomer(ctx context.Context, customerID string, page, pageSize int32) ([]domain.Order, int64, error) {
	_ = ctx

	r.mu.RLock()
	defer r.mu.RUnlock()

	filtered := make([]domain.Order, 0)
	for _, order := range r.orders {
		if order.CustomerID == customerID {
			filtered = append(filtered, cloneOrder(order))
		}
	}

	sort.Slice(filtered, func(i, j int) bool {
		if filtered[i].CreatedAt.Equal(filtered[j].CreatedAt) {
			return filtered[i].ID < filtered[j].ID
		}

		return filtered[i].CreatedAt.Before(filtered[j].CreatedAt)
	})

	total := int64(len(filtered))
	if pageSize <= 0 {
		pageSize = int32(len(filtered))
	}
	if page <= 0 {
		page = 1
	}

	start := int((page - 1) * pageSize)
	if start >= len(filtered) {
		return []domain.Order{}, total, nil
	}

	end := start + int(pageSize)
	if end > len(filtered) {
		end = len(filtered)
	}

	return filtered[start:end], total, nil
}

func (r *MemoryRepository) UpdateOrder(ctx context.Context, order domain.Order) (domain.Order, error) {
	_ = ctx

	r.mu.Lock()
	defer r.mu.Unlock()

	if strings.TrimSpace(order.ID) == "" || strings.TrimSpace(order.CustomerID) == "" {
		return domain.Order{}, domain.ErrInvalidOrder
	}

	if _, exists := r.orders[order.ID]; !exists {
		return domain.Order{}, domain.ErrOrderNotFound
	}

	cloned := cloneOrder(order)
	r.orders[order.ID] = cloned

	return cloneOrder(cloned), nil
}

func cloneOrder(order domain.Order) domain.Order {
	cloned := order
	cloned.Items = append([]domain.OrderItem(nil), order.Items...)
	return cloned
}
