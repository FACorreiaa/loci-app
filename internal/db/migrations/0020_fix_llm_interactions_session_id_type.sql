-- +goose Up
ALTER TABLE llm_interactions 
ALTER COLUMN session_id TYPE UUID USING session_id::UUID;