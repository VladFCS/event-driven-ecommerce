package service

import (
	"context"
	"testing"

	orderv1 "github.com/vladfc/event-driven-ecommerce-app/gen/order/v1"
	"github.com/vladfc/event-driven-ecommerce-app/internal/order/domain"
)

type stubOrderRepository struct {
	ordersByID              map[string]domain.Order
	ordersByIdempotencyKey  map[string]string
	createErr               error
	updateErr               error
}

func newStubOrderRepository() *stubOrderRepository {
	return &stubOrderRepository{
		ordersByID:             make(map[string]domain.Order),
		ordersByIdempotencyKey: make(map[string]string),
	}
}

func (r *stubOrderRepository) CreateOrder(_ context.Context, order domain.Order) (domain.Order, error) {
	if r.createErr != nil {
		return domain.Order{}, r.createErr
	}

	r.ordersByID[order.ID] = order
	if order.IdempotencyKey != "" {
		r.ordersByIdempotencyKey[order.IdempotencyKey] = order.ID
	}

	return order, nil
}

func (r *stubOrderRepository) GetOrderByID(_ context.Context, orderID string) (domain.Order, error) {
	order, ok := r.ordersByID[orderID]
	if !ok {
		return domain.Order{}, domain.ErrOrderNotFound
	}

	return order, nil
}

func (r *stubOrderRepository) GetOrderByIdempotencyKey(_ context.Context, key string) (domain.Order, error) {
	orderID, ok := r.ordersByIdempotencyKey[key]
	if !ok {
		return domain.Order{}, domain.ErrOrderNotFound
	}

	return r.ordersByID[orderID], nil
}

func (r *stubOrderRepository) ListOrdersByCustomer(_ context.Context, customerID string, page, pageSize int32) ([]domain.Order, int64, error) {
	return nil, 0, nil
}

func (r *stubOrderRepository) UpdateOrder(_ context.Context, order domain.Order) (domain.Order, error) {
	if r.updateErr != nil {
		return domain.Order{}, r.updateErr
	}

	r.ordersByID[order.ID] = order
	if order.IdempotencyKey != "" {
		r.ordersByIdempotencyKey[order.IdempotencyKey] = order.ID
	}

	return order, nil
}

func TestCreateOrderRejectsIncompleteAddress(t *testing.T) {
	t.Parallel()

	service := NewOrderService(newStubOrderRepository())
	order := validOrder()
	order.ShippingAddress.City = ""

	_, err := service.CreateOrder(context.Background(), order)
	if err != domain.ErrInvalidOrder {
		t.Fatalf("expected ErrInvalidOrder, got %v", err)
	}
}

func TestCreateOrderRejectsMissingPaymentMethod(t *testing.T) {
	t.Parallel()

	service := NewOrderService(newStubOrderRepository())
	order := validOrder()
	order.Payment.Method = orderv1.PaymentMethodType_PAYMENT_METHOD_TYPE_UNSPECIFIED

	_, err := service.CreateOrder(context.Background(), order)
	if err != domain.ErrInvalidOrder {
		t.Fatalf("expected ErrInvalidOrder, got %v", err)
	}
}

func TestCreateOrderReturnsExistingOrderForMatchingIdempotencyKey(t *testing.T) {
	t.Parallel()

	repo := newStubOrderRepository()
	existing := validOrder()
	existing.ID = "ord-existing"
	existing.IdempotencyKey = "same-key"
	existing.Status = orderv1.OrderStatus_ORDER_STATUS_PENDING
	repo.ordersByID[existing.ID] = existing
	repo.ordersByIdempotencyKey[existing.IdempotencyKey] = existing.ID

	service := NewOrderService(repo)
	order := validOrder()
	order.IdempotencyKey = "same-key"

	created, err := service.CreateOrder(context.Background(), order)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if created.ID != existing.ID {
		t.Fatalf("expected existing order %q, got %q", existing.ID, created.ID)
	}
}

