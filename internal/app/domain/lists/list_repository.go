package lists

import (
	"context"
	"database/sql"
	"fmt"
	"go.uber.org/zap"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/FACorreiaa/go-templui/internal/app/models"
)

// Ensure RepositoryImpl implements the Repository interface
var _ Repository = (*RepositoryImpl)(nil)

// RepositoryImpl struct holds the logger and database connection pool
type RepositoryImpl struct {
	logger *zap.Logger
	pgpool *pgxpool.Pool
}

// Repository defines the interface for lists and lists item operations
type Repository interface {
	CreateList(ctx context.Context, list models.List) error
	GetList(ctx context.Context, listID uuid.UUID) (models.List, error)
	UpdateList(ctx context.Context, list models.List) error
	GetSubLists(ctx context.Context, parentListID uuid.UUID) ([]*models.List, error)
	GetListItems(ctx context.Context, listID uuid.UUID) ([]*models.ListItem, error)

	// Generic lists item methods (support all content types)
	GetListItemByID(ctx context.Context, listID, itemID uuid.UUID) (models.ListItem, error)
	DeleteListItemByID(ctx context.Context, listID, itemID uuid.UUID) error

	// Saved Lists functionality
	SaveList(ctx context.Context, userID, listID uuid.UUID) error
	UnsaveList(ctx context.Context, userID, listID uuid.UUID) error
	GetUserSavedLists(ctx context.Context, userID uuid.UUID) ([]*models.List, error)

	// Content type specific methods
	GetListItemsByContentType(ctx context.Context, listID uuid.UUID, contentType models.ContentType) ([]*models.ListItem, error)

	// Search and filtering
	SearchLists(ctx context.Context, searchTerm, contentType string, cityID *uuid.UUID) ([]*models.List, error)

	// Legacy POI-specific methods (for backward compatibility)
	GetListItem(ctx context.Context, listID, itemID uuid.UUID, contentType string) (models.ListItem, error)
	AddListItem(ctx context.Context, item models.ListItem) error
	UpdateListItem(ctx context.Context, item models.ListItem) error
	DeleteListItem(ctx context.Context, listID, itemID uuid.UUID, contentType string) error
	DeleteList(ctx context.Context, listID uuid.UUID) error
	GetUserLists(ctx context.Context, userID uuid.UUID, isItinerary bool) ([]*models.List, error)
}

func NewRepository(pgxpool *pgxpool.Pool, logger *zap.Logger) *RepositoryImpl {
	return &RepositoryImpl{
		logger: logger,
		pgpool: pgxpool,
	}
}

// CreateList inserts a new lists into the lists table
func (r *RepositoryImpl) CreateList(ctx context.Context, list models.List) error {
	query := `
        INSERT INTO lists (
            id, user_id, name, description, image_url, is_public, is_itinerary,
            parent_list_id, city_id, view_count, save_count, created_at, updated_at
        ) VALUES (
            $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13
        )
    `
	_, err := r.pgpool.Exec(ctx, query,
		list.ID, list.UserID, list.Name, list.Description, list.ImageURL, list.IsPublic, list.IsItinerary,
		list.ParentListID, list.CityID, list.ViewCount, list.SaveCount, list.CreatedAt, list.UpdatedAt,
	)
	if err != nil {
		r.logger.Error("Failed to create lists", zap.Error(err))
		return fmt.Errorf("failed to create lists: %w", err)
	}
	return nil
}

// GetList retrieves a lists by its ID from the lists table
func (r *RepositoryImpl) GetList(ctx context.Context, listID uuid.UUID) (models.List, error) {
	query := `
        SELECT id, user_id, name, description, image_url, is_public, is_itinerary,
               parent_list_id, city_id, view_count, save_count, created_at, updated_at
        FROM lists
        WHERE id = $1
    `
	row := r.pgpool.QueryRow(ctx, query, listID)
	var list models.List
	err := row.Scan(
		&list.ID, &list.UserID, &list.Name, &list.Description, &list.ImageURL, &list.IsPublic, &list.IsItinerary,
		&list.ParentListID, &list.CityID, &list.ViewCount, &list.SaveCount, &list.CreatedAt, &list.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return models.List{}, fmt.Errorf("lists not found: %w", err)
		}
		r.logger.Error("Failed to get lists", zap.Error(err))
		return models.List{}, fmt.Errorf("failed to get lists: %w", err)
	}
	return list, nil
}

