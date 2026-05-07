package service

import (
	"context"
	"errors"
	"strings"

	orderv1 "github.com/vladfc/event-driven-ecommerce-app/gen/order/v1"
	paymentv1 "github.com/vladfc/event-driven-ecommerce-app/gen/payment/v1"
	inventoryclient "github.com/vladfc/event-driven-ecommerce-app/internal/gateway/client/inventory"
	orderclient "github.com/vladfc/event-driven-ecommerce-app/internal/gateway/client/order"
	paymentclient "github.com/vladfc/event-driven-ecommerce-app/internal/gateway/client/payment"
)

type OrderClient interface {
	CreateOrder(ctx context.Context, req *orderclient.CreateOrderRequest) (*orderclient.CreateOrderResponse, error)
	GetOrder(ctx context.Context, orderID string) (*orderclient.GetOrderResponse, error)
	CancelOrder(ctx context.Context, req *orderclient.CancelOrderRequest) (*orderclient.CancelOrderResponse, error)
}

type InventoryClient interface {
	ReserveStock(ctx context.Context, req *inventoryclient.ReserveStockRequest) (*inventoryclient.ReserveStockResponse, error)
	ReleaseStock(ctx context.Context, req *inventoryclient.ReleaseStockRequest) (*inventoryclient.ReleaseStockResponse, error)
}

type PaymentClient interface {
	CreatePayment(ctx context.Context, req *paymentclient.CreatePaymentRequest) (*paymentclient.CreatePaymentResponse, error)
}

type GatewayService struct {
	orderClient     OrderClient
	inventoryClient InventoryClient
	paymentClient   PaymentClient
}

type Option func(*GatewayService)

func WithInventoryClient(client InventoryClient) Option {
	return func(s *GatewayService) {
		s.inventoryClient = client
	}
}

func WithPaymentClient(client PaymentClient) Option {
	return func(s *GatewayService) {
		s.paymentClient = client
	}
}

func NewGatewayService(orderClient OrderClient, opts ...Option) *GatewayService {
	service := &GatewayService{
		orderClient: orderClient,
	}

	for _, opt := range opts {
		if opt != nil {
			opt(service)
		}
	}

	return service
}

type CheckoutInput struct {
	CustomerID      string
	Items           []CheckoutItem
	ShippingAddress Address
	IdempotencyKey  string
	Payment         PaymentDetails
}

type CheckoutItem struct {
	ProductID   string
	SKU         string
	ProductName string
	Quantity    int32
	UnitPrice   Money
	TotalPrice  Money
}

type Money struct {
	Currency    string
	AmountCents int64
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
	Method        string
	MethodDetails string
}

type CheckoutResult struct {
	OrderID       string
	PaymentID     string
	OrderStatus   string
	PaymentStatus string
}

type GetOrderByIDInput struct {
	OrderID string
}

type GetOrderByIDResult struct {
	OrderID         string
	CustomerID      string
	OrderStatus     string
	Items           []CheckoutItem
	TotalAmount     Money
	ShippingAddress Address
	CreatedAt       string
	UpdatedAt       string
}

