package handlers

import (
	"log/slog"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/FACorreiaa/go-templui/app/internal/features/results"
	"github.com/FACorreiaa/go-templui/app/internal/models"
	"github.com/FACorreiaa/go-templui/app/pkg/logger"
	"github.com/FACorreiaa/go-templui/app/pkg/middleware"
)

type FilterHandlers struct {
	logger *slog.Logger
}

func NewFilterHandlers(logger *slog.Logger) *FilterHandlers {
	return &FilterHandlers{
		logger: logger,
	}
}

// HandleFilterRestaurants filters restaurants based on form parameters
func (h *FilterHandlers) HandleFilterRestaurants(c *gin.Context) {
	cacheKey := c.Query("cacheKey")
	categories := c.QueryArray("categories[]")
	priceRanges := c.QueryArray("priceRange[]")
	ratingStr := c.Query("rating")
	features := c.QueryArray("features[]")

	logger.Log.Info("Filtering restaurants",
		zap.String("cacheKey", cacheKey),
		zap.Strings("categories", categories),
		zap.Strings("priceRanges", priceRanges),
		zap.String("rating", ratingStr))

	// Get restaurants from cache
	restaurantsData, found := middleware.RestaurantsCache.Get(cacheKey)
	if !found {
		logger.Log.Warn("No restaurants found in cache for filtering", zap.String("cacheKey", cacheKey))
		c.HTML(200, "", results.EmptyFilterResults("restaurants"))
		return
	}

	// Apply filters
	filtered := filterRestaurantsByCriteria(restaurantsData, categories, priceRanges, ratingStr, features)

	logger.Log.Info("Restaurants filtered",
		zap.Int("original", len(restaurantsData)),
		zap.Int("filtered", len(filtered)))

	// Render filtered restaurant cards
	c.HTML(200, "", results.RestaurantsList(filtered))
}

// HandleFilterHotels filters hotels based on form parameters
func (h *FilterHandlers) HandleFilterHotels(c *gin.Context) {
	cacheKey := c.Query("cacheKey")
	categories := c.QueryArray("categories[]")
	priceRanges := c.QueryArray("priceRange[]")
	ratingStr := c.Query("rating")
	features := c.QueryArray("features[]")

	logger.Log.Info("Filtering hotels",
		zap.String("cacheKey", cacheKey),
		zap.Strings("categories", categories),
		zap.Strings("priceRanges", priceRanges),
		zap.String("rating", ratingStr))

	// Get hotels from cache
	hotelsData, found := middleware.HotelsCache.Get(cacheKey)
	if !found {
		logger.Log.Warn("No hotels found in cache for filtering", zap.String("cacheKey", cacheKey))
		c.HTML(200, "", results.EmptyFilterResults("hotels"))
		return
	}

	// Apply filters
	filtered := filterHotelsByCriteria(hotelsData, categories, priceRanges, ratingStr, features)

	logger.Log.Info("Hotels filtered",
		zap.Int("original", len(hotelsData)),
		zap.Int("filtered", len(filtered)))

	// Render filtered hotel cards
	c.HTML(200, "", results.HotelsList(filtered))
}

// HandleFilterActivities filters activities based on form parameters
func (h *FilterHandlers) HandleFilterActivities(c *gin.Context) {
	cacheKey := c.Query("cacheKey")
	categories := c.QueryArray("categories[]")
	priceRanges := c.QueryArray("priceRange[]")
	ratingStr := c.Query("rating")
	features := c.QueryArray("features[]")

	logger.Log.Info("Filtering activities",
		zap.String("cacheKey", cacheKey),
		zap.Strings("categories", categories),
		zap.Strings("priceRanges", priceRanges),
		zap.String("rating", ratingStr))

	// Get activities from cache
	activitiesData, found := middleware.ActivitiesCache.Get(cacheKey)
	if !found {
		logger.Log.Warn("No activities found in cache for filtering", zap.String("cacheKey", cacheKey))
		c.HTML(200, "", results.EmptyFilterResults("activities"))
		return
	}

	// Apply filters
	filtered := filterActivitiesByCriteria(activitiesData, categories, priceRanges, ratingStr, features)

	logger.Log.Info("Activities filtered",
		zap.Int("original", len(activitiesData)),
		zap.Int("filtered", len(filtered)))

	// Render filtered activity cards
	c.HTML(200, "", results.ActivitiesList(filtered))
}

