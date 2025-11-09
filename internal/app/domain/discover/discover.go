package discover

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/a-h/templ"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	llmchat "github.com/FACorreiaa/go-templui/internal/app/domain/chat_prompt"
	"github.com/FACorreiaa/go-templui/internal/app/domain/poi"
	"github.com/FACorreiaa/go-templui/internal/app/handlers"
	"github.com/FACorreiaa/go-templui/internal/app/middleware"

	generativeAI "github.com/FACorreiaa/go-genai-sdk/lib"
	genai "google.golang.org/genai"

	"github.com/FACorreiaa/go-templui/internal/app/models"
)

type DiscoverHandlers struct {
	*handlers.BaseHandler
	poiRepo    poi.Repository
	chatRepo   llmchat.Repository
	llmService llmchat.LlmInteractiontService
	aiClient   *generativeAI.LLMChatClient
	logger     *zap.Logger
	llmLogger  *llmchat.LLMLogger
}

func NewDiscoverHandlers(base *handlers.BaseHandler, poiRepo poi.Repository, chatRepo llmchat.Repository, llmService llmchat.LlmInteractiontService, logger *zap.Logger) *DiscoverHandlers {
	// Initialize AI client for discover search
	apiKey := os.Getenv("GEMINI_API_KEY")
	aiClient, err := generativeAI.NewLLMChatClient(context.Background(), apiKey)
	if err != nil {
		logger.Error("Failed to initialize AI client", zap.Any("error", err))
	}

	// Initialize LLM logger
	llmLogger := llmchat.NewLLMLogger(logger, chatRepo)

	return &DiscoverHandlers{
		BaseHandler: base,
		poiRepo:     poiRepo,
		chatRepo:    chatRepo,
		llmService:  llmService,
		aiClient:    aiClient,
		logger:      logger,
		llmLogger:   llmLogger,
	}
}

func (h *DiscoverHandlers) ShowDiscoverPage(c *gin.Context) {
	var recentDiscoveries []models.ChatSession
	var trending []models.TrendingDiscovery
	var featured []models.FeaturedCollection

	ctx := c.Request.Context()

	trendingData, err := h.llmService.GetTrendingDiscoveries(ctx, 5)
	if err == nil {
		trending = trendingData
	} else {
		h.Logger.Error("Failed to get trending discoveries", zap.Error(err))
	}

	featuredData, err := h.llmService.GetFeaturedCollections(ctx, 4)
	if err == nil {
		featured = featuredData
	} else {
		h.Logger.Error("Failed to get featured collections", zap.Error(err))
	}

	user := middleware.GetUserFromContext(c)

	if user != nil {
		userUUID, err := uuid.Parse(user.ID)
		if err == nil {
			response, err := h.llmService.GetRecentDiscoveries(ctx, userUUID, 6)
			if err == nil && response != nil {
				recentDiscoveries = response.Sessions
			} else if err != nil {
				h.Logger.Error("Failed to get recent discoveries", zap.Error(err))
			}
		}
	}

	content := DiscoverPage(recentDiscoveries, trending, featured)
	h.RenderPage(c, "Discover - Loci", "Discover", content)
}
func (h *DiscoverHandlers) ShowDetail(c *gin.Context) templ.Component {
	sessionID := c.Param("sessionId")
	if sessionID == "" {
		h.logger.Error("Session ID is required")
		// Return error page component
		return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
			_, err := io.WriteString(w, "<div class='text-red-500 text-center py-8'>Session ID is required</div>")
			return err
		})
	}

	sessionUUID, err := uuid.Parse(sessionID)
	if err != nil {
		h.logger.Error("Invalid session ID", zap.String("sessionId", sessionID), zap.Any("error", err))
		return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
			_, err := io.WriteString(w, "<div class='text-red-500 text-center py-8'>Invalid session ID</div>")
			return err
		})
	}

	// Get the session details
	session, err := h.chatRepo.GetSession(c.Request.Context(), sessionUUID)
	if err != nil {
		h.logger.Error("Failed to get session", zap.String("sessionId", sessionID), zap.Any("error", err))
		return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
			_, err := io.WriteString(w, "<div class='text-red-500 text-center py-8'>Discovery not found</div>")
			return err
		})
	}

	return DiscoveryDetailPage(*session)
}

