package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"sync"
	"time"

	"github.com/FACorreiaa/go-templui/app/internal/features/nearby"
	"github.com/FACorreiaa/go-templui/app/internal/models"
	llmchat "github.com/FACorreiaa/go-templui/app/pkg/domain/chat_prompt"
	"github.com/FACorreiaa/go-templui/app/pkg/domain/location"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"google.golang.org/genai"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// In production, you should validate the origin
		return true
	},
}

type NearbyHandler struct {
	logger           *slog.Logger
	chatService      *llmchat.ServiceImpl
	locationRepo     location.Repository
	connections      map[*websocket.Conn]bool
	connectionsMu    sync.RWMutex
	messageLimiter   *MessageRateLimiter
	clientLimiters   map[string]*ClientLimit
	clientLimitersMu sync.RWMutex
}

// ClientLimit tracks requests for a single client (moved from middleware for WebSocket use)
type ClientLimit struct {
	requests  []time.Time
	mu        sync.Mutex
	lastSeen  time.Time
	websocket bool
}

// MessageRateLimiter tracks message rates (embedded from middleware)
type MessageRateLimiter struct {
	maxMessages int
	window      time.Duration
	logger      *slog.Logger
}

func NewNearbyHandler(logger *slog.Logger, chatService *llmchat.ServiceImpl, locationRepo location.Repository) *NearbyHandler {
	return &NearbyHandler{
		logger:       logger,
		chatService:  chatService,
		locationRepo: locationRepo,
		connections:  make(map[*websocket.Conn]bool),
		messageLimiter: &MessageRateLimiter{
			maxMessages: 30,              // 30 messages
			window:      1 * time.Minute, // per minute
			logger:      logger,
		},
		clientLimiters: make(map[string]*ClientLimit),
	}
}

// Page renders the nearby page
func (h *NearbyHandler) Page(c *gin.Context) {
	c.HTML(http.StatusOK, "", nearby.NearbyPage())
}

// LocationUpdate represents a location update from the client
type LocationUpdate struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Radius    float64 `json:"radius"`
}

// POIResponse represents a single POI in the response
type POIResponse struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Category    string  `json:"category"`
	Description string  `json:"description"`
	Emoji       string  `json:"emoji"`
	Rating      float64 `json:"rating"`
	Distance    float64 `json:"distance"`
	Latitude    float64 `json:"latitude"`
	Longitude   float64 `json:"longitude"`
}

// WebSocketMessage represents messages sent to the client
type WebSocketMessage struct {
	Type    string        `json:"type"`
	POIs    []POIResponse `json:"pois,omitempty"`
	Message string        `json:"message,omitempty"`
}

