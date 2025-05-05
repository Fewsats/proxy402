-- Add payment_address column to users table
ALTER TABLE users ADD COLUMN payment_address TEXT NOT NULL DEFAULT '';

-- Create index on payment_address for faster lookups
CREATE INDEX IF NOT EXISTS idx_users_payment_address ON users (payment_address);
