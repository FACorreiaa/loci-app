package interests

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/a-h/templ"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/FACorreiaa/go-templui/internal/app/domain/itinerary"
	results2 "github.com/FACorreiaa/go-templui/internal/app/domain/results"
	"github.com/FACorreiaa/go-templui/internal/app/middleware"
	"github.com/FACorreiaa/go-templui/internal/app/models"
	"github.com/FACorreiaa/go-templui/internal/app/services"
	"github.com/FACorreiaa/go-templui/internal/pkg/cache"
)

// ChatRepository defines the minimal interface needed from chat repository
type ChatRepository interface {
	GetLatestInteractionBySessionID(ctx context.Context, sessionID uuid.UUID) (*models.LlmInteraction, error)
}

type ItineraryHandlers struct {
	chatRepo         ChatRepository
	itineraryService *services.ItineraryService
	logger           *zap.Logger
}

func NewItineraryHandlers(chatRepo ChatRepository,
	itineraryService *services.ItineraryService,
	logger *zap.Logger) *ItineraryHandlers {
	return &ItineraryHandlers{
		chatRepo:         chatRepo,
		itineraryService: itineraryService,
		logger:           logger,
	}
}

func (h *ItineraryHandlers) HandleDestination(c *gin.Context) {
	destination := c.PostForm("destination-input")
	if destination == "" {
		c.HTML(http.StatusOK, "", itinerary.ItineraryDestinationSuggestions([]string{}))
		return
	}

	h.logger.Info("Destination search request",
		zap.String("destination", destination),
	)

	// Mock destination suggestions
	suggestions := h.getDestinationSuggestions(destination)
	c.HTML(http.StatusOK, "", itinerary.ItineraryDestinationSuggestions(suggestions))
}

func (h *ItineraryHandlers) HandleChat(c *gin.Context) {
	message := c.PostForm("message")
	destination := c.PostForm("destination-input")
	duration := c.DefaultPostForm("duration", "5")
	budget := c.DefaultPostForm("budget", "moderate")
	style := c.DefaultPostForm("style", "relaxation")

	h.logger.Info("Itinerary chat request",
		zap.String("message", message),
		zap.String("destination", destination),
		zap.String("duration", duration),
		zap.String("budget", budget),
		zap.String("style", style),
	)

	if message == "" {
		c.Status(http.StatusBadRequest)
		return
	}

	// Create user message
	userMessage := models.ChatMessage{
		Content:   message,
		Timestamp: time.Now().Format("3:04 PM"),
	}

	// Generate AI response
	aiResponse := h.generateItineraryResponse(message, destination, duration, budget, style)
	aiMessage := models.ChatMessage{
		Content:   aiResponse,
		Timestamp: time.Now().Format("3:04 PM"),
	}

	c.HTML(http.StatusOK, "", templ.Raw(
		fmt.Sprintf(`%s%s`,
			renderChatMessage(userMessage, true),
			renderChatMessage(aiMessage, false),
		),
	))
}

func (h *ItineraryHandlers) AddPOI(c *gin.Context) {
	poiID := c.Param("id")

	h.logger.Info("Add POI to itinerary",
		zap.String("poi_id", poiID),
	)

	// Mock response
	c.HTML(http.StatusOK, "", templ.Raw(
		`<div class="text-green-600 text-sm">
			<i class="fas fa-check mr-1"></i>
			Added to your itinerary!
		</div>`,
	))
}

func (h *ItineraryHandlers) RemovePOI(c *gin.Context) {
	poiID := c.Param("id")

	h.logger.Info("Remove POI from itinerary",
		zap.String("poi_id", poiID),
	)

	c.Status(http.StatusOK)
}

func (h *ItineraryHandlers) GetItinerarySummary(c *gin.Context) {
	// Mock itinerary data
	day1 := []models.NearbyPOI{
		{
			ID:       "poi_1",
			Name:     "Eiffel Tower",
			Category: "Attraction",
			Rating:   4.8,
		},
		{
			ID:       "poi_2",
			Name:     "Louvre Museum",
			Category: "Museum",
			Rating:   4.7,
		},
	}

	c.HTML(http.StatusOK, "", itinerary.ItinerarySummaryItem(1, day1))
}

