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

func (s *InventoryService) ReserveStock(ctx context.Context, reservation domain.StockReservation) (domain.Stock, error) {
	if err := validateReservationRequest(reservation); err != nil {
		return domain.Stock{}, err
	}

	return s.repository.ReserveStock(ctx, reservation)
}

func (s *InventoryService) ReleaseStock(ctx context.Context, reservation domain.StockReservation) (domain.Stock, error) {
	if err := validateReservationRequest(reservation); err != nil {
		return domain.Stock{}, err
	}

	return s.repository.ReleaseStock(ctx, reservation)
}

func (s *InventoryService) ReleaseReservationsByOrderID(ctx context.Context, orderID string) error {
	if strings.TrimSpace(orderID) == "" {
		return domain.ErrInvalidStock
	}

	return s.repository.ReleaseReservationsByOrderID(ctx, orderID)
}

func validateReservationRequest(reservation domain.StockReservation) error {
	if strings.TrimSpace(reservation.ProductID) == "" || strings.TrimSpace(reservation.OrderID) == "" || reservation.Quantity <= 0 {
		return domain.ErrInvalidStock
	}

	return nil
}
