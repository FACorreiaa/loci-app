-- +migrate Down
DROP INDEX IF EXISTS idx_user_favorite_hotels_added_at;
DROP INDEX IF EXISTS idx_user_favorite_hotels_hotel_id;
DROP INDEX IF EXISTS idx_user_favorite_hotels_user_id;
DROP TABLE IF EXISTS user_favorite_hotels;
