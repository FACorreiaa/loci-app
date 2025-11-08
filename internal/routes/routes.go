package routes

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/pprof"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/FACorreiaa/go-templui/internal/app/domain"
	llmchat "github.com/FACorreiaa/go-templui/internal/app/domain/chat_prompt"
	"github.com/FACorreiaa/go-templui/internal/app/domain/home"
	"github.com/FACorreiaa/go-templui/internal/app/domain/lists"
	pages2 "github.com/FACorreiaa/go-templui/internal/app/domain/pages"
	"github.com/FACorreiaa/go-templui/internal/app/domain/user"
	"github.com/FACorreiaa/go-templui/internal/app/services"

	"github.com/FACorreiaa/go-templui/internal/app/domain/profiles"

	"github.com/FACorreiaa/go-templui/internal/app/domain/reviews"

	"github.com/FACorreiaa/go-templui/internal/app/common"
	"github.com/FACorreiaa/go-templui/internal/app/domain/activities"
	"github.com/FACorreiaa/go-templui/internal/app/domain/billing"
	"github.com/FACorreiaa/go-templui/internal/app/domain/bookmarks"
	cityPkg "github.com/FACorreiaa/go-templui/internal/app/domain/city"
	"github.com/FACorreiaa/go-templui/internal/app/domain/discover"
	"github.com/FACorreiaa/go-templui/internal/app/domain/favorites"
	"github.com/FACorreiaa/go-templui/internal/app/domain/hotels"
	interestsPkg "github.com/FACorreiaa/go-templui/internal/app/domain/interests"
	locationPkg "github.com/FACorreiaa/go-templui/internal/app/domain/location"
	"github.com/FACorreiaa/go-templui/internal/app/domain/nearby"
	"github.com/FACorreiaa/go-templui/internal/app/domain/poi"
	"github.com/FACorreiaa/go-templui/internal/app/domain/recents"
	"github.com/FACorreiaa/go-templui/internal/app/domain/restaurants"
	"github.com/FACorreiaa/go-templui/internal/app/domain/results"
	"github.com/FACorreiaa/go-templui/internal/app/domain/settings"
	streamingfeatures "github.com/FACorreiaa/go-templui/internal/app/domain/streaming"
	tagsPkg "github.com/FACorreiaa/go-templui/internal/app/domain/tags"
	"github.com/FACorreiaa/go-templui/internal/app/models"
	"github.com/FACorreiaa/go-templui/internal/app/renderer"
	"github.com/FACorreiaa/go-templui/internal/pkg/config"
	"github.com/FACorreiaa/go-templui/internal/pkg/middleware"

	generativeAI "github.com/FACorreiaa/go-genai-sdk/lib"

	"github.com/FACorreiaa/go-templui/internal/app/domain/auth"

	"github.com/jackc/pgx/v5/pgxpool"
)

type AppHandlers struct {
	User                *user.Handler
	Home                *home.HomeHandlers
	Auth                *auth.AuthHandlers
	Discover            *discover.DiscoverHandlers
	Favorites           *favorites.FavoritesHandlers
	HotelFavorites      *favorites.HotelFavoritesHandlers
	RestaurantFavorites *favorites.RestaurantFavoritesHandlers
	Bookmarks           *bookmarks.BookmarksHandlers
	Lists               *lists.Handler
	Profiles            *profiles.ProfilesHandler
	Interests           *interestsPkg.InterestsHandler
	Tags                *tagsPkg.TagsHandler
	Chat                *llmchat.ChatHandlers
	Nearby              *nearby.NearbyHandler
	Recents             *recents.RecentsHandlers
	Settings            *settings.SettingsHandlers
	//Billing             *billing.BillingHandlers
	//Reviews             *reviews.ReviewsHandlers
	Activities  *activities.ActivitiesHandlers
	Hotels      *hotels.HotelsHandlers
	Restaurants *restaurants.RestaurantsHandlers
	Itinerary   *interestsPkg.ItineraryHandlers
	Results     *results.ResultsHandlers
	Filter      *common.FilterHandlers
	StaticPages *domain.BaseHandler

	// Add other handlers here
}

