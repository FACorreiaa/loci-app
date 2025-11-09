package user

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/FACorreiaa/go-templui/internal/app/domain/profiles"
	"github.com/FACorreiaa/go-templui/internal/app/handlers"
	"github.com/FACorreiaa/go-templui/internal/app/middleware"
)

type Handler struct {
	*handlers.BaseHandler
	Service UserService
}

func NewHandler(base *handlers.BaseHandler, service UserService) *Handler {
	return &Handler{
		BaseHandler: base,
		Service:     service,
	}
}

func (h *Handler) ShowProfilePage(c *gin.Context) {
	userID := middleware.GetUserIDFromContext(c)
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		h.Logger.Error("Invalid user ID", zap.String("userID", userID), zap.Error(err))
		// Render an error page
		return
	}

	userProfile, err := h.Service.GetUserProfile(c.Request.Context(), userUUID)
	if err != nil {
		h.Logger.Error("Failed to fetch user profile", zap.String("userID", userID), zap.Error(err))
		// Still show the page but with nil profile to handle gracefully
		userProfile = nil
	}

	h.RenderPage(c, "Profile - Loci", "Profile", profiles.ProfilePage(userProfile))
}