func (h *DiscoverHandlers) ShowResults(c *gin.Context) templ.Component {
	sessionID := c.Param("sessionId")
	if sessionID == "" {
		h.logger.Error("Session ID is required")
		return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
			_, err := io.WriteString(w, "<div class='text-red-500 text-center py-8'>Session ID is required</div>")
			return err
		})
	}

	sessionUUID, err := uuid.Parse(sessionID)
	if err != nil {
		h.logger.Error("Invalid session ID", zap.String("sessionId", sessionID), zap.Any("error", err))
		return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
			_, err := io.WriteString(w, "<div class='text-red-500 text-center py-8'>Invalid session ID</div>")
			return err
		})
	}

	// Get the session details
	session, err := h.chatRepo.GetSession(c.Request.Context(), sessionUUID)
	if err != nil {
		h.logger.Error("Failed to get session", zap.String("sessionId", sessionID), zap.Any("error", err))
		return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
			_, err := io.WriteString(w, "<div class='text-red-500 text-center py-8'>Discovery session not found</div>")
			return err
		})
	}

	// Find the LLM interaction ID from the conversation messages
	var llmInteractionID uuid.UUID
	for _, msg := range session.ConversationHistory {
		if msg.Metadata.LlmInteractionID != nil {
			llmInteractionID = *msg.Metadata.LlmInteractionID
			break
		}
	}

	if llmInteractionID == uuid.Nil {
		h.logger.Error("No LLM interaction ID found in session", zap.String("sessionId", sessionID))
		return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
			_, err := io.WriteString(w, "<div class='text-red-500 text-center py-8'>No results available for this discovery</div>")
			return err
		})
	}

	// Get POIs by LLM interaction ID
	pois, err := h.poiRepo.GetPOIsByLLMInteraction(c.Request.Context(), llmInteractionID)
	if err != nil {
		h.logger.Error("Failed to get POIs by LLM interaction",
			zap.String("llmInteractionID", llmInteractionID.String()),
			zap.Any("error", err))
		return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
			_, err := io.WriteString(w, "<div class='text-red-500 text-center py-8'>Failed to load results</div>")
			return err
		})
	}

	h.logger.Info("Loaded POIs for discovery results",
		zap.String("sessionId", sessionID),
		zap.String("llmInteractionID", llmInteractionID.String()),
		zap.Int("count", len(pois)))

	// Convert POIs to DiscoverResult for display
	results := make([]DiscoverResult, len(pois))
	for i, poi := range pois {
		results[i] = DiscoverResult{
			ID:          poi.ID.String(),
			Name:        poi.Name,
			Latitude:    poi.Latitude,
			Longitude:   poi.Longitude,
			Category:    poi.Category,
			Description: poi.DescriptionPOI,
			Address:     poi.Address,
			PriceLevel:  poi.PriceLevel,
			Rating:      poi.Rating,
		}
		// Handle optional fields
		if poi.Website != "" {
			results[i].Website = &poi.Website
		}
		if poi.PhoneNumber != "" {
			results[i].PhoneNumber = &poi.PhoneNumber
		}
	}

	return DiscoveryResultsPage(session.CityName, results)
}

