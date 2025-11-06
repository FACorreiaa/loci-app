-- +goose Up
-- This prevents duplicate POIs with same name and location
ALTER TABLE llm_suggested_pois 
ADD CONSTRAINT unique_llm_suggested_poi_name_location 
UNIQUE (name, latitude, longitude);
-- +goose Down
ALTER TABLE llm_suggested_pois 
DROP CONSTRAINT IF EXISTS unique_llm_suggested_poi_name_location;