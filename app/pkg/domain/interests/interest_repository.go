package interests

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"

	"github.com/FACorreiaa/go-templui/app/lib/models"
)

var _ Repository = (*RepositoryImpl)(nil)

// interestsRepo defines the contract for user data persistence.
type Repository interface {
	// CreateInterest ---  / Interests ---
	CreateInterest(ctx context.Context, name string, description *string, isActive bool, userID string) (*models.Interest, error)
	Removeinterests(ctx context.Context, userID uuid.UUID, interestID uuid.UUID) error
	GetAllInterests(ctx context.Context) ([]*models.Interest, error)
	GetInterest(ctx context.Context, interestID uuid.UUID) (*models.Interest, error)
	Updateinterests(ctx context.Context, userID uuid.UUID, interestID uuid.UUID, params models.UpdateinterestsParams) error
	AddInterestToProfile(ctx context.Context, profileID, interestID uuid.UUID) error
	// GetInterestsForProfile retrieves all interests associated with a profile
	GetInterestsForProfile(ctx context.Context, profileID uuid.UUID) ([]*models.Interest, error)
	// GetUserEnhancedInterests retrieves all interests for a user with their preference levels
	//GetUserEnhancedInterests(ctx context.Context, userID uuid.UUID) ([]models.EnhancedInterest, error)
}

type RepositoryImpl struct {
	logger *slog.Logger
	pgpool *pgxpool.Pool
}

func NewRepositoryImpl(pgxpool *pgxpool.Pool, logger *slog.Logger) *RepositoryImpl {
	return &RepositoryImpl{
		logger: logger,
		pgpool: pgxpool,
	}
}

// CreateInterest implements user.CreateInterest
func (r *RepositoryImpl) CreateInterest(ctx context.Context, name string, description *string, isActive bool, userID string) (*models.Interest, error) {
	ctx, span := otel.Tracer("UserRepo").Start(ctx, "CreateInterest", trace.WithAttributes(
		semconv.DBSystemPostgreSQL,
		attribute.String("db.operation", "INSERT"),
		attribute.String("db.sql.table", "interests"),
		attribute.String("interest.name", name), // Add relevant attributes
		attribute.Bool("interest.active", isActive),
	))
	defer span.End()

	l := r.logger.With(slog.String("method", "CreateInterest"), slog.String("name", name))
	l.DebugContext(ctx, "Creating new global interest")

	// Input validation basic check
	if name == "" {
		span.SetStatus(codes.Error, "Interest name cannot be empty")
		return nil, fmt.Errorf("interest name cannot be empty: %w", models.ErrBadRequest) // Example domain error
	}

	var interest models.Interest
	query := `
        INSERT INTO user_custom_interests (name, description, active, created_at, updated_at, user_id)
        VALUES ($1, $2, $3, Now(), Now(), $4)
        RETURNING id, name, description, active, created_at, updated_at`

	// Note: Use current time for both created_at (via DEFAULT) and updated_at on insert
	err := r.pgpool.QueryRow(ctx, query, name, description, isActive, userID).Scan(
		&interest.ID,
		&interest.Name,
		&interest.Description,
		&interest.Active,
		&interest.CreatedAt,
		&interest.UpdatedAt, // Scan the updated_at timestamp set by the query
	)

	if err != nil {
		// Check for unique constraint violation (name already exists)
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" { // Unique violation
			l.WarnContext(ctx, "Attempted to create interest with duplicate name", slog.Any("error", err))
			span.RecordError(err)
			span.SetStatus(codes.Error, "Duplicate interest name")
			return nil, fmt.Errorf("interest with name '%s' already exists: %w", name, models.ErrConflict)
		}
		// Handle other potential errors
		l.ErrorContext(ctx, "Failed to insert new interest", slog.Any("error", err))
		span.RecordError(err)
		span.SetStatus(codes.Error, "DB INSERT failed")
		return nil, fmt.Errorf("database error creating interest: %w", err)
	}

	l.InfoContext(ctx, "Global interest created successfully", slog.String("interestID", interest.ID.String()))
	span.SetAttributes(attribute.String("db.interest.id", interest.ID.String()))
	span.SetStatus(codes.Ok, "Interest created")
	return &interest, nil
}