func (h *DiscoverHandlers) Search(c *gin.Context) {
	ctx := c.Request.Context()
	query := strings.TrimSpace(c.PostForm("query"))
	location := strings.TrimSpace(c.PostForm("location"))
	user := middleware.GetUserIDFromContext(c)

	h.logger.Info("Discovery search requested",
		zap.String("query", query),
		zap.String("location", location),
		zap.String("user", user),
		zap.String("ip", c.ClientIP()),
	)
	h.logger.Info("User from context in Search", zap.String("user", user))

	if query == "" {
		h.logger.Warn("Empty search query")
		c.HTML(http.StatusBadRequest, "", `<div class="text-red-500 text-center py-8">Please enter a search term</div>`)
		return
	}

	if location == "" {
		c.HTML(http.StatusBadRequest, "", `<div class="text-red-500 text-center py-8">Please enter a location</div>`)
		return
	}

	// Call LLM with discover search prompt
	prompt := llmchat.GetDiscoverSearchPrompt(query, location)
	h.logger.Info("Calling LLM for discover search", zap.String("query", query), zap.String("location", location))

	// Prepare logging configuration
	startTime := time.Now()
	sessionID := uuid.New()

	// Get user ID for logging
	userUUID := uuid.Nil
	if user != "" && user != "anonymous" {
		if parsedUserID, err := uuid.Parse(user); err == nil {
			userUUID = parsedUserID
		}
	}

	logConfig := llmchat.LoggingConfig{
		UserID:      userUUID,
		SessionID:   sessionID,
		Intent:      "discover",
		Prompt:      prompt,
		CityName:    location,
		ModelName:   "gemini-2.0-flash",
		Provider:    "google",
		Temperature: genai.Ptr[float32](0.5),
		IsStreaming: false,
	}

	response, err := h.aiClient.GenerateResponse(ctx, prompt, &genai.GenerateContentConfig{
		Temperature: genai.Ptr[float32](0.5), // Balanced temperature for diverse but consistent results
	})

	// Prepare LLM response for logging
	llmResponse := llmchat.LLMResponse{
		StatusCode: 200,
	}

	if err != nil {
		// Log failed LLM interaction
		llmResponse.StatusCode = 500
		llmResponse.ErrorMessage = err.Error()
		h.llmLogger.LogInteractionAsync(ctx, logConfig, llmResponse, time.Since(startTime).Milliseconds())

		h.logger.Error("LLM request failed", zap.Any("error", err))
		c.HTML(http.StatusInternalServerError, "", `<div class="text-red-500 text-center py-8">Failed to generate search results. Please try again.</div>`)
		return
	}

	// Extract text from response
	if response == nil || len(response.Candidates) == 0 {
		llmResponse.StatusCode = 500
		llmResponse.ErrorMessage = "Empty LLM response"
		h.llmLogger.LogInteractionAsync(ctx, logConfig, llmResponse, time.Since(startTime).Milliseconds())

		h.logger.Error("Empty LLM response")
		c.HTML(http.StatusInternalServerError, "", `<div class="text-red-500 text-center py-8">No results returned. Please try again.</div>`)
		return
	}

	var responseText strings.Builder
	for _, part := range response.Candidates[0].Content.Parts {
		if part.Text != "" {
			responseText.WriteString(string(part.Text))
		}
	}

	// Extract token counts from response metadata
	if response.UsageMetadata != nil {
		llmResponse.PromptTokens = int(response.UsageMetadata.PromptTokenCount)
		llmResponse.CompletionTokens = int(response.UsageMetadata.CandidatesTokenCount)
		llmResponse.TotalTokens = int(response.UsageMetadata.TotalTokenCount)
	}
	llmResponse.ResponseText = responseText.String()

	// Parse JSON response
	var searchResponse struct {
		Results []DiscoverResult `json:"results"`
	}

	responseStr := responseText.String()

	// Clean markdown code blocks if present
	responseStr = strings.TrimSpace(responseStr)
	if strings.HasPrefix(responseStr, "```json") {
		responseStr = strings.TrimPrefix(responseStr, "```json")
		responseStr = strings.TrimPrefix(responseStr, "```")
		responseStr = strings.TrimSuffix(responseStr, "```")
		responseStr = strings.TrimSpace(responseStr)
	} else if strings.HasPrefix(responseStr, "```") {
		responseStr = strings.TrimPrefix(responseStr, "```")
		responseStr = strings.TrimSuffix(responseStr, "```")
		responseStr = strings.TrimSpace(responseStr)
	}

	if err := json.Unmarshal([]byte(responseStr), &searchResponse); err != nil {
		llmResponse.StatusCode = 500
		llmResponse.ErrorMessage = fmt.Sprintf("Failed to parse JSON: %v", err)
		h.llmLogger.LogInteractionAsync(ctx, logConfig, llmResponse, time.Since(startTime).Milliseconds())

		h.logger.Error("Failed to parse LLM response", zap.Any("error", err), zap.String("response", responseStr))
		c.HTML(http.StatusInternalServerError, "", `<div class="text-red-500 text-center py-8">Failed to parse search results. Please try again.</div>`)
		return
	}

	// Log successful LLM interaction synchronously to get the interaction ID
	llmInteractionID, err := h.llmLogger.LogInteraction(ctx, logConfig, llmResponse, time.Since(startTime).Milliseconds())
	if err != nil {
		h.logger.Error("Failed to log LLM interaction", zap.Any("error", err))
		// Continue even if logging fails
		llmInteractionID = uuid.Nil
	}

	h.logger.Info("Search completed",
		zap.String("query", query),
		zap.String("location", location),
		zap.Int("results_count", len(searchResponse.Results)),
		zap.String("user", user),
	)

	// Convert handler DiscoverResult to models.POIDetailedInfo for database storage
	poiDetails := make([]models.POIDetailedInfo, len(searchResponse.Results))
	templResults := make([]DiscoverResult, len(searchResponse.Results))

	for i, result := range searchResponse.Results {
		// Generate a UUID for this LLM-generated result
		poiID := uuid.New()
		result.ID = poiID.String()
		templResults[i] = result

		// Convert to POIDetailedInfo for database storage
		poiDetails[i] = models.POIDetailedInfo{
			ID:          poiID,
			Name:        result.Name,
			Latitude:    result.Latitude,
			Longitude:   result.Longitude,
			Category:    result.Category,
			Description: result.Description,
			Address:     result.Address,
			Rating:      result.Rating,
			PriceLevel:  result.PriceLevel,
			City:        location,
			// Other fields from DiscoverResult
		}

		// Handle optional fields
		if result.Website != nil {
			poiDetails[i].Website = *result.Website
		}
		if result.PhoneNumber != nil {
			poiDetails[i].PhoneNumber = *result.PhoneNumber
		}
		// Note: OpeningHours in DiscoverResult is *string but POIDetailedInfo expects map[string]string
		// This would require parsing, so skipping for now
		if result.CuisineType != nil {
			poiDetails[i].CuisineType = *result.CuisineType
		}
	}

	// Save LLM POIs to database for authenticated users
	if userUUID != uuid.Nil && llmInteractionID != uuid.Nil && len(poiDetails) > 0 {
		if err := h.poiRepo.SaveLlmPoisToDatabase(ctx, userUUID, poiDetails, nil, llmInteractionID); err != nil {
			h.logger.Error("Failed to save LLM POIs to database",
				zap.Any("error", err),
				zap.String("user_id", userUUID.String()),
				zap.String("llm_interaction_id", llmInteractionID.String()))
			// Don't fail the request if POI saving fails
		} else {
			h.logger.Info("Successfully saved LLM POIs to database",
				zap.Int("count", len(poiDetails)),
				zap.String("user_id", userUUID.String()),
				zap.String("llm_interaction_id", llmInteractionID.String()))
		}
	}

	// Save discover search as chat session for authenticated users
	if user != "" && user != "anonymous" {
		userID, err := uuid.Parse(user)
		if err == nil {
			sessionID := uuid.New()
			now := time.Now()
			expiresAt := now.Add(72 * time.Hour) // 3 days expiry

			// Create conversation message for the search
			conversationHistory := []models.ConversationMessage{
				{
					ID:          uuid.New(),
					Role:        models.RoleUser,
					Content:     fmt.Sprintf("%s in %s", query, location),
					MessageType: models.TypeInitialRequest,
					Timestamp:   now,
				},
				{
					ID:          uuid.New(),
					Role:        models.RoleAssistant,
					Content:     fmt.Sprintf("Found %d results", len(searchResponse.Results)),
					MessageType: models.TypeResponse,
					Timestamp:   now,
					Metadata: models.MessageMetadata{
						LlmInteractionID: &llmInteractionID,
					},
				},
			}

			session := models.ChatSession{
				ID:                  sessionID,
				UserID:              userID,
				ProfileID:           uuid.Nil, // No profile for discover searches
				CityName:            location,
				ConversationHistory: conversationHistory,
				SessionContext: models.SessionContext{
					CityName: location,
				},
				CreatedAt:  now,
				UpdatedAt:  now,
				ExpiresAt:  expiresAt,
				Status:     models.StatusActive,
				SearchType: models.SearchTypeDiscover, // Mark as discover search
			}

			if err := h.chatRepo.CreateSession(ctx, session); err != nil {
				h.logger.Error("Failed to save discover session", zap.Any("error", err))
				// Don't fail the request if session saving fails
			} else {
				h.logger.Info("Discover session saved",
					zap.String("session_id", sessionID.String()),
					zap.String("user_id", userID.String()),
					zap.String("location", location))
			}
		} else {
			h.logger.Error("Failed to parse user ID from context", zap.Any("error", err), zap.String("user", user))
		}
	} else {
		h.logger.Info("User is anonymous, not saving session")
	}

	// Render results using templ component
	component := DiscoverSearchResults(templResults, query, location)
	component.Render(c.Request.Context(), c.Writer)
}

