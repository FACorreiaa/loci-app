package models

import "time"

// LocationHistory represents a user's location history
type LocationHistory struct {
	ID        string    `json:"id" db:"id"`
	UserID    string    `json:"user_id" db:"user_id"`
	Latitude  float64   `json:"latitude" db:"latitude"`
	Longitude float64   `json:"longitude" db:"longitude"`
	Radius    float64   `json:"radius" db:"radius"`
	Timestamp time.Time `json:"timestamp" db:"timestamp"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// POIInteraction represents a user's interaction with a POI
type POIInteraction struct {
	ID             string    `json:"id" db:"id"`
	UserID         string    `json:"user_id" db:"user_id"`
	POIID          string    `json:"poi_id" db:"poi_id"`
	POIName        string    `json:"poi_name" db:"poi_name"`
	POICategory    string    `json:"poi_category" db:"poi_category"`
	InteractionType string   `json:"interaction_type" db:"interaction_type"` // "view", "click", "favorite"
	UserLatitude   float64   `json:"user_latitude" db:"user_latitude"`
	UserLongitude  float64   `json:"user_longitude" db:"user_longitude"`
	POILatitude    float64   `json:"poi_latitude" db:"poi_latitude"`
	POILongitude   float64   `json:"poi_longitude" db:"poi_longitude"`
	Distance       float64   `json:"distance" db:"distance"`
	Timestamp      time.Time `json:"timestamp" db:"timestamp"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
}
