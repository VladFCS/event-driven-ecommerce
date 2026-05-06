package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	paymentv1 "github.com/vladfc/event-driven-ecommerce-app/gen/payment/v1"
	"github.com/vladfc/event-driven-ecommerce-app/internal/payment/domain"
	"github.com/vladfc/event-driven-ecommerce-app/internal/payment/repository"
)

type PaymentService struct {
	repository repository.PaymentRepository
}

func NewPaymentService(repository repository.PaymentRepository) *PaymentService {
	return &PaymentService{
		repository: repository,
	}
}

func (s *PaymentService) CreatePayment(ctx context.Context, payment domain.Payment) (domain.Payment, error) {
	if err := validateCreatePayment(payment); err != nil {
		return domain.Payment{}, err
	}

	if key := strings.TrimSpace(payment.IdempotencyKey); key != "" {
		existing, err := s.repository.GetPaymentByIdempotencyKey(ctx, key)
		switch {
		case err == nil:
			return existing, nil
		case !errors.Is(err, domain.ErrPaymentNotFound):
			return domain.Payment{}, err
		}
	}

	if strings.TrimSpace(payment.ID) == "" {
		payment.ID = newPaymentID()
	}

	now := time.Now()
	payment.Status = paymentv1.PaymentStatus_PAYMENT_STATUS_PENDING
	payment.CreatedAt = now
	payment.UpdatedAt = now
	payment.CancelReason = ""

	created, err := s.repository.CreatePayment(ctx, payment)
	if err == nil {
		return created, nil
	}

	if strings.TrimSpace(payment.IdempotencyKey) != "" && errors.Is(err, domain.ErrIdempotencyKeyAlreadyExists) {
		return s.repository.GetPaymentByIdempotencyKey(ctx, payment.IdempotencyKey)
	}

	return domain.Payment{}, err
}

func (s *PaymentService) GetPaymentByID(ctx context.Context, paymentID string) (domain.Payment, error) {
	if strings.TrimSpace(paymentID) == "" {
		return domain.Payment{}, domain.ErrInvalidPaymentID
	}

	return s.repository.GetPaymentByID(ctx, paymentID)
}

func (s *PaymentService) CancelPayment(ctx context.Context, paymentID string, reason string) (domain.Payment, error) {
	if strings.TrimSpace(paymentID) == "" {
		return domain.Payment{}, domain.ErrInvalidPaymentID
	}

	payment, err := s.repository.GetPaymentByID(ctx, paymentID)
	if err != nil {
		return domain.Payment{}, err
	}

	switch payment.Status {
	case paymentv1.PaymentStatus_PAYMENT_STATUS_CANCELLED:
		return payment, nil
	case paymentv1.PaymentStatus_PAYMENT_STATUS_PENDING,
		paymentv1.PaymentStatus_PAYMENT_STATUS_REQUIRES_ACTION,
		paymentv1.PaymentStatus_PAYMENT_STATUS_AUTHORIZED:
		payment.Status = paymentv1.PaymentStatus_PAYMENT_STATUS_CANCELLED
		payment.CancelReason = strings.TrimSpace(reason)
		payment.UpdatedAt = time.Now()
		return s.repository.UpdatePayment(ctx, payment)
	default:
		return domain.Payment{}, domain.ErrPaymentCannotBeCancelled
	}
}

func validateCreatePayment(payment domain.Payment) error {
	if strings.TrimSpace(payment.OrderID) == "" || strings.TrimSpace(payment.CustomerID) == "" {
		return domain.ErrInvalidPayment
	}

	if payment.Amount.AmountCents <= 0 || payment.Amount.Currency == paymentv1.Currency_CURRENCY_UNSPECIFIED {
		return domain.ErrInvalidPayment
	}

	if payment.PaymentMethod == paymentv1.PaymentMethodType_PAYMENT_METHOD_TYPE_UNSPECIFIED {
		return domain.ErrInvalidPayment
	}

	return nil
}

func newPaymentID() string {
	return fmt.Sprintf("pay-%d", time.Now().UTC().UnixNano())
}