func (h *DiscoverHandlers) GetRecentDiscoveries(c *gin.Context) {
	ctx := c.Request.Context()
	var recentDiscoveries []models.ChatSession

	// Get user-specific recent discoveries (only for authenticated users)
	userID, exists := c.Get("user_id")
	if exists && userID != "" && userID != "anonymous" {
		userUUID, err := uuid.Parse(userID.(string))
		if err == nil {
			response, err := h.llmService.GetRecentDiscoveries(ctx, userUUID, 6)
			if err == nil && response != nil {
				recentDiscoveries = response.Sessions
			} else {
				h.logger.Error("Failed to get recent discoveries", zap.Any("error", err))
			}
		}
	}

	// Render the recent discoveries list component
	component := RecentDiscoveriesList(recentDiscoveries)
	component.Render(c.Request.Context(), c.Writer)
}

func (h *DiscoverHandlers) GetCategory(c *gin.Context) {
	category := c.Param("category")
	user := middleware.GetUserIDFromContext(c)

	h.logger.Info("Category search requested",
		zap.String("category", category),
		zap.String("user", user),
		zap.String("ip", c.ClientIP()),
	)

	// Generate category-specific results
	results := generateCategoryResults(category)

	h.logger.Info("Category search completed",
		zap.String("category", category),
		zap.Int("results_count", len(results)),
		zap.String("user", user),
	)

	// Return category results HTML
	tc := cases.Title(language.English)
	html := `<div><h2 class="text-lg font-semibold text-foreground mb-4">` + tc.String(category) + `s Near You</h2><div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">`

	for _, result := range results {
		html += `
		<div class="bg-card rounded-xl shadow-sm border hover:shadow-md transition-shadow group cursor-pointer">
			<div class="p-6">
				<div class="flex items-start justify-between mb-4">
					<div class="flex items-center gap-3">
						<span class="text-3xl">` + result.Emoji + `</span>
						<span class="inline-flex items-center px-3 py-1 rounded-full text-sm font-medium bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-300">
							` + result.Category + `
						</span>
					</div>
				</div>
				<div class="space-y-3">
					<h3 class="font-semibold text-card-foreground group-hover:text-blue-600 transition-colors">` + result.Title + `</h3>
					<div class="flex items-center gap-2 text-sm text-muted-foreground">
						<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17.657 16.657L13.414 20.9a1.998 1.998 0 01-2.827 0l-4.244-4.243a8 8 0 1111.314 0z"></path>
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 11a3 3 0 11-6 0 3 3 0 016 0z"></path>
						</svg>
						` + result.Location + `
					</div>
					<p class="text-muted-foreground text-sm">` + result.Description + `</p>
				</div>
			</div>
		</div>`
	}

	html += `</div></div>`

	c.HTML(http.StatusOK, "", html)
}

