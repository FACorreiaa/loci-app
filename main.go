package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/FACorreiaa/go-templui/app/features/landing"
	"github.com/FACorreiaa/go-templui/app/features/auth"
	"github.com/FACorreiaa/go-templui/app/features/chat"
	"github.com/FACorreiaa/go-templui/app/features/favorites"
	"github.com/FACorreiaa/go-templui/app/features/bookmarks"
	"github.com/FACorreiaa/go-templui/app/features/lists"
	"github.com/FACorreiaa/go-templui/app/features/profile"
	"github.com/FACorreiaa/go-templui/app/features/recents"
	"github.com/FACorreiaa/go-templui/app/features/discover"
	"github.com/FACorreiaa/go-templui/app/features/settings"
	"github.com/FACorreiaa/go-templui/app/pages"
	"github.com/FACorreiaa/go-templui/assets"
	"github.com/FACorreiaa/go-templui/pkg/logger"
	authPkg "github.com/FACorreiaa/go-templui/pkg/auth"
	"github.com/a-h/templ"
	"github.com/gin-gonic/gin"
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

	// Set Gin mode
	gin.SetMode(gin.ReleaseMode)
	
	// Create Gin router
	r := gin.New()
	
	// Setup middleware
	r.Use(LoggerMiddleware())
	r.Use(gin.Recovery())
	r.Use(CORSMiddleware())

	// Setup routes
	setupRoutes(r)

	// Start server
	logger.Log.Info("Server starting on port 8090")
	if err := r.Run(":8090"); err != nil {
		logger.Log.Fatal("Failed to start server", zap.Error(err))
	}
}

func setupRoutes(r *gin.Engine) {
	// Assets
	r.Static("/assets", "./assets")
	
	// Auth handlers
	authHandlers := authPkg.NewAuthHandlers()
	
	// Public routes
	r.GET("/", func(c *gin.Context) {
		logger.Log.Info("Landing page accessed",
			zap.String("ip", c.ClientIP()),
			zap.String("user_agent", c.GetHeader("User-Agent")),
		)
		templ.Handler(pages.Layout(landing.PublicLandingPage(), "Loci - Discover Amazing Places")).ServeHTTP(c.Writer, c.Request)
	})
	
	// Auth routes
	authGroup := r.Group("/auth")
	{
		authGroup.GET("/signin", func(c *gin.Context) {
			logger.Log.Info("Sign in page accessed")
			templ.Handler(pages.Layout(auth.SignIn(), "Sign In - Loci")).ServeHTTP(c.Writer, c.Request)
		})
		
		authGroup.GET("/signup", func(c *gin.Context) {
			logger.Log.Info("Sign up page accessed")
			templ.Handler(pages.Layout(auth.SignUp(), "Sign Up - Loci")).ServeHTTP(c.Writer, c.Request)
		})
		
		authGroup.GET("/forgot-password", func(c *gin.Context) {
			logger.Log.Info("Forgot password page accessed")
			templ.Handler(pages.Layout(auth.ForgotPassword(), "Reset Password - Loci")).ServeHTTP(c.Writer, c.Request)
		})
		
		authGroup.POST("/signin", gin.HandlerFunc(authHandlers.LoginHandler))
		authGroup.POST("/signup", gin.HandlerFunc(authHandlers.RegisterHandler))
		authGroup.POST("/logout", gin.HandlerFunc(authHandlers.LogoutHandler))
		authGroup.POST("/change-password", gin.HandlerFunc(authHandlers.ChangePasswordHandler))
	}
	
	// Protected routes
	protected := r.Group("/")
	protected.Use(AuthMiddleware())
	{
		// Dashboard (authenticated landing)
		protected.GET("/dashboard", func(c *gin.Context) {
			logger.Log.Info("Dashboard accessed", zap.String("user", getUserFromContext(c)))
			templ.Handler(pages.Layout(landing.LoggedInDashboard(), "Dashboard - Loci")).ServeHTTP(c.Writer, c.Request)
		})
		
		// Chat
		protected.GET("/chat", func(c *gin.Context) {
			logger.Log.Info("Chat page accessed", zap.String("user", getUserFromContext(c)))
			templ.Handler(pages.Layout(chat.ChatPage(), "AI Chat - Loci")).ServeHTTP(c.Writer, c.Request)
		})
		
		// Favorites
		protected.GET("/favorites", func(c *gin.Context) {
			logger.Log.Info("Favorites page accessed", zap.String("user", getUserFromContext(c)))
			templ.Handler(pages.Layout(favorites.FavoritesPage(), "Favorites - Loci")).ServeHTTP(c.Writer, c.Request)
		})
		
		// Bookmarks
		protected.GET("/bookmarks", func(c *gin.Context) {
			logger.Log.Info("Bookmarks page accessed", zap.String("user", getUserFromContext(c)))
			templ.Handler(pages.Layout(bookmarks.BookmarksPage(), "Bookmarks - Loci")).ServeHTTP(c.Writer, c.Request)
		})
		
		// Lists
		protected.GET("/lists", func(c *gin.Context) {
			logger.Log.Info("Lists page accessed", zap.String("user", getUserFromContext(c)))
			templ.Handler(pages.Layout(lists.ListsPage(), "Travel Lists - Loci")).ServeHTTP(c.Writer, c.Request)
		})
		
		// Profile
		protected.GET("/profile", func(c *gin.Context) {
			logger.Log.Info("Profile page accessed", zap.String("user", getUserFromContext(c)))
			templ.Handler(pages.Layout(profile.ProfilePage(), "Profile - Loci")).ServeHTTP(c.Writer, c.Request)
		})
		
		// Recents
		protected.GET("/recents", func(c *gin.Context) {
			logger.Log.Info("Recents page accessed", zap.String("user", getUserFromContext(c)))
			templ.Handler(pages.Layout(recents.RecentsPage(), "Recent Activity - Loci")).ServeHTTP(c.Writer, c.Request)
		})
		
		// Settings
		protected.GET("/settings", func(c *gin.Context) {
			logger.Log.Info("Settings page accessed", zap.String("user", getUserFromContext(c)))
			templ.Handler(pages.Layout(settings.SettingsPage(), "Settings - Loci")).ServeHTTP(c.Writer, c.Request)
		})
	}
	
	// Discover (public but enhanced when authenticated)
	r.GET("/discover", func(c *gin.Context) {
		logger.Log.Info("Discover page accessed", zap.String("ip", c.ClientIP()))
		templ.Handler(pages.Layout(discover.DiscoverPage(), "Discover - Loci")).ServeHTTP(c.Writer, c.Request)
	})
	
	// API routes for HTMX interactions
	api := r.Group("/api")
	api.Use(LoggerMiddleware())
	{
		// Chat API
		api.POST("/chat/message", func(c *gin.Context) {
			logger.Log.Info("Chat message received", zap.String("user", getUserFromContext(c)))
			c.JSON(http.StatusOK, gin.H{"status": "received"})
		})
		
		// Favorites API
		api.POST("/favorites/add/:id", func(c *gin.Context) {
			id := c.Param("id")
			logger.Log.Info("Adding to favorites", zap.String("id", id), zap.String("user", getUserFromContext(c)))
			c.JSON(http.StatusOK, gin.H{"status": "added"})
		})
		
		api.DELETE("/favorites/:id", func(c *gin.Context) {
			id := c.Param("id")
			logger.Log.Info("Removing from favorites", zap.String("id", id), zap.String("user", getUserFromContext(c)))
			c.JSON(http.StatusOK, gin.H{"status": "removed"})
		})
		
		// Search APIs
		api.POST("/discover/search", func(c *gin.Context) {
			query := c.PostForm("query")
			location := c.PostForm("location")
			logger.Log.Info("Discovery search", 
				zap.String("query", query), 
				zap.String("location", location),
				zap.String("user", getUserFromContext(c)),
			)
			c.JSON(http.StatusOK, gin.H{"results": []string{}})
		})
	}
}