// HandleFilterItinerary filters itinerary POIs based on form parameters
func (h *FilterHandlers) HandleFilterItinerary(c *gin.Context) {
	cacheKey := c.Query("cacheKey")
	categories := c.QueryArray("categories[]")
	priceRanges := c.QueryArray("priceRange[]")
	ratingStr := c.Query("rating")

	logger.Log.Info("Filtering itinerary",
		zap.String("cacheKey", cacheKey),
		zap.Strings("categories", categories),
		zap.Strings("priceRanges", priceRanges),
		zap.String("rating", ratingStr))

	// Get complete itinerary from cache
	completeData, found := middleware.CompleteItineraryCache.Get(cacheKey)
	if !found {
		logger.Log.Warn("No itinerary found in cache for filtering", zap.String("cacheKey", cacheKey))
		c.HTML(200, "", results.EmptyFilterResults("itinerary"))
		return
	}

	// Apply filters to POIs
	filtered := filterPOIsByCriteria(completeData.PointsOfInterest, categories, priceRanges, ratingStr)

	logger.Log.Info("Itinerary POIs filtered",
		zap.Int("original", len(completeData.PointsOfInterest)),
		zap.Int("filtered", len(filtered)))

	// Render filtered POI cards
	c.HTML(200, "", results.POIsList(filtered))
}

// HandleClearFilters clears all filters and returns full results
func (h *FilterHandlers) HandleClearFilters(c *gin.Context) {
	domain := c.Param("domain")
	cacheKey := c.Query("cacheKey")

	logger.Log.Info("Clearing filters", zap.String("domain", domain), zap.String("cacheKey", cacheKey))

	switch domain {
	case "restaurants":
		if restaurantsData, found := middleware.RestaurantsCache.Get(cacheKey); found {
			c.HTML(200, "", results.RestaurantsList(restaurantsData))
			return
		}
	case "hotels":
		if hotelsData, found := middleware.HotelsCache.Get(cacheKey); found {
			c.HTML(200, "", results.HotelsList(hotelsData))
			return
		}
	case "activities":
		if activitiesData, found := middleware.ActivitiesCache.Get(cacheKey); found {
			c.HTML(200, "", results.ActivitiesList(activitiesData))
			return
		}
	case "itinerary":
		if completeData, found := middleware.CompleteItineraryCache.Get(cacheKey); found {
			c.HTML(200, "", results.POIsList(completeData.PointsOfInterest))
			return
		}
	}

	c.HTML(200, "", results.EmptyFilterResults(domain))
}

// Filter helper functions

func filterRestaurantsByCriteria(restaurants []models.RestaurantDetailedInfo, categories []string, priceRanges []string, ratingStr string, features []string) []models.RestaurantDetailedInfo {
	var filtered []models.RestaurantDetailedInfo

	minRating := parseRating(ratingStr)

	for _, restaurant := range restaurants {
		// Category filter
		if len(categories) > 0 && !matchesCategory(restaurant.Category, restaurant.CuisineType, restaurant.Tags, categories) {
			continue
		}

		// Price range filter
		if len(priceRanges) > 0 && !matchesPriceRange(getPriceLevel(restaurant.PriceLevel), priceRanges) {
			continue
		}

		// Rating filter
		if minRating > 0 && restaurant.Rating < minRating {
			continue
		}

		// Feature filter
		if len(features) > 0 && !matchesFeatures(restaurant.Tags, restaurant.Description, features) {
			continue
		}

		filtered = append(filtered, restaurant)
	}

	return filtered
}

