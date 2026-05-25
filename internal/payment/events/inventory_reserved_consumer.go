package events

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/segmentio/kafka-go"
	paymentv1 "github.com/vladfc/event-driven-ecommerce-app/gen/payment/v1"
	"github.com/vladfc/event-driven-ecommerce-app/internal/payment/domain"
	"github.com/vladfc/event-driven-ecommerce-app/internal/payment/service"
	sharedevents "github.com/vladfc/event-driven-ecommerce-app/internal/shared/events"
)

const paymentIdempotencyKeyPrefix = "payment:create:"

type InventoryReservedConsumer struct {
	reader    *kafka.Reader
	service   *service.PaymentService
	deduper   EventDeduplicator
	publisher Publisher
	logger    *slog.Logger
}

func NewInventoryReservedConsumer(
	brokers []string,
	topic string,
	groupID string,
	service *service.PaymentService,
	deduper EventDeduplicator,
	publisher Publisher,
	logger *slog.Logger,
) *InventoryReservedConsumer {
	if logger == nil {
		logger = slog.Default()
	}

	return &InventoryReservedConsumer{
		reader: kafka.NewReader(kafka.ReaderConfig{
			Brokers: brokers,
			Topic:   topic,
			GroupID: groupID,
		}),
		service:   service,
		deduper:   deduper,
		publisher: publisher,
		logger:    logger,
	}
}

func (c *InventoryReservedConsumer) Close() error {
	if c == nil || c.reader == nil {
		return nil
	}
	return c.reader.Close()
}

func (c *InventoryReservedConsumer) Run(ctx context.Context) error {
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

func (c *InventoryReservedConsumer) handleMessage(ctx context.Context, msg kafka.Message) (bool, error) {
	var envelope sharedevents.Envelope
	if err := json.Unmarshal(msg.Value, &envelope); err != nil {
		c.logger.ErrorContext(ctx, "failed to decode event envelope", slog.Any("error", err))
		return true, nil
	}

	if envelope.EventType != sharedevents.InventoryReservedEventType {
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

	var event sharedevents.InventoryReserved
	if err := json.Unmarshal(envelope.Payload, &event); err != nil {
		c.logger.ErrorContext(ctx, "failed to decode inventory.reserved payload", slog.Any("error", err))
		return true, nil
	}

	payment, creationErr := c.createPayment(ctx, event)
	if creationErr != nil {
		if err := c.publisher.PublishPaymentCreationFailed(ctx, envelope.EventID, sharedevents.PaymentCreationFailed{
			OrderID:       event.OrderID,
			CustomerID:    event.CustomerID,
			FailureReason: creationErr.Error(),
		}); err != nil {
			return false, err
		}

		if err := c.deduper.MarkProcessed(ctx, envelope.EventID); err != nil {
			return false, err
		}
		return true, nil
	}

	if err := c.publisher.PublishPaymentCreated(ctx, envelope.EventID, sharedevents.PaymentCreated{
		OrderID:    payment.OrderID,
		CustomerID: payment.CustomerID,
		PaymentID:  payment.ID,
		Status:     payment.Status.String(),
	}); err != nil {
		return false, err
	}

	if err := c.deduper.MarkProcessed(ctx, envelope.EventID); err != nil {
		return false, err
	}

	return true, nil
}

func (c *InventoryReservedConsumer) createPayment(ctx context.Context, event sharedevents.InventoryReserved) (domain.Payment, error) {
	currency, err := parseCurrency(event.Amount.Currency)
	if err != nil {
		return domain.Payment{}, err
	}

	method, err := parsePaymentMethod(event.Payment.Method)
	if err != nil {
		return domain.Payment{}, err
	}

	return c.service.CreatePayment(ctx, domain.Payment{
		OrderID:    event.OrderID,
		CustomerID: event.CustomerID,
		Amount: domain.Money{
			Currency:    currency,
			AmountCents: event.Amount.AmountCents,
		},
		PaymentMethod:        method,
		PaymentMethodDetails: event.Payment.MethodDetails,
		IdempotencyKey:       paymentIdempotencyKeyPrefix + event.OrderID,
	})
}

func parseCurrency(value string) (paymentv1.Currency, error) {
	switch value {
	case "CURRENCY_USD", "USD":
		return paymentv1.Currency_CURRENCY_USD, nil
	case "CURRENCY_EUR", "EUR":
		return paymentv1.Currency_CURRENCY_EUR, nil
	default:
		return paymentv1.Currency_CURRENCY_UNSPECIFIED, fmt.Errorf("unsupported payment currency: %q", value)
	}
}

func parsePaymentMethod(value string) (paymentv1.PaymentMethodType, error) {
	switch value {
	case "PAYMENT_METHOD_TYPE_CARD", "CARD":
		return paymentv1.PaymentMethodType_PAYMENT_METHOD_TYPE_CARD, nil
	case "PAYMENT_METHOD_TYPE_CASH", "CASH":
		return paymentv1.PaymentMethodType_PAYMENT_METHOD_TYPE_CASH, nil
	default:
		return paymentv1.PaymentMethodType_PAYMENT_METHOD_TYPE_UNSPECIFIED, fmt.Errorf("unsupported payment method: %q", value)
	}
}
