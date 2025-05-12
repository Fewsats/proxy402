-- Add credit system columns to paid_routes
ALTER TABLE paid_routes
ADD COLUMN type VARCHAR(50) NOT NULL DEFAULT 'credit',
ADD COLUMN credits INTEGER NOT NULL DEFAULT 1;

-- Add credit system columns and payment header to purchases
ALTER TABLE purchases
ADD COLUMN type VARCHAR(50) NOT NULL DEFAULT 'credit', -- Assuming default matches routes
ADD COLUMN credits_available INTEGER NOT NULL DEFAULT 0, -- Default 0, will be set on creation
ADD COLUMN credits_used INTEGER NOT NULL DEFAULT 0,
ADD COLUMN payment_header TEXT;

-- Create the new composite index on payment_header and paid_route_id
CREATE INDEX IF NOT EXISTS idx_purchases_payment_header_route_id ON purchases (payment_header, paid_route_id);
