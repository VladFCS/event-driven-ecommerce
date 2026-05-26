package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
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
	order = normalizeOrderInput(order)

	if err := validateCreateOrder(order); err != nil {
		return domain.Order{}, domain.ErrInvalidOrder
	}
	if order.IdempotencyKey != "" {
		existing, err := s.repository.GetOrderByIdempotencyKey(ctx, order.IdempotencyKey)
		switch {
		case err == nil:
			if !ordersMatchForIdempotency(existing, order) {
				return domain.Order{}, domain.ErrIdempotencyKeyConflict
			}
			return existing, nil
		case !errors.Is(err, domain.ErrOrderNotFound):
			return domain.Order{}, err
		}
	}
	if strings.TrimSpace(order.ID) == "" {
		order.ID = newOrderID()
	}

	var totalAmount int64
	totalCurrency := orderv1.Currency_CURRENCY_UNSPECIFIED

	for i, item := range order.Items {
		item.ProductID = strings.TrimSpace(item.ProductID)
		order.Items[i].ProductID = item.ProductID

		if item.ProductID == "" ||
			item.Quantity <= 0 ||
			item.UnitPrice.AmountCents <= 0 ||
			item.UnitPrice.Currency == orderv1.Currency_CURRENCY_UNSPECIFIED {
			return domain.Order{}, domain.ErrInvalidOrder
		}
		if totalCurrency == orderv1.Currency_CURRENCY_UNSPECIFIED {
			totalCurrency = item.UnitPrice.Currency
		} else if item.UnitPrice.Currency != totalCurrency {
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
		Currency:    totalCurrency,
		AmountCents: totalAmount,
	}

	order.Status = orderv1.OrderStatus_ORDER_STATUS_PENDING
	order.CancelReason = ""
	order.CreatedAt = time.Now()
	order.UpdatedAt = order.CreatedAt

	// Order creation succeeds once the order and its outbox event are committed together.
	// Kafka delivery is handled asynchronously by the outbox dispatcher.
	created, err := s.repository.CreateOrder(ctx, order)
	if err == nil {
		return created, nil
	}

	if order.IdempotencyKey != "" && errors.Is(err, domain.ErrIdempotencyKeyAlreadyExists) {
		existing, getErr := s.repository.GetOrderByIdempotencyKey(ctx, order.IdempotencyKey)
		if getErr != nil {
			return domain.Order{}, getErr
		}
		if !ordersMatchForIdempotency(existing, order) {
			return domain.Order{}, domain.ErrIdempotencyKeyConflict
		}
		return existing, nil
	}

	return domain.Order{}, err
}

func (s *OrderService) GetOrderByID(ctx context.Context, orderID string) (domain.Order, error) {
	if strings.TrimSpace(orderID) == "" {
		return domain.Order{}, domain.ErrInvalidOrderID
	}
	return s.repository.GetOrderByID(ctx, orderID)
}

func (s *OrderService) CancelOrder(ctx context.Context, orderID string, reason string) (domain.Order, error) {
	if strings.TrimSpace(orderID) == "" {
		return domain.Order{}, domain.ErrInvalidOrderID
	}
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return domain.Order{}, domain.ErrInvalidOrder
	}

	order, err := s.repository.GetOrderByID(ctx, orderID)
	if err != nil {
		return domain.Order{}, err
	}

	if order.Status == orderv1.OrderStatus_ORDER_STATUS_CANCELLED {
		return order, nil
	}

	if !canCancelOrder(order.Status) {
		return domain.Order{}, domain.ErrOrderCannotBeCancelled
	}

	order = cancelOrderWithReason(order, reason)

	return s.repository.UpdateOrder(ctx, order)
}

func (s *OrderService) MarkInventoryReserved(ctx context.Context, orderID string) (domain.Order, error) {
	return s.transitionOrder(ctx, orderID, func(order domain.Order) (domain.Order, bool, error) {
		switch order.Status {
		case orderv1.OrderStatus_ORDER_STATUS_AWAITING_PAYMENT, orderv1.OrderStatus_ORDER_STATUS_CONFIRMED:
			return order, false, nil
		case orderv1.OrderStatus_ORDER_STATUS_CANCELLED:
			return order, false, nil
		case orderv1.OrderStatus_ORDER_STATUS_PENDING:
			order.Status = orderv1.OrderStatus_ORDER_STATUS_AWAITING_PAYMENT
			order.CancelReason = ""
			order.UpdatedAt = time.Now()
			return order, true, nil
		default:
			return order, false, nil
		}
	})
}

func (s *OrderService) MarkInventoryReservationFailed(ctx context.Context, orderID string, reason string) (domain.Order, error) {
	if strings.TrimSpace(reason) == "" {
		reason = "inventory reservation failed"
	}

	return s.transitionOrder(ctx, orderID, func(order domain.Order) (domain.Order, bool, error) {
		switch order.Status {
		case orderv1.OrderStatus_ORDER_STATUS_CANCELLED:
			return order, false, nil
		case orderv1.OrderStatus_ORDER_STATUS_PENDING, orderv1.OrderStatus_ORDER_STATUS_AWAITING_PAYMENT:
			order = cancelOrderWithReason(order, reason)
			return order, true, nil
		default:
			return order, false, nil
		}
	})
}

