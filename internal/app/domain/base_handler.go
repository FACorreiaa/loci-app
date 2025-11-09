package domain

import (
	"github.com/a-h/templ"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/FACorreiaa/go-templui/internal/app/domain/pages"
	"github.com/FACorreiaa/go-templui/internal/app/models"
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

func (h *BaseHandler) NewLayoutData(c *gin.Context, title, activeNav string, content templ.Component) models.LayoutTempl {
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

//func (h *BaseHandler) ShowSignInPage(c *gin.Context) {
//	h.Logger.Info("Sign in page accessed")
//
//	initialValues := auth.SignInFormValues{}
//	initialErrors := auth.SignInFormErrors{}
//	//var err error
//	//switch {
//	//case errors.Is(err, models.ErrInvalidEmailFormat):
//	//	initialErrors.Email = "Please enter a valid email address."
//	//case errors.Is(err, models.ErrUserNotFound):
//	//	initialErrors.General = "Invalid credentials. Please check your email and password."
//	//case errors.Is(err, models.ErrInvalidPassword):
//	//	initialErrors.General = "Invalid credentials. Please check your email and password."
//	//default:
//	//	initialErrors.General = ""
//	//}
//
//	content := auth.SignIn(initialValues, initialErrors)
//	layoutData := h.newLayoutData(c, "SignIn - Loci", "SignIn", content)
//	layoutData.Nav = offlineNav
//	layoutData.User = nil
//
//	h.render(c, http.StatusOK, pages.LayoutPage(layoutData))
//}

//func (h *BaseHandler) ShowSignUpPage(c *gin.Context) {
//	h.Logger.Info("Sign up page accessed")
//
//	content := auth.SignUp()
//	layoutData := h.NewLayoutData(c, "Signup - Loci", "Signup", content)
//	layoutData.Nav = offlineNav
//	layoutData.User = nil
//
//	h.Render(c, http.StatusOK, pages.LayoutPage(layoutData))
//}
//
//func (h *BaseHandler) ShowForgotPasswordPage(c *gin.Context) {
//	h.Logger.Info("Forgot password page accessed")
//
//	content := auth.ForgotPassword()
//	layoutData := h.NewLayoutData(c, "Forgot password - Loci", "Forgot password", content)
//	layoutData.Nav = offlineNav
//	layoutData.User = nil
//
//	h.Render(c, http.StatusOK, pages.LayoutPage(layoutData))
//}

//// ShowSignUpPage renders the sign-up page.
//func (h *BaseHandler) ShowSignUpPage(c *gin.Context) {
//	h.Logger.Info("Sign up page accessed")
//
//	layoutData := h.NewLayoutData(c, "Sign Up - Loci", "", SignUp())
//	layoutData.Nav = offlineNav
//	layoutData.User = nil
//
//	h.Render(c, http.StatusOK, pages.LayoutPage(layoutData))
//}
//
//// ShowForgotPasswordPage renders the forgot password page.
//func (h *BaseHandler) ShowForgotPasswordPage(c *gin.Context) {
//	h.Logger.Info("Forgot password page accessed")
//
//	layoutData := h.NewLayoutData(c, "Reset Password - Loci", "", ForgotPassword())
//	layoutData.Nav = offlineNav
//	layoutData.User = nil
//
//	h.Render(c, http.StatusOK, pages.LayoutPage(layoutData))
//}
