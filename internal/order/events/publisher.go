package events

import (
	"context"

	sharedevents "github.com/vladfc/event-driven-ecommerce-app/internal/shared/events"
)

type Publisher interface {
	PublishOrderCreated(ctx context.Context, event sharedevents.OrderCreated) error
}
