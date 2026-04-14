package repository

import (
	"context"

	"github.com/vladfc/event-driven-ecommerce-app/internal/catalog/domain"
)

type ProductRepository interface {
	GetProductByID(ctx context.Context, id string) (domain.Product, error)
	CreateProduct(ctx context.Context, product domain.Product) (domain.Product, error)
}