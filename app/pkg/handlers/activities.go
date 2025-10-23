package handlers

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

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

	logger.Log.Info("Activities page accessed",
		zap.String("ip", c.ClientIP()),
		zap.String("query", query),
		zap.String("sessionId", sessionIDParam))

	// Handle session-based loading
	if sessionIDParam != "" {
		return h.loadActivitiesBySession(sessionIDParam)
	}

	// Handle query-based loading
	if query != "" {
		return activities.ActivitiesPageWithQuery(query)
	}

	// Default activities page
	return activities.ActivitiesPage()
}

// loadActivitiesBySession loads activities using IDENTICAL logic as itinerary
func (h *ActivitiesHandlers) loadActivitiesBySession(sessionIDParam string) templ.Component {
	logger.Log.Info("Attempting to load activities from cache", zap.String("sessionID", sessionIDParam))

	// Try complete cache first (IDENTICAL to itinerary logic)
	if completeData, found := middleware.CompleteItineraryCache.Get(sessionIDParam); found {

		logger.Log.Info("Complete activities found in cache. Rendering results.",
			zap.String("city", completeData.GeneralCityData.City),
			zap.Int("generalPOIs", len(completeData.PointsOfInterest)),
			zap.Int("personalizedPOIs", len(completeData.AIItineraryResponse.PointsOfInterest)))

		// Filter POIs for activities and render (IDENTICAL to itinerary results pattern)
		activityPOIs := filterPOIsForActivities(completeData.PointsOfInterest)
		return results.ActivitiesResults(
			completeData.GeneralCityData,
			activityPOIs,
			true, true, 5, []string{})
	}

	//// Try legacy cache (IDENTICAL to itinerary logic)
	//if itineraryData, found := middleware.ItineraryCache.Get(sessionIDParam); found {
	//	logger.Log.Info("Legacy activities found in cache. Rendering results.",
	//		zap.Int("personalizedPOIs", len(itineraryData.PointsOfInterest)))
	//
	//	// Create empty city data for legacy cached data (IDENTICAL to itinerary)
	//	emptyCityData := models.GeneralCityData{}
	//
	//	// Filter activities from legacy data
	//	activityPOIs := filterPOIsForActivities(itineraryData.PointsOfInterest)
	//	return results.ActivitiesResults(emptyCityData, activityPOIs, true, true, 5, []string{})
	//}

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

	logger.Log.Info("Activities SSE page accessed",
		zap.String("ip", c.ClientIP()),
		zap.String("query", query),
		zap.String("sessionId", sessionIDParam))

	if sessionIDParam == "" {
		logger.Log.Info("Direct navigation to /activities SSE. Showing default page.")
		return activities.ActivitiesPage()
	}

	// Load activities data for session with SSE support
	return h.loadActivitiesBySessionSSE(sessionIDParam)
}

// loadActivitiesBySessionSSE loads activities with SSE support
func (h *ActivitiesHandlers) loadActivitiesBySessionSSE(sessionIDParam string) templ.Component {
	logger.Log.Info("Attempting to load activities from cache with SSE", zap.String("sessionID", sessionIDParam))

	// Try complete cache first
	if completeData, found := middleware.CompleteItineraryCache.Get(sessionIDParam); found {
		// Filter POIs for activities
		activityPOIs := filterPOIsForActivities(completeData.PointsOfInterest)

		// Print JSON data for debugging what activities will be displayed
		jsonData, err := json.MarshalIndent(map[string]interface{}{
			"city_data":     completeData.GeneralCityData,
			"activity_pois": activityPOIs,
		}, "", "  ")
		if err != nil {
			logger.Log.Error("Failed to marshal activities data to JSON", zap.Error(err))
		} else {
			logger.Log.Info("Activities data being displayed in view", zap.String("json", string(jsonData)))
		}

		logger.Log.Info("Complete activities found in cache. Rendering SSE results with data.",
			zap.String("city", completeData.GeneralCityData.City),
			zap.Int("totalPOIs", len(completeData.PointsOfInterest)),
			zap.Int("activityPOIs", len(activityPOIs)))

		return results.ActivitiesResultsSSE(
			sessionIDParam,
			completeData.GeneralCityData,
			activityPOIs,
			true) // hasData = true
	}

	// Try legacy cache
	if itineraryData, found := middleware.ItineraryCache.Get(sessionIDParam); found {
		logger.Log.Info("Legacy activities found in cache. Rendering SSE results with data.",
			zap.Int("personalizedPOIs", len(itineraryData.PointsOfInterest)))

		// Create empty city data for legacy cached data
		emptyCityData := models.GeneralCityData{}

		// Filter activities from legacy data
		activityPOIs := filterPOIsForActivities(itineraryData.PointsOfInterest)

		return results.ActivitiesResultsSSE(
			sessionIDParam,
			emptyCityData,
			activityPOIs,
			true) // hasData = true
	}

	// No cached data found - show loading interface with SSE
	logger.Log.Info("No activities found in cache. Rendering SSE loading interface.",
		zap.String("sessionID", sessionIDParam))

	emptyCityData := models.GeneralCityData{}
	emptyActivities := []models.POIDetailedInfo{}

	return results.ActivitiesResultsSSE(
		sessionIDParam,
		emptyCityData,
		emptyActivities,
		false) // hasData = false, will show loading and connect to SSE
}

