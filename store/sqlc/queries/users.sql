-- name: GetUserByID :one
-- GetUserByID returns a user by ID.
SELECT * FROM users
WHERE id = $1;

-- name: GetUserByEmail :one
-- GetUserByEmail returns a user by email.
SELECT * FROM users
WHERE email = $1;

-- name: GetUserByGoogleID :one
-- GetUserByGoogleID returns a user by Google ID.
SELECT * FROM users
WHERE google_id = $1;

-- name: CreateUser :one
-- CreateUser creates a new user record.
INSERT INTO users (
    email, name, google_id, proxy_402_secret, payment_address, created_at, updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
) RETURNING id;

-- name: UpdateUserProxySecret :one
UPDATE users SET
    proxy_402_secret = $2,
    updated_at = $3
WHERE id = $1
RETURNING *;

-- name: UpdateUserPaymentAddress :one
-- UpdateUserPaymentAddress updates a user's payment address.
UPDATE users SET
    payment_address = $2,
    updated_at = $3
WHERE id = $1
RETURNING *;