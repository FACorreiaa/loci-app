-- +goose Up

BEGIN;

-- Add session_id column to store the chat session ID
ALTER TABLE user_saved_itineraries 
ADD COLUMN session_id UUID NULL REFERENCES chat_sessions (id) ON DELETE SET NULL;

-- Add index for faster lookups by session_id
CREATE INDEX idx_user_saved_itineraries_session_id ON user_saved_itineraries (session_id);

-- Add composite index for user + session lookups
CREATE INDEX idx_user_saved_itineraries_user_session ON user_saved_itineraries (user_id, session_id);

COMMIT;
-- +goose Down

BEGIN;

-- Drop the indexes first
DROP INDEX IF EXISTS idx_user_saved_itineraries_session_id;
DROP INDEX IF EXISTS idx_user_saved_itineraries_user_session;

-- Drop the session_id column
ALTER TABLE user_saved_itineraries 
DROP COLUMN IF EXISTS session_id;

COMMIT;