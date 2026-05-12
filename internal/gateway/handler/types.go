package handler

type HealthResponse struct {
	Status              string   `json:"status"`
	Service             string   `json:"service"`
	MissingDependencies []string `json:"missing_dependencies,omitempty"`
}

type CheckoutRequest struct {
	CustomerID      string                `json:"customer_id" binding:"required"`
	Items           []CheckoutItemRequest `json:"items" binding:"required,min=1"`
	ShippingAddress AddressRequest        `json:"shipping_address" binding:"required"`
	Payment         PaymentRequest        `json:"payment" binding:"required"`
	IdempotencyKey  string                `json:"idempotency_key"`
}

type GetOrderByIDRequest struct {
	OrderID string `uri:"order_id" binding:"required"`
}

type GetOrderByIDResponse struct {
	OrderID         string              `json:"order_id"`
	CustomerID      string              `json:"customer_id"`
	OrderStatus     string              `json:"order_status"`
	Items           []OrderItemResponse `json:"items"`
	TotalAmount     MoneyResponse       `json:"total_amount"`
	ShippingAddress AddressResponse     `json:"shipping_address"`
	CreatedAt       string              `json:"created_at"`
	UpdatedAt       string              `json:"updated_at"`
}

type CheckoutItemRequest struct {
	ProductID   string       `json:"product_id" binding:"required"`
	SKU         string       `json:"sku"`
	ProductName string       `json:"product_name"`
	Quantity    int32        `json:"quantity" binding:"required,gt=0"`
	UnitPrice   MoneyRequest `json:"unit_price" binding:"required"`
}

type MoneyRequest struct {
	Currency    string `json:"currency" binding:"required"`
	AmountCents int64  `json:"amount_cents" binding:"required,gt=0"`
}

type AddressRequest struct {
	Country    string `json:"country" binding:"required"`
	City       string `json:"city" binding:"required"`
	Street     string `json:"street" binding:"required"`
	PostalCode string `json:"postal_code" binding:"required"`
	House      string `json:"house" binding:"required"`
	Apartment  string `json:"apartment"`
}

type PaymentRequest struct {
	Method        string `json:"method" binding:"required"`
	MethodDetails string `json:"method_details"`
}

type OrderItemResponse struct {
	ProductID   string        `json:"product_id"`
	SKU         string        `json:"sku"`
	ProductName string        `json:"product_name"`
	Quantity    int32         `json:"quantity"`
	UnitPrice   MoneyResponse `json:"unit_price"`
	TotalPrice  MoneyResponse `json:"total_price"`
}

type MoneyResponse struct {
	Currency    string `json:"currency"`
	AmountCents int64  `json:"amount_cents"`
}

type AddressResponse struct {
	Country    string `json:"country"`
	City       string `json:"city"`
	Street     string `json:"street"`
	PostalCode string `json:"postal_code"`
	House      string `json:"house"`
	Apartment  string `json:"apartment"`
}

type CheckoutResponse struct {
	OrderID       string `json:"order_id"`
	PaymentID     string `json:"payment_id"`
	OrderStatus   string `json:"order_status"`
	PaymentStatus string `json:"payment_status"`
}