func (h *ItineraryHandlers) getDestinationSuggestions(query string) []string {
	// Mock destination suggestions based on query
	allDestinations := []string{
		"Paris, France",
		"Tokyo, Japan",
		"New York City, USA",
		"London, England",
		"Rome, Italy",
		"Barcelona, Spain",
		"Amsterdam, Netherlands",
		"Prague, Czech Republic",
		"Vienna, Austria",
		"Budapest, Hungary",
		"Bangkok, Thailand",
		"Singapore",
		"Hong Kong",
		"Sydney, Australia",
		"Melbourne, Australia",
	}

	suggestions := []string{}
	queryLower := strings.ToLower(query)

	for _, dest := range allDestinations {
		if strings.Contains(strings.ToLower(dest), queryLower) {
			suggestions = append(suggestions, dest)
		}
	}

	// Limit to 5 suggestions
	if len(suggestions) > 5 {
		suggestions = suggestions[:5]
	}

	return suggestions
}

func (h *ItineraryHandlers) generateItineraryResponse(message, destination, duration, budget, style string) string {
	messageLower := strings.ToLower(message)

	// Generate contextual responses based on message content
	if strings.Contains(messageLower, "itinerary") || strings.Contains(messageLower, "plan") {
		if destination == "" {
			return "I'd love to help you create an itinerary! First, please let me know where you'd like to travel."
		}
		return fmt.Sprintf(
			"Great! I'll create a %s-day %s itinerary for %s. Based on your %s budget and %s style preferences, here are my recommendations:\n\n"+
				"🗓️ **Day 1**: Arrival and city center exploration\n"+
				"🏛️ **Day 2**: Major attractions and cultural sites\n"+
				"🍽️ **Day 3**: Local cuisine and neighborhood discovery\n\n"+
				"Would you like me to add specific places or activities to any of these days?",
			duration, budget, destination, budget, style,
		)
	}

	if strings.Contains(messageLower, "restaurant") || strings.Contains(messageLower, "food") || strings.Contains(messageLower, "eat") {
		if destination == "" {
			return "I'd be happy to recommend restaurants! Where are you planning to travel?"
		}
		return fmt.Sprintf(
			"Here are some excellent restaurant recommendations for %s based on your %s budget:\n\n"+
				"🍴 **Fine Dining**: Michelin-starred establishments\n"+
				"🥘 **Local Cuisine**: Authentic traditional dishes\n"+
				"☕ **Casual Dining**: Popular local favorites\n\n"+
				"Would you like me to add any specific restaurants to your itinerary, or do you have dietary preferences I should consider?",
			destination, budget,
		)
	}

	if strings.Contains(messageLower, "hotel") || strings.Contains(messageLower, "accommodation") || strings.Contains(messageLower, "stay") {
		if destination == "" {
			return "I can help you find great accommodations! Which city are you visiting?"
		}
		return fmt.Sprintf(
			"Here are some accommodation options in %s for your %s budget:\n\n"+
				"🏨 **Luxury Hotels**: Premium service and amenities\n"+
				"🏩 **Boutique Hotels**: Unique character and local charm\n"+
				"🏠 **Alternative Stays**: Apartments and unique properties\n\n"+
				"What area of the city would you prefer to stay in?",
			destination, budget,
		)
	}

	if strings.Contains(messageLower, "attraction") || strings.Contains(messageLower, "sightseeing") || strings.Contains(messageLower, "visit") {
		if destination == "" {
			return "I'd love to suggest attractions! What destination are you planning to visit?"
		}
		return fmt.Sprintf(
			"Here are must-see attractions in %s perfect for %s travel:\n\n"+
				"🎨 **Cultural Sites**: Museums, galleries, and historic landmarks\n"+
				"🌳 **Outdoor Attractions**: Parks, gardens, and scenic viewpoints\n"+
				"🎭 **Entertainment**: Shows, tours, and local experiences\n\n"+
				"Which types of attractions interest you most?",
			destination, style,
		)
	}

	// Default response
	if destination != "" {
		return fmt.Sprintf(
			"I understand you're planning a trip to %s! I can help you with:\n\n"+
				"📅 Creating detailed daily itineraries\n"+
				"🍽️ Finding amazing restaurants\n"+
				"🏨 Recommending accommodations\n"+
				"🎯 Suggesting attractions and activities\n\n"+
				"What would you like to explore first?",
			destination,
		)
	}

	return "I'm here to help you plan an amazing trip! Please let me know your destination and I'll create a personalized itinerary for you. You can also ask me about specific aspects like restaurants, hotels, or attractions."
}

