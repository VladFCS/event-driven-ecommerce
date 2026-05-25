package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	paymentv1 "github.com/vladfc/event-driven-ecommerce-app/gen/payment/v1"
	"github.com/vladfc/event-driven-ecommerce-app/internal/payment/domain"
	paymentdb "github.com/vladfc/event-driven-ecommerce-app/internal/payment/repository/sqlc"
)

type PostgresRepository struct {
	pool    *pgxpool.Pool
	queries *paymentdb.Queries
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{
		pool:    pool,
		queries: paymentdb.New(pool),
	}
}

func (r *PostgresRepository) GetPaymentByID(ctx context.Context, paymentID string) (domain.Payment, error) {
	if strings.TrimSpace(paymentID) == "" {
		return domain.Payment{}, domain.ErrInvalidPaymentID
	}

	row, err := r.queries.GetPaymentByID(ctx, paymentID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return domain.Payment{}, domain.ErrPaymentNotFound
		}

		return domain.Payment{}, fmt.Errorf("get payment by id from postgres: %w", err)
	}

	return mapDBPayment(row)
}

func mapDBPayment(row paymentdb.Payment) (domain.Payment, error) {
	currency, err := parseCurrency(row.AmountCurrency)
	if err != nil {
		return domain.Payment{}, err
	}

	method, err := parsePaymentMethod(row.PaymentMethod)
	if err != nil {
		return domain.Payment{}, err
	}

	status, err := parsePaymentStatus(row.Status)
	if err != nil {
		return domain.Payment{}, err
	}

	payment := domain.Payment{
		ID:         row.ID,
		OrderID:    row.OrderID,
		CustomerID: row.CustomerID,
		Amount: domain.Money{
			Currency:    currency,
			AmountCents: row.AmountCents,
		},
		PaymentMethod:        method,
		PaymentMethodDetails: row.PaymentMethodDetails,
		Status:               status,
		CancelReason:         row.CancelReason,
	}

	if row.IdempotencyKey.Valid {
		payment.IdempotencyKey = row.IdempotencyKey.String
	}
	if row.CreatedAt.Valid {
		payment.CreatedAt = row.CreatedAt.Time
	}
	if row.UpdatedAt.Valid {
		payment.UpdatedAt = row.UpdatedAt.Time
	}

	return payment, nil
}

func parseCurrency(value string) (paymentv1.Currency, error) {
	enum, ok := paymentv1.Currency_value[value]
	if !ok {
		return paymentv1.Currency_CURRENCY_UNSPECIFIED, fmt.Errorf("unknown payment currency in db: %q", value)
	}

	return paymentv1.Currency(enum), nil
}

func parsePaymentMethod(value string) (paymentv1.PaymentMethodType, error) {
	enum, ok := paymentv1.PaymentMethodType_value[value]
	if !ok {
		return paymentv1.PaymentMethodType_PAYMENT_METHOD_TYPE_UNSPECIFIED, fmt.Errorf("unknown payment method in db: %q", value)
	}

	return paymentv1.PaymentMethodType(enum), nil
}

func parsePaymentStatus(value string) (paymentv1.PaymentStatus, error) {
	enum, ok := paymentv1.PaymentStatus_value[value]
	if !ok {
		return paymentv1.PaymentStatus_PAYMENT_STATUS_UNSPECIFIED, fmt.Errorf("unknown payment status in db: %q", value)
	}

	return paymentv1.PaymentStatus(enum), nil
}