// Helper structs and functions
type SearchResult struct {
	ID          string
	Title       string
	Category    string
	Location    string
	Description string
	Emoji       string
	Rating      string
}

func generateSearchResults(query, location string) []SearchResult {
	// Simulate search results based on query
	return []SearchResult{
		{
			ID:          "1",
			Title:       "Amazing Local Restaurant",
			Category:    "Restaurant",
			Location:    location,
			Description: "Great local spot matching your search for: " + query,
			Emoji:       "🍽️",
			Rating:      "4.5",
		},
		{
			ID:          "2",
			Title:       "Popular Activity",
			Category:    "Activity",
			Location:    location,
			Description: "Top-rated activity related to: " + query,
			Emoji:       "🎯",
			Rating:      "4.7",
		},
	}
}

func generateCategoryResults(category string) []SearchResult {
	categoryMap := map[string]SearchResult{
		"restaurant": {
			ID:          "cat-1",
			Title:       "Top Local Restaurant",
			Category:    "Restaurant",
			Location:    "Your Area",
			Description: "Highly recommended dining experience",
			Emoji:       "🍽️",
			Rating:      "4.6",
		},
		"hotel": {
			ID:          "cat-2",
			Title:       "Boutique Hotel",
			Category:    "Hotel",
			Location:    "City Center",
			Description: "Charming accommodations with great service",
			Emoji:       "🏨",
			Rating:      "4.8",
		},
		"activity": {
			ID:          "cat-3",
			Title:       "Fun Local Activity",
			Category:    "Activity",
			Location:    "Nearby",
			Description: "Exciting things to do in the area",
			Emoji:       "🎯",
			Rating:      "4.5",
		},
	}

	if result, exists := categoryMap[category]; exists {
		return []SearchResult{result}
	}

	return []SearchResult{}
}

