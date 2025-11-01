-- +migrate Down
-- Remove enhanced LLM logging fields

-- Drop trigger and function
DROP TRIGGER IF EXISTS trigger_update_llm_interactions_updated_at ON llm_interactions;
DROP FUNCTION IF EXISTS update_llm_interactions_updated_at();

-- Drop indexes
DROP INDEX IF EXISTS idx_llm_interactions_request_id;
DROP INDEX IF EXISTS idx_llm_interactions_status_code;
DROP INDEX IF EXISTS idx_llm_interactions_intent;
DROP INDEX IF EXISTS idx_llm_interactions_search_type;
DROP INDEX IF EXISTS idx_llm_interactions_provider;
DROP INDEX IF EXISTS idx_llm_interactions_cache_hit;
DROP INDEX IF EXISTS idx_llm_interactions_updated_at;
DROP INDEX IF EXISTS idx_llm_interactions_device_type;
DROP INDEX IF EXISTS idx_llm_interactions_user_intent;
DROP INDEX IF EXISTS idx_llm_interactions_city_intent;
DROP INDEX IF EXISTS idx_llm_interactions_performance;

-- Remove columns (in reverse order of addition)
ALTER TABLE llm_interactions
    DROP COLUMN IF EXISTS stream_duration_ms,
    DROP COLUMN IF EXISTS stream_chunks_count,
    DROP COLUMN IF EXISTS is_streaming,
    DROP COLUMN IF EXISTS is_pii_redacted,
    DROP COLUMN IF EXISTS prompt_hash,
    DROP COLUMN IF EXISTS user_agent,
    DROP COLUMN IF EXISTS platform,
    DROP COLUMN IF EXISTS device_type,
    DROP COLUMN IF EXISTS cache_key,
    DROP COLUMN IF EXISTS cache_hit,
    DROP COLUMN IF EXISTS user_feedback_timestamp,
    DROP COLUMN IF EXISTS user_feedback_comment,
    DROP COLUMN IF EXISTS user_feedback_rating,
    DROP COLUMN IF EXISTS cost_estimate_usd,
    DROP COLUMN IF EXISTS max_tokens,
    DROP COLUMN IF EXISTS top_k,
    DROP COLUMN IF EXISTS top_p,
    DROP COLUMN IF EXISTS temperature,
    DROP COLUMN IF EXISTS search_type,
    DROP COLUMN IF EXISTS intent,
    DROP COLUMN IF EXISTS provider,
    DROP COLUMN IF EXISTS error_message,
    DROP COLUMN IF EXISTS status_code,
    DROP COLUMN IF EXISTS updated_at,
    DROP COLUMN IF EXISTS request_id;
