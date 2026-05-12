package order

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
