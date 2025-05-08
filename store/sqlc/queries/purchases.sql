-- name: CreatePurchase :one
-- CreatePurchase creates a new purchase record.
INSERT INTO purchases (
    short_code, target_url, method, price, is_test,
    payment_payload, settle_response, paid_route_id, paid_to_address,
    created_at, updated_at,
    type, credits_available, credits_used, payment_header
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11,
    $12, $13, $14, $15
) RETURNING id;

-- name: GetPurchaseByID :one
-- GetPurchaseByID returns a purchase by ID.
SELECT * FROM purchases
WHERE id = $1;

-- name: ListPurchasesByUserID :many
-- ListPurchasesByUserID retrieves all purchases for a specific user via paid_routes.
SELECT p.* FROM purchases p
JOIN paid_routes pr ON p.paid_route_id = pr.id
WHERE pr.user_id = $1
ORDER BY p.created_at DESC;

-- name: GetDailyStats :many
-- GetDailyStats retrieves daily purchase stats for a specific user.
SELECT 
    to_char(p.created_at AT TIME ZONE 'UTC', 'YYYY-MM-DD') AS date,
    COUNT(*) AS count,
    SUM(p.price) AS earnings,
    COUNT(CASE WHEN p.is_test = true THEN 1 END) AS test_count,
    COALESCE(SUM(CASE WHEN p.is_test = true THEN p.price ELSE 0 END), 0) AS test_earnings,
    COUNT(CASE WHEN p.is_test = false THEN 1 END) AS real_count,
    COALESCE(SUM(CASE WHEN p.is_test = false THEN p.price ELSE 0 END), 0) AS real_earnings
FROM 
    purchases p
JOIN 
    paid_routes pr ON p.paid_route_id = pr.id
WHERE 
    pr.user_id = $1
    AND p.created_at >= CURRENT_TIMESTAMP - ($2 || ' days')::INTERVAL
GROUP BY 
    date
ORDER BY 
    date DESC;

-- name: GetTotalStats :one
-- GetTotalStats retrieves total purchase stats for a specific user.
SELECT 
    COALESCE(SUM(p.price), 0) AS total_earnings,
    COUNT(*) AS total_count
FROM 
    purchases p
JOIN 
    paid_routes pr ON p.paid_route_id = pr.id
WHERE 
    pr.user_id = $1;

-- name: GetPurchaseByRouteIDAndPaymentHeader :one
SELECT * FROM purchases
WHERE paid_route_id = $1 AND payment_header = $2
ORDER BY created_at DESC LIMIT 1;

-- name: IncrementPurchaseCreditsUsed :exec
UPDATE purchases
SET credits_used = credits_used + 1, updated_at = $2
WHERE id = $1 AND credits_used < credits_available; 