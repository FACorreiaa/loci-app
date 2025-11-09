package handlers

import (
	"github.com/a-h/templ"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/FACorreiaa/go-templui/internal/app/middleware"
	"github.com/FACorreiaa/go-templui/internal/app/models"
	"github.com/FACorreiaa/go-templui/internal/app/pages"
)

type BaseHandler struct {
	Logger *zap.Logger
}

func NewBaseHandler(logger *zap.Logger) *BaseHandler {
	return &BaseHandler{Logger: logger}
}

func (h *BaseHandler) NewLayoutData(c *gin.Context, title, activeNav string, content templ.Component) models.LayoutTempl {
	user := middleware.GetUserFromContext(c)
	nav := models.MainNav
	if user == nil {
		nav = models.OfflineNav
	}

	return models.LayoutTempl{
		Title:     title,
		Content:   content,
		Nav:       nav,
		ActiveNav: activeNav,
		User:      user,
	}
}

func (h *BaseHandler) Render(c *gin.Context, status int, component templ.Component) {
	c.Status(status)
	component.Render(c.Request.Context(), c.Writer)
}

func (h *BaseHandler) RenderPage(c *gin.Context, title, activeNav string, content templ.Component) {
	// Always render the full layout with navbar
	// hx-boost will automatically swap the body content
	layoutData := h.NewLayoutData(c, title, activeNav, content)
	h.Render(c, 200, pages.LayoutPage(layoutData))
}

func (h *BaseHandler) ShowPricingPage(c *gin.Context) {
	h.RenderPage(c, "Pricing - Loci", "Pricing", pages.PricingPage())
}

func (h *BaseHandler) ShowAboutPage(c *gin.Context) {
	h.RenderPage(c, "About - Loci", "About", pages.AboutPage())
}
