package service

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

type CancelOrderInput struct {
	OrderID        string
	Reason         string
	IdempotencyKey string
}

type CancelOrderResult struct {
	OrderID     string
	CustomerID  string
	OrderStatus string
	UpdatedAt   string
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

type GetPaymentByIDInput struct {
	PaymentID string
}

type GetPaymentByIDResult struct {
	PaymentID     string
	OrderID       string
	CustomerID    string
	Status        string
	Amount        Money
	PaymentMethod string
}

type CancelPaymentInput struct {
	PaymentID string
	Reason    string
}

type CancelPaymentResult struct {
	PaymentID  string
	OrderID    string
	CustomerID string
	Status     string
}

type ListOrdersByCustomerInput struct {
	CustomerID string
	Page       int
	PageSize   int
}

type ListOrdersByCustomerResult struct {
	Orders   []GetOrderByIDResult
	Total    int64
	Page     int
	PageSize int
}

type GetProductByIDInput struct {
	ProductID string
}

type GetProductByIDResult struct {
	ProductID   string
	Name        string
	Description string
	PriceCents  int64
	Currency    string
}

type ListProductsInput struct {
	Page     int
	PageSize int
}

type ProductResult struct {
	ProductID   string
	Name        string
	Description string
	PriceCents  int64
	Currency    string
}

type ListProductsResult struct {
	Products []ProductResult
	Page     int
	PageSize int
	Total    int64
}

type GetStockByProductIDInput struct {
	ProductID string
}

type GetStockByProductIDResult struct {
	ProductID string
	Available int64
	Reserved  int64
}
