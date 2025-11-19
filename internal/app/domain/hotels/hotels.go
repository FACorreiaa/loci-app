package hotels

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/a-h/templ"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/FACorreiaa/go-templui/internal/app/common"
	llmchat "github.com/FACorreiaa/go-templui/internal/app/domain/chat_prompt"
	results2 "github.com/FACorreiaa/go-templui/internal/app/domain/results"
	"github.com/FACorreiaa/go-templui/internal/app/models"
	"github.com/FACorreiaa/go-templui/internal/app/services"
	"github.com/FACorreiaa/go-templui/internal/pkg/cache"
)

type HotelsHandlers struct {
	chatRepo         llmchat.Repository
	itineraryService *services.ItineraryService
	logger           *zap.Logger
}

func NewHotelsHandlers(chatRepo llmchat.Repository, logger *zap.Logger) *HotelsHandlers {
	return &HotelsHandlers{
		chatRepo:         chatRepo,
		itineraryService: services.NewItineraryService(),
		logger:           logger,
	}
}

// HandleHotelsPage handles the main hotels page logic
func (h *HotelsHandlers) HandleHotelsPage(c *gin.Context) templ.Component {
	query := c.Query("q")
	sessionIDParam := c.Query("sessionId")
	cacheKey := c.Query("cacheKey")

	h.logger.Info("Hotels page accessed",
		zap.String("ip", c.ClientIP()),
		zap.String("query", query),
		zap.String("sessionId", sessionIDParam),
		zap.String("cacheKey", cacheKey))

	// Handle session-based loading
	if sessionIDParam != "" {
		return h.loadHotelsBySession(sessionIDParam, cacheKey)
	}

	// Handle query-based loading
	if query != "" {
		return HotelsPageWithQuery(query)
	}

	// Default hotels page
	return HotelsPage()
}

// loadHotelsBySession loads hotels using IDENTICAL logic as itinerary
func (h *HotelsHandlers) loadHotelsBySession(sessionIDParam string, cacheKey string) templ.Component {
	h.logger.Info("Attempting to load hotels from cache",
		zap.String("sessionID", sessionIDParam),
		zap.String("cacheKey", cacheKey))

	// Try hotels cache first with cacheKey (for reusable cache hits)
	if cacheKey != "" {
		if hotelsData, found := cache.HotelsCache.Get(cacheKey); found {
			h.logger.Info("Hotels found in cache. Rendering results with data.",
				zap.Int("hotels", len(hotelsData)))

			// Try to get city data from complete cache
			var cityData models.GeneralCityData
			if completeData, found := cache.CompleteItineraryCache.Get(cacheKey); found {
				cityData = completeData.GeneralCityData
				h.logger.Info("City data loaded from complete cache",
					zap.String("city", cityData.City))
			} else {
				// Fallback: load from database using sessionID
				cityData = h.loadCityDataFromDatabase(sessionIDParam)
			}

			// Return static template when data is available
			return results2.HotelsResults(
				cityData,
				hotelsData,
				true, true, 15, []string{}, sessionIDParam)
		}
	}

	// Load from database (IDENTICAL to itinerary logic)
	return h.loadHotelsFromDatabase(sessionIDParam)
}

// loadHotelsFromDatabase loads hotels from database when not found in cache
func (h *HotelsHandlers) loadHotelsFromDatabase(sessionIDParam string) templ.Component {
	h.logger.Info("Hotels not found in cache, attempting to load from database", zap.String("sessionID", sessionIDParam))

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
		emptyHotels := []models.HotelDetailedInfo{}
		return results2.HotelsResults(emptyCityData, emptyHotels, true, true, 5, []string{}, sessionIDParam)
	}

	// Parse the stored response as complete data
	completeData, err := h.itineraryService.ParseCompleteItineraryResponse(interaction.ResponseText, h.logger)
	if err != nil || completeData == nil {
		h.logger.Warn("Could not parse complete data from stored response",
			zap.String("sessionID", sessionIDParam),
			zap.Error(err))
		if fallback, ok := h.parseHotelsOnlyResponse(sessionIDParam, interaction); ok {
			return fallback
		}
		// Return empty results instead of PageNotFound for parsing errors
		emptyCityData := models.GeneralCityData{}
		emptyHotels := []models.HotelDetailedInfo{}
		return results2.HotelsResults(emptyCityData, emptyHotels, true, true, 5, []string{}, sessionIDParam)
	}

	h.logger.Info("Successfully loaded complete data from database for hotels",
		zap.String("sessionID", sessionIDParam),
		zap.String("city", completeData.GeneralCityData.City),
		zap.Int("totalPOIs", len(completeData.PointsOfInterest)))

	// Filter POIs for hotels and render (IDENTICAL to itinerary results pattern)
	hotelPOIs := common.FilterPOIsForHotels(completeData.PointsOfInterest)
	return results2.HotelsResults(
		completeData.GeneralCityData,
		hotelPOIs,
		true, true, 5, []string{}, sessionIDParam)
}

