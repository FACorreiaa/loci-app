package lists

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/FACorreiaa/go-templui/internal/app/models"
)

var _ Service = (*ServiceImpl)(nil)

type Service interface {
	CreateTopLevelList(ctx context.Context, userID uuid.UUID, name, description string, cityID *uuid.UUID, isItinerary, isPublic bool) (*models.List, error)
	CreateItineraryForList(ctx context.Context, userID, parentListID uuid.UUID, name, description string, isPublic bool) (*models.List, error)
	GetListDetails(ctx context.Context, listID, userID uuid.UUID) (*models.ListWithItems, error)
	UpdateListDetails(ctx context.Context, listID, userID uuid.UUID, params models.UpdateListRequest) (*models.List, error)
	DeleteUserList(ctx context.Context, listID, userID uuid.UUID) error

	// Generic lists item methods (support all content types)
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
	logger         *zap.Logger
	listRepository Repository
}

// NewService creates a new instance of ServiceImpl
func NewService(repo Repository, logger *zap.Logger) *ServiceImpl {
	return &ServiceImpl{
		logger:         logger,
		listRepository: repo,
	}
}

// CreateTopLevelList creates a new top-level lists
func (s *ServiceImpl) CreateTopLevelList(ctx context.Context, userID uuid.UUID, name, description string, cityID *uuid.UUID, isItinerary, isPublic bool) (*models.List, error) {
	ctx, span := otel.Tracer("ItineraryListService").Start(ctx, "CreateTopLevelList", trace.WithAttributes(
		attribute.String("user.id", userID.String()),
		attribute.String("lists.name", name),
		attribute.Bool("lists.is_itinerary", isItinerary),
	))
	defer span.End()

	l := s.logger.With(zap.String("method", "CreateTopLevelList"), zap.String("userID", userID.String()))
	l.Debug("Creating top-level lists")

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
		l.Error("Failed to create top-level lists", zap.Any("error", err))
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to create lists")
		return nil, fmt.Errorf("failed to create lists: %w", err)
	}

	l.Info("Top-level lists created successfully", zap.String("listID", list.ID.String()))
	span.SetStatus(codes.Ok, "List created")
	return &list, nil
}

// CreateItineraryForList creates a new itinerary within a parent lists
func (s *ServiceImpl) CreateItineraryForList(ctx context.Context, userID, parentListID uuid.UUID, name, description string, isPublic bool) (*models.List, error) {
	ctx, span := otel.Tracer("ItineraryListService").Start(ctx, "CreateItineraryForList", trace.WithAttributes(
		attribute.String("user.id", userID.String()),
		attribute.String("parent_list.id", parentListID.String()),
		attribute.String("itinerary.name", name),
	))
	defer span.End()

	l := s.logger.With(zap.String("method", "CreateItineraryForList"),
		zap.String("userID", userID.String()),
		zap.String("parentListID", parentListID.String()))
	l.Debug("Creating itinerary for lists")

	// Fetch parent lists to verify ownership and inherit cityID
	parentList, err := s.listRepository.GetList(ctx, parentListID)
	if err != nil {
		l.Error("Failed to fetch parent lists", zap.Any("error", err))
		span.RecordError(err)
		span.SetStatus(codes.Error, "Parent lists not found")
		return nil, fmt.Errorf("parent lists not found: %w", err)
	}

	// Verify ownership
	if parentList.UserID != userID {
		l.Warn("User does not own parent lists",
			zap.String("listOwnerID", parentList.UserID.String()))
		span.SetStatus(codes.Error, "User does not own parent lists")
		return nil, fmt.Errorf("user does not own parent lists")
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
		l.Error("Failed to create itinerary", zap.Any("error", err))
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to create itinerary")
		return nil, fmt.Errorf("failed to create itinerary: %w", err)
	}

	l.Info("Itinerary created successfully", zap.String("itineraryID", itinerary.ID.String()))
	span.SetStatus(codes.Ok, "Itinerary created")
	return &itinerary, nil
}

