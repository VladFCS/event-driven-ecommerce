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
	writer *kafka.Writer
	logger *slog.Logger
}

func NewKafkaPublisher(brokers []string, topic string, logger *slog.Logger) *KafkaPublisher {
	if logger == nil {
		logger = slog.Default()
	}
	return &KafkaPublisher{
		writer: &kafka.Writer{
			Addr:         kafka.TCP(brokers...),
			Topic:        topic,
			RequiredAcks: kafka.RequireAll,
			Async:        false,
		},
		logger: logger,
	}
}

func (p *KafkaPublisher) PublishOrderCreated(ctx context.Context, event sharedevents.OrderCreated) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal order created payload: %w", err)
	}

	envelope := sharedevents.Envelope{
		EventID:    fmt.Sprintf("evt-%d", time.Now().UTC().UnixNano()),
		EventType:  sharedevents.OrderCreatedEventType,
		OccurredAt: time.Now().UTC().Format(time.RFC3339),
		Payload:    payload,
	}

	messageBytes, err := json.Marshal(envelope)
	if err != nil {
		return fmt.Errorf("marshal order created event: %w", err)
	}

	msg := kafka.Message{
		Key:   []byte(event.OrderID),
		Value: messageBytes,
		Time:  time.Now().UTC(),
	}

	if err := p.writer.WriteMessages(ctx, msg); err != nil {
		return fmt.Errorf("write order created event to kafka: %w", err)
	}

	p.logger.InfoContext(
		ctx,
		"order.created event published",
		slog.String("event_type", sharedevents.OrderCreatedEventType),
		slog.String("order_id", event.OrderID),
	)

	return nil
}

func (p *KafkaPublisher) Close() error {
	if p == nil || p.writer == nil {
		return nil
	}

	return p.writer.Close()
}