// HandleHotelsPageSSE handles the hotels page with SSE support
func (h *HotelsHandlers) HandleHotelsPageSSE(c *gin.Context) templ.Component {
	query := c.Query("q")
	sessionIDParam := c.Query("sessionId")
	cacheKey := c.Query("cacheKey")

	h.logger.Info("Hotels SSE page accessed",
		zap.String("ip", c.ClientIP()),
		zap.String("query", query),
		zap.String("sessionId", sessionIDParam),
		zap.String("cacheKey", cacheKey))

	if sessionIDParam == "" {
		h.logger.Info("Direct navigation to /hotels SSE. Showing default page.")
		return HotelsPage()
	}

	// Load hotels data for session with SSE support
	return h.loadHotelsBySession(sessionIDParam, cacheKey)
}

// StreamHotelUpdates streams domain mutations for hotels so HTMX can update independently from chat SSE.
func (h *HotelsHandlers) StreamHotelUpdates(c *gin.Context) {
	sessionID := c.Query("sessionId")
	if sessionID == "" {
		h.logger.Warn("Hotel updates stream requested without sessionId")
		c.String(http.StatusBadRequest, "sessionId query parameter is required")
		return
	}

	domain := c.DefaultQuery("domain", "hotels")
	updateCh, cancel := llmchat.DomainUpdates.Subscribe(sessionID, domain)
	defer cancel()

	w := c.Writer
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	flusher, ok := w.(http.Flusher)
	if !ok {
		h.logger.Warn("Response writer does not support flushing for hotel stream")
		c.Status(http.StatusInternalServerError)
		return
	}

	ctx := c.Request.Context()
	h.logger.Info("Hotel updates stream connected",
		zap.String("sessionID", sessionID),
		zap.String("domain", domain))

	for {
		select {
		case <-ctx.Done():
			h.logger.Info("Hotel updates stream closed by client", zap.String("sessionID", sessionID))
			return
		case event, ok := <-updateCh:
			if !ok {
				h.logger.Info("Hotel updates channel closed", zap.String("sessionID", sessionID))
				return
			}

			payload, err := json.Marshal(event)
			if err != nil {
				h.logger.Warn("Failed to marshal hotel update event", zap.Error(err))
				continue
			}

			fmt.Fprintf(w, "event: %s\n", event.Type)
			fmt.Fprintf(w, "data: %s\n\n", payload)
			flusher.Flush()
		}
	}
}

// loadCityDataFromDatabase loads city data from database by sessionID
func (h *HotelsHandlers) loadCityDataFromDatabase(sessionIDParam string) models.GeneralCityData {
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

func (h *HotelsHandlers) parseHotelsOnlyResponse(sessionID string, interaction *models.LlmInteraction) (templ.Component, bool) {
	cleaned := llmchat.NormalizeHotelJSONResponse(interaction.ResponseText)
	if cleaned == "" {
		return nil, false
	}

	var hotelWrapper struct {
		CityData    *models.GeneralCityData    `json:"general_city_data"`
		CityDataAlt *models.GeneralCityData    `json:"city_data"`
		Hotels      []models.HotelDetailedInfo `json:"hotels"`
	}
	var lastErr error
	if err := json.Unmarshal([]byte(cleaned), &hotelWrapper); err == nil && len(hotelWrapper.Hotels) > 0 {
		cityData := models.GeneralCityData{City: interaction.CityName}
		if hotelWrapper.CityData != nil {
			cityData = *hotelWrapper.CityData
		} else if hotelWrapper.CityDataAlt != nil {
			cityData = *hotelWrapper.CityDataAlt
		}
		h.logger.Info("Parsed hotels fallback response",
			zap.String("sessionID", sessionID),
			zap.Int("hotels", len(hotelWrapper.Hotels)))
		return results2.HotelsResults(cityData, hotelWrapper.Hotels, true, true, 5, []string{}, sessionID), true
	} else if err != nil {
		lastErr = err
	}

	var hotelsOnly []models.HotelDetailedInfo
	if err := json.Unmarshal([]byte(cleaned), &hotelsOnly); err == nil && len(hotelsOnly) > 0 {
		h.logger.Info("Parsed hotels fallback array response",
			zap.String("sessionID", sessionID),
			zap.Int("hotels", len(hotelsOnly)))
		return results2.HotelsResults(models.GeneralCityData{City: interaction.CityName}, hotelsOnly, true, true, 5, []string{}, sessionID), true
	} else if err != nil {
		lastErr = err
	}

	if lastErr != nil {
		h.logger.Warn("Hotels fallback parse failed",
			zap.String("sessionID", sessionID),
			zap.Error(lastErr))
	}
	return nil, false
}
