-- +migrate Down
DROP INDEX IF EXISTS idx_user_favorite_restaurants_added_at;
DROP INDEX IF EXISTS idx_user_favorite_restaurants_restaurant_id;
DROP INDEX IF EXISTS idx_user_favorite_restaurants_user_id;
DROP TABLE IF EXISTS user_favorite_restaurants;
