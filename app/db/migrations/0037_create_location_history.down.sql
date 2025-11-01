-- Drop location_history table and its indexes
DROP INDEX IF EXISTS idx_location_history_coordinates;
DROP INDEX IF EXISTS idx_location_history_user_timestamp;
DROP INDEX IF EXISTS idx_location_history_timestamp;
DROP INDEX IF EXISTS idx_location_history_user_id;
DROP TABLE IF EXISTS location_history;
