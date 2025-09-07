package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/FACorreiaa/go-templui/app/pkg/config"
	"github.com/FACorreiaa/go-templui/app/pkg/logger"
	"github.com/FACorreiaa/go-templui/app/pkg/middleware"
)

type ChatHandlers struct{
	config *config.Config
}

func NewChatHandlers() *ChatHandlers {
	cfg, _ := config.Load()
	return &ChatHandlers{
		config: cfg,
	}
}

func (h *ChatHandlers) SendMessage(c *gin.Context) {
	logger.Log.Info("Chat message received",
		zap.String("user", middleware.GetUserIDFromContext(c)),
		zap.String("ip", c.ClientIP()),
	)

	message := c.PostForm("message")
	if message == "" {
		logger.Log.Warn("Empty chat message received")
		c.String(http.StatusBadRequest, `<div class="text-red-500">Message cannot be empty</div>`)
		return
	}

	logger.Log.Info("Processing chat message",
		zap.String("message", message),
		zap.String("user", middleware.GetUserIDFromContext(c)),
	)

	// Simulate AI response (in real app, this would call your AI service)
	response := "Thanks for your message! I'm here to help you discover amazing places. What would you like to explore?"

	// Return HTMX response with user message and AI response
	c.String(http.StatusOK, `
		<!-- User Message -->
		<div class="flex justify-end mb-4">
			<div class="max-w-xs sm:max-w-lg">
				<div class="bg-blue-600 text-white rounded-2xl rounded-tr-md p-3">
					<p class="text-sm">`+message+`</p>
				</div>
				<p class="text-xs text-muted-foreground mt-1 mr-1 text-right">You • now</p>
			</div>
		</div>

		<!-- AI Response -->
		<div class="flex items-start gap-3 mb-4">
			<div class="w-8 h-8 bg-gradient-to-r from-purple-500 to-pink-500 rounded-full flex items-center justify-center flex-shrink-0">
				<svg class="w-4 h-4 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11.049 2.927c.3-.921 1.603-.921 1.902 0l1.519 4.674a1 1 0 00.95.69h4.915c.969 0 1.371 1.24.588 1.81l-3.976 2.888a1 1 0 00-.363 1.118l1.518 4.674c.3.922-.755 1.688-1.538 1.118l-3.976-2.888a1 1 0 00-1.176 0l-3.976 2.888c-.783.57-1.838-.197-1.538-1.118l1.518-4.674a1 1 0 00-.363-1.118l-3.976-2.888c-.784-.57-.38-1.81.588-1.81h4.914a1 1 0 00.951-.69l1.519-4.674z"></path>
				</svg>
			</div>
			<div class="flex-1 max-w-xs sm:max-w-lg">
				<div class="bg-accent rounded-2xl rounded-tl-md p-3">
					<p class="text-sm text-card-foreground">`+response+`</p>
				</div>
				<p class="text-xs text-muted-foreground mt-1 ml-1">Loci AI • now</p>
			</div>
		</div>
	`)

	logger.Log.Info("Chat message processed successfully",
		zap.String("user", middleware.GetUserIDFromContext(c)),
	)
}

// HandleSearch processes search requests from the landing page and returns streaming AI responses
func (h *ChatHandlers) HandleSearch(c *gin.Context) {
	logger.Log.Info("Search request received",
		zap.String("ip", c.ClientIP()),
		zap.String("user_agent", c.GetHeader("User-Agent")),
	)

	query := c.PostForm("search-input")
	if query == "" {
		logger.Log.Warn("Empty search query received")
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

	logger.Log.Info("Processing search query",
		zap.String("query", query),
		zap.String("ip", c.ClientIP()),
	)

	// Call the LLM streaming service
	go func() {
		time.Sleep(100 * time.Millisecond) // Small delay to ensure client receives the response
		h.streamLLMResponse(query, c.ClientIP())
	}()

	// Return immediate response with streaming placeholder
	c.String(http.StatusOK, h.generateStreamingResponse(query))
}

// streamLLMResponse calls the LLM streaming endpoint and processes the results
func (h *ChatHandlers) streamLLMResponse(query, clientIP string) {
	// Prepare request to LLM service
	requestBody := map[string]interface{}{
		"message": query,
		"user_location": nil,
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		logger.Log.Error("Failed to marshal request", zap.Error(err))
		return
	}

	// Make request to LLM streaming endpoint
	llmEndpoint := "http://localhost:8000/api/v1/llm/chat/stream/free"
	if h.config != nil && h.config.LLM.StreamEndpoint != "" {
		llmEndpoint = h.config.LLM.StreamEndpoint + "/chat/stream/free"
	}

	resp, err := http.Post(llmEndpoint, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		logger.Log.Error("Failed to call LLM service", 
			zap.Error(err),
			zap.String("endpoint", llmEndpoint),
			zap.String("query", query))
		return
	}
	defer resp.Body.Close()

	// Process streaming response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Log.Error("Failed to read LLM response", zap.Error(err))
		return
	}

	logger.Log.Info("LLM response received",
		zap.String("query", query),
		zap.String("client_ip", clientIP),
		zap.Int("response_size", len(body)))
}

