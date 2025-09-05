package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/FACorreiaa/go-templui/app/pkg/logger"
	"github.com/FACorreiaa/go-templui/app/pkg/middleware"
)

type ChatHandlers struct{}

func NewChatHandlers() *ChatHandlers {
	return &ChatHandlers{}
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
