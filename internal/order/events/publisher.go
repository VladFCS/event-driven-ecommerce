package events

import "context"

type Publisher interface {
	PublishOrderCreated(ctx context.Context, event OrderCreated) error
}

