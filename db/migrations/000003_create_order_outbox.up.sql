CREATE TABLE order_outbox_events (
    id TEXT PRIMARY KEY,
    aggregate_type TEXT NOT NULL,
    aggregate_id TEXT NOT NULL,
    event_type TEXT NOT NULL,
    topic TEXT NOT NULL,
    event_key TEXT NOT NULL,
    payload JSONB NOT NULL,
    attempt_count INT NOT NULL DEFAULT 0,
    last_error TEXT NOT NULL DEFAULT '',
    locked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL,
    published_at TIMESTAMPTZ
);

CREATE INDEX order_outbox_events_pending_idx
    ON order_outbox_events (published_at, locked_at, created_at);
