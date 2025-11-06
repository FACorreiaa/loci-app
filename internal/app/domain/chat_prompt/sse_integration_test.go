//go:build integration

package llmchat

import (
	"bufio"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/FACorreiaa/go-templui/internal/app/domain/activities"
	"github.com/FACorreiaa/go-templui/internal/app/domain/hotels"
	profiles2 "github.com/FACorreiaa/go-templui/internal/app/domain/profiles"
	"github.com/FACorreiaa/go-templui/internal/app/domain/restaurants"
	"github.com/FACorreiaa/go-templui/internal/pkg/middleware"
)

var testDB *pgxpool.Pool
var testChatService LlmInteractiontService
var testChatRepo Repository
var testProfileService profiles2.Service

// Integration test setup - using the shared TestMain from handlers_basic_test.go
func init() {
	// Initialize integration test environment if TEST_DATABASE_URL is available
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		return // Skip integration test setup if no database
	}

	config, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		panic("Unable to parse TEST_DATABASE_URL: " + err.Error())
	}
	config.MaxConns = 5

	testDB, err = pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		panic("Unable to create connection pool: " + err.Error())
	}

	if err := testDB.Ping(context.Background()); err != nil {
		panic("Unable to ping test database: " + err.Error())
	}

	// Initialize services
	testChatRepo = llmchat.NewRepository(testDB)
	testChatService = llmchat.NewService(testChatRepo)
	testProfileService = profiles.NewService(profiles.NewRepository(testDB))
}

func TestSSE_EndToEndDataFlow(t *testing.T) {
	if testDB == nil {
		t.Skip("Integration test skipped: no test database available")
	}

	gin.SetMode(gin.TestMode)

	// Setup common
	chatHandlers := NewChatHandlers(testChatService, testProfileService, testChatRepo)
	activitiesHandlers := activities.NewActivitiesHandlers(testChatRepo)
	hotelsHandlers := hotels.NewHotelsHandlers(testChatRepo)
	restaurantsHandlers := restaurants.NewRestaurantsHandlers(testChatRepo)

	t.Run("should stream itinerary data and filter for domain-specific common", func(t *testing.T) {
		// Setup router
		r := gin.Default()
		r.Use(func(c *gin.Context) {
			c.Set("user_id", "test-user-id-sse")
			c.Next()
		})

		// Setup routes
		r.POST("/chat/stream/connect", chatHandlers.HandleChatStreamConnect)
		r.GET("/chat/stream/:sessionId", chatHandlers.HandleChatStream)
		r.GET("/activities", func(c *gin.Context) {
			component := activitiesHandlers.HandleActivitiesPage(c)
			c.String(http.StatusOK, "activities component rendered: %v", component != nil)
		})
		r.GET("/hotels", func(c *gin.Context) {
			component := hotelsHandlers.HandleHotelsPage(c)
			c.String(http.StatusOK, "hotels component rendered: %v", component != nil)
		})
		r.GET("/restaurants", func(c *gin.Context) {
			component := restaurantsHandlers.HandleRestaurantsPage(c)
			c.String(http.StatusOK, "restaurants component rendered: %v", component != nil)
		})

		// Test 1: Initiate SSE stream
		form := url.Values{}
		form.Add("dashboard-search", "Paris travel itinerary")
		form.Add("profile-id", uuid.New().String())

		req := httptest.NewRequest("POST", "/chat/stream/connect", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Header().Get("Content-Type"), "text/event-stream")

		// Parse the SSE response to extract session ID
		sessionID := extractSessionIDFromSSEResponse(t, w.Body.String())
		require.NotEmpty(t, sessionID, "Session ID should be extracted from SSE response")

		// Wait a bit for data to be processed and cached
		time.Sleep(2 * time.Second)

		// Test 2: Verify activities filtering works with the session data
		req = httptest.NewRequest("GET", "/activities?sessionId="+sessionID, nil)
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "activities component rendered: true")

		// Test 3: Verify hotels filtering works with the session data
		req = httptest.NewRequest("GET", "/hotels?sessionId="+sessionID, nil)
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "hotels component rendered: true")

		// Test 4: Verify restaurants filtering works with the session data
		req = httptest.NewRequest("GET", "/restaurants?sessionId="+sessionID, nil)
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "restaurants component rendered: true")
	})
}

