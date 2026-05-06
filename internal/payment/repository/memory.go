package repository

import (
	"context"
	"strings"
	"sync"

	"github.com/vladfc/event-driven-ecommerce-app/internal/payment/domain"
)

type MemoryRepository struct {
	mu                 sync.RWMutex
	payments           map[string]domain.Payment
	idempotencyKeyToID map[string]string
}

func NewMemoryRepository(seed []domain.Payment) *MemoryRepository {
	payments := make(map[string]domain.Payment, len(seed))
	idempotencyKeys := make(map[string]string, len(seed))

	for _, payment := range seed {
		cloned := clonePayment(payment)
		payments[cloned.ID] = cloned

		if key := strings.TrimSpace(cloned.IdempotencyKey); key != "" {
			idempotencyKeys[key] = cloned.ID
		}
	}

	return &MemoryRepository{
		payments:           payments,
		idempotencyKeyToID: idempotencyKeys,
	}
}

func (r *MemoryRepository) CreatePayment(ctx context.Context, payment domain.Payment) (domain.Payment, error) {
	_ = ctx

	r.mu.Lock()
	defer r.mu.Unlock()

	if err := validatePayment(payment); err != nil {
		return domain.Payment{}, err
	}

	if _, exists := r.payments[payment.ID]; exists {
		return domain.Payment{}, domain.ErrPaymentAlreadyExists
	}

	if key := strings.TrimSpace(payment.IdempotencyKey); key != "" {
		if _, exists := r.idempotencyKeyToID[key]; exists {
			return domain.Payment{}, domain.ErrIdempotencyKeyAlreadyExists
		}
	}

	cloned := clonePayment(payment)
	r.payments[cloned.ID] = cloned

	if key := strings.TrimSpace(cloned.IdempotencyKey); key != "" {
		r.idempotencyKeyToID[key] = cloned.ID
	}

	return clonePayment(cloned), nil
}

func (r *MemoryRepository) GetPaymentByID(ctx context.Context, id string) (domain.Payment, error) {
	_ = ctx

	r.mu.RLock()
	defer r.mu.RUnlock()

	if strings.TrimSpace(id) == "" {
		return domain.Payment{}, domain.ErrInvalidPaymentID
	}

	payment, ok := r.payments[id]
	if !ok {
		return domain.Payment{}, domain.ErrPaymentNotFound
	}

	return clonePayment(payment), nil
}

func (r *MemoryRepository) GetPaymentByIdempotencyKey(ctx context.Context, key string) (domain.Payment, error) {
	_ = ctx

	r.mu.RLock()
	defer r.mu.RUnlock()

	key = strings.TrimSpace(key)
	if key == "" {
		return domain.Payment{}, domain.ErrInvalidIdempotencyKey
	}

	paymentID, ok := r.idempotencyKeyToID[key]
	if !ok {
		return domain.Payment{}, domain.ErrPaymentNotFound
	}

	payment, ok := r.payments[paymentID]
	if !ok {
		return domain.Payment{}, domain.ErrPaymentNotFound
	}

	return clonePayment(payment), nil
}

func (r *MemoryRepository) UpdatePayment(ctx context.Context, payment domain.Payment) (domain.Payment, error) {
	_ = ctx

	r.mu.Lock()
	defer r.mu.Unlock()

	if err := validatePayment(payment); err != nil {
		return domain.Payment{}, err
	}

	existing, exists := r.payments[payment.ID]
	if !exists {
		return domain.Payment{}, domain.ErrPaymentNotFound
	}

	oldKey := strings.TrimSpace(existing.IdempotencyKey)
	newKey := strings.TrimSpace(payment.IdempotencyKey)
	if newKey != "" {
		if paymentID, exists := r.idempotencyKeyToID[newKey]; exists && paymentID != payment.ID {
			return domain.Payment{}, domain.ErrIdempotencyKeyAlreadyExists
		}
	}

	if oldKey != "" && oldKey != newKey {
		delete(r.idempotencyKeyToID, oldKey)
	}

	cloned := clonePayment(payment)
	r.payments[cloned.ID] = cloned

	if newKey != "" {
		r.idempotencyKeyToID[newKey] = cloned.ID
	}

	return clonePayment(cloned), nil
}

func validatePayment(payment domain.Payment) error {
	if strings.TrimSpace(payment.ID) == "" {
		return domain.ErrInvalidPaymentID
	}

	if strings.TrimSpace(payment.OrderID) == "" || strings.TrimSpace(payment.CustomerID) == "" {
		return domain.ErrInvalidPayment
	}

	return nil
}

func clonePayment(payment domain.Payment) domain.Payment {
	return payment
}
