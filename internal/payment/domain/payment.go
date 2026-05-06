package domain

import (
	"errors"
	"time"

	paymentv1 "github.com/vladfc/event-driven-ecommerce-app/gen/payment/v1"
)

var (
	ErrPaymentNotFound             = errors.New("payment not found")
	ErrInvalidPayment              = errors.New("invalid payment")
	ErrPaymentAlreadyExists        = errors.New("payment already exists")
	ErrInvalidPaymentID            = errors.New("invalid payment id")
	ErrInvalidIdempotencyKey       = errors.New("invalid idempotency key")
	ErrIdempotencyKeyAlreadyExists = errors.New("payment already exists for idempotency key")
	ErrPaymentCannotBeCancelled    = errors.New("payment cannot be cancelled in current status")
)

type Money struct {
	Currency    paymentv1.Currency
	AmountCents int64
}

type Payment struct {
	ID                   string
	OrderID              string
	CustomerID           string
	Amount               Money
	PaymentMethod        paymentv1.PaymentMethodType
	PaymentMethodDetails string
	IdempotencyKey       string
	Status               paymentv1.PaymentStatus
	CancelReason         string
	CreatedAt            time.Time
	UpdatedAt            time.Time
}
