package profiles

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/FACorreiaa/go-templui/internal/app/models"
	"github.com/FACorreiaa/go-templui/internal/pkg/logger"
	"github.com/FACorreiaa/go-templui/internal/pkg/middleware"
)

type ProfilesHandler struct {
	profileService Service
}

func NewProfilesHandler(profileService Service) *ProfilesHandler {
	return &ProfilesHandler{
		profileService: profileService,
	}
}

// handleProfileError provides consistent error handling for profile operations
func (h *ProfilesHandler) handleProfileError(c *gin.Context, err error, operation string) {
	logger.Log.Error("Profile operation failed", zap.String("operation", operation), zap.Error(err))

	switch {
	case errors.Is(err, models.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "Profile not found",
			"details": "The requested profile does not exist or you don't have permission to access it",
		})
	case errors.Is(err, models.ErrConflict):
		c.JSON(http.StatusConflict, gin.H{
			"error":   "Profile name conflict",
			"details": "A profile with this name already exists for your account",
		})
	case errors.Is(err, models.ErrBadRequest):
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid profile data",
			"details": err.Error(),
		})
	case errors.Is(err, models.ErrProfileNameEmpty):
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Profile name required",
			"details": "Profile name cannot be empty",
		})
	case errors.Is(err, models.ErrProfileNameTooLong):
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Profile name too long",
			"details": "Profile name cannot exceed 100 characters",
		})
	case errors.Is(err, models.ErrCannotDeleteDefault):
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Cannot delete default profile",
			"details": "You must have at least one profile, and cannot delete your default profile",
		})
	case errors.Is(err, models.ErrForbidden):
		c.JSON(http.StatusForbidden, gin.H{
			"error":   "Access denied",
			"details": "You don't have permission to perform this operation on this profile",
		})
	case errors.Is(err, models.ErrValidation):
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Validation failed",
			"details": err.Error(),
		})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   fmt.Sprintf("Failed to %s profile", operation),
			"details": "An app error occurred. Please try again later.",
		})
	}
}

