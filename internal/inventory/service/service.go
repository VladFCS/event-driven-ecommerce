package service

import (
	"context"
	"strings"

	"github.com/vladfc/event-driven-ecommerce-app/internal/inventory/domain"
	"github.com/vladfc/event-driven-ecommerce-app/internal/inventory/repository"
)

type InventoryService struct {
	repository repository.InventoryRepository
}

func NewInventoryService(repository repository.InventoryRepository) *InventoryService {
	return &InventoryService{
		repository: repository,
	}
}

func (s *InventoryService) GetStockByProductID(ctx context.Context, productID string) (domain.Stock, error) {
	if strings.TrimSpace(productID) == "" {
		return domain.Stock{}, domain.ErrInvalidStock
	}
	return s.repository.GetStockByProductID(ctx, productID)
}

func (s *InventoryService) ReserveStock(ctx context.Context, productID string, quantity int64, orderID string) (domain.Stock, error) {
	if err := validateReservationRequest(productID, quantity, orderID); err != nil {
		return domain.Stock{}, err
	}

	return s.repository.ReserveStock(ctx, productID, quantity, orderID)
}

func (s *InventoryService) ReleaseStock(ctx context.Context, productID string, quantity int64, orderID string) (domain.Stock, error) {
	if err := validateReservationRequest(productID, quantity, orderID); err != nil {
		return domain.Stock{}, err
	}

	return s.repository.ReleaseStock(ctx, productID, quantity, orderID)
}

func validateReservationRequest(productID string, quantity int64, orderID string) error {
	if strings.TrimSpace(productID) == "" || strings.TrimSpace(orderID) == "" || quantity <= 0 {
		return domain.ErrInvalidStock
	}

	return nil
}
