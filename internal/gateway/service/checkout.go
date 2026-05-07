package service

import (
	"context"
	"errors"
	"strings"

	orderv1 "github.com/vladfc/event-driven-ecommerce-app/gen/order/v1"
	orderclient "github.com/vladfc/event-driven-ecommerce-app/internal/gateway/client/order"
)

var ErrCheckoutNotImplemented = errors.New("checkout not implemented")

type OrderClient interface {
	CreateOrder(ctx context.Context, req *orderclient.CreateOrderRequest) (*orderclient.CreateOrderResponse, error)
	GetOrder(ctx context.Context, orderID string) (*orderclient.GetOrderResponse, error)
	CancelOrder(ctx context.Context, req *orderclient.CancelOrderRequest) (*orderclient.CancelOrderResponse, error)
}

type GatewayService struct {
	orderClient OrderClient
}

func NewGatewayService(orderClient OrderClient) *GatewayService {
	return &GatewayService{
		orderClient: orderClient,
	}
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
	return nil, ErrCheckoutNotImplemented
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