func Setup(r *gin.Engine, dbPool *pgxpool.Pool, log *zap.Logger) {
	//r.Use(middleware.AuthMiddleware())
	// Setup custom templ renderer
	slogLog := slog.Default()
	ginHTMLRenderer := r.HTMLRender
	r.HTMLRender = &renderer.HTMLTemplRenderer{FallbackHTMLRenderer: ginHTMLRenderer}

	handlers, err := setupDependencies(dbPool, log, slogLog)
	if err != nil {
		log.Fatal("Failed to setup dependencies", zap.Error(err))
	}
	setupRouter(r, handlers, log)
}

func setupDependencies(dbPool *pgxpool.Pool, log *zap.Logger, slogLog *slog.Logger) (*AppHandlers, error) {
	cfg, err := config.Load()
	if err != nil {
		log.Warn("Failed to load config, using default values", zap.Error(err))
		cfg = &config.Config{}
	}
	baseHandler := domain.NewBaseHandler(log)
	// Initialize repositories and services
	authRepo := auth.NewPostgresAuthRepo(dbPool, log)
	authService := auth.NewAuthService(authRepo, cfg, log)
	// Create repositories
	profilesRepo := profiles.NewPostgresUserRepo(dbPool, log)
	interestsRepo := interestsPkg.NewRepositoryImpl(dbPool, log)
	cityRepo := cityPkg.NewCityRepository(dbPool, log)
	poiRepo := poi.NewRepository(dbPool, log)
	tagsRepo := tagsPkg.NewRepositoryImpl(dbPool, log)
	userRepo := user.NewPostgresUserRepo(dbPool, log)
	listsRepo := lists.NewRepository(dbPool, log)

	// Create services
	profilesService := profiles.NewUserProfilesService(profilesRepo, interestsRepo, tagsRepo, log)
	userService := user.NewUserService(userRepo, log)
	listsService := lists.NewService(listsRepo, log)

	// Create embedding service for POI (can be nil if not using semantic search)
	// Change later GeminiAPIKey env variable
	var embeddingService *generativeAI.EmbeddingService
	if os.Getenv("GEMINI_API_KEY") != "" { // Check env var directly
		ctx := context.Background()                                            // Use a background context for initialization
		embeddingService, err = generativeAI.NewEmbeddingService(ctx, slogLog) // Call with correct args
		if err != nil {
			log.Error("Failed to create embedding service", zap.Error(err))
		}
	}

	// Create chat LLM repository (needed by poiService for LLM logging)
	chatRepo := llmchat.NewRepositoryImpl(dbPool, log)

	poiService := poi.NewServiceImpl(poiRepo, embeddingService, cityRepo, chatRepo, log)

	// Create recents repository and service
	recentsRepo := recents.NewRepository(dbPool, log)
	recentsService := recents.NewService(recentsRepo, log)
	chatService := llmchat.NewLlmInteractiontService(
		interestsRepo,
		profilesRepo,
		profilesService,
		tagsRepo,
		chatRepo,
		cityRepo,
		poiRepo,
		log,
	)
	itineraryService := services.NewItineraryService()
	locationRepo := locationPkg.NewRepository(dbPool)

	handlers := &AppHandlers{
		Home:                home.NewHomeHandlers(baseHandler),
		User:                user.NewHandler(baseHandler, userService),
		Auth:                auth.NewAuthHandlers(authService, log),
		Discover:            discover.NewDiscoverHandlers(baseHandler, poiRepo, chatRepo, chatService, log),
		Favorites:           favorites.NewFavoritesHandlers(poiService, log, baseHandler),
		HotelFavorites:      favorites.NewHotelFavoritesHandlers(poiService, log),
		RestaurantFavorites: favorites.NewRestaurantFavoritesHandlers(poiService, log),
		Bookmarks:           bookmarks.NewBookmarksHandlers(poiService, log),
		Lists:               lists.NewHandler(listsService, log),
		Profiles:            profiles.NewProfilesHandler(profilesService, log),
		Interests:           interestsPkg.NewInterestsHandler(interestsRepo, log),
		Tags:                tagsPkg.NewTagsHandler(tagsRepo, log),
		Chat:                llmchat.NewChatHandlers(chatService, profilesService, chatRepo, log),
		Nearby:              nearby.NewNearbyHandler(log, chatService, locationRepo),
		Recents:             recents.NewRecentsHandlers(recentsService, log),
		Settings:            settings.NewSettingsHandlers(baseHandler, log),
		//Billing:             billing.NewBillingHandlers(baseHandler),
		//Reviews:             reviews.NewReviewsHandlers(baseHandler),
		Activities:  activities.NewActivitiesHandlers(chatRepo, log),
		Hotels:      hotels.NewHotelsHandlers(chatRepo, log),
		Restaurants: restaurants.NewRestaurantsHandlers(chatRepo, log),
		Itinerary:   interestsPkg.NewItineraryHandlers(chatRepo, itineraryService, log),
		Results:     results.NewResultsHandlers(log),
		Filter:      common.NewFilterHandlers(log.Sugar()),
		StaticPages: domain.NewBaseHandler(log),
	}

	return handlers, nil

}

