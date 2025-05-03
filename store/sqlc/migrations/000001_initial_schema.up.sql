-- users is a table that stores user information.
CREATE TABLE IF NOT EXISTS users (
    id BIGSERIAL PRIMARY KEY,
    
    -- email is the email associated with this user.
    email TEXT NOT NULL,
    
    -- name is the display name of the user.
    name TEXT NOT NULL,
    
    -- google_id is the unique identifier from Google OAuth.
    google_id TEXT NOT NULL,
    
    -- proxy_402_secret is the secret for the proxy 402 header.
    proxy_402_secret TEXT NOT NULL,
    
    -- Standard timestamp fields
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

-- Create unique index on email
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email ON users (email);

-- Create index on google_id for faster lookups
CREATE INDEX IF NOT EXISTS idx_users_google_id ON users (google_id);

-- paid_routes is a table that stores configurable, paid API routes.
CREATE TABLE IF NOT EXISTS paid_routes (
    id BIGSERIAL PRIMARY KEY,
    
    -- short_code is the unique identifier for the route in URLs.
    short_code TEXT NOT NULL,
    
    -- target_url is the destination URL to proxy requests to.
    target_url TEXT NOT NULL,
    
    -- method is the HTTP method allowed for this route (GET, POST, etc).
    method TEXT NOT NULL,
    
    -- price is the amount charged for accessing this route (USDC * 10^6).
    price INT NOT NULL,
    
    -- is_test indicates whether this route uses testnet or mainnet.
    is_test BOOLEAN NOT NULL,
    
    -- user_id is the owner of this route.
    user_id INT NOT NULL,
    
    -- is_enabled controls whether the route is active.
    is_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    
    -- Statistics counters
    attempt_count INT NOT NULL DEFAULT 0,
    payment_count INT NOT NULL DEFAULT 0,
    access_count INT NOT NULL DEFAULT 0,
    
    -- Standard timestamp fields
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    
    -- Soft delete support
    deleted_at TIMESTAMPTZ
);

-- Create unique index on short_code
CREATE UNIQUE INDEX IF NOT EXISTS idx_paid_routes_short_code ON paid_routes (short_code);

-- Create index on deleted_at for soft deletes
CREATE INDEX IF NOT EXISTS idx_paid_routes_deleted_at ON paid_routes (deleted_at);

-- purchases is a table that records information about successful payment transactions.
CREATE TABLE IF NOT EXISTS purchases (
    id BIGSERIAL PRIMARY KEY,
    
    -- short_code is the short code of the accessed route.
    short_code TEXT NOT NULL,
    
    -- target_url is the destination URL that was accessed.
    target_url TEXT NOT NULL,
    
    -- method is the HTTP method used.
    method TEXT NOT NULL,
    
    -- price is the amount charged (USDC * 10^6).
    price INT NOT NULL,
    
    -- is_test indicates whether this was a testnet or mainnet transaction.
    is_test BOOLEAN NOT NULL,
    
    -- payment_payload stores the X-Payment header as JSON.
    payment_payload JSONB NOT NULL,
    
    -- settle_response stores the settled response as JSON.
    settle_response JSONB NOT NULL,
    
    -- paid_route_id is the associated PaidRoute.
    paid_route_id INT NOT NULL,
    
    -- Standard timestamp fields
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    
    -- Foreign key to paid_routes
    CONSTRAINT fk_purchases_paid_route FOREIGN KEY (paid_route_id) REFERENCES paid_routes(id) ON DELETE RESTRICT
);

-- Create index on short_code for faster lookups
CREATE INDEX IF NOT EXISTS idx_purchases_short_code ON purchases (short_code);

-- Create index on paid_route_id for joins
CREATE INDEX IF NOT EXISTS idx_purchases_paid_route_id ON purchases (paid_route_id); 