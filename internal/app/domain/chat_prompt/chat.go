package llmchat

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/FACorreiaa/go-templui/internal/app/domain/profiles"
	"github.com/FACorreiaa/go-templui/internal/app/models"
	"github.com/FACorreiaa/go-templui/internal/pkg/config"
	"github.com/FACorreiaa/go-templui/internal/pkg/middleware"
)

type ChatHandlers struct {
	config         *config.Config
	llmService     LlmInteractiontService
	profileService profiles.Service
	chatRepository Repository
	logger         *zap.Logger
}

func NewChatHandlers(llmService LlmInteractiontService,
	profileService profiles.Service,
	chatRepository Repository,
	logger *zap.Logger) *ChatHandlers {
	cfg, err := config.Load()
	if err != nil {
		// Use default config if loading fails
		cfg = &config.Config{}
	}
	return &ChatHandlers{
		config:         cfg,
		llmService:     llmService,
		profileService: profileService,
		chatRepository: chatRepository,
		logger:         logger,
	}
}

// HandleChatStreamConnect creates an SSE connection setup for HTMX
func (h *ChatHandlers) HandleChatStreamConnect(c *gin.Context) {
	h.logger.Info("Chat stream connect request received",
		zap.String("ip", c.ClientIP()),
		zap.String("user", middleware.GetUserIDFromContext(c)),
	)

	// Get form parameters
	query := c.PostForm("dashboard-search")
	profileID := c.PostForm("profile-id")

	if query == "" {
		c.Header("Content-Type", "text/html")
		c.String(http.StatusBadRequest, `
			<div class="bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-700 rounded-xl p-6 mt-4">
				<div class="flex items-center gap-3">
					<svg class="w-5 h-5 text-red-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"></path>
					</svg>
					<span class="text-red-700 dark:text-red-300 text-sm">
						Please provide a search query.
					</span>
				</div>
			</div>
		`)
		return
	}

	// Get user ID for session management
	userIDStr := middleware.GetUserIDFromContext(c)
	var sessionID string

	if userIDStr != "" && userIDStr != "anonymous" {
		userID, err := uuid.Parse(userIDStr)
		if err == nil {
			// Check for existing recent sessions for this user and query combination
			sessionsResp, err := h.chatRepository.GetUserChatSessions(c.Request.Context(), userID, 1, 5) // Get recent 5 sessions
			if err != nil {
				h.logger.Warn("Failed to get user sessions, creating new one", zap.Error(err))
			} else {
				// Look for a recent session with the same query (within last 10 minutes)
				cutoffTime := time.Now().Add(-10 * time.Minute)
				for _, session := range sessionsResp.Sessions {
					if session.CreatedAt.After(cutoffTime) && session.Status == models.StatusActive {
						// Check if the last message in conversation matches current query
						if len(session.ConversationHistory) > 0 {
							lastMsg := session.ConversationHistory[len(session.ConversationHistory)-1]
							if lastMsg.Content == query && lastMsg.Role == models.RoleUser {
								sessionID = session.ID.String()
								h.logger.Info("Reusing existing session",
									zap.String("sessionID", sessionID),
									zap.String("userID", userIDStr),
									zap.String("query", query))
								break
							}
						}
					}
				}
			}
		}
	}

	// If no existing session found, create a new one
	if sessionID == "" {
		sessionID = uuid.New().String()
		h.logger.Info("Creating new session",
			zap.String("sessionID", sessionID),
			zap.String("userID", userIDStr),
			zap.String("query", query))
	}

	// Create the SSE connection URL with parameters
	sseURL := fmt.Sprintf("/chat/stream?session_id=%s&dashboard-search=%s",
		sessionID,
		url.QueryEscape(query))

	if profileID != "" {
		sseURL += "&profile-id=" + url.QueryEscape(profileID)
	}

	// Return HTML that sets up the SSE connection with event listeners
	c.Header("Content-Type", "text/html")
	c.String(http.StatusOK, fmt.Sprintf(`
		<div 
			id="sse-container"
			hx-ext="sse" 
			sse-connect="%s"
		>
			<div id="initial-loading" class="bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-700 rounded-xl p-6">
				<div class="flex items-start gap-4">
					<div class="flex-shrink-0">
						<div class="w-10 h-10 bg-gradient-to-r from-purple-500 to-pink-500 rounded-lg flex items-center justify-center">
							<svg class="w-5 h-5 text-white animate-pulse" fill="none" stroke="currentColor" viewBox="0 0 24 24">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11.049 2.927c.3-.921 1.603-.921 1.902 0l1.519 4.674a1 1 0 00.95.69h4.915c.969 0 1.371 1.24.588 1.81l-3.976 2.888a1 1 0 00-.363 1.118l1.518 4.674c.3.922-.755 1.688-1.538 1.118l-3.976-2.888a1 1 0 00-1.176 0l-3.976 2.888c-.783.57-1.838-.197-1.538-1.118l1.518-4.674a1 1 0 00-.363-1.118l-3.976-2.888c-.784-.57-.38-1.81.588-1.81h4.914a1 1 0 00.951-.69l1.519-4.674z"></path>
							</svg>
						</div>
					</div>
					<div class="flex-1">
						<h3 class="font-semibold text-blue-800 dark:text-blue-200 mb-2">AI is analyzing your request...</h3>
						<div class="space-y-2">
							<div class="flex items-center gap-3 text-blue-700 dark:text-blue-300">
								<div class="w-4 h-4 border-2 border-blue-600 border-t-transparent rounded-full animate-spin"></div>
								<span id="status-text" class="text-sm">Processing with Gemini AI (this may take 5-6 seconds)...</span>
							</div>
							<div id="streaming-content" 
								class="text-sm text-blue-600 dark:text-blue-400 space-y-1"
								sse-swap="chunk"
								hx-swap="beforeend"
							>
								<div class="text-xs opacity-75">Starting AI analysis...</div>
							</div>
						</div>
					</div>
				</div>
			</div>
			<!-- Navigation/completion target -->
			<div id="navigation-target" 
				sse-swap="navigation" 
				hx-swap="outerHTML"
			></div>
			<!-- Status updates target -->
			<div id="status-target" 
				sse-swap="status" 
				hx-swap="innerHTML"
				hx-target="#status-text"
			></div>
		</div>
		
		<script>
		(function() {
			var isProcessingComplete = false;
			var sseContainer = document.getElementById('sse-container');
			var eventSource = null;
			
			console.log('SSE event common initialized');
			
			// Function to handle navigation
			function handleNavigation(url) {
				console.log('Handling navigation to:', url);
				
				// Update status to show completion
				var statusText = document.getElementById('status-text');
				if (statusText) {
					statusText.textContent = 'Analysis complete! Redirecting to results...';
				}
				
				// Close SSE connection before navigation
				if (eventSource) {
					console.log('Closing EventSource connection');
					eventSource.close();
					eventSource = null;
				}
				
				if (sseContainer) {
					console.log('Updating SSE container');
					// Remove the sse-connect attribute to prevent reconnection
					sseContainer.removeAttribute('sse-connect');
					sseContainer.innerHTML = '<div class="text-center text-green-600">Redirecting to results...</div>';
				}
				
				// Navigate to the results page
				setTimeout(function() {
					console.log('Navigating to:', url);
					window.location.href = url;
				}, 500);
			}
			
			// Setup native EventSource connection for SSE
			function setupEventSource() {
				// Get the SSE URL from the container
				var sseUrl = sseContainer ? sseContainer.getAttribute('sse-connect') : null;
				if (!sseUrl) {
					console.error('No SSE URL found in sse-connect attribute');
					return;
				}
				
				console.log('Setting up EventSource for URL:', sseUrl);
				eventSource = new EventSource(sseUrl);
				
				eventSource.onmessage = function(event) {
					if (isProcessingComplete) {
						console.log('Ignoring message - processing already complete');
						return;
					}
					
					try {
						var eventData = JSON.parse(event.data);
						console.log('Native SSE Event received:', eventData);
						
						// Handle completion event with navigation
						if (eventData.type === 'complete' && eventData.navigation && !isProcessingComplete) {
							isProcessingComplete = true;
							console.log('Processing complete detected, navigating to:', eventData.navigation.url);
							handleNavigation(eventData.navigation.url);
							return;
						}
						
						// Handle explicit SSE close event
						if (eventData.type === 'sse-close' && !isProcessingComplete) {
							isProcessingComplete = true;
							console.log('SSE connection explicitly closed');
							
							if (eventSource) {
								eventSource.close();
								eventSource = null;
							}
							return;
						}
						
						// Update status for other events
						if (eventData.type === 'start') {
							var statusText = document.getElementById('status-text');
							if (statusText) {
								statusText.textContent = 'AI analysis started...';
							}
						} else if (eventData.type === 'chunk') {
							var statusText = document.getElementById('status-text');
							if (statusText) {
								statusText.textContent = 'Generating recommendations...';
							}
						}
						
					} catch (error) {
						console.error('Error parsing SSE event:', error);
					}
				};
				
				eventSource.onerror = function(error) {
					console.error('EventSource error:', error);
					if (!isProcessingComplete) {
						var statusText = document.getElementById('status-text');
						if (statusText) {
							statusText.textContent = 'Connection error. Please try again.';
						}
					}
				};
				
				eventSource.onopen = function(event) {
					console.log('EventSource connection opened:', event);
				};
			}
			
			// Initialize EventSource if we have an SSE container
			if (sseContainer) {
				setupEventSource();
			}
			
			// Also add fallback HTMX event listeners in case they're needed
			function addHTMXEventListeners() {
				document.body.addEventListener('htmx:sseMessage', function(e) {
					console.log('HTMX SSE message received (fallback):', e);
					if (!e.detail || !e.detail.data) return;
					
					try {
						var eventData = JSON.parse(e.detail.data);
						if (eventData.type === 'complete' && eventData.navigation && !isProcessingComplete) {
							isProcessingComplete = true;
							handleNavigation(eventData.navigation.url);
						}
					} catch (error) {
						console.error('Error parsing HTMX SSE event:', error);
					}
				});
			}
			
			addHTMXEventListeners();
			
			// Cleanup on page unload
			window.addEventListener('beforeunload', function() {
				if (eventSource) {
					eventSource.close();
				}
			});
			
		})();
		</script>
	`, sseURL))
}

