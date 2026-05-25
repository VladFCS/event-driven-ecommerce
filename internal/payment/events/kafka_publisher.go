package events

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/segmentio/kafka-go"
	sharedevents "github.com/vladfc/event-driven-ecommerce-app/internal/shared/events"
)

type KafkaPublisher struct {
	createdWriter *kafka.Writer
	failedWriter  *kafka.Writer
	logger        *slog.Logger
}

func NewKafkaPublisher(brokers []string, createdTopic, failedTopic string, logger *slog.Logger) *KafkaPublisher {
	if logger == nil {
		logger = slog.Default()
	}

	return &KafkaPublisher{
		createdWriter: newWriter(brokers, createdTopic),
		failedWriter:  newWriter(brokers, failedTopic),
		logger:        logger,
	}
}

func (p *KafkaPublisher) PublishPaymentCreated(ctx context.Context, sourceEventID string, event sharedevents.PaymentCreated) error {
	return p.writeEvent(ctx, p.createdWriter, sharedevents.PaymentCreatedEventType, sourceEventID, event.OrderID, event)
}

func (p *KafkaPublisher) PublishPaymentCreationFailed(ctx context.Context, sourceEventID string, event sharedevents.PaymentCreationFailed) error {
	return p.writeEvent(ctx, p.failedWriter, sharedevents.PaymentCreationFailedEventType, sourceEventID, event.OrderID, event)
}

func (p *KafkaPublisher) writeEvent(ctx context.Context, writer *kafka.Writer, eventType, sourceEventID, key string, payload any) error {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal %s payload: %w", eventType, err)
	}

	envelope := sharedevents.Envelope{
		EventID:    fmt.Sprintf("%s:%d", sourceEventID, time.Now().UTC().UnixNano()),
		EventType:  eventType,
		OccurredAt: time.Now().UTC().Format(time.RFC3339),
		Payload:    payloadBytes,
	}

	messageBytes, err := json.Marshal(envelope)
	if err != nil {
		return fmt.Errorf("marshal %s event envelope: %w", eventType, err)
	}

	msg := kafka.Message{
		Key:   []byte(key),
		Value: messageBytes,
		Time:  time.Now().UTC(),
	}

	if err := writer.WriteMessages(ctx, msg); err != nil {
		return fmt.Errorf("write %s event to kafka: %w", eventType, err)
	}

	p.logger.InfoContext(
		ctx,
		"payment event published",
		slog.String("event_type", eventType),
		slog.String("order_id", key),
	)

	return nil
}

func (p *KafkaPublisher) Close() error {
	if p == nil {
		return nil
	}

	if p.createdWriter != nil {
		if err := p.createdWriter.Close(); err != nil {
			return err
		}
	}

	if p.failedWriter != nil {
		if err := p.failedWriter.Close(); err != nil {
			return err
		}
	}

	return nil
}

func newWriter(brokers []string, topic string) *kafka.Writer {
	return &kafka.Writer{
		Addr:         kafka.TCP(brokers...),
		Topic:        topic,
		RequiredAcks: kafka.RequireAll,
		Async:        false,
	}
}
