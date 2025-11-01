package discover

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
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

	"github.com/FACorreiaa/go-templui/internal/app/domain/discover"

	llmchat2 "github.com/FACorreiaa/go-templui/internal/app/domain/chat_prompt"
	"github.com/FACorreiaa/go-templui/internal/app/domain/poi"

	generativeAI "github.com/FACorreiaa/go-genai-sdk/lib"
	genai "google.golang.org/genai"

	"github.com/FACorreiaa/go-templui/internal/app/models"
	"github.com/FACorreiaa/go-templui/internal/pkg/logger"
	"github.com/FACorreiaa/go-templui/internal/pkg/middleware"
)

type DiscoverHandlers struct {
	poiRepo    poi.Repository
	chatRepo   llmchat2.Repository
	llmService llmchat2.LlmInteractiontService
	aiClient   *generativeAI.LLMChatClient
	logger     *slog.Logger
	llmLogger  *llmchat2.LLMLogger
}

func NewDiscoverHandlers(poiRepo poi.Repository, chatRepo llmchat2.Repository, llmService llmchat2.LlmInteractiontService, logger *slog.Logger) *DiscoverHandlers {
	// Initialize AI client for discover search
	apiKey := os.Getenv("GEMINI_API_KEY")
	aiClient, err := generativeAI.NewLLMChatClient(context.Background(), apiKey)
	if err != nil {
		logger.Error("Failed to initialize AI client", slog.Any("error", err))
	}

	// Initialize LLM logger
	llmLogger := llmchat2.NewLLMLogger(logger, chatRepo)

	return &DiscoverHandlers{
		poiRepo:    poiRepo,
		chatRepo:   chatRepo,
		llmService: llmService,
		aiClient:   aiClient,
		logger:     logger,
		llmLogger:  llmLogger,
	}
}

func (h *DiscoverHandlers) Show(c *gin.Context) templ.Component {
	var recentDiscoveries []models.ChatSession
	var trending []models.TrendingDiscovery
	var featured []models.FeaturedCollection

	ctx := c.Request.Context()

	// Get trending discoveries (public data)
	trendingData, err := h.llmService.GetTrendingDiscoveries(ctx, 5)
	if err == nil {
		trending = trendingData
	} else {
		h.logger.Error("Failed to get trending discoveries", slog.Any("error", err))
	}

	// Get featured collections (public data)
	featuredData, err := h.llmService.GetFeaturedCollections(ctx, 4)
	if err == nil {
		featured = featuredData
	} else {
		h.logger.Error("Failed to get featured collections", slog.Any("error", err))
	}

	// Get user-specific recent discoveries (only for authenticated users)
	userID, exists := c.Get("user_id")
	if exists && userID != "" && userID != "anonymous" {
		userUUID, err := uuid.Parse(userID.(string))
		if err == nil {
			response, err := h.llmService.GetRecentDiscoveries(ctx, userUUID, 6) // Fetching 6 recent discoveries
			if err == nil && response != nil {
				recentDiscoveries = response.Sessions
			} else {
				h.logger.Error("Failed to get recent discoveries", slog.Any("error", err))
			}
		}
	}
	return discover.DiscoverPage(recentDiscoveries, trending, featured)
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
		h.logger.Error("Invalid session ID", slog.String("sessionId", sessionID), slog.Any("error", err))
		return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
			_, err := io.WriteString(w, "<div class='text-red-500 text-center py-8'>Invalid session ID</div>")
			return err
		})
	}

	// Get the session details
	session, err := h.chatRepo.GetSession(c.Request.Context(), sessionUUID)
	if err != nil {
		h.logger.Error("Failed to get session", slog.String("sessionId", sessionID), slog.Any("error", err))
		return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
			_, err := io.WriteString(w, "<div class='text-red-500 text-center py-8'>Discovery not found</div>")
			return err
		})
	}

	return discover.DiscoveryDetailPage(*session)
}

