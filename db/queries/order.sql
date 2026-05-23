-- name: CreateOrder :one
INSERT INTO orders (
    id,
    customer_id,
    idempotency_key,
    total_amount_currency,
    total_amount_cents,
    status,
    shipping_country,
    shipping_city,
    shipping_street,
    shipping_postal_code,
    shipping_house,
    shipping_apartment,
    payment_method,
    payment_method_details,
    cancel_reason,
    created_at,
    updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17
)
RETURNING *;

-- name: CreateOrderItem :exec
INSERT INTO order_items (
    order_id,
    item_position,
    product_id,
    sku,
    product_name,
    quantity,
    unit_price_currency,
    unit_price_amount_cents,
    total_price_currency,
    total_price_amount_cents
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10
);

-- name: GetOrderByID :one
SELECT *
FROM orders
WHERE id = $1;

-- name: GetOrderByIdempotencyKey :one
SELECT *
FROM orders
WHERE idempotency_key = $1;

-- name: ListOrderItemsByOrderID :many
SELECT *
FROM order_items
WHERE order_id = $1
ORDER BY item_position ASC;

-- name: ListOrdersByCustomer :many
SELECT *
FROM orders
WHERE customer_id = $1
ORDER BY created_at ASC, id ASC
LIMIT $2 OFFSET $3;

-- name: CountOrdersByCustomer :one
SELECT COUNT(*)
FROM orders
WHERE customer_id = $1;

-- name: ListOrderItemsByOrderIDs :many
SELECT *
FROM order_items
WHERE order_id = ANY($1::text[])
ORDER BY order_id ASC, item_position ASC;

-- name: UpdateOrder :one
UPDATE orders
SET
    customer_id = $2,
    idempotency_key = $3,
    total_amount_currency = $4,
    total_amount_cents = $5,
    status = $6,
    shipping_country = $7,
    shipping_city = $8,
    shipping_street = $9,
    shipping_postal_code = $10,
    shipping_house = $11,
    shipping_apartment = $12,
    payment_method = $13,
    payment_method_details = $14,
    cancel_reason = $15,
    created_at = $16,
    updated_at = $17
WHERE id = $1
RETURNING *;

-- name: DeleteOrderItemsByOrderID :exec
DELETE FROM order_items
WHERE order_id = $1;

-- name: CreateOrderOutboxEvent :exec
INSERT INTO order_outbox_events (
    id,
    aggregate_type,
    aggregate_id,
    event_type,
    topic,
    event_key,
    payload,
    attempt_count,
    last_error,
    locked_at,
    created_at,
    published_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
);

-- name: ClaimPendingOrderOutboxEvents :many
WITH pending AS (
    SELECT id
    FROM order_outbox_events AS pending_events
    WHERE pending_events.published_at IS NULL
      AND (pending_events.locked_at IS NULL OR pending_events.locked_at < $2)
    ORDER BY pending_events.created_at ASC
    LIMIT $1
    FOR UPDATE SKIP LOCKED
)
UPDATE order_outbox_events AS o
SET locked_at = $3
FROM pending
WHERE o.id = pending.id
RETURNING o.*;

-- name: MarkOrderOutboxEventPublished :exec
UPDATE order_outbox_events
SET
    published_at = $2,
    locked_at = NULL,
    last_error = ''
WHERE id = $1;

-- name: ReleaseOrderOutboxEvent :exec
UPDATE order_outbox_events
SET
    locked_at = NULL,
    attempt_count = attempt_count + 1,
    last_error = $2
WHERE id = $1;

-- name: GetOrderOutboxStats :one
SELECT
    COUNT(*) FILTER (WHERE published_at IS NULL) AS pending_count,
    COUNT(*) FILTER (WHERE published_at IS NULL AND attempt_count > 0) AS retrying_count,
    MIN(created_at) FILTER (WHERE published_at IS NULL) AS oldest_pending_created_at
FROM order_outbox_events;