func (s *GatewayService) Checkout(ctx context.Context, in *CheckoutInput) (*CheckoutResult, error) {
	if in == nil {
		return nil, errors.New("checkout request is nil")
	}
	if strings.TrimSpace(in.CustomerID) == "" {
		return nil, errors.New("customer id is required")
	}
	if len(in.Items) == 0 {
		return nil, errors.New("at least one item is required")
	}
	if strings.TrimSpace(in.ShippingAddress.Country) == "" ||
		strings.TrimSpace(in.ShippingAddress.City) == "" ||
		strings.TrimSpace(in.ShippingAddress.Street) == "" ||
		strings.TrimSpace(in.ShippingAddress.PostalCode) == "" ||
		strings.TrimSpace(in.ShippingAddress.House) == "" {
		return nil, errors.New("complete shipping address is required")
	}
	if strings.TrimSpace(in.Payment.Method) == "" {
		return nil, errors.New("payment method is required")
	}
	if s.orderClient == nil {
		return nil, errors.New("order client is not configured")
	}
	if s.inventoryClient == nil {
		return nil, errors.New("inventory client is not configured")
	}
	if s.paymentClient == nil {
		return nil, errors.New("payment client is not configured")
	}

	orderItems, err := mapCheckoutItemsToOrderItems(in.Items)
	if err != nil {
		return nil, err
	}

	orderResp, err := s.orderClient.CreateOrder(ctx, &orderclient.CreateOrderRequest{
		CustomerID:      strings.TrimSpace(in.CustomerID),
		Items:           orderItems,
		ShippingAddress: mapAddressToOrderProto(in.ShippingAddress),
		IdempotencyKey:  strings.TrimSpace(in.IdempotencyKey),
	})
	if err != nil {
		return nil, err
	}
	if orderResp == nil || orderResp.Order == nil {
		return nil, errors.New("order response is empty")
	}

	order := orderResp.Order
	reservedItems := make([]*orderv1.OrderItem, 0, len(order.GetItems()))
	for _, item := range order.GetItems() {
		if item == nil {
			continue
		}

		_, err := s.inventoryClient.ReserveStock(ctx, &inventoryclient.ReserveStockRequest{
			ProductID: item.GetProductId(),
			Quantity:  int64(item.GetQuantity()),
			OrderID:   order.GetOrderId(),
		})
		if err != nil {
			return nil, s.compensateCheckoutFailure(ctx, order.GetOrderId(), reservedItems, err)
		}

		reservedItems = append(reservedItems, item)
	}

	totalAmount := order.GetTotalAmount()
	if totalAmount == nil {
		return nil, s.compensateCheckoutFailure(ctx, order.GetOrderId(), reservedItems, errors.New("order total amount is empty"))
	}

	paymentMethod, err := parsePaymentMethod(in.Payment.Method)
	if err != nil {
		return nil, s.compensateCheckoutFailure(ctx, order.GetOrderId(), reservedItems, err)
	}

	paymentCurrency, err := mapOrderCurrencyToPayment(totalAmount.GetCurrency())
	if err != nil {
		return nil, s.compensateCheckoutFailure(ctx, order.GetOrderId(), reservedItems, err)
	}

	paymentResp, err := s.paymentClient.CreatePayment(ctx, &paymentclient.CreatePaymentRequest{
		OrderID:    order.GetOrderId(),
		CustomerID: order.GetCustomerId(),
		Amount: &paymentv1.Money{
			Currency:    paymentCurrency,
			AmountCents: totalAmount.GetAmountCents(),
		},
		PaymentMethod:        paymentMethod,
		PaymentMethodDetails: strings.TrimSpace(in.Payment.MethodDetails),
		IdempotencyKey:       strings.TrimSpace(in.IdempotencyKey),
	})
	if err != nil {
		return nil, s.compensateCheckoutFailure(ctx, order.GetOrderId(), reservedItems, err)
	}
	if paymentResp == nil || paymentResp.Payment == nil {
		return nil, s.compensateCheckoutFailure(ctx, order.GetOrderId(), reservedItems, errors.New("payment response is empty"))
	}

	return &CheckoutResult{
		OrderID:       order.GetOrderId(),
		PaymentID:     paymentResp.Payment.GetPaymentId(),
		OrderStatus:   order.GetStatus().String(),
		PaymentStatus: paymentResp.Payment.GetStatus().String(),
	}, nil
}

func (s *GatewayService) GetOrderByID(ctx context.Context, in *GetOrderByIDInput) (*GetOrderByIDResult, error) {
	if in == nil {
		return nil, errors.New("get order request is nil")
	}
	if strings.TrimSpace(in.OrderID) == "" {
		return nil, errors.New("order id is required")
	}

	resp, err := s.orderClient.GetOrder(ctx, in.OrderID)
	if err != nil {
		return nil, err
	}
	if resp == nil || resp.Order == nil {
		return nil, errors.New("order response is empty")
	}

	order := resp.Order
	result := &GetOrderByIDResult{
		OrderID:         order.GetOrderId(),
		CustomerID:      order.GetCustomerId(),
		OrderStatus:     order.GetStatus().String(),
		Items:           make([]CheckoutItem, 0, len(order.GetItems())),
		TotalAmount:     mapProtoMoney(order.GetTotalAmount()),
		ShippingAddress: mapProtoAddress(order.GetShippingAddress()),
		CreatedAt:       order.GetCreatedAt(),
		UpdatedAt:       order.GetUpdatedAt(),
	}

	for _, item := range order.GetItems() {
		result.Items = append(result.Items, CheckoutItem{
			ProductID:   item.GetProductId(),
			SKU:         item.GetSku(),
			ProductName: item.GetProductName(),
			Quantity:    item.GetQuantity(),
			UnitPrice:   mapProtoMoney(item.GetUnitPrice()),
			TotalPrice:  mapProtoMoney(item.GetTotalPrice()),
		})
	}

	return result, nil
}

func mapProtoMoney(money *orderv1.Money) Money {
	if money == nil {
		return Money{}
	}

	return Money{
		Currency:    money.GetCurrency().String(),
		AmountCents: money.GetAmountCents(),
	}
}