// GetNearbyPOIs returns POIs within specified distance using PostGIS
func (h *DiscoverHandlers) GetNearbyPOIs(c *gin.Context) {
	ctx := c.Request.Context()

	// Parse query parameters
	latStr := c.Query("lat")
	lonStr := c.Query("lon")
	radiusStr := c.Query("radius")  // in kilometers
	category := c.Query("category") // optional category filter

	// Default location (Lisbon) if not provided
	lat := 38.7223
	lon := -9.1393
	radiusKm := 10.0 // default 10km

	if latStr != "" {
		if parsedLat, err := strconv.ParseFloat(latStr, 64); err == nil {
			lat = parsedLat
		}
	}

	if lonStr != "" {
		if parsedLon, err := strconv.ParseFloat(lonStr, 64); err == nil {
			lon = parsedLon
		}
	}

	if radiusStr != "" {
		if parsedRadius, err := strconv.ParseFloat(radiusStr, 64); err == nil {
			radiusKm = parsedRadius
		}
	}

	// Convert km to meters for PostGIS
	radiusMeters := radiusKm * 1000

	h.logger.Info("Nearby POIs requested",
		zap.Float64("lat", lat),
		zap.Float64("lon", lon),
		zap.Float64("radius_km", radiusKm),
		zap.String("category", category))

	// Query database using PostGIS
	var pois []models.POIDetailedInfo
	var err error

	if category != "" {
		pois, err = h.poiRepo.GetPOIsByLocationAndDistanceWithCategory(ctx, lat, lon, radiusMeters, category)
	} else {
		pois, err = h.poiRepo.GetPOIsByLocationAndDistance(ctx, lat, lon, radiusMeters)
	}

	if err != nil {
		h.logger.Error("Failed to get nearby POIs", zap.Any("error", err))
		c.HTML(http.StatusInternalServerError, "", `<div class="text-red-500 text-center py-8">Failed to load nearby places</div>`)
		return
	}

	h.logger.Info("Nearby POIs retrieved",
		zap.Int("count", len(pois)),
		zap.Float64("radius_km", radiusKm))

	// Render results HTML
	html := h.renderPOIResults(pois, radiusKm)
	c.HTML(http.StatusOK, "", html)
}

// renderPOIResults generates HTML for POI results
func (h *DiscoverHandlers) renderPOIResults(pois []models.POIDetailedInfo, radiusKm float64) string {
	if len(pois) == 0 {
		return `<div class="text-center py-12">
			<svg class="w-16 h-16 mx-auto text-gray-400 mb-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9.172 16.172a4 4 0 015.656 0M9 10h.01M15 10h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"></path>
			</svg>
			<p class="text-gray-600 dark:text-gray-400">No places found within ` + fmt.Sprintf("%.0f km", radiusKm) + `</p>
		</div>`
	}

	html := `<div><h2 class="text-lg font-semibold text-foreground mb-4">` +
		fmt.Sprintf("Found %d places within %.0f km", len(pois), radiusKm) +
		`</h2><div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">`

	for _, poi := range pois {
		emoji := getCategoryEmojiForPOI(poi.Category)
		categoryColor := getCategoryColorForPOI(poi.Category)

		html += fmt.Sprintf(`
		<div class="bg-card rounded-xl shadow-sm border hover:shadow-md transition-shadow group cursor-pointer">
			<div class="p-6">
				<div class="flex items-start justify-between mb-4">
					<div class="flex items-center gap-3">
						<span class="text-3xl">%s</span>
						<span class="inline-flex items-center px-3 py-1 rounded-full text-sm font-medium %s">
							%s
						</span>
					</div>
					<button
						hx-post="/favorites/add/%s"
						hx-target="this"
						hx-swap="outerHTML"
						class="p-2 text-muted-foreground hover:text-red-500 rounded-lg hover:bg-red-50 dark:hover:bg-red-900/20 transition-colors"
					>
						<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M20.84 4.61a5.5 5.5 0 0 0-7.78 0L12 5.67l-1.06-1.06a5.5 5.5 0 0 0-7.78 7.78l1.06 1.06L12 21.23l7.78-7.78 1.06-1.06a5.5 5.5 0 0 0 0-7.78z"></path>
						</svg>
					</button>
				</div>
				<div class="space-y-3">
					<h3 class="font-semibold text-card-foreground group-hover:text-blue-600 transition-colors">%s</h3>
					<div class="flex items-center gap-2 text-sm text-muted-foreground">
						<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17.657 16.657L13.414 20.9a1.998 1.998 0 01-2.827 0l-4.244-4.243a8 8 0 1111.314 0z"></path>
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 11a3 3 0 11-6 0 3 3 0 016 0z"></path>
						</svg>
						%.1f km away
					</div>
					<p class="text-muted-foreground text-sm">%s</p>
					<div class="flex items-center gap-1">
						<svg class="w-4 h-4 text-yellow-500 fill-current" fill="currentColor" viewBox="0 0 24 24">
							<path d="M11.049 2.927c.3-.921 1.603-.921 1.902 0l1.519 4.674a1 1 0 00.95.69h4.915c.969 0 1.371 1.24.588 1.81l-3.976 2.888a1 1 0 00-.363 1.118l1.518 4.674c.3.922-.755 1.688-1.538 1.118l-3.976-2.888a1 1 0 00-1.176 0l-3.976 2.888c-.783.57-1.838-.197-1.538-1.118l1.518-4.674a1 1 0 00-.363-1.118l-3.976-2.888c-.784-.57-.38-1.81.588-1.81h4.914a1 1 0 00.951-.69l1.519-4.674z"></path>
						</svg>
						<span class="text-sm font-medium">%.1f</span>
					</div>
				</div>
			</div>
		</div>`,
			emoji,
			categoryColor,
			poi.Category,
			poi.ID.String(),
			poi.Name,
			poi.Distance,
			poi.Description,
			poi.Rating)
	}

	html += `</div></div>`
	return html
}

