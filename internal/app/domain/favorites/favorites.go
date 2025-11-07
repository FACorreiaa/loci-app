package favorites

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/FACorreiaa/go-templui/internal/app/domain"
	"github.com/FACorreiaa/go-templui/internal/app/domain/poi"
	"github.com/FACorreiaa/go-templui/internal/app/models"
	"github.com/FACorreiaa/go-templui/internal/app/pages"
	"github.com/FACorreiaa/go-templui/internal/pkg/middleware"
)

type FavoritesHandlers struct {
	*domain.BaseHandler
	poiService poi.Service
	logger     *zap.Logger
}

func NewFavoritesHandlers(poiService poi.Service,
	logger *zap.Logger,
	base *domain.BaseHandler) *FavoritesHandlers {
	return &FavoritesHandlers{
		poiService:  poiService,
		logger:      logger,
		BaseHandler: base,
	}
}

func (h *FavoritesHandlers) AddFavorite(c *gin.Context) {
	id := c.Param("id")
	userIDStr := middleware.GetUserIDFromContext(c)
	isLLMGenerated := c.Query("isLLM") == "true"

	// Check if user is authenticated
	if userIDStr == "" {
		c.HTML(http.StatusOK, "", `
			<div class="relative group">
				<button
					class="p-2 text-muted-foreground hover:text-red-500 rounded-lg hover:bg-red-50 dark:hover:bg-red-900/20 transition-colors cursor-not-allowed"
					title="Sign in to add favorites"
					disabled
				>
					<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M20.84 4.61a5.5 5.5 0 0 0-7.78 0L12 5.67l-1.06-1.06a5.5 5.5 0 0 0-7.78 7.78l1.06 1.06L12 21.23l7.78-7.78 1.06-1.06a5.5 5.5 0 0 0 0-7.78z"></path>
					</svg>
				</button>
				<div class="absolute bottom-full mb-2 left-1/2 transform -translate-x-1/2 hidden group-hover:block">
					<div class="bg-gray-900 text-white text-xs rounded py-1 px-2 whitespace-nowrap">
						Sign in to save favorites
						<div class="absolute top-full left-1/2 transform -translate-x-1/2 border-4 border-transparent border-t-gray-900"></div>
					</div>
				</div>
			</div>
		`)
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		h.logger.Error("Invalid user ID", zap.String("user_id", userIDStr), zap.Error(err))
		c.HTML(http.StatusBadRequest, "", `<div class="text-red-500 text-sm">Invalid user ID</div>`)
		return
	}

	poiID, err := uuid.Parse(id)
	if err != nil {
		h.logger.Error("Invalid POI ID", zap.String("poi_id", id), zap.Error(err))
		c.HTML(http.StatusBadRequest, "", `<div class="text-red-500 text-sm">Invalid place ID</div>`)
		return
	}

	h.logger.Info("Adding place to favorites",
		zap.String("place_id", id),
		zap.String("user_id", userIDStr),
		zap.Bool("is_llm", isLLMGenerated),
	)

	// Add to database
	_, err = h.poiService.AddPoiToFavourites(c.Request.Context(), userID, poiID, isLLMGenerated)
	if err != nil {
		h.logger.Error("Failed to add to favorites", zap.Error(err))
		c.HTML(http.StatusInternalServerError, "", `<div class="text-red-500 text-sm">Failed to add to favorites</div>`)
		return
	}

	// Return filled heart with animation
	component := UnfavoriteButton(id, isLLMGenerated)
	component.Render(c.Request.Context(), c.Writer)

	h.logger.Info("Successfully added to favorites",
		zap.String("place_id", id),
		zap.String("user_id", userIDStr),
	)
}

func (h *FavoritesHandlers) RemoveFavorite(c *gin.Context) {
	id := c.Param("id")
	userIDStr := middleware.GetUserIDFromContext(c)
	isLLMGenerated := c.Query("isLLM") == "true"

	if userIDStr == "" {
		c.HTML(http.StatusUnauthorized, "", `<div class="text-red-500 text-sm">Not authenticated</div>`)
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		h.logger.Error("Invalid user ID", zap.String("user_id", userIDStr), zap.Error(err))
		c.HTML(http.StatusBadRequest, "", `<div class="text-red-500 text-sm">Invalid user ID</div>`)
		return
	}

	poiID, err := uuid.Parse(id)
	if err != nil {
		h.logger.Error("Invalid POI ID", zap.String("poi_id", id), zap.Error(err))
		c.HTML(http.StatusBadRequest, "", `<div class="text-red-500 text-sm">Invalid place ID</div>`)
		return
	}

	h.logger.Info("Removing place from favorites",
		zap.String("place_id", id),
		zap.String("user_id", userIDStr),
		zap.Bool("is_llm", isLLMGenerated),
	)

	// Remove from database
	err = h.poiService.RemovePoiFromFavourites(c.Request.Context(), userID, poiID, isLLMGenerated)
	if err != nil {
		h.logger.Error("Failed to remove from favorites", zap.Error(err))
		c.HTML(http.StatusInternalServerError, "", `<div class="text-red-500 text-sm">Failed to remove from favorites</div>`)
		return
	}

	// Return outline heart
	component := FavoriteButton(id, isLLMGenerated)
	component.Render(c.Request.Context(), c.Writer)

	h.logger.Info("Successfully removed from favorites",
		zap.String("place_id", id),
		zap.String("user_id", userIDStr),
	)
}

func (h *FavoritesHandlers) SearchFavorites(c *gin.Context) {
	query := c.PostForm("query")
	user := middleware.GetUserIDFromContext(c)

	h.logger.Info("Searching favorites",
		zap.String("query", query),
		zap.String("user", user),
	)

	// In real app, search database and return filtered results
	// For now, return empty grid
	c.HTML(http.StatusOK, "", `<div class="text-center py-8 text-muted-foreground">No results found for "`+query+`"</div>`)
}

// ListFavorites displays the favourites lists page with search, filter, and pagination
func (h *FavoritesHandlers) ListFavorites(c *gin.Context) {
	userIDStr := middleware.GetUserIDFromContext(c)
	if userIDStr == "" || userIDStr == "anonymous" {
		c.Redirect(http.StatusFound, "/auth/signin")
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		h.Logger.Error("Invalid user ID", zap.String("user_id", userIDStr), zap.Error(err))
		h.RenderPage(c, "Error", "Favorites", pages.NotFoundPage())
		return
	}

	searchText := c.Query("search")
	category := c.Query("category")
	sortBy := c.DefaultQuery("sort_by", "added_at")
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

	filter := models.FavouritesFilter{
		UserID:     userID,
		SearchText: searchText,
		Category:   category,
		SortBy:     sortBy,
		SortOrder:  sortOrder,
		Limit:      pageSize,
		Offset:     (page - 1) * pageSize,
	}

	pois, total, err := h.poiService.GetFavouritesFiltered(c.Request.Context(), filter)
	if err != nil {
		h.Logger.Error("Failed to get favourites", zap.Error(err))
		h.RenderPage(c, "Error", "Favorites", pages.NotFoundPage())
		return
	}

	totalPages := (total + pageSize - 1) / pageSize

	data := FavoritesPageData{
		POIs:       pois,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
		SearchText: searchText,
		Category:   category,
		SortBy:     sortBy,
		SortOrder:  sortOrder,
	}

	h.RenderPage(c, "Favorites - Loci", "Favorites", FavoritesPage(data))
}
