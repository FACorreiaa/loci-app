-- Rollback: Remove search_type column from chat_sessions

DROP INDEX IF EXISTS idx_chat_sessions_user_search_type;
DROP INDEX IF EXISTS idx_chat_sessions_search_type;

ALTER TABLE chat_sessions DROP CONSTRAINT IF EXISTS chat_sessions_search_type_check;
ALTER TABLE chat_sessions DROP COLUMN IF EXISTS search_type;
