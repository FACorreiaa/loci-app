package routes

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/pprof"
	"os"
	"time"

	"github.com/a-h/templ"
	"github.com/google/uuid"

	"github.com/FACorreiaa/go-templui/internal/app/pages/home"

	"github.com/FACorreiaa/go-templui/internal/app/domain/about"

	"github.com/FACorreiaa/go-templui/internal/app/domain/chat"

	"github.com/FACorreiaa/go-templui/internal/app/domain/profile"

	"github.com/FACorreiaa/go-templui/internal/app/domain/lists"
	"github.com/FACorreiaa/go-templui/internal/app/domain/reviews"

	"github.com/FACorreiaa/go-templui/internal/app/common"
	"github.com/FACorreiaa/go-templui/internal/app/domain/activities"
	auth2 "github.com/FACorreiaa/go-templui/internal/app/domain/auth"
	"github.com/FACorreiaa/go-templui/internal/app/domain/billing"
	"github.com/FACorreiaa/go-templui/internal/app/domain/bookmarks"
	llmchat2 "github.com/FACorreiaa/go-templui/internal/app/domain/chat_prompt"
	cityPkg "github.com/FACorreiaa/go-templui/internal/app/domain/city"
	"github.com/FACorreiaa/go-templui/internal/app/domain/discover"
	"github.com/FACorreiaa/go-templui/internal/app/domain/favorites"
	"github.com/FACorreiaa/go-templui/internal/app/domain/hotels"
	interestsPkg "github.com/FACorreiaa/go-templui/internal/app/domain/interests"
	locationPkg "github.com/FACorreiaa/go-templui/internal/app/domain/location"
	"github.com/FACorreiaa/go-templui/internal/app/domain/nearby"
	"github.com/FACorreiaa/go-templui/internal/app/domain/poi"
	"github.com/FACorreiaa/go-templui/internal/app/domain/profiles"
	"github.com/FACorreiaa/go-templui/internal/app/domain/recents"
	"github.com/FACorreiaa/go-templui/internal/app/domain/restaurants"
	"github.com/FACorreiaa/go-templui/internal/app/domain/results"
	"github.com/FACorreiaa/go-templui/internal/app/domain/settings"
	streamingfeatures "github.com/FACorreiaa/go-templui/internal/app/domain/streaming"
	tagsPkg "github.com/FACorreiaa/go-templui/internal/app/domain/tags"
	"github.com/FACorreiaa/go-templui/internal/app/domain/user"
	"github.com/FACorreiaa/go-templui/internal/app/models"
	"github.com/FACorreiaa/go-templui/internal/app/renderer"
	"github.com/FACorreiaa/go-templui/internal/pkg/config"
	"github.com/FACorreiaa/go-templui/internal/pkg/logger"
	"github.com/FACorreiaa/go-templui/internal/pkg/middleware"

	generativeAI "github.com/FACorreiaa/go-genai-sdk/lib"

	"github.com/FACorreiaa/go-templui/internal/app/domain/auth"

	"github.com/FACorreiaa/go-templui/internal/app/domain/pricing"

	"github.com/FACorreiaa/go-templui/internal/app/pages"

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

