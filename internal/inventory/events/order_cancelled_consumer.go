package events

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/segmentio/kafka-go"
	"github.com/vladfc/event-driven-ecommerce-app/internal/inventory/service"
	sharedevents "github.com/vladfc/event-driven-ecommerce-app/internal/shared/events"
)

type OrderCancelledConsumer struct {
	reader  *kafka.Reader
	service *service.InventoryService
	deduper EventDeduplicator
	logger  *slog.Logger
}

func NewOrderCancelledConsumer(
	brokers []string,
	topic string,
	groupID string,
	service *service.InventoryService,
	deduper EventDeduplicator,
	logger *slog.Logger,
) *OrderCancelledConsumer {
	if logger == nil {
		logger = slog.Default()
	}

	return &OrderCancelledConsumer{
		reader: kafka.NewReader(kafka.ReaderConfig{
			Brokers: brokers,
			Topic:   topic,
			GroupID: groupID,
		}),
		service: service,
		deduper: deduper,
		logger:  logger,
	}
}

func (c *OrderCancelledConsumer) Close() error {
	if c == nil || c.reader == nil {
		return nil
	}
	return c.reader.Close()
}

func (c *OrderCancelledConsumer) Run(ctx context.Context) error {
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

func (c *OrderCancelledConsumer) handleMessage(ctx context.Context, msg kafka.Message) (bool, error) {
	var envelope sharedevents.Envelope
	if err := json.Unmarshal(msg.Value, &envelope); err != nil {
		c.logger.ErrorContext(ctx, "failed to decode event envelope", slog.Any("error", err))
		return true, nil
	}

	if envelope.EventType != sharedevents.OrderCancelledEventType {
		c.logger.WarnContext(ctx, "skipping unsupported event type", slog.String("event_type", envelope.EventType))
		return true, nil
	}

	processed, err := c.deduper.IsProcessed(ctx, envelope.EventID)
	if err != nil {
		return false, err
	}

	if processed {
		c.logger.InfoContext(ctx, "skipping duplicate event", slog.String("event_id", envelope.EventID))
		return true, nil
	}

	var event sharedevents.OrderCancelled
	if err := json.Unmarshal(envelope.Payload, &event); err != nil {
		c.logger.ErrorContext(ctx, "failed to decode order.cancelled payload", slog.Any("error", err))
		return true, nil
	}

	if err := c.service.ReleaseReservationsByOrderID(ctx, event.OrderID); err != nil {
		return false, err
	}

	if err := c.deduper.MarkProcessed(ctx, envelope.EventID); err != nil {
		return false, err
	}

	c.logger.InfoContext(
		ctx,
		"released inventory reservations for cancelled order",
		slog.String("order_id", event.OrderID),
		slog.String("customer_id", event.CustomerID),
		slog.String("cancel_reason", event.CancelReason),
	)

	return true, nil
}