// Helper function to render chat messages (temporary until proper template is set up)
func renderChatMessage(message models.ChatMessage, isUser bool) string {
	userClass := ""
	bgClass := "bg-gray-100"
	textClass := "text-gray-800"
	iconClass := "fa-robot"
	iconBg := "bg-blue-600"

	if isUser {
		userClass = "flex-row-reverse space-x-reverse"
		bgClass = "bg-blue-600 text-white"
		textClass = ""
		iconClass = "fa-user"
		iconBg = "bg-gray-600"
	}

	timestamp := ""
	if message.Timestamp != "" {
		timestampClass := "text-gray-500"
		if isUser {
			timestampClass = "text-blue-100"
		}
		timestamp = fmt.Sprintf(`<p class="text-xs mt-2 %s">%s</p>`, timestampClass, message.Timestamp)
	}

	return fmt.Sprintf(`
		<div class="flex items-start space-x-3 %s">
			<div class="w-8 h-8 %s rounded-full flex items-center justify-center flex-shrink-0">
				<i class="fas %s text-white text-sm"></i>
			</div>
			<div class="%s rounded-lg p-4 max-w-md">
				<p class="%s">%s</p>
				%s
			</div>
		</div>
	`, userClass, iconBg, iconClass, bgClass, textClass, message.Content, timestamp)
}

// HandleItineraryPage handles the main itinerary page logic
func (h *ItineraryHandlers) HandleItineraryPage(c *gin.Context) templ.Component {
	user := middleware.GetUserFromContext(c)
	if user == nil {
		c.Redirect(http.StatusFound, "/auth/signin")
		return nil
	}

	userID := user.ID
	query := c.Query("q")
	sessionIDParam := c.Query("sessionId")

	h.logger.Info("Itinerary page accessed",
		zap.String("user", userID),
		zap.String("query", query),
		zap.String("sessionId", sessionIDParam))

	if sessionIDParam == "" {
		h.logger.Info("Direct navigation to /itinerary. Showing default page.")
		return itinerary.ItineraryPage()
	}

	// Load itinerary data for session
	return h.loadItineraryBySession(sessionIDParam)
}

// HandleItineraryPageSSE handles the itinerary page with SSE support
func (h *ItineraryHandlers) HandleItineraryPageSSE(c *gin.Context) templ.Component {
	user := middleware.GetUserFromContext(c)
	if user == nil {
		c.Redirect(http.StatusFound, "/auth/signin")
		return nil
	}

	userID := user.ID
	query := c.Query("q")
	sessionIDParam := c.Query("sessionId")
	cacheKey := c.Query("cacheKey")

	h.logger.Info("Itinerary SSE page accessed",
		zap.String("user", userID),
		zap.String("query", query),
		zap.String("sessionId", sessionIDParam),
		zap.String("cacheKey", cacheKey))

	if sessionIDParam == "" {
		h.logger.Info("Direct navigation to /itinerary SSE. Showing default page.")
		return itinerary.ItineraryPage()
	}

	// Load itinerary data for session with SSE support
	return h.loadItineraryBySessionSSE(sessionIDParam, cacheKey)
}

