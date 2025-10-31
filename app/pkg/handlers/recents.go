package handlers

import (
	"net/http"
	"strconv"

	"github.com/FACorreiaa/go-templui/app/internal/features/recents"
	"github.com/FACorreiaa/go-templui/app/internal/models"
	"github.com/FACorreiaa/go-templui/app/internal/pages"
	recentsService "github.com/FACorreiaa/go-templui/app/pkg/domain/recents"
	"github.com/FACorreiaa/go-templui/app/pkg/middleware"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type RecentsHandlers struct {
	recentsService recentsService.Service
	logger         *zap.Logger
}

func NewRecentsHandlers(recentsService recentsService.Service, logger *zap.Logger) *RecentsHandlers {
	return &RecentsHandlers{
		recentsService: recentsService,
		logger:         logger,
	}
}

// HandleRecentsPage renders the recents page with user's recent interactions
func (h *RecentsHandlers) HandleRecentsPage(c *gin.Context) {
	userID := middleware.GetUserIDFromContext(c)
	if userID == "" {
		c.Redirect(http.StatusSeeOther, "/auth/login")
		return
	}

	userUUID, err := uuid.Parse(userID)
	if err != nil {
		h.logger.Error("Invalid user ID", zap.String("user_id", userID), zap.Error(err))
		c.HTML(http.StatusBadRequest, "", recents.RecentsPage(nil, models.RecentInteractionsFilter{}))
		return
	}

	// Get query parameters for filtering and pagination
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	sortBy := c.DefaultQuery("sort_by", "last_activity")
	sortOrder := c.DefaultQuery("sort_order", "desc")
	search := c.Query("search")
	searchType := c.Query("type") // itinerary, restaurant, hotel, activity

	// Build filter options
	filterOptions := &models.RecentInteractionsFilter{
		SortBy:           sortBy,
		SortOrder:        sortOrder,
		Search:           search,
		SearchType:       searchType,
		MinInteractions:  -1,
		MaxInteractions:  -1,
	}

	h.logger.Info("Fetching recent interactions",
		zap.String("user_id", userID),
		zap.Int("page", page),
		zap.Int("limit", limit),
		zap.String("sort_by", sortBy),
		zap.String("sort_order", sortOrder),
		zap.String("search", search),
		zap.String("type", searchType))

	// Get recent interactions from service
	response, err := h.recentsService.GetUserRecentInteractions(c.Request.Context(), userUUID, page, limit, filterOptions)
	if err != nil {
		h.logger.Error("Failed to get recent interactions", zap.String("user_id", userID), zap.Error(err))
		c.HTML(http.StatusInternalServerError, "", recents.RecentsPage(nil, *filterOptions))
		return
	}

	h.logger.Info("Successfully retrieved recent interactions",
		zap.String("user_id", userID),
		zap.Int("total_cities", response.Total),
		zap.Int("cities_returned", len(response.Cities)))

	// Get user from context for layout
	user := getUserFromContext(c)

	c.HTML(http.StatusOK, "", pages.LayoutPage(models.LayoutTempl{
		Title:   "Recent Activity - Loci",
		Content: recents.RecentsPage(response, *filterOptions),
		Nav: models.Navigation{
			Items: []models.NavItem{
				{Name: "Dashboard", URL: "/dashboard"},
				{Name: "Recents", URL: "/recents"},
				{Name: "Settings", URL: "/settings"},
			},
		},
		ActiveNav: "Recents",
		User:      user,
	}))
}

// Helper function to get user from context
func getUserFromContext(c *gin.Context) *models.User {
	userID, userEmail, userName := middleware.GetUserFromContext(c)
	if userID == "anonymous" {
		return nil
	}
	return &models.User{
		ID:       userID,
		Name:     userName,
		Email:    userEmail,
		IsActive: true,
	}
}