// getDefaultProfileID gets the user's default profile ID
func (h *ChatHandlers) getDefaultProfileID(ctx context.Context, userID uuid.UUID) (uuid.UUID, error) {
	// Get user's profiles
	profilesResp, err := h.profileService.GetSearchProfiles(ctx, userID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to get user profiles: %w", err)
	}

	// Find default profile or return the first one
	for _, profile := range profilesResp {
		if profile.IsDefault {
			return profile.ID, nil
		}
	}

	// If no default profile found, use the first one
	if len(profilesResp) > 0 {
		return profilesResp[0].ID, nil
	}

	// If no profiles exist, return error (should create a default profile)
	return uuid.Nil, fmt.Errorf("no profiles found for user")
}

func (h *ChatHandlers) SendMessage(c *gin.Context) {
	h.logger.Info("Chat message received",
		zap.String("user", middleware.GetUserIDFromContext(c)),
		zap.String("ip", c.ClientIP()),
	)

	message := c.PostForm("message")
	sessionID := c.PostForm("session_id")

	if message == "" {
		h.logger.Warn("Empty chat message received")
		c.String(http.StatusBadRequest, `<div class="text-red-500">Message cannot be empty</div>`)
		return
	}

	h.logger.Info("Processing chat message",
		zap.String("message", message),
		zap.String("sessionID", sessionID),
		zap.String("user", middleware.GetUserIDFromContext(c)),
	)

	// Get user ID for authenticated users
	userIDStr := middleware.GetUserIDFromContext(c)
	if userIDStr == "" || userIDStr == "anonymous" {
		h.logger.Warn("Chat message from unauthenticated user")
		c.String(http.StatusUnauthorized, `
			<div class="bg-yellow-50 dark:bg-yellow-900/20 border border-yellow-200 dark:border-yellow-700 rounded-lg p-3 text-sm">
				<p class="text-yellow-700 dark:text-yellow-300">Please sign in to use the chat feature.</p>
			</div>
		`)
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		h.logger.Error("Invalid user ID", zap.String("userID", userIDStr), zap.Error(err))
		c.String(http.StatusBadRequest, `<div class="text-red-500">Invalid user session</div>`)
		return
	}

	// Get user's default profile
	profile, err := h.profileService.GetDefaultSearchProfile(c, userID)
	if err != nil {
		h.logger.Error("Failed to get user profile", zap.Error(err))
		c.String(http.StatusInternalServerError, `<div class="text-red-500">Unable to process request</div>`)
		return
	}

	// Process the message through AI service for itinerary modification
	// Create event channel for potential streaming response
	eventCh := make(chan models.StreamEvent, 100)

	// Process in goroutine
	go func() {
		err := h.llmService.ProcessUnifiedChatMessageStream(
			c.Request.Context(),
			userID,
			profile.ID,
			"", // cityName - empty for context-based
			message,
			nil, // userLocation
			eventCh,
		)
		if err != nil {
			h.logger.Error("Failed to process chat message", zap.Error(err))
			eventCh <- models.StreamEvent{
				Type:      models.EventTypeError,
				Message:   "Failed to process request",
				Error:     err.Error(),
				Timestamp: time.Now(),
				EventID:   uuid.New().String(),
			}
		}
	}()

	// For chat interface, we want to return quick response and then handle streaming updates
	// First return the AI response bubble
	response := "I'm analyzing your request and updating your itinerary. Please wait a moment..."

	c.String(http.StatusOK, fmt.Sprintf(`
		<!-- AI Response -->
		<div class="flex justify-start mb-4">
			<div class="max-w-[80%%] p-3 rounded-lg text-sm bg-gray-100 text-gray-800">
				<p class="whitespace-pre-wrap">%s</p>
				<p class="text-xs mt-1 opacity-70 text-gray-500">
					Loci AI • %s
				</p>
			</div>
		</div>

		<!-- SSE Connection for streaming updates -->
		<div id="chat-sse-container" 
			hx-ext="sse" 
			sse-connect="/chat/stream?message=%s&session_id=%s"
			style="display: none;">
			<div sse-swap="itinerary" hx-target="#itinerary-container" hx-swap="outerHTML"></div>
		</div>
	`, response, time.Now().Format("15:04"), message, sessionID))

	h.logger.Info("Chat message processed successfully",
		zap.String("user", middleware.GetUserIDFromContext(c)),
	)
}

