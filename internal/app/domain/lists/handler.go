package lists

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/FACorreiaa/go-templui/internal/app/domain/pages"
	"github.com/FACorreiaa/go-templui/internal/app/models"
	"github.com/FACorreiaa/go-templui/internal/pkg/middleware"
)

type Handler struct {
	service Service
	log     *zap.Logger
}

func NewHandler(service Service, log *zap.Logger) *Handler {
	return &Handler{
		service: service,
		log:     log,
	}
}

// ShowListsPage renders the main lists page with user's lists
func (h *Handler) ShowListsPage(c *gin.Context) {
	// Get user ID from context
	userIDStr := middleware.GetUserIDFromContext(c)
	if userIDStr == "" || userIDStr == "anonymous" {
		h.log.Error("User not authenticated")
		c.Redirect(http.StatusSeeOther, "/auth/signin")
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		h.log.Error("Invalid user ID", zap.String("userID", userIDStr), zap.Error(err))
		c.String(http.StatusBadRequest, "Invalid user ID")
		return
	}

	// Get user's lists (non-itinerary lists)
	lists, err := (h.service).GetUserLists(c.Request.Context(), userID, false)
	if err != nil {
		h.log.Error("Failed to get user lists",
			zap.String("userID", userID.String()),
			zap.Error(err))
		c.String(http.StatusInternalServerError, "Failed to load lists")
		return
	}

	// Get user info for the layout
	user := middleware.GetUserFromContext(c)
	if user == nil {
		h.log.Error("Failed to get user from context")
		c.Redirect(http.StatusSeeOther, "/auth/signin")
		return
	}

	// Render the lists page with actual data
	c.HTML(http.StatusOK, "", pages.LayoutPage(models.LayoutTempl{
		Title:   "Travel Lists - Loci",
		Content: ListsPageWithData(lists),
		Nav: models.Navigation{
			Items: []models.NavItem{
				{Name: "Dashboard", URL: "/dashboard"},
				{Name: "Discover", URL: "/discover"},
				{Name: "Nearby", URL: "/nearby"},
				{Name: "Chat", URL: "/chat"},
				{Name: "Lists", URL: "/lists"},
			},
		},
		ActiveNav: "Lists",
		User:      user,
	}))
}

// ShowCreateModal renders the create list modal
func (h *Handler) ShowCreateModal(c *gin.Context) {
	c.HTML(http.StatusOK, "", CreateListModal())
}

// CreateList handles the POST request to create a new list
func (h *Handler) CreateList(c *gin.Context) {
	// Get user ID from context
	userIDStr := middleware.GetUserIDFromContext(c)
	if userIDStr == "" || userIDStr == "anonymous" {
		h.log.Error("User not authenticated")
		c.String(http.StatusUnauthorized, "Unauthorized")
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		h.log.Error("Invalid user ID", zap.String("userID", userIDStr), zap.Error(err))
		c.String(http.StatusBadRequest, "Invalid user ID")
		return
	}

	// Parse form data
	name := c.PostForm("name")
	description := c.PostForm("description")
	isPublicStr := c.PostForm("is_public")

	// Validate required fields
	if name == "" {
		h.log.Error("List name is required")
		c.String(http.StatusBadRequest, "List name is required")
		return
	}

	// Parse isPublic
	isPublic, err := strconv.ParseBool(isPublicStr)
	if err != nil {
		// Default to private if parsing fails
		isPublic = false
	}

	// Create the list using the service (as a top-level list)
	list, err := (h.service).CreateTopLevelList(c.Request.Context(), userID, name, description, nil, false, isPublic)
	if err != nil {
		h.log.Error("Failed to create list",
			zap.String("userID", userID.String()),
			zap.String("name", name),
			zap.Error(err))
		c.String(http.StatusInternalServerError, "Failed to create list")
		return
	}

	h.log.Info("List created successfully",
		zap.String("listID", list.ID.String()),
		zap.String("userID", userID.String()),
		zap.String("name", name))

	// Redirect back to lists page with full layout
	// Get user info for the layout
	user := middleware.GetUserFromContext(c)
	if user == nil {
		h.log.Error("Failed to get user from context")
		c.Redirect(http.StatusSeeOther, "/auth/signin")
		return
	}

	c.HTML(http.StatusOK, "", pages.LayoutPage(models.LayoutTempl{
		Title:   "Travel Lists - Loci",
		Content: ListsPage(),
		Nav: models.Navigation{
			Items: []models.NavItem{
				{Name: "Dashboard", URL: "/dashboard"},
				{Name: "Discover", URL: "/discover"},
				{Name: "Nearby", URL: "/nearby"},
				{Name: "Chat", URL: "/chat"},
				{Name: "Lists", URL: "/lists"},
			},
		},
		ActiveNav: "Lists",
		User:      user,
	}))
}

