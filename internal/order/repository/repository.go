package repository

import (
	"context"

	"github.com/vladfc/event-driven-ecommerce-app/internal/order/domain"
)

type OrderRepository interface {
	CreateOrder(ctx context.Context, order domain.Order) (domain.Order, error)
	GetOrderByID(ctx context.Context, id string) (domain.Order, error)
	ListOrdersByCustomer(ctx context.Context, customerID string, page, pageSize int32) ([]domain.Order, int64, error)
	UpdateOrder(ctx context.Context, order domain.Order) (domain.Order, error)
}