// HandleSearch processes search requests from the landing page and redirects to appropriate domain
func (h *ChatHandlers) HandleSearch(c *gin.Context) {
	h.logger.Info("Search request received",
		zap.String("ip", c.ClientIP()),
		zap.String("user_agent", c.GetHeader("User-Agent")),
	)

	query := c.PostForm("search-input")
	if query == "" {
		h.logger.Warn("Empty search query received")
		c.String(http.StatusBadRequest, `
			<div class="bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-700 rounded-lg p-4 mb-4">
				<div class="flex items-center gap-2">
					<svg class="w-5 h-5 text-red-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"></path>
					</svg>
					<span class="text-red-700 dark:text-red-300 text-sm">Please enter a search query to discover amazing places.</span>
				</div>
			</div>
		`)
		return
	}

	h.logger.Info("Processing search query",
		zap.String("query", query),
		zap.String("ip", c.ClientIP()),
	)

	// Detect domain using DomainDetector
	domainDetector := &models.DomainDetector{}
	domain := domainDetector.DetectDomain(context.Background(), query)

	h.logger.Info("Domain detected",
		zap.String("query", query),
		zap.String("detected_domain", string(domain)),
	)

	// Map domain to appropriate URL and redirect
	redirectURL := h.mapDomainToURL(domain, query)

	// Return HTMX redirect response
	c.Header("HX-Redirect", redirectURL)
	c.String(http.StatusOK, fmt.Sprintf(`
		<div class="bg-green-50 dark:bg-green-900/20 border border-green-200 dark:border-green-700 rounded-lg p-4 mb-4">
			<div class="flex items-center gap-2">
				<svg class="w-5 h-5 text-green-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"></path>
				</svg>
				<span class="text-green-700 dark:text-green-300 text-sm">Redirecting to %s results...</span>
			</div>
		</div>
	`, string(domain)))
}

