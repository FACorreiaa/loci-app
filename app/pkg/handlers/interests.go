package handlers

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/FACorreiaa/go-templui/app/internal/models"
	"github.com/FACorreiaa/go-templui/app/pkg/domain/interests"
	"github.com/FACorreiaa/go-templui/app/pkg/logger"
	"github.com/FACorreiaa/go-templui/app/pkg/middleware"
)

// InterestsServiceInterface defines the methods we need from the interests service
type InterestsServiceInterface interface {
	GetAllInterests(ctx context.Context) ([]*models.Interest, error)
	CreateInterest(ctx context.Context, name string, description *string, isActive bool, userID string) (*models.Interest, error)
	Removeinterests(ctx context.Context, userID uuid.UUID, interestID uuid.UUID) error
}

type InterestsHandler struct {
	interestService interests.Repository
}

func NewInterestsHandler(interestRepo interests.Repository) *InterestsHandler {
	return &InterestsHandler{
		interestService: interestRepo,
	}
}

// GetInterests godoc
// @Summary Get all interests
// @Description Retrieve all available interests
// @Tags interests
// @Produce json
// @Success 200 {array} models.Interest
// @Failure 500 {object} map[string]string
// @Router /api/interests [get]
func (h *InterestsHandler) GetInterests(c *gin.Context) {
	interests, err := h.interestService.GetAllInterests(c.Request.Context())
	if err != nil {
		logger.Log.Error("Failed to get interests", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve interests"})
		return
	}

	c.JSON(http.StatusOK, interests)
}

// CreateInterest godoc
// @Summary Create a new interest
// @Description Create a new interest
// @Tags interests
// @Accept json
// @Produce json
// @Param interest body models.CreateInterestRequest true "Interest data"
// @Success 201 {object} models.Interest
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/interests [post]
func (h *InterestsHandler) CreateInterest(c *gin.Context) {
	userIDStr := middleware.GetUserIDFromContext(c)
	
	var req models.CreateInterestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Log.Error("Invalid request body", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	interest, err := h.interestService.CreateInterest(c.Request.Context(), req.Name, req.Description, req.Active, userIDStr)
	if err != nil {
		logger.Log.Error("Failed to create interest", zap.String("userID", userIDStr), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create interest"})
		return
	}

	c.JSON(http.StatusCreated, interest)
}

// RemoveInterest godoc
// @Summary Remove an interest
// @Description Remove an interest from the system
// @Tags interests
// @Produce json
// @Param id path string true "Interest ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/interests/{id} [delete]
func (h *InterestsHandler) RemoveInterest(c *gin.Context) {
	userIDStr := middleware.GetUserIDFromContext(c)
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		logger.Log.Error("Invalid user ID", zap.String("userID", userIDStr), zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	interestIDStr := c.Param("id")
	interestID, err := uuid.Parse(interestIDStr)
	if err != nil {
		logger.Log.Error("Invalid interest ID", zap.String("interestID", interestIDStr), zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid interest ID"})
		return
	}

	if err := h.interestService.Removeinterests(c.Request.Context(), userID, interestID); err != nil {
		logger.Log.Error("Failed to remove interest", zap.String("userID", userIDStr), zap.String("interestID", interestIDStr), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove interest"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Interest removed successfully"})
}