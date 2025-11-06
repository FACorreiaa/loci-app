-- +goose Up

-- Make poi_id column nullable to support non-POI content types
ALTER TABLE list_items ALTER COLUMN poi_id DROP NOT NULL;
-- +goose Down

-- Revert poi_id column to NOT NULL (this may fail if there are NULL values)
ALTER TABLE list_items ALTER COLUMN poi_id SET NOT NULL;