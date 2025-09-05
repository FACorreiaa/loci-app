package routes

import (
	"log/slog"
	"net/http"

	"github.com/a-h/templ"
	"github.com/FACorreiaa/go-templui/app/lib/models"
	"github.com/FACorreiaa/go-templui/app/lib/renderer"
	authPkg "github.com/FACorreiaa/go-templui/app/pkg/domain/auth"
	handlers2 "github.com/FACorreiaa/go-templui/app/pkg/handlers"
	"github.com/FACorreiaa/go-templui/app/pkg/config"
	"github.com/FACorreiaa/go-templui/app/pkg/logger"
	"github.com/FACorreiaa/go-templui/app/pkg/middleware"

	features "github.com/FACorreiaa/go-templui/app/lib/features"
	"github.com/FACorreiaa/go-templui/app/lib/features/auth"
	"github.com/FACorreiaa/go-templui/app/lib/features/bookmarks"
	"github.com/FACorreiaa/go-templui/app/lib/features/chat"
	"github.com/FACorreiaa/go-templui/app/lib/features/discover"
	"github.com/FACorreiaa/go-templui/app/lib/features/favorites"
	"github.com/FACorreiaa/go-templui/app/lib/features/lists"
	"github.com/FACorreiaa/go-templui/app/lib/features/profile"
	"github.com/FACorreiaa/go-templui/app/lib/features/recents"
	"github.com/FACorreiaa/go-templui/app/lib/features/settings"
	"github.com/FACorreiaa/go-templui/app/lib/pages"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

func getUserFromContext(c *gin.Context) *models.User {
	userID, userEmail, userName := middleware.GetUserFromContext(c)
	if userID == "anonymous" {
		return nil
	}
	return &models.User{
		ID:       userID,
		Name:     userName,
		Email:    userEmail,
		IsActive: true,
	}
}

func getDBFromContext(c *gin.Context) *pgxpool.Pool {
	if db, exists := c.Get("db"); exists {
		return db.(*pgxpool.Pool)
	}
	return nil
}

func Setup(r *gin.Engine) {
	// Setup custom templ renderer
	ginHtmlRenderer := r.HTMLRender
	r.HTMLRender = &renderer.HTMLTemplRenderer{FallbackHtmlRenderer: ginHtmlRenderer}

	// Assets
	r.Static("/assets", "./assets")
	r.Static("/static", "./assets/static")
	r.StaticFile("/sw.js", "./assets/static/sw.js")
	r.StaticFile("/manifest.json", "./assets/static/manifest.json")

	// Initialize handlers
	cfg, _ := config.Load()
	
	// For now, create a minimal auth handler that can work without full database setup
	// The handlers will handle missing database gracefully for basic functionality
	authRepo := authPkg.NewPostgresAuthRepo(nil, slog.Default())
	authHandlers := authPkg.NewAuthHandlers(authRepo, cfg, slog.Default())
	chatHandlers := handlers2.NewChatHandlers()
	favoritesHandlers := handlers2.NewFavoritesHandlers()
	bookmarksHandlers := handlers2.NewBookmarksHandlers()
	discoverHandlers := handlers2.NewDiscoverHandlers()
	settingsHandlers := handlers2.NewSettingsHandlers()

	// Public routes (with optional auth)
	r.GET("/", middleware.OptionalAuthMiddleware(), func(c *gin.Context) {
		logger.Log.Info("Landing page accessed",
			zap.String("ip", c.ClientIP()),
			zap.String("user_agent", c.GetHeader("User-Agent")),
		)
		
		user := getUserFromContext(c)
		var content templ.Component
		if user != nil {
			content = features.LoggedInDashboard()
		} else {
			content = features.PublicLandingPage()
		}
		
		c.HTML(http.StatusOK, "", pages.LayoutPage(models.LayoutTempl{
			Title:   "Loci - Discover Amazing Places",
			Content: content,
			Nav: models.Navigation{
				Items: []models.NavItem{
					{Name: "Home", URL: "/"},
					{Name: "Discover", URL: "/discover"},
					{Name: "About", URL: "/about"},
				},
			},
			ActiveNav: "Home",
			User:      user,
		}))
	})

	// Discover (public but enhanced when authenticated)
	r.GET("/discover", middleware.OptionalAuthMiddleware(), func(c *gin.Context) {
		logger.Log.Info("Discover page accessed", zap.String("ip", c.ClientIP()))
		c.HTML(http.StatusOK, "", pages.LayoutPage(models.LayoutTempl{
			Title:   "Discover - Loci",
			Content: discover.DiscoverPage(),
			Nav: models.Navigation{
				Items: []models.NavItem{
					{Name: "Home", URL: "/"},
					{Name: "Discover", URL: "/discover"},
					{Name: "About", URL: "/about"},
				},
			},
			ActiveNav: "Discover",
			User:      getUserFromContext(c),
		}))
	})

	// Auth routes
	authGroup := r.Group("/auth")
	{
		authGroup.GET("/signin", func(c *gin.Context) {
			logger.Log.Info("Sign in page accessed")
			c.HTML(http.StatusOK, "", pages.LayoutPage(models.LayoutTempl{
				Title:   "Sign In - Loci",
				Content: auth.SignIn(),
				Nav: models.Navigation{
					Items: []models.NavItem{
						{Name: "Home", URL: "/"},
						{Name: "Discover", URL: "/discover"},
					},
				},
				ActiveNav: "",
				User:      nil,
			}))
		})

		authGroup.GET("/signup", func(c *gin.Context) {
			logger.Log.Info("Sign up page accessed")
			c.HTML(http.StatusOK, "", pages.LayoutPage(models.LayoutTempl{
				Title:   "Sign Up - Loci",
				Content: auth.SignUp(),
				Nav: models.Navigation{
					Items: []models.NavItem{
						{Name: "Home", URL: "/"},
						{Name: "Discover", URL: "/discover"},
					},
				},
				ActiveNav: "",
				User:      nil,
			}))
		})

		authGroup.GET("/forgot-password", func(c *gin.Context) {
			logger.Log.Info("Forgot password page accessed")
			c.HTML(http.StatusOK, "", pages.LayoutPage(models.LayoutTempl{
				Title:   "Reset Password - Loci",
				Content: auth.ForgotPassword(),
				Nav: models.Navigation{
					Items: []models.NavItem{
						{Name: "Home", URL: "/"},
						{Name: "Discover", URL: "/discover"},
					},
				},
				ActiveNav: "",
				User:      nil,
			}))
		})

		authGroup.POST("/signin", gin.WrapF(authHandlers.LoginHandler))
		authGroup.POST("/signup", gin.WrapF(authHandlers.RegisterHandler))
		authGroup.POST("/logout", gin.WrapF(authHandlers.LogoutHandler))
		authGroup.POST("/forgot-password", gin.WrapF(authHandlers.ForgotPasswordHandler))
		authGroup.POST("/change-password", gin.WrapF(authHandlers.ChangePasswordHandler))
		authGroup.POST("/check-username", gin.WrapF(authHandlers.CheckUsernameHandler))
	}

	// Protected routes
	protected := r.Group("/")
	protected.Use(middleware.AuthMiddleware())
	{
		// Dashboard (authenticated landing)
		protected.GET("/dashboard", func(c *gin.Context) {
			logger.Log.Info("Dashboard accessed", zap.String("user", middleware.GetUserIDFromContext(c)))
			c.HTML(http.StatusOK, "", pages.LayoutPage(models.LayoutTempl{
				Title:   "Dashboard - Loci",
				Content: features.LoggedInDashboard(),
				Nav: models.Navigation{
					Items: []models.NavItem{
						{Name: "Dashboard", URL: "/dashboard"},
						{Name: "Discover", URL: "/discover"},
						{Name: "Chat", URL: "/chat"},
						{Name: "Favorites", URL: "/favorites"},
					},
				},
				ActiveNav: "Dashboard",
				User:      getUserFromContext(c),
			}))
		})

		// Chat
		protected.GET("/chat", func(c *gin.Context) {
			logger.Log.Info("Chat page accessed", zap.String("user", middleware.GetUserIDFromContext(c)))
			c.HTML(http.StatusOK, "", pages.LayoutPage(models.LayoutTempl{
				Title:   "AI Chat - Loci",
				Content: chat.ChatPage(),
				Nav: models.Navigation{
					Items: []models.NavItem{
						{Name: "Dashboard", URL: "/dashboard"},
						{Name: "Discover", URL: "/discover"},
						{Name: "Chat", URL: "/chat"},
						{Name: "Favorites", URL: "/favorites"},
					},
				},
				ActiveNav: "Chat",
				User:      getUserFromContext(c),
			}))
		})

		// Favorites
		protected.GET("/favorites", func(c *gin.Context) {
			logger.Log.Info("Favorites page accessed", zap.String("user", middleware.GetUserIDFromContext(c)))
			c.HTML(http.StatusOK, "", pages.LayoutPage(models.LayoutTempl{
				Title:   "Favorites - Loci",
				Content: favorites.FavoritesPage(),
				Nav: models.Navigation{
					Items: []models.NavItem{
						{Name: "Dashboard", URL: "/dashboard"},
						{Name: "Discover", URL: "/discover"},
						{Name: "Chat", URL: "/chat"},
						{Name: "Favorites", URL: "/favorites"},
					},
				},
				ActiveNav: "Favorites",
				User:      getUserFromContext(c),
			}))
		})

		// Bookmarks
		protected.GET("/bookmarks", func(c *gin.Context) {
			logger.Log.Info("Bookmarks page accessed", zap.String("user", middleware.GetUserIDFromContext(c)))
			c.HTML(http.StatusOK, "", pages.LayoutPage(models.LayoutTempl{
				Title:   "Bookmarks - Loci",
				Content: bookmarks.BookmarksPage(),
				Nav: models.Navigation{
					Items: []models.NavItem{
						{Name: "Dashboard", URL: "/dashboard"},
						{Name: "Discover", URL: "/discover"},
						{Name: "Chat", URL: "/chat"},
						{Name: "Favorites", URL: "/favorites"},
					},
				},
				ActiveNav: "Bookmarks",
				User:      getUserFromContext(c),
			}))
		})

		// Lists
		protected.GET("/lists", func(c *gin.Context) {
			logger.Log.Info("Lists page accessed", zap.String("user", middleware.GetUserIDFromContext(c)))
			c.HTML(http.StatusOK, "", pages.LayoutPage(models.LayoutTempl{
				Title:   "Travel Lists - Loci",
				Content: lists.ListsPage(),
				Nav: models.Navigation{
					Items: []models.NavItem{
						{Name: "Dashboard", URL: "/dashboard"},
						{Name: "Discover", URL: "/discover"},
						{Name: "Chat", URL: "/chat"},
						{Name: "Lists", URL: "/lists"},
					},
				},
				ActiveNav: "Lists",
				User:      getUserFromContext(c),
			}))
		})

		// Profile
		protected.GET("/profile", func(c *gin.Context) {
			logger.Log.Info("Profile page accessed", zap.String("user", middleware.GetUserIDFromContext(c)))
			c.HTML(http.StatusOK, "", pages.LayoutPage(models.LayoutTempl{
				Title:   "Profile - Loci",
				Content: profile.ProfilePage(),
				Nav: models.Navigation{
					Items: []models.NavItem{
						{Name: "Dashboard", URL: "/dashboard"},
						{Name: "Discover", URL: "/discover"},
						{Name: "Profile", URL: "/profile"},
						{Name: "Settings", URL: "/settings"},
					},
				},
				ActiveNav: "Profile",
				User:      getUserFromContext(c),
			}))
		})

		// Recents
		protected.GET("/recents", func(c *gin.Context) {
			logger.Log.Info("Recents page accessed", zap.String("user", middleware.GetUserIDFromContext(c)))
			c.HTML(http.StatusOK, "", pages.LayoutPage(models.LayoutTempl{
				Title:   "Recent Activity - Loci",
				Content: recents.RecentsPage(),
				Nav: models.Navigation{
					Items: []models.NavItem{
						{Name: "Dashboard", URL: "/dashboard"},
						{Name: "Recents", URL: "/recents"},
						{Name: "Settings", URL: "/settings"},
					},
				},
				ActiveNav: "Recents",
				User:      getUserFromContext(c),
			}))
		})

		// Settings
		protected.GET("/settings", func(c *gin.Context) {
			logger.Log.Info("Settings page accessed", zap.String("user", middleware.GetUserIDFromContext(c)))
			c.HTML(http.StatusOK, "", pages.LayoutPage(models.LayoutTempl{
				Title:   "Settings - Loci",
				Content: settings.SettingsPage(),
				Nav: models.Navigation{
					Items: []models.NavItem{
						{Name: "Dashboard", URL: "/dashboard"},
						{Name: "Profile", URL: "/profile"},
						{Name: "Settings", URL: "/settings"},
					},
				},
				ActiveNav: "Settings",
				User:      getUserFromContext(c),
			}))
		})
	}

	// HTMX API routes
	htmxGroup := r.Group("/")
	{
		// Chat endpoints
		htmxGroup.POST("/chat/message", chatHandlers.SendMessage)

		// Favorites endpoints
		htmxGroup.POST("/favorites/add/:id", favoritesHandlers.AddFavorite)
		htmxGroup.DELETE("/favorites/:id", favoritesHandlers.RemoveFavorite)
		htmxGroup.POST("/favorites/search", favoritesHandlers.SearchFavorites)

		// Bookmarks endpoints
		htmxGroup.POST("/bookmarks/add/:id", bookmarksHandlers.AddBookmark)
		htmxGroup.DELETE("/bookmarks/:id", bookmarksHandlers.RemoveBookmark)
		htmxGroup.POST("/bookmarks/search", bookmarksHandlers.SearchBookmarks)

		// Discover endpoints
		htmxGroup.POST("/discover/search", discoverHandlers.Search)
		htmxGroup.GET("/discover/category/:category", discoverHandlers.GetCategory)

		// Settings endpoints (protected)
		settingsGroup := htmxGroup.Group("/settings")
		settingsGroup.Use(middleware.AuthMiddleware())
		{
			settingsGroup.POST("/profile", settingsHandlers.UpdateProfile)
			settingsGroup.POST("/preferences", settingsHandlers.UpdatePreferences)
			settingsGroup.POST("/notifications", settingsHandlers.UpdateNotifications)
			settingsGroup.DELETE("/account", settingsHandlers.DeleteAccount)
			settingsGroup.POST("/export", settingsHandlers.ExportData)
		}
	}
}