// HandleWebSocket handles WebSocket connections for real-time nearby updates
func (h *NearbyHandler) HandleWebSocket(c *gin.Context) {
	// Get user ID from context (set by middleware)
	userID := "anonymous"
	if id, exists := c.Get("user_id"); exists {
		userID = id.(string)
	}

	ws, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		h.logger.Error("Failed to upgrade to WebSocket", slog.Any("error", err))
		return
	}
	defer ws.Close()

	// Register connection
	h.connectionsMu.Lock()
	h.connections[ws] = true
	h.connectionsMu.Unlock()

	// Unregister connection on exit
	defer func() {
		h.connectionsMu.Lock()
		delete(h.connections, ws)
		h.connectionsMu.Unlock()
	}()

	h.logger.Info("WebSocket connection established", slog.String("user_id", userID))

	// Get or create client limiter
	h.clientLimitersMu.Lock()
	clientLimit, exists := h.clientLimiters[userID]
	if !exists {
		clientLimit = &ClientLimit{
			requests:  make([]time.Time, 0),
			lastSeen:  time.Now(),
			websocket: true,
		}
		h.clientLimiters[userID] = clientLimit
	}
	h.clientLimitersMu.Unlock()

	// Cleanup on disconnect
	defer func() {
		h.clientLimitersMu.Lock()
		if cl, exists := h.clientLimiters[userID]; exists {
			cl.websocket = false
		}
		h.clientLimitersMu.Unlock()
	}()

	// Read messages from client
	for {
		var update LocationUpdate
		err := ws.ReadJSON(&update)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				h.logger.Error("WebSocket error", slog.Any("error", err))
			}
			break
		}

		// Check message rate limit
		if !h.allowMessage(clientLimit, userID) {
			h.logger.Warn("Message rate limit exceeded, sending error",
				slog.String("user_id", userID))
			ws.WriteJSON(WebSocketMessage{
				Type:    "error",
				Message: "Too many requests. Please slow down.",
			})
			continue
		}

		h.logger.Info("Received location update",
			slog.String("user_id", userID),
			slog.Float64("latitude", update.Latitude),
			slog.Float64("longitude", update.Longitude),
			slog.Float64("radius", update.Radius))

		// Save location history asynchronously
		go func() {
			historyCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			history := &models.LocationHistory{
				UserID:    userID,
				Latitude:  update.Latitude,
				Longitude: update.Longitude,
				Radius:    update.Radius,
			}

			if err := h.locationRepo.CreateLocationHistory(historyCtx, history); err != nil {
				h.logger.Error("Failed to save location history",
					slog.String("user_id", userID),
					slog.Any("error", err))
			}
		}()

		// Get POIs for this location
		pois, err := h.getNearbyPOIs(c.Request.Context(), update, userID)
		if err != nil {
			h.logger.Error("Failed to get nearby POIs", slog.Any("error", err))
			ws.WriteJSON(WebSocketMessage{
				Type:    "error",
				Message: "Failed to get nearby places",
			})
			continue
		}

		// Send POIs to client
		err = ws.WriteJSON(WebSocketMessage{
			Type: "pois",
			POIs: pois,
		})
		if err != nil {
			h.logger.Error("Failed to send POIs", slog.Any("error", err))
			break
		}

		// Track POI views asynchronously
		go func(userID string, pois []POIResponse, userLat, userLon float64) {
			interactionCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			for _, poi := range pois {
				interaction := &models.POIInteraction{
					UserID:         userID,
					POIID:          poi.ID,
					POIName:        poi.Name,
					POICategory:    poi.Category,
					InteractionType: "view",
					UserLatitude:   userLat,
					UserLongitude:  userLon,
					POILatitude:    poi.Latitude,
					POILongitude:   poi.Longitude,
					Distance:       poi.Distance,
				}

				if err := h.locationRepo.CreatePOIInteraction(interactionCtx, interaction); err != nil {
					h.logger.Error("Failed to save POI interaction",
						slog.String("user_id", userID),
						slog.String("poi_id", poi.ID),
						slog.Any("error", err))
				}
			}
		}(userID, pois, update.Latitude, update.Longitude)
	}
}