func setupRouter(r *gin.Engine, h *AppHandlers, log *zap.Logger) {
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

	public := r.Group("/")
	public.Use(middleware.OptionalAuthMiddleware())
	{
		public.GET("/", h.Home.ShowHomePage)
		public.GET("/discover", h.Discover.ShowDiscoverPage)
		public.GET("/discover/detail/:sessionId", func(c *gin.Context) {
			renderer.New(c, http.StatusOK, h.Discover.ShowDetail(c)).Render(c.Writer)
		})
		public.GET("/discover/results/:sessionId", func(c *gin.Context) {
			renderer.New(c, http.StatusOK, h.Discover.ShowResults(c)).Render(c.Writer)
		})
		public.GET("/nearby", h.Nearby.Page)
		public.GET("/pricing", h.StaticPages.ShowPricingPage)
		public.GET("/about", h.StaticPages.ShowAboutPage)
	}

	// WebSocket endpoint for real-time nearby POI updates
	// Configure JWT authentication (optional - allows anonymous users)
	jwtSecret := os.Getenv("JWT_SECRET_KEY")
	if jwtSecret == "" {
		jwtSecret = "default-secret-key-change-in-production-min-32-chars"
		log.Warn("JWT_SECRET_KEY not set, using default (INSECURE - set environment variable in production)")
	}

	jwtConfig := auth.JWTConfig{
		SecretKey:       jwtSecret,
		TokenExpiration: 24 * time.Hour,
		Logger:          log,
		Optional:        true, // Allow both authenticated and anonymous users
	}

	// Configure rate limiting for WebSocket connections
	wsRateLimiter := middleware.NewRateLimiter(
		log,
		10,            // Max 10 WebSocket connections
		1*time.Minute, // Per minute
	)

	// Apply middleware: JWT auth (optional) + rate limiting
	r.GET("/ws/nearby",
		auth.JWTAuthMiddleware(jwtConfig),
		middleware.WebSocketRateLimitMiddleware(wsRateLimiter),
		h.Nearby.HandleWebSocket,
	)

	// Auth routes
	authGroup := r.Group("/auth")
	{
		authGroup.GET("/signin", func(c *gin.Context) {
			log.Info("Sign in page accessed")
			c.HTML(http.StatusOK, "", pages2.LayoutPage(models.LayoutTempl{
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
			log.Info("Sign up page accessed")
			c.HTML(http.StatusOK, "", pages2.LayoutPage(models.LayoutTempl{
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
			log.Info("Forgot password page accessed")
			c.HTML(http.StatusOK, "", pages2.LayoutPage(models.LayoutTempl{
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

		authGroup.POST("/signin", gin.WrapF(h.Auth.LoginHandler))
		authGroup.POST("/signup", gin.WrapF(h.Auth.RegisterHandler))
		authGroup.POST("/logout", gin.WrapF(h.Auth.LogoutHandler))
		authGroup.POST("/forgot-password", gin.WrapF(h.Auth.ForgotPasswordHandler))
		authGroup.POST("/change-password", gin.WrapF(h.Auth.ChangePasswordHandler))
		authGroup.POST("/check-username", gin.WrapF(h.Auth.CheckUsernameHandler))
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
				c.HTML(http.StatusOK, "", pages2.LayoutPage(models.LayoutTempl{
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
					User:      middleware.GetUserFromContext(c),
				}))
				return
			}

			// For sessionId cases or default page, call the SSE handler
			// This returns templ.Component that should be wrapped in layout
			content := h.Itinerary.HandleItineraryPageSSE(c)
			c.HTML(http.StatusOK, "", pages2.LayoutPage(models.LayoutTempl{
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
				User:      middleware.GetUserFromContext(c),
			}))
		})

		//Activities (public but enhanced when authenticated)
		protected.GET("/activities", func(c *gin.Context) {
			content := h.Activities.HandleActivitiesPage(c)
			c.HTML(http.StatusOK, "", pages2.LayoutPage(models.LayoutTempl{
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
				User:      middleware.GetUserFromContext(c),
			}))
		})

		//Hotels (public but enhanced when authenticated)
		protected.GET("/hotels", func(c *gin.Context) {
			content := h.Hotels.HandleHotelsPage(c)
			c.HTML(http.StatusOK, "", pages2.LayoutPage(models.LayoutTempl{
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
				User:      middleware.GetUserFromContext(c),
			}))
		})

		protected.GET("/restaurants", func(c *gin.Context) {
			query := c.Query("q")
			sessionIDParam := c.Query("sessionId")

			// If there's a query but no sessionID, start new streaming
			if query != "" && sessionIDParam == "" {
				// Return the streaming trigger page wrapped in layout
				content := streamingfeatures.StreamingTriggerPage(query, "restaurants")
				c.HTML(http.StatusOK, "", pages2.LayoutPage(models.LayoutTempl{
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
					User:      middleware.GetUserFromContext(c),
				}))
				return
			}

			// For sessionId cases or default page, call the SSE handler
			// This returns templ.Component that should be wrapped in layout
			content := h.Restaurants.HandleRestaurantsPageSSE(c)
			c.HTML(http.StatusOK, "", pages2.LayoutPage(models.LayoutTempl{
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
				User:      middleware.GetUserFromContext(c),
			}))
		})

		// Dashboard (authenticated landing)
		protected.GET("/dashboard", func(c *gin.Context) {
			log.Info("Dashboard accessed", zap.String("user", middleware.GetUserIDFromContext(c)))
			c.HTML(http.StatusOK, "", pages2.LayoutPage(models.LayoutTempl{
				Title:   "Dashboard - Loci",
				Content: pages2.LoggedInDashboard(),
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
				User:      middleware.GetUserFromContext(c),
			}))
		})

		// Chat
		protected.GET("/chat", func(c *gin.Context) {
			log.Info("Chat page accessed", zap.String("user", middleware.GetUserIDFromContext(c)))
			c.HTML(http.StatusOK, "", pages2.LayoutPage(models.LayoutTempl{
				Title:   "AI Chat - Loci",
				Content: llmchat.ChatPage(),
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
				User:      middleware.GetUserFromContext(c),
			}))
		})

		//protected.GET("/favorites", func(c *gin.Context) {
		//	log.Info("Favorites page accessed", zap.String("user", middleware.GetUserIDFromContext(c)))
		//	c.HTML(http.StatusOK, "", pages.LayoutPage(models.LayoutTempl{
		//		Title:   "Favorites - Loci",
		//		Content: favorites.FavoritesPage(),
		//		Nav: models.Navigation{
		//			Items: []models.NavItem{
		//				{Name: "Dashboard", URL: "/dashboard"},
		//				{Name: "Discover", URL: "/discover"},
		//				{Name: "Nearby", URL: "/nearby"},
		//				{Name: "Chat", URL: "/chat"},
		//				{Name: "Favorites", URL: "/favorites"},
		//			},
		//		},
		//		ActiveNav: "Chat",
		//		User:      middleware.GetUserFromContext(c),
		//	}))
		//})

		// Favorites - use the handler that fetches and displays actual data
		protected.GET("/favorites", h.Favorites.ListFavorites)

		// Bookmarks - use the handler that fetches and displays actual data
		protected.GET("/bookmarks", h.Bookmarks.ListBookmarks)

		// Lists
		protected.GET("/lists", h.Lists.ShowListsPage)
		protected.GET("/lists/saved", h.Lists.ShowSavedListsPage)

		// Lists modal and actions
		protected.GET("/lists/new", h.Lists.ShowCreateModal)
		protected.POST("/lists/create", h.Lists.CreateList)
		protected.GET("/lists/select", h.Lists.ShowAddToListModal)
		protected.GET("/lists/:id", h.Lists.ShowListDetail)
		protected.POST("/lists/:id/items", h.Lists.AddItemToList)
		protected.DELETE("/lists/:id/items/:itemId", h.Lists.RemoveListItem)
		protected.GET("/lists/:id/edit", h.Lists.ShowEditModal)
		protected.PUT("/lists/:id", h.Lists.UpdateList)
		protected.GET("/lists/:id/delete", h.Lists.ShowDeleteModal)
		protected.DELETE("/lists/:id", h.Lists.DeleteList)
		protected.POST("/lists/:id/save", h.Lists.SaveListAction)
		protected.DELETE("/lists/:id/unsave", h.Lists.UnsaveListAction)

		protected.GET("/profile", h.User.ShowProfilePage)
		// Recents
		protected.GET("/recents", h.Recents.HandleRecentsPage)

		// Settings
		protected.GET("/settings", h.Settings.ShowSettingsPage)

		// Reviews
		protected.GET("/reviews", func(c *gin.Context) {
			log.Info("Reviews page accessed", zap.String("user", middleware.GetUserIDFromContext(c)))
			c.HTML(http.StatusOK, "", pages2.LayoutPage(models.LayoutTempl{
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
				User:      middleware.GetUserFromContext(c),
			}))
		})

		// Billing
		protected.GET("/billing", func(c *gin.Context) {
			log.Info("Billing page accessed", zap.String("user", middleware.GetUserIDFromContext(c)))
			c.HTML(http.StatusOK, "", pages2.LayoutPage(models.LayoutTempl{
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
				User:      middleware.GetUserFromContext(c),
			}))
		})
	}

	// HTMX API routes
	htmxGroup := r.Group("/")
	htmxGroup.Use(middleware.AuthMiddleware())
	{
		// Search endpoint (public - no auth required)
		htmxGroup.POST("/search", h.Chat.HandleSearch)

		// Discover endpoint (requires auth)
		htmxGroup.POST("/discover", middleware.AuthMiddleware(), h.Chat.HandleDiscover)

		// Chat endpoints
		htmxGroup.POST("/chat/message", h.Chat.SendMessage)
		htmxGroup.POST("/chat/stream/connect", middleware.OptionalAuthMiddleware(), h.Chat.HandleChatStreamConnect)

		// SSE streaming endpoints
		htmxGroup.GET("/chat/stream", middleware.OptionalAuthMiddleware(), h.Chat.ProcessUnifiedChatMessageStream)
		htmxGroup.POST("/chat/stream", middleware.OptionalAuthMiddleware(), h.Chat.ProcessUnifiedChatMessageStream)

		// Continue chat session endpoint (for adding/removing items to existing sessions)
		htmxGroup.POST("/chat/continue/:sessionID", middleware.OptionalAuthMiddleware(), h.Chat.ContinueChatSession)

		// Favorites endpoints
		htmxGroup.POST("/favorites/add/:id", h.Favorites.AddFavorite)
		htmxGroup.DELETE("/favorites/:id", h.Favorites.RemoveFavorite)
		htmxGroup.POST("/favorites/search", h.Favorites.SearchFavorites)

		// Hotel favorites endpoints
		htmxGroup.POST("/favorites/hotels/:id", h.HotelFavorites.AddHotelFavorite)
		htmxGroup.DELETE("/favorites/hotels/:id", h.HotelFavorites.RemoveHotelFavorite)

		// Restaurant favorites endpoints
		htmxGroup.POST("/favorites/restaurants/:id", h.RestaurantFavorites.AddRestaurantFavorite)
		htmxGroup.DELETE("/favorites/restaurants/:id", h.RestaurantFavorites.RemoveRestaurantFavorite)

		// Bookmarks endpoints
		htmxGroup.POST("/bookmarks/add/:id", h.Bookmarks.AddBookmark)
		htmxGroup.DELETE("/bookmarks/:id", h.Bookmarks.RemoveBookmark)
		// htmxGroup.POST("/bookmarks/search", bookmarksHandlers.SearchBookmarks) // TODO: Implement SearchBookmarks

		// Discover endpoints
		htmxGroup.POST("/discover/search", h.Discover.Search)
		htmxGroup.GET("/discover/recent", h.Discover.GetRecentDiscoveries)
		htmxGroup.GET("/discover/category/:category", h.Discover.GetCategory)

		// Nearby endpoints (location-based discovery)
		htmxGroup.GET("/nearby/search", h.Discover.GetNearbyPOIs)

		// Results endpoints (LLM-backed searches)
		htmxGroup.POST("/restaurants/search", h.Results.HandleRestaurantSearch)
		htmxGroup.POST("/activities/search", h.Results.HandleActivitySearch)
		htmxGroup.POST("/hotels/search", h.Results.HandleHotelSearch)
		htmxGroup.POST("/itinerary/search", h.Results.HandleItinerarySearch)
		htmxGroup.GET("/itinerary/stream/results", h.Results.HandleItinerarySearch)

		// Nearby endpoints - using PostGIS-based discover common
		// (old nearby common with mock data are deprecated)

		// Itinerary endpoints
		htmxGroup.POST("/itinerary/destination", h.Itinerary.HandleDestination)
		htmxGroup.POST("/itinerary/chat", h.Itinerary.HandleChat)
		htmxGroup.POST("/itinerary/add/:id", h.Itinerary.AddPOI)
		htmxGroup.DELETE("/itinerary/remove/:id", h.Itinerary.RemovePOI)
		htmxGroup.GET("/itinerary/summary", h.Itinerary.GetItinerarySummary)
		htmxGroup.GET("/itinerary/stream", h.Chat.HandleItineraryStream)
		htmxGroup.GET("/itinerary/sse", h.Itinerary.HandleItinerarySSE)

		// Filter endpoints (HTMX fragments)
		htmxGroup.GET("/api/filter/restaurants", h.Filter.HandleFilterRestaurants)
		htmxGroup.GET("/api/filter/hotels", h.Filter.HandleFilterHotels)
		htmxGroup.GET("/api/filter/activities", h.Filter.HandleFilterActivities)
		htmxGroup.GET("/api/filter/itinerary", h.Filter.HandleFilterItinerary)
		htmxGroup.GET("/api/filter/:domain/clear", h.Filter.HandleClearFilters)

		// Settings endpoints (protected)
		settingsGroup := htmxGroup.Group("/settings")
		settingsGroup.Use(middleware.AuthMiddleware())
		{
			settingsGroup.POST("/profile", h.Settings.UpdateProfile)
			settingsGroup.POST("/preferences", h.Settings.UpdatePreferences)
			settingsGroup.POST("/notifications", h.Settings.UpdateNotifications)
			settingsGroup.DELETE("/account", h.Settings.DeleteAccount)
			settingsGroup.POST("/export", h.Settings.ExportData)
		}
	}

	// API routes (JSON endpoints for settings UI)
	apiGroup := r.Group("/api")
	{
		// Auth token endpoints (public - for development/testing)
		authTokenHandler := auth.NewAuthTokenHandler(log, jwtConfig)
		authGroup := apiGroup.Group("/auth")
		{
			authGroup.POST("/token", authTokenHandler.GenerateToken)
			authGroup.GET("/verify", auth.JWTAuthMiddleware(jwtConfig), authTokenHandler.VerifyToken)
			authGroup.GET("/example", authTokenHandler.GetTokenExample)
		}

		// Protected API routes
		protectedAPI := apiGroup.Group("/")
		protectedAPI.Use(middleware.AuthMiddleware())
		{
			// Profiles endpoints
			profilesGroup := protectedAPI.Group("/profiles")
			{
				profilesGroup.GET("", h.Profiles.GetProfiles)
				profilesGroup.POST("", h.Profiles.CreateProfile)
				profilesGroup.GET("/:id", h.Profiles.GetProfile)
				profilesGroup.PUT("/:id", h.Profiles.UpdateProfile)
				profilesGroup.DELETE("/:id", h.Profiles.DeleteProfile)
				profilesGroup.PUT("/:id/default", h.Profiles.SetDefaultProfile)
			}

			// Interests endpoints
			interestsGroup := protectedAPI.Group("/interests")
			{
				interestsGroup.GET("", h.Interests.GetInterests)
				interestsGroup.POST("", h.Interests.CreateInterest)
				interestsGroup.DELETE("/:id", h.Interests.RemoveInterest)
			}

			// Tags endpoints
			tagsGroup := protectedAPI.Group("/tags")
			{
				tagsGroup.GET("", h.Tags.GetTags)
				tagsGroup.POST("", h.Tags.CreateTag)
				tagsGroup.PUT("/:id", h.Tags.UpdateTag)
				tagsGroup.DELETE("/:id", h.Tags.DeleteTag)
			}
		}
	}

	// 404 handler - must be last
	r.NoRoute(func(c *gin.Context) {
		log.Info("404 - Page not found",
			zap.String("path", c.Request.URL.Path),
			zap.String("method", c.Request.Method),
			zap.String("ip", c.ClientIP()),
		)

		user := middleware.GetUserFromContext(c)
		c.HTML(http.StatusNotFound, "", pages2.LayoutPage(models.LayoutTempl{
			Title:   "Page Not Found - Loci",
			Content: pages2.NotFoundPage(),
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
