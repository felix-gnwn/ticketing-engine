CREATE TABLE assets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    object_key VARCHAR(255) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'PENDING',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Indexing:
-- 1. Index for object_key (Use case: Fast lookup when MinIO webhook/callback notifies that a specific file has been uploaded)
CREATE INDEX idx_assets_object_key ON assets(object_key);

-- 2. Index to quickly retrieve all assets belonging to a specific user
CREATE INDEX idx_assets_user_id ON assets(user_id);