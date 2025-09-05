package handlers

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/FACorreiaa/go-templui/app/pkg/logger"
)

func TestMain(m *testing.M) {
	// Initialize logger
	err := logger.Init(zapcore.InfoLevel, zap.String("service", "loci-templui-test"))
	if err != nil {
		panic("Failed to initialize logger")
	}
	// Run tests
	exitCode := m.Run()
	// Exit
	os.Exit(exitCode)
}

func TestChatHandlers_SendMessage(t *testing.T) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Setup the router
	r := gin.Default()
	r.Static("/static", "./assets/static")
	r.StaticFile("/sw.js", "./static/sw.js")
	r.Use(func(c *gin.Context) {
		// Mock the user ID in the context
		c.Set("user_id", "test-user-id")
		c.Next()
	})
	chatHandlers := NewChatHandlers()
	r.POST("/chat/message", chatHandlers.SendMessage)

	t.Run("it returns a successful response with a valid message", func(t *testing.T) {
		// Create a new HTTP request
		body := strings.NewReader("message=Hello")
		req, _ := http.NewRequest(http.MethodPost, "/chat/message", body)
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		// Create a response recorder
		w := httptest.NewRecorder()

		// Serve the HTTP request
		r.ServeHTTP(w, req)

		// Assert the status code
		assert.Equal(t, http.StatusOK, w.Code)

		// Assert the response body
		responseBody := w.Body.String()
		assert.Contains(t, responseBody, "Hello", "Response should contain the user message")
		assert.Contains(t, responseBody, "Thanks for your message!", "Response should contain the AI message")
	})

	t.Run("it returns a bad request with an empty message", func(t *testing.T) {
		// Create a new HTTP request
		body := strings.NewReader("message=")
		req, _ := http.NewRequest(http.MethodPost, "/chat/message", body)
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