// callLLMStreamingServiceWithData calls the app LLM service and returns both data and redirect URL
func (h *ChatHandlers) callLLMStreamingServiceWithData(query, userID string) (map[string]interface{}, string, error) {
	ctx := context.Background()

	// Parse user ID
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, "", fmt.Errorf("invalid user ID: %w", err)
	}

	// Get profile ID
	profileID, err := h.getDefaultProfileID(ctx, userUUID)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get user profile: %w", err)
	}

	// Create event channel for streaming
	eventCh := make(chan models.StreamEvent, 100)

	// Process the request using app service
	go func() {
		// Don't close eventCh here - let the service handle it

		err := h.llmService.ProcessUnifiedChatMessageStream(
			ctx,
			userUUID,
			profileID,
			"", // cityName - empty for auto-detection
			query,
			nil, // userLocation
			eventCh,
		)
		if err != nil {
			h.logger.Error("Internal LLM service error", zap.Error(err))
		}
	}()

	// Collect all events to extract domain and data
	var llmData map[string]interface{}
	domain := "activities" // default

	// Wait for events and extract data
	for event := range eventCh {
		switch event.Type {
		case models.EventTypeDomainDetected:
			if data, ok := event.Data.(map[string]interface{}); ok {
				if detectedDomain, ok := data["domain"].(string); ok {
					domain = strings.ToLower(detectedDomain)
				}
			}
		case models.EventTypeItinerary:
			if event.Data != nil {
				// Use the data directly as map[string]interface{}
				if itineraryMap, ok := event.Data.(map[string]interface{}); ok {
					llmData = itineraryMap
				}
			}
		case models.EventTypeHotels, models.EventTypeRestaurants:
			if event.Data != nil {
				llmData = event.Data.(map[string]interface{})
			}
		case models.EventTypeComplete:
			if event.Navigation != nil {
				domain = event.Navigation.RouteType
			}
		}
	}

	// Map domain to URL (legacy function for string domains)
	redirectURL := h.mapDomainToURLLegacy(domain)
	return llmData, redirectURL, nil
}

// mapDomainToURL maps domain types to appropriate URLs with query parameters
func (h *ChatHandlers) mapDomainToURL(domain models.DomainType, query string) string {
	// URL encode the query
	encodedQuery := url.QueryEscape(query)

	switch domain {
	case models.DomainAccommodation:
		return fmt.Sprintf("/hotels?q=%s", encodedQuery)
	case models.DomainDining:
		return fmt.Sprintf("/restaurants?q=%s", encodedQuery)
	case models.DomainActivities:
		return fmt.Sprintf("/activities?q=%s", encodedQuery)
	case models.DomainItinerary:
		return fmt.Sprintf("/itinerary?q=%s", encodedQuery)
	case models.DomainGeneral:
		// For general queries, default to activities page
		return fmt.Sprintf("/activities?q=%s", encodedQuery)
	default:
		// Default fallback to activities
		return fmt.Sprintf("/activities?q=%s", encodedQuery)
	}
}

// Legacy mapDomainToURL for string-based domains (used by other functions)
func (h *ChatHandlers) mapDomainToURLLegacy(domain string) string {
	d := strings.ToLower(domain)
	switch d {
	case "dining", "restaurants", "food", "restaurant":
		return "/restaurants"
	case "accommodation", "hotels", "hotel", "stay":
		return "/hotels"
	case "itinerary", "planning", "plan":
		return "/itinerary"
	case "activities", "activity", "attractions", "things_to_do":
		return "/activities"
	default:
		return "/activities"
	}
}

// HandleDiscover processes discovery requests from the dashboard and integrates with LLM streaming
func (h *ChatHandlers) HandleDiscover(c *gin.Context) {
	h.logger.Info("Discover request received",
		zap.String("ip", c.ClientIP()),
		zap.String("user_agent", c.GetHeader("User-Agent")),
	)

	query := c.PostForm("dashboard-search")
	if query == "" {
		h.logger.Warn("Empty discover query received")
		c.String(http.StatusBadRequest, `
			<div class="bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-700 rounded-lg p-4 mb-4">
				<div class="flex items-center gap-2">
					<svg class="w-5 h-5 text-red-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"></path>
					</svg>
					<span class="text-red-700 dark:text-red-300 text-sm">Please describe what you'd like to discover.</span>
				</div>
			</div>
		`)
		return
	}

	h.logger.Info("Processing discover query",
		zap.String("query", query),
		zap.String("ip", c.ClientIP()),
	)

	// Get user ID from middleware context
	userID := middleware.GetUserIDFromContext(c)
	if userID == "" {
		h.logger.Error("User ID not found in context")
		c.String(http.StatusUnauthorized, "Authentication required")
		return
	}

	// Call LLM service and get streaming data
	llmData, redirectURL, err := h.callLLMStreamingServiceWithData(query, userID)
	if err != nil {
		h.logger.Warn("LLM service unavailable, using fallback classification", zap.Error(err))
		// Fallback to local intent classification
		intent := h.classifyIntent(query)
		redirectURL = h.getRedirectURL(intent)
		// Set redirect without data
		c.Header("HX-Redirect", redirectURL+"?q="+query)
		c.String(http.StatusOK, h.generateDiscoveryResponseByURL(query, redirectURL))
		return
	}

	// Store LLM data in session for the destination page
	sessionKey := fmt.Sprintf("llm_data_%s", userID)
	// In a real app, you'd use Redis or similar. For now, we'll pass data via different means
	h.logger.Info("LLM data received",
		zap.String("query", query),
		zap.String("redirect_url", redirectURL),
		zap.Bool("has_data", llmData != nil))

	// Store the LLM response data in a way the destination page can access it
	// For now, encode it in the URL or use session storage approach
	encodedData := ""
	if llmData != nil {
		if _, err := json.Marshal(llmData); err == nil {
			// For simplicity, we'll pass a session identifier and store data server-side
			// In production, use proper session management
			encodedData = fmt.Sprintf("&session_data=%s", sessionKey)

		}
	}

	// Set HTMX redirect header with session data
	c.Header("HX-Redirect", redirectURL+"?q="+query+encodedData)
	c.String(http.StatusOK, h.generateDiscoveryResponseByURL(query, redirectURL))
}

