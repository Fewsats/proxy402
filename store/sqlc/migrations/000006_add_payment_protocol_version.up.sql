ALTER TABLE paid_routes
ADD COLUMN payment_protocol_version SMALLINT NOT NULL DEFAULT 1
CHECK (payment_protocol_version IN (1, 2));
