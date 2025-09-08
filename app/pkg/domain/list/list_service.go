package itinerarylist

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/FACorreiaa/go-templui/app/internal/models"
)

var _ Service = (*ServiceImpl)(nil)

type Service interface {
	CreateTopLevelList(ctx context.Context, userID uuid.UUID, name, description string, cityID *uuid.UUID, isItinerary, isPublic bool) (*models.List, error)
	CreateItineraryForList(ctx context.Context, userID, parentListID uuid.UUID, name, description string, isPublic bool) (*models.List, error)
	GetListDetails(ctx context.Context, listID, userID uuid.UUID) (*models.ListWithItems, error)
	UpdateListDetails(ctx context.Context, listID, userID uuid.UUID, params models.UpdateListRequest) (*models.List, error)
	DeleteUserList(ctx context.Context, listID, userID uuid.UUID) error

	// Generic list item methods (support all content types)
	AddListItem(ctx context.Context, userID, listID uuid.UUID, params models.AddListItemRequest) (*models.ListItem, error)
	UpdateListItem(ctx context.Context, userID, listID, itemID uuid.UUID, params models.UpdateListItemRequest) (*models.ListItem, error)
	RemoveListItem(ctx context.Context, userID, listID, itemID uuid.UUID) error

	// Saved Lists functionality
	SaveList(ctx context.Context, userID, listID uuid.UUID) error
	UnsaveList(ctx context.Context, userID, listID uuid.UUID) error
	GetUserSavedLists(ctx context.Context, userID uuid.UUID) ([]*models.List, error)

	// Content type specific methods
	GetListItemsByContentType(ctx context.Context, userID, listID uuid.UUID, contentType models.ContentType) ([]*models.ListItem, error)

	// Search and filtering
	SearchLists(ctx context.Context, searchTerm, contentType string, cityID *uuid.UUID) ([]*models.List, error)

	// Legacy POI-specific methods (for backward compatibility)
	AddPOIListItem(ctx context.Context, userID, listID, poiID uuid.UUID, params models.AddListItemRequest) (*models.ListItem, error)
	UpdatePOIListItem(ctx context.Context, userID, listID, poiID uuid.UUID, params models.UpdateListItemRequest) (*models.ListItem, error)
	RemovePOIListItem(ctx context.Context, userID, listID, poiID uuid.UUID) error

	GetUserLists(ctx context.Context, userID uuid.UUID, isItinerary bool) ([]*models.List, error)
}

type ServiceImpl struct {
	logger         *slog.Logger
	listRepository Repository
}

// NewServiceImpl creates a new instance of ServiceImpl
func NewServiceImpl(repo Repository, logger *slog.Logger) *ServiceImpl {
	return &ServiceImpl{
		logger:         logger,
		listRepository: repo,
	}
}

