-- name: CreateOrder :one
INSERT INTO orders (
    id,
    customer_id,
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
    created_at,
    updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15
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
    total_amount_currency = $3,
    total_amount_cents = $4,
    status = $5,
    shipping_country = $6,
    shipping_city = $7,
    shipping_street = $8,
    shipping_postal_code = $9,
    shipping_house = $10,
    shipping_apartment = $11,
    payment_method = $12,
    payment_method_details = $13,
    created_at = $14,
    updated_at = $15
WHERE id = $1
RETURNING *;

-- name: DeleteOrderItemsByOrderID :exec
DELETE FROM order_items
WHERE order_id = $1;
