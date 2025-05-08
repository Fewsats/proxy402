-- Remove credit system columns from paid_routes
ALTER TABLE paid_routes
DROP COLUMN IF EXISTS type,
DROP COLUMN IF EXISTS credits;

-- Remove credit system columns and payment header from purchases
ALTER TABLE purchases
DROP COLUMN IF EXISTS type,
DROP COLUMN IF EXISTS credits_available,
DROP COLUMN IF EXISTS credits_used,
DROP COLUMN IF EXISTS payment_header;

DROP INDEX IF EXISTS idx_purchases_payment_header_route_id;