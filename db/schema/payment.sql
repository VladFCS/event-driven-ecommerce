CREATE TABLE payments (
    id TEXT PRIMARY KEY,
    order_id TEXT NOT NULL UNIQUE,
    customer_id TEXT NOT NULL,
    amount_currency TEXT NOT NULL,
    amount_cents BIGINT NOT NULL CHECK (amount_cents > 0),
    payment_method TEXT NOT NULL,
    payment_method_details TEXT NOT NULL DEFAULT '',
    idempotency_key TEXT UNIQUE,
    status TEXT NOT NULL,
    cancel_reason TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX payments_customer_id_created_at_idx
    ON payments (customer_id, created_at);
