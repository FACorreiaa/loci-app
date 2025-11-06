package activities

import (
	"context"
	"strings"

	"github.com/a-h/templ"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	llmchat "github.com/FACorreiaa/go-templui/internal/app/domain/chat_prompt"
	results2 "github.com/FACorreiaa/go-templui/internal/app/domain/results"
	"github.com/FACorreiaa/go-templui/internal/app/models"
	"github.com/FACorreiaa/go-templui/internal/app/services"
	"github.com/FACorreiaa/go-templui/internal/pkg/middleware"
)

type ActivitiesHandlers struct {
	chatRepo         llmchat.Repository
	itineraryService *services.ItineraryService
	logger           *zap.Logger
}

func NewActivitiesHandlers(chatRepo llmchat.Repository, logger *zap.Logger) *ActivitiesHandlers {
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

	h.logger.Info("Activities page accessed",
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
		return ActivitiesPageWithQuery(query)
	}

	// Default activities page
	return ActivitiesPage()
}

// loadActivitiesBySession loads activities using IDENTICAL logic as itinerary
func (h *ActivitiesHandlers) loadActivitiesBySession(sessionIDParam string, cacheKey string) templ.Component {
	h.logger.Info("Attempting to load activities from cache",
		zap.String("sessionID", sessionIDParam),
		zap.String("cacheKey", cacheKey))

	// Try activities cache first with cacheKey (for reusable cache hits)
	if cacheKey != "" {
		if activitiesData, found := middleware.ActivitiesCache.Get(cacheKey); found {
			h.logger.Info("Activities found in cache. Rendering results with data.",
				zap.Int("activities", len(activitiesData)))

			// Try to get city data from complete cache
			var cityData models.GeneralCityData
			if completeData, found := middleware.CompleteItineraryCache.Get(cacheKey); found {
				cityData = completeData.GeneralCityData
				h.logger.Info("City data loaded from complete cache",
					zap.String("city", cityData.City))
			} else {
				// Fallback: load from database using sessionID
				cityData = h.loadCityDataFromDatabase(sessionIDParam)
			}

			// Return static template when data is available
			return results2.ActivitiesResults(
				cityData,
				activitiesData,
				true, true, 15, []string{}, sessionIDParam)
		}
	}

	// Load from database (IDENTICAL to itinerary logic)
	return h.loadActivitiesFromDatabase(sessionIDParam)
}

// loadActivitiesFromDatabase loads activities from database when not found in cache
func (h *ActivitiesHandlers) loadActivitiesFromDatabase(sessionIDParam string) templ.Component {
	h.logger.Info("Activities not found in cache, attempting to load from database", zap.String("sessionID", sessionIDParam))

	// Parse sessionID as UUID
	sessionID, err := uuid.Parse(sessionIDParam)
	if err != nil {
		h.logger.Warn("Invalid session ID format", zap.String("sessionID", sessionIDParam), zap.Error(err))
		return results2.PageNotFound("Invalid session ID")
	}

	// Get the latest interaction for this session from database
	ctx := context.Background()
	interaction, err := h.chatRepo.GetLatestInteractionBySessionID(ctx, sessionID)
	if err != nil || interaction == nil {
		h.logger.Warn("No interaction found in database for session",
			zap.String("sessionID", sessionIDParam),
			zap.Error(err))
		// Return empty results instead of PageNotFound - data might still be processing
		emptyCityData := models.GeneralCityData{}
		emptyActivities := []models.POIDetailedInfo{}
		return results2.ActivitiesResults(emptyCityData, emptyActivities, true, true, 5, []string{}, sessionIDParam)
	}

	// Parse the stored response as complete data
	completeData, err := h.itineraryService.ParseCompleteItineraryResponse(interaction.ResponseText, h.logger)
	if err != nil || completeData == nil {
		h.logger.Warn("Could not parse complete data from stored response",
			zap.String("sessionID", sessionIDParam),
			zap.Error(err))
		// Return empty results instead of PageNotFound for parsing errors
		emptyCityData := models.GeneralCityData{}
		emptyActivities := []models.POIDetailedInfo{}
		return results2.ActivitiesResults(emptyCityData, emptyActivities, true, true, 5, []string{}, sessionIDParam)
	}

	h.logger.Info("Successfully loaded complete data from database for activities",
		zap.String("sessionID", sessionIDParam),
		zap.String("city", completeData.GeneralCityData.City),
		zap.Int("totalPOIs", len(completeData.PointsOfInterest)))

	// Filter POIs for activities and render (IDENTICAL to itinerary results pattern)
	activityPOIs := filterPOIsForActivities(completeData.PointsOfInterest)
	return results2.ActivitiesResults(
		completeData.GeneralCityData,
		activityPOIs,
		true, true, 5, []string{}, sessionIDParam)
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

	h.logger.Info("Activities SSE page accessed",
		zap.String("ip", c.ClientIP()),
		zap.String("query", query),
		zap.String("sessionId", sessionIDParam),
		zap.String("cacheKey", cacheKey))

	if sessionIDParam == "" {
		h.logger.Info("Direct navigation to /activities SSE. Showing default page.")
		return ActivitiesPage()
	}

	// Load activities data for session with SSE support
	return h.loadActivitiesBySession(sessionIDParam, cacheKey)
}

// loadCityDataFromDatabase loads city data from database by sessionID
func (h *ActivitiesHandlers) loadCityDataFromDatabase(sessionIDParam string) models.GeneralCityData {
	sessionID, err := uuid.Parse(sessionIDParam)
	if err != nil {
		h.logger.Warn("Invalid session ID format when loading city data", zap.String("sessionID", sessionIDParam), zap.Error(err))
		return models.GeneralCityData{}
	}

	ctx := context.Background()
	interaction, err := h.chatRepo.GetLatestInteractionBySessionID(ctx, sessionID)
	if err != nil || interaction == nil {
		h.logger.Warn("No interaction found in database for city data",
			zap.String("sessionID", sessionIDParam),
			zap.Error(err))
		return models.GeneralCityData{}
	}

	completeData, err := h.itineraryService.ParseCompleteItineraryResponse(interaction.ResponseText, h.logger)
	if err != nil || completeData == nil {
		h.logger.Warn("Could not parse complete data from stored response for city data",
			zap.String("sessionID", sessionIDParam),
			zap.Error(err))
		return models.GeneralCityData{}
	}

	h.logger.Info("City data loaded from database",
		zap.String("sessionID", sessionIDParam),
		zap.String("city", completeData.GeneralCityData.City),
		zap.String("country", completeData.GeneralCityData.Country),
		zap.String("description", completeData.GeneralCityData.Description))

	return completeData.GeneralCityData
}
