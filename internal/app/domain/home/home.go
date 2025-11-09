package home

import (
	"github.com/a-h/templ"
	"github.com/gin-gonic/gin"

	"github.com/FACorreiaa/go-templui/internal/app/handlers"
	"github.com/FACorreiaa/go-templui/internal/app/middleware"
	pages2 "github.com/FACorreiaa/go-templui/internal/app/pages"
)

type HomeHandlers struct {
	*handlers.BaseHandler
}

func NewHomeHandlers(base *handlers.BaseHandler) *HomeHandlers {
	return &HomeHandlers{BaseHandler: base}
}

func (h *HomeHandlers) ShowHomePage(c *gin.Context) {
	var content templ.Component
	var activeNav string

	user := middleware.GetUserFromContext(c)

	if user != nil {
		content = pages2.LoggedInDashboard()
		activeNav = "Dashboard"
	} else {
		content = pages2.PublicLandingPage()
		activeNav = "Home"
	}

	h.RenderPage(c, "Loci - Discover Amazing Places", activeNav, content)
}
