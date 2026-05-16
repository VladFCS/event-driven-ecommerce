package service

import (
	"context"
	"strings"

	catalogv1 "github.com/vladfc/event-driven-ecommerce-app/gen/catalog/v1"
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

func (s *CatalogService) GetProductByID(ctx context.Context, productID string) (domain.Product, error) {
	if strings.TrimSpace(productID) == "" {
		return domain.Product{}, domain.ErrInvalidProduct
	}
	return s.repository.GetProductByID(ctx, productID)
}

func (s *CatalogService) ListProducts(ctx context.Context, page, pageSize int32) ([]domain.Product, int64, error) {
	if page < 0 || pageSize < 0 {
		return nil, 0, domain.ErrInvalidProduct
	}

	return s.repository.ListProducts(ctx, page, pageSize)
}

func (s *CatalogService) CreateProduct(ctx context.Context, product domain.Product) (domain.Product, error) {
	if strings.TrimSpace(product.ID) == "" || strings.TrimSpace(product.Name) == "" {
		return domain.Product{}, domain.ErrInvalidProduct
	}
	return s.repository.CreateProduct(ctx, product)
}

func (s *CatalogService) UpdateProduct(ctx context.Context, productID string, patch domain.ProductPatch) (domain.Product, error) {
	if strings.TrimSpace(productID) == "" || patch.Empty() {
		return domain.Product{}, domain.ErrInvalidProduct
	}

	if patch.Name != nil {
		trimmedName := strings.TrimSpace(*patch.Name)
		if trimmedName == "" {
			return domain.Product{}, domain.ErrInvalidProduct
		}
		patch.Name = &trimmedName
	}

	if patch.Description != nil {
		trimmedDescription := strings.TrimSpace(*patch.Description)
		patch.Description = &trimmedDescription
	}

	if patch.PriceCents != nil && *patch.PriceCents <= 0 {
		return domain.Product{}, domain.ErrInvalidProduct
	}

	if patch.Currency != nil && *patch.Currency == catalogv1.Currency_CURRENCY_UNSPECIFIED {
		return domain.Product{}, domain.ErrInvalidProduct
	}

	return s.repository.UpdateProduct(ctx, productID, patch)
}

func (s *CatalogService) DeleteProduct(ctx context.Context, productID string) error {
	if strings.TrimSpace(productID) == "" {
		return domain.ErrInvalidProduct
	}

	return s.repository.DeleteProduct(ctx, productID)
}
