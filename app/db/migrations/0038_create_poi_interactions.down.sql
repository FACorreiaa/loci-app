-- Drop poi_interactions table and its indexes
DROP INDEX IF EXISTS idx_poi_interactions_user_category;
DROP INDEX IF EXISTS idx_poi_interactions_type;
DROP INDEX IF EXISTS idx_poi_interactions_category;
DROP INDEX IF EXISTS idx_poi_interactions_user_timestamp;
DROP INDEX IF EXISTS idx_poi_interactions_timestamp;
DROP INDEX IF EXISTS idx_poi_interactions_poi_id;
DROP INDEX IF EXISTS idx_poi_interactions_user_id;
DROP TABLE IF EXISTS poi_interactions;
