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

type BaseHandler struct {
	Logger *zap.Logger
}

func NewBaseHandler(logger *zap.Logger) *BaseHandler {
	return &BaseHandler{Logger: logger}
}

func (h *BaseHandler) newLayoutData(c *gin.Context, title, activeNav string, content templ.Component) models.LayoutTempl {
	user := middleware.GetUserFromContext(c)
	return models.LayoutTempl{
		Title:     title,
		Content:   content,
		Nav:       mainNav,
		ActiveNav: activeNav,
		User:      user,
	}
}

func (h *BaseHandler) render(c *gin.Context, status int, component templ.Component) {
	c.Status(status)
	component.Render(c.Request.Context(), c.Writer)
}

func (h *BaseHandler) RenderPage(c *gin.Context, title, activeNav string, content templ.Component) {
	isHTMX := c.GetHeader("HX-Request") == "true"
	if isHTMX {
		h.render(c, 200, content)
	} else {
		layoutData := h.newLayoutData(c, title, activeNav, content)
		h.render(c, 200, pages.LayoutPage(layoutData))
	}
}