func (h *DiscoverHandlers) Search(c *gin.Context) {
	ctx := c.Request.Context()
	query := strings.TrimSpace(c.PostForm("query"))
	location := strings.TrimSpace(c.PostForm("location"))
	user := middleware.GetUserIDFromContext(c)

	logger.Log.Info("Discovery search requested",
		zap.String("query", query),
		zap.String("location", location),
		zap.String("user", user),
		zap.String("ip", c.ClientIP()),
	)
	h.logger.Info("User from context in Search", slog.String("user", user))

	if query == "" {
		logger.Log.Warn("Empty search query")
		c.HTML(http.StatusBadRequest, "", `<div class="text-red-500 text-center py-8">Please enter a search term</div>`)
		return
	}

	if location == "" {
		c.HTML(http.StatusBadRequest, "", `<div class="text-red-500 text-center py-8">Please enter a location</div>`)
		return
	}

	// Call LLM with discover search prompt
	prompt := llmchat2.GetDiscoverSearchPrompt(query, location)
	h.logger.InfoContext(ctx, "Calling LLM for discover search", slog.String("query", query), slog.String("location", location))

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

	logConfig := llmchat2.LoggingConfig{
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
	llmResponse := llmchat2.LLMResponse{
		StatusCode: 200,
	}

	if err != nil {
		// Log failed LLM interaction
		llmResponse.StatusCode = 500
		llmResponse.ErrorMessage = err.Error()
		h.llmLogger.LogInteractionAsync(ctx, logConfig, llmResponse, time.Since(startTime).Milliseconds())

		h.logger.ErrorContext(ctx, "LLM request failed", slog.Any("error", err))
		c.HTML(http.StatusInternalServerError, "", `<div class="text-red-500 text-center py-8">Failed to generate search results. Please try again.</div>`)
		return
	}

	// Extract text from response
	if response == nil || len(response.Candidates) == 0 {
		llmResponse.StatusCode = 500
		llmResponse.ErrorMessage = "Empty LLM response"
		h.llmLogger.LogInteractionAsync(ctx, logConfig, llmResponse, time.Since(startTime).Milliseconds())

		h.logger.ErrorContext(ctx, "Empty LLM response")
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

		h.logger.ErrorContext(ctx, "Failed to parse LLM response", slog.Any("error", err), slog.String("response", responseStr))
		c.HTML(http.StatusInternalServerError, "", `<div class="text-red-500 text-center py-8">Failed to parse search results. Please try again.</div>`)
		return
	}

	// Log successful LLM interaction
	h.llmLogger.LogInteractionAsync(ctx, logConfig, llmResponse, time.Since(startTime).Milliseconds())

	logger.Log.Info("Search completed",
		zap.String("query", query),
		zap.String("location", location),
		zap.Int("results_count", len(searchResponse.Results)),
		zap.String("user", user),
	)

	// Convert handler DiscoverResult to templ DiscoverResult and generate UUIDs
	templResults := make([]discover.DiscoverResult, len(searchResponse.Results))
	for i, result := range searchResponse.Results {
		// Generate a UUID for this LLM-generated result
		result.ID = uuid.New().String()
		templResults[i] = discover.DiscoverResult(result)
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
				h.logger.ErrorContext(ctx, "Failed to save discover session", slog.Any("error", err))
				// Don't fail the request if session saving fails
			} else {
				h.logger.InfoContext(ctx, "Discover session saved",
					slog.String("session_id", sessionID.String()),
					slog.String("user_id", userID.String()),
					slog.String("location", location))
			}
		} else {
			h.logger.ErrorContext(ctx, "Failed to parse user ID from context", slog.Any("error", err), slog.String("user", user))
		}
	} else {
		h.logger.InfoContext(ctx, "User is anonymous, not saving session")
	}

	// Render results using templ component
	component := discover.DiscoverSearchResults(templResults, query, location)
	component.Render(c.Request.Context(), c.Writer)
}

func (h *DiscoverHandlers) GetCategory(c *gin.Context) {
	category := c.Param("category")
	user := middleware.GetUserIDFromContext(c)

	logger.Log.Info("Category search requested",
		zap.String("category", category),
		zap.String("user", user),
		zap.String("ip", c.ClientIP()),
	)

	// Generate category-specific results
	results := generateCategoryResults(category)

	logger.Log.Info("Category search completed",
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

	h.logger.InfoContext(ctx, "Nearby POIs requested",
		slog.Float64("lat", lat),
		slog.Float64("lon", lon),
		slog.Float64("radius_km", radiusKm),
		slog.String("category", category))

	// Query database using PostGIS
	var pois []models.POIDetailedInfo
	var err error

	if category != "" {
		pois, err = h.poiRepo.GetPOIsByLocationAndDistanceWithCategory(ctx, lat, lon, radiusMeters, category)
	} else {
		pois, err = h.poiRepo.GetPOIsByLocationAndDistance(ctx, lat, lon, radiusMeters)
	}

	if err != nil {
		h.logger.ErrorContext(ctx, "Failed to get nearby POIs", slog.Any("error", err))
		c.HTML(http.StatusInternalServerError, "", `<div class="text-red-500 text-center py-8">Failed to load nearby places</div>`)
		return
	}

	h.logger.InfoContext(ctx, "Nearby POIs retrieved",
		slog.Int("count", len(pois)),
		slog.Float64("radius_km", radiusKm))

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
