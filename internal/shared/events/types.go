package events

import "encoding/json"

const (
	OrderCreatedEventType               = "order.created"
	InventoryReservedEventType          = "inventory.reserved"
	InventoryReservationFailedEventType = "inventory.reservation_failed"
)

type Envelope struct {
	EventID    string          `json:"event_id"`
	EventType  string          `json:"event_type"`
	OccurredAt string          `json:"occurred_at"`
	Payload    json.RawMessage `json:"payload"`
}

type OrderCreated struct {
	OrderID         string             `json:"order_id"`
	CustomerID      string             `json:"customer_id"`
	Status          string             `json:"status"`
	TotalAmount     Money              `json:"total_amount"`
	ShippingAddress Address            `json:"shipping_address"`
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

type InventoryReservationItem struct {
	ProductID string `json:"product_id"`
	Quantity  int64  `json:"quantity"`
}

type InventoryReserved struct {
	OrderID    string                     `json:"order_id"`
	CustomerID string                     `json:"customer_id"`
	Items      []InventoryReservationItem `json:"items"`
}

type InventoryReservationFailed struct {
	OrderID    string                    `json:"order_id"`
	CustomerID string                    `json:"customer_id"`
	FailedItem *InventoryReservationItem `json:"failed_item,omitempty"`
	Reason     string                    `json:"reason"`
}
