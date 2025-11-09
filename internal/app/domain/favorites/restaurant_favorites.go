package favorites

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/FACorreiaa/go-templui/internal/app/domain/poi"
	"github.com/FACorreiaa/go-templui/internal/app/middleware"
)

type RestaurantFavoritesHandlers struct {
	poiService poi.Service
	logger     *zap.Logger
}

func NewRestaurantFavoritesHandlers(poiService poi.Service, logger *zap.Logger) *RestaurantFavoritesHandlers {
	return &RestaurantFavoritesHandlers{
		poiService: poiService,
		logger:     logger,
	}
}

func (h *RestaurantFavoritesHandlers) AddRestaurantFavorite(c *gin.Context) {
	id := c.Param("id")
	userIDStr := middleware.GetUserIDFromContext(c)

	// Check if user is authenticated
	if userIDStr == "" {
		c.HTML(http.StatusOK, "", `
			<div class="relative group">
				<button
					class="p-1.5 rounded-lg text-gray-400 hover:text-red-600 dark:hover:text-red-400 hover:bg-red-50 dark:hover:bg-red-900/20 transition-colors cursor-not-allowed"
					title="Sign in to add favorites"
					disabled
				>
					<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4.318 6.318a4.5 4.5 0 000 6.364L12 20.364l7.682-7.682a4.5 4.5 0 00-6.364-6.364L12 7.636l-1.318-1.318a4.5 4.5 0 00-6.364 0z"></path>
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

	restaurantID, err := uuid.Parse(id)
	if err != nil {
		h.logger.Error("Invalid restaurant ID", zap.String("restaurant_id", id), zap.Error(err))
		c.HTML(http.StatusBadRequest, "", `<div class="text-red-500 text-sm">Invalid restaurant ID</div>`)
		return
	}

	h.logger.Info("Adding restaurant to favorites",
		zap.String("restaurant_id", id),
		zap.String("user_id", userIDStr),
	)

	// Add to database
	_, err = h.poiService.AddRestaurantToFavourites(c.Request.Context(), userID, restaurantID)
	if err != nil {
		h.logger.Error("Failed to add restaurant to favorites", zap.Error(err))
		c.HTML(http.StatusInternalServerError, "", `<div class="text-red-500 text-sm">Failed to add to favorites</div>`)
		return
	}

	// Return filled heart with animation
	c.HTML(http.StatusOK, "", `
		<button
			hx-delete="/favorites/restaurants/`+id+`"
			hx-target="this"
			hx-swap="outerHTML"
			class="p-1.5 rounded-lg text-red-600 dark:text-red-400 hover:text-red-700 dark:hover:text-red-300 hover:bg-red-50 dark:hover:bg-red-900/20 transition-all duration-200 animate-heartbeat"
			title="Remove from favorites"
		>
			<svg class="w-4 h-4 fill-current" fill="currentColor" viewBox="0 0 24 24">
				<path d="M4.318 6.318a4.5 4.5 0 000 6.364L12 20.364l7.682-7.682a4.5 4.5 0 00-6.364-6.364L12 7.636l-1.318-1.318a4.5 4.5 0 00-6.364 0z"></path>
			</svg>
		</button>
	`)

	h.logger.Info("Successfully added restaurant to favorites",
		zap.String("restaurant_id", id),
		zap.String("user_id", userIDStr),
	)
}

func (h *RestaurantFavoritesHandlers) RemoveRestaurantFavorite(c *gin.Context) {
	id := c.Param("id")
	userIDStr := middleware.GetUserIDFromContext(c)

	if userIDStr == "" {
		c.HTML(http.StatusUnauthorized, "", `<div class="text-red-500 text-sm">Unauthorized</div>`)
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		h.logger.Error("Invalid user ID", zap.String("user_id", userIDStr), zap.Error(err))
		c.HTML(http.StatusBadRequest, "", `<div class="text-red-500 text-sm">Invalid user ID</div>`)
		return
	}

	restaurantID, err := uuid.Parse(id)
	if err != nil {
		h.logger.Error("Invalid restaurant ID", zap.String("restaurant_id", id), zap.Error(err))
		c.HTML(http.StatusBadRequest, "", `<div class="text-red-500 text-sm">Invalid restaurant ID</div>`)
		return
	}

	h.logger.Info("Removing restaurant from favorites",
		zap.String("restaurant_id", id),
		zap.String("user_id", userIDStr),
	)

	// Remove from database
	err = h.poiService.RemoveRestaurantFromFavourites(c.Request.Context(), userID, restaurantID)
	if err != nil {
		h.logger.Error("Failed to remove restaurant from favorites", zap.Error(err))
		c.HTML(http.StatusInternalServerError, "", `<div class="text-red-500 text-sm">Failed to remove from favorites</div>`)
		return
	}

	// Return empty heart
	c.HTML(http.StatusOK, "", `
		<button
			hx-post="/favorites/restaurants/`+id+`"
			hx-target="this"
			hx-swap="outerHTML"
			class="p-1.5 rounded-lg text-gray-400 hover:text-red-600 dark:hover:text-red-400 hover:bg-red-50 dark:hover:bg-red-900/20 transition-colors"
			title="Add to favorites"
		>
			<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4.318 6.318a4.5 4.5 0 000 6.364L12 20.364l7.682-7.682a4.5 4.5 0 00-6.364-6.364L12 7.636l-1.318-1.318a4.5 4.5 0 00-6.364 0z"></path>
			</svg>
		</button>
	`)

	h.logger.Info("Successfully removed restaurant from favorites",
		zap.String("restaurant_id", id),
		zap.String("user_id", userIDStr),
	)
}
