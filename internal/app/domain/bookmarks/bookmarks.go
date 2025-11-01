package bookmarks

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/FACorreiaa/go-templui/internal/app/domain/poi"
	"github.com/FACorreiaa/go-templui/internal/app/models"
	"github.com/FACorreiaa/go-templui/internal/pkg/logger"
	"github.com/FACorreiaa/go-templui/internal/pkg/middleware"
)

type BookmarksHandlers struct {
	poiService poi.Service
}

func NewBookmarksHandlers(poiService poi.Service) *BookmarksHandlers {
	return &BookmarksHandlers{
		poiService: poiService,
	}
}

func (h *BookmarksHandlers) AddBookmark(c *gin.Context) {
	id := c.Param("id")
	userIDStr := middleware.GetUserIDFromContext(c)

	if userIDStr == "" {
		c.HTML(http.StatusOK, "", `
			<div class="relative group">
				<button
					class="p-2 text-muted-foreground hover:text-blue-500 rounded-lg hover:bg-blue-50 dark:hover:bg-blue-900/20 transition-colors cursor-not-allowed"
					title="Sign in to bookmark itineraries"
					disabled
				>
					<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 5a2 2 0 012-2h10a2 2 0 012 2v16l-7-3.5L5 21V5z"></path>
					</svg>
				</button>
				<div class="absolute bottom-full mb-2 left-1/2 transform -translate-x-1/2 hidden group-hover:block">
					<div class="bg-gray-900 text-white text-xs rounded py-1 px-2 whitespace-nowrap">
						Sign in to bookmark itineraries
						<div class="absolute top-full left-1/2 transform -translate-x-1/2 border-4 border-transparent border-t-gray-900"></div>
					</div>
				</div>
			</div>
		`)
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		logger.Log.Error("Invalid user ID", zap.String("user_id", userIDStr), zap.Error(err))
		c.HTML(http.StatusBadRequest, "", `<div class="text-red-500 text-sm">Invalid user ID</div>`)
		return
	}

	itineraryID, err := uuid.Parse(id)
	if err != nil {
		logger.Log.Error("Invalid itinerary ID", zap.String("itinerary_id", id), zap.Error(err))
		c.HTML(http.StatusBadRequest, "", `<div class="text-red-500 text-sm">Invalid itinerary ID</div>`)
		return
	}

	logger.Log.Info("Adding itinerary to bookmarks",
		zap.String("itinerary_id", id),
		zap.String("user_id", userIDStr),
	)

	// Add to database
	_, err = h.poiService.AddItineraryToBookmarks(c.Request.Context(), userID, itineraryID)
	if err != nil {
		logger.Log.Error("Failed to add to bookmarks", zap.Error(err))
		c.HTML(http.StatusInternalServerError, "", `<div class="text-red-500 text-sm">Failed to add to bookmarks</div>`)
		return
	}

	// Return filled bookmark icon with animation
	c.HTML(http.StatusOK, "", `
		<button
			hx-delete="/bookmarks/`+id+`"
			hx-target="this"
			hx-swap="outerHTML"
			class="p-2 text-blue-500 hover:text-blue-600 rounded-lg hover:bg-blue-50 dark:hover:bg-blue-900/20 transition-all duration-200 animate-heartbeat"
			title="Remove from bookmarks"
		>
			<svg class="w-5 h-5 fill-current" fill="currentColor" viewBox="0 0 24 24">
				<path d="M5 5a2 2 0 012-2h10a2 2 0 012 2v16l-7-3.5L5 21V5z"></path>
			</svg>
		</button>
	`)

	logger.Log.Info("Successfully added to bookmarks",
		zap.String("itinerary_id", id),
		zap.String("user_id", userIDStr),
	)
}