// Removeinterests implements user.UserRepo.
func (r *RepositoryImpl) Removeinterests(ctx context.Context, userID uuid.UUID, interestID uuid.UUID) error {
	ctx, span := otel.Tracer("UserRepo").Start(ctx, "Removeinterests", trace.WithAttributes(
		semconv.DBSystemPostgreSQL,
		attribute.String("db.operation", "DELETE"),
		attribute.String("db.sql.table", "user_custom_interests"),
		attribute.String("db.user.id", userID.String()),
		attribute.String("db.interest.id", interestID.String()),
	))
	defer span.End()

	l := r.logger.With(slog.String("method", "Removeinterests"), slog.String("userID", userID.String()), slog.String("interestID", interestID.String()))
	l.DebugContext(ctx, "Removing user interest")

	query := "DELETE FROM user_custom_interests WHERE user_id = $1 AND id = $2"
	tag, err := r.pgpool.Exec(ctx, query, userID, interestID)
	if err != nil {
		l.ErrorContext(ctx, "Failed to delete user interest", slog.Any("error", err))
		span.RecordError(err)
		span.SetStatus(codes.Error, "DB DELETE failed")
		return fmt.Errorf("database error removing interest: %w", err)
	}

	if tag.RowsAffected() == 0 {
		l.WarnContext(ctx, "Attempted to remove non-existent user interest association")
		// Return an error so the service/HandlerImpl knows the operation didn't change anything
		span.SetStatus(codes.Error, "Association not found")
		return fmt.Errorf("interest association not found: %w", models.ErrNotFound)
	}

	l.InfoContext(ctx, "User interest removed successfully")
	span.SetStatus(codes.Ok, "Interest removed")
	return nil
}

// GetAllInterests TODO does it make sense to only return the active interests ? Just mark active on the UI ?
// GetAllInterests implements user.UserRepo.
func (r *RepositoryImpl) GetAllInterests(ctx context.Context) ([]*models.Interest, error) {
	ctx, span := otel.Tracer("UserRepo").Start(ctx, "GetAllInterests", trace.WithAttributes(
		semconv.DBSystemPostgreSQL,
		attribute.String("db.sql.table", "interests"),
	))
	defer span.End()

	l := r.logger.With(slog.String("method", "GetAllInterests"))
	l.DebugContext(ctx, "Fetching all active interests")

	query := `
        SELECT id, name, description, 
               CASE WHEN 'global' = 'global' THEN false ELSE active END AS active, 
               created_at, updated_at, 'global' AS type
		FROM interests
		UNION
		SELECT id, name, description, active, created_at, updated_at, 'custom' AS type
		FROM user_custom_interests 
        ORDER BY name`

	rows, err := r.pgpool.Query(ctx, query)
	if err != nil {
		l.ErrorContext(ctx, "Failed to query all interests", slog.Any("error", err))
		span.RecordError(err)
		span.SetStatus(codes.Error, "DB query failed")
		return nil, fmt.Errorf("database error fetching interests: %w", err)
	}
	defer rows.Close()

	var interests []*models.Interest
	for rows.Next() {
		var i models.Interest
		err := rows.Scan(
			&i.ID, &i.Name, &i.Description, &i.Active, &i.CreatedAt, &i.UpdatedAt, &i.Source,
		)
		if err != nil {
			l.ErrorContext(ctx, "Failed to scan interest row", slog.Any("error", err))
			span.RecordError(err)
			return nil, fmt.Errorf("database error scanning interest: %w", err)
		}
		interests = append(interests, &i)
	}

	if err = rows.Err(); err != nil {
		l.ErrorContext(ctx, "Error iterating interests rows", slog.Any("error", err))
		span.RecordError(err)
		return nil, fmt.Errorf("database error reading interests: %w", err)
	}

	l.DebugContext(ctx, "Fetched all active interests successfully", slog.Int("count", len(interests)))
	span.SetStatus(codes.Ok, "Interests fetched")
	return interests, nil
}

