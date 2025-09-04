package main

import (
	"context"
	"fmt"

	database "github.com/FACorreiaa/go-templui/app/db"
	"github.com/FACorreiaa/go-templui/app/features"
	"github.com/FACorreiaa/go-templui/app/features/auth"
	"github.com/FACorreiaa/go-templui/app/features/discover"
	"github.com/FACorreiaa/go-templui/app/features/settings"
	"github.com/FACorreiaa/go-templui/app/features/chat"
	"github.com/FACorreiaa/go-templui/app/features/favorites"
	"github.com/FACorreiaa/go-templui/app/features/lists"
	"github.com/FACorreiaa/go-templui/app/features/bookmarks"
	"github.com/FACorreiaa/go-templui/app/features/profile"
	"github.com/FACorreiaa/go-templui/app/features/recents"
	"github.com/FACorreiaa/go-templui/app/pages"
	"github.com/FACorreiaa/go-templui/app/models"
	"github.com/FACorreiaa/go-templui/pkg/config"
	"github.com/FACorreiaa/go-templui/pkg/logger"
	authPkg "github.com/FACorreiaa/go-templui/pkg/auth"
	"github.com/FACorreiaa/go-templui/pkg/handlers"
	"github.com/FACorreiaa/go-templui/pkg/middleware"
	"github.com/a-h/templ"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func main() {
	// Initialize logger
	err := logger.Init(zapcore.InfoLevel, zap.String("service", "loci-templui"))
	if err != nil {
		panic(fmt.Sprintf("Failed to initialize logger: %v", err))
	}

	logger.Log.Info("Starting Loci TemplUI application")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		logger.Log.Fatal("Failed to load configuration", zap.Error(err))
	}

	// Setup database
	ctx := context.Background()
	dbPool, err := setupDatabase(ctx, cfg)
	if err != nil {
		logger.Log.Fatal("Failed to setup database", zap.Error(err))
	}
	defer dbPool.Close()

	// Set Gin mode
	gin.SetMode(gin.ReleaseMode)
	
	// Create Gin router
	r := gin.New()
	
	// Setup middleware (inject database pool into context)
	r.Use(middleware.LoggerMiddleware())
	r.Use(gin.Recovery())
	r.Use(middleware.CORSMiddleware())
	r.Use(middleware.SecurityMiddleware())
	r.Use(func(c *gin.Context) {
		c.Set("db", dbPool)
		c.Next()
	})

	// Setup routes
	setupRoutes(r)

	// Start server
	serverPort := ":" + cfg.ServerPort
	logger.Log.Info("Server starting", zap.String("port", cfg.ServerPort))
	if err := r.Run(serverPort); err != nil {
		logger.Log.Fatal("Failed to start server", zap.Error(err))
	}
}

func setupDatabase(ctx context.Context, cfg *config.Config) (*pgxpool.Pool, error) {
	logger.Log.Info("Setting up database connection and migrations")

	dbConfig, err := database.NewDatabaseConfig(cfg, logger.Log)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database configuration: %w", err)
	}

	pool, err := database.Init(dbConfig.ConnectionURL, logger.Log)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database pool: %w", err)
	}

	database.WaitForDB(ctx, pool, logger.Log)
	logger.Log.Info("Connected to Postgres",
		zap.String("host", cfg.Repositories.Postgres.Host),
		zap.String("port", cfg.Repositories.Postgres.Port),
		zap.String("database", cfg.Repositories.Postgres.DB))

	if err = database.RunMigrations(dbConfig.ConnectionURL, logger.Log); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	logger.Log.Info("Database setup completed successfully")
	return pool, nil
}

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

