package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)


func TestChatHandlers_SendMessage(t *testing.T) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Create mock dependencies
	mockLlmService := &MockLlmService{}
	mockProfileService := &MockProfileService{}
	mockChatRepo := &MockRepository{}

	// Setup the router
	r := gin.Default()
	r.Static("/static", "./assets/static")
	r.StaticFile("/sw.js", "./static/sw.js")
	r.Use(func(c *gin.Context) {
		// Mock the user ID in the context with a valid UUID
		c.Set("user_id", "550e8400-e29b-41d4-a716-446655440000")
		c.Next()
	})
	chatHandlers := NewChatHandlers(mockLlmService, mockProfileService, mockChatRepo)
	r.POST("/chat/message", chatHandlers.SendMessage)

	t.Run("it returns a successful response with a valid message", func(t *testing.T) {
		// Create a new HTTP request
		body := strings.NewReader("message=Hello")
		req, err := http.NewRequest(http.MethodPost, "/chat/message", body)
		if err != nil {
			t.Fatalf("Failed to create HTTP request: %v", err)
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		// Create a response recorder
		w := httptest.NewRecorder()

		// Serve the HTTP request
		r.ServeHTTP(w, req)

		// Assert the status code
		assert.Equal(t, http.StatusOK, w.Code)

		// Assert the response body contains SSE components and processing message
		responseBody := w.Body.String()
		assert.Contains(t, responseBody, "I'm analyzing your request and updating your itinerary", "Response should contain the AI processing message")
		assert.Contains(t, responseBody, "sse-connect=\"/chat/stream?message=Hello", "Response should contain SSE connection for the message")
	})

	t.Run("it returns a bad request with an empty message", func(t *testing.T) {
		// Create a new HTTP request
		body := strings.NewReader("message=")
		req, err := http.NewRequest(http.MethodPost, "/chat/message", body)
		if err != nil {
			t.Fatalf("Failed to create HTTP request: %v", err)
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		// Create a response recorder
		w := httptest.NewRecorder()

		// Serve the HTTP request
		r.ServeHTTP(w, req)

		// Assert the status code
		assert.Equal(t, http.StatusBadRequest, w.Code)

		// Assert the response body
		responseBody := w.Body.String()
		assert.Contains(t, responseBody, "Message cannot be empty", "Response should contain the error message")
	})
}
