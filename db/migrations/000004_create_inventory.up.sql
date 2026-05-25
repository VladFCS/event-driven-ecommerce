CREATE TABLE inventory_stocks (
    product_id TEXT PRIMARY KEY,
    available_quantity BIGINT NOT NULL CHECK (available_quantity >= 0),
    reserved_quantity BIGINT NOT NULL CHECK (reserved_quantity >= 0),
    total_quantity BIGINT NOT NULL CHECK (total_quantity >= 0),
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    CHECK (available_quantity + reserved_quantity = total_quantity)
);

CREATE TABLE inventory_reservations (
    product_id TEXT NOT NULL REFERENCES inventory_stocks (product_id) ON DELETE CASCADE,
    order_id TEXT NOT NULL,
    quantity BIGINT NOT NULL CHECK (quantity > 0),
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (product_id, order_id)
);

CREATE INDEX inventory_reservations_order_id_idx
    ON inventory_reservations (order_id);
