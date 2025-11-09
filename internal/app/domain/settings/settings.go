package settings

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/FACorreiaa/go-templui/internal/app/domain"
	"github.com/FACorreiaa/go-templui/internal/app/middleware"
)

type SettingsHandlers struct {
	*domain.BaseHandler
	logger *zap.Logger
}

func NewSettingsHandlers(base *domain.BaseHandler, logger *zap.Logger) *SettingsHandlers {
	return &SettingsHandlers{
		BaseHandler: base,
		logger:      logger,
	}
}

func (h *SettingsHandlers) UpdateProfile(c *gin.Context) {
	user := middleware.GetUserIDFromContext(c)
	db := middleware.GetDBFromContext(c)

	h.logger.Info("Profile update requested",
		zap.String("user", user),
		zap.Bool("db_connected", db != nil),
	)

	firstName := c.PostForm("firstName")
	lastName := c.PostForm("lastName")
	email := c.PostForm("email")
	bio := c.PostForm("bio")
	location := c.PostForm("location")

	h.logger.Info("Updating profile",
		zap.String("user", user),
		zap.String("first_name", firstName),
		zap.String("last_name", lastName),
		zap.String("email", email),
		zap.String("bio", bio),
		zap.String("location", location),
	)

	// In real app, update database

	c.HTML(http.StatusOK, "", `<div class="text-green-500 text-sm mb-4">Profile updated successfully!</div>`)

	h.logger.Info("Profile updated successfully", zap.String("user", user))
}

func (h *SettingsHandlers) UpdatePreferences(c *gin.Context) {
	user := middleware.GetUserIDFromContext(c)

	h.logger.Info("Preferences update requested",
		zap.String("user", user),
	)

	// Get form values
	currency := c.PostForm("currency")
	language := c.PostForm("language")
	units := c.PostForm("units")
	timezone := c.PostForm("timezone")

	h.logger.Info("Updating preferences",
		zap.String("user", user),
		zap.String("currency", currency),
		zap.String("language", language),
		zap.String("units", units),
		zap.String("timezone", timezone),
	)

	// In real app, update user preferences in database

	c.HTML(http.StatusOK, "", `<div class="text-green-500 text-sm mb-4">Preferences updated successfully!</div>`)

	h.logger.Info("Preferences updated successfully", zap.String("user", user))
}

func (h *SettingsHandlers) UpdateNotifications(c *gin.Context) {
	user := middleware.GetUserIDFromContext(c)

	h.logger.Info("Notification settings update requested",
		zap.String("user", user),
	)

	// Get checkbox values
	emailNotifs := c.PostForm("emailNotifications") == "on"
	pushNotifs := c.PostForm("pushNotifications") == "on"
	recommendations := c.PostForm("recommendations") == "on"
	newsletter := c.PostForm("newsletter") == "on"

	h.logger.Info("Updating notification preferences",
		zap.String("user", user),
		zap.Bool("email_notifications", emailNotifs),
		zap.Bool("push_notifications", pushNotifs),
		zap.Bool("recommendations", recommendations),
		zap.Bool("newsletter", newsletter),
	)

	// In real app, update notification preferences in database

	c.HTML(http.StatusOK, "", `<div class="text-green-500 text-sm mb-4">Notification preferences updated!</div>`)

	h.logger.Info("Notification preferences updated successfully", zap.String("user", user))
}

func (h *SettingsHandlers) DeleteAccount(c *gin.Context) {
	user := middleware.GetUserIDFromContext(c)

	h.logger.Warn("Account deletion requested",
		zap.String("user", user),
		zap.String("ip", c.ClientIP()),
	)

	// In real app, this would require additional confirmation and actually delete the account
	c.HTML(http.StatusOK, "", `<div class="text-red-500 text-sm mb-4">Account deletion is not implemented in demo mode</div>`)

	h.logger.Info("Account deletion request processed (demo mode)", zap.String("user", user))
}

func (h *SettingsHandlers) ExportData(c *gin.Context) {
	user := middleware.GetUserIDFromContext(c)

	h.logger.Info("Data export requested",
		zap.String("user", user),
		zap.String("ip", c.ClientIP()),
	)

	// In real app, generate and return user data export
	c.HTML(http.StatusOK, "", `<div class="text-blue-500 text-sm mb-4">Data export will be sent to your email within 24 hours</div>`)

	h.logger.Info("Data export request processed", zap.String("user", user))
}

func (h *SettingsHandlers) ShowSettingsPage(c *gin.Context) {
	h.logger.Info("Settings page accessed", zap.String("user", middleware.GetUserIDFromContext(c)))

	// Render the final layout with our custom data.
	h.RenderPage(c, "Settings - Loci", "Settings", SettingsPage())
}