func (s *OrderService) MarkPaymentCreated(ctx context.Context, orderID string) (domain.Order, error) {
	return s.transitionOrder(ctx, orderID, func(order domain.Order) (domain.Order, bool, error) {
		switch order.Status {
		case orderv1.OrderStatus_ORDER_STATUS_CONFIRMED:
			return order, false, nil
		case orderv1.OrderStatus_ORDER_STATUS_CANCELLED:
			return order, false, nil
		case orderv1.OrderStatus_ORDER_STATUS_PENDING, orderv1.OrderStatus_ORDER_STATUS_AWAITING_PAYMENT:
			order.Status = orderv1.OrderStatus_ORDER_STATUS_CONFIRMED
			order.CancelReason = ""
			order.UpdatedAt = time.Now()
			return order, true, nil
		default:
			return order, false, nil
		}
	})
}

func (s *OrderService) MarkPaymentCreationFailed(ctx context.Context, orderID string, reason string) (domain.Order, error) {
	if strings.TrimSpace(reason) == "" {
		reason = "payment creation failed"
	}

	return s.transitionOrder(ctx, orderID, func(order domain.Order) (domain.Order, bool, error) {
		switch order.Status {
		case orderv1.OrderStatus_ORDER_STATUS_CANCELLED:
			return order, false, nil
		case orderv1.OrderStatus_ORDER_STATUS_PENDING, orderv1.OrderStatus_ORDER_STATUS_AWAITING_PAYMENT:
			order = cancelOrderWithReason(order, reason)
			return order, true, nil
		default:
			return order, false, nil
		}
	})
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

func normalizeOrderInput(order domain.Order) domain.Order {
	order.CustomerID = strings.TrimSpace(order.CustomerID)
	order.IdempotencyKey = strings.TrimSpace(order.IdempotencyKey)
	order.ShippingAddress.Country = strings.TrimSpace(order.ShippingAddress.Country)
	order.ShippingAddress.City = strings.TrimSpace(order.ShippingAddress.City)
	order.ShippingAddress.Street = strings.TrimSpace(order.ShippingAddress.Street)
	order.ShippingAddress.PostalCode = strings.TrimSpace(order.ShippingAddress.PostalCode)
	order.ShippingAddress.House = strings.TrimSpace(order.ShippingAddress.House)
	order.ShippingAddress.Apartment = strings.TrimSpace(order.ShippingAddress.Apartment)
	order.Payment.MethodDetails = strings.TrimSpace(order.Payment.MethodDetails)

	for i, item := range order.Items {
		order.Items[i].ProductID = strings.TrimSpace(item.ProductID)
		order.Items[i].SKU = strings.TrimSpace(item.SKU)
		order.Items[i].ProductName = strings.TrimSpace(item.ProductName)
	}

	return order
}

func validateCreateOrder(order domain.Order) error {
	if order.CustomerID == "" || len(order.Items) == 0 {
		return domain.ErrInvalidOrder
	}
	if order.ShippingAddress.Country == "" ||
		order.ShippingAddress.City == "" ||
		order.ShippingAddress.Street == "" ||
		order.ShippingAddress.PostalCode == "" ||
		order.ShippingAddress.House == "" {
		return domain.ErrInvalidOrder
	}
	if order.Payment.Method == orderv1.PaymentMethodType_PAYMENT_METHOD_TYPE_UNSPECIFIED {
		return domain.ErrInvalidOrder
	}

	return nil
}

func ordersMatchForIdempotency(existing domain.Order, incoming domain.Order) bool {
	if existing.CustomerID != incoming.CustomerID {
		return false
	}
	if existing.ShippingAddress != incoming.ShippingAddress {
		return false
	}
	if existing.Payment != incoming.Payment {
		return false
	}
	if len(existing.Items) != len(incoming.Items) {
		return false
	}

	for i := range incoming.Items {
		existingItem := existing.Items[i]
		incomingItem := incoming.Items[i]

		if existingItem.ProductID != incomingItem.ProductID ||
			existingItem.SKU != incomingItem.SKU ||
			existingItem.ProductName != incomingItem.ProductName ||
			existingItem.Quantity != incomingItem.Quantity ||
			existingItem.UnitPrice != incomingItem.UnitPrice {
			return false
		}
	}

	return true
}

func cancelOrderWithReason(order domain.Order, reason string) domain.Order {
	order.Status = orderv1.OrderStatus_ORDER_STATUS_CANCELLED
	order.CancelReason = strings.TrimSpace(reason)
	order.UpdatedAt = time.Now()
	return order
}

func canCancelOrder(status orderv1.OrderStatus) bool {
	switch status {
	case orderv1.OrderStatus_ORDER_STATUS_PENDING,
		orderv1.OrderStatus_ORDER_STATUS_AWAITING_PAYMENT,
		orderv1.OrderStatus_ORDER_STATUS_CONFIRMED:
		return true
	default:
		return false
	}
}

func (s *OrderService) transitionOrder(
	ctx context.Context,
	orderID string,
	apply func(order domain.Order) (domain.Order, bool, error),
) (domain.Order, error) {
	order, err := s.repository.GetOrderByID(ctx, orderID)
	if err != nil {
		return domain.Order{}, err
	}

	updated, shouldPersist, err := apply(order)
	if err != nil {
		return domain.Order{}, err
	}
	if !shouldPersist {
		return updated, nil
	}

	return s.repository.UpdateOrder(ctx, updated)
}
