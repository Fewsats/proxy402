-- Add cover_url, title, and description columns to paid_routes table
ALTER TABLE paid_routes ADD COLUMN cover_url TEXT;
ALTER TABLE paid_routes ADD COLUMN title TEXT;
ALTER TABLE paid_routes ADD COLUMN description TEXT;