// GetListDetails retrieves a lists with all its items
func (s *ServiceImpl) GetListDetails(ctx context.Context, listID, userID uuid.UUID) (*models.ListWithItems, error) {
	ctx, span := otel.Tracer("ItineraryListService").Start(ctx, "GetListDetails", trace.WithAttributes(
		attribute.String("lists.id", listID.String()),
		attribute.String("user.id", userID.String()),
	))
	defer span.End()

	l := s.logger.With(zap.String("method", "GetListDetails"),
		zap.String("listID", listID.String()),
		zap.String("userID", userID.String()))
	l.Debug("Getting lists details")

	// Fetch the lists
	list, err := s.listRepository.GetList(ctx, listID)
	if err != nil {
		l.Error("Failed to fetch lists", zap.Any("error", err))
		span.RecordError(err)
		span.SetStatus(codes.Error, "List not found")
		return nil, fmt.Errorf("lists not found: %w", err)
	}

	// Check if user has access (owner or public lists)
	if list.UserID != userID && !list.IsPublic {
		l.Warn("Access denied to lists",
			zap.String("listOwnerID", list.UserID.String()))
		span.SetStatus(codes.Error, "Access denied")
		return nil, fmt.Errorf("access denied to lists")
	}

	// Fetch lists items if it's an itinerary
	var items []*models.ListItem
	if list.IsItinerary {
		items, err = s.listRepository.GetListItems(ctx, listID)
		if err != nil {
			l.Error("Failed to fetch lists items", zap.Any("error", err))
			span.RecordError(err)
			span.SetStatus(codes.Error, "Failed to fetch lists items")
			return nil, fmt.Errorf("failed to fetch lists items: %w", err)
		}
	}

	result := &models.ListWithItems{
		List:  list,
		Items: items,
	}

	l.Info("List details fetched successfully",
		zap.Int("itemCount", len(items)))
	span.SetStatus(codes.Ok, "List details fetched")
	return result, nil
}

