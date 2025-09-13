package routes

import (
	"log/slog"
	"net/http"

	"github.com/a-h/templ"

	"github.com/FACorreiaa/go-templui/app/internal/features"
	"github.com/FACorreiaa/go-templui/app/internal/models"
	"github.com/FACorreiaa/go-templui/app/internal/renderer"
	"github.com/FACorreiaa/go-templui/app/pkg/config"
	authPkg "github.com/FACorreiaa/go-templui/app/pkg/domain/auth"
	llmchat "github.com/FACorreiaa/go-templui/app/pkg/domain/chat_prompt"
	cityPkg "github.com/FACorreiaa/go-templui/app/pkg/domain/city"
	interestsPkg "github.com/FACorreiaa/go-templui/app/pkg/domain/interests"
	poiPkg "github.com/FACorreiaa/go-templui/app/pkg/domain/poi"
	profilesPkg "github.com/FACorreiaa/go-templui/app/pkg/domain/profiles"
	tagsPkg "github.com/FACorreiaa/go-templui/app/pkg/domain/tags"
	handlers2 "github.com/FACorreiaa/go-templui/app/pkg/handlers"
	"github.com/FACorreiaa/go-templui/app/pkg/logger"
	"github.com/FACorreiaa/go-templui/app/pkg/middleware"

	"github.com/FACorreiaa/go-templui/app/internal/features/about"
	"github.com/FACorreiaa/go-templui/app/internal/features/auth"
	"github.com/FACorreiaa/go-templui/app/internal/features/billing"
	"github.com/FACorreiaa/go-templui/app/internal/features/bookmarks"
	"github.com/FACorreiaa/go-templui/app/internal/features/chat"
	"github.com/FACorreiaa/go-templui/app/internal/features/discover"
	"github.com/FACorreiaa/go-templui/app/internal/features/favorites"
	"github.com/FACorreiaa/go-templui/app/internal/features/home"
	"github.com/FACorreiaa/go-templui/app/internal/features/lists"
	"github.com/FACorreiaa/go-templui/app/internal/features/nearby"
	"github.com/FACorreiaa/go-templui/app/internal/features/pricing"
	streamingfeatures "github.com/FACorreiaa/go-templui/app/internal/features/streaming"
	"github.com/FACorreiaa/go-templui/app/internal/features/profile"
	"github.com/FACorreiaa/go-templui/app/internal/features/recents"
	"github.com/FACorreiaa/go-templui/app/internal/features/reviews"
	"github.com/FACorreiaa/go-templui/app/internal/features/settings"
	"github.com/FACorreiaa/go-templui/app/internal/pages"

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

func Setup(r *gin.Engine, dbPool *pgxpool.Pool) {
	//r.Use(middleware.AuthMiddleware())
	// Setup custom templ renderer
	ginHTMLRenderer := r.HTMLRender
	r.HTMLRender = &renderer.HTMLTemplRenderer{FallbackHTMLRenderer: ginHTMLRenderer}

	// Assets
	r.Static("/assets", "./assets")
	r.Static("/static", "./assets/static")
	r.StaticFile("/sw.js", "./assets/static/sw.js")
	r.StaticFile("/manifest.json", "./assets/static/manifest.json")

	// Initialize handlers
	cfg, err := config.Load()
	if err != nil {
		// Use default config if loading fails
		cfg = &config.Config{}
	}

	// Initialize repositories and services
	authRepo := authPkg.NewPostgresAuthRepo(dbPool, slog.Default())

	// Create repositories
	profilesRepo := profilesPkg.NewPostgresUserRepo(dbPool, slog.Default())
	interestsRepo := interestsPkg.NewRepositoryImpl(dbPool, slog.Default())
	cityRepo := cityPkg.NewCityRepository(dbPool, slog.Default())
	poiRepo := poiPkg.NewRepository(dbPool, slog.Default())
	tagsRepo := tagsPkg.NewRepositoryImpl(dbPool, slog.Default())

	// Create services
	profilesService := profilesPkg.NewUserProfilesService(profilesRepo, interestsRepo, tagsRepo, slog.Default())

	// Create handlers
	authHandlers := authPkg.NewAuthHandlers(authRepo, cfg, slog.Default())
	profilesHandlers := handlers2.NewProfilesHandler(profilesService)
	interestsHandlers := handlers2.NewInterestsHandler(interestsRepo)
	tagsHandlers := handlers2.NewTagsHandler(tagsRepo)
	// Create chat LLM service
	chatRepo := llmchat.NewRepositoryImpl(dbPool, slog.Default())
	chatService := llmchat.NewLlmInteractiontService(
		interestsRepo,
		profilesRepo,
		profilesService,
		tagsRepo,
		chatRepo,
		cityRepo,
		poiRepo,
		slog.Default(),
	)
	chatHandlers := handlers2.NewChatHandlers(chatService, profilesService, chatRepo)
	favoritesHandlers := handlers2.NewFavoritesHandlers()
	bookmarksHandlers := handlers2.NewBookmarksHandlers()
	discoverHandlers := handlers2.NewDiscoverHandlers()
	nearbyHandlers := handlers2.NewNearbyHandlers()
	itineraryHandlers := handlers2.NewItineraryHandlers(chatRepo)
	activitiesHandlers := handlers2.NewActivitiesHandlers(chatRepo)
	hotelsHandlers := handlers2.NewHotelsHandlers(chatRepo)
	restaurantsHandlers := handlers2.NewRestaurantsHandlers(chatRepo)
	settingsHandlers := handlers2.NewSettingsHandlers()
	resultsHandlers := handlers2.NewResultsHandlers()

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
		// Itinerary
		// protected.GET("/itinerary", func(c *gin.Context) {
		// 	content := itineraryHandlers.HandleItineraryPage(c)
		// 	c.HTML(http.StatusOK, "", pages.LayoutPage(models.LayoutTempl{
		// 		Title:   "Travel Planner - Loci",
		// 		Content: content,
		// 		Nav: models.Navigation{
		// 			Items: []models.NavItem{
		// 				{Name: "Dashboard", URL: "/dashboard"},
		// 				{Name: "Discover", URL: "/discover"},
		// 				{Name: "Nearby", URL: "/nearby"},
		// 				{Name: "Itinerary", URL: "/itinerary"},
		// 				{Name: "Chat", URL: "/chat"},
		// 				{Name: "Favorites", URL: "/favorites"},
		// 			},
		// 		},
		// 		ActiveNav: "Itinerary",
		// 		User:      getUserFromContext(c),
		// 	}))
		// })

		// Itinerary SSE
		protected.GET("/itinerary", func(c *gin.Context) {
			query := c.Query("q")
			sessionIdParam := c.Query("sessionId")
			
			// If there's a query but no sessionId, start new streaming
			if query != "" && sessionIdParam == "" {
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
			
			// For sessionId cases or default page, call the SSE handler directly
			// This returns complete HTML pages, not components for layout
			content := itineraryHandlers.HandleItineraryPageSSE(c)
			if sessionIdParam == "" {
				// Default empty page - wrap in layout
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
			} else {
				// Cached results - render directly without layout wrapper
				c.HTML(http.StatusOK, "", content)
			}
		})

		//Activities (public but enhanced when authenticated)
		protected.GET("/activities", func(c *gin.Context) {
			content := activitiesHandlers.HandleActivitiesPageSSE(c)
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
			content := hotelsHandlers.HandleHotelsPageSSE(c)
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
		protected.GET("/restaurants", func(c *gin.Context) {
			content := restaurantsHandlers.HandleRestaurantsPageSSE(c)
			c.HTML(http.StatusOK, "", pages.LayoutPage(models.LayoutTempl{
				Title:   "Restaurants - Loci",
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
				ActiveNav: "Restaurants",
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

		// Nearby
		protected.GET("/nearby", func(c *gin.Context) {
			logger.Log.Info("Nearby page accessed", zap.String("user", middleware.GetUserIDFromContext(c)))
			c.HTML(http.StatusOK, "", pages.LayoutPage(models.LayoutTempl{
				Title:   "Nearby Places - Loci",
				Content: nearby.NearbyPage(),
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
				ActiveNav: "Nearby",
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

		// Results endpoints (LLM-backed searches)
		htmxGroup.POST("/restaurants/search", resultsHandlers.HandleRestaurantSearch)
		htmxGroup.POST("/activities/search", resultsHandlers.HandleActivitySearch)
		htmxGroup.POST("/hotels/search", resultsHandlers.HandleHotelSearch)
		htmxGroup.POST("/itinerary/search", resultsHandlers.HandleItinerarySearch)
		htmxGroup.GET("/itinerary/stream/results", resultsHandlers.HandleItinerarySearch)

		// Nearby endpoints
		htmxGroup.POST("/nearby/search", nearbyHandlers.SearchPOIs)
		htmxGroup.POST("/nearby/category/:category", nearbyHandlers.GetPOIsByCategory)
		htmxGroup.POST("/nearby/filter", nearbyHandlers.FilterPOIs)
		htmxGroup.GET("/nearby/map", nearbyHandlers.GetMapData)

		// Itinerary endpoints
		htmxGroup.POST("/itinerary/destination", itineraryHandlers.HandleDestination)
		htmxGroup.POST("/itinerary/chat", itineraryHandlers.HandleChat)
		htmxGroup.POST("/itinerary/add/:id", itineraryHandlers.AddPOI)
		htmxGroup.DELETE("/itinerary/remove/:id", itineraryHandlers.RemovePOI)
		htmxGroup.GET("/itinerary/summary", itineraryHandlers.GetItinerarySummary)
		htmxGroup.GET("/itinerary/stream", chatHandlers.HandleItineraryStream)
		htmxGroup.GET("/itinerary/sse", itineraryHandlers.HandleItinerarySSE)

		// Restaurants SSE endpoints
		htmxGroup.GET("/restaurants/sse", restaurantsHandlers.HandleRestaurantsSSE)

		// Activities SSE endpoints
		htmxGroup.GET("/activities/sse", activitiesHandlers.HandleActivitiesSSE)

		// Hotels SSE endpoints
		htmxGroup.GET("/hotels/sse", hotelsHandlers.HandleHotelsSSE)

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
	apiGroup.Use(middleware.AuthMiddleware())
	{
		// Profiles endpoints
		profilesGroup := apiGroup.Group("/profiles")
		{
			profilesGroup.GET("", profilesHandlers.GetProfiles)
			profilesGroup.POST("", profilesHandlers.CreateProfile)
			profilesGroup.GET("/:id", profilesHandlers.GetProfile)
			profilesGroup.PUT("/:id", profilesHandlers.UpdateProfile)
			profilesGroup.DELETE("/:id", profilesHandlers.DeleteProfile)
			profilesGroup.PUT("/:id/default", profilesHandlers.SetDefaultProfile)
		}

		// Interests endpoints
		interestsGroup := apiGroup.Group("/interests")
		{
			interestsGroup.GET("", interestsHandlers.GetInterests)
			interestsGroup.POST("", interestsHandlers.CreateInterest)
			interestsGroup.DELETE("/:id", interestsHandlers.RemoveInterest)
		}

		// Tags endpoints
		tagsGroup := apiGroup.Group("/tags")
		{
			tagsGroup.GET("", tagsHandlers.GetTags)
			tagsGroup.POST("", tagsHandlers.CreateTag)
			tagsGroup.PUT("/:id", tagsHandlers.UpdateTag)
			tagsGroup.DELETE("/:id", tagsHandlers.DeleteTag)
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
