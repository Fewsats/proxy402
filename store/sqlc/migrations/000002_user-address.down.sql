-- Drop index on payment_address
DROP INDEX IF EXISTS idx_users_payment_address;

-- Remove payment_address column from users table
ALTER TABLE users DROP COLUMN IF EXISTS payment_address;
