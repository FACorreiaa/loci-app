package nearby

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"github.com/FACorreiaa/go-templui/internal/app/models"
)

// MockLocationRepository implements location.Repository for testing
type MockLocationRepository struct {
	locationHistory []models.LocationHistory
	poiInteractions []models.POIInteraction
}

func NewMockLocationRepository() *MockLocationRepository {
	return &MockLocationRepository{
		locationHistory: make([]models.LocationHistory, 0),
		poiInteractions: make([]models.POIInteraction, 0),
	}
}

func (m *MockLocationRepository) CreateLocationHistory(ctx context.Context, history *models.LocationHistory) error {
	history.ID = "test-id"
	history.CreatedAt = time.Now()
	history.Timestamp = time.Now()
	m.locationHistory = append(m.locationHistory, *history)
	return nil
}

func (m *MockLocationRepository) GetLocationHistory(ctx context.Context, userID string, limit, offset int) ([]models.LocationHistory, error) {
	return m.locationHistory, nil
}

func (m *MockLocationRepository) GetLocationHistoryByTimeRange(ctx context.Context, userID string, start, end time.Time) ([]models.LocationHistory, error) {
	return m.locationHistory, nil
}

func (m *MockLocationRepository) CreatePOIInteraction(ctx context.Context, interaction *models.POIInteraction) error {
	interaction.ID = "test-interaction-id"
	interaction.CreatedAt = time.Now()
	interaction.Timestamp = time.Now()
	m.poiInteractions = append(m.poiInteractions, *interaction)
	return nil
}

func (m *MockLocationRepository) GetPOIInteractions(ctx context.Context, userID string, limit, offset int) ([]models.POIInteraction, error) {
	return m.poiInteractions, nil
}

func (m *MockLocationRepository) GetPOIInteractionsByType(ctx context.Context, userID, interactionType string, limit, offset int) ([]models.POIInteraction, error) {
	var filtered []models.POIInteraction
	for _, interaction := range m.poiInteractions {
		if interaction.InteractionType == interactionType {
			filtered = append(filtered, interaction)
		}
	}
	return filtered, nil
}

func (m *MockLocationRepository) GetPOIInteractionStats(ctx context.Context, userID string) (map[string]int, error) {
	stats := make(map[string]int)
	for _, interaction := range m.poiInteractions {
		stats[interaction.POICategory]++
	}
	return stats, nil
}

// setupTestServer creates a test server with WebSocket handler
func setupTestServer(t *testing.T) (*httptest.Server, *MockLocationRepository) {
	gin.SetMode(gin.TestMode)

	mockRepo := NewMockLocationRepository()
	logger := zap.Default()

	// Create a minimal chat service (we won't actually use AI in tests)
	chatService := &llmchat.ServiceImpl{}

	handler := NewNearbyHandler(logger, chatService, mockRepo)

	router := gin.New()
	router.GET("/ws/nearby", handler.HandleWebSocket)

	server := httptest.NewServer(router)
	return server, mockRepo
}

func TestWebSocketConnection(t *testing.T) {
	server, _ := setupTestServer(t)
	defer server.Close()

	// Convert http://... to ws://...
	wsURL := "ws" + server.URL[4:] + "/ws/nearby"

	// Connect to WebSocket
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect to WebSocket: %v", err)
	}
	defer ws.Close()

	// Connection successful
	t.Log("WebSocket connection established successfully")
}

func TestWebSocketLocationUpdate(t *testing.T) {
	server, mockRepo := setupTestServer(t)
	defer server.Close()

	wsURL := "ws" + server.URL[4:] + "/ws/nearby"

	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect to WebSocket: %v", err)
	}
	defer ws.Close()

	// Send location update
	update := LocationUpdate{
		Latitude:  38.7223,
		Longitude: -9.1393,
		Radius:    5.0,
	}

	err = ws.WriteJSON(update)
	if err != nil {
		t.Fatalf("Failed to send location update: %v", err)
	}

	// Set read deadline to avoid hanging
	ws.SetReadDeadline(time.Now().Add(10 * time.Second))

	// Read response
	var response WebSocketMessage
	err = ws.ReadJSON(&response)
	if err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	// Verify response type
	if response.Type != "pois" && response.Type != "error" {
		t.Errorf("Expected response type 'pois' or 'error', got: %s", response.Type)
	}

	// Give some time for async operations
	time.Sleep(100 * time.Millisecond)

	// Verify location history was saved
	if len(mockRepo.locationHistory) == 0 {
		t.Error("Expected location history to be saved, but got 0 entries")
	}

	// Verify location data
	if len(mockRepo.locationHistory) > 0 {
		savedLocation := mockRepo.locationHistory[0]
		if savedLocation.Latitude != update.Latitude {
			t.Errorf("Expected latitude %f, got %f", update.Latitude, savedLocation.Latitude)
		}
		if savedLocation.Longitude != update.Longitude {
			t.Errorf("Expected longitude %f, got %f", update.Longitude, savedLocation.Longitude)
		}
		if savedLocation.Radius != update.Radius {
			t.Errorf("Expected radius %f, got %f", update.Radius, savedLocation.Radius)
		}
	}
}