func setupRoutes(r *gin.Engine) {
	// Assets
	r.Static("/assets", "./assets")
	
	// Initialize handlers
	authHandlers := authPkg.NewAuthHandlers()
	chatHandlers := handlers.NewChatHandlers()
	favoritesHandlers := handlers.NewFavoritesHandlers()
	bookmarksHandlers := handlers.NewBookmarksHandlers()
	discoverHandlers := handlers.NewDiscoverHandlers()
	settingsHandlers := handlers.NewSettingsHandlers()
	
	// Public routes
	r.GET("/", func(c *gin.Context) {
		logger.Log.Info("Landing page accessed",
			zap.String("ip", c.ClientIP()),
			zap.String("user_agent", c.GetHeader("User-Agent")),
		)
		templ.Handler(pages.LayoutPage(models.LayoutTempl{
			Title: "Loci - Discover Amazing Places",
			Content: features.PublicLandingPage(),
			Nav: models.Navigation{
				Items: []models.NavItem{
					{Name: "Home", URL: "/"},
					{Name: "Discover", URL: "/discover"},
					{Name: "About", URL: "/about"},
				},
			},
			ActiveNav: "Home",
			User: nil,
		})).ServeHTTP(c.Writer, c.Request)
	})
	
	// Discover (public but enhanced when authenticated)
	r.GET("/discover", func(c *gin.Context) {
		logger.Log.Info("Discover page accessed", zap.String("ip", c.ClientIP()))
		templ.Handler(pages.LayoutPage(models.LayoutTempl{
			Title: "Discover - Loci",
			Content: discover.DiscoverPage(),
			Nav: models.Navigation{
				Items: []models.NavItem{
					{Name: "Home", URL: "/"},
					{Name: "Discover", URL: "/discover"},
					{Name: "About", URL: "/about"},
				},
			},
			ActiveNav: "Discover",
			User: nil,
		})).ServeHTTP(c.Writer, c.Request)
	})
	
	// Auth routes
	authGroup := r.Group("/auth")
	{
		authGroup.GET("/signin", func(c *gin.Context) {
			logger.Log.Info("Sign in page accessed")
			templ.Handler(pages.LayoutPage(models.LayoutTempl{
				Title: "Sign In - Loci",
				Content: auth.SignIn(),
				Nav: models.Navigation{
					Items: []models.NavItem{
						{Name: "Home", URL: "/"},
						{Name: "Discover", URL: "/discover"},
					},
				},
				ActiveNav: "",
				User: nil,
			})).ServeHTTP(c.Writer, c.Request)
		})
		
		authGroup.GET("/signup", func(c *gin.Context) {
			logger.Log.Info("Sign up page accessed")
			templ.Handler(pages.LayoutPage(models.LayoutTempl{
				Title: "Sign Up - Loci",
				Content: auth.SignUp(),
				Nav: models.Navigation{
					Items: []models.NavItem{
						{Name: "Home", URL: "/"},
						{Name: "Discover", URL: "/discover"},
					},
				},
				ActiveNav: "",
				User: nil,
			})).ServeHTTP(c.Writer, c.Request)
		})
		
		authGroup.GET("/forgot-password", func(c *gin.Context) {
			logger.Log.Info("Forgot password page accessed")
			templ.Handler(pages.LayoutPage(models.LayoutTempl{
				Title: "Reset Password - Loci",
				Content: auth.ForgotPassword(),
				Nav: models.Navigation{
					Items: []models.NavItem{
						{Name: "Home", URL: "/"},
						{Name: "Discover", URL: "/discover"},
					},
				},
				ActiveNav: "",
				User: nil,
			})).ServeHTTP(c.Writer, c.Request)
		})
		
		authGroup.POST("/signin", gin.WrapF(authHandlers.LoginHandler))
		authGroup.POST("/signup", gin.WrapF(authHandlers.RegisterHandler))
		authGroup.POST("/logout", gin.WrapF(authHandlers.LogoutHandler))
		authGroup.POST("/change-password", gin.WrapF(authHandlers.ChangePasswordHandler))
	}
	
	// Protected routes
	protected := r.Group("/")
	protected.Use(middleware.AuthMiddleware())
	{
		// Dashboard (authenticated landing)
		protected.GET("/dashboard", func(c *gin.Context) {
			logger.Log.Info("Dashboard accessed", zap.String("user", middleware.GetUserIDFromContext(c)))
			templ.Handler(pages.LayoutPage(models.LayoutTempl{
				Title: "Dashboard - Loci",
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
				User: getUserFromContext(c),
			})).ServeHTTP(c.Writer, c.Request)
		})
		
		// Chat
		protected.GET("/chat", func(c *gin.Context) {
			logger.Log.Info("Chat page accessed", zap.String("user", middleware.GetUserIDFromContext(c)))
			templ.Handler(pages.LayoutPage(models.LayoutTempl{
				Title: "AI Chat - Loci",
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
				User: getUserFromContext(c),
			})).ServeHTTP(c.Writer, c.Request)
		})
		
		// Favorites
		protected.GET("/favorites", func(c *gin.Context) {
			logger.Log.Info("Favorites page accessed", zap.String("user", middleware.GetUserIDFromContext(c)))
			templ.Handler(pages.LayoutPage(models.LayoutTempl{
				Title: "Favorites - Loci",
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
				User: getUserFromContext(c),
			})).ServeHTTP(c.Writer, c.Request)
		})
		
		// Bookmarks
		protected.GET("/bookmarks", func(c *gin.Context) {
			logger.Log.Info("Bookmarks page accessed", zap.String("user", middleware.GetUserIDFromContext(c)))
			templ.Handler(pages.LayoutPage(models.LayoutTempl{
				Title: "Bookmarks - Loci",
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
				User: getUserFromContext(c),
			})).ServeHTTP(c.Writer, c.Request)
		})
		
		// Lists
		protected.GET("/lists", func(c *gin.Context) {
			logger.Log.Info("Lists page accessed", zap.String("user", middleware.GetUserIDFromContext(c)))
			templ.Handler(pages.LayoutPage(models.LayoutTempl{
				Title: "Travel Lists - Loci",
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
				User: getUserFromContext(c),
			})).ServeHTTP(c.Writer, c.Request)
		})
		
		// Profile
		protected.GET("/profile", func(c *gin.Context) {
			logger.Log.Info("Profile page accessed", zap.String("user", middleware.GetUserIDFromContext(c)))
			templ.Handler(pages.LayoutPage(models.LayoutTempl{
				Title: "Profile - Loci",
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
				User: getUserFromContext(c),
			})).ServeHTTP(c.Writer, c.Request)
		})
		
		// Recents
		protected.GET("/recents", func(c *gin.Context) {
			logger.Log.Info("Recents page accessed", zap.String("user", middleware.GetUserIDFromContext(c)))
			templ.Handler(pages.LayoutPage(models.LayoutTempl{
				Title: "Recent Activity - Loci",
				Content: recents.RecentsPage(),
				Nav: models.Navigation{
					Items: []models.NavItem{
						{Name: "Dashboard", URL: "/dashboard"},
						{Name: "Recents", URL: "/recents"},
						{Name: "Settings", URL: "/settings"},
					},
				},
				ActiveNav: "Recents",
				User: getUserFromContext(c),
			})).ServeHTTP(c.Writer, c.Request)
		})
		
		// Settings
		protected.GET("/settings", func(c *gin.Context) {
			logger.Log.Info("Settings page accessed", zap.String("user", middleware.GetUserIDFromContext(c)))
			templ.Handler(pages.LayoutPage(models.LayoutTempl{
				Title: "Settings - Loci",
				Content: settings.SettingsPage(),
				Nav: models.Navigation{
					Items: []models.NavItem{
						{Name: "Dashboard", URL: "/dashboard"},
						{Name: "Profile", URL: "/profile"},
						{Name: "Settings", URL: "/settings"},
					},
				},
				ActiveNav: "Settings",
				User: getUserFromContext(c),
			})).ServeHTTP(c.Writer, c.Request)
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