// GetSubLists retrieves all sub-lists with a given parent_list_id
func (r *RepositoryImpl) GetSubLists(ctx context.Context, parentListID uuid.UUID) ([]*models.List, error) {
	query := `
        SELECT id, user_id, name, description, image_url, is_public, is_itinerary,
               parent_list_id, city_id, view_count, save_count, created_at, updated_at
        FROM lists
        WHERE parent_list_id = $1
    `
	rows, err := r.pgpool.Query(ctx, query, parentListID)
	if err != nil {
		r.logger.Error("Failed to get sub-lists", zap.Error(err))
		return nil, fmt.Errorf("failed to get sub-lists: %w", err)
	}
	defer rows.Close()

	var subLists []*models.List
	for rows.Next() {
		var list models.List
		err := rows.Scan(
			&list.ID, &list.UserID, &list.Name, &list.Description, &list.ImageURL, &list.IsPublic, &list.IsItinerary,
			&list.ParentListID, &list.CityID, &list.ViewCount, &list.SaveCount, &list.CreatedAt, &list.UpdatedAt,
		)
		if err != nil {
			r.logger.Error("Failed to scan sub-lists", zap.Error(err))
			return nil, fmt.Errorf("failed to scan sub-lists: %w", err)
		}
		subLists = append(subLists, &list)
	}
	if err = rows.Err(); err != nil {
		r.logger.Error("Error iterating sub-lists rows", zap.Error(err))
		return nil, fmt.Errorf("error iterating sub-lists rows: %w", err)
	}
	return subLists, nil
}

// GetListItems retrieves all items associated with a specific lists, ordered by position
func (r *RepositoryImpl) GetListItems(ctx context.Context, listID uuid.UUID) ([]*models.ListItem, error) {
	query := `
        SELECT list_id, item_id, content_type, position, notes, day_number, time_slot, duration, 
               source_llm_interaction_id, item_ai_description, created_at, updated_at
        FROM list_items
        WHERE list_id = $1
        ORDER BY position
    `
	rows, err := r.pgpool.Query(ctx, query, listID)
	if err != nil {
		r.logger.Error("Failed to get lists items", zap.Error(err))
		return nil, fmt.Errorf("failed to get lists items: %w", err)
	}
	defer rows.Close()

	var items []*models.ListItem
	for rows.Next() {
		var item models.ListItem
		var dayNumber sql.NullInt32
		var timeSlot sql.NullTime
		var duration sql.NullInt32
		var sourceLlmInteractionID sql.NullString
		var itemAIDescription sql.NullString
		err := rows.Scan(
			&item.ListID, &item.ItemID, &item.ContentType, &item.Position, &item.Notes,
			&dayNumber, &timeSlot, &duration, &sourceLlmInteractionID, &itemAIDescription,
			&item.CreatedAt, &item.UpdatedAt,
		)
		if err != nil {
			r.logger.Error("Failed to scan lists item", zap.Error(err))
			return nil, fmt.Errorf("failed to scan lists item: %w", err)
		}
		if dayNumber.Valid {
			dn := int(dayNumber.Int32)
			item.DayNumber = &dn
		}
		if timeSlot.Valid {
			item.TimeSlot = &timeSlot.Time
		}
		if duration.Valid {
			dur := int(duration.Int32)
			item.Duration = &dur
		}
		if sourceLlmInteractionID.Valid {
			id, err := uuid.Parse(sourceLlmInteractionID.String)
			if err == nil {
				item.SourceLlmInteractionID = &id
			}
		}
		if itemAIDescription.Valid {
			item.ItemAIDescription = itemAIDescription.String
		}
		items = append(items, &item)
	}
	if err = rows.Err(); err != nil {
		r.logger.Error("Error iterating lists item rows", zap.Error(err))
		return nil, fmt.Errorf("error iterating lists item rows: %w", err)
	}
	return items, nil
}