func TestWebSocketMultipleUpdates(t *testing.T) {
	server, mockRepo := setupTestServer(t)
	defer server.Close()

	wsURL := "ws" + server.URL[4:] + "/ws/nearby"

	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect to WebSocket: %v", err)
	}
	defer ws.Close()

	// Send multiple location updates
	updates := []LocationUpdate{
		{Latitude: 38.7223, Longitude: -9.1393, Radius: 5.0},
		{Latitude: 38.7224, Longitude: -9.1394, Radius: 5.0},
		{Latitude: 38.7225, Longitude: -9.1395, Radius: 5.0},
	}

	for _, update := range updates {
		err = ws.WriteJSON(update)
		if err != nil {
			t.Fatalf("Failed to send location update: %v", err)
		}

		// Read response
		ws.SetReadDeadline(time.Now().Add(5 * time.Second))
		var response WebSocketMessage
		err = ws.ReadJSON(&response)
		if err != nil {
			t.Fatalf("Failed to read response: %v", err)
		}
	}

	// Give time for async operations
	time.Sleep(200 * time.Millisecond)

	// Verify all updates were saved
	if len(mockRepo.locationHistory) != len(updates) {
		t.Errorf("Expected %d location history entries, got %d", len(updates), len(mockRepo.locationHistory))
	}
}

func TestWebSocketInvalidJSON(t *testing.T) {
	server, _ := setupTestServer(t)
	defer server.Close()

	wsURL := "ws" + server.URL[4:] + "/ws/nearby"

	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect to WebSocket: %v", err)
	}
	defer ws.Close()

	// Send invalid JSON
	err = ws.WriteMessage(websocket.TextMessage, []byte("{invalid json}"))
	if err != nil {
		t.Fatalf("Failed to send invalid JSON: %v", err)
	}

	// Connection should close due to invalid JSON
	ws.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, _, err = ws.ReadMessage()

	// We expect an error here (connection closed or EOF)
	if err == nil {
		t.Error("Expected connection to close after invalid JSON, but it remained open")
	}
}

func TestWebSocketConcurrentConnections(t *testing.T) {
	server, mockRepo := setupTestServer(t)
	defer server.Close()

	wsURL := "ws" + server.URL[4:] + "/ws/nearby"

	numConnections := 5
	done := make(chan bool, numConnections)

	// Create multiple concurrent connections
	for i := 0; i < numConnections; i++ {
		go func(id int) {
			ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
			if err != nil {
				t.Errorf("Connection %d failed: %v", id, err)
				done <- false
				return
			}
			defer ws.Close()

			// Send location update
			update := LocationUpdate{
				Latitude:  38.7223 + float64(id)*0.001,
				Longitude: -9.1393 + float64(id)*0.001,
				Radius:    5.0,
			}

			err = ws.WriteJSON(update)
			if err != nil {
				t.Errorf("Connection %d failed to send update: %v", id, err)
				done <- false
				return
			}

			// Read response
			ws.SetReadDeadline(time.Now().Add(5 * time.Second))
			var response WebSocketMessage
			err = ws.ReadJSON(&response)
			if err != nil {
				t.Errorf("Connection %d failed to read response: %v", id, err)
				done <- false
				return
			}

			done <- true
		}(i)
	}

	// Wait for all connections
	successCount := 0
	for i := 0; i < numConnections; i++ {
		if <-done {
			successCount++
		}
	}

	if successCount != numConnections {
		t.Errorf("Expected %d successful connections, got %d", numConnections, successCount)
	}

	// Give time for async operations
	time.Sleep(300 * time.Millisecond)

	// Verify all location updates were saved
	if len(mockRepo.locationHistory) != numConnections {
		t.Errorf("Expected %d location history entries, got %d", numConnections, len(mockRepo.locationHistory))
	}
}