// classifyIntent analyzes the query to determine user intent
func (h *ChatHandlers) classifyIntent(query string) string {
	queryLower := strings.ToLower(query)

	// Score-based approach to handle overlapping keywords
	scores := map[string]int{
		"restaurants": 0,
		"hotels":      0,
		"activities":  0,
		"itinerary":   0,
	}

	// Restaurant/food keywords (higher specificity scores)
	restaurantKeywords := map[string]int{
		"dinner": 3, "lunch": 3, "breakfast": 3, "restaurant": 3, "cafe": 3, "bar": 3,
		"food": 2, "eat": 2, "dining": 2, "meal": 2, "kitchen": 2, "cuisine": 2, "taste": 2, "flavor": 2, "dish": 2, "menu": 2,
		"cooking": 1, "chef": 1, "wine": 1, "beer": 1, "cocktail": 1,
	}

	// Hotel/accommodation keywords
	hotelKeywords := map[string]int{
		"hotel": 3, "stay": 3, "accommodation": 3, "room": 3, "lodge": 3, "resort": 3, "inn": 3, "booking": 3,
		"sleep": 2, "night": 1, "bed": 1,
	}

	// Activity keywords
	activityKeywords := map[string]int{
		"museum": 3, "attraction": 3, "tour": 3, "adventure": 3, "entertainment": 3, "park": 3,
		"visit": 2, "see": 2, "explore": 2, "activity": 2, "activities": 2, "experience": 2, "fun": 2,
		"do": 1, "go": 1, "walk": 1, "hiking": 2, "cultural": 2, "art": 2, "history": 2,
	}

	// Itinerary/planning keywords
	itineraryKeywords := map[string]int{
		"plan": 3, "itinerary": 3, "schedule": 3, "trip": 3, "journey": 3, "route": 3,
		"day": 2, "weekend": 2, "guide": 2, "organize": 2,
		"week": 1, "vacation": 2, "holiday": 2,
	}

	// Calculate scores for each category
	for keyword, weight := range restaurantKeywords {
		if strings.Contains(queryLower, keyword) {
			scores["restaurants"] += weight
		}
	}

	for keyword, weight := range hotelKeywords {
		if strings.Contains(queryLower, keyword) {
			scores["hotels"] += weight
		}
	}

	for keyword, weight := range activityKeywords {
		if strings.Contains(queryLower, keyword) {
			scores["activities"] += weight
		}
	}

	for keyword, weight := range itineraryKeywords {
		if strings.Contains(queryLower, keyword) {
			scores["itinerary"] += weight
		}
	}

	// Find the category with highest score
	maxScore := 0
	result := "activities" // default

	for category, score := range scores {
		if score > maxScore {
			maxScore = score
			result = category
		}
	}

	return result
}

// getRedirectURL returns the appropriate page URL based on intent
func (h *ChatHandlers) getRedirectURL(intent string) string {
	switch intent {
	case "restaurants":
		return "/restaurants"
	case "hotels":
		return "/hotels"
	case "activities":
		return "/activities"
	case "itinerary":
		return "/itinerary"
	default:
		return "/activities"
	}
}

// generateDiscoveryResponseByURL creates response based on redirect URL
func (h *ChatHandlers) generateDiscoveryResponseByURL(query, redirectURL string) string {
	var intentLabel string
	var icon string

	switch redirectURL {
	case "/restaurants":
		intentLabel = "Food & Dining"
		icon = "🍽️"
	case "/hotels":
		intentLabel = "Hotels & Accommodation"
		icon = "🏨"
	case "/activities":
		intentLabel = "Activities & Attractions"
		icon = "🎯"
	case "/itinerary":
		intentLabel = "Travel Planning"
		icon = "📋"
	default:
		intentLabel = "Discovery"
		icon = "✨"
	}

	return fmt.Sprintf(`
		<div class="bg-gradient-to-r from-blue-50 to-purple-50 dark:from-gray-800 dark:to-gray-700 rounded-xl p-6 border mb-4">
			<div class="flex items-center gap-4">
				<div class="text-4xl">%s</div>
				<div class="flex-1">
					<h3 class="font-semibold text-card-foreground mb-1">Taking you to %s</h3>
					<p class="text-sm text-muted-foreground">Based on AI analysis: "%s"</p>
					<div class="mt-3">
						<div class="flex items-center gap-2 text-blue-600">
							<div class="w-4 h-4 border-2 border-blue-600 border-t-transparent rounded-full animate-spin"></div>
							<span class="text-sm font-medium">AI processed your request - redirecting to personalized recommendations...</span>
						</div>
					</div>
				</div>
			</div>
		</div>
	`, icon, intentLabel, query)
}