func (h *ItineraryHandlers) loadItineraryBySession(sessionIDParam string) templ.Component {
	h.logger.Info("Attempting to load itinerary from cache", zap.String("sessionID", sessionIDParam))

	// Try complete cache first
	if completeData, found := cache.CompleteItineraryCache.Get(sessionIDParam); found {
		jsonData, err := json.MarshalIndent(completeData, "", "  ")
		if err != nil {
			h.logger.Error("Failed to marshal completeData to JSON", zap.Error(err))
		} else {
			h.logger.Info("Complete itinerary JSON structure", zap.String("json", string(jsonData)))
		}

		if err := os.WriteFile("completeData.json", jsonData, 0644); err != nil {
			h.logger.Error("Failed to write JSON to file", zap.Error(err))
		}
		h.logger.Info("Complete itinerary found in cache. Rendering results.",
			zap.String("city", completeData.GeneralCityData.City),
			zap.Int("generalPOIs", len(completeData.PointsOfInterest)),
			zap.Int("personalizedPOIs", len(completeData.AIItineraryResponse.PointsOfInterest)))

		return results2.ItineraryResults(
			completeData.GeneralCityData,
			completeData.PointsOfInterest,
			completeData.AIItineraryResponse,
			true, true, 15, []string{}, sessionIDParam)
	}

	// Try legacy cache
	if itineraryData, found := cache.ItineraryCache.Get(sessionIDParam); found {
		h.logger.Info("Legacy itinerary found in cache. Rendering results.",
			zap.Int("personalizedPOIs", len(itineraryData.PointsOfInterest)))

		// Create empty city data and general POIs for legacy cached data
		emptyCityData := models.GeneralCityData{}
		emptyGeneralPOIs := []models.POIDetailedInfo{}

		return results2.ItineraryResults(emptyCityData, emptyGeneralPOIs, itineraryData, true, true, 5, []string{}, sessionIDParam)
	}

	// Load from database
	return h.loadItineraryFromDatabase(sessionIDParam)
}

// loadItineraryBySessionSSE loads itinerary with SSE support
func (h *ItineraryHandlers) loadItineraryBySessionSSE(sessionIDParam string, cacheKey string) templ.Component {
	h.logger.Info("Attempting to load itinerary from cache with SSE",
		zap.String("sessionID", sessionIDParam),
		zap.String("cacheKey", cacheKey))

	// Try complete cache first with cacheKey (for reusable cache hits)
	if cacheKey != "" {
		if completeData, found := cache.CompleteItineraryCache.Get(cacheKey); found {
			h.logger.Info("Complete itinerary found in cache. Rendering SSE results with data.",
				zap.String("city", completeData.GeneralCityData.City),
				zap.Int("generalPOIs", len(completeData.PointsOfInterest)),
				zap.Int("personalizedPOIs", len(completeData.AIItineraryResponse.PointsOfInterest)))

			return results2.ItineraryResults(
				completeData.GeneralCityData,
				completeData.PointsOfInterest,
				completeData.AIItineraryResponse,
				true,
				true,
				15,
				[]string{}, sessionIDParam)
		}
	}

	return h.loadItineraryFromDatabase(sessionIDParam)
}