// getNearbyPOIs fetches POIs from the AI service
func (h *NearbyHandler) getNearbyPOIs(ctx context.Context, update LocationUpdate, userID string) ([]POIResponse, error) {
	// Create a prompt for the AI
	prompt := fmt.Sprintf(`Find interesting places near coordinates %.6f, %.6f within %.1f km radius.

Return a JSON array of 5-10 diverse places including restaurants, cafes, attractions, parks, museums, etc.
Each place should have:
- id: unique identifier
- name: place name
- category: type of place (restaurant, cafe, museum, park, etc.)
- description: brief description (max 100 chars)
- emoji: relevant emoji for the category
- rating: rating from 1.0 to 5.0
- latitude: approximate latitude
- longitude: approximate longitude

Focus on real, notable places in that area. Return ONLY valid JSON array, no additional text.`,
		update.Latitude, update.Longitude, update.Radius)

	// Call AI client directly to generate nearby POIs
	config := &genai.GenerateContentConfig{
		Temperature: genai.Ptr[float32](0.5),
	}

	// Access the internal aiClient field (we need to make it public or add a method)
	// For now, let's create a simple wrapper method in the service
	response, err := h.chatService.GenerateNearbyPOIs(ctx, prompt, config)
	if err != nil {
		return nil, fmt.Errorf("failed to generate nearby POIs: %w", err)
	}

	// Parse the AI response
	var pois []POIResponse
	if err := json.Unmarshal([]byte(response), &pois); err != nil {
		// If parsing fails, try to extract JSON from response
		h.logger.Warn("Failed to parse AI response, trying to extract JSON", slog.Any("error", err))

		// Try to find JSON array in response
		start := -1
		end := -1
		for i, char := range response {
			if char == '[' && start == -1 {
				start = i
			}
			if char == ']' {
				end = i + 1
			}
		}

		if start >= 0 && end > start {
			jsonStr := response[start:end]
			if err := json.Unmarshal([]byte(jsonStr), &pois); err != nil {
				return nil, fmt.Errorf("failed to parse extracted JSON: %w", err)
			}
		} else {
			return nil, fmt.Errorf("no valid JSON found in response")
		}
	}

	// Calculate distances and ensure all fields are populated
	for i := range pois {
		if pois[i].Latitude != 0 && pois[i].Longitude != 0 {
			pois[i].Distance = calculateDistance(
				update.Latitude, update.Longitude,
				pois[i].Latitude, pois[i].Longitude,
			)
		}

		// Ensure emoji is set
		if pois[i].Emoji == "" {
			pois[i].Emoji = getCategoryEmoji(pois[i].Category)
		}

		// Ensure ID is set
		if pois[i].ID == "" {
			pois[i].ID = fmt.Sprintf("poi_%d_%d", time.Now().Unix(), i)
		}
	}

	return pois, nil
}

// calculateDistance calculates the distance between two coordinates using the Haversine formula
func calculateDistance(lat1, lon1, lat2, lon2 float64) float64 {
	const R = 6371 // Earth's radius in kilometers

	dLat := (lat2 - lat1) * (math.Pi / 180)
	dLon := (lon2 - lon1) * (math.Pi / 180)

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*(math.Pi/180))*math.Cos(lat2*(math.Pi/180))*
			math.Sin(dLon/2)*math.Sin(dLon/2)

	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return R * c
}

// allowMessage checks if a WebSocket message should be allowed based on rate limits
func (h *NearbyHandler) allowMessage(client *ClientLimit, clientID string) bool {
	client.mu.Lock()
	defer client.mu.Unlock()

	now := time.Now()
	client.lastSeen = now

	// Remove old messages outside the time window
	validRequests := make([]time.Time, 0, len(client.requests))
	cutoff := now.Add(-h.messageLimiter.window)
	for _, reqTime := range client.requests {
		if reqTime.After(cutoff) {
			validRequests = append(validRequests, reqTime)
		}
	}
	client.requests = validRequests

	// Check if limit is exceeded
	if len(client.requests) >= h.messageLimiter.maxMessages {
		return false
	}

	// Add current message
	client.requests = append(client.requests, now)
	return true
}

// getCategoryEmoji returns an emoji for a given category
func getCategoryEmoji(category string) string {
	emojiMap := map[string]string{
		"restaurant": "🍽️",
		"cafe":       "☕",
		"bar":        "🍺",
		"museum":     "🎨",
		"park":       "🌳",
		"attraction": "🏛️",
		"hotel":      "🏨",
		"shopping":   "🛍️",
		"nightlife":  "🌃",
		"beach":      "🏖️",
		"monument":   "🗿",
		"gallery":    "🖼️",
		"theater":    "🎭",
		"cinema":     "🎬",
		"library":    "📚",
		"church":     "⛪",
		"mosque":     "🕌",
		"temple":     "🛕",
		"market":     "🏪",
		"gym":        "💪",
		"hospital":   "🏥",
		"pharmacy":   "💊",
	}

	if emoji, ok := emojiMap[category]; ok {
		return emoji
	}
	return "📍"
}
