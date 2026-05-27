package consumers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/segmentio/kafka-go"
	"github.com/vladfc/event-driven-ecommerce-app/internal/order/service"
	sharedevents "github.com/vladfc/event-driven-ecommerce-app/internal/shared/events"
)

type TopicConsumer struct {
	reader            *kafka.Reader
	service           *service.OrderService
	deduper           EventDeduplicator
	logger            *slog.Logger
	expectedEventType string
	handleEvent       func(ctx context.Context, service *service.OrderService, envelope sharedevents.Envelope) error
}

func newTopicConsumer(
	brokers []string,
	topic string,
	groupID string,
	service *service.OrderService,
	deduper EventDeduplicator,
	logger *slog.Logger,
	expectedEventType string,
	handleEvent func(ctx context.Context, service *service.OrderService, envelope sharedevents.Envelope) error,
) *TopicConsumer {
	if logger == nil {
		logger = slog.Default()
	}

	return &TopicConsumer{
		reader: kafka.NewReader(kafka.ReaderConfig{
			Brokers: brokers,
			Topic:   topic,
			GroupID: groupID,
		}),
		service:           service,
		deduper:           deduper,
		logger:            logger,
		expectedEventType: expectedEventType,
		handleEvent:       handleEvent,
	}
}

func (c *TopicConsumer) Run(ctx context.Context) error {
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

func (c *TopicConsumer) Close() error {
	if c == nil || c.reader == nil {
		return nil
	}

	return c.reader.Close()
}

func (c *TopicConsumer) handleMessage(ctx context.Context, msg kafka.Message) (bool, error) {
	var envelope sharedevents.Envelope
	if err := json.Unmarshal(msg.Value, &envelope); err != nil {
		c.logger.ErrorContext(ctx, "failed to decode event envelope", slog.Any("error", err))
		return true, nil
	}

	if envelope.EventType != c.expectedEventType {
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

	if err := c.handleEvent(ctx, c.service, envelope); err != nil {
		return false, err
	}

	if err := c.deduper.MarkProcessed(ctx, envelope.EventID); err != nil {
		return false, err
	}

	return true, nil
}

func NewInventoryReservedConsumer(
	brokers []string,
	topic string,
	groupID string,
	orderService *service.OrderService,
	deduper EventDeduplicator,
	logger *slog.Logger,
) *TopicConsumer {
	if logger == nil {
		logger = slog.Default()
	}

	return newTopicConsumer(
		brokers,
		topic,
		groupID,
		orderService,
		deduper,
		logger,
		sharedevents.InventoryReservedEventType,
		func(ctx context.Context, orderService *service.OrderService, envelope sharedevents.Envelope) error {
			var event sharedevents.InventoryReserved
			if err := json.Unmarshal(envelope.Payload, &event); err != nil {
				return fmt.Errorf("decode inventory.reserved payload: %w", err)
			}

			order, err := orderService.MarkInventoryReserved(ctx, event.OrderID)
			if err != nil {
				return err
			}

			logger.InfoContext(ctx, "order updated from inventory.reserved", slog.String("order_id", order.ID), slog.String("status", order.Status.String()))
			return nil
		},
	)
}

func NewInventoryReservationFailedConsumer(
	brokers []string,
	topic string,
	groupID string,
	orderService *service.OrderService,
	deduper EventDeduplicator,
	logger *slog.Logger,
) *TopicConsumer {
	if logger == nil {
		logger = slog.Default()
	}

	return newTopicConsumer(
		brokers,
		topic,
		groupID,
		orderService,
		deduper,
		logger,
		sharedevents.InventoryReservationFailedEventType,
		func(ctx context.Context, orderService *service.OrderService, envelope sharedevents.Envelope) error {
			var event sharedevents.InventoryReservationFailed
			if err := json.Unmarshal(envelope.Payload, &event); err != nil {
				return fmt.Errorf("decode inventory.reservation_failed payload: %w", err)
			}

			order, err := orderService.MarkInventoryReservationFailed(ctx, event.OrderID, event.Reason)
			if err != nil {
				return err
			}

			logger.InfoContext(
				ctx,
				"order cancelled from inventory.reservation_failed",
				slog.String("order_id", order.ID),
				slog.String("status", order.Status.String()),
				slog.String("cancel_reason", order.CancelReason),
			)
			return nil
		},
	)
}

func NewPaymentCapturedConsumer(
	brokers []string,
	topic string,
	groupID string,
	orderService *service.OrderService,
	deduper EventDeduplicator,
	logger *slog.Logger,
) *TopicConsumer {
	if logger == nil {
		logger = slog.Default()
	}

	return newTopicConsumer(
		brokers,
		topic,
		groupID,
		orderService,
		deduper,
		logger,
		sharedevents.PaymentCapturedEventType,
		func(ctx context.Context, orderService *service.OrderService, envelope sharedevents.Envelope) error {
			var event sharedevents.PaymentCaptured
			if err := json.Unmarshal(envelope.Payload, &event); err != nil {
				return fmt.Errorf("decode payment.captured payload: %w", err)
			}

			order, err := orderService.MarkPaymentCaptured(ctx, event.OrderID)
			if err != nil {
				return err
			}

			logger.InfoContext(
				ctx,
				"order confirmed from payment.captured",
				slog.String("order_id", order.ID),
				slog.String("status", order.Status.String()),
				slog.String("payment_id", event.PaymentID),
			)
			return nil
		},
	)
}

func NewPaymentFailedConsumer(
	brokers []string,
	topic string,
	groupID string,
	orderService *service.OrderService,
	deduper EventDeduplicator,
	logger *slog.Logger,
) *TopicConsumer {
	if logger == nil {
		logger = slog.Default()
	}

	return newTopicConsumer(
		brokers,
		topic,
		groupID,
		orderService,
		deduper,
		logger,
		sharedevents.PaymentFailedEventType,
		func(ctx context.Context, orderService *service.OrderService, envelope sharedevents.Envelope) error {
			var event sharedevents.PaymentFailed
			if err := json.Unmarshal(envelope.Payload, &event); err != nil {
				return fmt.Errorf("decode payment.failed payload: %w", err)
			}

			order, err := orderService.MarkPaymentFailed(ctx, event.OrderID, event.FailureReason)
			if err != nil {
				return err
			}

			logger.InfoContext(
				ctx,
				"order cancelled from payment.failed",
				slog.String("order_id", order.ID),
				slog.String("status", order.Status.String()),
				slog.String("cancel_reason", order.CancelReason),
				slog.String("payment_id", event.PaymentID),
			)
			return nil
		},
	)
}

func NewPaymentCreationFailedConsumer(
	brokers []string,
	topic string,
	groupID string,
	orderService *service.OrderService,
	deduper EventDeduplicator,
	logger *slog.Logger,
) *TopicConsumer {
	if logger == nil {
		logger = slog.Default()
	}

	return newTopicConsumer(
		brokers,
		topic,
		groupID,
		orderService,
		deduper,
		logger,
		sharedevents.PaymentCreationFailedEventType,
		func(ctx context.Context, orderService *service.OrderService, envelope sharedevents.Envelope) error {
			var event sharedevents.PaymentCreationFailed
			if err := json.Unmarshal(envelope.Payload, &event); err != nil {
				return fmt.Errorf("decode payment.creation_failed payload: %w", err)
			}

			order, err := orderService.MarkPaymentCreationFailed(ctx, event.OrderID, event.FailureReason)
			if err != nil {
				return err
			}

			logger.InfoContext(
				ctx,
				"order cancelled from payment.creation_failed",
				slog.String("order_id", order.ID),
				slog.String("status", order.Status.String()),
				slog.String("cancel_reason", order.CancelReason),
			)
			return nil
		},
	)
}
