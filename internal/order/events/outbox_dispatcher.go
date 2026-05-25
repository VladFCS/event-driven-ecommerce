package events

import (
	"context"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	orderdb "github.com/vladfc/event-driven-ecommerce-app/internal/order/repository/sqlc"
)

type OutboxDispatcher struct {
	queries      *orderdb.Queries
	publisher    Publisher
	logger       *slog.Logger
	pollInterval time.Duration
	lockTimeout  time.Duration
	batchSize    int32
}

func NewOutboxDispatcher(pool *pgxpool.Pool, publisher Publisher, logger *slog.Logger) *OutboxDispatcher {
	if logger == nil {
		logger = slog.Default()
	}

	return &OutboxDispatcher{
		queries:      orderdb.New(pool),
		publisher:    publisher,
		logger:       logger,
		pollInterval: time.Second,
		lockTimeout:  30 * time.Second,
		batchSize:    10,
	}
}

func (d *OutboxDispatcher) Run(ctx context.Context) {
	if d == nil || d.publisher == nil {
		return
	}

	ticker := time.NewTicker(d.pollInterval)
	defer ticker.Stop()

	d.dispatchOnce(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			d.dispatchOnce(ctx)
		}
	}
}

func (d *OutboxDispatcher) dispatchOnce(ctx context.Context) {
	claimedAt := time.Now().UTC()
	rows, err := d.queries.ClaimPendingOrderOutboxEvents(ctx, orderdb.ClaimPendingOrderOutboxEventsParams{
		Limit: d.batchSize,
		LockedAt: pgtype.Timestamptz{
			Time:  claimedAt.Add(-d.lockTimeout),
			Valid: true,
		},
		LockedAt_2: pgtype.Timestamptz{
			Time:  claimedAt,
			Valid: true,
		},
	})
	if err != nil {
		d.logger.ErrorContext(ctx, "claim order outbox events failed", slog.Any("error", err))
		return
	}

	for _, row := range rows {
		publishCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		err := d.publisher.Publish(publishCtx, row.Topic, row.EventKey, row.Payload)
		cancel()
		if err != nil {
			releaseErr := d.queries.ReleaseOrderOutboxEvent(ctx, orderdb.ReleaseOrderOutboxEventParams{
				ID:        row.ID,
				LastError: err.Error(),
			})
			if releaseErr != nil {
				d.logger.ErrorContext(ctx, "release order outbox event failed", slog.String("outbox_id", row.ID), slog.Any("error", releaseErr))
			}
			d.logger.WarnContext(ctx, "publish order outbox event failed", slog.String("outbox_id", row.ID), slog.String("event_type", row.EventType), slog.Any("error", err))
			continue
		}

		if err := d.queries.MarkOrderOutboxEventPublished(ctx, orderdb.MarkOrderOutboxEventPublishedParams{
			ID: row.ID,
			PublishedAt: pgtype.Timestamptz{
				Time:  time.Now().UTC(),
				Valid: true,
			},
		}); err != nil {
			d.logger.ErrorContext(ctx, "mark order outbox event published failed", slog.String("outbox_id", row.ID), slog.Any("error", err))
			continue
		}

		d.logger.InfoContext(ctx, "order outbox event published", slog.String("outbox_id", row.ID), slog.String("event_type", row.EventType), slog.String("aggregate_id", row.AggregateID))
	}
}