func filterHotelsByCriteria(hotels []models.HotelDetailedInfo, categories []string, priceRanges []string, ratingStr string, features []string) []models.HotelDetailedInfo {
	var filtered []models.HotelDetailedInfo

	minRating := parseRating(ratingStr)

	for _, hotel := range hotels {
		// Category filter
		if len(categories) > 0 && !matchesCategory(hotel.Category, nil, hotel.Tags, categories) {
			continue
		}

		// Price range filter
		if len(priceRanges) > 0 && !matchesPriceRange(getPriceLevel(hotel.PriceRange), priceRanges) {
			continue
		}

		// Rating filter
		if minRating > 0 && hotel.Rating < minRating {
			continue
		}

		// Feature filter
		if len(features) > 0 && !matchesFeatures(hotel.Tags, hotel.Description, features) {
			continue
		}

		filtered = append(filtered, hotel)
	}

	return filtered
}

func filterActivitiesByCriteria(activities []models.POIDetailedInfo, categories []string, priceRanges []string, ratingStr string, features []string) []models.POIDetailedInfo {
	var filtered []models.POIDetailedInfo

	minRating := parseRating(ratingStr)

	for _, activity := range activities {
		// Category filter
		if len(categories) > 0 && !matchesCategory(activity.Category, nil, activity.Tags, categories) {
			continue
		}

		// Price range filter
		if len(priceRanges) > 0 && !matchesPriceRange(activity.PriceRange, priceRanges) {
			continue
		}

		// Rating filter
		if minRating > 0 && activity.Rating < minRating {
			continue
		}

		// Feature filter
		if len(features) > 0 && !matchesFeatures(activity.Tags, activity.Description, features) {
			continue
		}

		filtered = append(filtered, activity)
	}

	return filtered
}

func filterPOIsByCriteria(pois []models.POIDetailedInfo, categories []string, priceRanges []string, ratingStr string) []models.POIDetailedInfo {
	var filtered []models.POIDetailedInfo

	minRating := parseRating(ratingStr)

	for _, poi := range pois {
		// Category filter
		if len(categories) > 0 && !matchesCategory(poi.Category, nil, poi.Tags, categories) {
			continue
		}

		// Price range filter
		if len(priceRanges) > 0 && !matchesPriceRange(poi.PriceRange, priceRanges) {
			continue
		}

		// Rating filter
		if minRating > 0 && poi.Rating < minRating {
			continue
		}

		filtered = append(filtered, poi)
	}

	return filtered
}

// Helper functions

func parseRating(ratingStr string) float64 {
	if ratingStr == "" {
		return 0
	}
	rating, err := strconv.ParseFloat(ratingStr, 64)
	if err != nil {
		return 0
	}
	return rating
}

func getPriceLevel(priceStr *string) string {
	if priceStr == nil {
		return ""
	}
	return *priceStr
}

func matchesCategory(category string, cuisineType *string, tags []string, filters []string) bool {
	categoryLower := strings.ToLower(category)

	for _, filter := range filters {
		filterLower := strings.ToLower(filter)

		// Check category
		if strings.Contains(categoryLower, filterLower) {
			return true
		}

		// Check cuisine type for restaurants
		if cuisineType != nil && strings.Contains(strings.ToLower(*cuisineType), filterLower) {
			return true
		}

		// Check tags
		for _, tag := range tags {
			if strings.Contains(strings.ToLower(tag), filterLower) {
				return true
			}
		}
	}

	return false
}

func matchesPriceRange(priceLevel string, filters []string) bool {
	if priceLevel == "" {
		return false
	}

	priceLower := strings.ToLower(priceLevel)

	for _, filter := range filters {
		filterLower := strings.ToLower(filter)

		// Match exact price level
		if strings.Contains(priceLower, filterLower) {
			return true
		}

		// Handle free filter
		if filterLower == "free" && (priceLower == "free" || priceLower == "") {
			return true
		}
	}

	return false
}

func matchesFeatures(tags []string, description string, filters []string) bool {
	descLower := strings.ToLower(description)

	for _, filter := range filters {
		filterLower := strings.ToLower(filter)
		matched := false

		// Check tags
		for _, tag := range tags {
			if strings.Contains(strings.ToLower(tag), filterLower) {
				matched = true
				break
			}
		}

		// Check description
		if !matched && strings.Contains(descLower, filterLower) {
			matched = true
		}

		// If this filter doesn't match, exclude the item
		if !matched {
			return false
		}
	}

	return true
}
