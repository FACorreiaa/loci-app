package handlers

import (
	"context"
	"log/slog"
	"strings"

	"github.com/a-h/templ"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/FACorreiaa/go-templui/app/internal/features/activities"
	"github.com/FACorreiaa/go-templui/app/internal/features/results"
	"github.com/FACorreiaa/go-templui/app/internal/models"
	llmchat "github.com/FACorreiaa/go-templui/app/pkg/domain/chat_prompt"
	"github.com/FACorreiaa/go-templui/app/pkg/logger"
	"github.com/FACorreiaa/go-templui/app/pkg/middleware"
	"github.com/FACorreiaa/go-templui/app/pkg/services"
)

type ActivitiesHandlers struct {
	chatRepo         llmchat.Repository
	itineraryService *services.ItineraryService
	logger           *slog.Logger
}

func NewActivitiesHandlers(chatRepo llmchat.Repository, logger *slog.Logger) *ActivitiesHandlers {
	return &ActivitiesHandlers{
		chatRepo:         chatRepo,
		itineraryService: services.NewItineraryService(),
		logger:           logger,
	}
}

// HandleActivitiesPage handles the main activities page logic
func (h *ActivitiesHandlers) HandleActivitiesPage(c *gin.Context) templ.Component {
	query := c.Query("q")
	sessionIDParam := c.Query("sessionId")
	cacheKey := c.Query("cacheKey")

	logger.Log.Info("Activities page accessed",
		zap.String("ip", c.ClientIP()),
		zap.String("query", query),
		zap.String("sessionId", sessionIDParam),
		zap.String("cacheKey", cacheKey))

	// Handle session-based loading
	if sessionIDParam != "" {
		return h.loadActivitiesBySession(sessionIDParam, cacheKey)
	}

	// Handle query-based loading
	if query != "" {
		return activities.ActivitiesPageWithQuery(query)
	}

	// Default activities page
	return activities.ActivitiesPage()
}

// loadActivitiesBySession loads activities using IDENTICAL logic as itinerary
func (h *ActivitiesHandlers) loadActivitiesBySession(sessionIDParam string, cacheKey string) templ.Component {
	logger.Log.Info("Attempting to load activities from cache",
		zap.String("sessionID", sessionIDParam),
		zap.String("cacheKey", cacheKey))

	// Try activities cache first with cacheKey (for reusable cache hits)
	if cacheKey != "" {
		if activitiesData, found := middleware.ActivitiesCache.Get(cacheKey); found {
			logger.Log.Info("Activities found in cache. Rendering results with data.",
				zap.Int("activities", len(activitiesData)))

			// Create empty city data for now
			emptyCityData := models.GeneralCityData{}

			// Return static template when data is available
			return results.ActivitiesResults(
				emptyCityData,
				activitiesData,
				true, true, 15, []string{})
		}
	}

	// Load from database (IDENTICAL to itinerary logic)
	return h.loadActivitiesFromDatabase(sessionIDParam)
}

// loadActivitiesFromDatabase loads activities from database when not found in cache
func (h *ActivitiesHandlers) loadActivitiesFromDatabase(sessionIDParam string) templ.Component {
	logger.Log.Info("Activities not found in cache, attempting to load from database", zap.String("sessionID", sessionIDParam))

	// Parse sessionID as UUID
	sessionID, err := uuid.Parse(sessionIDParam)
	if err != nil {
		logger.Log.Warn("Invalid session ID format", zap.String("sessionID", sessionIDParam), zap.Error(err))
		return results.PageNotFound("Invalid session ID")
	}

	// Get the latest interaction for this session from database
	ctx := context.Background()
	interaction, err := h.chatRepo.GetLatestInteractionBySessionID(ctx, sessionID)
	if err != nil || interaction == nil {
		logger.Log.Warn("No interaction found in database for session",
			zap.String("sessionID", sessionIDParam),
			zap.Error(err))
		// Return empty results instead of PageNotFound - data might still be processing
		emptyCityData := models.GeneralCityData{}
		emptyActivities := []models.POIDetailedInfo{}
		return results.ActivitiesResults(emptyCityData, emptyActivities, true, true, 5, []string{})
	}

	// Parse the stored response as complete data
	completeData, err := h.itineraryService.ParseCompleteItineraryResponse(interaction.ResponseText, slog.Default())
	if err != nil || completeData == nil {
		logger.Log.Warn("Could not parse complete data from stored response",
			zap.String("sessionID", sessionIDParam),
			zap.Error(err))
		// Return empty results instead of PageNotFound for parsing errors
		emptyCityData := models.GeneralCityData{}
		emptyActivities := []models.POIDetailedInfo{}
		return results.ActivitiesResults(emptyCityData, emptyActivities, true, true, 5, []string{})
	}

	logger.Log.Info("Successfully loaded complete data from database for activities",
		zap.String("sessionID", sessionIDParam),
		zap.String("city", completeData.GeneralCityData.City),
		zap.Int("totalPOIs", len(completeData.PointsOfInterest)))

	// Filter POIs for activities and render (IDENTICAL to itinerary results pattern)
	activityPOIs := filterPOIsForActivities(completeData.PointsOfInterest)
	return results.ActivitiesResults(
		completeData.GeneralCityData,
		activityPOIs,
		true, true, 5, []string{})
}

// filterPOIsForActivities filters POIs to show only activity-related categories
func filterPOIsForActivities(allPOIs []models.POIDetailedInfo) []models.POIDetailedInfo {
	var activityPOIs []models.POIDetailedInfo
	activityCategories := map[string]bool{
		"activity":      true,
		"attraction":    true,
		"entertainment": true,
		"park":          true,
		"museum":        true,
		"gallery":       true,
		"theater":       true,
		"sports":        true,
		"outdoor":       true,
		"adventure":     true,
		"cultural":      true,
		"recreation":    true,
	}

	for _, poi := range allPOIs {
		if activityCategories[strings.ToLower(poi.Category)] {
			activityPOIs = append(activityPOIs, poi)
		}
	}

	return activityPOIs
}

// HandleActivitiesPageSSE handles the activities page with SSE support
func (h *ActivitiesHandlers) HandleActivitiesPageSSE(c *gin.Context) templ.Component {
	query := c.Query("q")
	sessionIDParam := c.Query("sessionId")
	cacheKey := c.Query("cacheKey")

	logger.Log.Info("Activities SSE page accessed",
		zap.String("ip", c.ClientIP()),
		zap.String("query", query),
		zap.String("sessionId", sessionIDParam),
		zap.String("cacheKey", cacheKey))

	if sessionIDParam == "" {
		logger.Log.Info("Direct navigation to /activities SSE. Showing default page.")
		return activities.ActivitiesPage()
	}

	// Load activities data for session with SSE support
	return h.loadActivitiesBySession(sessionIDParam, cacheKey)
}