// HandleItinerarySSE handles Server-Sent Events for itinerary updates
func (h *ItineraryHandlers) HandleItinerarySSE(c *gin.Context) {
	sessionID := c.Query("sessionId")
	if sessionID == "" {
		c.Status(http.StatusBadRequest)
		return
	}

	h.logger.Info("SSE connection established for itinerary",
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
	go h.monitorItineraryUpdates(sessionID, updateChan)

	// Stream updates to client
	flusher := c.Writer.(http.Flusher)
	for {
		select {
		case event := <-updateChan:
			if event.Type == "complete" {
				h.logger.Info("Sending completion event",
					zap.String("sessionId", sessionID))

				// Send final completion event
				c.SSEvent("itinerary-complete", map[string]interface{}{
					"sessionId": sessionID,
					"message":   "Itinerary generation complete",
				})
				flusher.Flush()
				return
			}

			flusher.Flush()

		case <-c.Request.Context().Done():
			h.logger.Info("SSE connection closed",
				zap.String("sessionId", sessionID))
			return
		}
	}
}

// monitorItineraryUpdates monitors for itinerary updates and sends SSE events
func (h *ItineraryHandlers) monitorItineraryUpdates(sessionID string, updateChan chan<- models.ItinerarySSEEvent) {
	// Check for cached data first
	if completeData, found := cache.CompleteItineraryCache.Get(sessionID); found {
		h.logger.Info("Complete data found in cache, sending completion immediately",
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
				"sessionId":        sessionID,
				"city":             completeData.GeneralCityData.City,
				"totalPOIs":        len(completeData.PointsOfInterest),
				"personalizedPOIs": len(completeData.AIItineraryResponse.PointsOfInterest),
			},
		}
		return
	}

	// Legacy cache check
	if itineraryData, found := cache.ItineraryCache.Get(sessionID); found {
		h.logger.Info("Legacy data found in cache, sending completion immediately",
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
			if completeData, found := cache.CompleteItineraryCache.Get(sessionID); found {
				h.logger.Info("Complete data appeared in cache",
					zap.String("sessionId", sessionID))

				updateChan <- models.ItinerarySSEEvent{
					Type: "complete",
					Data: map[string]interface{}{
						"sessionId":        sessionID,
						"city":             completeData.GeneralCityData.City,
						"totalPOIs":        len(completeData.PointsOfInterest),
						"personalizedPOIs": len(completeData.AIItineraryResponse.PointsOfInterest),
					},
				}
				return
			}

			if itineraryData, found := cache.ItineraryCache.Get(sessionID); found {
				h.logger.Info("Legacy data appeared in cache",
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
					"message":   "Processing your itinerary...",
					"timestamp": time.Now().Unix(),
				},
			}

		case <-timeout:
			h.logger.Warn("SSE monitoring timed out", zap.String("sessionId", sessionID))
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

// loadItineraryFromDatabase loads itinerary from database when not found in cache
func (h *ItineraryHandlers) loadItineraryFromDatabase(sessionIDParam string) templ.Component {
	h.logger.Info("Itinerary not found in cache, attempting to load from database", zap.String("sessionID", sessionIDParam))

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
		return results2.PageNotFound("Itinerary session expired. Please create a new itinerary.")
	}

	// Parse the stored response as complete itinerary data
	completeData, err := h.itineraryService.ParseCompleteItineraryResponse(interaction.ResponseText, h.logger)
	if err != nil || completeData == nil {
		h.logger.Warn("Could not parse complete itinerary from stored response",
			zap.String("sessionID", sessionIDParam),
			zap.Error(err))
		return results2.PageNotFound("Could not load itinerary data. Please create a new itinerary.")
	}

	// Print JSON data for debugging what will be displayed from database
	jsonData, err := json.MarshalIndent(completeData, "", "  ")
	if err != nil {
		h.logger.Error("Failed to marshal database completeData to JSON", zap.Error(err))
	} else {
		h.logger.Info("Complete itinerary data from database being displayed in view", zap.String("json", string(jsonData)))
	}

	h.logger.Info("Successfully loaded complete itinerary from database",
		zap.String("sessionID", sessionIDParam),
		zap.String("city", completeData.GeneralCityData.City),
		zap.Int("generalPOIs", len(completeData.PointsOfInterest)),
		zap.Int("personalizedPOIs", len(completeData.AIItineraryResponse.PointsOfInterest)))

	// Render the results page with the complete database data
	return results2.ItineraryResults(
		completeData.GeneralCityData,
		completeData.PointsOfInterest,
		completeData.AIItineraryResponse,
		true, true, 5, []string{}, sessionIDParam)
}
