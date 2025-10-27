package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/FACorreiaa/go-templui/app/pkg/domain/poi"
	"github.com/FACorreiaa/go-templui/app/pkg/logger"
	"github.com/FACorreiaa/go-templui/app/pkg/middleware"
)

type FavoritesHandlers struct {
	poiService poi.Service
}

func NewFavoritesHandlers(poiService poi.Service) *FavoritesHandlers {
	return &FavoritesHandlers{
		poiService: poiService,
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
		logger.Log.Error("Invalid user ID", zap.String("user_id", userIDStr), zap.Error(err))
		c.HTML(http.StatusBadRequest, "", `<div class="text-red-500 text-sm">Invalid user ID</div>`)
		return
	}

	poiID, err := uuid.Parse(id)
	if err != nil {
		logger.Log.Error("Invalid POI ID", zap.String("poi_id", id), zap.Error(err))
		c.HTML(http.StatusBadRequest, "", `<div class="text-red-500 text-sm">Invalid place ID</div>`)
		return
	}

	logger.Log.Info("Adding place to favorites",
		zap.String("place_id", id),
		zap.String("user_id", userIDStr),
		zap.Bool("is_llm", isLLMGenerated),
	)

	// Add to database
	_, err = h.poiService.AddPoiToFavourites(c.Request.Context(), userID, poiID, isLLMGenerated)
	if err != nil {
		logger.Log.Error("Failed to add to favorites", zap.Error(err))
		c.HTML(http.StatusInternalServerError, "", `<div class="text-red-500 text-sm">Failed to add to favorites</div>`)
		return
	}

	// Return filled heart with animation
	c.HTML(http.StatusOK, "", `
		<button
			hx-delete="/favorites/`+id+`?isLLM=`+c.Query("isLLM")+`"
			hx-target="this"
			hx-swap="outerHTML"
			class="p-2 text-red-500 hover:text-red-600 rounded-lg hover:bg-red-50 dark:hover:bg-red-900/20 transition-all duration-200 animate-heartbeat"
			title="Remove from favorites"
		>
			<svg class="w-5 h-5 fill-current" fill="currentColor" viewBox="0 0 24 24">
				<path d="M20.84 4.61a5.5 5.5 0 0 0-7.78 0L12 5.67l-1.06-1.06a5.5 5.5 0 0 0-7.78 7.78l1.06 1.06L12 21.23l7.78-7.78 1.06-1.06a5.5 5.5 0 0 0 0-7.78z"></path>
			</svg>
		</button>
	`)

	logger.Log.Info("Successfully added to favorites",
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
		logger.Log.Error("Invalid user ID", zap.String("user_id", userIDStr), zap.Error(err))
		c.HTML(http.StatusBadRequest, "", `<div class="text-red-500 text-sm">Invalid user ID</div>`)
		return
	}

	poiID, err := uuid.Parse(id)
	if err != nil {
		logger.Log.Error("Invalid POI ID", zap.String("poi_id", id), zap.Error(err))
		c.HTML(http.StatusBadRequest, "", `<div class="text-red-500 text-sm">Invalid place ID</div>`)
		return
	}

	logger.Log.Info("Removing place from favorites",
		zap.String("place_id", id),
		zap.String("user_id", userIDStr),
		zap.Bool("is_llm", isLLMGenerated),
	)

	// Remove from database
	err = h.poiService.RemovePoiFromFavourites(c.Request.Context(), userID, poiID, isLLMGenerated)
	if err != nil {
		logger.Log.Error("Failed to remove from favorites", zap.Error(err))
		c.HTML(http.StatusInternalServerError, "", `<div class="text-red-500 text-sm">Failed to remove from favorites</div>`)
		return
	}

	// Return outline heart
	c.HTML(http.StatusOK, "", `
		<button
			hx-post="/favorites/add/`+id+`?isLLM=`+c.Query("isLLM")+`"
			hx-target="this"
			hx-swap="outerHTML"
			class="p-2 text-muted-foreground hover:text-red-500 rounded-lg hover:bg-red-50 dark:hover:bg-red-900/20 transition-all duration-200"
			title="Add to favorites"
		>
			<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M20.84 4.61a5.5 5.5 0 0 0-7.78 0L12 5.67l-1.06-1.06a5.5 5.5 0 0 0-7.78 7.78l1.06 1.06L12 21.23l7.78-7.78 1.06-1.06a5.5 5.5 0 0 0 0-7.78z"></path>
			</svg>
		</button>
	`)

	logger.Log.Info("Successfully removed from favorites",
		zap.String("place_id", id),
		zap.String("user_id", userIDStr),
	)
}

func (h *FavoritesHandlers) SearchFavorites(c *gin.Context) {
	query := c.PostForm("query")
	user := middleware.GetUserIDFromContext(c)

	logger.Log.Info("Searching favorites",
		zap.String("query", query),
		zap.String("user", user),
	)

	// In real app, search database and return filtered results
	// For now, return empty grid
	c.HTML(http.StatusOK, "", `<div class="text-center py-8 text-muted-foreground">No results found for "`+query+`"</div>`)
}