func TestSSE_StreamingEvents(t *testing.T) {
	if testDB == nil {
		t.Skip("Integration test skipped: no test database available")
	}

	gin.SetMode(gin.TestMode)

	chatHandlers := NewChatHandlers(testChatService, testProfileService, testChatRepo)

	t.Run("should handle SSE stream events correctly", func(t *testing.T) {
		// Setup router
		r := gin.Default()
		r.Use(func(c *gin.Context) {
			c.Set("user_id", "test-user-sse-events")
			c.Next()
		})
		r.POST("/chat/stream/connect", chatHandlers.HandleChatStreamConnect)
		r.GET("/chat/stream/:sessionId", chatHandlers.HandleChatStream)

		// Initiate stream
		form := url.Values{}
		form.Add("dashboard-search", "Tokyo travel guide")
		form.Add("profile-id", uuid.New().String())

		req := httptest.NewRequest("POST", "/chat/stream/connect", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		// Verify SSE headers
		assert.Equal(t, "text/event-stream", w.Header().Get("Content-Type"))
		assert.Equal(t, "no-cache", w.Header().Get("Cache-Control"))
		assert.Equal(t, "keep-alive", w.Header().Get("Connection"))
		assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))

		// Parse SSE events
		sseEvents := parseSSEEvents(t, w.Body.String())
		assert.Greater(t, len(sseEvents), 0, "Should receive at least one SSE event")

		// Verify event structure
		for _, event := range sseEvents {
			assert.Contains(t, []string{"progress", "data", "complete", "error"}, event.Type, "Event type should be valid")
			if event.Type == "data" || event.Type == "complete" {
				assert.NotEmpty(t, event.Data, "Data events should contain content")
			}
		}
	})
}

func TestSSE_CacheIntegration(t *testing.T) {
	if testDB == nil {
		t.Skip("Integration test skipped: no test database available")
	}

	gin.SetMode(gin.TestMode)

	chatHandlers := NewChatHandlers(testChatService, testProfileService, testChatRepo)
	activitiesHandlers := activities.NewActivitiesHandlers(testChatRepo)

	t.Run("should populate cache during SSE stream and be accessible by domain common", func(t *testing.T) {
		// Setup
		r := gin.Default()
		r.Use(func(c *gin.Context) {
			c.Set("user_id", "test-user-cache-integration")
			c.Next()
		})
		r.POST("/chat/stream/connect", chatHandlers.HandleChatStreamConnect)

		// Initiate stream
		form := url.Values{}
		form.Add("dashboard-search", "Barcelona attractions and restaurants")

		req := httptest.NewRequest("POST", "/chat/stream/connect", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		sessionID := extractSessionIDFromSSEResponse(t, w.Body.String())
		require.NotEmpty(t, sessionID)

		// Wait for cache population
		time.Sleep(3 * time.Second)

		// Verify cache contains data
		cachedData, found := middleware.CompleteItineraryCache.Get(sessionID)
		if found {
			assert.NotEmpty(t, cachedData.GeneralCityData.City)
			assert.Greater(t, len(cachedData.PointsOfInterest), 0)

			// Test that activities handler can use this cached data
			w2 := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w2)
			c.Request, _ = http.NewRequest("GET", "/activities?sessionId="+sessionID, nil)

			component := activitiesHandlers.HandleActivitiesPage(c)
			assert.NotNil(t, component)
		}
	})
}

