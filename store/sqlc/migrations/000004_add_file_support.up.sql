-- Add file support columns to paid_routes
ALTER TABLE paid_routes
ADD COLUMN resource_type VARCHAR(10) NOT NULL DEFAULT 'url' CHECK (resource_type IN ('url', 'file')),
ADD COLUMN original_filename TEXT;

-- Note: target_url will be used as the R2 object key when resource_type = 'file' 