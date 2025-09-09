package routes

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"regexp"
	"strings"

	"github.com/a-h/templ"
	"github.com/google/uuid"

	"github.com/FACorreiaa/go-templui/app/internal/features"
	"github.com/FACorreiaa/go-templui/app/internal/features/itinerary"
	"github.com/FACorreiaa/go-templui/app/internal/features/results"
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
	"github.com/FACorreiaa/go-templui/app/internal/features/activities"
	"github.com/FACorreiaa/go-templui/app/internal/features/auth"
	"github.com/FACorreiaa/go-templui/app/internal/features/billing"
	"github.com/FACorreiaa/go-templui/app/internal/features/bookmarks"
	"github.com/FACorreiaa/go-templui/app/internal/features/chat"
	"github.com/FACorreiaa/go-templui/app/internal/features/discover"
	"github.com/FACorreiaa/go-templui/app/internal/features/favorites"
	"github.com/FACorreiaa/go-templui/app/internal/features/home"
	"github.com/FACorreiaa/go-templui/app/internal/features/hotels"
	"github.com/FACorreiaa/go-templui/app/internal/features/lists"
	"github.com/FACorreiaa/go-templui/app/internal/features/nearby"
	"github.com/FACorreiaa/go-templui/app/internal/features/pricing"
	"github.com/FACorreiaa/go-templui/app/internal/features/profile"
	"github.com/FACorreiaa/go-templui/app/internal/features/recents"
	"github.com/FACorreiaa/go-templui/app/internal/features/restaurants"
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

// parseItineraryFromResponse parses an AIItineraryResponse from a stored LLM response
func parseItineraryFromResponse(responseText string, logger *slog.Logger) (*models.AIItineraryResponse, error) {
	if responseText == "" {
		return nil, nil
	}

	// Clean the JSON response (similar to existing parsePOIsFromResponse)
	cleanedResponse := cleanJSONResponse(responseText)

	// Try to parse as unified chat response format with "data" wrapper first
	var unifiedResponse struct {
		Data models.AiCityResponse `json:"data"`
	}
	err := json.Unmarshal([]byte(cleanedResponse), &unifiedResponse)
	if err == nil && (unifiedResponse.Data.AIItineraryResponse.ItineraryName != "" || len(unifiedResponse.Data.AIItineraryResponse.PointsOfInterest) > 0) {
		logger.Debug("parseItineraryFromResponse: Parsed as unified chat response")
		return &unifiedResponse.Data.AIItineraryResponse, nil
	}

	// Try to parse as direct AiCityResponse
	var cityResponse models.AiCityResponse
	err = json.Unmarshal([]byte(cleanedResponse), &cityResponse)
	if err == nil && (cityResponse.AIItineraryResponse.ItineraryName != "" || len(cityResponse.AIItineraryResponse.PointsOfInterest) > 0) {
		logger.Debug("parseItineraryFromResponse: Parsed as AiCityResponse")
		return &cityResponse.AIItineraryResponse, nil
	}

	// Try to parse directly as AIItineraryResponse
	var itineraryResponse models.AIItineraryResponse
	err = json.Unmarshal([]byte(cleanedResponse), &itineraryResponse)
	if err == nil && (itineraryResponse.ItineraryName != "" || len(itineraryResponse.PointsOfInterest) > 0) {
		logger.Debug("parseItineraryFromResponse: Parsed as direct AIItineraryResponse")
		return &itineraryResponse, nil
	}

	logger.Debug("parseItineraryFromResponse: Could not parse response as itinerary", "error", err)
	return nil, err
}

// cleanJSONResponse cleans the response text for JSON parsing (reused from existing code)
func cleanJSONResponse(response string) string {
	// Remove any leading/trailing whitespace
	cleaned := strings.TrimSpace(response)

	// Remove code block markers if present
	if strings.HasPrefix(cleaned, "```json") {
		cleaned = strings.TrimPrefix(cleaned, "```json")
	}
	if strings.HasPrefix(cleaned, "```") {
		cleaned = strings.TrimPrefix(cleaned, "```")
	}
	if strings.HasSuffix(cleaned, "```") {
		cleaned = strings.TrimSuffix(cleaned, "```")
	}

	// Remove any non-JSON text before the first {
	startIndex := strings.Index(cleaned, "{")
	if startIndex != -1 {
		cleaned = cleaned[startIndex:]
	}

	// Remove any non-JSON text after the last }
	endIndex := strings.LastIndex(cleaned, "}")
	if endIndex != -1 {
		cleaned = cleaned[:endIndex+1]
	}

	// Clean up any remaining non-JSON prefixes/suffixes
	re := regexp.MustCompile(`^[^{]*({.*})[^}]*$`)
	if matches := re.FindStringSubmatch(cleaned); len(matches) > 1 {
		cleaned = matches[1]
	}

	return strings.TrimSpace(cleaned)
}