// ShowAddToListModal renders the modal for selecting which list to add an item to
func (h *Handler) ShowAddToListModal(c *gin.Context) {
	// Get user ID from context
	userIDStr := middleware.GetUserIDFromContext(c)
	if userIDStr == "" || userIDStr == "anonymous" {
		h.log.Error("User not authenticated")
		c.String(http.StatusUnauthorized, "Unauthorized")
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		h.log.Error("Invalid user ID", zap.String("userID", userIDStr), zap.Error(err))
		c.String(http.StatusBadRequest, "Invalid user ID")
		return
	}

	// Get query parameters
	itemID := c.Query("item_id")
	itemName := c.Query("item_name")
	contentType := c.Query("content_type")

	// Validate required parameters
	if itemID == "" || contentType == "" {
		h.log.Error("Missing required parameters", zap.String("itemID", itemID), zap.String("contentType", contentType))
		c.String(http.StatusBadRequest, "Missing item_id or content_type")
		return
	}

	// Get user's lists (non-itinerary lists)
	lists, err := (h.service).GetUserLists(c.Request.Context(), userID, false)
	if err != nil {
		h.log.Error("Failed to get user lists",
			zap.String("userID", userID.String()),
			zap.Error(err))
		c.String(http.StatusInternalServerError, "Failed to load lists")
		return
	}

	// Convert []*models.List to []models.List
	var userLists []models.List
	for _, list := range lists {
		if list != nil {
			userLists = append(userLists, *list)
		}
	}

	// Render the modal
	c.HTML(http.StatusOK, "", AddToListModal(itemID, itemName, contentType, userLists))
}