// HandleActivitiesSSE handles Server-Sent Events for activity updates
func (h *ActivitiesHandlers) HandleActivitiesSSE(c *gin.Context) {
	sessionID := c.Query("sessionId")
	if sessionID == "" {
		c.Status(http.StatusBadRequest)
		return
	}

	logger.Log.Info("SSE connection established for activities",
		zap.String("sessionId", sessionID))

	// Set SSE headers
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")

	// Create a channel for updates
	updateChan := make(chan models.ItinerarySSEEvent)
	defer close(updateChan)

	// Start monitoring for updates in a separate goroutine
	go h.monitorActivityUpdates(sessionID, updateChan)

	// Stream updates to client
	flusher := c.Writer.(http.Flusher)
	for {
		select {
		case event := <-updateChan:
			if event.Type == "complete" {
				logger.Log.Info("Sending activities completion event",
					zap.String("sessionId", sessionID))

				// Send final completion event
				c.SSEvent("activities-complete", map[string]interface{}{
					"sessionId": sessionID,
					"message":   "Activity search complete",
				})
				flusher.Flush()
				return
			}

			// Send HTML fragment updates based on event type
			switch event.Type {
			case "header-update":
				if data, ok := event.Data.(map[string]interface{}); ok {
					if completeData, exists := data["completeData"]; exists {
						if complete, valid := completeData.(models.AiCityResponse); valid {
							activityPOIs := filterPOIsForActivities(complete.PointsOfInterest)
							headerHTML := h.renderActivitiesHeaderHTML(complete.GeneralCityData, activityPOIs)
							c.SSEvent("activities-header", headerHTML)
						}
					}
				}
			case "content-update":
				if data, ok := event.Data.(map[string]interface{}); ok {
					if completeData, exists := data["completeData"]; exists {
						if complete, valid := completeData.(models.AiCityResponse); valid {
							activityPOIs := filterPOIsForActivities(complete.PointsOfInterest)
							contentHTML := h.renderActivitiesContentHTML(complete.GeneralCityData, activityPOIs)
							c.SSEvent("activities-content", contentHTML)
						}
					}
				}
			default:
				// Send progress update
				c.SSEvent(event.Type, event.Data)
			}
			flusher.Flush()

		case <-c.Request.Context().Done():
			logger.Log.Info("SSE connection closed",
				zap.String("sessionId", sessionID))
			return
		}
	}
}

