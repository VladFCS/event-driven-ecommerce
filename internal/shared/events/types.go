package events

import "encoding/json"

const (
	OrderCreatedEventType               = "order.created"
	InventoryReservedEventType          = "inventory.reserved"
	InventoryReservationFailedEventType = "inventory.reservation_failed"
	PaymentCreatedEventType             = "payment.created"
	PaymentCreationFailedEventType      = "payment.creation_failed"
)

type Envelope struct {
	EventID    string          `json:"event_id"`
	EventType  string          `json:"event_type"`
	OccurredAt string          `json:"occurred_at"`
	Payload    json.RawMessage `json:"payload"`
}

type Money struct {
	Currency    string `json:"currency"`
	AmountCents int64  `json:"amount_cents"`
}

type Address struct {
	Country    string `json:"country"`
	City       string `json:"city"`
	Street     string `json:"street"`
	PostalCode string `json:"postal_code"`
	House      string `json:"house"`
	Apartment  string `json:"apartment"`
}

type PaymentDetails struct {
	Method        string `json:"method"`
	MethodDetails string `json:"method_details"`
}

type OrderCreated struct {
	OrderID         string             `json:"order_id"`
	CustomerID      string             `json:"customer_id"`
	Status          string             `json:"status"`
	TotalAmount     Money              `json:"total_amount"`
	ShippingAddress Address            `json:"shipping_address"`
	Payment         PaymentDetails     `json:"payment"`
	Items           []OrderCreatedItem `json:"items"`
}

type OrderCreatedItem struct {
	ProductID   string `json:"product_id"`
	SKU         string `json:"sku"`
	ProductName string `json:"product_name"`
	Quantity    int32  `json:"quantity"`
	UnitPrice   Money  `json:"unit_price"`
	TotalPrice  Money  `json:"total_price"`
}

type InventoryReservationItem struct {
	ProductID string `json:"product_id"`
	Quantity  int64  `json:"quantity"`
}

type InventoryReserved struct {
	OrderID    string                     `json:"order_id"`
	CustomerID string                     `json:"customer_id"`
	Amount     Money                      `json:"amount"`
	Payment    PaymentDetails             `json:"payment"`
	Items      []InventoryReservationItem `json:"items"`
}

type InventoryReservationFailed struct {
	OrderID    string                    `json:"order_id"`
	CustomerID string                    `json:"customer_id"`
	FailedItem *InventoryReservationItem `json:"failed_item,omitempty"`
	Reason     string                    `json:"reason"`
}

type PaymentCreated struct {
	OrderID    string `json:"order_id"`
	CustomerID string `json:"customer_id"`
	PaymentID  string `json:"payment_id"`
	Status     string `json:"status"`
}

type PaymentCreationFailed struct {
	OrderID       string `json:"order_id"`
	CustomerID    string `json:"customer_id"`
	FailureReason string `json:"failure_reason"`
}