// GetUserEnhancedInterests implements user.UserRepo.
//func (r *RepositoryImpl) GetUserEnhancedInterests(ctx context.Context, userID uuid.UUID) ([]models.EnhancedInterest, error) {
//	ctx, span := otel.Tracer("UserRepo").Start(ctx, "GetUserEnhancedInterests", trace.WithAttributes(
//		semconv.DBSystemPostgreSQL,
//		attribute.String("db.sql.table", "user_custom_interests, interests"),
//		attribute.String("db.user.id", userID.String()),
//	))
//	defer span.End()
//
//	l := r.logger.With(slog.String("method", "GetUserEnhancedInterests"), slog.String("userID", userID.String()))
//	l.DebugContext(ctx, "Fetching user enhanced interests")
//
//	query := `
//        SELECT i.id, i.name, i.description, i.active, i.created_at, i.updated_at, ui.preference_level
//        FROM interests i
//        JOIN user_custom_interests ui ON i.id = ui.interest_id
//        WHERE ui.user_id = $1 AND i.active = TRUE
//        ORDER BY ui.preference_level DESC, i.name`
//
//	rows, err := r.pgpool.Query(ctx, query, userID)
//	if err != nil {
//		l.ErrorContext(ctx, "Failed to query user enhanced interests", slog.Any("error", err))
//		span.RecordError(err)
//		span.SetStatus(codes.Error, "DB query failed")
//		return nil, fmt.Errorf("database error fetching enhanced interests: %w", err)
//	}
//	defer rows.Close()
//
//	var interests []models.EnhancedInterest
//	for rows.Next() {
//		var i models.EnhancedInterest
//		err := rows.Scan(
//			&i.ID, &i.Name, &i.Description, &i.Active, &i.CreatedAt, &i.UpdatedAt, &i.PreferenceLevel,
//		)
//		if err != nil {
//			l.ErrorContext(ctx, "Failed to scan enhanced interest row", slog.Any("error", err))
//			span.RecordError(err)
//			return nil, fmt.Errorf("database error scanning enhanced interest: %w", err)
//		}
//		interests = append(interests, i)
//	}
//
//	if err = rows.Err(); err != nil {
//		l.ErrorContext(ctx, "Error iterating enhanced interest rows", slog.Any("error", err))
//		span.RecordError(err)
//		return nil, fmt.Errorf("database error reading enhanced interests: %w", err)
//	}
//
//	l.DebugContext(ctx, "Fetched user enhanced interests successfully", slog.Int("count", len(interests)))
//	span.SetStatus(codes.Ok, "Enhanced interests fetched")
//	return interests, nil
//}

