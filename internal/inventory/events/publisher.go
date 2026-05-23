package events

import (
	"context"

	sharedevents "github.com/vladfc/event-driven-ecommerce-app/internal/shared/events"
)

type Publisher interface {
	PublishInventoryReserved(ctx context.Context, sourceEventID string, event sharedevents.InventoryReserved) error
	PublishInventoryReservationFailed(ctx context.Context, sourceEventID string, event sharedevents.InventoryReservationFailed) error
	Close() error
}