// Middleware functions
func LoggerMiddleware() gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		// Process request
		c.Next()

		// Log request
		param := gin.LogFormatterParams{
			Request:    c.Request,
			TimeStamp:  time.Now(),
			Latency:    time.Since(start),
			ClientIP:   c.ClientIP(),
			Method:     c.Request.Method,
			StatusCode: c.Writer.Status(),
			ErrorMessage: c.Errors.ByType(gin.ErrorTypePrivate).String(),
		}

		if raw != "" {
			param.Path = path + "?" + raw
		} else {
			param.Path = path
		}

		logger.Log.Info("HTTP Request",
			zap.String("method", param.Method),
			zap.String("path", param.Path),
			zap.String("ip", param.ClientIP),
			zap.Int("status", param.StatusCode),
			zap.Duration("latency", param.Latency),
			zap.String("user_agent", c.GetHeader("User-Agent")),
		)
	})
}

func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := c.Cookie("auth_token")
		if err != nil {
			logger.Log.Warn("No auth token found", zap.String("path", c.Request.URL.Path))
			c.Redirect(http.StatusFound, "/auth/signin")
			c.Abort()
			return
		}

		// Validate token (simplified for demo)
		if token == "" {
			logger.Log.Warn("Empty auth token", zap.String("path", c.Request.URL.Path))
			c.Redirect(http.StatusFound, "/auth/signin")
			c.Abort()
			return
		}

		// Set user context (simplified)
		c.Set("user_id", "demo-user")
		c.Set("user_email", "demo@loci.app")
		c.Next()
	}
}

func getUserFromContext(c *gin.Context) string {
	if userID, exists := c.Get("user_id"); exists {
		return userID.(string)
	}
	return "anonymous"
}