func TestLocationHistoryPersistence(t *testing.T) {
	mockRepo := NewMockLocationRepository()

	// Create test location history
	history := &models.LocationHistory{
		UserID:    "test-user",
		Latitude:  38.7223,
		Longitude: -9.1393,
		Radius:    5.0,
	}

	err := mockRepo.CreateLocationHistory(context.Background(), history)
	if err != nil {
		t.Fatalf("Failed to create location history: %v", err)
	}

	// Retrieve location history
	results, err := mockRepo.GetLocationHistory(context.Background(), "test-user", 10, 0)
	if err != nil {
		t.Fatalf("Failed to get location history: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("Expected 1 location history entry, got %d", len(results))
	}

	if results[0].UserID != "test-user" {
		t.Errorf("Expected user ID 'test-user', got '%s'", results[0].UserID)
	}
}

func TestPOIInteractionTracking(t *testing.T) {
	mockRepo := NewMockLocationRepository()

	// Create test POI interaction
	interaction := &models.POIInteraction{
		UserID:          "test-user",
		POIID:           "poi-123",
		POIName:         "Test Restaurant",
		POICategory:     "restaurant",
		InteractionType: "view",
		UserLatitude:    38.7223,
		UserLongitude:   -9.1393,
		POILatitude:     38.7225,
		POILongitude:    -9.1395,
		Distance:        0.25,
	}

	err := mockRepo.CreatePOIInteraction(context.Background(), interaction)
	if err != nil {
		t.Fatalf("Failed to create POI interaction: %v", err)
	}

	// Retrieve POI interactions
	results, err := mockRepo.GetPOIInteractions(context.Background(), "test-user", 10, 0)
	if err != nil {
		t.Fatalf("Failed to get POI interactions: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("Expected 1 POI interaction, got %d", len(results))
	}

	if results[0].POICategory != "restaurant" {
		t.Errorf("Expected category 'restaurant', got '%s'", results[0].POICategory)
	}
}

func TestPOIInteractionStats(t *testing.T) {
	mockRepo := NewMockLocationRepository()

	// Create multiple interactions with different categories
	categories := []string{"restaurant", "cafe", "restaurant", "museum", "cafe", "cafe"}

	for i, category := range categories {
		interaction := &models.POIInteraction{
			UserID:          "test-user",
			POIID:           string(rune('a' + i)),
			POIName:         "Test POI",
			POICategory:     category,
			InteractionType: "view",
			UserLatitude:    38.7223,
			UserLongitude:   -9.1393,
			POILatitude:     38.7225,
			POILongitude:    -9.1395,
			Distance:        0.25,
		}
		mockRepo.CreatePOIInteraction(context.Background(), interaction)
	}

	// Get stats
	stats, err := mockRepo.GetPOIInteractionStats(context.Background(), "test-user")
	if err != nil {
		t.Fatalf("Failed to get POI interaction stats: %v", err)
	}

	// Verify stats
	expectedStats := map[string]int{
		"restaurant": 2,
		"cafe":       3,
		"museum":     1,
	}

	for category, expectedCount := range expectedStats {
		if stats[category] != expectedCount {
			t.Errorf("Expected %d interactions for category '%s', got %d", expectedCount, category, stats[category])
		}
	}
}

func TestResponseMessageFormat(t *testing.T) {
	// Test that WebSocketMessage can be properly marshaled and unmarshaled
	pois := []POIResponse{
		{
			ID:          "poi-1",
			Name:        "Test Restaurant",
			Category:    "restaurant",
			Description: "A great place to eat",
			Emoji:       "🍽️",
			Rating:      4.5,
			Distance:    0.5,
			Latitude:    38.7225,
			Longitude:   -9.1395,
		},
	}

	msg := WebSocketMessage{
		Type: "pois",
		POIs: pois,
	}

	// Marshal to JSON
	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Failed to marshal WebSocketMessage: %v", err)
	}

	// Unmarshal from JSON
	var decoded WebSocketMessage
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Failed to unmarshal WebSocketMessage: %v", err)
	}

	// Verify
	if decoded.Type != "pois" {
		t.Errorf("Expected type 'pois', got '%s'", decoded.Type)
	}

	if len(decoded.POIs) != 1 {
		t.Errorf("Expected 1 POI, got %d", len(decoded.POIs))
	}

	if decoded.POIs[0].Name != "Test Restaurant" {
		t.Errorf("Expected name 'Test Restaurant', got '%s'", decoded.POIs[0].Name)
	}
}
