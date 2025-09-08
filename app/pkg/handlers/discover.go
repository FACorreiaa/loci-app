package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/FACorreiaa/go-templui/app/pkg/logger"
	"github.com/FACorreiaa/go-templui/app/pkg/middleware"
)

type DiscoverHandlers struct{}

func NewDiscoverHandlers() *DiscoverHandlers {
	return &DiscoverHandlers{}
}

func (h *DiscoverHandlers) Search(c *gin.Context) {
	query := strings.TrimSpace(c.PostForm("query"))
	location := strings.TrimSpace(c.PostForm("location"))
	user := middleware.GetUserIDFromContext(c)

	logger.Log.Info("Discovery search requested",
		zap.String("query", query),
		zap.String("location", location),
		zap.String("user", user),
		zap.String("ip", c.ClientIP()),
	)

	if query == "" {
		logger.Log.Warn("Empty search query")
		c.HTML(http.StatusBadRequest, "", `<div class="text-red-500 text-center py-8">Please enter a search term</div>`)
		return
	}

	// Simulate search results (in real app, this would query your backend/API)
	results := generateSearchResults(query, location)

	logger.Log.Info("Search completed",
		zap.String("query", query),
		zap.String("location", location),
		zap.Int("results_count", len(results)),
		zap.String("user", user),
	)

	// Return search results HTML
	html := `<div><h2 class="text-lg font-semibold text-foreground mb-4">Search Results for "` + query + `"</h2><div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">`

	for _, result := range results {
		html += `
		<div class="bg-card rounded-xl shadow-sm border hover:shadow-md transition-shadow group cursor-pointer">
			<div class="p-6">
				<div class="flex items-start justify-between mb-4">
					<div class="flex items-center gap-3">
						<span class="text-3xl">` + result.Emoji + `</span>
						<span class="inline-flex items-center px-3 py-1 rounded-full text-sm font-medium bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-300">
							` + result.Category + `
						</span>
					</div>
					<button 
						hx-post="/favorites/add/` + result.ID + `"
						hx-target="this"
						hx-swap="outerHTML"
						class="p-2 text-muted-foreground hover:text-red-500 rounded-lg hover:bg-red-50 dark:hover:bg-red-900/20 transition-colors"
					>
						<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M20.84 4.61a5.5 5.5 0 0 0-7.78 0L12 5.67l-1.06-1.06a5.5 5.5 0 0 0-7.78 7.78l1.06 1.06L12 21.23l7.78-7.78 1.06-1.06a5.5 5.5 0 0 0 0-7.78z"></path>
						</svg>
					</button>
				</div>
				<div class="space-y-3">
					<h3 class="font-semibold text-card-foreground group-hover:text-blue-600 transition-colors">` + result.Title + `</h3>
					<div class="flex items-center gap-2 text-sm text-muted-foreground">
						<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17.657 16.657L13.414 20.9a1.998 1.998 0 01-2.827 0l-4.244-4.243a8 8 0 1111.314 0z"></path>
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 11a3 3 0 11-6 0 3 3 0 016 0z"></path>
						</svg>
						` + result.Location + `
					</div>
					<p class="text-muted-foreground text-sm">` + result.Description + `</p>
				</div>
			</div>
		</div>`
	}

	html += `</div></div>`

	c.HTML(http.StatusOK, "", html)
}

func (h *DiscoverHandlers) GetCategory(c *gin.Context) {
	category := c.Param("category")
	user := middleware.GetUserIDFromContext(c)

	logger.Log.Info("Category search requested",
		zap.String("category", category),
		zap.String("user", user),
		zap.String("ip", c.ClientIP()),
	)

	// Generate category-specific results
	results := generateCategoryResults(category)

	logger.Log.Info("Category search completed",
		zap.String("category", category),
		zap.Int("results_count", len(results)),
		zap.String("user", user),
	)

	// Return category results HTML
	tc := cases.Title(language.English)
	html := `<div><h2 class="text-lg font-semibold text-foreground mb-4">` + tc.String(category) + `s Near You</h2><div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">`

	for _, result := range results {
		html += `
		<div class="bg-card rounded-xl shadow-sm border hover:shadow-md transition-shadow group cursor-pointer">
			<div class="p-6">
				<div class="flex items-start justify-between mb-4">
					<div class="flex items-center gap-3">
						<span class="text-3xl">` + result.Emoji + `</span>
						<span class="inline-flex items-center px-3 py-1 rounded-full text-sm font-medium bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-300">
							` + result.Category + `
						</span>
					</div>
				</div>
				<div class="space-y-3">
					<h3 class="font-semibold text-card-foreground group-hover:text-blue-600 transition-colors">` + result.Title + `</h3>
					<div class="flex items-center gap-2 text-sm text-muted-foreground">
						<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17.657 16.657L13.414 20.9a1.998 1.998 0 01-2.827 0l-4.244-4.243a8 8 0 1111.314 0z"></path>
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 11a3 3 0 11-6 0 3 3 0 016 0z"></path>
						</svg>
						` + result.Location + `
					</div>
					<p class="text-muted-foreground text-sm">` + result.Description + `</p>
				</div>
			</div>
		</div>`
	}

	html += `</div></div>`

	c.HTML(http.StatusOK, "", html)
}

// Helper structs and functions
type SearchResult struct {
	ID          string
	Title       string
	Category    string
	Location    string
	Description string
	Emoji       string
	Rating      string
}

func generateSearchResults(query, location string) []SearchResult {
	// Simulate search results based on query
	return []SearchResult{
		{
			ID:          "1",
			Title:       "Amazing Local Restaurant",
			Category:    "Restaurant",
			Location:    location,
			Description: "Great local spot matching your search for: " + query,
			Emoji:       "🍽️",
			Rating:      "4.5",
		},
		{
			ID:          "2",
			Title:       "Popular Activity",
			Category:    "Activity",
			Location:    location,
			Description: "Top-rated activity related to: " + query,
			Emoji:       "🎯",
			Rating:      "4.7",
		},
	}
}

func generateCategoryResults(category string) []SearchResult {
	categoryMap := map[string]SearchResult{
		"restaurant": {
			ID:          "cat-1",
			Title:       "Top Local Restaurant",
			Category:    "Restaurant",
			Location:    "Your Area",
			Description: "Highly recommended dining experience",
			Emoji:       "🍽️",
			Rating:      "4.6",
		},
		"hotel": {
			ID:          "cat-2",
			Title:       "Boutique Hotel",
			Category:    "Hotel",
			Location:    "City Center",
			Description: "Charming accommodations with great service",
			Emoji:       "🏨",
			Rating:      "4.8",
		},
		"activity": {
			ID:          "cat-3",
			Title:       "Fun Local Activity",
			Category:    "Activity",
			Location:    "Nearby",
			Description: "Exciting things to do in the area",
			Emoji:       "🎯",
			Rating:      "4.5",
		},
	}

	if result, exists := categoryMap[category]; exists {
		return []SearchResult{result}
	}

	return []SearchResult{}
}