// ProcessUnifiedChatMessageStream handles unified chat message streaming - equivalent to your old REST API method
func (h *ChatHandlers) ProcessUnifiedChatMessageStream(c *gin.Context) {
	h.logger.Info("Unified chat message stream request received",
		zap.String("ip", c.ClientIP()),
		zap.String("user", middleware.GetUserIDFromContext(c)),
	)

	// Get message from form data (POST) or query parameters (GET for SSE)
	var message string
	if c.Request.Method == "GET" {
		// For SSE connections, get from query params
		message = c.Query("dashboard-search")
		if message == "" {
			message = c.Query("message")
		}
	} else {
		// For POST requests, get from form data
		message = c.PostForm("message")
		if message == "" {
			message = c.PostForm("dashboard-search")
		}
	}

	if message == "" {
		h.logger.Warn("Empty message for chat stream")
		c.String(http.StatusBadRequest, "Message parameter is required")
		return
	}

	// Get user ID for authenticated users
	userIDStr := middleware.GetUserIDFromContext(c)
	fmt.Printf("userIDStr: %s\n", userIDStr)
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		h.logger.Error("Invalid user ID", zap.String("userID", userIDStr), zap.Error(err))
		c.String(http.StatusBadRequest, "Invalid user ID")
		return
	}
	// Check if a specific profile ID was provided in form or query params
	var profileID uuid.UUID
	var profileIDStr string
	if c.Request.Method == "GET" {
		profileIDStr = c.Query("profile-id")
	} else {
		profileIDStr = c.PostForm("profile-id")
	}
	if profileIDStr != "" {
		// Use the provided profile ID
		parsedProfileID, err := uuid.Parse(profileIDStr)
		if err != nil {
			h.logger.Error("Invalid profile ID provided", zap.String("profileID", profileIDStr), zap.Error(err))
			c.String(http.StatusBadRequest, "Invalid profile ID")
			return
		}

		// Verify the profile belongs to this user
		profile, err := h.profileService.GetSearchProfile(c, userID, parsedProfileID)
		if err != nil {
			h.logger.Error("Profile not found or doesn't belong to user",
				zap.String("userID", userID.String()),
				zap.String("profileID", profileIDStr),
				zap.Error(err))

			// Provide user-friendly error based on error type
			if errors.Is(err, models.ErrNotFound) {
				c.String(http.StatusNotFound, "Profile not found or you don't have access to it")
			} else if errors.Is(err, models.ErrForbidden) {
				c.String(http.StatusForbidden, "Access denied to this profile")
			} else {
				c.String(http.StatusInternalServerError, "Unable to verify profile access")
			}
			return
		}
		profileID = profile.ID
	} else {
		// Use default profile
		profile, err := h.profileService.GetDefaultSearchProfile(c, userID)
		if err != nil {
			h.logger.Error("Failed to get default profile", zap.Error(err))

			// Provide specific error messages
			if errors.Is(err, models.ErrNotFound) {
				c.String(http.StatusNotFound, "No default profile found. Please create a profile first.")
			} else {
				c.String(http.StatusInternalServerError, "Unable to retrieve user profile")
			}
			return
		}
		profileID = profile.ID
	}
	fmt.Printf("profileID: %s\n", profileID.String())

	// Set SSE headers
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")
	c.Header("Access-Control-Allow-Headers", "Cache-Control")
	c.Header("X-Accel-Buffering", "no") // Disable nginx buffering

	// Set up flusher for real-time streaming
	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		h.logger.Error("Response writer does not support flushing")
		c.String(http.StatusInternalServerError, "Streaming not supported")
		return
	}

	// Create event channel for streaming (larger buffer for Gemini SDK)
	eventCh := make(chan models.StreamEvent, 200)

	// Process the request in a goroutine
	go func() {
		// Don't close eventCh here - let the service handle it
		h.logger.Info("Processing authenticated user request",
			zap.String("userID", userID.String()),
			zap.String("profileID", profileID.String()),
			zap.String("message", message))

		// Call the LLM service with proper user and profile IDs
		err := h.llmService.ProcessUnifiedChatMessageStream(
			middleware.CreateContextWithUser(c),
			userID,
			profileID,
			"", // cityName - empty for auto-detection
			message,
			nil, // userLocation
			eventCh,
		)
		if err != nil {
			h.logger.Error("Failed to process authenticated chat stream", zap.Error(err))
			eventCh <- models.StreamEvent{
				Type:      models.EventTypeError,
				Message:   "Failed to process request",
				Error:     err.Error(),
				Timestamp: time.Now(),
				EventID:   uuid.New().String(),
			}
		}
	}()

	// Stream events to client as they arrive
	for {
		select {
		case event, ok := <-eventCh:
			if !ok {
				h.logger.Info("Event channel closed, ending stream")
				return
			}

			eventData, err := json.Marshal(event)
			if err != nil {
				h.logger.Error("Failed to marshal event", zap.Error(err))
				continue
			}

			// Print streamed response to terminal
			fmt.Printf("SSE >> %s\n", eventData)

			//debugger.DebugPrintEvents(_, eventData)
			//filePath := "events.json" // Change to "events.txt" for text format
			//f, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			//if err != nil {
			//	h.logger.Error("Failed to open file", zap.Error(err), zap.String("file", filePath))
			//	continue
			//}
			//defer f.Close()

			// For JSON: Write eventData with a newline for separation
			//if _, err := f.Write(append(eventData, '\n')); err != nil {
			//	h.logger.Error("Failed to write to file", zap.Error(err), zap.String("file", filePath))
			//	continue
			//}

			fmt.Fprintf(c.Writer, "data: %s\n\n", eventData)
			flusher.Flush()

			// End stream on complete or error
			if event.Type == models.EventTypeComplete || event.Type == models.EventTypeError {
				h.logger.Info("Stream completed", zap.String("eventType", event.Type))

				// Send a final SSE close message to help HTMX understand the connection is intentionally closed
				fmt.Fprintf(c.Writer, "data: {\"type\":\"sse-close\"}\n\n")
				flusher.Flush()

				return
			}

		case <-c.Request.Context().Done():
			h.logger.Info("Client disconnected")
			return
		}
	}
}

