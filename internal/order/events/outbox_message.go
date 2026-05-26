package events

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/vladfc/event-driven-ecommerce-app/internal/order/domain"
	sharedevents "github.com/vladfc/event-driven-ecommerce-app/internal/shared/events"
)

type OutboxMessage struct {
	ID            string
	AggregateType string
	AggregateID   string
	EventType     string
	Topic         string
	Key           string
	Payload       []byte
	CreatedAt     time.Time
}

func NewOrderCreatedOutboxMessage(order domain.Order, topic string) (OutboxMessage, error) {
	event := sharedevents.OrderCreated{
		OrderID:    order.ID,
		CustomerID: order.CustomerID,
		Status:     order.Status.String(),
		TotalAmount: sharedevents.Money{
			Currency:    order.TotalAmount.Currency.String(),
			AmountCents: order.TotalAmount.AmountCents,
		},
		ShippingAddress: sharedevents.Address{
			Country:    order.ShippingAddress.Country,
			City:       order.ShippingAddress.City,
			Street:     order.ShippingAddress.Street,
			PostalCode: order.ShippingAddress.PostalCode,
			House:      order.ShippingAddress.House,
			Apartment:  order.ShippingAddress.Apartment,
		},
		Payment: sharedevents.PaymentDetails{
			Method:        order.Payment.Method.String(),
			MethodDetails: order.Payment.MethodDetails,
		},
		Items: make([]sharedevents.OrderCreatedItem, 0, len(order.Items)),
	}

	for _, item := range order.Items {
		event.Items = append(event.Items, sharedevents.OrderCreatedItem{
			ProductID:   item.ProductID,
			SKU:         item.SKU,
			ProductName: item.ProductName,
			Quantity:    item.Quantity,
			UnitPrice: sharedevents.Money{
				Currency:    item.UnitPrice.Currency.String(),
				AmountCents: item.UnitPrice.AmountCents,
			},
			TotalPrice: sharedevents.Money{
				Currency:    item.TotalPrice.Currency.String(),
				AmountCents: item.TotalPrice.AmountCents,
			},
		})
	}

	payload, err := json.Marshal(event)
	if err != nil {
		return OutboxMessage{}, fmt.Errorf("marshal order.created payload: %w", err)
	}

	createdAt := time.Now().UTC()
	envelopeBytes, err := json.Marshal(sharedevents.Envelope{
		EventID:    fmt.Sprintf("evt-%d", createdAt.UnixNano()),
		EventType:  sharedevents.OrderCreatedEventType,
		OccurredAt: createdAt.Format(time.RFC3339),
		Payload:    payload,
	})
	if err != nil {
		return OutboxMessage{}, fmt.Errorf("marshal order.created envelope: %w", err)
	}

	return OutboxMessage{
		ID:            fmt.Sprintf("outbox-%s", order.ID),
		AggregateType: "order",
		AggregateID:   order.ID,
		EventType:     sharedevents.OrderCreatedEventType,
		Topic:         topic,
		Key:           order.ID,
		Payload:       envelopeBytes,
		CreatedAt:     createdAt,
	}, nil
}

func NewOrderCancelledOutboxMessage(order domain.Order, topic string) (OutboxMessage, error) {
	event := sharedevents.OrderCancelled{
		OrderID:      order.ID,
		CustomerID:   order.CustomerID,
		CancelReason: order.CancelReason,
	}

	payload, err := json.Marshal(event)
	if err != nil {
		return OutboxMessage{}, fmt.Errorf("marshal order.cancelled payload: %w", err)
	}

	createdAt := time.Now().UTC()
	envelopeBytes, err := json.Marshal(sharedevents.Envelope{
		EventID:    fmt.Sprintf("evt-%d", createdAt.UnixNano()),
		EventType:  sharedevents.OrderCancelledEventType,
		OccurredAt: createdAt.Format(time.RFC3339),
		Payload:    payload,
	})
	if err != nil {
		return OutboxMessage{}, fmt.Errorf("marshal order.cancelled envelope: %w", err)
	}

	return OutboxMessage{
		ID:            fmt.Sprintf("outbox-cancelled-%s", order.ID),
		AggregateType: "order",
		AggregateID:   order.ID,
		EventType:     sharedevents.OrderCancelledEventType,
		Topic:         topic,
		Key:           order.ID,
		Payload:       envelopeBytes,
		CreatedAt:     createdAt,
	}, nil
}
