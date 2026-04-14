package service

import (
	"context"
	"time"

	orderv1 "github.com/vladfc/event-driven-ecommerce-app/gen/order/v1"
	"github.com/vladfc/event-driven-ecommerce-app/internal/order/domain"
	"github.com/vladfc/event-driven-ecommerce-app/internal/order/repository"
)


type OrderService struct {
	repository repository.OrderRepository
}

func NewOrderService(repository repository.OrderRepository) *OrderService {
	return &OrderService{
		repository: repository,			
	}
}

func (s *OrderService) CreateOrder(ctx context.Context, order domain.Order) (domain.Order, error) {
	if order.ID == "" || order.CustomerID == "" || len(order.Items) == 0 {
		return domain.Order{}, domain.ErrInvalidOrder
	}

	var totalAmount int64

	for i, item := range order.Items {
		if item.ProductID == "" || item.Quantity <= 0 || item.UnitPrice.AmountCents <= 0 {
			return domain.Order{}, domain.ErrInvalidOrder
		}

		totalPriceCents := int64(item.Quantity) * item.UnitPrice.AmountCents
		order.Items[i].TotalPrice = domain.Money{
			Currency:    item.UnitPrice.Currency,
			AmountCents: totalPriceCents,
		}

		totalAmount += totalPriceCents
	}

	order.TotalAmount = domain.Money{
		Currency:    order.Items[0].UnitPrice.Currency,
		AmountCents: totalAmount,
	}

	order.Status = orderv1.OrderStatus_ORDER_STATUS_AWAITING_PAYMENT
	order.CreatedAt = time.Now()
	order.UpdatedAt = order.CreatedAt

	return s.repository.CreateOrder(ctx, order)
}