// CreateTopLevelList creates a new top-level list
func (s *ServiceImpl) CreateTopLevelList(ctx context.Context, userID uuid.UUID, name, description string, cityID *uuid.UUID, isItinerary, isPublic bool) (*models.List, error) {
	ctx, span := otel.Tracer("ItineraryListService").Start(ctx, "CreateTopLevelList", trace.WithAttributes(
		attribute.String("user.id", userID.String()),
		attribute.String("list.name", name),
		attribute.Bool("list.is_itinerary", isItinerary),
	))
	defer span.End()

	l := s.logger.With(slog.String("method", "CreateTopLevelList"), slog.String("userID", userID.String()))
	l.DebugContext(ctx, "Creating top-level list")

	list := models.List{
		ID:          uuid.New(),
		UserID:      userID,
		Name:        name,
		Description: description,
		IsPublic:    isPublic,
		IsItinerary: isItinerary,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Set CityID if provided
	if cityID != nil {
		list.CityID = cityID
	}

	err := s.listRepository.CreateList(ctx, list)
	if err != nil {
		l.ErrorContext(ctx, "Failed to create top-level list", slog.Any("error", err))
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to create list")
		return nil, fmt.Errorf("failed to create list: %w", err)
	}

	l.InfoContext(ctx, "Top-level list created successfully", slog.String("listID", list.ID.String()))
	span.SetStatus(codes.Ok, "List created")
	return &list, nil
}

// CreateItineraryForList creates a new itinerary within a parent list
func (s *ServiceImpl) CreateItineraryForList(ctx context.Context, userID, parentListID uuid.UUID, name, description string, isPublic bool) (*models.List, error) {
	ctx, span := otel.Tracer("ItineraryListService").Start(ctx, "CreateItineraryForList", trace.WithAttributes(
		attribute.String("user.id", userID.String()),
		attribute.String("parent_list.id", parentListID.String()),
		attribute.String("itinerary.name", name),
	))
	defer span.End()

	l := s.logger.With(slog.String("method", "CreateItineraryForList"),
		slog.String("userID", userID.String()),
		slog.String("parentListID", parentListID.String()))
	l.DebugContext(ctx, "Creating itinerary for list")

	// Fetch parent list to verify ownership and inherit cityID
	parentList, err := s.listRepository.GetList(ctx, parentListID)
	if err != nil {
		l.ErrorContext(ctx, "Failed to fetch parent list", slog.Any("error", err))
		span.RecordError(err)
		span.SetStatus(codes.Error, "Parent list not found")
		return nil, fmt.Errorf("parent list not found: %w", err)
	}

	// Verify ownership
	if parentList.UserID != userID {
		l.WarnContext(ctx, "User does not own parent list",
			slog.String("listOwnerID", parentList.UserID.String()))
		span.SetStatus(codes.Error, "User does not own parent list")
		return nil, fmt.Errorf("user does not own parent list")
	}

	// Create the itinerary
	itinerary := models.List{
		ID:           uuid.New(),
		UserID:       userID,
		Name:         name,
		Description:  description,
		IsPublic:     isPublic,
		IsItinerary:  true,
		ParentListID: &parentListID,
		CityID:       parentList.CityID, // Inherit from parent
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	err = s.listRepository.CreateList(ctx, itinerary)
	if err != nil {
		l.ErrorContext(ctx, "Failed to create itinerary", slog.Any("error", err))
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to create itinerary")
		return nil, fmt.Errorf("failed to create itinerary: %w", err)
	}

	l.InfoContext(ctx, "Itinerary created successfully", slog.String("itineraryID", itinerary.ID.String()))
	span.SetStatus(codes.Ok, "Itinerary created")
	return &itinerary, nil
}

// GetListDetails retrieves a list with all its items
func (s *ServiceImpl) GetListDetails(ctx context.Context, listID, userID uuid.UUID) (*models.ListWithItems, error) {
	ctx, span := otel.Tracer("ItineraryListService").Start(ctx, "GetListDetails", trace.WithAttributes(
		attribute.String("list.id", listID.String()),
		attribute.String("user.id", userID.String()),
	))
	defer span.End()

	l := s.logger.With(slog.String("method", "GetListDetails"),
		slog.String("listID", listID.String()),
		slog.String("userID", userID.String()))
	l.DebugContext(ctx, "Getting list details")

	// Fetch the list
	list, err := s.listRepository.GetList(ctx, listID)
	if err != nil {
		l.ErrorContext(ctx, "Failed to fetch list", slog.Any("error", err))
		span.RecordError(err)
		span.SetStatus(codes.Error, "List not found")
		return nil, fmt.Errorf("list not found: %w", err)
	}

	// Check if user has access (owner or public list)
	if list.UserID != userID && !list.IsPublic {
		l.WarnContext(ctx, "Access denied to list",
			slog.String("listOwnerID", list.UserID.String()))
		span.SetStatus(codes.Error, "Access denied")
		return nil, fmt.Errorf("access denied to list")
	}

	// Fetch list items if it's an itinerary
	var items []*models.ListItem
	if list.IsItinerary {
		items, err = s.listRepository.GetListItems(ctx, listID)
		if err != nil {
			l.ErrorContext(ctx, "Failed to fetch list items", slog.Any("error", err))
			span.RecordError(err)
			span.SetStatus(codes.Error, "Failed to fetch list items")
			return nil, fmt.Errorf("failed to fetch list items: %w", err)
		}
	}

	result := &models.ListWithItems{
		List:  list,
		Items: items,
	}

	l.InfoContext(ctx, "List details fetched successfully",
		slog.Int("itemCount", len(items)))
	span.SetStatus(codes.Ok, "List details fetched")
	return result, nil
}

// UpdateListDetails updates a list's details
func (s *ServiceImpl) UpdateListDetails(ctx context.Context, listID, userID uuid.UUID, params models.UpdateListRequest) (*models.List, error) {
	ctx, span := otel.Tracer("ItineraryListService").Start(ctx, "UpdateListDetails", trace.WithAttributes(
		attribute.String("list.id", listID.String()),
		attribute.String("user.id", userID.String()),
	))
	defer span.End()

	l := s.logger.With(slog.String("method", "UpdateListDetails"),
		slog.String("listID", listID.String()),
		slog.String("userID", userID.String()))
	l.DebugContext(ctx, "Updating list details")

	// Fetch the list to verify ownership
	list, err := s.listRepository.GetList(ctx, listID)
	if err != nil {
		l.ErrorContext(ctx, "Failed to fetch list", slog.Any("error", err))
		span.RecordError(err)
		span.SetStatus(codes.Error, "List not found")
		return nil, fmt.Errorf("list not found: %w", err)
	}

	// Verify ownership
	if list.UserID != userID {
		l.WarnContext(ctx, "User does not own list",
			slog.String("listOwnerID", list.UserID.String()))
		span.SetStatus(codes.Error, "User does not own list")
		return nil, fmt.Errorf("user does not own list")
	}

	// Update fields if provided
	if params.Name != nil {
		list.Name = *params.Name
	}
	if params.Description != nil {
		list.Description = *params.Description
	}
	if params.ImageURL != nil {
		list.ImageURL = *params.ImageURL
	}
	if params.IsPublic != nil {
		list.IsPublic = *params.IsPublic
	}
	if params.CityID != nil {
		list.CityID = params.CityID
	}
	list.UpdatedAt = time.Now()

	// Update the list in the repository
	// Note: We need to add an UpdateList method to the repository
	err = s.listRepository.UpdateList(ctx, list)
	if err != nil {
		l.ErrorContext(ctx, "Failed to update list", slog.Any("error", err))
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to update list")
		return nil, fmt.Errorf("failed to update list: %w", err)
	}

	l.InfoContext(ctx, "List updated successfully")
	span.SetStatus(codes.Ok, "List updated")
	return &list, nil
}

// DeleteUserList deletes a list
func (s *ServiceImpl) DeleteUserList(ctx context.Context, listID, userID uuid.UUID) error {
	ctx, span := otel.Tracer("ItineraryListService").Start(ctx, "DeleteUserList", trace.WithAttributes(
		attribute.String("list.id", listID.String()),
		attribute.String("user.id", userID.String()),
	))
	defer span.End()

	l := s.logger.With(slog.String("method", "DeleteUserList"),
		slog.String("listID", listID.String()),
		slog.String("userID", userID.String()))
	l.DebugContext(ctx, "Deleting list")

	// Fetch the list to verify ownership
	list, err := s.listRepository.GetList(ctx, listID)
	if err != nil {
		l.ErrorContext(ctx, "Failed to fetch list", slog.Any("error", err))
		span.RecordError(err)
		span.SetStatus(codes.Error, "List not found")
		return fmt.Errorf("list not found: %w", err)
	}

	// Verify ownership
	if list.UserID != userID {
		l.WarnContext(ctx, "User does not own list",
			slog.String("listOwnerID", list.UserID.String()))
		span.SetStatus(codes.Error, "User does not own list")
		return fmt.Errorf("user does not own list")
	}

	// Delete the list
	err = s.listRepository.DeleteList(ctx, listID)
	if err != nil {
		l.ErrorContext(ctx, "Failed to delete list", slog.Any("error", err))
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to delete list")
		return fmt.Errorf("failed to delete list: %w", err)
	}

	l.InfoContext(ctx, "List deleted successfully")
	span.SetStatus(codes.Ok, "List deleted")
	return nil
}

// Generic list item methods (support all content types)

// AddListItem adds any type of content to a list
func (s *ServiceImpl) AddListItem(ctx context.Context, userID, listID uuid.UUID, params models.AddListItemRequest) (*models.ListItem, error) {
	ctx, span := otel.Tracer("ItineraryListService").Start(ctx, "AddListItem", trace.WithAttributes(
		attribute.String("list.id", listID.String()),
		attribute.String("user.id", userID.String()),
		attribute.String("item.id", params.ItemID.String()),
		attribute.String("content.type", string(params.ContentType)),
	))
	defer span.End()

	l := s.logger.With(slog.String("method", "AddListItem"),
		slog.String("listID", listID.String()),
		slog.String("userID", userID.String()),
		slog.String("itemID", params.ItemID.String()),
		slog.String("contentType", string(params.ContentType)))
	l.DebugContext(ctx, "Adding item to list")

	// Fetch the list to verify ownership
	list, err := s.listRepository.GetList(ctx, listID)
	if err != nil {
		l.ErrorContext(ctx, "Failed to fetch list", slog.Any("error", err))
		span.RecordError(err)
		span.SetStatus(codes.Error, "List not found")
		return nil, fmt.Errorf("list not found: %w", err)
	}

	// Verify ownership
	if list.UserID != userID {
		l.WarnContext(ctx, "User does not own list",
			slog.String("listOwnerID", list.UserID.String()))
		span.SetStatus(codes.Error, "User does not own list")
		return nil, fmt.Errorf("user does not own list")
	}

	// Create the list item with the new structure
	item := models.ListItem{
		ListID:                 listID,
		ItemID:                 params.ItemID,
		ContentType:            params.ContentType,
		Position:               params.Position,
		Notes:                  params.Notes,
		DayNumber:              params.DayNumber,
		TimeSlot:               params.TimeSlot,
		Duration:               params.DurationMinutes,
		SourceLlmInteractionID: params.SourceLlmInteractionID,
		ItemAIDescription:      params.ItemAIDescription,
		CreatedAt:              time.Now(),
		UpdatedAt:              time.Now(),
	}

	// Add the item to the list
	err = s.listRepository.AddListItem(ctx, item)
	if err != nil {
		l.ErrorContext(ctx, "Failed to add item to list", slog.Any("error", err))
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to add item to list")
		return nil, fmt.Errorf("failed to add item to list: %w", err)
	}

	l.InfoContext(ctx, "Item added to list successfully")
	span.SetStatus(codes.Ok, "Item added to list")
	return &item, nil
}

// UpdateListItem updates any type of content in a list
func (s *ServiceImpl) UpdateListItem(ctx context.Context, userID, listID, itemID uuid.UUID, params models.UpdateListItemRequest) (*models.ListItem, error) {
	ctx, span := otel.Tracer("ItineraryListService").Start(ctx, "UpdateListItem", trace.WithAttributes(
		attribute.String("list.id", listID.String()),
		attribute.String("user.id", userID.String()),
		attribute.String("item.id", itemID.String()),
	))
	defer span.End()

	l := s.logger.With(slog.String("method", "UpdateListItem"),
		slog.String("listID", listID.String()),
		slog.String("userID", userID.String()),
		slog.String("itemID", itemID.String()))
	l.DebugContext(ctx, "Updating item in list")

	// Fetch the list to verify ownership
	list, err := s.listRepository.GetList(ctx, listID)
	if err != nil {
		l.ErrorContext(ctx, "Failed to fetch list", slog.Any("error", err))
		span.RecordError(err)
		span.SetStatus(codes.Error, "List not found")
		return nil, fmt.Errorf("list not found: %w", err)
	}

	// Verify ownership
	if list.UserID != userID {
		l.WarnContext(ctx, "User does not own list",
			slog.String("listOwnerID", list.UserID.String()))
		span.SetStatus(codes.Error, "User does not own list")
		return nil, fmt.Errorf("user does not own list")
	}

	// Fetch the current item by generic item ID
	item, err := s.listRepository.GetListItemByID(ctx, listID, itemID)
	if err != nil {
		l.ErrorContext(ctx, "Failed to fetch list item", slog.Any("error", err))
		span.RecordError(err)
		span.SetStatus(codes.Error, "List item not found")
		return nil, fmt.Errorf("list item not found: %w", err)
	}

	// Update fields if provided
	if params.ItemID != nil {
		item.ItemID = *params.ItemID
	}
	if params.ContentType != nil {
		item.ContentType = *params.ContentType
	}
	if params.Position != nil {
		item.Position = *params.Position
	}
	if params.Notes != nil {
		item.Notes = *params.Notes
	}
	if params.DayNumber != nil {
		item.DayNumber = params.DayNumber
	}
	if params.TimeSlot != nil {
		item.TimeSlot = params.TimeSlot
	}
	if params.DurationMinutes != nil {
		item.Duration = params.DurationMinutes
	}
	if params.SourceLlmInteractionID != nil {
		item.SourceLlmInteractionID = params.SourceLlmInteractionID
	}
	if params.ItemAIDescription != nil {
		item.ItemAIDescription = *params.ItemAIDescription
	}
	item.UpdatedAt = time.Now()

	// Update the item in the repository
	err = s.listRepository.UpdateListItem(ctx, item)
	if err != nil {
		l.ErrorContext(ctx, "Failed to update list item", slog.Any("error", err))
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to update list item")
		return nil, fmt.Errorf("failed to update list item: %w", err)
	}

	l.InfoContext(ctx, "List item updated successfully")
	span.SetStatus(codes.Ok, "List item updated")
	return &item, nil
}

// RemoveListItem removes any type of content from a list
func (s *ServiceImpl) RemoveListItem(ctx context.Context, userID, listID, itemID uuid.UUID) error {
	ctx, span := otel.Tracer("ItineraryListService").Start(ctx, "RemoveListItem", trace.WithAttributes(
		attribute.String("list.id", listID.String()),
		attribute.String("user.id", userID.String()),
		attribute.String("item.id", itemID.String()),
	))
	defer span.End()

	l := s.logger.With(slog.String("method", "RemoveListItem"),
		slog.String("listID", listID.String()),
		slog.String("userID", userID.String()),
		slog.String("itemID", itemID.String()))
	l.DebugContext(ctx, "Removing item from list")

	// Fetch the list to verify ownership
	list, err := s.listRepository.GetList(ctx, listID)
	if err != nil {
		l.ErrorContext(ctx, "Failed to fetch list", slog.Any("error", err))
		span.RecordError(err)
		span.SetStatus(codes.Error, "List not found")
		return fmt.Errorf("list not found: %w", err)
	}

	// Verify ownership
	if list.UserID != userID {
		l.WarnContext(ctx, "User does not own list",
			slog.String("listOwnerID", list.UserID.String()))
		span.SetStatus(codes.Error, "User does not own list")
		return fmt.Errorf("user does not own list")
	}

	// Delete the item by generic item ID
	err = s.listRepository.DeleteListItemByID(ctx, listID, itemID)
	if err != nil {
		l.ErrorContext(ctx, "Failed to delete list item", slog.Any("error", err))
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to delete list item")
		return fmt.Errorf("failed to delete list item: %w", err)
	}

	l.InfoContext(ctx, "List item deleted successfully")
	span.SetStatus(codes.Ok, "List item deleted")
	return nil
}

// Legacy POI-specific methods (for backward compatibility)

// AddPOIListItem adds a POI to a list
func (s *ServiceImpl) AddPOIListItem(ctx context.Context, userID, listID, poiID uuid.UUID, params models.AddListItemRequest) (*models.ListItem, error) {
	ctx, span := otel.Tracer("ItineraryListService").Start(ctx, "AddPOIListItem", trace.WithAttributes(
		attribute.String("list.id", listID.String()),
		attribute.String("user.id", userID.String()),
		attribute.String("poi.id", poiID.String()),
	))
	defer span.End()

	l := s.logger.With(slog.String("method", "AddPOIListItem"),
		slog.String("listID", listID.String()),
		slog.String("userID", userID.String()),
		slog.String("poiID", poiID.String()))
	l.DebugContext(ctx, "Adding POI to list")

	// Fetch the list to verify ownership and check if it's an itinerary
	list, err := s.listRepository.GetList(ctx, listID)
	if err != nil {
		l.ErrorContext(ctx, "Failed to fetch list", slog.Any("error", err))
		span.RecordError(err)
		span.SetStatus(codes.Error, "List not found")
		return nil, fmt.Errorf("list not found: %w", err)
	}

	// Verify ownership
	if list.UserID != userID {
		l.WarnContext(ctx, "User does not own list",
			slog.String("listOwnerID", list.UserID.String()))
		span.SetStatus(codes.Error, "User does not own list")
		return nil, fmt.Errorf("user does not own list")
	}

	// Check if the list is an itinerary
	if !list.IsItinerary {
		l.WarnContext(ctx, "List is not an itinerary")
		span.SetStatus(codes.Error, "List is not an itinerary")
		return nil, fmt.Errorf("list is not an itinerary")
	}

	// Create the list item
	item := models.ListItem{
		ListID:    listID,
		ItemID:    poiID,
		Position:  params.Position,
		Notes:     params.Notes,
		DayNumber: params.DayNumber,
		TimeSlot:  params.TimeSlot,
		Duration:  params.DurationMinutes,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Add the item to the list
	err = s.listRepository.AddListItem(ctx, item)
	if err != nil {
		l.ErrorContext(ctx, "Failed to add POI to list", slog.Any("error", err))
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to add POI to list")
		return nil, fmt.Errorf("failed to add POI to list: %w", err)
	}

	l.InfoContext(ctx, "POI added to list successfully")
	span.SetStatus(codes.Ok, "POI added to list")
	return &item, nil
}

// UpdatePOIListItem updates a POI in a list
func (s *ServiceImpl) UpdatePOIListItem(ctx context.Context, userID, listID, poiID uuid.UUID, params models.UpdateListItemRequest) (*models.ListItem, error) {
	ctx, span := otel.Tracer("ItineraryListService").Start(ctx, "UpdatePOIListItem", trace.WithAttributes(
		attribute.String("list.id", listID.String()),
		attribute.String("user.id", userID.String()),
		attribute.String("poi.id", poiID.String()),
	))
	defer span.End()

	l := s.logger.With(slog.String("method", "UpdatePOIListItem"),
		slog.String("listID", listID.String()),
		slog.String("userID", userID.String()),
		slog.String("poiID", poiID.String()))
	l.DebugContext(ctx, "Updating POI in list")

	// Fetch the list to verify ownership
	list, err := s.listRepository.GetList(ctx, listID)
	if err != nil {
		l.ErrorContext(ctx, "Failed to fetch list", slog.Any("error", err))
		span.RecordError(err)
		span.SetStatus(codes.Error, "List not found")
		return nil, fmt.Errorf("list not found: %w", err)
	}

	// Verify ownership
	if list.UserID != userID {
		l.WarnContext(ctx, "User does not own list",
			slog.String("listOwnerID", list.UserID.String()))
		span.SetStatus(codes.Error, "User does not own list")
		return nil, fmt.Errorf("user does not own list")
	}

	// Fetch the current item
	// Note: We need to add a GetListItem method to the repository
	item, err := s.listRepository.GetListItem(ctx, listID, poiID, "poi")
	if err != nil {
		l.ErrorContext(ctx, "Failed to fetch list item", slog.Any("error", err))
		span.RecordError(err)
		span.SetStatus(codes.Error, "List item not found")
		return nil, fmt.Errorf("list item not found: %w", err)
	}

	// Update fields if provided
	if params.Position != nil {
		item.Position = *params.Position
	}
	if params.Notes != nil {
		item.Notes = *params.Notes
	}
	if params.DayNumber != nil {
		item.DayNumber = params.DayNumber
	}
	if params.TimeSlot != nil {
		item.TimeSlot = params.TimeSlot
	}
	if params.DurationMinutes != nil {
		item.Duration = params.DurationMinutes
	}
	item.UpdatedAt = time.Now()

	// Update the item in the repository
	// Note: We need to add an UpdateListItem method to the repository
	err = s.listRepository.UpdateListItem(ctx, item)
	if err != nil {
		l.ErrorContext(ctx, "Failed to update list item", slog.Any("error", err))
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to update list item")
		return nil, fmt.Errorf("failed to update list item: %w", err)
	}

	l.InfoContext(ctx, "List item updated successfully")
	span.SetStatus(codes.Ok, "List item updated")
	return &item, nil
}

// RemovePOIListItem removes a POI from a list
func (s *ServiceImpl) RemovePOIListItem(ctx context.Context, userID, listID, poiID uuid.UUID) error {
	ctx, span := otel.Tracer("ItineraryListService").Start(ctx, "RemovePOIListItem", trace.WithAttributes(
		attribute.String("list.id", listID.String()),
		attribute.String("user.id", userID.String()),
		attribute.String("poi.id", poiID.String()),
	))
	defer span.End()

	l := s.logger.With(slog.String("method", "RemovePOIListItem"),
		slog.String("listID", listID.String()),
		slog.String("userID", userID.String()),
		slog.String("poiID", poiID.String()))
	l.DebugContext(ctx, "Removing POI from list")

	// Fetch the list to verify ownership
	list, err := s.listRepository.GetList(ctx, listID)
	if err != nil {
		l.ErrorContext(ctx, "Failed to fetch list", slog.Any("error", err))
		span.RecordError(err)
		span.SetStatus(codes.Error, "List not found")
		return fmt.Errorf("list not found: %w", err)
	}

	// Verify ownership
	if list.UserID != userID {
		l.WarnContext(ctx, "User does not own list",
			slog.String("listOwnerID", list.UserID.String()))
		span.SetStatus(codes.Error, "User does not own list")
		return fmt.Errorf("user does not own list")
	}

	// Delete the item
	err = s.listRepository.DeleteListItem(ctx, listID, poiID, "poi")
	if err != nil {
		l.ErrorContext(ctx, "Failed to delete list item", slog.Any("error", err))
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to delete list item")
		return fmt.Errorf("failed to delete list item: %w", err)
	}

	l.InfoContext(ctx, "List item deleted successfully")
	span.SetStatus(codes.Ok, "List item deleted")
	return nil
}

// GetUserLists retrieves all lists for a user
func (s *ServiceImpl) GetUserLists(ctx context.Context, userID uuid.UUID, isItinerary bool) ([]*models.List, error) {
	ctx, span := otel.Tracer("ItineraryListService").Start(ctx, "GetUserLists", trace.WithAttributes(
		attribute.String("user.id", userID.String()),
		attribute.Bool("is_itinerary", isItinerary),
	))
	defer span.End()

	l := s.logger.With(slog.String("method", "GetUserLists"),
		slog.String("userID", userID.String()),
		slog.Bool("isItinerary", isItinerary))
	l.DebugContext(ctx, "Getting user lists")

	// Note: We need to add a GetUserLists method to the repository
	lists, err := s.listRepository.GetUserLists(ctx, userID, isItinerary)
	if err != nil {
		l.ErrorContext(ctx, "Failed to get user lists", slog.Any("error", err))
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to get user lists")
		return nil, fmt.Errorf("failed to get user lists: %w", err)
	}

	l.InfoContext(ctx, "User lists fetched successfully", slog.Int("count", len(lists)))
	span.SetStatus(codes.Ok, "User lists fetched")
	return lists, nil
}

func (s *ServiceImpl) SaveList(ctx context.Context, userID, listID uuid.UUID) error {
	ctx, span := otel.Tracer("ItineraryListService").Start(ctx, "SaveList", trace.WithAttributes(
		attribute.String("user.id", userID.String()),
		attribute.String("list.id", listID.String()),
	))
	defer span.End()

	l := s.logger.With(slog.String("method", "SaveList"),
		slog.String("userID", userID.String()),
		slog.String("listID", listID.String()))
	l.DebugContext(ctx, "Saving list for user")

	// Verify the list exists and is public (or belongs to user)
	list, err := s.listRepository.GetList(ctx, listID)
	if err != nil {
		l.ErrorContext(ctx, "Failed to get list", slog.Any("error", err))
		span.RecordError(err)
		span.SetStatus(codes.Error, "List not found")
		return fmt.Errorf("list not found: %w", err)
	}

	// User cannot save their own list
	if list.UserID == userID {
		l.WarnContext(ctx, "User cannot save their own list")
		span.SetStatus(codes.Error, "Cannot save own list")
		return fmt.Errorf("cannot save your own list")
	}

	// List must be public to be saved by others
	if !list.IsPublic {
		l.WarnContext(ctx, "Cannot save private list")
		span.SetStatus(codes.Error, "Cannot save private list")
		return fmt.Errorf("cannot save private list")
	}

	// Save the list
	err = s.listRepository.SaveList(ctx, userID, listID)
	if err != nil {
		l.ErrorContext(ctx, "Failed to save list", slog.Any("error", err))
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to save list")
		return fmt.Errorf("failed to save list: %w", err)
	}

	l.InfoContext(ctx, "List saved successfully")
	span.SetStatus(codes.Ok, "List saved")
	return nil
}

func (s *ServiceImpl) UnsaveList(ctx context.Context, userID, listID uuid.UUID) error {
	ctx, span := otel.Tracer("ItineraryListService").Start(ctx, "UnsaveList", trace.WithAttributes(
		attribute.String("user.id", userID.String()),
		attribute.String("list.id", listID.String()),
	))
	defer span.End()

	l := s.logger.With(slog.String("method", "UnsaveList"),
		slog.String("userID", userID.String()),
		slog.String("listID", listID.String()))
	l.DebugContext(ctx, "Unsaving list for user")

	// Unsave the list
	err := s.listRepository.UnsaveList(ctx, userID, listID)
	if err != nil {
		l.ErrorContext(ctx, "Failed to unsave list", slog.Any("error", err))
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to unsave list")
		return fmt.Errorf("failed to unsave list: %w", err)
	}

	l.InfoContext(ctx, "List unsaved successfully")
	span.SetStatus(codes.Ok, "List unsaved")
	return nil
}

func (s *ServiceImpl) GetUserSavedLists(ctx context.Context, userID uuid.UUID) ([]*models.List, error) {
	ctx, span := otel.Tracer("ItineraryListService").Start(ctx, "GetUserSavedLists", trace.WithAttributes(
		attribute.String("user.id", userID.String()),
	))
	defer span.End()

	l := s.logger.With(slog.String("method", "GetUserSavedLists"),
		slog.String("userID", userID.String()))
	l.DebugContext(ctx, "Getting user saved lists")

	// Get saved lists from repository
	lists, err := s.listRepository.GetUserSavedLists(ctx, userID)
	if err != nil {
		l.ErrorContext(ctx, "Failed to get user saved lists", slog.Any("error", err))
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to get saved lists")
		return nil, fmt.Errorf("failed to get saved lists: %w", err)
	}

	l.InfoContext(ctx, "User saved lists fetched successfully", slog.Int("count", len(lists)))
	span.SetStatus(codes.Ok, "Saved lists fetched")
	return lists, nil
}

func (s *ServiceImpl) GetListItemsByContentType(ctx context.Context, userID, listID uuid.UUID, contentType models.ContentType) ([]*models.ListItem, error) {
	ctx, span := otel.Tracer("ItineraryListService").Start(ctx, "GetListItemsByContentType", trace.WithAttributes(
		attribute.String("user.id", userID.String()),
		attribute.String("list.id", listID.String()),
		attribute.String("content.type", string(contentType)),
	))
	defer span.End()

	l := s.logger.With(slog.String("method", "GetListItemsByContentType"),
		slog.String("userID", userID.String()),
		slog.String("listID", listID.String()),
		slog.String("contentType", string(contentType)))
	l.DebugContext(ctx, "Getting list items by content type")

	// Verify user has access to the list
	list, err := s.listRepository.GetList(ctx, listID)
	if err != nil {
		l.ErrorContext(ctx, "Failed to get list", slog.Any("error", err))
		span.RecordError(err)
		span.SetStatus(codes.Error, "List not found")
		return nil, fmt.Errorf("list not found: %w", err)
	}

	// Check if user has access (owner or public list)
	if list.UserID != userID && !list.IsPublic {
		l.WarnContext(ctx, "Access denied to list",
			slog.String("listOwnerID", list.UserID.String()))
		span.SetStatus(codes.Error, "Access denied")
		return nil, fmt.Errorf("access denied to list")
	}

	// Get items by content type
	items, err := s.listRepository.GetListItemsByContentType(ctx, listID, contentType)
	if err != nil {
		l.ErrorContext(ctx, "Failed to get list items by content type", slog.Any("error", err))
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to get list items")
		return nil, fmt.Errorf("failed to get list items: %w", err)
	}

	l.InfoContext(ctx, "List items by content type fetched successfully", slog.Int("count", len(items)))
	span.SetStatus(codes.Ok, "List items fetched")
	return items, nil
}

func (s *ServiceImpl) SearchLists(ctx context.Context, searchTerm, contentType string, cityID *uuid.UUID) ([]*models.List, error) {
	ctx, span := otel.Tracer("ItineraryListService").Start(ctx, "SearchLists", trace.WithAttributes(
		attribute.String("search.term", searchTerm),
		attribute.String("content.type", contentType),
	))
	defer span.End()

	l := s.logger.With(slog.String("method", "SearchLists"),
		slog.String("searchTerm", searchTerm),
		slog.String("contentType", contentType))
	l.DebugContext(ctx, "Searching lists")

	if cityID != nil {
		span.SetAttributes(attribute.String("city.id", cityID.String()))
		l = l.With(slog.String("cityID", cityID.String()))
	}

	// Search lists using repository
	lists, err := s.listRepository.SearchLists(ctx, searchTerm, contentType, cityID)
	if err != nil {
		l.ErrorContext(ctx, "Failed to search lists", slog.Any("error", err))
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to search lists")
		return nil, fmt.Errorf("failed to search lists: %w", err)
	}

	l.InfoContext(ctx, "Lists search completed successfully", slog.Int("resultCount", len(lists)))
	span.SetStatus(codes.Ok, "Lists search completed")
	return lists, nil
}
