package service

import (
	"context"
	"strings"

	"github.com/vladfc/event-driven-ecommerce-app/internal/catalog/domain"
	"github.com/vladfc/event-driven-ecommerce-app/internal/catalog/repository"
)

type CatalogService struct {
	repository repository.ProductRepository
}

func NewCatalogService(repository repository.ProductRepository) *CatalogService {
	return &CatalogService{
		repository: repository,
	}
}

func (s *CatalogService) GetProductByID(ctx context.Context, id string) (domain.Product, error) {
	if strings.TrimSpace(id) == "" {
		return domain.Product{}, domain.ErrInvalidProduct
	}
	return s.repository.GetProductByID(ctx, id)
}

func (s *CatalogService) CreateProduct(ctx context.Context, product domain.Product) (domain.Product, error) {
	if strings.TrimSpace(product.ID) == "" || strings.TrimSpace(product.Name) == "" {
		return domain.Product{}, domain.ErrInvalidProduct
	}
	return s.repository.CreateProduct(ctx, product)
}