func Setup(r *gin.Engine, dbPool *pgxpool.Pool, log *zap.Logger) {
	//r.Use(middleware.AuthMiddleware())
	// Setup custom templ renderer
	ginHTMLRenderer := r.HTMLRender
	r.HTMLRender = &renderer.HTMLTemplRenderer{FallbackHTMLRenderer: ginHTMLRenderer}

	// Pprof debugging routes
	debugGroup := r.Group("/debug/pprof")
	{
		debugGroup.GET("/", gin.WrapH(http.HandlerFunc(pprof.Index)))
		debugGroup.GET("/cmdline", gin.WrapH(http.HandlerFunc(pprof.Cmdline)))
		debugGroup.GET("/profile", gin.WrapH(http.HandlerFunc(pprof.Profile)))
		debugGroup.POST("/symbol", gin.WrapH(http.HandlerFunc(pprof.Symbol)))
		debugGroup.GET("/symbol", gin.WrapH(http.HandlerFunc(pprof.Symbol)))
		debugGroup.GET("/trace", gin.WrapH(http.HandlerFunc(pprof.Trace)))
		debugGroup.GET("/allocs", gin.WrapH(pprof.Handler("allocs")))
		debugGroup.GET("/block", gin.WrapH(pprof.Handler("block")))
		debugGroup.GET("/goroutine", gin.WrapH(pprof.Handler("goroutine")))
		debugGroup.GET("/heap", gin.WrapH(pprof.Handler("heap")))
		debugGroup.GET("/mutex", gin.WrapH(pprof.Handler("mutex")))
		debugGroup.GET("/threadcreate", gin.WrapH(pprof.Handler("threadcreate")))
	}

	// Initialize common
	cfg, err := config.Load()
	if err != nil {
		// Use default config if loading fails
		cfg = &config.Config{}
	}

	// Initialize repositories and services
	authRepo := auth2.NewPostgresAuthRepo(dbPool, slog.Default())

	// Create repositories
	profilesRepo := profiles.NewPostgresUserRepo(dbPool, slog.Default())
	interestsRepo := interestsPkg.NewRepositoryImpl(dbPool, slog.Default())
	cityRepo := cityPkg.NewCityRepository(dbPool, slog.Default())
	poiRepo := poi.NewRepository(dbPool, slog.Default())
	tagsRepo := tagsPkg.NewRepositoryImpl(dbPool, slog.Default())
	userRepo := user.NewPostgresUserRepo(dbPool, slog.Default())

	// Create services
	profilesService := profiles.NewUserProfilesService(profilesRepo, interestsRepo, tagsRepo, slog.Default())
	userService := user.NewUserService(userRepo, slog.Default())

	// Create embedding service for POI (can be nil if not using semantic search)
	// Change later GeminiAPIKey env variable
	var embeddingService *generativeAI.EmbeddingService
	if os.Getenv("GEMINI_API_KEY") != "" { // Check env var directly
		ctx := context.Background()                                      // Use a background context for initialization
		l := slog.Default()                                              // Use default logger
		embeddingService, err = generativeAI.NewEmbeddingService(ctx, l) // Call with correct args
		if err != nil {
			slog.Error("Failed to create embedding service", "error", err) // Handle error appropriately
		}
	}

	// Create chat LLM repository (needed by poiService for LLM logging)
	chatRepo := llmchat2.NewRepositoryImpl(dbPool, slog.Default())

	poiService := poi.NewServiceImpl(poiRepo, embeddingService, cityRepo, chatRepo, slog.Default())

	// Create recents repository and service
	recentsRepo := recents.NewRepository(dbPool, slog.Default())
	recentsService := recents.NewService(recentsRepo, slog.Default())

	// Create common
	authHandlers := auth2.NewAuthHandlers(authRepo, cfg, slog.Default())
	profilesHandlers := profiles.NewProfilesHandler(profilesService)
	interestsHandlers := interestsPkg.NewInterestsHandler(interestsRepo)
	tagsHandlers := tagsPkg.NewTagsHandler(tagsRepo)
	// Create chat LLM service
	chatService := llmchat2.NewLlmInteractiontService(
		interestsRepo,
		profilesRepo,
		profilesService,
		tagsRepo,
		chatRepo,
		cityRepo,
		poiRepo,
		slog.Default(),
	)
	chatHandlers := llmchat2.NewChatHandlers(chatService, profilesService, chatRepo)
	favoritesHandlers := favorites.NewFavoritesHandlers(poiService)
	bookmarksHandlers := bookmarks.NewBookmarksHandlers()
	discoverHandlers := discover.NewDiscoverHandlers(poiRepo, chatRepo, chatService, slog.Default())
	itineraryHandlers := interestsPkg.NewItineraryHandlers(chatRepo)
	activitiesHandlers := activities.NewActivitiesHandlers(chatRepo, log)
	hotelsHandlers := hotels.NewHotelsHandlers(chatRepo, log)
	restaurantsHandlers := restaurants.NewRestaurantsHandlers(chatRepo, log)
	settingsHandlers := settings.NewSettingsHandlers()
	resultsHandlers := results.NewResultsHandlers()
	filterHandlers := common.NewFilterHandlers(logger.Log.Sugar())
	recentsHandlers := recents.NewRecentsHandlers(recentsService, log)

	// Initialize location repository for nearby feature
	locationRepo := locationPkg.NewRepository(dbPool)
	nearbyHandler := nearby.NewNearbyHandler(slog.Default(), chatService, locationRepo)

	// Public routes (with optional auth)
	r.GET("/", middleware.OptionalAuthMiddleware(), func(c *gin.Context) {
		logger.Log.Info("Landing page accessed",
			zap.String("ip", c.ClientIP()),
			zap.String("user_agent", c.GetHeader("User-Agent")),
		)

		user := getUserFromContext(c)
		var content templ.Component
		if user != nil {
			content = home.LoggedInDashboard()
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

	// Pricing (public)
	r.GET("/pricing", func(c *gin.Context) {
		logger.Log.Info("Pricing page accessed", zap.String("ip", c.ClientIP()))
		c.HTML(http.StatusOK, "", pages.LayoutPage(models.LayoutTempl{
			Title:   "Pricing - Loci",
			Content: pricing.PricingPage(),
			Nav: models.Navigation{
				Items: []models.NavItem{
					{Name: "Home", URL: "/"},
					{Name: "Discover", URL: "/discover"},
					{Name: "Pricing", URL: "/pricing"},
					{Name: "About", URL: "/about"},
				},
			},
			ActiveNav: "Pricing",
			User:      getUserFromContext(c),
		}))
	})

	// About (public)
	r.GET("/about", func(c *gin.Context) {
		logger.Log.Info("About page accessed", zap.String("ip", c.ClientIP()))
		c.HTML(http.StatusOK, "", pages.LayoutPage(models.LayoutTempl{
			Title:   "About - Loci",
			Content: about.AboutPage(),
			Nav: models.Navigation{
				Items: []models.NavItem{
					{Name: "Home", URL: "/"},
					{Name: "Discover", URL: "/discover"},
					{Name: "Pricing", URL: "/pricing"},
					{Name: "About", URL: "/about"},
				},
			},
			ActiveNav: "About",
			User:      getUserFromContext(c),
		}))
	})

	// Discover (public but enhanced when authenticated)
	r.GET("/discover", middleware.OptionalAuthMiddleware(), func(c *gin.Context) {
		logger.Log.Info("Discover page accessed", zap.String("ip", c.ClientIP()))
		c.HTML(http.StatusOK, "", pages.LayoutPage(models.LayoutTempl{
			Title:   "Discover - Loci",
			Content: discoverHandlers.Show(c),
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

	// Discovery detail page
	r.GET("/discover/detail/:sessionId", middleware.OptionalAuthMiddleware(), func(c *gin.Context) {
		logger.Log.Info("Discovery detail page accessed",
			zap.String("sessionId", c.Param("sessionId")),
			zap.String("ip", c.ClientIP()))
		c.HTML(http.StatusOK, "", pages.LayoutPage(models.LayoutTempl{
			Title:   "Discovery Details - Loci",
			Content: discoverHandlers.ShowDetail(c),
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

	// Nearby (location-based discovery - public)
	r.GET("/nearby", middleware.OptionalAuthMiddleware(), func(c *gin.Context) {
		logger.Log.Info("Nearby page accessed", zap.String("ip", c.ClientIP()))
		c.HTML(http.StatusOK, "", pages.LayoutPage(models.LayoutTempl{
			Title:   "Nearby - Loci",
			Content: nearby.NearbyPage(),
			Nav: models.Navigation{
				Items: []models.NavItem{
					{Name: "Home", URL: "/"},
					{Name: "Discover", URL: "/discover"},
					{Name: "Nearby", URL: "/nearby"},
					{Name: "About", URL: "/about"},
				},
			},
			ActiveNav: "Nearby",
			User:      getUserFromContext(c),
		}))
	})

	// WebSocket endpoint for real-time nearby POI updates
	// Configure JWT authentication (optional - allows anonymous users)
	jwtSecret := os.Getenv("JWT_SECRET_KEY")
	if jwtSecret == "" {
		jwtSecret = "default-secret-key-change-in-production-min-32-chars"
		slog.Warn("JWT_SECRET_KEY not set, using default (INSECURE - set environment variable in production)")
	}

	jwtConfig := middleware.JWTConfig{
		SecretKey:       jwtSecret,
		TokenExpiration: 24 * time.Hour,
		Logger:          slog.Default(),
		Optional:        true, // Allow both authenticated and anonymous users
	}

	// Configure rate limiting for WebSocket connections
	wsRateLimiter := middleware.NewRateLimiter(
		slog.Default(),
		10,            // Max 10 WebSocket connections
		1*time.Minute, // Per minute
	)

	// Apply middleware: JWT auth (optional) + rate limiting
	r.GET("/ws/nearby",
		middleware.JWTAuthMiddleware(jwtConfig),
		middleware.WebSocketRateLimitMiddleware(wsRateLimiter),
		nearbyHandler.HandleWebSocket,
	)

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
		// Itinerary SSE
		protected.GET("/itinerary", func(c *gin.Context) {
			query := c.Query("q")
			sessionIDParam := c.Query("sessionId")

			// If there's a query but no sessionID, start new streaming
			if query != "" && sessionIDParam == "" {
				// Return the streaming trigger page wrapped in layout
				content := streamingfeatures.StreamingTriggerPage(query, "itinerary")
				c.HTML(http.StatusOK, "", pages.LayoutPage(models.LayoutTempl{
					Title:   "Travel Planner - Loci",
					Content: content,
					Nav: models.Navigation{
						Items: []models.NavItem{
							{Name: "Dashboard", URL: "/dashboard"},
							{Name: "Discover", URL: "/discover"},
							{Name: "Nearby", URL: "/nearby"},
							{Name: "Itinerary", URL: "/itinerary"},
							{Name: "Chat", URL: "/chat"},
							{Name: "Favorites", URL: "/favorites"},
						},
					},
					ActiveNav: "Itinerary",
					User:      getUserFromContext(c),
				}))
				return
			}

			// For sessionId cases or default page, call the SSE handler
			// This returns templ.Component that should be wrapped in layout
			content := itineraryHandlers.HandleItineraryPageSSE(c)
			c.HTML(http.StatusOK, "", pages.LayoutPage(models.LayoutTempl{
				Title:   "Travel Planner - Loci",
				Content: content,
				Nav: models.Navigation{
					Items: []models.NavItem{
						{Name: "Dashboard", URL: "/dashboard"},
						{Name: "Discover", URL: "/discover"},
						{Name: "Nearby", URL: "/nearby"},
						{Name: "Itinerary", URL: "/itinerary"},
						{Name: "Chat", URL: "/chat"},
						{Name: "Favorites", URL: "/favorites"},
					},
				},
				ActiveNav: "Itinerary",
				User:      getUserFromContext(c),
			}))
		})

		//Activities (public but enhanced when authenticated)
		protected.GET("/activities", func(c *gin.Context) {
			content := activitiesHandlers.HandleActivitiesPage(c)
			c.HTML(http.StatusOK, "", pages.LayoutPage(models.LayoutTempl{
				Title:   "Activities - Loci",
				Content: content,
				Nav: models.Navigation{
					Items: []models.NavItem{
						{Name: "Home", URL: "/"},
						{Name: "Discover", URL: "/discover"},
						{Name: "Activities", URL: "/activities"},
						{Name: "Hotels", URL: "/hotels"},
						{Name: "Restaurants", URL: "/restaurants"},
					},
				},
				ActiveNav: "Activities",
				User:      getUserFromContext(c),
			}))
		})

		//Hotels (public but enhanced when authenticated)
		protected.GET("/hotels", func(c *gin.Context) {
			content := hotelsHandlers.HandleHotelsPage(c)
			c.HTML(http.StatusOK, "", pages.LayoutPage(models.LayoutTempl{
				Title:   "Hotels - Loci",
				Content: content,
				Nav: models.Navigation{
					Items: []models.NavItem{
						{Name: "Home", URL: "/"},
						{Name: "Discover", URL: "/discover"},
						{Name: "Activities", URL: "/activities"},
						{Name: "Hotels", URL: "/hotels"},
						{Name: "Restaurants", URL: "/restaurants"},
					},
				},
				ActiveNav: "Hotels",
				User:      getUserFromContext(c),
			}))
		})

		// Restaurants (public but enhanced when authenticated)
		// protected.GET("/restaurants", func(c *gin.Context) {
		// 	content := restaurantsHandlers.HandleRestaurantsPage(c)
		// 	c.HTML(http.StatusOK, "", pages.LayoutPage(models.LayoutTempl{
		// 		Title:   "Restaurants - Loci",
		// 		Content: content,
		// 		Nav: models.Navigation{
		// 			Items: []models.NavItem{
		// 				{Name: "Home", URL: "/"},
		// 				{Name: "Discover", URL: "/discover"},
		// 				{Name: "Activities", URL: "/activities"},
		// 				{Name: "Hotels", URL: "/hotels"},
		// 				{Name: "Restaurants", URL: "/restaurants"},
		// 			},
		// 		},
		// 		ActiveNav: "Restaurants",
		// 		User:      getUserFromContext(c),
		// 	}))
		// })

		// Restaurants SSE
		//protected.GET("/restaurants", func(c *gin.Context) {
		//	content := restaurantsHandlers.HandleRestaurantsPageSSE(c)
		//	c.HTML(http.StatusOK, "", pages.LayoutPage(models.LayoutTempl{
		//		Title:   "Restaurants - Loci",
		//		Content: content,
		//		Nav: models.Navigation{
		//			Items: []models.NavItem{
		//				{Name: "Home", URL: "/"},
		//				{Name: "Discover", URL: "/discover"},
		//				{Name: "Activities", URL: "/activities"},
		//				{Name: "Hotels", URL: "/hotels"},
		//				{Name: "Restaurants", URL: "/restaurants"},
		//			},
		//		},
		//		ActiveNav: "Restaurants",
		//		User:      getUserFromContext(c),
		//	}))
		//})

		protected.GET("/restaurants", func(c *gin.Context) {
			query := c.Query("q")
			sessionIDParam := c.Query("sessionId")

			// If there's a query but no sessionID, start new streaming
			if query != "" && sessionIDParam == "" {
				// Return the streaming trigger page wrapped in layout
				content := streamingfeatures.StreamingTriggerPage(query, "restaurants")
				c.HTML(http.StatusOK, "", pages.LayoutPage(models.LayoutTempl{
					Title:   "Travel Planner - Loci",
					Content: content,
					Nav: models.Navigation{
						Items: []models.NavItem{
							{Name: "Dashboard", URL: "/dashboard"},
							{Name: "Discover", URL: "/discover"},
							{Name: "Nearby", URL: "/nearby"},
							{Name: "Itinerary", URL: "/itinerary"},
							{Name: "Chat", URL: "/chat"},
							{Name: "Favorites", URL: "/favorites"},
						},
					},
					ActiveNav: "Restaurants",
					User:      getUserFromContext(c),
				}))
				return
			}

			// For sessionId cases or default page, call the SSE handler
			// This returns templ.Component that should be wrapped in layout
			content := restaurantsHandlers.HandleRestaurantsPageSSE(c)
			c.HTML(http.StatusOK, "", pages.LayoutPage(models.LayoutTempl{
				Title:   "Travel Planner - Loci",
				Content: content,
				Nav: models.Navigation{
					Items: []models.NavItem{
						{Name: "Dashboard", URL: "/dashboard"},
						{Name: "Discover", URL: "/discover"},
						{Name: "Nearby", URL: "/nearby"},
						{Name: "Itinerary", URL: "/itinerary"},
						{Name: "Chat", URL: "/chat"},
						{Name: "Favorites", URL: "/favorites"},
					},
				},
				ActiveNav: "Itinerary",
				User:      getUserFromContext(c),
			}))
		})

		// Dashboard (authenticated landing)
		protected.GET("/dashboard", func(c *gin.Context) {
			logger.Log.Info("Dashboard accessed", zap.String("user", middleware.GetUserIDFromContext(c)))
			c.HTML(http.StatusOK, "", pages.LayoutPage(models.LayoutTempl{
				Title:   "Dashboard - Loci",
				Content: home.LoggedInDashboard(),
				Nav: models.Navigation{
					Items: []models.NavItem{
						{Name: "Dashboard", URL: "/dashboard"},
						{Name: "Discover", URL: "/discover"},
						{Name: "Nearby", URL: "/nearby"},
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
						{Name: "Nearby", URL: "/nearby"},
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
				Content: favorites.FavoritesPage(favorites.FavoritesPageData{}),
				Nav: models.Navigation{
					Items: []models.NavItem{
						{Name: "Dashboard", URL: "/dashboard"},
						{Name: "Discover", URL: "/discover"},
						{Name: "Nearby", URL: "/nearby"},
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
						{Name: "Nearby", URL: "/nearby"},
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
						{Name: "Nearby", URL: "/nearby"},
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
			userID := middleware.GetUserIDFromContext(c)
			logger.Log.Info("Profile page accessed", zap.String("user", userID))

			// Parse user ID from string to UUID
			userUUID, err := uuid.Parse(userID)
			if err != nil {
				logger.Log.Error("Invalid user ID", zap.String("userID", userID), zap.Error(err))
				c.HTML(http.StatusBadRequest, "", pages.LayoutPage(models.LayoutTempl{
					Title:   "Error - Loci",
					Content: profile.ProfilePage(nil),
					User:    getUserFromContext(c),
				}))
				return
			}

			// Fetch user profile from database
			userProfile, err := userService.GetUserProfile(c.Request.Context(), userUUID)
			if err != nil {
				logger.Log.Error("Failed to fetch user profile", zap.String("userID", userID), zap.Error(err))
				// Still show the page but with nil profile
				userProfile = nil
			}

			c.HTML(http.StatusOK, "", pages.LayoutPage(models.LayoutTempl{
				Title:   "Profile - Loci",
				Content: profile.ProfilePage(userProfile),
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
		protected.GET("/recents", recentsHandlers.HandleRecentsPage)

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

		// Reviews
		protected.GET("/reviews", func(c *gin.Context) {
			logger.Log.Info("Reviews page accessed", zap.String("user", middleware.GetUserIDFromContext(c)))
			c.HTML(http.StatusOK, "", pages.LayoutPage(models.LayoutTempl{
				Title:   "My Reviews - Loci",
				Content: reviews.ReviewsPage(),
				Nav: models.Navigation{
					Items: []models.NavItem{
						{Name: "Dashboard", URL: "/dashboard"},
						{Name: "Profile", URL: "/profile"},
						{Name: "Reviews", URL: "/reviews"},
						{Name: "Settings", URL: "/settings"},
					},
				},
				ActiveNav: "Reviews",
				User:      getUserFromContext(c),
			}))
		})

		// Billing
		protected.GET("/billing", func(c *gin.Context) {
			logger.Log.Info("Billing page accessed", zap.String("user", middleware.GetUserIDFromContext(c)))
			c.HTML(http.StatusOK, "", pages.LayoutPage(models.LayoutTempl{
				Title:   "Billing & Subscription - Loci",
				Content: billing.BillingPage(),
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
	htmxGroup.Use(middleware.AuthMiddleware())
	{
		// Search endpoint (public - no auth required)
		htmxGroup.POST("/search", chatHandlers.HandleSearch)

		// Discover endpoint (requires auth)
		htmxGroup.POST("/discover", middleware.AuthMiddleware(), chatHandlers.HandleDiscover)

		// Chat endpoints
		htmxGroup.POST("/chat/message", chatHandlers.SendMessage)
		htmxGroup.POST("/chat/stream/connect", middleware.OptionalAuthMiddleware(), chatHandlers.HandleChatStreamConnect)

		// SSE streaming endpoints
		htmxGroup.GET("/chat/stream", middleware.OptionalAuthMiddleware(), chatHandlers.ProcessUnifiedChatMessageStream)
		htmxGroup.POST("/chat/stream", middleware.OptionalAuthMiddleware(), chatHandlers.ProcessUnifiedChatMessageStream)

		// Continue chat session endpoint (for adding/removing items to existing sessions)
		htmxGroup.POST("/chat/continue/:sessionID", middleware.OptionalAuthMiddleware(), chatHandlers.ContinueChatSession)

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

		// Nearby endpoints (location-based discovery)
		htmxGroup.GET("/nearby/search", discoverHandlers.GetNearbyPOIs)

		// Results endpoints (LLM-backed searches)
		htmxGroup.POST("/restaurants/search", resultsHandlers.HandleRestaurantSearch)
		htmxGroup.POST("/activities/search", resultsHandlers.HandleActivitySearch)
		htmxGroup.POST("/hotels/search", resultsHandlers.HandleHotelSearch)
		htmxGroup.POST("/itinerary/search", resultsHandlers.HandleItinerarySearch)
		htmxGroup.GET("/itinerary/stream/results", resultsHandlers.HandleItinerarySearch)

		// Nearby endpoints - using PostGIS-based discover common
		// (old nearby common with mock data are deprecated)

		// Itinerary endpoints
		htmxGroup.POST("/itinerary/destination", itineraryHandlers.HandleDestination)
		htmxGroup.POST("/itinerary/chat", itineraryHandlers.HandleChat)
		htmxGroup.POST("/itinerary/add/:id", itineraryHandlers.AddPOI)
		htmxGroup.DELETE("/itinerary/remove/:id", itineraryHandlers.RemovePOI)
		htmxGroup.GET("/itinerary/summary", itineraryHandlers.GetItinerarySummary)
		htmxGroup.GET("/itinerary/stream", chatHandlers.HandleItineraryStream)
		htmxGroup.GET("/itinerary/sse", itineraryHandlers.HandleItinerarySSE)

		// Filter endpoints (HTMX fragments)
		htmxGroup.GET("/api/filter/restaurants", filterHandlers.HandleFilterRestaurants)
		htmxGroup.GET("/api/filter/hotels", filterHandlers.HandleFilterHotels)
		htmxGroup.GET("/api/filter/activities", filterHandlers.HandleFilterActivities)
		htmxGroup.GET("/api/filter/itinerary", filterHandlers.HandleFilterItinerary)
		htmxGroup.GET("/api/filter/:domain/clear", filterHandlers.HandleClearFilters)

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

	// API routes (JSON endpoints for settings UI)
	apiGroup := r.Group("/api")
	{
		// Auth token endpoints (public - for development/testing)
		authTokenHandler := auth2.NewAuthTokenHandler(slog.Default(), jwtConfig)
		authGroup := apiGroup.Group("/auth")
		{
			authGroup.POST("/token", authTokenHandler.GenerateToken)
			authGroup.GET("/verify", middleware.JWTAuthMiddleware(jwtConfig), authTokenHandler.VerifyToken)
			authGroup.GET("/example", authTokenHandler.GetTokenExample)
		}

		// Protected API routes
		protectedAPI := apiGroup.Group("/")
		protectedAPI.Use(middleware.AuthMiddleware())
		{
			// Profiles endpoints
			profilesGroup := protectedAPI.Group("/profiles")
			{
				profilesGroup.GET("", profilesHandlers.GetProfiles)
				profilesGroup.POST("", profilesHandlers.CreateProfile)
				profilesGroup.GET("/:id", profilesHandlers.GetProfile)
				profilesGroup.PUT("/:id", profilesHandlers.UpdateProfile)
				profilesGroup.DELETE("/:id", profilesHandlers.DeleteProfile)
				profilesGroup.PUT("/:id/default", profilesHandlers.SetDefaultProfile)
			}

			// Interests endpoints
			interestsGroup := protectedAPI.Group("/interests")
			{
				interestsGroup.GET("", interestsHandlers.GetInterests)
				interestsGroup.POST("", interestsHandlers.CreateInterest)
				interestsGroup.DELETE("/:id", interestsHandlers.RemoveInterest)
			}

			// Tags endpoints
			tagsGroup := protectedAPI.Group("/tags")
			{
				tagsGroup.GET("", tagsHandlers.GetTags)
				tagsGroup.POST("", tagsHandlers.CreateTag)
				tagsGroup.PUT("/:id", tagsHandlers.UpdateTag)
				tagsGroup.DELETE("/:id", tagsHandlers.DeleteTag)
			}
		}
	}

	// 404 handler - must be last
	r.NoRoute(func(c *gin.Context) {
		logger.Log.Info("404 - Page not found",
			zap.String("path", c.Request.URL.Path),
			zap.String("method", c.Request.Method),
			zap.String("ip", c.ClientIP()),
		)

		user := getUserFromContext(c)
		c.HTML(http.StatusNotFound, "", pages.LayoutPage(models.LayoutTempl{
			Title:   "Page Not Found - Loci",
			Content: pages.NotFoundPage(),
			Nav: models.Navigation{
				Items: []models.NavItem{
					{Name: "Home", URL: "/"},
					{Name: "Discover", URL: "/discover"},
				},
			},
			ActiveNav: "",
			User:      user,
		}))
	})
}
