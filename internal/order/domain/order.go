package domain

import (
	"errors"
	"time"

	orderv1 "github.com/vladfc/event-driven-ecommerce-app/gen/order/v1"
)

var (
	ErrOrderNotFound               = errors.New("order not found")
	ErrInvalidOrder                = errors.New("invalid order")
	ErrOrderAlreadyExists          = errors.New("order already exists")
	ErrIdempotencyKeyAlreadyExists = errors.New("order already exists for idempotency key")
	ErrIdempotencyKeyConflict      = errors.New("idempotency key already used for a different order payload")
	ErrInvalidOrderID              = errors.New("invalid order id")
	ErrInvalidCustomerID           = errors.New("invalid customer id")
	ErrOrderCannotBeCancelled      = errors.New("order cannot be cancelled in current status")
)

type Money struct {
	Currency    orderv1.Currency
	AmountCents int64
}

type OrderItem struct {
	ProductID   string
	SKU         string
	ProductName string
	Quantity    int32
	UnitPrice   Money
	TotalPrice  Money
}

type Address struct {
	Country    string
	City       string
	Street     string
	PostalCode string
	House      string
	Apartment  string
}

type PaymentDetails struct {
	Method        orderv1.PaymentMethodType
	MethodDetails string
}

type Order struct {
	ID              string
	CustomerID      string
	IdempotencyKey  string
	Items           []OrderItem
	TotalAmount     Money
	Status          orderv1.OrderStatus
	ShippingAddress Address
	Payment         PaymentDetails
	CancelReason    string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}
