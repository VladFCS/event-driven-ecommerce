package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	orderv1 "github.com/vladfc/event-driven-ecommerce-app/gen/order/v1"
	"github.com/vladfc/event-driven-ecommerce-app/internal/order/domain"
	"github.com/vladfc/event-driven-ecommerce-app/internal/order/events"
	"github.com/vladfc/event-driven-ecommerce-app/internal/order/repository"
)

type OrderService struct {
	repository repository.OrderRepository
	publisher  events.Publisher
}

func NewOrderService(repository repository.OrderRepository, publisher events.Publisher) *OrderService {
	return &OrderService{
		repository: repository,
		publisher:  publisher,
	}
}

func (s *OrderService) CreateOrder(ctx context.Context, order domain.Order) (domain.Order, error) {
	if strings.TrimSpace(order.CustomerID) == "" || len(order.Items) == 0 {
		return domain.Order{}, domain.ErrInvalidOrder
	}
	if strings.TrimSpace(order.ID) == "" {
		order.ID = newOrderID()
	}

	var totalAmount int64

	for i, item := range order.Items {
		if strings.TrimSpace(item.ProductID) == "" || item.Quantity <= 0 || item.UnitPrice.AmountCents <= 0 {
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

	created, err := s.repository.CreateOrder(ctx, order)
	if err != nil {
		return domain.Order{}, err
	}

	if s.publisher != nil {
		if err := s.publisher.PublishOrderCreated(ctx, toOrderCreatedEvent(created)); err != nil {
			return domain.Order{}, err
		}
	}

	return created, nil
}

func (s *OrderService) GetOrderByID(ctx context.Context, orderID string) (domain.Order, error) {
	if strings.TrimSpace(orderID) == "" {
		return domain.Order{}, domain.ErrInvalidOrderID
	}
	return s.repository.GetOrderByID(ctx, orderID)
}

func (s *OrderService) CancelOrder(ctx context.Context, orderID string) (domain.Order, error) {
	if strings.TrimSpace(orderID) == "" {
		return domain.Order{}, domain.ErrInvalidOrderID
	}

	order, err := s.repository.GetOrderByID(ctx, orderID)
	if err != nil {
		return domain.Order{}, err
	}

	if order.Status == orderv1.OrderStatus_ORDER_STATUS_CANCELLED {
		return order, nil
	}

	order.Status = orderv1.OrderStatus_ORDER_STATUS_CANCELLED
	order.UpdatedAt = time.Now()

	return s.repository.UpdateOrder(ctx, order)
}

func (s *OrderService) ListOrdersByCustomer(ctx context.Context, customerID string, page, pageSize int32) ([]domain.Order, int64, error) {
	if strings.TrimSpace(customerID) == "" {
		return nil, 0, domain.ErrInvalidCustomerID
	}

	return s.repository.ListOrdersByCustomer(ctx, customerID, page, pageSize)
}

func newOrderID() string {
	return fmt.Sprintf("ord-%d", time.Now().UTC().UnixNano())
}

func toOrderCreatedEvent(order domain.Order) events.OrderCreated {
	items := make([]events.OrderCreatedItem, 0, len(order.Items))
	for _, item := range order.Items {
		items = append(items, events.OrderCreatedItem{
			ProductID:   item.ProductID,
			SKU:         item.SKU,
			ProductName: item.ProductName,
			Quantity:    item.Quantity,
			UnitPrice: events.Money{
				Currency:    item.UnitPrice.Currency.String(),
				AmountCents: item.UnitPrice.AmountCents,
			},
			TotalPrice: events.Money{
				Currency:    item.TotalPrice.Currency.String(),
				AmountCents: item.TotalPrice.AmountCents,
			},
		})
	}

	return events.OrderCreated{
		OrderID:    order.ID,
		CustomerID: order.CustomerID,
		Status:     order.Status.String(),
		TotalAmount: events.Money{
			Currency:    order.TotalAmount.Currency.String(),
			AmountCents: order.TotalAmount.AmountCents,
		},
		ShippingAddress: events.Address{
			Country:    order.ShippingAddress.Country,
			City:       order.ShippingAddress.City,
			Street:     order.ShippingAddress.Street,
			PostalCode: order.ShippingAddress.PostalCode,
			House:      order.ShippingAddress.House,
			Apartment:  order.ShippingAddress.Apartment,
		},
		Items: items,
	}
}
