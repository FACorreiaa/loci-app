package handlers

import (
	"net/http"

	"github.com/FACorreiaa/go-templui/pkg/logger"
	"github.com/FACorreiaa/go-templui/pkg/middleware"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type FavoritesHandlers struct{}

func NewFavoritesHandlers() *FavoritesHandlers {
	return &FavoritesHandlers{}
}

func (h *FavoritesHandlers) AddFavorite(c *gin.Context) {
	id := c.Param("id")
	user := middleware.GetUserIDFromContext(c)
	
	logger.Log.Info("Adding place to favorites",
		zap.String("place_id", id),
		zap.String("user", user),
	)

	// In real app, add to database
	// For now, just return success response
	
	c.HTML(http.StatusOK, "", `
		<button 
			hx-delete="/favorites/`+id+`"
			hx-target="this"
			hx-swap="outerHTML"
			class="p-2 text-red-500 hover:text-red-600 rounded-lg hover:bg-red-50 dark:hover:bg-red-900/20 transition-colors"
			title="Remove from favorites"
		>
			<svg class="w-5 h-5 fill-current" fill="currentColor" viewBox="0 0 24 24">
				<path d="M20.84 4.61a5.5 5.5 0 0 0-7.78 0L12 5.67l-1.06-1.06a5.5 5.5 0 0 0-7.78 7.78l1.06 1.06L12 21.23l7.78-7.78 1.06-1.06a5.5 5.5 0 0 0 0-7.78z"></path>
			</svg>
		</button>
	`)

	logger.Log.Info("Successfully added to favorites",
		zap.String("place_id", id),
		zap.String("user", user),
	)
}

func (h *FavoritesHandlers) RemoveFavorite(c *gin.Context) {
	id := c.Param("id")
	user := middleware.GetUserIDFromContext(c)
	
	logger.Log.Info("Removing place from favorites",
		zap.String("place_id", id),
		zap.String("user", user),
	)

	// In real app, remove from database
	
	c.HTML(http.StatusOK, "", `
		<button 
			hx-post="/favorites/add/`+id+`"
			hx-target="this"
			hx-swap="outerHTML"
			class="p-2 text-muted-foreground hover:text-red-500 rounded-lg hover:bg-red-50 dark:hover:bg-red-900/20 transition-colors"
			title="Add to favorites"
		>
			<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M20.84 4.61a5.5 5.5 0 0 0-7.78 0L12 5.67l-1.06-1.06a5.5 5.5 0 0 0-7.78 7.78l1.06 1.06L12 21.23l7.78-7.78 1.06-1.06a5.5 5.5 0 0 0 0-7.78z"></path>
			</svg>
		</button>
	`)

	logger.Log.Info("Successfully removed from favorites",
		zap.String("place_id", id),
		zap.String("user", user),
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