func mapProtoAddress(address *orderv1.Address) Address {
	if address == nil {
		return Address{}
	}

	return Address{
		Country:    address.GetCountry(),
		City:       address.GetCity(),
		Street:     address.GetStreet(),
		PostalCode: address.GetPostalCode(),
		House:      address.GetHouse(),
		Apartment:  address.GetApartment(),
	}
}

func mapCheckoutItemsToOrderItems(items []CheckoutItem) ([]*orderv1.CreateOrderItem, error) {
	converted := make([]*orderv1.CreateOrderItem, 0, len(items))
	for _, item := range items {
		if strings.TrimSpace(item.ProductID) == "" || item.Quantity <= 0 || item.UnitPrice.AmountCents <= 0 {
			return nil, errors.New("invalid checkout item")
		}

		currency, err := parseOrderCurrency(item.UnitPrice.Currency)
		if err != nil {
			return nil, err
		}

		converted = append(converted, &orderv1.CreateOrderItem{
			ProductId:   strings.TrimSpace(item.ProductID),
			Sku:         strings.TrimSpace(item.SKU),
			ProductName: strings.TrimSpace(item.ProductName),
			Quantity:    item.Quantity,
			UnitPrice: &orderv1.Money{
				Currency:    currency,
				AmountCents: item.UnitPrice.AmountCents,
			},
		})
	}

	return converted, nil
}

func mapAddressToOrderProto(address Address) *orderv1.Address {
	return &orderv1.Address{
		Country:    strings.TrimSpace(address.Country),
		City:       strings.TrimSpace(address.City),
		Street:     strings.TrimSpace(address.Street),
		PostalCode: strings.TrimSpace(address.PostalCode),
		House:      strings.TrimSpace(address.House),
		Apartment:  strings.TrimSpace(address.Apartment),
	}
}

func parseOrderCurrency(value string) (orderv1.Currency, error) {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "USD", "CURRENCY_USD":
		return orderv1.Currency_CURRENCY_USD, nil
	case "EUR", "CURRENCY_EUR":
		return orderv1.Currency_CURRENCY_EUR, nil
	default:
		return orderv1.Currency_CURRENCY_UNSPECIFIED, errors.New("unsupported currency")
	}
}

func mapOrderCurrencyToPayment(currency orderv1.Currency) (paymentv1.Currency, error) {
	switch currency {
	case orderv1.Currency_CURRENCY_USD:
		return paymentv1.Currency_CURRENCY_USD, nil
	case orderv1.Currency_CURRENCY_EUR:
		return paymentv1.Currency_CURRENCY_EUR, nil
	default:
		return paymentv1.Currency_CURRENCY_UNSPECIFIED, errors.New("unsupported payment currency")
	}
}

func parsePaymentMethod(value string) (paymentv1.PaymentMethodType, error) {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "CARD", "PAYMENT_METHOD_TYPE_CARD":
		return paymentv1.PaymentMethodType_PAYMENT_METHOD_TYPE_CARD, nil
	case "CASH", "PAYMENT_METHOD_TYPE_CASH":
		return paymentv1.PaymentMethodType_PAYMENT_METHOD_TYPE_CASH, nil
	default:
		return paymentv1.PaymentMethodType_PAYMENT_METHOD_TYPE_UNSPECIFIED, errors.New("unsupported payment method")
	}
}

func (s *GatewayService) compensateCheckoutFailure(ctx context.Context, orderID string, reservedItems []*orderv1.OrderItem, originalErr error) error {
	compensationErrs := make([]error, 0, len(reservedItems)+1)

	for i := len(reservedItems) - 1; i >= 0; i-- {
		item := reservedItems[i]
		if item == nil {
			continue
		}

		_, err := s.inventoryClient.ReleaseStock(ctx, &inventoryclient.ReleaseStockRequest{
			ProductID: item.GetProductId(),
			Quantity:  int64(item.GetQuantity()),
			OrderID:   orderID,
		})
		if err != nil {
			compensationErrs = append(compensationErrs, err)
		}
	}

	if strings.TrimSpace(orderID) != "" {
		_, err := s.orderClient.CancelOrder(ctx, &orderclient.CancelOrderRequest{
			OrderID: orderID,
			Reason:  "checkout failed",
		})
		if err != nil {
			compensationErrs = append(compensationErrs, err)
		}
	}

	if len(compensationErrs) == 0 {
		return originalErr
	}

	compensationErrs = append([]error{originalErr}, compensationErrs...)
	return errors.Join(compensationErrs...)
}