// AddItemToList handles adding an item to a selected list
func (h *Handler) AddItemToList(c *gin.Context) {
	// Get user ID from context
	userIDStr := middleware.GetUserIDFromContext(c)
	if userIDStr == "" || userIDStr == "anonymous" {
		h.log.Error("User not authenticated")
		c.String(http.StatusUnauthorized, "Unauthorized")
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		h.log.Error("Invalid user ID", zap.String("userID", userIDStr), zap.Error(err))
		c.String(http.StatusBadRequest, "Invalid user ID")
		return
	}

	// Get list ID from URL parameter
	listIDStr := c.Param("id")
	listID, err := uuid.Parse(listIDStr)
	if err != nil {
		h.log.Error("Invalid list ID", zap.String("listID", listIDStr), zap.Error(err))
		c.String(http.StatusBadRequest, "Invalid list ID")
		return
	}

	// Parse request body
	var req struct {
		ItemID      string `json:"item_id" binding:"required"`
		ContentType string `json:"content_type" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		h.log.Error("Failed to parse request body", zap.Error(err))
		c.String(http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate item ID
	itemUUID, err := uuid.Parse(req.ItemID)
	if err != nil {
		h.log.Error("Invalid item ID", zap.String("itemID", req.ItemID), zap.Error(err))
		c.String(http.StatusBadRequest, "Invalid item ID")
		return
	}

	// Create the request for the service
	addItemReq := models.AddListItemRequest{
		ItemID:      itemUUID,
		ContentType: models.ContentType(req.ContentType),
	}

	// Add item to list using the service
	listItem, err := (h.service).AddListItem(c.Request.Context(), userID, listID, addItemReq)
	if err != nil {
		h.log.Error("Failed to add item to list",
			zap.String("userID", userID.String()),
			zap.String("listID", listID.String()),
			zap.String("itemID", req.ItemID),
			zap.String("contentType", req.ContentType),
			zap.Error(err))
		c.String(http.StatusInternalServerError, "Failed to add item to list")
		return
	}

	h.log.Info("Item added to list successfully",
		zap.String("listID", listItem.ListID.String()),
		zap.String("itemID", listItem.ItemID.String()),
		zap.String("contentType", string(listItem.ContentType)))

	// Return success response
	c.Status(http.StatusOK)
}

// ShowListDetail renders the detail page for a specific list
func (h *Handler) ShowListDetail(c *gin.Context) {
	// Get user ID from context
	userIDStr := middleware.GetUserIDFromContext(c)
	if userIDStr == "" || userIDStr == "anonymous" {
		h.log.Error("User not authenticated")
		c.Redirect(http.StatusSeeOther, "/auth/signin")
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		h.log.Error("Invalid user ID", zap.String("userID", userIDStr), zap.Error(err))
		c.String(http.StatusBadRequest, "Invalid user ID")
		return
	}

	// Get list ID from URL parameter
	listIDStr := c.Param("id")
	listID, err := uuid.Parse(listIDStr)
	if err != nil {
		h.log.Error("Invalid list ID", zap.String("listID", listIDStr), zap.Error(err))
		c.String(http.StatusBadRequest, "Invalid list ID")
		return
	}

	// Get list details with items using the service
	listWithItems, err := (h.service).GetListDetails(c.Request.Context(), listID, userID)
	if err != nil {
		h.log.Error("Failed to get list details",
			zap.String("userID", userID.String()),
			zap.String("listID", listID.String()),
			zap.Error(err))
		c.String(http.StatusInternalServerError, "Failed to load list details")
		return
	}

	// Get user info for the layout
	user := middleware.GetUserFromContext(c)
	if user == nil {
		h.log.Error("Failed to get user from context")
		c.Redirect(http.StatusSeeOther, "/auth/signin")
		return
	}

	// Render the list detail page
	c.HTML(http.StatusOK, "", pages.LayoutPage(models.LayoutTempl{
		Title:   listWithItems.List.Name + " - Lists - Loci",
		Content: ListDetailPage(listWithItems),
		Nav: models.Navigation{
			Items: []models.NavItem{
				{Name: "Dashboard", URL: "/dashboard"},
				{Name: "Discover", URL: "/discover"},
				{Name: "Nearby", URL: "/nearby"},
				{Name: "Chat", URL: "/chat"},
				{Name: "Lists", URL: "/lists"},
			},
		},
		ActiveNav: "Lists",
		User:      user,
	}))
}

// RemoveListItem handles removing an item from a list
func (h *Handler) RemoveListItem(c *gin.Context) {
	// Get user ID from context
	userIDStr := middleware.GetUserIDFromContext(c)
	if userIDStr == "" || userIDStr == "anonymous" {
		h.log.Error("User not authenticated")
		c.String(http.StatusUnauthorized, "Unauthorized")
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		h.log.Error("Invalid user ID", zap.String("userID", userIDStr), zap.Error(err))
		c.String(http.StatusBadRequest, "Invalid user ID")
		return
	}

	// Get list ID and item ID from URL parameters
	listIDStr := c.Param("id")
	listID, err := uuid.Parse(listIDStr)
	if err != nil {
		h.log.Error("Invalid list ID", zap.String("listID", listIDStr), zap.Error(err))
		c.String(http.StatusBadRequest, "Invalid list ID")
		return
	}

	itemIDStr := c.Param("itemId")
	itemID, err := uuid.Parse(itemIDStr)
	if err != nil {
		h.log.Error("Invalid item ID", zap.String("itemID", itemIDStr), zap.Error(err))
		c.String(http.StatusBadRequest, "Invalid item ID")
		return
	}

	// Remove the item from the list using the service
	err = (h.service).RemoveListItem(c.Request.Context(), userID, listID, itemID)
	if err != nil {
		h.log.Error("Failed to remove item from list",
			zap.String("userID", userID.String()),
			zap.String("listID", listID.String()),
			zap.String("itemID", itemID.String()),
			zap.Error(err))
		c.String(http.StatusInternalServerError, "Failed to remove item from list")
		return
	}

	h.log.Info("Item removed from list successfully",
		zap.String("listID", listID.String()),
		zap.String("itemID", itemID.String()))

	// Return empty response (HTMX will remove the element)
	c.Status(http.StatusOK)
}

// ShowEditModal renders the edit list modal
func (h *Handler) ShowEditModal(c *gin.Context) {
	// Get user ID from context
	userIDStr := middleware.GetUserIDFromContext(c)
	if userIDStr == "" || userIDStr == "anonymous" {
		h.log.Error("User not authenticated")
		c.String(http.StatusUnauthorized, "Unauthorized")
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		h.log.Error("Invalid user ID", zap.String("userID", userIDStr), zap.Error(err))
		c.String(http.StatusBadRequest, "Invalid user ID")
		return
	}

	// Get list ID from URL parameter
	listIDStr := c.Param("id")
	listID, err := uuid.Parse(listIDStr)
	if err != nil {
		h.log.Error("Invalid list ID", zap.String("listID", listIDStr), zap.Error(err))
		c.String(http.StatusBadRequest, "Invalid list ID")
		return
	}

	// Get list details
	listWithItems, err := (h.service).GetListDetails(c.Request.Context(), listID, userID)
	if err != nil {
		h.log.Error("Failed to get list details", zap.Error(err))
		c.String(http.StatusInternalServerError, "Failed to load list")
		return
	}

	// Render the edit modal
	c.HTML(http.StatusOK, "", EditListModal(&listWithItems.List))
}

// UpdateList handles updating a list
func (h *Handler) UpdateList(c *gin.Context) {
	// Get user ID from context
	userIDStr := middleware.GetUserIDFromContext(c)
	if userIDStr == "" || userIDStr == "anonymous" {
		h.log.Error("User not authenticated")
		c.String(http.StatusUnauthorized, "Unauthorized")
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		h.log.Error("Invalid user ID", zap.Error(err))
		c.String(http.StatusBadRequest, "Invalid user ID")
		return
	}

	// Get list ID from URL parameter
	listIDStr := c.Param("id")
	listID, err := uuid.Parse(listIDStr)
	if err != nil {
		h.log.Error("Invalid list ID", zap.Error(err))
		c.String(http.StatusBadRequest, "Invalid list ID")
		return
	}

	// Parse form data
	name := c.PostForm("name")
	description := c.PostForm("description")
	isPublicStr := c.PostForm("is_public")

	// Validate required fields
	if name == "" {
		h.log.Error("List name is required")
		c.String(http.StatusBadRequest, "List name is required")
		return
	}

	// Parse isPublic
	isPublic, err := strconv.ParseBool(isPublicStr)
	if err != nil {
		isPublic = false
	}

	// Create update request
	updateReq := models.UpdateListRequest{
		Name:        &name,
		Description: &description,
		IsPublic:    &isPublic,
	}

	// Update the list
	_, err = (h.service).UpdateListDetails(c.Request.Context(), listID, userID, updateReq)
	if err != nil {
		h.log.Error("Failed to update list", zap.Error(err))
		c.String(http.StatusInternalServerError, "Failed to update list")
		return
	}

	h.log.Info("List updated successfully", zap.String("listID", listID.String()))

	// Redirect back to list detail page
	c.Redirect(http.StatusSeeOther, "/lists/"+listID.String())
}

// ShowDeleteModal renders the delete confirmation modal
func (h *Handler) ShowDeleteModal(c *gin.Context) {
	// Get user ID from context
	userIDStr := middleware.GetUserIDFromContext(c)
	if userIDStr == "" || userIDStr == "anonymous" {
		h.log.Error("User not authenticated")
		c.String(http.StatusUnauthorized, "Unauthorized")
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		h.log.Error("Invalid user ID", zap.Error(err))
		c.String(http.StatusBadRequest, "Invalid user ID")
		return
	}

	// Get list ID from URL parameter
	listIDStr := c.Param("id")
	listID, err := uuid.Parse(listIDStr)
	if err != nil {
		h.log.Error("Invalid list ID", zap.Error(err))
		c.String(http.StatusBadRequest, "Invalid list ID")
		return
	}

	// Get list details
	listWithItems, err := (h.service).GetListDetails(c.Request.Context(), listID, userID)
	if err != nil {
		h.log.Error("Failed to get list details", zap.Error(err))
		c.String(http.StatusInternalServerError, "Failed to load list")
		return
	}

	// Render the delete modal
	c.HTML(http.StatusOK, "", DeleteListModal(&listWithItems.List))
}

// DeleteList handles deleting a list
func (h *Handler) DeleteList(c *gin.Context) {
	// Get user ID from context
	userIDStr := middleware.GetUserIDFromContext(c)
	if userIDStr == "" || userIDStr == "anonymous" {
		h.log.Error("User not authenticated")
		c.String(http.StatusUnauthorized, "Unauthorized")
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		h.log.Error("Invalid user ID", zap.Error(err))
		c.String(http.StatusBadRequest, "Invalid user ID")
		return
	}

	// Get list ID from URL parameter
	listIDStr := c.Param("id")
	listID, err := uuid.Parse(listIDStr)
	if err != nil {
		h.log.Error("Invalid list ID", zap.Error(err))
		c.String(http.StatusBadRequest, "Invalid list ID")
		return
	}

	// Delete the list
	err = (h.service).DeleteUserList(c.Request.Context(), listID, userID)
	if err != nil {
		h.log.Error("Failed to delete list", zap.Error(err))
		c.String(http.StatusInternalServerError, "Failed to delete list")
		return
	}

	h.log.Info("List deleted successfully", zap.String("listID", listID.String()))

	// Redirect to lists page
	c.Redirect(http.StatusSeeOther, "/lists")
}

// ShowSavedListsPage renders the saved lists page
func (h *Handler) ShowSavedListsPage(c *gin.Context) {
	// Get user ID from context
	userIDStr := middleware.GetUserIDFromContext(c)
	if userIDStr == "" || userIDStr == "anonymous" {
		h.log.Error("User not authenticated")
		c.Redirect(http.StatusSeeOther, "/auth/signin")
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		h.log.Error("Invalid user ID", zap.String("userID", userIDStr), zap.Error(err))
		c.String(http.StatusBadRequest, "Invalid user ID")
		return
	}

	// Get saved lists from service
	lists, err := (h.service).GetUserSavedLists(c.Request.Context(), userID)
	if err != nil {
		h.log.Error("Failed to get saved lists", zap.Error(err))
		c.String(http.StatusInternalServerError, "Failed to load saved lists")
		return
	}

	user := middleware.GetUserFromContext(c)
	if user == nil {
		h.log.Error("Failed to get user from context")
		c.Redirect(http.StatusSeeOther, "/auth/signin")
		return
	}

	c.HTML(http.StatusOK, "", pages.LayoutPage(models.LayoutTempl{
		Title:   "Saved Lists - Loci",
		Content: SavedListsPage(lists),
		Nav: models.Navigation{
			Items: []models.NavItem{
				{Name: "Dashboard", URL: "/dashboard"},
				{Name: "Discover", URL: "/discover"},
				{Name: "Nearby", URL: "/nearby"},
				{Name: "Chat", URL: "/chat"},
				{Name: "Lists", URL: "/lists"},
			},
		},
		ActiveNav: "Lists",
		User:      user,
	}))
}

// SaveListAction handles saving a list for the current user
func (h *Handler) SaveListAction(c *gin.Context) {
	// Get user ID from context
	userIDStr := middleware.GetUserIDFromContext(c)
	if userIDStr == "" || userIDStr == "anonymous" {
		h.log.Error("User not authenticated")
		c.String(http.StatusUnauthorized, "Unauthorized")
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		h.log.Error("Invalid user ID", zap.String("userID", userIDStr), zap.Error(err))
		c.String(http.StatusBadRequest, "Invalid user ID")
		return
	}

	// Get list ID from URL parameter
	listIDStr := c.Param("id")
	listID, err := uuid.Parse(listIDStr)
	if err != nil {
		h.log.Error("Invalid list ID", zap.Error(err))
		c.String(http.StatusBadRequest, "Invalid list ID")
		return
	}

	// Save the list
	err = (h.service).SaveList(c.Request.Context(), userID, listID)
	if err != nil {
		h.log.Error("Failed to save list", zap.Error(err))
		c.String(http.StatusInternalServerError, "Failed to save list")
		return
	}

	h.log.Info("List saved successfully", zap.String("listID", listID.String()))

	// Return success response (for HTMX)
	c.Status(http.StatusOK)
}

// UnsaveListAction handles unsaving a list for the current user
func (h *Handler) UnsaveListAction(c *gin.Context) {
	// Get user ID from context
	userIDStr := middleware.GetUserIDFromContext(c)
	if userIDStr == "" || userIDStr == "anonymous" {
		h.log.Error("User not authenticated")
		c.String(http.StatusUnauthorized, "Unauthorized")
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		h.log.Error("Invalid user ID", zap.String("userID", userIDStr), zap.Error(err))
		c.String(http.StatusBadRequest, "Invalid user ID")
		return
	}

	// Get list ID from URL parameter
	listIDStr := c.Param("id")
	listID, err := uuid.Parse(listIDStr)
	if err != nil {
		h.log.Error("Invalid list ID", zap.Error(err))
		c.String(http.StatusBadRequest, "Invalid list ID")
		return
	}

	// Unsave the list
	err = (h.service).UnsaveList(c.Request.Context(), userID, listID)
	if err != nil {
		h.log.Error("Failed to unsave list", zap.Error(err))
		c.String(http.StatusInternalServerError, "Failed to unsave list")
		return
	}

	h.log.Info("List unsaved successfully", zap.String("listID", listID.String()))

	// Return success response (for HTMX)
	c.Status(http.StatusOK)
}
