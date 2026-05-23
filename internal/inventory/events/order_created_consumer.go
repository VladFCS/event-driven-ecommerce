package events

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"

	"github.com/segmentio/kafka-go"
	"github.com/vladfc/event-driven-ecommerce-app/internal/inventory/service"
	sharedevents "github.com/vladfc/event-driven-ecommerce-app/internal/shared/events"
)

type OrderCreatedConsumer struct {
	reader    *kafka.Reader
	service   *service.InventoryService
	publisher Publisher
	logger    *slog.Logger

	mu                sync.Mutex
	processedEventIDs map[string]struct{}
}

func NewOrderCreatedConsumer(
	brokers []string,
	topic string,
	groupID string,
	service *service.InventoryService,
	publisher Publisher,
	logger *slog.Logger,
) *OrderCreatedConsumer {
	if logger == nil {
		logger = slog.Default()
	}

	return &OrderCreatedConsumer{
		reader: kafka.NewReader(kafka.ReaderConfig{
			Brokers: brokers,
			Topic:   topic,
			GroupID: groupID,
		}),
		service:           service,
		publisher:         publisher,
		logger:            logger,
		processedEventIDs: make(map[string]struct{}),
	}
}

func (c *OrderCreatedConsumer) Close() error {
	if c == nil || c.reader == nil {
		return nil
	}
	return c.reader.Close()
}

func (c *OrderCreatedConsumer) Run(ctx context.Context) error {
	for {
		msg, err := c.reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return fmt.Errorf("fetch kafka message: %w", err)
		}

		commit, err := c.handleMessage(ctx, msg)
		if err != nil {
			return err
		}

		if commit {
			if err := c.reader.CommitMessages(ctx, msg); err != nil {
				return fmt.Errorf("commit kafka message: %w", err)
			}
		}
	}
}

func (c *OrderCreatedConsumer) handleMessage(ctx context.Context, msg kafka.Message) (bool, error) {
	var envelope sharedevents.Envelope
	if err := json.Unmarshal(msg.Value, &envelope); err != nil {
		c.logger.ErrorContext(ctx, "failed to decode event envelope", slog.Any("error", err))
		return true, nil
	}

	if envelope.EventType != sharedevents.OrderCreatedEventType {
		c.logger.WarnContext(ctx, "skipping unsupported event type", slog.String("event_type", envelope.EventType))
		return true, nil
	}

	if c.isProcessed(envelope.EventID) {
		c.logger.InfoContext(ctx, "skipping duplicate event", slog.String("event_id", envelope.EventID))
		return true, nil
	}

	var event sharedevents.OrderCreated
	if err := json.Unmarshal(envelope.Payload, &event); err != nil {
		c.logger.ErrorContext(ctx, "failed to decode order.created payload", slog.Any("error", err))
		return true, nil
	}

	reservedItems, failedItem, reservationErr, rollbackErr := c.reserveAll(ctx, event)
	if rollbackErr != nil {
		return false, rollbackErr
	}

	if reservationErr != nil {
		err := c.publisher.PublishInventoryReservationFailed(ctx, envelope.EventID, sharedevents.InventoryReservationFailed{
			OrderID:    event.OrderID,
			CustomerID: event.CustomerID,
			FailedItem: failedItem,
			Reason:     reservationErr.Error(),
		})
		if err != nil {
			return false, err
		}

		c.markProcessed(envelope.EventID)
		return true, nil
	}

	err := c.publisher.PublishInventoryReserved(ctx, envelope.EventID, sharedevents.InventoryReserved{
		OrderID:    event.OrderID,
		CustomerID: event.CustomerID,
		Items:      reservedItems,
	})
	if err != nil {
		return false, err
	}

	c.markProcessed(envelope.EventID)
	return true, nil
}

func (c *OrderCreatedConsumer) reserveAll(ctx context.Context, event sharedevents.OrderCreated) ([]sharedevents.InventoryReservationItem, *sharedevents.InventoryReservationItem, error, error) {
	reservedItems := make([]sharedevents.InventoryReservationItem, 0, len(event.Items))

	for _, item := range event.Items {
		_, err := c.service.ReserveStock(ctx, item.ProductID, int64(item.Quantity), event.OrderID)
		if err != nil {
			failedItem := &sharedevents.InventoryReservationItem{
				ProductID: item.ProductID,
				Quantity:  int64(item.Quantity),
			}

			if rollbackErr := c.releaseReserved(ctx, event.OrderID, reservedItems); rollbackErr != nil {
				return nil, failedItem, err, rollbackErr
			}

			return nil, failedItem, err, nil
		}

		reservedItems = append(reservedItems, sharedevents.InventoryReservationItem{
			ProductID: item.ProductID,
			Quantity:  int64(item.Quantity),
		})
	}

	return reservedItems, nil, nil, nil
}

func (c *OrderCreatedConsumer) releaseReserved(ctx context.Context, orderID string, reservedItems []sharedevents.InventoryReservationItem) error {
	for i := len(reservedItems) - 1; i >= 0; i-- {
		item := reservedItems[i]
		if _, err := c.service.ReleaseStock(ctx, item.ProductID, item.Quantity, orderID); err != nil {
			return fmt.Errorf("rollback reservation for product %s: %w", item.ProductID, err)
		}
	}
	return nil
}

func (c *OrderCreatedConsumer) isProcessed(eventID string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	_, ok := c.processedEventIDs[eventID]
	return ok
}

func (c *OrderCreatedConsumer) markProcessed(eventID string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.processedEventIDs[eventID] = struct{}{}
}