// UpdateListDetails updates a lists's details
func (s *ServiceImpl) UpdateListDetails(ctx context.Context, listID, userID uuid.UUID, params models.UpdateListRequest) (*models.List, error) {
	ctx, span := otel.Tracer("ItineraryListService").Start(ctx, "UpdateListDetails", trace.WithAttributes(
		attribute.String("lists.id", listID.String()),
		attribute.String("user.id", userID.String()),
	))
	defer span.End()

	l := s.logger.With(zap.String("method", "UpdateListDetails"),
		zap.String("listID", listID.String()),
		zap.String("userID", userID.String()))
	l.Debug("Updating lists details")

	// Fetch the lists to verify ownership
	list, err := s.listRepository.GetList(ctx, listID)
	if err != nil {
		l.Error("Failed to fetch lists", zap.Any("error", err))
		span.RecordError(err)
		span.SetStatus(codes.Error, "List not found")
		return nil, fmt.Errorf("lists not found: %w", err)
	}

	// Verify ownership
	if list.UserID != userID {
		l.Warn("User does not own lists",
			zap.String("listOwnerID", list.UserID.String()))
		span.SetStatus(codes.Error, "User does not own lists")
		return nil, fmt.Errorf("user does not own lists")
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

	// Update the lists in the repository
	// Note: We need to add an UpdateList method to the repository
	err = s.listRepository.UpdateList(ctx, list)
	if err != nil {
		l.Error("Failed to update lists", zap.Any("error", err))
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to update lists")
		return nil, fmt.Errorf("failed to update lists: %w", err)
	}

	l.Info("List updated successfully")
	span.SetStatus(codes.Ok, "List updated")
	return &list, nil
}

// DeleteUserList deletes a lists
func (s *ServiceImpl) DeleteUserList(ctx context.Context, listID, userID uuid.UUID) error {
	ctx, span := otel.Tracer("ItineraryListService").Start(ctx, "DeleteUserList", trace.WithAttributes(
		attribute.String("lists.id", listID.String()),
		attribute.String("user.id", userID.String()),
	))
	defer span.End()

	l := s.logger.With(zap.String("method", "DeleteUserList"),
		zap.String("listID", listID.String()),
		zap.String("userID", userID.String()))
	l.Debug("Deleting lists")

	// Fetch the lists to verify ownership
	list, err := s.listRepository.GetList(ctx, listID)
	if err != nil {
		l.Error("Failed to fetch lists", zap.Any("error", err))
		span.RecordError(err)
		span.SetStatus(codes.Error, "List not found")
		return fmt.Errorf("lists not found: %w", err)
	}

	// Verify ownership
	if list.UserID != userID {
		l.Warn("User does not own lists",
			zap.String("listOwnerID", list.UserID.String()))
		span.SetStatus(codes.Error, "User does not own lists")
		return fmt.Errorf("user does not own lists")
	}

	// Delete the lists
	err = s.listRepository.DeleteList(ctx, listID)
	if err != nil {
		l.Error("Failed to delete lists", zap.Any("error", err))
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to delete lists")
		return fmt.Errorf("failed to delete lists: %w", err)
	}

	l.Info("List deleted successfully")
	span.SetStatus(codes.Ok, "List deleted")
	return nil
}

// Generic lists item methods (support all content types)

// AddListItem adds any type of content to a lists
func (s *ServiceImpl) AddListItem(ctx context.Context, userID, listID uuid.UUID, params models.AddListItemRequest) (*models.ListItem, error) {
	ctx, span := otel.Tracer("ItineraryListService").Start(ctx, "AddListItem", trace.WithAttributes(
		attribute.String("lists.id", listID.String()),
		attribute.String("user.id", userID.String()),
		attribute.String("item.id", params.ItemID.String()),
		attribute.String("content.type", string(params.ContentType)),
	))
	defer span.End()

	l := s.logger.With(zap.String("method", "AddListItem"),
		zap.String("listID", listID.String()),
		zap.String("userID", userID.String()),
		zap.String("itemID", params.ItemID.String()),
		zap.String("contentType", string(params.ContentType)))
	l.Debug("Adding item to lists")

	// Fetch the lists to verify ownership
	list, err := s.listRepository.GetList(ctx, listID)
	if err != nil {
		l.Error("Failed to fetch lists", zap.Any("error", err))
		span.RecordError(err)
		span.SetStatus(codes.Error, "List not found")
		return nil, fmt.Errorf("lists not found: %w", err)
	}

	// Verify ownership
	if list.UserID != userID {
		l.Warn("User does not own lists",
			zap.String("listOwnerID", list.UserID.String()))
		span.SetStatus(codes.Error, "User does not own lists")
		return nil, fmt.Errorf("user does not own lists")
	}

	// Create the lists item with the new structure
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

	// Add the item to the lists
	err = s.listRepository.AddListItem(ctx, item)
	if err != nil {
		l.Error("Failed to add item to lists", zap.Any("error", err))
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to add item to lists")
		return nil, fmt.Errorf("failed to add item to lists: %w", err)
	}

	l.Info("Item added to lists successfully")
	span.SetStatus(codes.Ok, "Item added to lists")
	return &item, nil
}

// UpdateListItem updates any type of content in a lists
func (s *ServiceImpl) UpdateListItem(ctx context.Context, userID, listID, itemID uuid.UUID, params models.UpdateListItemRequest) (*models.ListItem, error) {
	ctx, span := otel.Tracer("ItineraryListService").Start(ctx, "UpdateListItem", trace.WithAttributes(
		attribute.String("lists.id", listID.String()),
		attribute.String("user.id", userID.String()),
		attribute.String("item.id", itemID.String()),
	))
	defer span.End()

	l := s.logger.With(zap.String("method", "UpdateListItem"),
		zap.String("listID", listID.String()),
		zap.String("userID", userID.String()),
		zap.String("itemID", itemID.String()))
	l.Debug("Updating item in lists")

	// Fetch the lists to verify ownership
	list, err := s.listRepository.GetList(ctx, listID)
	if err != nil {
		l.Error("Failed to fetch lists", zap.Any("error", err))
		span.RecordError(err)
		span.SetStatus(codes.Error, "List not found")
		return nil, fmt.Errorf("lists not found: %w", err)
	}

	// Verify ownership
	if list.UserID != userID {
		l.Warn("User does not own lists",
			zap.String("listOwnerID", list.UserID.String()))
		span.SetStatus(codes.Error, "User does not own lists")
		return nil, fmt.Errorf("user does not own lists")
	}

	// Fetch the current item by generic item ID
	item, err := s.listRepository.GetListItemByID(ctx, listID, itemID)
	if err != nil {
		l.Error("Failed to fetch lists item", zap.Any("error", err))
		span.RecordError(err)
		span.SetStatus(codes.Error, "List item not found")
		return nil, fmt.Errorf("lists item not found: %w", err)
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
		l.Error("Failed to update lists item", zap.Any("error", err))
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to update lists item")
		return nil, fmt.Errorf("failed to update lists item: %w", err)
	}

	l.Info("List item updated successfully")
	span.SetStatus(codes.Ok, "List item updated")
	return &item, nil
}

// RemoveListItem removes any type of content from a lists
func (s *ServiceImpl) RemoveListItem(ctx context.Context, userID, listID, itemID uuid.UUID) error {
	ctx, span := otel.Tracer("ItineraryListService").Start(ctx, "RemoveListItem", trace.WithAttributes(
		attribute.String("lists.id", listID.String()),
		attribute.String("user.id", userID.String()),
		attribute.String("item.id", itemID.String()),
	))
	defer span.End()

	l := s.logger.With(zap.String("method", "RemoveListItem"),
		zap.String("listID", listID.String()),
		zap.String("userID", userID.String()),
		zap.String("itemID", itemID.String()))
	l.Debug("Removing item from lists")

	// Fetch the lists to verify ownership
	list, err := s.listRepository.GetList(ctx, listID)
	if err != nil {
		l.Error("Failed to fetch lists", zap.Any("error", err))
		span.RecordError(err)
		span.SetStatus(codes.Error, "List not found")
		return fmt.Errorf("lists not found: %w", err)
	}

	// Verify ownership
	if list.UserID != userID {
		l.Warn("User does not own lists",
			zap.String("listOwnerID", list.UserID.String()))
		span.SetStatus(codes.Error, "User does not own lists")
		return fmt.Errorf("user does not own lists")
	}

	// Delete the item by generic item ID
	err = s.listRepository.DeleteListItemByID(ctx, listID, itemID)
	if err != nil {
		l.Error("Failed to delete lists item", zap.Any("error", err))
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to delete lists item")
		return fmt.Errorf("failed to delete lists item: %w", err)
	}

	l.Info("List item deleted successfully")
	span.SetStatus(codes.Ok, "List item deleted")
	return nil
}

// Legacy POI-specific methods (for backward compatibility)

// AddPOIListItem adds a POI to a lists
func (s *ServiceImpl) AddPOIListItem(ctx context.Context, userID, listID, poiID uuid.UUID, params models.AddListItemRequest) (*models.ListItem, error) {
	ctx, span := otel.Tracer("ItineraryListService").Start(ctx, "AddPOIListItem", trace.WithAttributes(
		attribute.String("lists.id", listID.String()),
		attribute.String("user.id", userID.String()),
		attribute.String("poi.id", poiID.String()),
	))
	defer span.End()

	l := s.logger.With(zap.String("method", "AddPOIListItem"),
		zap.String("listID", listID.String()),
		zap.String("userID", userID.String()),
		zap.String("poiID", poiID.String()))
	l.Debug("Adding POI to lists")

	// Fetch the lists to verify ownership and check if it's an itinerary
	list, err := s.listRepository.GetList(ctx, listID)
	if err != nil {
		l.Error("Failed to fetch lists", zap.Any("error", err))
		span.RecordError(err)
		span.SetStatus(codes.Error, "List not found")
		return nil, fmt.Errorf("lists not found: %w", err)
	}

	// Verify ownership
	if list.UserID != userID {
		l.Warn("User does not own lists",
			zap.String("listOwnerID", list.UserID.String()))
		span.SetStatus(codes.Error, "User does not own lists")
		return nil, fmt.Errorf("user does not own lists")
	}

	// Check if the lists is an itinerary
	if !list.IsItinerary {
		l.Warn("List is not an itinerary")
		span.SetStatus(codes.Error, "List is not an itinerary")
		return nil, fmt.Errorf("lists is not an itinerary")
	}

	// Create the lists item
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

	// Add the item to the lists
	err = s.listRepository.AddListItem(ctx, item)
	if err != nil {
		l.Error("Failed to add POI to lists", zap.Any("error", err))
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to add POI to lists")
		return nil, fmt.Errorf("failed to add POI to lists: %w", err)
	}

	l.Info("POI added to lists successfully")
	span.SetStatus(codes.Ok, "POI added to lists")
	return &item, nil
}

// UpdatePOIListItem updates a POI in a lists
func (s *ServiceImpl) UpdatePOIListItem(ctx context.Context, userID, listID, poiID uuid.UUID, params models.UpdateListItemRequest) (*models.ListItem, error) {
	ctx, span := otel.Tracer("ItineraryListService").Start(ctx, "UpdatePOIListItem", trace.WithAttributes(
		attribute.String("lists.id", listID.String()),
		attribute.String("user.id", userID.String()),
		attribute.String("poi.id", poiID.String()),
	))
	defer span.End()

	l := s.logger.With(zap.String("method", "UpdatePOIListItem"),
		zap.String("listID", listID.String()),
		zap.String("userID", userID.String()),
		zap.String("poiID", poiID.String()))
	l.Debug("Updating POI in lists")

	// Fetch the lists to verify ownership
	list, err := s.listRepository.GetList(ctx, listID)
	if err != nil {
		l.Error("Failed to fetch lists", zap.Any("error", err))
		span.RecordError(err)
		span.SetStatus(codes.Error, "List not found")
		return nil, fmt.Errorf("lists not found: %w", err)
	}

	// Verify ownership
	if list.UserID != userID {
		l.Warn("User does not own lists",
			zap.String("listOwnerID", list.UserID.String()))
		span.SetStatus(codes.Error, "User does not own lists")
		return nil, fmt.Errorf("user does not own lists")
	}

	// Fetch the current item
	// Note: We need to add a GetListItem method to the repository
	item, err := s.listRepository.GetListItem(ctx, listID, poiID, "poi")
	if err != nil {
		l.Error("Failed to fetch lists item", zap.Any("error", err))
		span.RecordError(err)
		span.SetStatus(codes.Error, "List item not found")
		return nil, fmt.Errorf("lists item not found: %w", err)
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
		l.Error("Failed to update lists item", zap.Any("error", err))
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to update lists item")
		return nil, fmt.Errorf("failed to update lists item: %w", err)
	}

	l.Info("List item updated successfully")
	span.SetStatus(codes.Ok, "List item updated")
	return &item, nil
}

// RemovePOIListItem removes a POI from a lists
func (s *ServiceImpl) RemovePOIListItem(ctx context.Context, userID, listID, poiID uuid.UUID) error {
	ctx, span := otel.Tracer("ItineraryListService").Start(ctx, "RemovePOIListItem", trace.WithAttributes(
		attribute.String("lists.id", listID.String()),
		attribute.String("user.id", userID.String()),
		attribute.String("poi.id", poiID.String()),
	))
	defer span.End()

	l := s.logger.With(zap.String("method", "RemovePOIListItem"),
		zap.String("listID", listID.String()),
		zap.String("userID", userID.String()),
		zap.String("poiID", poiID.String()))
	l.Debug("Removing POI from lists")

	// Fetch the lists to verify ownership
	list, err := s.listRepository.GetList(ctx, listID)
	if err != nil {
		l.Error("Failed to fetch lists", zap.Any("error", err))
		span.RecordError(err)
		span.SetStatus(codes.Error, "List not found")
		return fmt.Errorf("lists not found: %w", err)
	}

	// Verify ownership
	if list.UserID != userID {
		l.Warn("User does not own lists",
			zap.String("listOwnerID", list.UserID.String()))
		span.SetStatus(codes.Error, "User does not own lists")
		return fmt.Errorf("user does not own lists")
	}

	// Delete the item
	err = s.listRepository.DeleteListItem(ctx, listID, poiID, "poi")
	if err != nil {
		l.Error("Failed to delete lists item", zap.Any("error", err))
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to delete lists item")
		return fmt.Errorf("failed to delete lists item: %w", err)
	}

	l.Info("List item deleted successfully")
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

	l := s.logger.With(zap.String("method", "GetUserLists"),
		zap.String("userID", userID.String()),
		zap.Bool("isItinerary", isItinerary))
	l.Debug("Getting user lists")

	// Note: We need to add a GetUserLists method to the repository
	lists, err := s.listRepository.GetUserLists(ctx, userID, isItinerary)
	if err != nil {
		l.Error("Failed to get user lists", zap.Any("error", err))
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to get user lists")
		return nil, fmt.Errorf("failed to get user lists: %w", err)
	}

	l.Info("User lists fetched successfully", zap.Int("count", len(lists)))
	span.SetStatus(codes.Ok, "User lists fetched")
	return lists, nil
}

func (s *ServiceImpl) SaveList(ctx context.Context, userID, listID uuid.UUID) error {
	ctx, span := otel.Tracer("ItineraryListService").Start(ctx, "SaveList", trace.WithAttributes(
		attribute.String("user.id", userID.String()),
		attribute.String("lists.id", listID.String()),
	))
	defer span.End()

	l := s.logger.With(zap.String("method", "SaveList"),
		zap.String("userID", userID.String()),
		zap.String("listID", listID.String()))
	l.Debug("Saving lists for user")

	// Verify the lists exists and is public (or belongs to user)
	list, err := s.listRepository.GetList(ctx, listID)
	if err != nil {
		l.Error("Failed to get lists", zap.Any("error", err))
		span.RecordError(err)
		span.SetStatus(codes.Error, "List not found")
		return fmt.Errorf("lists not found: %w", err)
	}

	// User cannot save their own lists
	if list.UserID == userID {
		l.Warn("User cannot save their own lists")
		span.SetStatus(codes.Error, "Cannot save own lists")
		return fmt.Errorf("cannot save your own lists")
	}

	// List must be public to be saved by others
	if !list.IsPublic {
		l.Warn("Cannot save private lists")
		span.SetStatus(codes.Error, "Cannot save private lists")
		return fmt.Errorf("cannot save private lists")
	}

	// Save the lists
	err = s.listRepository.SaveList(ctx, userID, listID)
	if err != nil {
		l.Error("Failed to save lists", zap.Any("error", err))
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to save lists")
		return fmt.Errorf("failed to save lists: %w", err)
	}

	l.Info("List saved successfully")
	span.SetStatus(codes.Ok, "List saved")
	return nil
}

func (s *ServiceImpl) UnsaveList(ctx context.Context, userID, listID uuid.UUID) error {
	ctx, span := otel.Tracer("ItineraryListService").Start(ctx, "UnsaveList", trace.WithAttributes(
		attribute.String("user.id", userID.String()),
		attribute.String("lists.id", listID.String()),
	))
	defer span.End()

	l := s.logger.With(zap.String("method", "UnsaveList"),
		zap.String("userID", userID.String()),
		zap.String("listID", listID.String()))
	l.Debug("Unsaving lists for user")

	// Unsave the lists
	err := s.listRepository.UnsaveList(ctx, userID, listID)
	if err != nil {
		l.Error("Failed to unsave lists", zap.Any("error", err))
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to unsave lists")
		return fmt.Errorf("failed to unsave lists: %w", err)
	}

	l.Info("List unsaved successfully")
	span.SetStatus(codes.Ok, "List unsaved")
	return nil
}

func (s *ServiceImpl) GetUserSavedLists(ctx context.Context, userID uuid.UUID) ([]*models.List, error) {
	ctx, span := otel.Tracer("ItineraryListService").Start(ctx, "GetUserSavedLists", trace.WithAttributes(
		attribute.String("user.id", userID.String()),
	))
	defer span.End()

	l := s.logger.With(zap.String("method", "GetUserSavedLists"),
		zap.String("userID", userID.String()))
	l.Debug("Getting user saved lists")

	// Get saved lists from repository
	lists, err := s.listRepository.GetUserSavedLists(ctx, userID)
	if err != nil {
		l.Error("Failed to get user saved lists", zap.Any("error", err))
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to get saved lists")
		return nil, fmt.Errorf("failed to get saved lists: %w", err)
	}

	l.Info("User saved lists fetched successfully", zap.Int("count", len(lists)))
	span.SetStatus(codes.Ok, "Saved lists fetched")
	return lists, nil
}

func (s *ServiceImpl) GetListItemsByContentType(ctx context.Context, userID, listID uuid.UUID, contentType models.ContentType) ([]*models.ListItem, error) {
	ctx, span := otel.Tracer("ItineraryListService").Start(ctx, "GetListItemsByContentType", trace.WithAttributes(
		attribute.String("user.id", userID.String()),
		attribute.String("lists.id", listID.String()),
		attribute.String("content.type", string(contentType)),
	))
	defer span.End()

	l := s.logger.With(zap.String("method", "GetListItemsByContentType"),
		zap.String("userID", userID.String()),
		zap.String("listID", listID.String()),
		zap.String("contentType", string(contentType)))
	l.Debug("Getting lists items by content type")

	// Verify user has access to the lists
	list, err := s.listRepository.GetList(ctx, listID)
	if err != nil {
		l.Error("Failed to get lists", zap.Any("error", err))
		span.RecordError(err)
		span.SetStatus(codes.Error, "List not found")
		return nil, fmt.Errorf("lists not found: %w", err)
	}

	// Check if user has access (owner or public lists)
	if list.UserID != userID && !list.IsPublic {
		l.Warn("Access denied to lists",
			zap.String("listOwnerID", list.UserID.String()))
		span.SetStatus(codes.Error, "Access denied")
		return nil, fmt.Errorf("access denied to lists")
	}

	// Get items by content type
	items, err := s.listRepository.GetListItemsByContentType(ctx, listID, contentType)
	if err != nil {
		l.Error("Failed to get lists items by content type", zap.Any("error", err))
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to get lists items")
		return nil, fmt.Errorf("failed to get lists items: %w", err)
	}

	l.Info("List items by content type fetched successfully", zap.Int("count", len(items)))
	span.SetStatus(codes.Ok, "List items fetched")
	return items, nil
}

func (s *ServiceImpl) SearchLists(ctx context.Context, searchTerm, contentType string, cityID *uuid.UUID) ([]*models.List, error) {
	ctx, span := otel.Tracer("ItineraryListService").Start(ctx, "SearchLists", trace.WithAttributes(
		attribute.String("search.term", searchTerm),
		attribute.String("content.type", contentType),
	))
	defer span.End()

	l := s.logger.With(zap.String("method", "SearchLists"),
		zap.String("searchTerm", searchTerm),
		zap.String("contentType", contentType))
	l.Debug("Searching lists")

	if cityID != nil {
		span.SetAttributes(attribute.String("city.id", cityID.String()))
		l = l.With(zap.String("cityID", cityID.String()))
	}

	// Search lists using repository
	lists, err := s.listRepository.SearchLists(ctx, searchTerm, contentType, cityID)
	if err != nil {
		l.Error("Failed to search lists", zap.Any("error", err))
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to search lists")
		return nil, fmt.Errorf("failed to search lists: %w", err)
	}

	l.Info("Lists search completed successfully", zap.Int("resultCount", len(lists)))
	span.SetStatus(codes.Ok, "Lists search completed")
	return lists, nil
}