func (r *RepositoryImpl) Updateinterests(ctx context.Context, userID uuid.UUID, interestID uuid.UUID, params models.UpdateinterestsParams) error {
	ctx, span := otel.Tracer("UserRepo").Start(ctx, "UpdateUserCustomInterest", trace.WithAttributes(
		semconv.DBSystemPostgreSQL,
		attribute.String("db.operation", "UPDATE"),
		attribute.String("db.sql.table", "user_custom_interests"),
		attribute.String("db.user.id", userID.String()),
		attribute.String("db.interest.id", interestID.String()),
	))
	defer span.End()

	l := r.logger.With(
		slog.String("method", "UpdateUserCustomInterest"),
		slog.String("userID", userID.String()),
		slog.String("interestID", interestID.String()),
	)
	l.DebugContext(ctx, "Updating user custom interest", slog.Any("params", params))

	// Build dynamic query
	setClauses := []string{}
	args := []interface{}{}
	argID := 1 // Start placeholders at $1

	// --- Add parameters dynamically ---
	if params.Name != nil {
		if *params.Name == "" { // Basic validation
			err := errors.New("custom interest name cannot be empty")
			span.RecordError(err)
			span.SetStatus(codes.Error, "Invalid input: empty name")
			return fmt.Errorf("%w: %w", models.ErrBadRequest, err)
		}
		setClauses = append(setClauses, fmt.Sprintf("name = $%d", argID))
		args = append(args, *params.Name)
		argID++
		span.SetAttributes(attribute.Bool("update.name", true))
	}
	// Description can be explicitly set to null/empty if needed
	if params.Description != nil {
		setClauses = append(setClauses, fmt.Sprintf("description = $%d", argID))
		args = append(args, params.Description) // Pass pointer directly, pgx handles nil
		argID++
		span.SetAttributes(attribute.Bool("update.description", true))
	}
	if params.Active != nil {
		setClauses = append(setClauses, fmt.Sprintf("active = $%d", argID))
		args = append(args, *params.Active)
		argID++
		span.SetAttributes(attribute.Bool("update.active", true))
	}

	// If no fields to update, return early
	if len(setClauses) == 0 {
		l.InfoContext(ctx, "No fields provided to update custom interest")
		span.SetStatus(codes.Ok, "No update fields")
		return nil // Or return models.ErrBadRequest("no fields provided for update")
	}

	// Always update updated_at
	setClauses = append(setClauses, fmt.Sprintf("updated_at = $%d", argID))
	args = append(args, time.Now())
	argID++

	// Add WHERE clause parameters last
	args = append(args, interestID) // Placeholder corresponding to WHERE id = $N
	idPlaceholder := argID
	argID++
	args = append(args, userID) // Placeholder corresponding to WHERE user_id = $N+1
	userIDPlaceholder := argID

	// Construct query
	query := fmt.Sprintf(`UPDATE user_custom_interests
                          SET %s
                          WHERE id = $%d AND user_id = $%d`,
		strings.Join(setClauses, ", "),
		idPlaceholder,
		userIDPlaceholder,
	)

	l.DebugContext(ctx, "Executing dynamic update query", slog.String("query", query))

	// Execute query
	tag, err := r.pgpool.Exec(ctx, query, args...)
	if err != nil {
		// Check for unique constraint on (user_id, name) if name was updated
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" && params.Name != nil {
			l.WarnContext(ctx, "Attempted to update custom interest to a duplicate name for this user", slog.Any("error", err))
			span.RecordError(err)
			span.SetStatus(codes.Error, "Duplicate custom interest name")
			return fmt.Errorf("you already have a custom interest named '%s': %w", *params.Name, models.ErrConflict)
		}
		// Handle other potential errors
		l.ErrorContext(ctx, "Failed to execute update custom interest query", slog.Any("error", err))
		span.RecordError(err)
		span.SetStatus(codes.Error, "DB UPDATE failed")
		return fmt.Errorf("database error updating custom interest: %w", err)
	}

	// Check if the specific interest owned by the user was found and updated
	if tag.RowsAffected() == 0 {
		l.WarnContext(ctx, "Custom interest not found for update or user mismatch", slog.Int64("rows_affected", tag.RowsAffected()))
		span.SetStatus(codes.Error, "Custom interest not found or permission denied")
		// It's crucial to return NotFound here, as the combination wasn't found
		return fmt.Errorf("custom interest with ID %s not found for user %s: %w", interestID.String(), userID.String(), models.ErrNotFound)
	}

	l.InfoContext(ctx, "User custom interest updated successfully")
	span.SetStatus(codes.Ok, "Custom interest updated")
	return nil
}

func (r *RepositoryImpl) GetInterest(ctx context.Context, interestID uuid.UUID) (*models.Interest, error) {
	var interest models.Interest
	ctx, span := otel.Tracer("UserRepo").Start(ctx, "GetInterest", trace.WithAttributes(
		semconv.DBSystemPostgreSQL,
		attribute.String("db.sql.table", "interests"),
		attribute.String("db.interest.id", interestID.String()),
	))
	defer span.End()

	l := r.logger.With(slog.String("method", "GetInterest"), slog.String("interestID", interestID.String()))
	l.DebugContext(ctx, "Fetching interest")

	query := `
		SELECT id, name, description, active, created_at, updated_at, type FROM (
			SELECT id, name, description, 
			       CASE WHEN 'global' = 'global' THEN false ELSE active END AS active, 
			       created_at, updated_at, 'global' AS type
			FROM interests
			UNION
			SELECT id, name, description, active, created_at, updated_at, 'custom' AS type
			FROM user_custom_interests 
		) AS combined_interests
        WHERE id = $1`

	err := r.pgpool.QueryRow(ctx, query, interestID).Scan(
		&interest.ID,
		&interest.Name,
		&interest.Description,
		&interest.Active,
		&interest.CreatedAt,
		&interest.UpdatedAt,
		&interest.Source,
	)

	if err != nil {
		return nil, fmt.Errorf("database error fetching interest: %w", err)
	}

	return &interest, nil
}

