-- name: GetPaidRouteByID :one
-- GetPaidRouteByID returns a paid route by ID.
SELECT * FROM paid_routes
WHERE id = $1 AND deleted_at IS NULL;

-- name: GetPaidRouteByShortCode :one
-- GetPaidRouteByShortCode returns a paid route by its short code.
SELECT * FROM paid_routes
WHERE short_code = $1 AND deleted_at IS NULL;

-- name: GetEnabledPaidRouteByShortCode :one
-- GetEnabledPaidRouteByShortCode returns an enabled paid route by its short code.
SELECT * FROM paid_routes
WHERE short_code = $1 AND is_enabled = true AND deleted_at IS NULL;

-- name: CheckShortCodeExists :one
-- CheckShortCodeExists checks if a short code already exists.
SELECT EXISTS(
  SELECT 1 FROM paid_routes
  WHERE short_code = $1
) as exists;

-- name: ListUserPaidRoutes :many
-- ListUserPaidRoutes returns all paid routes for a specific user.
SELECT * FROM paid_routes
WHERE user_id = $1 AND deleted_at IS NULL
ORDER BY created_at DESC;

-- name: CreatePaidRoute :one
-- CreatePaidRoute creates a new paid route.
INSERT INTO paid_routes (
    short_code, target_url, method, price, is_test,
    user_id, is_enabled, attempt_count, payment_count, access_count,
    created_at, updated_at,
    type, credits
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12,
    $13, $14
) RETURNING *;

-- name: UpdatePaidRoute :exec
-- UpdatePaidRoute updates a paid route.
UPDATE paid_routes SET
    target_url = $2,
    method = $3,
    price = $4,
    is_test = $5,
    is_enabled = $6,
    updated_at = $7
WHERE id = $1 AND deleted_at IS NULL;

-- name: IncrementAttemptCount :exec
-- IncrementAttemptCount increments the attempt_count for a route.
UPDATE paid_routes SET
    attempt_count = attempt_count + 1,
    updated_at = $2
WHERE short_code = $1 AND deleted_at IS NULL;

-- name: IncrementPaymentCount :exec
-- IncrementPaymentCount increments the payment_count for a route.
UPDATE paid_routes SET
    payment_count = payment_count + 1,
    updated_at = $2
WHERE short_code = $1 AND deleted_at IS NULL;

-- name: IncrementAccessCount :exec
-- IncrementAccessCount increments the access_count for a route.
UPDATE paid_routes SET
    access_count = access_count + 1,
    updated_at = $2
WHERE short_code = $1 AND deleted_at IS NULL;

-- name: DeletePaidRoute :exec
-- DeletePaidRoute soft-deletes a paid route.
UPDATE paid_routes SET
    deleted_at = $3
WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL; 