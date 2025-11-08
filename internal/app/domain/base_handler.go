package domain

import (
	"github.com/a-h/templ"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/FACorreiaa/go-templui/internal/app/models"
	"github.com/FACorreiaa/go-templui/internal/app/pages"
	"github.com/FACorreiaa/go-templui/internal/pkg/middleware"
)

var mainNav = models.Navigation{
	Items: []models.NavItem{
		{Name: "Dashboard", URL: "/dashboard"},
		{Name: "Discover", URL: "/discover"},
		{Name: "Nearby", URL: "/nearby"},
		{Name: "Chat", URL: "/chat"},
		{Name: "Favorites", URL: "/favorites"},
	},
}

var offlineNav = models.Navigation{
	Items: []models.NavItem{
		{Name: "About", URL: "/about"},
		{Name: "Features", URL: "/features"},
		{Name: "Pricing", URL: "/pricing"},
	},
}

type BaseHandler struct {
	Logger *zap.Logger
}

func NewBaseHandler(logger *zap.Logger) *BaseHandler {
	return &BaseHandler{Logger: logger}
}

func (h *BaseHandler) newLayoutData(c *gin.Context, title, activeNav string, content templ.Component) models.LayoutTempl {
	user := middleware.GetUserFromContext(c)
	nav := mainNav
	if user == nil {
		nav = offlineNav
	}

	return models.LayoutTempl{
		Title:     title,
		Content:   content,
		Nav:       nav,
		ActiveNav: activeNav,
		User:      user,
	}
}

func (h *BaseHandler) render(c *gin.Context, status int, component templ.Component) {
	c.Status(status)
	component.Render(c.Request.Context(), c.Writer)
}

func (h *BaseHandler) RenderPage(c *gin.Context, title, activeNav string, content templ.Component) {
	// Always render the full layout with navbar
	// hx-boost will automatically swap the body content
	layoutData := h.newLayoutData(c, title, activeNav, content)
	h.render(c, 200, pages.LayoutPage(layoutData))
}