func (r *RepositoryImpl) AddInterestToProfile(ctx context.Context, profileID, interestID uuid.UUID) error {
	ctx, span := otel.Tracer("UserRepo").Start(ctx, "AddInterestToProfile", trace.WithAttributes(
		semconv.DBSystemPostgreSQL,
		attribute.String("db.operation", "INSERT"),
		attribute.String("db.sql.table", "user_profile_interests"),
		attribute.String("db.profile.id", profileID.String()),
		attribute.String("db.interest.id", interestID.String()),
	))
	defer span.End()

	l := r.logger.With(slog.String("method", "AddInterestToProfile"), slog.String("profileID", profileID.String()), slog.String("interestID", interestID.String()))
	l.DebugContext(ctx, "Linking interest to profile")

	query := `
        INSERT INTO user_profile_interests (profile_id, interest_id, preference_level)
        VALUES ($1, $2, $3)
        ON CONFLICT DO NOTHING`

	_, err := r.pgpool.Exec(ctx, query, profileID, interestID, 1) // Default preference_level = 1
	if err != nil {
		l.ErrorContext(ctx, "Failed to link interest to profile", slog.Any("error", err))
		span.RecordError(err)
		span.SetStatus(codes.Error, "DB INSERT failed")
		return fmt.Errorf("database error linking interest to profile: %w", err)
	}

	l.DebugContext(ctx, "Interest linked to profile successfully")
	span.SetStatus(codes.Ok, "Interest linked")
	return nil
}

// GetInterestsForProfile retrieves all interests associated with a profile
func (r *RepositoryImpl) GetInterestsForProfile(ctx context.Context, profileID uuid.UUID) ([]*models.Interest, error) {
	ctx, span := otel.Tracer("UserRepo").Start(ctx, "GetInterestsForProfile", trace.WithAttributes(
		semconv.DBSystemPostgreSQL,
		attribute.String("db.operation", "SELECT"),
		attribute.String("db.sql.table", "interests"),
		attribute.String("db.profile.id", profileID.String()),
	))
	defer span.End()

	l := r.logger.With(slog.String("method", "GetInterestsForProfile"), slog.String("profileID", profileID.String()))
	l.DebugContext(ctx, "Fetching interests for profile")

	query := `
        SELECT i.id, i.name, i.description, i.active
        FROM interests i
        JOIN user_profile_interests upi ON i.id = upi.interest_id
        WHERE upi.profile_id = $1`

	rows, err := r.pgpool.Query(ctx, query, profileID)
	if err != nil {
		l.ErrorContext(ctx, "Failed to query interests for profile", slog.Any("error", err))
		span.RecordError(err)
		span.SetStatus(codes.Error, "DB query failed")
		return nil, fmt.Errorf("database error fetching interests for profile: %w", err)
	}
	defer rows.Close()

	var interests []*models.Interest
	for rows.Next() {
		var interest models.Interest
		err := rows.Scan(
			&interest.ID,
			&interest.Name,
			&interest.Description,
			&interest.Active,
		)
		if err != nil {
			l.ErrorContext(ctx, "Failed to scan interest row", slog.Any("error", err))
			span.RecordError(err)
			return nil, fmt.Errorf("database error scanning interest: %w", err)
		}
		interests = append(interests, &interest)
	}

	if err = rows.Err(); err != nil {
		l.ErrorContext(ctx, "Error iterating interest rows", slog.Any("error", err))
		span.RecordError(err)
		return nil, fmt.Errorf("database error reading interests: %w", err)
	}

	l.DebugContext(ctx, "Fetched interests for profile successfully", slog.Int("count", len(interests)))
	span.SetStatus(codes.Ok, "Interests fetched")
	return interests, nil
}
