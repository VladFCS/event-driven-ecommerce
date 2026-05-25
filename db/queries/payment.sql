-- name: CreatePayment :one
INSERT INTO payments (
    id,
    order_id,
    customer_id,
    amount_currency,
    amount_cents,
    payment_method,
    payment_method_details,
    idempotency_key,
    status,
    cancel_reason,
    created_at,
    updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
)
RETURNING *;

-- name: GetPaymentByID :one
SELECT *
FROM payments
WHERE id = $1;

-- name: GetPaymentByOrderID :one
SELECT *
FROM payments
WHERE order_id = $1;

-- name: GetPaymentByIdempotencyKey :one
SELECT *
FROM payments
WHERE idempotency_key = $1;

-- name: ListPaymentsByCustomer :many
SELECT *
FROM payments
WHERE customer_id = $1
ORDER BY created_at ASC, id ASC
LIMIT $2 OFFSET $3;

-- name: CountPaymentsByCustomer :one
SELECT COUNT(*)
FROM payments
WHERE customer_id = $1;

-- name: UpdatePayment :one
UPDATE payments
SET
    order_id = $2,
    customer_id = $3,
    amount_currency = $4,
    amount_cents = $5,
    payment_method = $6,
    payment_method_details = $7,
    idempotency_key = $8,
    status = $9,
    cancel_reason = $10,
    created_at = $11,
    updated_at = $12
WHERE id = $1
RETURNING *;