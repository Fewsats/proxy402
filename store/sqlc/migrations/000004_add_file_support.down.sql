-- Drop indexes first
DROP INDEX IF EXISTS idx_file_resources_r2_object_key;
DROP INDEX IF EXISTS idx_file_resources_paid_route_id;

-- Drop file_resources table
DROP TABLE IF EXISTS file_resources;

-- Remove file support columns from paid_routes
ALTER TABLE paid_routes
DROP COLUMN IF EXISTS resource_type,
DROP COLUMN IF EXISTS original_filename; 