// HandleChatStream handles SSE streaming for chat messages in itinerary modification
func (h *ChatHandlers) HandleChatStream(c *gin.Context) {
	h.logger.Info("Chat stream request received",
		zap.String("ip", c.ClientIP()),
		zap.String("user", middleware.GetUserIDFromContext(c)),
	)

	// Support both parameter names for backward compatibility
	message := c.Query("message")
	if message == "" {
		message = c.Query("dashboard-search")
	}

	sessionID := c.Query("session_id")

	if message == "" {
		h.logger.Warn("Empty message for chat stream")
		c.String(http.StatusBadRequest, "Message parameter is required (use 'message' or 'dashboard-search')")
		return
	}

	// Get user ID for authenticated users
	userIDStr := middleware.GetUserIDFromContext(c)
	if userIDStr == "" || userIDStr == "anonymous" {
		h.logger.Warn("Chat stream from unauthenticated user")
		c.String(http.StatusUnauthorized, "Authentication required")
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		h.logger.Error("Invalid user ID", zap.String("userID", userIDStr), zap.Error(err))
		c.String(http.StatusBadRequest, "Invalid user ID")
		return
	}

	// Get user's default profile
	profile, err := h.profileService.GetDefaultSearchProfile(c, userID)
	if err != nil {
		h.logger.Error("Failed to get user profile", zap.Error(err))
		c.String(http.StatusInternalServerError, "Unable to retrieve user profile")
		return
	}

	// Set SSE headers
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")
	c.Header("Access-Control-Allow-Headers", "Cache-Control")
	c.Header("X-Accel-Buffering", "no") // Disable nginx buffering

	// Set up flusher for real-time streaming
	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		h.logger.Error("Response writer does not support flushing")
		c.String(http.StatusInternalServerError, "Streaming not supported")
		return
	}

	// Create event channel for streaming
	eventCh := make(chan models.StreamEvent, 200)

	// Process the request in a goroutine
	go func() {
		h.logger.Info("Processing chat stream request",
			zap.String("userID", userID.String()),
			zap.String("profileID", profile.ID.String()),
			zap.String("message", message),
			zap.String("sessionID", sessionID))

		// Call the LLM service for itinerary modification
		err := h.llmService.ProcessUnifiedChatMessageStream(
			c.Request.Context(),
			userID,
			profile.ID,
			"", // cityName - empty for context-based
			message,
			nil, // userLocation
			eventCh,
		)
		if err != nil {
			h.logger.Error("Failed to process chat stream", zap.Error(err))
			eventCh <- models.StreamEvent{
				Type:      models.EventTypeError,
				Message:   "Failed to process request",
				Error:     err.Error(),
				Timestamp: time.Now(),
				EventID:   uuid.New().String(),
			}
		}
	}()

	// Stream events to client as they arrive
	for {
		select {
		case event, ok := <-eventCh:
			if !ok {
				h.logger.Info("Chat stream channel closed, ending stream")
				return
			}

			eventData, err := json.Marshal(event)
			if err != nil {
				h.logger.Error("Failed to marshal chat event", zap.Error(err))
				continue
			}

			// Print streamed response to terminal
			fmt.Printf("Chat SSE >> %s\n", eventData)

			fmt.Fprintf(c.Writer, "data: %s\n\n", eventData)
			flusher.Flush()

			// End stream on complete or error
			if event.Type == models.EventTypeComplete || event.Type == models.EventTypeError {
				h.logger.Info("Chat stream completed", zap.String("eventType", event.Type))

				// Send a final SSE close message
				fmt.Fprintf(c.Writer, "data: {\"type\":\"sse-close\"}\n\n")
				flusher.Flush()

				return
			}

		case <-c.Request.Context().Done():
			h.logger.Info("Chat stream client disconnected")
			return
		}
	}
}

