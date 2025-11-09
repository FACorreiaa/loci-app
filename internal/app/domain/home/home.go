package home

import (
	"github.com/a-h/templ"
	"github.com/gin-gonic/gin"

	"github.com/FACorreiaa/go-templui/internal/app/domain"
	"github.com/FACorreiaa/go-templui/internal/app/domain/pages"
)

type HomeHandlers struct {
	*domain.BaseHandler
}

func NewHomeHandlers(base *domain.BaseHandler) *HomeHandlers {
	return &HomeHandlers{BaseHandler: base}
}

func (h *HomeHandlers) ShowHomePage(c *gin.Context) {
	var content templ.Component
	var activeNav string

	user := common.GetUserFromContext(c)

	if user != nil {
		content = pages.LoggedInDashboard()
		activeNav = "Dashboard"
	} else {
		content = pages.PublicLandingPage()
		activeNav = "Home"
	}

	h.RenderPage(c, "Loci - Discover Amazing Places", activeNav, content)
}
