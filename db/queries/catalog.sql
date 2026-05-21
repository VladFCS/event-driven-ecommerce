-- name: CreateCatalogProduct :one
INSERT INTO catalog_products (
    id,
    name,
    description,
    price_cents,
    currency,
    created_at,
    updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
)
ON CONFLICT (id)
DO UPDATE
SET
    name = EXCLUDED.name,
    description = EXCLUDED.description,
    price_cents = EXCLUDED.price_cents,
    currency = EXCLUDED.currency,
    updated_at = EXCLUDED.updated_at
RETURNING *;

-- name: GetCatalogProductByID :one
SELECT *
FROM catalog_products
WHERE id = $1;

-- name: CountCatalogProducts :one
SELECT COUNT(*)
FROM catalog_products;

-- name: ListCatalogProducts :many
SELECT *
FROM catalog_products
ORDER BY id
LIMIT $1 OFFSET $2;

-- name: UpdateCatalogProduct :one
UPDATE catalog_products
SET
    name = $2,
    description = $3,
    price_cents = $4,
    currency = $5,
    updated_at = $6
WHERE id = $1
RETURNING *;

-- name: DeleteCatalogProduct :one
DELETE FROM catalog_products
WHERE id = $1
RETURNING id;
