package repository

import (
	"context"
	"sync"

	"github.com/vladfc/event-driven-ecommerce-app/internal/catalog/domain"
)

type MemoryRepository struct {
	mu sync.RWMutex
	product map[string]domain.Product
}

func NewMemoryRepository(seed []domain.Product) *MemoryRepository {
	products := make(map[string]domain.Product, len(seed))
	for _, p := range seed {
		products[p.ID] = p
	}
	return &MemoryRepository{
		product: products,
	}
}

func (r *MemoryRepository) GetProductByID(ctx context.Context, id string)	 (domain.Product, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	product, ok := r.product[id]
	if !ok {
		return domain.Product{}, domain.ErrProductNotFound
	}

	return product, nil
}

func (r *MemoryRepository) CreateProduct(ctx context.Context, product domain.Product) (domain.Product, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.product[product.ID] = product
	return product, nil
}