func TestSSE_ErrorHandling(t *testing.T) {
	if testDB == nil {
		t.Skip("Integration test skipped: no test database available")
	}

	gin.SetMode(gin.TestMode)

	chatHandlers := NewChatHandlers(testChatService, testProfileService, testChatRepo)

	t.Run("should handle empty query gracefully", func(t *testing.T) {
		r := gin.Default()
		r.POST("/chat/stream/connect", chatHandlers.HandleChatStreamConnect)

		// Empty query
		form := url.Values{}
		form.Add("dashboard-search", "")

		req := httptest.NewRequest("POST", "/chat/stream/connect", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "Please provide a search query")
	})

	t.Run("should handle invalid session ID in stream endpoint", func(t *testing.T) {
		r := gin.Default()
		r.GET("/chat/stream/:sessionId", chatHandlers.HandleChatStream)

		req := httptest.NewRequest("GET", "/chat/stream/invalid-session-id", nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		// Should handle gracefully (specific behavior depends on implementation)
		assert.NotEqual(t, http.StatusInternalServerError, w.Code)
	})
}

func TestSSE_ConcurrentAccess(t *testing.T) {
	if testDB == nil {
		t.Skip("Integration test skipped: no test database available")
	}

	gin.SetMode(gin.TestMode)

	chatHandlers := NewChatHandlers(testChatService, testProfileService, testChatRepo)
	activitiesHandlers := activities.NewActivitiesHandlers(testChatRepo)

	t.Run("should handle concurrent SSE streams and domain handler access", func(t *testing.T) {
		r := gin.Default()
		r.Use(func(c *gin.Context) {
			c.Set("user_id", "test-user-concurrent")
			c.Next()
		})
		r.POST("/chat/stream/connect", chatHandlers.HandleChatStreamConnect)

		// Create multiple concurrent streams
		numStreams := 3
		sessionIDs := make(chan string, numStreams)

		for i := 0; i < numStreams; i++ {
			go func(index int) {
				form := url.Values{}
				form.Add("dashboard-search", "London travel guide "+string(rune(65+index))) // A, B, C

				req := httptest.NewRequest("POST", "/chat/stream/connect", strings.NewReader(form.Encode()))
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
				w := httptest.NewRecorder()

				r.ServeHTTP(w, req)

				if w.Code == http.StatusOK {
					sessionID := extractSessionIDFromSSEResponse(t, w.Body.String())
					if sessionID != "" {
						sessionIDs <- sessionID
					}
				}
			}(i)
		}

		// Wait for all streams to complete
		receivedSessions := 0
		timeout := time.After(10 * time.Second)

		for receivedSessions < numStreams {
			select {
			case sessionID := <-sessionIDs:
				assert.NotEmpty(t, sessionID)
				receivedSessions++

				// Test concurrent access to domain common
				go func(sid string) {
					time.Sleep(2 * time.Second) // Wait for cache population
					w := httptest.NewRecorder()
					c, _ := gin.CreateTestContext(w)
					c.Request, _ = http.NewRequest("GET", "/activities?sessionId="+sid, nil)

					component := activitiesHandlers.HandleActivitiesPage(c)
					assert.NotNil(t, component)
				}(sessionID)

			case <-timeout:
				t.Fatalf("Timeout waiting for concurrent streams to complete. Received %d/%d", receivedSessions, numStreams)
			}
		}
	})
}

// Helper functions

func extractSessionIDFromSSEResponse(t *testing.T, sseResponse string) string {
	// Parse SSE response to find session ID
	lines := strings.Split(sseResponse, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")

			// Try to extract session ID from HTML content
			if strings.Contains(data, "sessionId=") {
				parts := strings.Split(data, "sessionId=")
				if len(parts) > 1 {
					sessionPart := strings.Split(parts[1], "&")[0]
					sessionPart = strings.Split(sessionPart, "\"")[0]
					if isValidUUID(sessionPart) {
						return sessionPart
					}
				}
			}

			// Try to parse as JSON for session ID
			if strings.HasPrefix(data, "{") {
				var jsonData map[string]interface{}
				if err := json.Unmarshal([]byte(data), &jsonData); err == nil {
					if sessionID, ok := jsonData["sessionId"].(string); ok {
						return sessionID
					}
				}
			}
		}
	}
	return ""
}

func parseSSEEvents(t *testing.T, sseResponse string) []SSEEvent {
	var events []SSEEvent
	scanner := bufio.NewScanner(strings.NewReader(sseResponse))

	var currentEvent SSEEvent

	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "event: ") {
			currentEvent.Type = strings.TrimPrefix(line, "event: ")
		} else if strings.HasPrefix(line, "data: ") {
			currentEvent.Data = strings.TrimPrefix(line, "data: ")
		} else if line == "" && currentEvent.Type != "" {
			events = append(events, currentEvent)
			currentEvent = SSEEvent{}
		}
	}

	// Don't forget the last event if file doesn't end with empty line
	if currentEvent.Type != "" {
		events = append(events, currentEvent)
	}

	return events
}

func isValidUUID(str string) bool {
	_, err := uuid.Parse(str)
	return err == nil
}

type SSEEvent struct {
	Type string
	Data string
}
