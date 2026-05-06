package repository

import (
	"context"

	"github.com/vladfc/event-driven-ecommerce-app/internal/payment/domain"
)

type PaymentRepository interface {
	CreatePayment(ctx context.Context, payment domain.Payment) (domain.Payment, error)
	GetPaymentByID(ctx context.Context, id string) (domain.Payment, error)
	GetPaymentByIdempotencyKey(ctx context.Context, key string) (domain.Payment, error)
	UpdatePayment(ctx context.Context, payment domain.Payment) (domain.Payment, error)
}