func TestCreateOrderRejectsIdempotencyKeyPayloadMismatch(t *testing.T) {
	t.Parallel()

	repo := newStubOrderRepository()
	existing := validOrder()
	existing.ID = "ord-existing"
	existing.IdempotencyKey = "same-key"
	repo.ordersByID[existing.ID] = existing
	repo.ordersByIdempotencyKey[existing.IdempotencyKey] = existing.ID

	service := NewOrderService(repo)
	order := validOrder()
	order.IdempotencyKey = "same-key"
	order.Items[0].Quantity = 99

	_, err := service.CreateOrder(context.Background(), order)
	if err != domain.ErrIdempotencyKeyConflict {
		t.Fatalf("expected ErrIdempotencyKeyConflict, got %v", err)
	}
}

func TestMarkInventoryReservedTransitionsToAwaitingPayment(t *testing.T) {
	t.Parallel()

	repo := newStubOrderRepository()
	order := validOrder()
	order.ID = "ord-1"
	order.Status = orderv1.OrderStatus_ORDER_STATUS_PENDING
	repo.ordersByID[order.ID] = order

	service := NewOrderService(repo)
	updated, err := service.MarkInventoryReserved(context.Background(), order.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if updated.Status != orderv1.OrderStatus_ORDER_STATUS_AWAITING_PAYMENT {
		t.Fatalf("expected awaiting payment, got %s", updated.Status.String())
	}
}

func TestMarkPaymentCreatedTransitionsToConfirmed(t *testing.T) {
	t.Parallel()

	repo := newStubOrderRepository()
	order := validOrder()
	order.ID = "ord-1"
	order.Status = orderv1.OrderStatus_ORDER_STATUS_AWAITING_PAYMENT
	repo.ordersByID[order.ID] = order

	service := NewOrderService(repo)
	updated, err := service.MarkPaymentCreated(context.Background(), order.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if updated.Status != orderv1.OrderStatus_ORDER_STATUS_CONFIRMED {
		t.Fatalf("expected confirmed, got %s", updated.Status.String())
	}
}

func TestMarkPaymentCreationFailedCancelsOrderWithReason(t *testing.T) {
	t.Parallel()

	repo := newStubOrderRepository()
	order := validOrder()
	order.ID = "ord-1"
	order.Status = orderv1.OrderStatus_ORDER_STATUS_AWAITING_PAYMENT
	repo.ordersByID[order.ID] = order

	service := NewOrderService(repo)
	updated, err := service.MarkPaymentCreationFailed(context.Background(), order.ID, "card declined")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if updated.Status != orderv1.OrderStatus_ORDER_STATUS_CANCELLED {
		t.Fatalf("expected cancelled, got %s", updated.Status.String())
	}
	if updated.CancelReason != "card declined" {
		t.Fatalf("expected cancel reason to be persisted, got %q", updated.CancelReason)
	}
}

func TestCancelOrderRequiresReason(t *testing.T) {
	t.Parallel()

	repo := newStubOrderRepository()
	order := validOrder()
	order.ID = "ord-1"
	order.Status = orderv1.OrderStatus_ORDER_STATUS_PENDING
	repo.ordersByID[order.ID] = order

	service := NewOrderService(repo)
	_, err := service.CancelOrder(context.Background(), order.ID, "   ")
	if err != domain.ErrInvalidOrder {
		t.Fatalf("expected ErrInvalidOrder, got %v", err)
	}
}

func validOrder() domain.Order {
	return domain.Order{
		CustomerID:     "cust-1",
		IdempotencyKey: "",
		Items: []domain.OrderItem{
			{
				ProductID:   "prod-1",
				SKU:         "sku-1",
				ProductName: "Product",
				Quantity:    2,
				UnitPrice: domain.Money{
					Currency:    orderv1.Currency_CURRENCY_USD,
					AmountCents: 500,
				},
			},
		},
		ShippingAddress: domain.Address{
			Country:    "US",
			City:       "New York",
			Street:     "Main",
			PostalCode: "10001",
			House:      "1",
			Apartment:  "2A",
		},
		Payment: domain.PaymentDetails{
			Method:        orderv1.PaymentMethodType_PAYMENT_METHOD_TYPE_CARD,
			MethodDetails: "tok_123",
		},
	}
}
