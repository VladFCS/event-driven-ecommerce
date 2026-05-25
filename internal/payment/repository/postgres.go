package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
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

func (r *PostgresRepository) CreatePayment(ctx context.Context, payment domain.Payment) (domain.Payment, error) {
	if err := validatePayment(payment); err != nil {
		return domain.Payment{}, err
	}

	row, err := r.queries.CreatePayment(ctx, toCreatePaymentParams(payment))
	if err != nil {
		return domain.Payment{}, mapCreatePaymentError(err)
	}

	mapped, err := mapDBPayment(row)
	if err != nil {
		return domain.Payment{}, fmt.Errorf("map created payment from postgres: %w", err)
	}

	return mapped, nil
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

func (r *PostgresRepository) GetPaymentByIdempotencyKey(ctx context.Context, key string) (domain.Payment, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return domain.Payment{}, domain.ErrInvalidIdempotencyKey
	}

	row, err := r.queries.GetPaymentByIdempotencyKey(ctx, pgtype.Text{
		String: key,
		Valid:  true,
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return domain.Payment{}, domain.ErrPaymentNotFound
		}

		return domain.Payment{}, fmt.Errorf("get payment by idempotency key from postgres: %w", err)
	}

	return mapDBPayment(row)
}

func (r *PostgresRepository) GetPaymentByOrderID(ctx context.Context, orderID string) (domain.Payment, error) {
	orderID = strings.TrimSpace(orderID)
	if orderID == "" {
		return domain.Payment{}, domain.ErrInvalidPayment
	}

	row, err := r.queries.GetPaymentByOrderID(ctx, orderID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return domain.Payment{}, domain.ErrPaymentNotFound
		}

		return domain.Payment{}, fmt.Errorf("get payment by order id from postgres: %w", err)
	}

	return mapDBPayment(row)
}

func (r *PostgresRepository) UpdatePayment(ctx context.Context, payment domain.Payment) (domain.Payment, error) {
	if err := validatePayment(payment); err != nil {
		return domain.Payment{}, err
	}

	row, err := r.queries.UpdatePayment(ctx, toUpdatePaymentParams(payment))
	if err != nil {
		return domain.Payment{}, mapUpdatePaymentError(err)
	}

	mapped, err := mapDBPayment(row)
	if err != nil {
		return domain.Payment{}, fmt.Errorf("map updated payment from postgres: %w", err)
	}

	return mapped, nil
}

func (r *PostgresRepository) ListPaymentsByCustomer(ctx context.Context, customerID string, page, pageSize int32) ([]domain.Payment, int64, error) {
	customerID = strings.TrimSpace(customerID)
	if customerID == "" {
		return nil, 0, domain.ErrInvalidPayment
	}

	total, err := r.queries.CountPaymentsByCustomer(ctx, customerID)
	if err != nil {
		return nil, 0, fmt.Errorf("count payments by customer from postgres: %w", err)
	}

	if total == 0 {
		return []domain.Payment{}, 0, nil
	}

	if page <= 0 {
		page = 1
	}

	limit := int64(pageSize)
	if pageSize <= 0 {
		limit = total
	}

	offset := int64(page-1) * limit
	if offset >= total {
		return []domain.Payment{}, total, nil
	}

	rows, err := r.queries.ListPaymentsByCustomer(ctx, paymentdb.ListPaymentsByCustomerParams{
		CustomerID: customerID,
		Limit:      clampInt64ToInt32(limit),
		Offset:     clampInt64ToInt32(offset),
	})
	if err != nil {
		return nil, 0, fmt.Errorf("list payments by customer from postgres: %w", err)
	}

	payments, err := mapDBPayments(rows)
	if err != nil {
		return nil, 0, fmt.Errorf("map listed payments from postgres: %w", err)
	}

	return payments, total, nil
}

func toCreatePaymentParams(payment domain.Payment) paymentdb.CreatePaymentParams {
	base := toPaymentDBParams(payment)

	return paymentdb.CreatePaymentParams{
		ID:                   base.ID,
		OrderID:              base.OrderID,
		CustomerID:           base.CustomerID,
		AmountCurrency:       base.AmountCurrency,
		AmountCents:          base.AmountCents,
		PaymentMethod:        base.PaymentMethod,
		PaymentMethodDetails: base.PaymentMethodDetails,
		IdempotencyKey:       base.IdempotencyKey,
		Status:               base.Status,
		CancelReason:         base.CancelReason,
		CreatedAt:            base.CreatedAt,
		UpdatedAt:            base.UpdatedAt,
	}
}

