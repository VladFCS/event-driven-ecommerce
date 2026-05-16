package repository

import (
	"context"
	"sort"
	"sync"

	"github.com/vladfc/event-driven-ecommerce-app/internal/catalog/domain"
)

type MemoryRepository struct {
	mu      sync.RWMutex
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

func (r *MemoryRepository) GetProductByID(ctx context.Context, productID string) (domain.Product, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	product, ok := r.product[productID]
	if !ok {
		return domain.Product{}, domain.ErrProductNotFound
	}

	return product, nil
}

func (r *MemoryRepository) ListProducts(ctx context.Context, page, pageSize int32) ([]domain.Product, int64, error) {
	_ = ctx

	r.mu.RLock()
	defer r.mu.RUnlock()

	products := make([]domain.Product, 0, len(r.product))
	for _, product := range r.product {
		products = append(products, product)
	}

	sort.Slice(products, func(i, j int) bool {
		return products[i].ID < products[j].ID
	})

	total := int64(len(products))
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = int32(len(products))
	}

	start := int((page - 1) * pageSize)
	if start >= len(products) {
		return []domain.Product{}, total, nil
	}

	end := start + int(pageSize)
	if end > len(products) {
		end = len(products)
	}

	return append([]domain.Product(nil), products[start:end]...), total, nil
}

func (r *MemoryRepository) CreateProduct(ctx context.Context, product domain.Product) (domain.Product, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.product[product.ID] = product
	return product, nil
}

func (r *MemoryRepository) UpdateProduct(ctx context.Context, productID string, patch domain.ProductPatch) (domain.Product, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	product, ok := r.product[productID]
	if !ok {
		return domain.Product{}, domain.ErrProductNotFound
	}

	if patch.Name != nil {
		product.Name = *patch.Name
	}
	if patch.Description != nil {
		product.Description = *patch.Description
	}
	if patch.PriceCents != nil {
		product.PriceCents = *patch.PriceCents
	}
	if patch.Currency != nil {
		product.Currency = *patch.Currency
	}

	r.product[productID] = product
	return product, nil
}

func (r *MemoryRepository) DeleteProduct(ctx context.Context, productID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.product[productID]; !ok {
		return domain.ErrProductNotFound
	}

	delete(r.product, productID)
	return nil
}
