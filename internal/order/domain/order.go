package domain

import (
	"errors"
	"time"

	orderv1 "github.com/vladfc/event-driven-ecommerce-app/gen/order/v1"
)

var (
	ErrOrderNotFound      = errors.New("order not found")
	ErrInvalidOrder       = errors.New("invalid order")
	ErrOrderAlreadyExists = errors.New("order already exists")
	ErrInvalidOrderID = errors.New("invalid order id")
	ErrInvalidCustomerID = errors.New("invalid customer id")
)

type Money struct {
	Currency    orderv1.Currency
	AmountCents int64
}

type OrderItem struct {
	ProductID   string
	SKU         string
	ProductName string
	Quantity    int32
	UnitPrice   Money
	TotalPrice  Money
}

type Address struct {
	Country    string
	City       string
	Street     string
	PostalCode string
	House      string
	Apartment  string
}

type Order struct {
	ID              string
	CustomerID      string
	Items           []OrderItem
	TotalAmount     Money
	Status          orderv1.OrderStatus
	ShippingAddress Address
	CreatedAt       time.Time
	UpdatedAt       time.Time
}
