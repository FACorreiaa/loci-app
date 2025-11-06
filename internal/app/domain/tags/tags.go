package tags

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/FACorreiaa/go-templui/internal/app/models"
	"github.com/FACorreiaa/go-templui/internal/pkg/middleware"
)

type TagsHandler struct {
	tagRepo Repository
	logger  *zap.Logger
}

func NewTagsHandler(tagRepo Repository, logger *zap.Logger) *TagsHandler {
	return &TagsHandler{
		tagRepo: tagRepo,
		logger:  logger,
	}
}

// GetTags godoc
// @Summary Get all tags
// @Description Retrieve all available tags for the authenticated user
// @Tags tags
// @Produce json
// @Success 200 {array} models.Tags
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/tags [get]
func (h *TagsHandler) GetTags(c *gin.Context) {
	userIDStr := middleware.GetUserIDFromContext(c)
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		h.logger.Error("Invalid user ID", zap.String("userID", userIDStr), zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	tags, err := h.tagRepo.GetAll(c.Request.Context(), userID)
	if err != nil {
		h.logger.Error("Failed to get tags", zap.String("userID", userIDStr), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve tags"})
		return
	}

	c.JSON(http.StatusOK, tags)
}

// CreateTag godoc
// @Summary Create a personal tag
// @Description Create a new personal tag for the authenticated user
// @Tags tags
// @Accept json
// @Produce json
// @Param tag body models.CreatePersonalTagParams true "Tag data"
// @Success 201 {object} models.PersonalTag
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/tags [post]
func (h *TagsHandler) CreateTag(c *gin.Context) {
	userIDStr := middleware.GetUserIDFromContext(c)
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		h.logger.Error("Invalid user ID", zap.String("userID", userIDStr), zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	var params models.CreatePersonalTagParams
	if err := c.ShouldBindJSON(&params); err != nil {
		h.logger.Error("Invalid request body", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	tag, err := h.tagRepo.Create(c.Request.Context(), userID, params)
	if err != nil {
		h.logger.Error("Failed to create personal tag", zap.String("userID", userIDStr), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create tag"})
		return
	}

	c.JSON(http.StatusCreated, tag)
}

// UpdateTag godoc
// @Summary Update a personal tag
// @Description Update a personal tag for the authenticated user
// @Tags tags
// @Accept json
// @Produce json
// @Param id path string true "Tag ID"
// @Param tag body models.UpdatePersonalTagParams true "Tag data"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/tags/{id} [put]
func (h *TagsHandler) UpdateTag(c *gin.Context) {
	userIDStr := middleware.GetUserIDFromContext(c)
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		h.logger.Error("Invalid user ID", zap.String("userID", userIDStr), zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	tagIDStr := c.Param("id")
	tagID, err := uuid.Parse(tagIDStr)
	if err != nil {
		h.logger.Error("Invalid tag ID", zap.String("tagID", tagIDStr), zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid tag ID"})
		return
	}

	var params models.UpdatePersonalTagParams
	if err := c.ShouldBindJSON(&params); err != nil {
		h.logger.Error("Invalid request body", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	if err := h.tagRepo.Update(c.Request.Context(), userID, tagID, params); err != nil {
		h.logger.Error("Failed to update personal tag", zap.String("userID", userIDStr), zap.String("tagID", tagIDStr), zap.Error(err))
		if err == models.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Tag not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update tag"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Tag updated successfully"})
}

// DeleteTag godoc
// @Summary Delete a personal tag
// @Description Delete a personal tag for the authenticated user
// @Tags tags
// @Produce json
// @Param id path string true "Tag ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/tags/{id} [delete]
func (h *TagsHandler) DeleteTag(c *gin.Context) {
	userIDStr := middleware.GetUserIDFromContext(c)
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		h.logger.Error("Invalid user ID", zap.String("userID", userIDStr), zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	tagIDStr := c.Param("id")
	tagID, err := uuid.Parse(tagIDStr)
	if err != nil {
		h.logger.Error("Invalid tag ID", zap.String("tagID", tagIDStr), zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid tag ID"})
		return
	}

	if err := h.tagRepo.Delete(c.Request.Context(), userID, tagID); err != nil {
		h.logger.Error("Failed to delete personal tag", zap.String("userID", userIDStr), zap.String("tagID", tagIDStr), zap.Error(err))
		if err == models.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Tag not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete tag"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Tag deleted successfully"})
}