// monitorActivityUpdates monitors for activity updates and sends SSE events
func (h *ActivitiesHandlers) monitorActivityUpdates(sessionID string, updateChan chan<- models.ItinerarySSEEvent) {
	// Check for cached data first
	if completeData, found := middleware.CompleteItineraryCache.Get(sessionID); found {
		logger.Log.Info("Complete data found in cache, sending activities completion immediately",
			zap.String("sessionId", sessionID))

		// Send header update
		updateChan <- models.ItinerarySSEEvent{
			Type: "header-update",
			Data: map[string]interface{}{
				"sessionId":    sessionID,
				"completeData": completeData,
			},
		}

		// Send content update
		updateChan <- models.ItinerarySSEEvent{
			Type: "content-update",
			Data: map[string]interface{}{
				"sessionId":    sessionID,
				"completeData": completeData,
			},
		}

		// Send completion
		updateChan <- models.ItinerarySSEEvent{
			Type: "complete",
			Data: map[string]interface{}{
				"sessionId": sessionID,
				"totalPOIs": len(completeData.PointsOfInterest),
			},
		}
		return
	}

	// Legacy cache check
	if itineraryData, found := middleware.ItineraryCache.Get(sessionID); found {
		logger.Log.Info("Legacy data found in cache, sending activities completion immediately",
			zap.String("sessionId", sessionID))

		updateChan <- models.ItinerarySSEEvent{
			Type: "complete",
			Data: map[string]interface{}{
				"sessionId":        sessionID,
				"personalizedPOIs": len(itineraryData.PointsOfInterest),
			},
		}
		return
	}

	// If no cached data, poll for updates
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	timeout := time.After(5 * time.Minute) // 5 minute timeout

	for {
		select {
		case <-ticker.C:
			// Check cache again
			if completeData, found := middleware.CompleteItineraryCache.Get(sessionID); found {
				logger.Log.Info("Complete data appeared in cache for activities",
					zap.String("sessionId", sessionID))

				// Send header update
				updateChan <- models.ItinerarySSEEvent{
					Type: "header-update",
					Data: map[string]interface{}{
						"sessionId":    sessionID,
						"completeData": completeData,
					},
				}

				// Send content update
				updateChan <- models.ItinerarySSEEvent{
					Type: "content-update",
					Data: map[string]interface{}{
						"sessionId":    sessionID,
						"completeData": completeData,
					},
				}

				// Send completion
				updateChan <- models.ItinerarySSEEvent{
					Type: "complete",
					Data: map[string]interface{}{
						"sessionId": sessionID,
						"totalPOIs": len(completeData.PointsOfInterest),
					},
				}
				return
			}

			if itineraryData, found := middleware.ItineraryCache.Get(sessionID); found {
				logger.Log.Info("Legacy data appeared in cache for activities",
					zap.String("sessionId", sessionID))

				updateChan <- models.ItinerarySSEEvent{
					Type: "complete",
					Data: map[string]interface{}{
						"sessionId":        sessionID,
						"personalizedPOIs": len(itineraryData.PointsOfInterest),
					},
				}
				return
			}

			// Send progress update
			updateChan <- models.ItinerarySSEEvent{
				Type: "progress",
				Data: map[string]interface{}{
					"sessionId": sessionID,
					"message":   "Finding activities...",
					"timestamp": time.Now().Unix(),
				},
			}

		case <-timeout:
			logger.Log.Warn("SSE monitoring timed out for activities", zap.String("sessionId", sessionID))
			updateChan <- models.ItinerarySSEEvent{
				Type: "error",
				Data: map[string]interface{}{
					"sessionId": sessionID,
					"message":   "Request timed out. Please try again.",
				},
			}
			return
		}
	}
}

// renderActivitiesHeaderHTML renders header HTML fragment for SSE
func (h *ActivitiesHandlers) renderActivitiesHeaderHTML(cityData models.GeneralCityData, activities []models.POIDetailedInfo) string {
	buf := &strings.Builder{}
	component := results.ActivitiesHeaderComplete(cityData, activities)
	err := component.Render(context.Background(), buf)
	if err != nil {
		h.logger.Error("failed to render activities header", slog.Any("error", err))
	}
	return buf.String()
}

// renderActivitiesContentHTML renders content HTML fragment for SSE
func (h *ActivitiesHandlers) renderActivitiesContentHTML(cityData models.GeneralCityData, activities []models.POIDetailedInfo) string {
	buf := &strings.Builder{}
	component := results.ActivitiesContentComplete(cityData, activities)
	err := component.Render(context.Background(), buf)
	if err != nil {
		h.logger.Error("failed to render activities content", slog.Any("error", err))
	}
	return buf.String()
}
