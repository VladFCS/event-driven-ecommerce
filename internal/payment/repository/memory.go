package repository

import (
	"context"
	"sort"
	"strings"
	"sync"

	"github.com/vladfc/event-driven-ecommerce-app/internal/payment/domain"
)

type MemoryRepository struct {
	mu                 sync.RWMutex
	payments           map[string]domain.Payment
	orderIDToID        map[string]string
	idempotencyKeyToID map[string]string
}

func NewMemoryRepository(seed []domain.Payment) *MemoryRepository {
	payments := make(map[string]domain.Payment, len(seed))
	orderIDs := make(map[string]string, len(seed))
	idempotencyKeys := make(map[string]string, len(seed))

	for _, payment := range seed {
		cloned := clonePayment(payment)
		payments[cloned.ID] = cloned
		orderIDs[cloned.OrderID] = cloned.ID

		if key := strings.TrimSpace(cloned.IdempotencyKey); key != "" {
			idempotencyKeys[key] = cloned.ID
		}
	}

	return &MemoryRepository{
		payments:           payments,
		orderIDToID:        orderIDs,
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
	r.orderIDToID[cloned.OrderID] = cloned.ID

	if key := strings.TrimSpace(cloned.IdempotencyKey); key != "" {
		r.idempotencyKeyToID[key] = cloned.ID
	}

	return clonePayment(cloned), nil
}

func (r *MemoryRepository) GetPaymentByID(ctx context.Context, paymentID string) (domain.Payment, error) {
	_ = ctx

	r.mu.RLock()
	defer r.mu.RUnlock()

	if strings.TrimSpace(paymentID) == "" {
		return domain.Payment{}, domain.ErrInvalidPaymentID
	}

	payment, ok := r.payments[paymentID]
	if !ok {
		return domain.Payment{}, domain.ErrPaymentNotFound
	}

	return clonePayment(payment), nil
}

func (r *MemoryRepository) GetPaymentByOrderID(ctx context.Context, orderID string) (domain.Payment, error) {
	_ = ctx

	r.mu.RLock()
	defer r.mu.RUnlock()

	orderID = strings.TrimSpace(orderID)
	if orderID == "" {
		return domain.Payment{}, domain.ErrInvalidPayment
	}

	paymentID, ok := r.orderIDToID[orderID]
	if !ok {
		return domain.Payment{}, domain.ErrPaymentNotFound
	}

	payment, ok := r.payments[paymentID]
	if !ok {
		return domain.Payment{}, domain.ErrPaymentNotFound
	}

	return clonePayment(payment), nil
}

func (r *MemoryRepository) ListPaymentsByCustomer(ctx context.Context, customerID string, page, pageSize int32) ([]domain.Payment, int64, error) {
	_ = ctx

	r.mu.RLock()
	defer r.mu.RUnlock()

	customerID = strings.TrimSpace(customerID)
	if customerID == "" {
		return nil, 0, domain.ErrInvalidPayment
	}

	filtered := make([]domain.Payment, 0)
	for _, payment := range r.payments {
		if payment.CustomerID == customerID {
			filtered = append(filtered, clonePayment(payment))
		}
	}

	sort.Slice(filtered, func(i, j int) bool {
		if filtered[i].CreatedAt.Equal(filtered[j].CreatedAt) {
			return filtered[i].ID < filtered[j].ID
		}

		return filtered[i].CreatedAt.Before(filtered[j].CreatedAt)
	})

	total := int64(len(filtered))
	if pageSize <= 0 {
		pageSize = int32(len(filtered))
	}
	if page <= 0 {
		page = 1
	}

	start := int((page - 1) * pageSize)
	if start >= len(filtered) {
		return []domain.Payment{}, total, nil
	}

	end := start + int(pageSize)
	if end > len(filtered) {
		end = len(filtered)
	}

	return filtered[start:end], total, nil
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
	r.orderIDToID[cloned.OrderID] = cloned.ID

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