// GetProfiles godoc
// @Summary Get all user profiles
// @Description Retrieve all preference profiles for the authenticated user
// @Tags profiles
// @Produce json
// @Success 200 {array} models.UserPreferenceProfileResponse
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/profiles [get]
func (h *ProfilesHandler) GetProfiles(c *gin.Context) {
	userIDStr := middleware.GetUserIDFromContext(c)
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		logger.Log.Error("Invalid user ID", zap.String("userID", userIDStr), zap.Error(err))

		// Check if this is an HTMX request (for select dropdown)
		if c.GetHeader("HX-Request") == "true" {
			c.Header("Content-Type", "text/html")
			c.String(http.StatusBadRequest, `<option value="">Error: Invalid user session</option>`)
			return
		}

		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	profiles, err := h.profileService.GetSearchProfiles(c.Request.Context(), userID)
	if err != nil {
		logger.Log.Error("Failed to get profiles", zap.String("userID", userIDStr), zap.Error(err))

		// Check if this is an HTMX request (for select dropdown)
		if c.GetHeader("HX-Request") == "true" {
			c.Header("Content-Type", "text/html")
			c.String(http.StatusInternalServerError, `<option value="">Error loading profiles</option>`)
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve profiles"})
		return
	}

	// Check if this is an HTMX request (for select dropdown)
	if c.GetHeader("HX-Request") == "true" {
		c.Header("Content-Type", "text/html")

		if len(profiles) == 0 {
			c.String(http.StatusOK, `<option value="">No profiles found - create one in Settings</option>`)
			return
		}

		// Build HTML options
		html := `<option value="">Auto-select profile</option>`
		for _, profile := range profiles {
			selected := ""
			if profile.IsDefault {
				selected = ` selected`
			}
			suffix := ""
			if profile.IsDefault {
				suffix = " (Default)"
			}
			html += fmt.Sprintf(`<option value="%s"%s>%s%s</option>`,
				profile.ID, selected, profile.ProfileName, suffix)
		}

		c.String(http.StatusOK, html)
		return
	}

	c.JSON(http.StatusOK, profiles)
}

// GetProfile godoc
// @Summary Get a specific user profile
// @Description Retrieve a specific preference profile by ID
// @Tags profiles
// @Produce json
// @Param id path string true "Profile ID"
// @Success 200 {object} models.UserPreferenceProfileResponse
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/profiles/{id} [get]
func (h *ProfilesHandler) GetProfile(c *gin.Context) {
	userIDStr := middleware.GetUserIDFromContext(c)
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		logger.Log.Error("Invalid user ID", zap.String("userID", userIDStr), zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	profileIDStr := c.Param("id")
	profileID, err := uuid.Parse(profileIDStr)
	if err != nil {
		logger.Log.Error("Invalid profile ID", zap.String("profileID", profileIDStr), zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid profile ID"})
		return
	}

	profile, err := h.profileService.GetSearchProfile(c.Request.Context(), userID, profileID)
	if err != nil {
		logger.Log.Error("Failed to get profile", zap.String("userID", userIDStr), zap.String("profileID", profileIDStr), zap.Error(err))
		if err == models.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Profile not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve profile"})
		return
	}

	c.JSON(http.StatusOK, profile)
}

// CreateProfile godoc
// @Summary Create a new user profile
// @Description Create a new preference profile for the authenticated user
// @Tags profiles
// @Accept json
// @Produce json
// @Param profile body models.CreateUserPreferenceProfileParams true "Profile data"
// @Success 201 {object} models.UserPreferenceProfileResponse
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/profiles [post]
func (h *ProfilesHandler) CreateProfile(c *gin.Context) {
	userIDStr := middleware.GetUserIDFromContext(c)
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		logger.Log.Error("Invalid user ID", zap.String("userID", userIDStr), zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	var params models.CreateUserPreferenceProfileParams
	if err := c.ShouldBindJSON(&params); err != nil {
		logger.Log.Error("Invalid request body", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	profile, err := h.profileService.CreateSearchProfile(c.Request.Context(), userID, params)
	if err != nil {
		h.handleProfileError(c, err, "create")
		return
	}

	c.JSON(http.StatusCreated, profile)
}

// UpdateProfile godoc
// @Summary Update a user profile
// @Description Update a specific preference profile
// @Tags profiles
// @Accept json
// @Produce json
// @Param id path string true "Profile ID"
// @Param profile body models.UpdateSearchProfileParams true "Profile data"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/profiles/{id} [put]
func (h *ProfilesHandler) UpdateProfile(c *gin.Context) {
	userIDStr := middleware.GetUserIDFromContext(c)
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		logger.Log.Error("Invalid user ID", zap.String("userID", userIDStr), zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	profileIDStr := c.Param("id")
	profileID, err := uuid.Parse(profileIDStr)
	if err != nil {
		logger.Log.Error("Invalid profile ID", zap.String("profileID", profileIDStr), zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid profile ID"})
		return
	}

	var params models.UpdateSearchProfileParams
	if err := c.ShouldBindJSON(&params); err != nil {
		logger.Log.Error("Invalid request body", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	if err := h.profileService.UpdateSearchProfile(c.Request.Context(), userID, profileID, params); err != nil {
		h.handleProfileError(c, err, "update")
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Profile updated successfully"})
}

// DeleteProfile godoc
// @Summary Delete a user profile
// @Description Delete a specific preference profile
// @Tags profiles
// @Produce json
// @Param id path string true "Profile ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/profiles/{id} [delete]
func (h *ProfilesHandler) DeleteProfile(c *gin.Context) {
	userIDStr := middleware.GetUserIDFromContext(c)
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		logger.Log.Error("Invalid user ID", zap.String("userID", userIDStr), zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	profileIDStr := c.Param("id")
	profileID, err := uuid.Parse(profileIDStr)
	if err != nil {
		logger.Log.Error("Invalid profile ID", zap.String("profileID", profileIDStr), zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid profile ID"})
		return
	}

	if err := h.profileService.DeleteSearchProfile(c.Request.Context(), userID, profileID); err != nil {
		h.handleProfileError(c, err, "delete")
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Profile deleted successfully"})
}

// SetDefaultProfile godoc
// @Summary Set a profile as default
// @Description Set a specific preference profile as the default for the user
// @Tags profiles
// @Produce json
// @Param id path string true "Profile ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/profiles/{id}/default [put]
func (h *ProfilesHandler) SetDefaultProfile(c *gin.Context) {
	userIDStr := middleware.GetUserIDFromContext(c)
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		logger.Log.Error("Invalid user ID", zap.String("userID", userIDStr), zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	profileIDStr := c.Param("id")
	profileID, err := uuid.Parse(profileIDStr)
	if err != nil {
		logger.Log.Error("Invalid profile ID", zap.String("profileID", profileIDStr), zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid profile ID"})
		return
	}

	if err := h.profileService.SetDefaultSearchProfile(c.Request.Context(), userID, profileID); err != nil {
		logger.Log.Error("Failed to set default profile", zap.String("userID", userIDStr), zap.String("profileID", profileIDStr), zap.Error(err))
		if err == models.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Profile not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to set default profile"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Default profile set successfully"})
}
