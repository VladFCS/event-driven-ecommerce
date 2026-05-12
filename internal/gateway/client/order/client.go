package order

import (
	"context"
	"errors"
	"fmt"
	"strings"

	orderv1 "github.com/vladfc/event-driven-ecommerce-app/gen/order/v1"
	"github.com/vladfc/event-driven-ecommerce-app/internal/shared/requestid"
	"google.golang.org/grpc"
)

var (
	ErrCreateOrderRequestNil    = errors.New("create order request is nil")
	ErrCancelOrderRequestNil    = errors.New("cancel order request is nil")
	ErrListOrdersRequestNil     = errors.New("list orders by customer request is nil")
	ErrOrderIDRequired          = errors.New("order id is required")
	ErrCustomerIDRequired       = errors.New("customer id is required")
	ErrUnsupportedOrderCurrency = errors.New("unsupported order currency")
)

type Client interface {
	CreateOrder(ctx context.Context, req *CreateOrderRequest) (*CreateOrderResponse, error)
	CancelOrder(ctx context.Context, req *CancelOrderRequest) (*CancelOrderResponse, error)
	GetOrderByID(ctx context.Context, orderID string) (*GetOrderByIDResponse, error)
	ListOrdersByCustomer(ctx context.Context, req *ListOrdersByCustomerRequest) (*ListOrdersByCustomerResponse, error)
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

type CreateOrderItem struct {
	ProductID   string
	SKU         string
	ProductName string
	Quantity    int32
	UnitPrice   Money
}

type OrderItem struct {
	ProductID   string
	SKU         string
	ProductName string
	Quantity    int32
	UnitPrice   Money
	TotalPrice  Money
}

type Order struct {
	ID              string
	CustomerID      string
	Items           []OrderItem
	TotalAmount     Money
	Status          string
	ShippingAddress Address
	CreatedAt       string
	UpdatedAt       string
}

type CreateOrderRequest struct {
	CustomerID      string
	Items           []CreateOrderItem
	ShippingAddress Address
	IdempotencyKey  string
}

type CreateOrderResponse struct {
	Order *Order
}

type CancelOrderRequest struct {
	OrderID string
	Reason  string
}

type CancelOrderResponse struct {
	Order *Order
}

type GetOrderByIDResponse struct {
	Order *Order
}

type ListOrdersByCustomerRequest struct {
	CustomerID string
	Page       int
	PageSize   int
}

type ListOrdersByCustomerResponse struct {
	Orders   []Order
	Page     int
	PageSize int
	Total    int64
}

type GRPCClient struct {
	grpcClient orderv1.OrderServiceClient
}

func NewClient(conn grpc.ClientConnInterface) *GRPCClient {
	return &GRPCClient{
		grpcClient: orderv1.NewOrderServiceClient(conn),
	}
}

func (c *GRPCClient) CreateOrder(ctx context.Context, req *CreateOrderRequest) (*CreateOrderResponse, error) {
	if req == nil {
		return nil, ErrCreateOrderRequestNil
	}

	items, err := mapCreateOrderItemsToProto(req.Items)
	if err != nil {
		return nil, err
	}

	grpcResp, err := c.grpcClient.CreateOrder(requestid.WithOutgoingMetadata(ctx), &orderv1.CreateOrderRequest{
		CustomerId:      req.CustomerID,
		Items:           items,
		ShippingAddress: mapAddressToProto(req.ShippingAddress),
		IdempotencyKey:  req.IdempotencyKey,
	})
	if err != nil {
		return nil, err
	}

	return &CreateOrderResponse{
		Order: mapProtoOrder(grpcResp.GetOrder()),
	}, nil
}

func (c *GRPCClient) GetOrderByID(ctx context.Context, orderID string) (*GetOrderByIDResponse, error) {
	if strings.TrimSpace(orderID) == "" {
		return nil, ErrOrderIDRequired
	}

	grpcResp, err := c.grpcClient.GetOrder(requestid.WithOutgoingMetadata(ctx), &orderv1.GetOrderRequest{
		OrderId: orderID,
	})
	if err != nil {
		return nil, err
	}

	return &GetOrderByIDResponse{
		Order: mapProtoOrder(grpcResp.GetOrder()),
	}, nil
}

func (c *GRPCClient) ListOrdersByCustomer(ctx context.Context, req *ListOrdersByCustomerRequest) (*ListOrdersByCustomerResponse, error) {
	if req == nil {
		return nil, ErrListOrdersRequestNil
	}
	if strings.TrimSpace(req.CustomerID) == "" {
		return nil, ErrCustomerIDRequired
	}

	grpcResp, err := c.grpcClient.ListOrdersByCustomer(requestid.WithOutgoingMetadata(ctx), &orderv1.ListOrdersByCustomerRequest{
		CustomerId: req.CustomerID,
		Page:       int32(req.Page),
		PageSize:   int32(req.PageSize),
	})
	if err != nil {
		return nil, err
	}

	return &ListOrdersByCustomerResponse{
		Orders:   mapProtoOrders(grpcResp.GetOrders()),
		Page:     int(grpcResp.GetPage()),
		PageSize: int(grpcResp.GetPageSize()),
		Total:    grpcResp.GetTotal(),
	}, nil
}

func (c *GRPCClient) CancelOrder(ctx context.Context, req *CancelOrderRequest) (*CancelOrderResponse, error) {
	if req == nil {
		return nil, ErrCancelOrderRequestNil
	}

	grpcResp, err := c.grpcClient.CancelOrder(requestid.WithOutgoingMetadata(ctx), &orderv1.CancelOrderRequest{
		OrderId: req.OrderID,
		Reason:  req.Reason,
	})
	if err != nil {
		return nil, err
	}

	return &CancelOrderResponse{
		Order: mapProtoOrder(grpcResp.GetOrder()),
	}, nil
}

func mapCreateOrderItemsToProto(items []CreateOrderItem) ([]*orderv1.CreateOrderItem, error) {
	converted := make([]*orderv1.CreateOrderItem, 0, len(items))
	for _, item := range items {
		unitPrice, err := mapMoneyToProto(item.UnitPrice)
		if err != nil {
			return nil, err
		}

		converted = append(converted, &orderv1.CreateOrderItem{
			ProductId:   strings.TrimSpace(item.ProductID),
			Sku:         strings.TrimSpace(item.SKU),
			ProductName: strings.TrimSpace(item.ProductName),
			Quantity:    item.Quantity,
			UnitPrice:   unitPrice,
		})
	}

	return converted, nil
}

func mapAddressToProto(address Address) *orderv1.Address {
	return &orderv1.Address{
		Country:    strings.TrimSpace(address.Country),
		City:       strings.TrimSpace(address.City),
		Street:     strings.TrimSpace(address.Street),
		PostalCode: strings.TrimSpace(address.PostalCode),
		House:      strings.TrimSpace(address.House),
		Apartment:  strings.TrimSpace(address.Apartment),
	}
}

func mapMoneyToProto(money Money) (*orderv1.Money, error) {
	currency, err := parseCurrency(money.Currency)
	if err != nil {
		return nil, err
	}

	return &orderv1.Money{
		Currency:    currency,
		AmountCents: money.AmountCents,
	}, nil
}

func mapProtoOrder(order *orderv1.Order) *Order {
	if order == nil {
		return nil
	}

	return &Order{
		ID:              order.GetOrderId(),
		CustomerID:      order.GetCustomerId(),
		Items:           mapProtoOrderItems(order.GetItems()),
		TotalAmount:     mapProtoMoney(order.GetTotalAmount()),
		Status:          order.GetStatus().String(),
		ShippingAddress: mapProtoAddress(order.GetShippingAddress()),
		CreatedAt:       order.GetCreatedAt(),
		UpdatedAt:       order.GetUpdatedAt(),
	}
}

func mapProtoOrders(orders []*orderv1.Order) []Order {
	converted := make([]Order, 0, len(orders))
	for _, order := range orders {
		if mapped := mapProtoOrder(order); mapped != nil {
			converted = append(converted, *mapped)
		}
	}

	return converted
}

func mapProtoOrderItems(items []*orderv1.OrderItem) []OrderItem {
	converted := make([]OrderItem, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}

		converted = append(converted, OrderItem{
			ProductID:   item.GetProductId(),
			SKU:         item.GetSku(),
			ProductName: item.GetProductName(),
			Quantity:    item.GetQuantity(),
			UnitPrice:   mapProtoMoney(item.GetUnitPrice()),
			TotalPrice:  mapProtoMoney(item.GetTotalPrice()),
		})
	}

	return converted
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

func parseCurrency(value string) (orderv1.Currency, error) {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "USD", "CURRENCY_USD":
		return orderv1.Currency_CURRENCY_USD, nil
	case "EUR", "CURRENCY_EUR":
		return orderv1.Currency_CURRENCY_EUR, nil
	default:
		return orderv1.Currency_CURRENCY_UNSPECIFIED, fmt.Errorf("%w: %q", ErrUnsupportedOrderCurrency, value)
	}
}
