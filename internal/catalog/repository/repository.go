package repository

import (
	"context"

	"github.com/vladfc/event-driven-ecommerce-app/internal/catalog/domain"
)

type ProductRepository interface {
	GetProductByID(ctx context.Context, productID string) (domain.Product, error)
	ListProducts(ctx context.Context, page, pageSize int32) ([]domain.Product, int64, error)
	CreateProduct(ctx context.Context, product domain.Product) (domain.Product, error)
}