func toUpdatePaymentParams(payment domain.Payment) paymentdb.UpdatePaymentParams {
	base := toPaymentDBParams(payment)

	return paymentdb.UpdatePaymentParams{
		ID:                   base.ID,
		OrderID:              base.OrderID,
		CustomerID:           base.CustomerID,
		AmountCurrency:       base.AmountCurrency,
		AmountCents:          base.AmountCents,
		PaymentMethod:        base.PaymentMethod,
		PaymentMethodDetails: base.PaymentMethodDetails,
		IdempotencyKey:       base.IdempotencyKey,
		Status:               base.Status,
		CancelReason:         base.CancelReason,
		CreatedAt:            base.CreatedAt,
		UpdatedAt:            base.UpdatedAt,
	}
}

type paymentDBParams struct {
	ID                   string
	OrderID              string
	CustomerID           string
	AmountCurrency       string
	AmountCents          int64
	PaymentMethod        string
	PaymentMethodDetails string
	IdempotencyKey       pgtype.Text
	Status               string
	CancelReason         string
	CreatedAt            pgtype.Timestamptz
	UpdatedAt            pgtype.Timestamptz
}

func toPaymentDBParams(payment domain.Payment) paymentDBParams {
	params := paymentDBParams{
		ID:                   payment.ID,
		OrderID:              payment.OrderID,
		CustomerID:           payment.CustomerID,
		AmountCurrency:       payment.Amount.Currency.String(),
		AmountCents:          payment.Amount.AmountCents,
		PaymentMethod:        payment.PaymentMethod.String(),
		PaymentMethodDetails: payment.PaymentMethodDetails,
		Status:               payment.Status.String(),
		CancelReason:         payment.CancelReason,
		CreatedAt: pgtype.Timestamptz{
			Time:  payment.CreatedAt,
			Valid: !payment.CreatedAt.IsZero(),
		},
		UpdatedAt: pgtype.Timestamptz{
			Time:  payment.UpdatedAt,
			Valid: !payment.UpdatedAt.IsZero(),
		},
	}

	if key := strings.TrimSpace(payment.IdempotencyKey); key != "" {
		params.IdempotencyKey = pgtype.Text{
			String: key,
			Valid:  true,
		}
	}

	return params
}

func mapCreatePaymentError(err error) error {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return fmt.Errorf("create payment in postgres: %w", err)
	}

	if pgErr.Code != "23505" {
		return fmt.Errorf("create payment in postgres: %w", err)
	}

	switch pgErr.ConstraintName {
	case "payments_pkey", "payments_order_id_key":
		return domain.ErrPaymentAlreadyExists
	case "payments_idempotency_key_key":
		return domain.ErrIdempotencyKeyAlreadyExists
	default:
		return fmt.Errorf("create payment in postgres: %w", err)
	}
}

func mapUpdatePaymentError(err error) error {
	if err == pgx.ErrNoRows {
		return domain.ErrPaymentNotFound
	}

	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return fmt.Errorf("update payment in postgres: %w", err)
	}

	if pgErr.Code != "23505" {
		return fmt.Errorf("update payment in postgres: %w", err)
	}

	switch pgErr.ConstraintName {
	case "payments_order_id_key":
		return domain.ErrPaymentAlreadyExists
	case "payments_idempotency_key_key":
		return domain.ErrIdempotencyKeyAlreadyExists
	default:
		return fmt.Errorf("update payment in postgres: %w", err)
	}
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

func mapDBPayments(rows []paymentdb.Payment) ([]domain.Payment, error) {
	payments := make([]domain.Payment, 0, len(rows))
	for _, row := range rows {
		payment, err := mapDBPayment(row)
		if err != nil {
			return nil, err
		}

		payments = append(payments, payment)
	}

	return payments, nil
}

func clampInt64ToInt32(value int64) int32 {
	const maxInt32 = int64(2147483647)

	if value > maxInt32 {
		return int32(maxInt32)
	}
	if value < 0 {
		return 0
	}

	return int32(value)
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

func validatePayment(payment domain.Payment) error {
	if strings.TrimSpace(payment.ID) == "" {
		return domain.ErrInvalidPaymentID
	}

	if strings.TrimSpace(payment.OrderID) == "" || strings.TrimSpace(payment.CustomerID) == "" {
		return domain.ErrInvalidPayment
	}

	return nil
}
