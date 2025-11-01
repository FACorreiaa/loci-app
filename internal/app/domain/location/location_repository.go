package location

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/FACorreiaa/go-templui/internal/app/models"
)

type Repository interface {
	// Location History
	CreateLocationHistory(ctx context.Context, history *models.LocationHistory) error
	GetLocationHistory(ctx context.Context, userID string, limit, offset int) ([]models.LocationHistory, error)
	GetLocationHistoryByTimeRange(ctx context.Context, userID string, start, end time.Time) ([]models.LocationHistory, error)

	// POI Interactions
	CreatePOIInteraction(ctx context.Context, interaction *models.POIInteraction) error
	GetPOIInteractions(ctx context.Context, userID string, limit, offset int) ([]models.POIInteraction, error)
	GetPOIInteractionsByType(ctx context.Context, userID, interactionType string, limit, offset int) ([]models.POIInteraction, error)
	GetPOIInteractionStats(ctx context.Context, userID string) (map[string]int, error)
}

type RepositoryImpl struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) Repository {
	return &RepositoryImpl{db: db}
}

// CreateLocationHistory creates a new location history record
func (r *RepositoryImpl) CreateLocationHistory(ctx context.Context, history *models.LocationHistory) error {
	if history.ID == "" {
		history.ID = uuid.New().String()
	}
	if history.CreatedAt.IsZero() {
		history.CreatedAt = time.Now()
	}
	if history.Timestamp.IsZero() {
		history.Timestamp = time.Now()
	}

	query := `
		INSERT INTO location_history (id, user_id, latitude, longitude, radius, timestamp, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err := r.db.Exec(ctx, query,
		history.ID,
		history.UserID,
		history.Latitude,
		history.Longitude,
		history.Radius,
		history.Timestamp,
		history.CreatedAt,
	)

	return err
}

// GetLocationHistory retrieves location history for a user with pagination
func (r *RepositoryImpl) GetLocationHistory(ctx context.Context, userID string, limit, offset int) ([]models.LocationHistory, error) {
	query := `
		SELECT id, user_id, latitude, longitude, radius, timestamp, created_at
		FROM location_history
		WHERE user_id = $1
		ORDER BY timestamp DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.Query(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var histories []models.LocationHistory
	for rows.Next() {
		var h models.LocationHistory
		err := rows.Scan(
			&h.ID,
			&h.UserID,
			&h.Latitude,
			&h.Longitude,
			&h.Radius,
			&h.Timestamp,
			&h.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		histories = append(histories, h)
	}

	return histories, rows.Err()
}

// GetLocationHistoryByTimeRange retrieves location history within a time range
func (r *RepositoryImpl) GetLocationHistoryByTimeRange(ctx context.Context, userID string, start, end time.Time) ([]models.LocationHistory, error) {
	query := `
		SELECT id, user_id, latitude, longitude, radius, timestamp, created_at
		FROM location_history
		WHERE user_id = $1 AND timestamp BETWEEN $2 AND $3
		ORDER BY timestamp DESC
	`

	rows, err := r.db.Query(ctx, query, userID, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var histories []models.LocationHistory
	for rows.Next() {
		var h models.LocationHistory
		err := rows.Scan(
			&h.ID,
			&h.UserID,
			&h.Latitude,
			&h.Longitude,
			&h.Radius,
			&h.Timestamp,
			&h.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		histories = append(histories, h)
	}

	return histories, rows.Err()
}

// CreatePOIInteraction creates a new POI interaction record
func (r *RepositoryImpl) CreatePOIInteraction(ctx context.Context, interaction *models.POIInteraction) error {
	if interaction.ID == "" {
		interaction.ID = uuid.New().String()
	}
	if interaction.CreatedAt.IsZero() {
		interaction.CreatedAt = time.Now()
	}
	if interaction.Timestamp.IsZero() {
		interaction.Timestamp = time.Now()
	}

	query := `
		INSERT INTO poi_interactions (
			id, user_id, poi_id, poi_name, poi_category, interaction_type,
			user_latitude, user_longitude, poi_latitude, poi_longitude,
			distance, timestamp, created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`

	_, err := r.db.Exec(ctx, query,
		interaction.ID,
		interaction.UserID,
		interaction.POIID,
		interaction.POIName,
		interaction.POICategory,
		interaction.InteractionType,
		interaction.UserLatitude,
		interaction.UserLongitude,
		interaction.POILatitude,
		interaction.POILongitude,
		interaction.Distance,
		interaction.Timestamp,
		interaction.CreatedAt,
	)

	return err
}

// GetPOIInteractions retrieves POI interactions for a user with pagination
func (r *RepositoryImpl) GetPOIInteractions(ctx context.Context, userID string, limit, offset int) ([]models.POIInteraction, error) {
	query := `
		SELECT id, user_id, poi_id, poi_name, poi_category, interaction_type,
			   user_latitude, user_longitude, poi_latitude, poi_longitude,
			   distance, timestamp, created_at
		FROM poi_interactions
		WHERE user_id = $1
		ORDER BY timestamp DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.Query(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var interactions []models.POIInteraction
	for rows.Next() {
		var i models.POIInteraction
		err := rows.Scan(
			&i.ID,
			&i.UserID,
			&i.POIID,
			&i.POIName,
			&i.POICategory,
			&i.InteractionType,
			&i.UserLatitude,
			&i.UserLongitude,
			&i.POILatitude,
			&i.POILongitude,
			&i.Distance,
			&i.Timestamp,
			&i.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		interactions = append(interactions, i)
	}

	return interactions, rows.Err()
}

// GetPOIInteractionsByType retrieves POI interactions by type
func (r *RepositoryImpl) GetPOIInteractionsByType(ctx context.Context, userID, interactionType string, limit, offset int) ([]models.POIInteraction, error) {
	query := `
		SELECT id, user_id, poi_id, poi_name, poi_category, interaction_type,
			   user_latitude, user_longitude, poi_latitude, poi_longitude,
			   distance, timestamp, created_at
		FROM poi_interactions
		WHERE user_id = $1 AND interaction_type = $2
		ORDER BY timestamp DESC
		LIMIT $3 OFFSET $4
	`

	rows, err := r.db.Query(ctx, query, userID, interactionType, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var interactions []models.POIInteraction
	for rows.Next() {
		var i models.POIInteraction
		err := rows.Scan(
			&i.ID,
			&i.UserID,
			&i.POIID,
			&i.POIName,
			&i.POICategory,
			&i.InteractionType,
			&i.UserLatitude,
			&i.UserLongitude,
			&i.POILatitude,
			&i.POILongitude,
			&i.Distance,
			&i.Timestamp,
			&i.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		interactions = append(interactions, i)
	}

	return interactions, rows.Err()
}

// GetPOIInteractionStats retrieves interaction statistics by category
func (r *RepositoryImpl) GetPOIInteractionStats(ctx context.Context, userID string) (map[string]int, error) {
	query := `
		SELECT poi_category, COUNT(*) as count
		FROM poi_interactions
		WHERE user_id = $1
		GROUP BY poi_category
		ORDER BY count DESC
	`

	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	stats := make(map[string]int)
	for rows.Next() {
		var category string
		var count int
		err := rows.Scan(&category, &count)
		if err != nil {
			return nil, err
		}
		stats[category] = count
	}

	return stats, rows.Err()
}
