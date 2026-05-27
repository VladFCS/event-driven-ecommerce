package events

import (
	"context"

	sharedevents "github.com/vladfc/event-driven-ecommerce-app/internal/shared/events"
)

type Publisher interface {
	PublishPaymentCreated(ctx context.Context, sourceEventID string, event sharedevents.PaymentCreated) error
	PublishPaymentCreationFailed(ctx context.Context, sourceEventID string, event sharedevents.PaymentCreationFailed) error
	PublishPaymentCaptured(ctx context.Context, sourceEventID string, event sharedevents.PaymentCaptured) error
	PublishPaymentFailed(ctx context.Context, sourceEventID string, event sharedevents.PaymentFailed) error
	Close() error
}