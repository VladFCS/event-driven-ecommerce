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
	orderv1 "github.com/vladfc/event-driven-ecommerce-app/gen/order/v1"
	"github.com/vladfc/event-driven-ecommerce-app/internal/order/domain"
	orderdb "github.com/vladfc/event-driven-ecommerce-app/internal/order/repository/sqlc"
)

type PostgresRepository struct {
	pool    *pgxpool.Pool
	queries *orderdb.Queries
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{
		pool:    pool,
		queries: orderdb.New(pool),
	}
}

func (r *PostgresRepository) CreateOrder(ctx context.Context, order domain.Order) (domain.Order, error) {
	if err := validateOrder(order); err != nil {
		return domain.Order{}, err
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.Order{}, fmt.Errorf("begin create order transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	qtx := r.queries.WithTx(tx)

	row, err := qtx.CreateOrder(ctx, toCreateOrderParams(order))
	if err != nil {
		return domain.Order{}, mapCreateOrderError(err)
	}

	if err := insertOrderItems(ctx, qtx, order); err != nil {
		return domain.Order{}, err
	}

	items, err := qtx.ListOrderItemsByOrderID(ctx, order.ID)
	if err != nil {
		return domain.Order{}, fmt.Errorf("load created order items from postgres: %w", err)
	}

	mapped, err := mapDBOrder(row, items)
	if err != nil {
		return domain.Order{}, fmt.Errorf("map created order from postgres: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.Order{}, fmt.Errorf("commit create order transaction: %w", err)
	}

	return mapped, nil
}

func (r *PostgresRepository) GetOrderByID(ctx context.Context, orderID string) (domain.Order, error) {
	row, err := r.queries.GetOrderByID(ctx, orderID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return domain.Order{}, domain.ErrOrderNotFound
		}

		return domain.Order{}, fmt.Errorf("get order by id from postgres: %w", err)
	}

	items, err := r.queries.ListOrderItemsByOrderID(ctx, orderID)
	if err != nil {
		return domain.Order{}, fmt.Errorf("list order items by order id from postgres: %w", err)
	}

	return mapDBOrder(row, items)
}

func (r *PostgresRepository) ListOrdersByCustomer(ctx context.Context, customerID string, page, pageSize int32) ([]domain.Order, int64, error) {
	total, err := r.queries.CountOrdersByCustomer(ctx, customerID)
	if err != nil {
		return nil, 0, fmt.Errorf("count orders by customer from postgres: %w", err)
	}

	if total == 0 {
		return []domain.Order{}, 0, nil
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
		return []domain.Order{}, total, nil
	}

	rows, err := r.queries.ListOrdersByCustomer(ctx, orderdb.ListOrdersByCustomerParams{
		CustomerID: customerID,
		Limit:      clampInt64ToInt32(limit),
		Offset:     clampInt64ToInt32(offset),
	})
	if err != nil {
		return nil, 0, fmt.Errorf("list orders by customer from postgres: %w", err)
	}

	orderIDs := make([]string, 0, len(rows))
	for _, row := range rows {
		orderIDs = append(orderIDs, row.ID)
	}

	itemRows, err := r.queries.ListOrderItemsByOrderIDs(ctx, orderIDs)
	if err != nil {
		return nil, 0, fmt.Errorf("list order items by order ids from postgres: %w", err)
	}

	groupedItems, err := groupDBOrderItems(itemRows)
	if err != nil {
		return nil, 0, fmt.Errorf("group listed order items from postgres: %w", err)
	}

	orders, err := mapDBOrders(rows, groupedItems)
	if err != nil {
		return nil, 0, fmt.Errorf("map listed orders from postgres: %w", err)
	}

	return orders, total, nil
}

func (r *PostgresRepository) UpdateOrder(ctx context.Context, order domain.Order) (domain.Order, error) {
	if err := validateOrder(order); err != nil {
		return domain.Order{}, err
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.Order{}, fmt.Errorf("begin update order transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	qtx := r.queries.WithTx(tx)

	row, err := qtx.UpdateOrder(ctx, toUpdateOrderParams(order))
	if err != nil {
		return domain.Order{}, mapUpdateOrderError(err)
	}

	if err := qtx.DeleteOrderItemsByOrderID(ctx, order.ID); err != nil {
		return domain.Order{}, fmt.Errorf("delete order items by order id in postgres: %w", err)
	}

	if err := insertOrderItems(ctx, qtx, order); err != nil {
		return domain.Order{}, err
	}

	items, err := qtx.ListOrderItemsByOrderID(ctx, order.ID)
	if err != nil {
		return domain.Order{}, fmt.Errorf("load updated order items from postgres: %w", err)
	}

	mapped, err := mapDBOrder(row, items)
	if err != nil {
		return domain.Order{}, fmt.Errorf("map updated order from postgres: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.Order{}, fmt.Errorf("commit update order transaction: %w", err)
	}

	return mapped, nil
}

func validateOrder(order domain.Order) error {
	if strings.TrimSpace(order.ID) == "" || strings.TrimSpace(order.CustomerID) == "" {
		return domain.ErrInvalidOrder
	}

	return nil
}

func insertOrderItems(ctx context.Context, q *orderdb.Queries, order domain.Order) error {
	for i, item := range order.Items {
		if err := q.CreateOrderItem(ctx, orderdb.CreateOrderItemParams{
			OrderID:               order.ID,
			ItemPosition:          int32(i),
			ProductID:             item.ProductID,
			Sku:                   item.SKU,
			ProductName:           item.ProductName,
			Quantity:              item.Quantity,
			UnitPriceCurrency:     item.UnitPrice.Currency.String(),
			UnitPriceAmountCents:  item.UnitPrice.AmountCents,
			TotalPriceCurrency:    item.TotalPrice.Currency.String(),
			TotalPriceAmountCents: item.TotalPrice.AmountCents,
		}); err != nil {
			return fmt.Errorf("create order item in postgres: %w", err)
		}
	}

	return nil
}

func toCreateOrderParams(order domain.Order) orderdb.CreateOrderParams {
	base := toOrderDBParams(order)

	return orderdb.CreateOrderParams{
		ID:                   base.ID,
		CustomerID:           base.CustomerID,
		TotalAmountCurrency:  base.TotalAmountCurrency,
		TotalAmountCents:     base.TotalAmountCents,
		Status:               base.Status,
		ShippingCountry:      base.ShippingCountry,
		ShippingCity:         base.ShippingCity,
		ShippingStreet:       base.ShippingStreet,
		ShippingPostalCode:   base.ShippingPostalCode,
		ShippingHouse:        base.ShippingHouse,
		ShippingApartment:    base.ShippingApartment,
		PaymentMethod:        base.PaymentMethod,
		PaymentMethodDetails: base.PaymentMethodDetails,
		CreatedAt:            base.CreatedAt,
		UpdatedAt:            base.UpdatedAt,
	}
}

func toUpdateOrderParams(order domain.Order) orderdb.UpdateOrderParams {
	base := toOrderDBParams(order)

	return orderdb.UpdateOrderParams{
		ID:                   base.ID,
		CustomerID:           base.CustomerID,
		TotalAmountCurrency:  base.TotalAmountCurrency,
		TotalAmountCents:     base.TotalAmountCents,
		Status:               base.Status,
		ShippingCountry:      base.ShippingCountry,
		ShippingCity:         base.ShippingCity,
		ShippingStreet:       base.ShippingStreet,
		ShippingPostalCode:   base.ShippingPostalCode,
		ShippingHouse:        base.ShippingHouse,
		ShippingApartment:    base.ShippingApartment,
		PaymentMethod:        base.PaymentMethod,
		PaymentMethodDetails: base.PaymentMethodDetails,
		CreatedAt:            base.CreatedAt,
		UpdatedAt:            base.UpdatedAt,
	}
}

type orderDBParams struct {
	ID                   string
	CustomerID           string
	TotalAmountCurrency  string
	TotalAmountCents     int64
	Status               string
	ShippingCountry      string
	ShippingCity         string
	ShippingStreet       string
	ShippingPostalCode   string
	ShippingHouse        string
	ShippingApartment    string
	PaymentMethod        string
	PaymentMethodDetails string
	CreatedAt            pgtype.Timestamptz
	UpdatedAt            pgtype.Timestamptz
}

func toOrderDBParams(order domain.Order) orderDBParams {
	return orderDBParams{
		ID:                   order.ID,
		CustomerID:           order.CustomerID,
		TotalAmountCurrency:  order.TotalAmount.Currency.String(),
		TotalAmountCents:     order.TotalAmount.AmountCents,
		Status:               order.Status.String(),
		ShippingCountry:      order.ShippingAddress.Country,
		ShippingCity:         order.ShippingAddress.City,
		ShippingStreet:       order.ShippingAddress.Street,
		ShippingPostalCode:   order.ShippingAddress.PostalCode,
		ShippingHouse:        order.ShippingAddress.House,
		ShippingApartment:    order.ShippingAddress.Apartment,
		PaymentMethod:        order.Payment.Method.String(),
		PaymentMethodDetails: order.Payment.MethodDetails,
		CreatedAt: pgtype.Timestamptz{
			Time:  order.CreatedAt,
			Valid: !order.CreatedAt.IsZero(),
		},
		UpdatedAt: pgtype.Timestamptz{
			Time:  order.UpdatedAt,
			Valid: !order.UpdatedAt.IsZero(),
		},
	}
}

func mapCreateOrderError(err error) error {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return fmt.Errorf("create order in postgres: %w", err)
	}

	if pgErr.Code != "23505" {
		return fmt.Errorf("create order in postgres: %w", err)
	}

	switch pgErr.ConstraintName {
	case "orders_pkey":
		return domain.ErrOrderAlreadyExists
	default:
		return fmt.Errorf("create order in postgres: %w", err)
	}
}

func mapUpdateOrderError(err error) error {
	if err == pgx.ErrNoRows {
		return domain.ErrOrderNotFound
	}

	return fmt.Errorf("update order in postgres: %w", err)
}

func mapDBOrders(rows []orderdb.Order, itemsByOrderID map[string][]domain.OrderItem) ([]domain.Order, error) {
	orders := make([]domain.Order, 0, len(rows))
	for _, row := range rows {
		order, err := mapDBOrder(row, toDBItems(itemsByOrderID[row.ID]))
		if err != nil {
			return nil, err
		}

		orders = append(orders, order)
	}

	return orders, nil
}

func groupDBOrderItems(rows []orderdb.OrderItem) (map[string][]domain.OrderItem, error) {
	grouped := make(map[string][]domain.OrderItem, len(rows))
	for _, row := range rows {
		item, err := mapDBOrderItem(row)
		if err != nil {
			return nil, err
		}

		grouped[row.OrderID] = append(grouped[row.OrderID], item)
	}

	return grouped, nil
}

func toDBItems(items []domain.OrderItem) []orderdb.OrderItem {
	dbItems := make([]orderdb.OrderItem, 0, len(items))
	for idx, item := range items {
		dbItems = append(dbItems, orderdb.OrderItem{
			ItemPosition:          int32(idx),
			ProductID:             item.ProductID,
			Sku:                   item.SKU,
			ProductName:           item.ProductName,
			Quantity:              item.Quantity,
			UnitPriceCurrency:     item.UnitPrice.Currency.String(),
			UnitPriceAmountCents:  item.UnitPrice.AmountCents,
			TotalPriceCurrency:    item.TotalPrice.Currency.String(),
			TotalPriceAmountCents: item.TotalPrice.AmountCents,
		})
	}

	return dbItems
}

func mapDBOrder(row orderdb.Order, itemRows []orderdb.OrderItem) (domain.Order, error) {
	totalCurrency, err := parseCurrency(row.TotalAmountCurrency)
	if err != nil {
		return domain.Order{}, err
	}

	status, err := parseOrderStatus(row.Status)
	if err != nil {
		return domain.Order{}, err
	}

	paymentMethod, err := parsePaymentMethod(row.PaymentMethod)
	if err != nil {
		return domain.Order{}, err
	}

	items, err := mapDBOrderItems(itemRows)
	if err != nil {
		return domain.Order{}, err
	}

	order := domain.Order{
		ID:         row.ID,
		CustomerID: row.CustomerID,
		Items:      items,
		TotalAmount: domain.Money{
			Currency:    totalCurrency,
			AmountCents: row.TotalAmountCents,
		},
		Status: status,
		ShippingAddress: domain.Address{
			Country:    row.ShippingCountry,
			City:       row.ShippingCity,
			Street:     row.ShippingStreet,
			PostalCode: row.ShippingPostalCode,
			House:      row.ShippingHouse,
			Apartment:  row.ShippingApartment,
		},
		Payment: domain.PaymentDetails{
			Method:        paymentMethod,
			MethodDetails: row.PaymentMethodDetails,
		},
	}

	if row.CreatedAt.Valid {
		order.CreatedAt = row.CreatedAt.Time
	}
	if row.UpdatedAt.Valid {
		order.UpdatedAt = row.UpdatedAt.Time
	}

	return order, nil
}

func mapDBOrderItems(rows []orderdb.OrderItem) ([]domain.OrderItem, error) {
	items := make([]domain.OrderItem, 0, len(rows))
	for _, row := range rows {
		item, err := mapDBOrderItem(row)
		if err != nil {
			return nil, err
		}

		items = append(items, item)
	}

	return items, nil
}

func mapDBOrderItem(row orderdb.OrderItem) (domain.OrderItem, error) {
	unitCurrency, err := parseCurrency(row.UnitPriceCurrency)
	if err != nil {
		return domain.OrderItem{}, err
	}

	totalCurrency, err := parseCurrency(row.TotalPriceCurrency)
	if err != nil {
		return domain.OrderItem{}, err
	}

	return domain.OrderItem{
		ProductID:   row.ProductID,
		SKU:         row.Sku,
		ProductName: row.ProductName,
		Quantity:    row.Quantity,
		UnitPrice: domain.Money{
			Currency:    unitCurrency,
			AmountCents: row.UnitPriceAmountCents,
		},
		TotalPrice: domain.Money{
			Currency:    totalCurrency,
			AmountCents: row.TotalPriceAmountCents,
		},
	}, nil
}

func parseCurrency(value string) (orderv1.Currency, error) {
	enum, ok := orderv1.Currency_value[value]
	if !ok {
		return orderv1.Currency_CURRENCY_UNSPECIFIED, fmt.Errorf("unknown order currency in db: %q", value)
	}

	return orderv1.Currency(enum), nil
}

func parseOrderStatus(value string) (orderv1.OrderStatus, error) {
	enum, ok := orderv1.OrderStatus_value[value]
	if !ok {
		return orderv1.OrderStatus_ORDER_STATUS_UNSPECIFIED, fmt.Errorf("unknown order status in db: %q", value)
	}

	return orderv1.OrderStatus(enum), nil
}

func parsePaymentMethod(value string) (orderv1.PaymentMethodType, error) {
	enum, ok := orderv1.PaymentMethodType_value[value]
	if !ok {
		return orderv1.PaymentMethodType_PAYMENT_METHOD_TYPE_UNSPECIFIED, fmt.Errorf("unknown order payment method in db: %q", value)
	}

	return orderv1.PaymentMethodType(enum), nil
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