// HandleDiscover processes discovery requests from the dashboard and routes based on intent
func (h *ChatHandlers) HandleDiscover(c *gin.Context) {
	logger.Log.Info("Discover request received",
		zap.String("ip", c.ClientIP()),
		zap.String("user_agent", c.GetHeader("User-Agent")),
	)

	query := c.PostForm("dashboard-search")
	if query == "" {
		logger.Log.Warn("Empty discover query received")
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

	logger.Log.Info("Processing discover query",
		zap.String("query", query),
		zap.String("ip", c.ClientIP()),
	)

	// Classify intent and determine appropriate route
	intent := h.classifyIntent(query)
	redirectURL := h.getRedirectURL(intent)

	// Return HTMX response with redirect
	c.Header("HX-Redirect", redirectURL+"?q="+query)
	c.String(http.StatusOK, h.generateDiscoveryResponse(query, intent, redirectURL))
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

// generateDiscoveryResponse creates the response before redirect
func (h *ChatHandlers) generateDiscoveryResponse(query, intent, redirectURL string) string {
	var intentLabel string
	var icon string
	
	switch intent {
	case "restaurants":
		intentLabel = "Food & Dining"
		icon = "🍽️"
	case "hotels":
		intentLabel = "Hotels & Accommodation"
		icon = "🏨"
	case "activities":
		intentLabel = "Activities & Attractions"
		icon = "🎯"
	case "itinerary":
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
					<p class="text-sm text-muted-foreground">Based on your query: "%s"</p>
					<div class="mt-3">
						<div class="flex items-center gap-2 text-blue-600">
							<div class="w-4 h-4 border-2 border-blue-600 border-t-transparent rounded-full animate-spin"></div>
							<span class="text-sm font-medium">Redirecting to personalized recommendations...</span>
						</div>
					</div>
				</div>
			</div>
		</div>
	`, icon, intentLabel, query)
}

// generateStreamingResponse creates the initial HTML response with streaming content
func (h *ChatHandlers) generateStreamingResponse(query string) string {
	return fmt.Sprintf(`
		<div class="max-w-4xl mx-auto mt-8 space-y-6">
			<!-- Search Query Display -->
			<div class="bg-card rounded-xl p-6 shadow-lg border">
				<div class="flex items-start gap-4">
					<div class="flex-shrink-0">
						<div class="w-10 h-10 bg-blue-100 dark:bg-blue-900/30 rounded-lg flex items-center justify-center">
							<svg class="w-5 h-5 text-blue-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"></path>
							</svg>
						</div>
					</div>
					<div class="flex-1">
						<h3 class="font-semibold text-card-foreground mb-1">Your Search</h3>
						<p class="text-muted-foreground">"%s"</p>
					</div>
				</div>
			</div>

			<!-- AI Response Section -->
			<div class="bg-card rounded-xl p-6 shadow-lg border">
				<div class="flex items-start gap-4">
					<div class="flex-shrink-0">
						<div class="w-10 h-10 bg-gradient-to-r from-purple-500 to-pink-500 rounded-lg flex items-center justify-center">
							<svg class="w-5 h-5 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11.049 2.927c.3-.921 1.603-.921 1.902 0l1.519 4.674a1 1 0 00.95.69h4.915c.969 0 1.371 1.24.588 1.81l-3.976 2.888a1 1 0 00-.363 1.118l1.518 4.674c.3.922-.755 1.688-1.538 1.118l-3.976-2.888a1 1 0 00-1.176 0l-3.976 2.888c-.783.57-1.838-.197-1.538-1.118l1.518-4.674a1 1 0 00-.363-1.118l-3.976-2.888c-.784-.57-.38-1.81.588-1.81h4.914a1 1 0 00.951-.69l1.519-4.674z"></path>
							</svg>
						</div>
					</div>
					<div class="flex-1">
						<h3 class="font-semibold text-card-foreground mb-4 flex items-center gap-2">
							Loci AI Recommendations
							<div class="w-2 h-2 bg-green-500 rounded-full animate-pulse"></div>
						</h3>
						<div id="llm-streaming-content" class="space-y-4">
							<div class="flex items-center gap-3 text-muted-foreground">
								<div class="w-5 h-5 border-2 border-blue-600 border-t-transparent rounded-full animate-spin"></div>
								<span class="text-sm">AI is analyzing your request and preparing personalized recommendations...</span>
							</div>
						</div>
					</div>
				</div>
			</div>

			<!-- Example Results Placeholder -->
			<div class="grid grid-cols-1 md:grid-cols-2 gap-6" id="results-placeholder">
				<div class="bg-card rounded-xl overflow-hidden shadow-sm border opacity-50">
					<div class="aspect-video bg-gradient-to-r from-gray-200 to-gray-300 dark:from-gray-700 dark:to-gray-600 animate-pulse"></div>
					<div class="p-4">
						<div class="h-4 bg-gray-200 dark:bg-gray-700 rounded animate-pulse mb-2"></div>
						<div class="h-3 bg-gray-200 dark:bg-gray-700 rounded animate-pulse w-3/4 mb-2"></div>
						<div class="h-3 bg-gray-200 dark:bg-gray-700 rounded animate-pulse w-1/2"></div>
					</div>
				</div>
				<div class="bg-card rounded-xl overflow-hidden shadow-sm border opacity-50">
					<div class="aspect-video bg-gradient-to-r from-gray-200 to-gray-300 dark:from-gray-700 dark:to-gray-600 animate-pulse"></div>
					<div class="p-4">
						<div class="h-4 bg-gray-200 dark:bg-gray-700 rounded animate-pulse mb-2"></div>
						<div class="h-3 bg-gray-200 dark:bg-gray-700 rounded animate-pulse w-3/4 mb-2"></div>
						<div class="h-3 bg-gray-200 dark:bg-gray-700 rounded animate-pulse w-1/2"></div>
					</div>
				</div>
			</div>

			<!-- CTA Section -->
			<div class="bg-gradient-to-r from-blue-50 to-purple-50 dark:from-gray-800 dark:to-gray-700 rounded-xl p-6 text-center border">
				<h3 class="text-lg font-semibold text-card-foreground mb-2">Want More Personalized Recommendations?</h3>
				<p class="text-muted-foreground mb-4 text-sm">
					Sign up for free to get AI recommendations tailored to your preferences, save your favorite places, and create custom itineraries.
				</p>
				<div class="flex gap-3 justify-center">
					<a href="/auth/signup" class="bg-blue-600 hover:bg-blue-700 text-white px-6 py-2 rounded-lg text-sm font-medium transition-colors">
						Get Started Free
					</a>
					<a href="/about" class="border hover:bg-accent text-foreground px-6 py-2 rounded-lg text-sm font-medium transition-colors">
						Learn More
					</a>
				</div>
			</div>
		</div>

		<script>
		// Auto-scroll to results
		document.getElementById('search-results').scrollIntoView({ behavior: 'smooth' });
		
		// Simulate streaming content update (in production, this would be real streaming)
		setTimeout(() => {
			const content = document.getElementById('llm-streaming-content');
			if (content) {
				content.innerHTML = ` + "`" + `
					<div class="space-y-4">
						<div class="border-l-4 border-blue-500 pl-4">
							<h4 class="font-medium text-card-foreground mb-2">Top Recommendations</h4>
							<p class="text-sm text-muted-foreground mb-3">Based on your search for "%s", here are some amazing places to discover:</p>
							<ul class="space-y-2 text-sm">
								<li class="flex items-start gap-2">
									<span class="text-blue-500 mt-1">•</span>
									<span class="text-muted-foreground">Hidden local cafés with authentic atmosphere</span>
								</li>
								<li class="flex items-start gap-2">
									<span class="text-blue-500 mt-1">•</span>
									<span class="text-muted-foreground">Cultural landmarks off the beaten path</span>
								</li>
								<li class="flex items-start gap-2">
									<span class="text-blue-500 mt-1">•</span>
									<span class="text-muted-foreground">Local markets and unique shopping experiences</span>
								</li>
							</ul>
						</div>
						<div class="bg-amber-50 dark:bg-amber-900/20 border border-amber-200 dark:border-amber-700 rounded-lg p-4">
							<div class="flex items-start gap-3">
								<svg class="w-5 h-5 text-amber-600 mt-0.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"></path>
								</svg>
								<div>
									<h5 class="font-medium text-amber-800 dark:text-amber-200 text-sm mb-1">Limited Free Results</h5>
									<p class="text-amber-700 dark:text-amber-300 text-sm">
										This is a sample of what our AI can discover. Sign up for free to get detailed recommendations, maps, and personalized itineraries!
									</p>
								</div>
							</div>
						</div>
					</div>
				` + "`" + `;
			}
			
			// Hide placeholder results
			const placeholder = document.getElementById('results-placeholder');
			if (placeholder) {
				placeholder.style.display = 'none';
			}
		}, 3000);
		</script>
	`, query, query)
}