//func getDBFromContext(c *gin.Context) *pgxpool.Pool {
//	if db, exists := c.Get("db"); exists {
//		return db.(*pgxpool.Pool)
//	}
//	return nil
//}

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
	itineraryHandlers := handlers2.NewItineraryHandlers()
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

	// Activities (public but enhanced when authenticated)
	r.GET("/activities", middleware.OptionalAuthMiddleware(), func(c *gin.Context) {
		query := c.Query("q")

		logger.Log.Info("Activities page accessed",
			zap.String("ip", c.ClientIP()),
			zap.String("query", query))

		// Create activities page with query context
		var content templ.Component
		if query != "" {
			// Page accessed with a query - this should trigger LLM streaming for activities
			content = activities.ActivitiesPageWithQuery(query)
		} else {
			// Regular activities page
			content = activities.ActivitiesPage()
		}

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

	// Hotels (public but enhanced when authenticated)
	r.GET("/hotels", middleware.OptionalAuthMiddleware(), func(c *gin.Context) {
		query := c.Query("q")

		logger.Log.Info("Hotels page accessed",
			zap.String("ip", c.ClientIP()),
			zap.String("query", query))

		// Create hotel page with query context
		var content templ.Component
		if query != "" {
			// Page accessed with a query - this should trigger LLM streaming for hotels
			content = hotels.HotelsPageWithQuery(query)
		} else {
			// Regular hotel page
			content = hotels.HotelsPage()
		}

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
	r.GET("/restaurants", middleware.OptionalAuthMiddleware(), func(c *gin.Context) {
		query := c.Query("q")

		logger.Log.Info("Restaurants page accessed",
			zap.String("ip", c.ClientIP()),
			zap.String("query", query))

		// Create restaurant page with query context
		var content templ.Component
		if query != "" {
			// Page accessed with a query - this should trigger LLM streaming for restaurants
			content = restaurants.RestaurantsPageWithQuery(query)
		} else {
			// Regular restaurant page
			content = restaurants.RestaurantsPage()
		}

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

		// Itinerary
		protected.GET("/itinerary", func(c *gin.Context) {
			userID := middleware.GetUserIDFromContext(c)
			query := c.Query("q")
			sessionIdParam := c.Query("sessionId")

			logger.Log.Info("Itinerary page accessed",
				zap.String("user", userID),
				zap.String("query", query))
			var content templ.Component

			if sessionIdParam != "" {
				logger.Log.Info("Attempting to load itinerary from cache",
					zap.String("user", userID),
					zap.String("sessionID", sessionIdParam))

				// Try to get the data from the cache
				if itineraryData, found := middleware.ItineraryCache.Get(sessionIdParam); found {
					// DATA FOUND! Render the static results page with the data.
					logger.Log.Info("Itinerary found in cache. Rendering results.")

					// NOTE: Using ItineraryResults, NOT ItineraryResultsStream
					content = results.ItineraryResults(itineraryData, true, true, 5, []string{}) // Adjust params as needed
				} else {
					// Data not found in cache - try to retrieve from database
					logger.Log.Info("Itinerary not found in cache, attempting to load from database", zap.String("sessionID", sessionIdParam))

					// Parse sessionID as UUID
					sessionID, err := uuid.Parse(sessionIdParam)
					if err != nil {
						logger.Log.Warn("Invalid session ID format", zap.String("sessionID", sessionIdParam), zap.Error(err))
						content = results.PageNotFound("Invalid session ID")
					} else {
						// Try to get the latest interaction for this session from database
						ctx := context.Background()
						interaction, err := chatRepo.GetLatestInteractionBySessionID(ctx, sessionID)
						if err != nil || interaction == nil {
							logger.Log.Warn("No interaction found in database for session",
								zap.String("sessionID", sessionIdParam),
								zap.Error(err))
							content = results.PageNotFound("Itinerary session expired. Please create a new itinerary.")
						} else {
							// Try to parse the stored response as itinerary data
							itineraryData, err := parseItineraryFromResponse(interaction.ResponseText, slog.Default())
							if err != nil || itineraryData == nil {
								logger.Log.Warn("Could not parse itinerary from stored response",
									zap.String("sessionID", sessionIdParam),
									zap.Error(err))
								content = results.PageNotFound("Could not load itinerary data. Please create a new itinerary.")
							} else {
								logger.Log.Info("Successfully loaded itinerary from database",
									zap.String("sessionID", sessionIdParam),
									zap.Int("poisCount", len(itineraryData.PointsOfInterest)))

								// Render the results page with the database data
								content = results.ItineraryResults(*itineraryData, true, true, 5, []string{})
							}
						}
					}
				}
			} else {
				// No session ID in the URL. This means the user navigated here directly.
				// Show the default page where they can start creating a new itinerary.
				logger.Log.Info("Direct navigation to /itinerary. Showing default page.")
				content = itinerary.ItineraryPage() // Your original default page
			}

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
