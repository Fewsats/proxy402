-- Remove cover_url, title, and description columns from paid_routes table
ALTER TABLE paid_routes DROP COLUMN cover_url;
ALTER TABLE paid_routes DROP COLUMN title;
ALTER TABLE paid_routes DROP COLUMN description; 