// AddListItem inserts a new item into the list_items table
func (r *RepositoryImpl) AddListItem(ctx context.Context, item models.ListItem) error {
	var poiID *uuid.UUID
	// Only set poi_id for POI content type to avoid foreign key constraint violations
	if item.ContentType == models.ContentTypePOI {
		poiID = &item.ItemID
	}

	query := `
        INSERT INTO list_items (list_id, item_id, content_type, position, notes, day_number, time_slot, 
            duration, source_llm_interaction_id, item_ai_description, created_at, updated_at, poi_id) 
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
    `
	_, err := r.pgpool.Exec(ctx, query,
		item.ListID, item.ItemID, item.ContentType, item.Position, item.Notes,
		item.DayNumber, item.TimeSlot, item.Duration, item.SourceLlmInteractionID,
		item.ItemAIDescription, item.CreatedAt, item.UpdatedAt, poiID,
	)
	if err != nil {
		r.logger.Error("Failed to add lists item", zap.Error(err))
		return fmt.Errorf("failed to add lists item: %w", err)
	}
	return nil
}

// DeleteListItem deletes a specific item from the list_items table using list_id, item_id, and content_type
func (r *RepositoryImpl) DeleteListItem(ctx context.Context, listID, itemID uuid.UUID, contentType string) error {
	query := `DELETE FROM list_items WHERE list_id = $1 AND item_id = $2 AND content_type = $3`
	result, err := r.pgpool.Exec(ctx, query, listID, itemID, contentType)
	if err != nil {
		r.logger.Error("Failed to delete lists item", zap.Error(err))
		return fmt.Errorf("failed to delete lists item: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("no lists item found for list_id %s, item_id %s, and content_type %s", listID, itemID, contentType)
	}
	return nil
}

// DeleteList deletes a lists by its ID from the lists table
func (r *RepositoryImpl) DeleteList(ctx context.Context, listID uuid.UUID) error {
	query := `DELETE FROM lists WHERE id = $1`
	result, err := r.pgpool.Exec(ctx, query, listID)
	if err != nil {
		r.logger.Error("Failed to delete lists", zap.Error(err))
		return fmt.Errorf("failed to delete lists: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("no lists found with ID %s", listID)
	}
	return nil
}

// UpdateList updates a lists in the lists table
func (r *RepositoryImpl) UpdateList(ctx context.Context, list models.List) error {
	query := `
        UPDATE lists
        SET name = $1, description = $2, image_url = $3, is_public = $4, 
            city_id = $5, updated_at = $6
        WHERE id = $7
    `
	result, err := r.pgpool.Exec(ctx, query,
		list.Name, list.Description, list.ImageURL, list.IsPublic,
		list.CityID, list.UpdatedAt, list.ID,
	)
	if err != nil {
		r.logger.Error("Failed to update lists", zap.Error(err))
		return fmt.Errorf("failed to update lists: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("no lists found with ID %s", list.ID)
	}
	return nil
}

// GetListItem retrieves a specific item from the list_items table using list_id, item_id, and content_type
func (r *RepositoryImpl) GetListItem(ctx context.Context, listID, itemID uuid.UUID, contentType string) (models.ListItem, error) {
	query := `
        SELECT list_id, item_id, content_type, position, notes, day_number, time_slot, duration, 
               source_llm_interaction_id, item_ai_description, created_at, updated_at
        FROM list_items
        WHERE list_id = $1 AND item_id = $2 AND content_type = $3
    `
	row := r.pgpool.QueryRow(ctx, query, listID, itemID, contentType)
	var item models.ListItem
	var dayNumber sql.NullInt32
	var timeSlot sql.NullTime
	var duration sql.NullInt32
	var sourceLlmInteractionID sql.NullString
	var itemAIDescription sql.NullString
	err := row.Scan(
		&item.ListID, &item.ItemID, &item.ContentType, &item.Position, &item.Notes,
		&dayNumber, &timeSlot, &duration, &sourceLlmInteractionID, &itemAIDescription,
		&item.CreatedAt, &item.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return models.ListItem{}, fmt.Errorf("lists item not found: %w", err)
		}
		r.logger.Error("Failed to get lists item", zap.Error(err))
		return models.ListItem{}, fmt.Errorf("failed to get lists item: %w", err)
	}
	if dayNumber.Valid {
		dn := int(dayNumber.Int32)
		item.DayNumber = &dn
	}
	if timeSlot.Valid {
		item.TimeSlot = &timeSlot.Time
	}
	if duration.Valid {
		dur := int(duration.Int32)
		item.Duration = &dur
	}
	if sourceLlmInteractionID.Valid {
		id, err := uuid.Parse(sourceLlmInteractionID.String)
		if err == nil {
			item.SourceLlmInteractionID = &id
		}
	}
	if itemAIDescription.Valid {
		item.ItemAIDescription = itemAIDescription.String
	}
	return item, nil
}

// UpdateListItem updates an item in the list_items table (supports new generic structure)
func (r *RepositoryImpl) UpdateListItem(ctx context.Context, item models.ListItem) error {
	query := `
        UPDATE list_items
        SET item_id = $1, content_type = $2, position = $3, notes = $4, day_number = $5, 
            time_slot = $6, duration = $7, source_llm_interaction_id = $8, 
            item_ai_description = $9, updated_at = $10
        WHERE list_id = $11 AND item_id = $12
    `
	result, err := r.pgpool.Exec(ctx, query,
		item.ItemID, item.ContentType, item.Position, item.Notes, item.DayNumber,
		item.TimeSlot, item.Duration, item.SourceLlmInteractionID, item.ItemAIDescription,
		item.UpdatedAt, item.ListID, item.ItemID,
	)
	if err != nil {
		r.logger.Error("Failed to update lists item", zap.Error(err))
		return fmt.Errorf("failed to update lists item: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("no lists item found for list_id %s and item_id %s", item.ListID, item.ItemID)
	}
	return nil
}

// GetUserLists retrieves all lists for a user, optionally filtered by isItinerary
func (r *RepositoryImpl) GetUserLists(ctx context.Context, userID uuid.UUID, isItinerary bool) ([]*models.List, error) {
	query := `
        SELECT id, user_id, name, description, image_url, is_public, is_itinerary,
               parent_list_id, city_id, view_count, save_count, created_at, updated_at
        FROM lists
        WHERE user_id = $1 AND is_itinerary = $2
        ORDER BY created_at DESC
    `
	rows, err := r.pgpool.Query(ctx, query, userID, isItinerary)
	if err != nil {
		r.logger.Error("Failed to get user lists", zap.Error(err))
		return nil, fmt.Errorf("failed to get user lists: %w", err)
	}
	defer rows.Close()

	var lists []*models.List
	for rows.Next() {
		var list models.List
		err := rows.Scan(
			&list.ID, &list.UserID, &list.Name, &list.Description, &list.ImageURL, &list.IsPublic, &list.IsItinerary,
			&list.ParentListID, &list.CityID, &list.ViewCount, &list.SaveCount, &list.CreatedAt, &list.UpdatedAt,
		)
		if err != nil {
			r.logger.Error("Failed to scan lists", zap.Error(err))
			return nil, fmt.Errorf("failed to scan lists: %w", err)
		}
		lists = append(lists, &list)
	}
	if err = rows.Err(); err != nil {
		r.logger.Error("Error iterating lists rows", zap.Error(err))
		return nil, fmt.Errorf("error iterating lists rows: %w", err)
	}
	return lists, nil
}

// Generic lists item methods (support all content types)

// GetListItemByID retrieves a specific item from a lists using generic item_id
func (r *RepositoryImpl) GetListItemByID(ctx context.Context, listID, itemID uuid.UUID) (models.ListItem, error) {
	query := `
        SELECT list_id, item_id, content_type, position, notes, day_number, 
               time_slot, duration, source_llm_interaction_id, item_ai_description, 
               created_at, updated_at
        FROM list_items 
        WHERE list_id = $1 AND item_id = $2
    `
	var item models.ListItem
	err := r.pgpool.QueryRow(ctx, query, listID, itemID).Scan(
		&item.ListID, &item.ItemID, &item.ContentType, &item.Position, &item.Notes,
		&item.DayNumber, &item.TimeSlot, &item.Duration, &item.SourceLlmInteractionID,
		&item.ItemAIDescription, &item.CreatedAt, &item.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return models.ListItem{}, fmt.Errorf("no lists item found for list_id %s and item_id %s", listID, itemID)
		}
		r.logger.Error("Failed to get lists item by ID", zap.Error(err))
		return models.ListItem{}, fmt.Errorf("failed to get lists item: %w", err)
	}
	return item, nil
}

// DeleteListItemByID deletes a specific item from a lists using generic item_id
func (r *RepositoryImpl) DeleteListItemByID(ctx context.Context, listID, itemID uuid.UUID) error {
	query := `DELETE FROM list_items WHERE list_id = $1 AND item_id = $2`
	result, err := r.pgpool.Exec(ctx, query, listID, itemID)
	if err != nil {
		r.logger.Error("Failed to delete lists item by ID", zap.Error(err))
		return fmt.Errorf("failed to delete lists item: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("no lists item found for list_id %s and item_id %s", listID, itemID)
	}
	return nil
}

// SaveList saves a lists for a user (adds to saved_lists table)
func (r *RepositoryImpl) SaveList(ctx context.Context, userID, listID uuid.UUID) error {
	query := `
		INSERT INTO saved_lists (user_id, list_id, saved_at)
		VALUES ($1, $2, NOW())
		ON CONFLICT (user_id, list_id) DO NOTHING
	`
	_, err := r.pgpool.Exec(ctx, query, userID, listID)
	if err != nil {
		r.logger.Error("Failed to save lists", zap.Error(err))
		return fmt.Errorf("failed to save lists: %w", err)
	}
	return nil
}

// UnsaveList removes a saved lists for a user
func (r *RepositoryImpl) UnsaveList(ctx context.Context, userID, listID uuid.UUID) error {
	query := `DELETE FROM saved_lists WHERE user_id = $1 AND list_id = $2`
	result, err := r.pgpool.Exec(ctx, query, userID, listID)
	if err != nil {
		r.logger.Error("Failed to unsave lists", zap.Error(err))
		return fmt.Errorf("failed to unsave lists: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("lists was not saved by user")
	}
	return nil
}

// GetUserSavedLists retrieves all lists saved by a user
func (r *RepositoryImpl) GetUserSavedLists(ctx context.Context, userID uuid.UUID) ([]*models.List, error) {
	query := `
		SELECT l.id, l.user_id, l.name, l.description, l.image_url, l.is_public, l.is_itinerary,
		       l.parent_list_id, l.city_id, l.view_count, l.save_count, l.created_at, l.updated_at
		FROM lists l
		INNER JOIN saved_lists sl ON l.id = sl.list_id
		WHERE sl.user_id = $1
		ORDER BY sl.saved_at DESC
	`
	rows, err := r.pgpool.Query(ctx, query, userID)
	if err != nil {
		r.logger.Error("Failed to get user saved lists", zap.Error(err))
		return nil, fmt.Errorf("failed to get user saved lists: %w", err)
	}
	defer rows.Close()

	var lists []*models.List
	for rows.Next() {
		var list models.List
		err := rows.Scan(
			&list.ID, &list.UserID, &list.Name, &list.Description, &list.ImageURL, &list.IsPublic, &list.IsItinerary,
			&list.ParentListID, &list.CityID, &list.ViewCount, &list.SaveCount, &list.CreatedAt, &list.UpdatedAt,
		)
		if err != nil {
			r.logger.Error("Failed to scan saved lists", zap.Error(err))
			return nil, fmt.Errorf("failed to scan saved lists: %w", err)
		}
		lists = append(lists, &list)
	}
	if err = rows.Err(); err != nil {
		r.logger.Error("Error iterating saved lists rows", zap.Error(err))
		return nil, fmt.Errorf("error iterating saved lists rows: %w", err)
	}
	return lists, nil
}

// GetListItemsByContentType retrieves all items of a specific content type from a lists
func (r *RepositoryImpl) GetListItemsByContentType(ctx context.Context, listID uuid.UUID, contentType models.ContentType) ([]*models.ListItem, error) {
	query := `
		SELECT list_id, item_id, content_type, position, notes, day_number, 
		       time_slot, duration, source_llm_interaction_id, item_ai_description, 
		       created_at, updated_at
		FROM list_items 
		WHERE list_id = $1 AND content_type = $2
		ORDER BY position
	`
	rows, err := r.pgpool.Query(ctx, query, listID, contentType)
	if err != nil {
		r.logger.Error("Failed to get lists items by content type", zap.Error(err))
		return nil, fmt.Errorf("failed to get lists items by content type: %w", err)
	}
	defer rows.Close()

	var items []*models.ListItem
	for rows.Next() {
		var item models.ListItem
		err := rows.Scan(
			&item.ListID, &item.ItemID, &item.ContentType, &item.Position, &item.Notes,
			&item.DayNumber, &item.TimeSlot, &item.Duration, &item.SourceLlmInteractionID,
			&item.ItemAIDescription, &item.CreatedAt, &item.UpdatedAt,
		)
		if err != nil {
			r.logger.Error("Failed to scan lists item", zap.Error(err))
			return nil, fmt.Errorf("failed to scan lists item: %w", err)
		}
		items = append(items, &item)
	}
	if err = rows.Err(); err != nil {
		r.logger.Error("Error iterating lists item rows", zap.Error(err))
		return nil, fmt.Errorf("error iterating lists item rows: %w", err)
	}
	return items, nil
}

// SearchLists searches for lists based on various criteria
func (r *RepositoryImpl) SearchLists(ctx context.Context, searchTerm, contentType string, cityID *uuid.UUID) ([]*models.List, error) {
	query := `
		SELECT DISTINCT l.id, l.user_id, l.name, l.description, l.image_url, l.is_public, l.is_itinerary,
		       l.parent_list_id, l.city_id, l.view_count, l.save_count, l.created_at, l.updated_at
		FROM lists l
		LEFT JOIN list_items li ON l.id = li.list_id
		WHERE l.is_public = true
	`

	var args []interface{}

	if searchTerm != "" {
		query += fmt.Sprintf(" AND (l.name ILIKE $%d OR l.description ILIKE $%d)", len(args)+1, len(args)+2)
		args = append(args, "%"+searchTerm+"%", "%"+searchTerm+"%")
	}

	if cityID != nil {
		query += fmt.Sprintf(" AND l.city_id = $%d", len(args)+1)
		args = append(args, *cityID)
	}

	if contentType != "" {
		query += fmt.Sprintf(" AND li.content_type = $%d", len(args)+1)
		args = append(args, contentType)
	}

	query += " ORDER BY l.save_count DESC, l.created_at DESC"

	rows, err := r.pgpool.Query(ctx, query, args...)
	if err != nil {
		r.logger.Error("Failed to search lists", zap.Error(err))
		return nil, fmt.Errorf("failed to search lists: %w", err)
	}
	defer rows.Close()

	var lists []*models.List
	for rows.Next() {
		var list models.List
		err := rows.Scan(
			&list.ID, &list.UserID, &list.Name, &list.Description, &list.ImageURL, &list.IsPublic, &list.IsItinerary,
			&list.ParentListID, &list.CityID, &list.ViewCount, &list.SaveCount, &list.CreatedAt, &list.UpdatedAt,
		)
		if err != nil {
			r.logger.Error("Failed to scan search result", zap.Error(err))
			return nil, fmt.Errorf("failed to scan search result: %w", err)
		}
		lists = append(lists, &list)
	}
	if err = rows.Err(); err != nil {
		r.logger.Error("Error iterating search result rows", zap.Error(err))
		return nil, fmt.Errorf("error iterating search result rows: %w", err)
	}
	return lists, nil
}
