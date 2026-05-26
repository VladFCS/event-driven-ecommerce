-- name: CreateInventoryStock :one
INSERT INTO inventory_stocks (
    product_id,
    available_quantity,
    reserved_quantity,
    total_quantity,
    created_at,
    updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6
)
RETURNING *;

-- name: GetStockByProductID :one
SELECT *
FROM inventory_stocks
WHERE product_id = $1;

-- name: GetStockByProductIDForUpdate :one
SELECT *
FROM inventory_stocks
WHERE product_id = $1
FOR UPDATE;

-- name: ReserveInventoryStock :one
UPDATE inventory_stocks
SET
    available_quantity = available_quantity - $2,
    reserved_quantity = reserved_quantity + $2,
    updated_at = $3
WHERE product_id = $1
  AND available_quantity >= $2
RETURNING *;

-- name: ReleaseInventoryStock :one
UPDATE inventory_stocks
SET
    available_quantity = available_quantity + $2,
    reserved_quantity = reserved_quantity - $2,
    updated_at = $3
WHERE product_id = $1
  AND reserved_quantity >= $2
RETURNING *;

-- name: GetReservationByProductIDAndOrderID :one
SELECT *
FROM inventory_reservations
WHERE product_id = $1
  AND order_id = $2;

-- name: GetReservationByProductIDAndOrderIDForUpdate :one
SELECT *
FROM inventory_reservations
WHERE product_id = $1
  AND order_id = $2
FOR UPDATE;

-- name: ListReservationsByOrderID :many
SELECT *
FROM inventory_reservations
WHERE order_id = $1
ORDER BY product_id ASC;

-- name: UpsertInventoryReservation :exec
INSERT INTO inventory_reservations (
    product_id,
    order_id,
    quantity,
    created_at,
    updated_at
) VALUES (
    $1, $2, $3, $4, $5
)
ON CONFLICT (product_id, order_id)
DO UPDATE
SET
    quantity = inventory_reservations.quantity + EXCLUDED.quantity,
    updated_at = EXCLUDED.updated_at;

-- name: UpdateInventoryReservationQuantity :exec
UPDATE inventory_reservations
SET
    quantity = $3,
    updated_at = $4
WHERE product_id = $1
  AND order_id = $2;

-- name: DeleteInventoryReservation :exec
DELETE FROM inventory_reservations
WHERE product_id = $1
  AND order_id = $2;
