package events

import (
	"context"
	"fmt"
	"log/slog"
	"runtime/debug"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	orderdb "github.com/vladfc/event-driven-ecommerce-app/internal/order/repository/sqlc"
)

type OutboxDispatcher struct {
	queries          *orderdb.Queries
	publisher        Publisher
	logger           *slog.Logger
	pollInterval     time.Duration
	lockTimeout      time.Duration
	publishTimeout   time.Duration
	batchSize        int32
	statsLogInterval time.Duration
	lastStatsLogAt   time.Time
}

type OutboxDispatcherConfig struct {
	PollInterval   time.Duration
	LockTimeout    time.Duration
	PublishTimeout time.Duration
	BatchSize      int32
}

func DefaultOutboxDispatcherConfig() OutboxDispatcherConfig {
	return OutboxDispatcherConfig{
		PollInterval:   time.Second,
		LockTimeout:    30 * time.Second,
		PublishTimeout: 5 * time.Second,
		BatchSize:      10,
	}
}

func (c OutboxDispatcherConfig) Validate() error {
	if c.PollInterval <= 0 {
		return fmt.Errorf("outbox poll interval must be greater than zero")
	}
	if c.LockTimeout <= 0 {
		return fmt.Errorf("outbox lock timeout must be greater than zero")
	}
	if c.PublishTimeout <= 0 {
		return fmt.Errorf("outbox publish timeout must be greater than zero")
	}
	if c.BatchSize <= 0 {
		return fmt.Errorf("outbox batch size must be greater than zero")
	}
	if c.LockTimeout < c.PublishTimeout {
		return fmt.Errorf("outbox lock timeout must be greater than or equal to publish timeout")
	}
	return nil
}

func NewOutboxDispatcher(pool *pgxpool.Pool, publisher Publisher, logger *slog.Logger, cfg OutboxDispatcherConfig) *OutboxDispatcher {
	if logger == nil {
		logger = slog.Default()
	}

	if err := cfg.Validate(); err != nil {
		logger.Error("invalid outbox dispatcher config", slog.Any("error", err))
	}

	return &OutboxDispatcher{
		queries:          orderdb.New(pool),
		publisher:        publisher,
		logger:           logger,
		pollInterval:     cfg.PollInterval,
		lockTimeout:      cfg.LockTimeout,
		publishTimeout:   cfg.PublishTimeout,
		batchSize:        cfg.BatchSize,
		statsLogInterval: 30 * time.Second,
	}
}

func (d *OutboxDispatcher) Run(ctx context.Context) (err error) {
	if d == nil || d.publisher == nil {
		return fmt.Errorf("outbox dispatcher is not configured")
	}
	if err := (OutboxDispatcherConfig{
		PollInterval:   d.pollInterval,
		LockTimeout:    d.lockTimeout,
		PublishTimeout: d.publishTimeout,
		BatchSize:      d.batchSize,
	}).Validate(); err != nil {
		return err
	}

	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("outbox dispatcher panic: %v\n%s", recovered, debug.Stack())
		}
	}()

	ticker := time.NewTicker(d.pollInterval)
	defer ticker.Stop()

	d.dispatchOnce(ctx)

	for {
		select {
		case <-ctx.Done():
			return nil
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

	stats, err := d.queries.GetOrderOutboxStats(ctx)
	if err != nil {
		d.logger.ErrorContext(ctx, "load order outbox stats failed", slog.Any("error", err))
	} else {
		d.logBacklogStats(ctx, len(rows), stats, claimedAt)
	}

	for _, row := range rows {
		publishCtx, cancel := context.WithTimeout(ctx, d.publishTimeout)
		err := d.publisher.Publish(publishCtx, row.Topic, row.EventKey, row.Payload)
		cancel()
		if err != nil {
			attempt := row.AttemptCount + 1
			releaseErr := d.queries.ReleaseOrderOutboxEvent(ctx, orderdb.ReleaseOrderOutboxEventParams{
				ID:        row.ID,
				LastError: err.Error(),
			})
			if releaseErr != nil {
				d.logger.ErrorContext(ctx, "release order outbox event failed", slog.String("outbox_id", row.ID), slog.Any("error", releaseErr))
			}
			d.logger.WarnContext(
				ctx,
				"publish order outbox event failed",
				slog.String("outbox_id", row.ID),
				slog.String("event_type", row.EventType),
				slog.Int64("attempt", int64(attempt)),
				slog.Any("error", err),
			)
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

		d.logger.InfoContext(
			ctx,
			"order outbox event published",
			slog.String("outbox_id", row.ID),
			slog.String("event_type", row.EventType),
			slog.String("aggregate_id", row.AggregateID),
			slog.Int64("attempt", int64(row.AttemptCount+1)),
		)
	}
}

func (d *OutboxDispatcher) logBacklogStats(ctx context.Context, claimedCount int, stats orderdb.GetOrderOutboxStatsRow, now time.Time) {
	if claimedCount > 0 {
		attrs := []any{
			slog.Int("claimed_count", claimedCount),
			slog.Int64("pending_count", stats.PendingCount),
			slog.Int64("retrying_count", stats.RetryingCount),
		}
		if oldestPendingAt, ok := outboxOldestPendingCreatedAt(stats.OldestPendingCreatedAt); ok {
			attrs = append(attrs, slog.String("oldest_pending_age", now.Sub(oldestPendingAt).String()))
		}

		d.logger.InfoContext(ctx, "claimed order outbox events", attrs...)
		d.lastStatsLogAt = now
		return
	}

	if stats.PendingCount == 0 {
		return
	}

	if !d.lastStatsLogAt.IsZero() && now.Sub(d.lastStatsLogAt) < d.statsLogInterval {
		return
	}

	attrs := []any{
		slog.Int64("pending_count", stats.PendingCount),
		slog.Int64("retrying_count", stats.RetryingCount),
	}
	if oldestPendingAt, ok := outboxOldestPendingCreatedAt(stats.OldestPendingCreatedAt); ok {
		attrs = append(attrs, slog.String("oldest_pending_age", now.Sub(oldestPendingAt).String()))
	}

	d.logger.WarnContext(ctx, "order outbox backlog pending", attrs...)
	d.lastStatsLogAt = now
}

func outboxOldestPendingCreatedAt(value interface{}) (time.Time, bool) {
	switch typed := value.(type) {
	case time.Time:
		return typed, true
	case *time.Time:
		if typed == nil {
			return time.Time{}, false
		}
		return *typed, true
	default:
		return time.Time{}, false
	}
}