func (h *BookmarksHandlers) RemoveBookmark(c *gin.Context) {
	id := c.Param("id")
	userIDStr := middleware.GetUserIDFromContext(c)

	if userIDStr == "" {
		c.HTML(http.StatusUnauthorized, "", `<div class="text-red-500 text-sm">Not authenticated</div>`)
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		logger.Log.Error("Invalid user ID", zap.String("user_id", userIDStr), zap.Error(err))
		c.HTML(http.StatusBadRequest, "", `<div class="text-red-500 text-sm">Invalid user ID</div>`)
		return
	}

	itineraryID, err := uuid.Parse(id)
	if err != nil {
		logger.Log.Error("Invalid itinerary ID", zap.String("itinerary_id", id), zap.Error(err))
		c.HTML(http.StatusBadRequest, "", `<div class="text-red-500 text-sm">Invalid itinerary ID</div>`)
		return
	}

	logger.Log.Info("Removing itinerary from bookmarks",
		zap.String("itinerary_id", id),
		zap.String("user_id", userIDStr),
	)

	// Remove from database
	err = h.poiService.RemoveItineraryFromBookmarks(c.Request.Context(), userID, itineraryID)
	if err != nil {
		logger.Log.Error("Failed to remove from bookmarks", zap.Error(err))
		c.HTML(http.StatusInternalServerError, "", `<div class="text-red-500 text-sm">Failed to remove from bookmarks</div>`)
		return
	}

	// Return outline bookmark icon
	c.HTML(http.StatusOK, "", `
		<button
			hx-post="/bookmarks/add/`+id+`"
			hx-target="this"
			hx-swap="outerHTML"
			class="p-2 text-muted-foreground hover:text-blue-500 rounded-lg hover:bg-blue-50 dark:hover:bg-blue-900/20 transition-all duration-200"
			title="Add to bookmarks"
		>
			<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 5a2 2 0 012-2h10a2 2 0 012 2v16l-7-3.5L5 21V5z"></path>
			</svg>
		</button>
	`)

	logger.Log.Info("Successfully removed from bookmarks",
		zap.String("itinerary_id", id),
		zap.String("user_id", userIDStr),
	)
}

// ListBookmarks displays the bookmarks list page with search, filter, and pagination
func (h *BookmarksHandlers) ListBookmarks(c *gin.Context) {
	userIDStr := middleware.GetUserIDFromContext(c)
	if userIDStr == "" || userIDStr == "anonymous" {
		c.Redirect(http.StatusFound, "/auth/signin")
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		logger.Log.Error("Invalid user ID", zap.String("user_id", userIDStr), zap.Error(err))
		c.HTML(http.StatusBadRequest, "", `<div class="text-red-500">Invalid user ID</div>`)
		return
	}

	// Parse query parameters
	searchText := c.Query("search")
	sortBy := c.DefaultQuery("sort_by", "bookmarked_at")
	sortOrder := c.DefaultQuery("sort_order", "desc")

	page := 1
	if pageStr := c.Query("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	pageSize := 20
	if pageSizeStr := c.Query("page_size"); pageSizeStr != "" {
		if ps, err := strconv.Atoi(pageSizeStr); err == nil && ps > 0 && ps <= 100 {
			pageSize = ps
		}
	}

	// Build filter
	filter := models.BookmarksFilter{
		UserID:     userID,
		SearchText: searchText,
		SortBy:     sortBy,
		SortOrder:  sortOrder,
		Limit:      pageSize,
		Offset:     (page - 1) * pageSize,
	}

	// Get filtered bookmarks
	itineraries, total, err := h.poiService.GetBookmarksFiltered(c.Request.Context(), filter)
	if err != nil {
		logger.Log.Error("Failed to get bookmarks", zap.Error(err))
		c.HTML(http.StatusInternalServerError, "", `<div class="text-red-500">Failed to load bookmarks</div>`)
		return
	}

	// Calculate pagination info
	totalPages := (total + pageSize - 1) / pageSize

	// Prepare data for template
	data := BookmarksPageData{
		Itineraries: itineraries,
		Total:       total,
		Page:        page,
		PageSize:    pageSize,
		TotalPages:  totalPages,
		SearchText:  searchText,
		SortBy:      sortBy,
		SortOrder:   sortOrder,
	}

	// Render the bookmarks list page using templ
	c.Writer.Header().Set("Content-Type", "text/html; charset=utf-8")
	BookmarksPage(data).Render(c.Request.Context(), c.Writer)
}