// HandleItineraryStream handles SSE streaming for itinerary queries using app service
func (h *ChatHandlers) HandleItineraryStream(c *gin.Context) {
	h.logger.Info("Itinerary stream request received",
		zap.String("ip", c.ClientIP()),
		zap.String("user", middleware.GetUserIDFromContext(c)),
	)

	message := c.Query("message")
	if message == "" {
		h.logger.Warn("Empty message for itinerary stream")
		c.String(http.StatusBadRequest, "Message parameter is required")
		return
	}

	// Set SSE headers
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")
	c.Header("Access-Control-Allow-Headers", "Cache-Control")

	// Get user info (same logic as ProcessUnifiedChatMessageStream)
	userIDStr := middleware.GetUserIDFromContext(c)
	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.String(http.StatusInternalServerError, "Streaming unsupported")
		return
	}

	// Create event channel for streaming
	eventCh := make(chan models.StreamEvent, 100)

	// Process the request in a goroutine
	go func() {
		// Don't close eventCh here - let the service handle it

		if userIDStr != "" && userIDStr != "anonymous" {
			// Authenticated user path
			userID, err := uuid.Parse(userIDStr)
			if err != nil {
				eventCh <- models.StreamEvent{
					Type:    models.EventTypeError,
					Message: "Invalid user ID",
					Error:   err.Error(),
				}
				return
			}

			// Get profile ID
			profileID, err := h.getDefaultProfileID(c.Request.Context(), userID)
			if err != nil {
				eventCh <- models.StreamEvent{
					Type:    models.EventTypeError,
					Message: "Failed to get user profile",
					Error:   err.Error(),
				}
				return
			}

			// Call the actual LLM service with profile
			err = h.llmService.ProcessUnifiedChatMessageStream(
				c.Request.Context(),
				userID,
				profileID,
				"", // cityName - empty for auto-detection
				message,
				nil, // userLocation
				eventCh,
			)
			if err != nil {
				eventCh <- models.StreamEvent{
					Type:    models.EventTypeError,
					Message: "LLM service error",
					Error:   err.Error(),
				}
			}
		} else {
			// Free/unauthenticated user path
			err := h.llmService.ProcessUnifiedChatMessageStreamFree(
				c.Request.Context(),
				"", // cityName
				message,
				nil, // userLocation
				eventCh,
			)
			if err != nil {
				eventCh <- models.StreamEvent{
					Type:    models.EventTypeError,
					Message: "LLM service error",
					Error:   err.Error(),
				}
			}
		}
	}()

	// Stream events to client as they arrive
	for {
		select {
		case event, ok := <-eventCh:
			if !ok {
				// Channel closed, streaming complete
				h.logger.Info("Itinerary stream completed",
					zap.String("message", message),
					zap.String("user", userIDStr),
				)
				return
			}

			// Convert event to JSON and send via SSE
			eventData, err := json.Marshal(event)
			if err != nil {
				h.logger.Error("Failed to marshal event", zap.Error(err))
				continue
			}

			// Print streamed response to terminal
			fmt.Printf("SSE >> %s\n", eventData)

			fmt.Fprintf(c.Writer, "data: %s\n\n", eventData)
			flusher.Flush()

		case <-c.Request.Context().Done():
			// Client disconnected
			h.logger.Info("Client disconnected from itinerary stream")
			return
		}
	}
}

// ContinueChatSession handles continuing an existing chat session with HTMX SSE
func (h *ChatHandlers) ContinueChatSession(c *gin.Context) {
	h.logger.Info("Continue chat session request received",
		zap.String("ip", c.ClientIP()),
		zap.String("user", middleware.GetUserIDFromContext(c)),
	)

	// Get session ID from URL parameter
	sessionIDStr := c.Param("sessionID")
	sessionID, err := uuid.Parse(sessionIDStr)
	if err != nil {
		h.logger.Error("Invalid session ID", zap.String("sessionID", sessionIDStr), zap.Error(err))
		c.HTML(http.StatusBadRequest, "", `<div class="text-red-500 text-sm p-4">Invalid session ID</div>`)
		return
	}

	// Parse request body
	var req struct {
		Message      string               `json:"message" form:"message"`
		UserLocation *models.UserLocation `json:"user_location,omitempty" form:"user_location"`
	}

	// Support both JSON and form-encoded requests (for HTMX)
	contentType := c.GetHeader("Content-Type")
	if strings.Contains(contentType, "application/json") {
		if err := c.ShouldBindJSON(&req); err != nil {
			h.logger.Error("Failed to decode request body", zap.Error(err))
			c.HTML(http.StatusBadRequest, "", `<div class="text-red-500 text-sm p-4">Invalid request body</div>`)
			return
		}
	} else {
		// Form-encoded for HTMX
		req.Message = c.PostForm("message")
	}

	if req.Message == "" {
		h.logger.Error("Message is required")
		c.HTML(http.StatusBadRequest, "", `<div class="text-red-500 text-sm p-4">Message is required</div>`)
		return
	}

	h.logger.Info("Processing continue chat session request",
		zap.String("sessionID", sessionID.String()),
		zap.String("message", req.Message),
	)

	// Set up SSE headers
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")
	c.Header("Access-Control-Allow-Headers", "Cache-Control")
	c.Header("X-Accel-Buffering", "no") // Disable nginx buffering

	// Get flusher
	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		h.logger.Error("Response writer does not support flushing")
		c.HTML(http.StatusInternalServerError, "", `<div class="text-red-500 text-sm p-4">Streaming not supported</div>`)
		return
	}

	// Create event channel
	eventCh := make(chan models.StreamEvent, 100)

	// Start processing in a goroutine
	go func() {
		defer close(eventCh)

		err := h.llmService.ContinueSessionStreamed(
			c.Request.Context(),
			sessionID,
			req.Message,
			req.UserLocation,
			eventCh,
		)
		if err != nil {
			h.logger.Error("Failed to continue session", zap.Error(err))
			// Send error event
			select {
			case eventCh <- models.StreamEvent{
				Type:      models.EventTypeError,
				Error:     err.Error(),
				Timestamp: time.Now(),
				EventID:   uuid.New().String(),
				IsFinal:   true,
			}:
			case <-c.Request.Context().Done():
				return
			}
		}
	}()

	// Process events in real-time as they arrive
	for {
		select {
		case event, ok := <-eventCh:
			if !ok {
				h.logger.Info("Event channel closed, ending stream")
				return
			}

			eventData, err := json.Marshal(event)
			if err != nil {
				h.logger.Error("Failed to marshal event", zap.Error(err))
				continue
			}

			// Print streamed response to terminal
			fmt.Printf("SSE >> %s\n", eventData)

			fmt.Fprintf(c.Writer, "data: %s\n\n", eventData)
			flusher.Flush()

			if event.Type == models.EventTypeComplete || event.Type == models.EventTypeError {
				h.logger.Info("Stream completed", zap.String("eventType", event.Type))
				return
			}

		case <-c.Request.Context().Done():
			// Client disconnected
			h.logger.Info("Client disconnected from continue session stream")
			return
		}
	}
}