// getCategoryEmojiForPOI returns emoji for POI category
func getCategoryEmojiForPOI(category string) string {
	categoryLower := strings.ToLower(category)
	emojiMap := map[string]string{
		"restaurant":    "🍽️",
		"hotel":         "🏨",
		"activity":      "🎯",
		"attraction":    "🏛️",
		"museum":        "🎨",
		"park":          "🌳",
		"beach":         "🏖️",
		"nightlife":     "🌃",
		"shopping":      "🛍️",
		"cultural":      "🎭",
		"market":        "🏪",
		"adventure":     "⛰️",
		"cafe":          "☕",
		"bar":           "🍺",
		"entertainment": "🎪",
	}

	if emoji, ok := emojiMap[categoryLower]; ok {
		return emoji
	}
	return "📍"
}

// getCategoryColorForPOI returns Tailwind color classes for POI category
func getCategoryColorForPOI(category string) string {
	categoryLower := strings.ToLower(category)
	colorMap := map[string]string{
		"restaurant": "bg-orange-100 text-orange-800 dark:bg-orange-900/30 dark:text-orange-300",
		"hotel":      "bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-300",
		"activity":   "bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-300",
		"attraction": "bg-purple-100 text-purple-800 dark:bg-purple-900/30 dark:text-purple-300",
		"museum":     "bg-indigo-100 text-indigo-800 dark:bg-indigo-900/30 dark:text-indigo-300",
		"park":       "bg-emerald-100 text-emerald-800 dark:bg-emerald-900/30 dark:text-emerald-300",
		"beach":      "bg-cyan-100 text-cyan-800 dark:bg-cyan-900/30 dark:text-cyan-300",
		"nightlife":  "bg-violet-100 text-violet-800 dark:bg-violet-900/30 dark:text-violet-300",
	}

	if color, ok := colorMap[categoryLower]; ok {
		return color
	}
	return "bg-gray-100 text-gray-800 dark:bg-gray-900/30 dark:text-gray-300"
}

// DiscoverResult represents a single search result from the LLM
// This matches the discover.DiscoverResult type in the templ package
type DiscoverResult struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Latitude     float64  `json:"latitude"`
	Longitude    float64  `json:"longitude"`
	Category     string   `json:"category"`
	Description  string   `json:"description"`
	Address      string   `json:"address"`
	Website      *string  `json:"website"`
	PhoneNumber  *string  `json:"phone_number"`
	OpeningHours *string  `json:"opening_hours"`
	PriceLevel   string   `json:"price_level"`
	Rating       float64  `json:"rating"`
	Tags         []string `json:"tags"`
	Images       []string `json:"images"`
	CuisineType  *string  `json:"cuisine_type"`
	StarRating   *string  `json:"star_rating"`
}
