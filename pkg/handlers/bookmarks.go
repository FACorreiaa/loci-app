package handlers

import (
	"net/http"

	"github.com/FACorreiaa/go-templui/pkg/logger"
	"github.com/FACorreiaa/go-templui/pkg/middleware"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type BookmarksHandlers struct{}

func NewBookmarksHandlers() *BookmarksHandlers {
	return &BookmarksHandlers{}
}

func (h *BookmarksHandlers) AddBookmark(c *gin.Context) {
	id := c.Param("id")
	user := middleware.GetUserIDFromContext(c)
	
	logger.Log.Info("Adding place to bookmarks",
		zap.String("place_id", id),
		zap.String("user", user),
	)

	c.HTML(http.StatusOK, "", `<div class="text-green-500 text-sm">Added to bookmarks!</div>`)

	logger.Log.Info("Successfully added to bookmarks",
		zap.String("place_id", id),
		zap.String("user", user),
	)
}

func (h *BookmarksHandlers) RemoveBookmark(c *gin.Context) {
	id := c.Param("id")
	user := middleware.GetUserIDFromContext(c)
	
	logger.Log.Info("Removing place from bookmarks",
		zap.String("place_id", id),
		zap.String("user", user),
	)

	c.HTML(http.StatusOK, "", ``)

	logger.Log.Info("Successfully removed from bookmarks",
		zap.String("place_id", id),
		zap.String("user", user),
	)
}

func (h *BookmarksHandlers) SearchBookmarks(c *gin.Context) {
	query := c.PostForm("query")
	user := middleware.GetUserIDFromContext(c)
	
	logger.Log.Info("Searching bookmarks",
		zap.String("query", query),
		zap.String("user", user),
	)

	c.HTML(http.StatusOK, "", `<div class="text-center py-8 text-muted-foreground">No bookmarks found for "`+query+`"</div>`)
}