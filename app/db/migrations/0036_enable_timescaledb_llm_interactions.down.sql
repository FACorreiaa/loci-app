-- +migrate Down
-- Remove TimescaleDB optimizations

-- Remove retention and compression policies
SELECT remove_retention_policy('llm_interactions', if_exists => true);
SELECT remove_compression_policy('llm_interactions', if_exists => true);

-- Remove continuous aggregate policies
SELECT remove_continuous_aggregate_policy('llm_city_usage_daily', if_exists => true);
SELECT remove_continuous_aggregate_policy('llm_hourly_performance', if_exists => true);
SELECT remove_continuous_aggregate_policy('llm_daily_stats_by_intent', if_exists => true);

-- Drop materialized views
DROP MATERIALIZED VIEW IF EXISTS llm_city_usage_daily CASCADE;
DROP MATERIALIZED VIEW IF EXISTS llm_hourly_performance CASCADE;
DROP MATERIALIZED VIEW IF EXISTS llm_daily_stats_by_intent CASCADE;

-- Note: Cannot easily revert hypertable conversion without data loss
-- The table will remain a hypertable but partitioning will stop being maintained
-- To fully revert, you would need to:
-- 1. Export data
-- 2. Drop hypertable
-- 3. Recreate regular table
-- 4. Import data
-- This is intentionally not automated to prevent accidental data loss
