CREATE TABLE orders (
    id TEXT PRIMARY KEY,
    customer_id TEXT NOT NULL,
    total_amount_currency TEXT NOT NULL,
    total_amount_cents BIGINT NOT NULL CHECK (total_amount_cents > 0),
    status TEXT NOT NULL,
    shipping_country TEXT NOT NULL,
    shipping_city TEXT NOT NULL,
    shipping_street TEXT NOT NULL,
    shipping_postal_code TEXT NOT NULL,
    shipping_house TEXT NOT NULL,
    shipping_apartment TEXT NOT NULL DEFAULT '',
    payment_method TEXT NOT NULL,
    payment_method_details TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE order_items (
    order_id TEXT NOT NULL REFERENCES orders (id) ON DELETE CASCADE,
    item_position INT NOT NULL CHECK (item_position >= 0),
    product_id TEXT NOT NULL,
    sku TEXT NOT NULL,
    product_name TEXT NOT NULL,
    quantity INT NOT NULL CHECK (quantity > 0),
    unit_price_currency TEXT NOT NULL,
    unit_price_amount_cents BIGINT NOT NULL CHECK (unit_price_amount_cents > 0),
    total_price_currency TEXT NOT NULL,
    total_price_amount_cents BIGINT NOT NULL CHECK (total_price_amount_cents > 0),
    PRIMARY KEY (order_id, item_position)
);

CREATE INDEX orders_customer_id_created_at_idx
    ON orders (customer_id, created_at);

CREATE INDEX order_items_product_id_idx
    ON order_items (product_id);
