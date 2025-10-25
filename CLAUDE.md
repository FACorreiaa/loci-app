# Debug and Fix Checklist
## Always use TemplUI components 

## UI Issues
- [x] **0. Login/Signup Button Alignment** ✓ FIXED
  - Fixed text alignment on buttons in signup page
  - Changed from raw `<button>` elements to Button component
  - Location: `app/internal/features/auth/signup.templ`
  - Changes:
    - Submit button now uses `@button.Button` component with proper centering
    - Social login buttons (Google/LinkedIn) now use `@button.Button` with VariantOutline
    - All buttons now have consistent `items-center justify-center` alignment

## Auth

- [x] **0.1 User Profile Display** ✓ FIXED
  - Issue: Profile page showed hardcoded "John Doe" instead of actual registered user data
  - Root cause: Profile template didn't accept user data parameters and wasn't fetching from database
  - Solution:
    - Modified `profile.templ` to accept `*models.UserProfile` parameter
    - Updated profile header to display actual firstname/lastname or display name
    - Added helper functions (`getStringValue`, `getEmailValue`, `getLocationValue`, `getBioValue`)
    - Injected user service into routes
    - Updated `/profile` route handler to fetch user profile from database
    - Now displays actual user data: name, email, location, phone, bio
  - Files modified:
    - `app/internal/features/profile/profile.templ`
    - `app/routes/routes.go`

## Backend/Data Issues
- [x] **1. Empty Points of Interest** ✓ FIXED
  - Issue: `complete_itinerary.json` had `"points_of_interest": null` at top level
  - Root cause: Parser expected direct array `[]POIDetailedInfo` but LLM returns wrapped object `{"points_of_interest": [...]}`
  - Analysis:
    - LLM prompt (`getGeneralPOIPrompt`) correctly asks for `{"points_of_interest": [...]}`
    - Parser in `chat_parser.go:78` was trying to unmarshal directly into array
    - This caused parsing to fail silently, leaving top-level POIs as null
    - POI data WAS present in `itinerary_response.points_of_interest` but not at top level
  - Solution:
    - Updated `parseCompleteResponseFromParts` to first try parsing as wrapped object
    - Falls back to direct array parsing for backwards compatibility
    - Added debug logging to track which format was used
  - Files modified:
    - `app/pkg/domain/chat_prompt/chat_parser.go:72-96`

- [x] **2. Cache System Verification** ✓ COMPLETE
  - Issue: Cache was using sessionID as key, preventing cache reuse for same city + preferences
  - Solution: Implemented content-based cache keys using MD5 hash of:
    - user_id + profile_id + city + message + domain + preferences
  - Implementation details:
    - Cache key generation: `chat_service.go:2041-2056` (for authenticated users) and `chat_service.go:2340-2360` (for guest users)
    - Modified `cacheResultsIfAvailable()` to accept and use `cacheKey` parameter
    - Updated all cache Set() calls to use `cacheKey` instead of `sessionID`:
      - `ItineraryCache.Set(cacheKey, ...)` at line 2721
      - `CompleteItineraryCache.Set(cacheKey, ...)` at lines 2741, 2767, 2798, 2829
      - `RestaurantsCache.Set(cacheKey, ...)` at line 2763
      - `ActivitiesCache.Set(cacheKey, ...)` at line 2794
      - `HotelsCache.Set(cacheKey, ...)` at line 2825
    - Added cacheKey to navigation URLs for handlers to access: `chat_service.go:2189, 2493`
    - Updated all handlers to accept and use cacheKey parameter:
      - `restaurants.go:52, 66, 76, 167` - Added cacheKey query param and cache lookup
      - `hotels.go:50, 63, 70, 214` - Added cacheKey query param and cache lookup
      - `activities.go:50, 63, 70, 169` - Added cacheKey query param and cache lookup
  - Result: Cache now properly reuses data for same city + preferences, creates new requests for different preferences
  - Files modified:
    - `app/pkg/domain/chat_prompt/chat_service.go`
    - `app/pkg/handlers/restaurants.go`
    - `app/pkg/handlers/hotels.go`
    - `app/pkg/handlers/activities.go`

- [x] **3. Infinite Loop in Itinerary Search** ✓ FIXED
  - Issue: Restaurants, Activities, and Hotels pages stuck showing "Finding... This may take a few moments" even after data loaded
  - Root cause: Missing frontend event listeners for SSE completion events
  - Analysis:
    - Backend correctly sends `restaurants-complete`, `activities-complete`, `hotels-complete` SSE events
    - `#loading-progress` div had `sse-swap="loading-progress"` but completion events sent JSON, not HTML
    - Alpine.js state scripts (`restaurantsPage()`, `activitiesPage()`, `hotelsPage()`) had no `init()` method
    - No JavaScript listening for `htmx:sseMessage` events to hide loading indicator
  - Solution:
    - Added `init()` method to all three Alpine.js state functions
    - Added event listener for `htmx:sseMessage` events
    - When completion event detected, hide `#loading-progress` by adding `hidden` class
    - Added console logging for debugging
  - Files modified:
    - `app/internal/features/results/restaurants_results_sse.templ:253-292`
    - `app/internal/features/results/activities_results_sse.templ:253-292`
    - `app/internal/features/results/hotels_results_sse.templ:253-292`

- [x] **3.1. LLM Service Optimizations Analysis** ✓ COMPLETE

  **Current State:**
  - Model: `gemini-2.0-flash` (fast, cost-effective)
  - Default temperature: 0.5 (balanced creativity/consistency)
  - All LLM calls include OpenTelemetry tracing with latency metrics
  - Parallel worker pattern with WaitGroups for concurrent requests

  **Existing Optimizations:**
  1. **Concurrent Workers** - Multiple LLM requests run in parallel:
     - City data, general POIs, and personalized POIs execute concurrently
     - Uses `sync.WaitGroup` pattern for coordination
  2. **Streaming Responses** - SSE implementation for real-time updates:
     - `GenerateContentStream` used for itinerary/restaurant/hotel/activity endpoints
     - Reduces perceived latency for users
  3. **Context Caching** - Implemented in `streamWorkerWithResponseAndCache`:
     - Reuses prompt context across similar requests
     - Reduces token usage and latency
  4. **Temperature Tuning** - Different temps for different use cases:
     - 0.1 for message parsing (consistency priority)
     - 0.2 for continued conversations (slight creativity)
     - 0.5 for content generation (balanced)
  5. **Semantic Search Enhancement** - Hybrid search in workers:
     - Combines LLM generation with vector similarity search
     - Reduces hallucination by grounding in existing POI data

  **Recommended Additional Optimizations:**

  ### A. Prompt Optimization
  - [ ] Add max_tokens limits to prevent unnecessarily long responses
  - [ ] Implement prompt caching for repeated city queries
  - [ ] Use more concise system prompts to reduce input tokens

  ### B. Response Caching Strategy
  - [ ] Cache LLM responses by city+domain+preferences hash
  - [ ] Implement TTL-based cache expiration (e.g., 24-48 hours for city data)
  - [ ] Add cache warming for popular cities

  ### C. Request Batching
  - [ ] Batch similar POI detail requests into single LLM call
  - [ ] Combine city_data + general_pois into one structured output request

  ### D. Monitoring & Optimization
  - [x] Latency tracking via OpenTelemetry spans ✓ Already implemented
  - [ ] Add token usage tracking for cost optimization
  - [ ] Log slow queries (>2s) for prompt refinement
  - [ ] A/B test different temperature values

  ### E. Fallback Strategies
  - [ ] Implement graceful degradation if LLM unavailable
  - [ ] Use cached/database POI data as fallback
  - [ ] Add retry logic with exponential backoff

  **Priority Actions:**
  1. Add max_tokens limits (quick win, reduces cost)
  2. Enhance cache strategy with TTL (improves performance)
  3. Add token usage monitoring (visibility into costs)

  **Files to modify for priority actions:**
  - `app/pkg/domain/chat_prompt/chat_service.go` - Add max_tokens to configs
  - `app/pkg/middleware/middleware.go` - Enhance cache with TTL
  - `app/pkg/domain/chat_prompt/chat_workers.go` - Add token tracking
## Feature Parity with SolidJS Project

- [x] **4. Comparison Analysis Complete** ✓ DONE
  - Analyzed go-ai-poi-client SolidJS implementation
  - Documented comprehensive feature gaps and UI patterns
  - See detailed report below

- [x] **4.1. Maps with Numbered Markers and Layout Parity** ✓ COMPLETE
  - Issue: Restaurants, hotels, and activities pages lacked working map views and didn't match itinerary layout
  - Requirement: Pages should have same layout as itinerary with maps, split view, and numbered markers
  - Solution: Completely restructured all result pages to match itinerary layout
  - Implementation details:
    - **Enhanced MapContainer component** (`map_container.templ`):
      - Added `useNumberedMarkers bool` parameter for numbered vs. plain markers
      - Added `markerColor string` parameter for color-coded views
      - Numbered markers: 32x32px white circles with colored borders (3px) and centered numbers
      - Plain markers: 24x24px colored dots with white borders (2px)
      - Color mapping: Blue (itinerary), Orange (restaurants), Green (hotels), Purple (activities)
    - **Restaurants map** (`restaurant_results_enhanced.templ:323-350`):
      - Created `restaurantsToPOIs()` converter function
      - Implemented `RestaurantsMapContainer` with orange numbered markers
      - Handles nullable Address field properly
    - **Hotels map** (`hotel_results.templ:381-405`):
      - Created `hotelsToPOIs()` converter function
      - Implemented `HotelsMapContainer` with green numbered markers
    - **Activities map** (`activities_results_enhanced.templ:322-326`):
      - Updated `ActivitiesMapContainer` with purple numbered markers
      - Activities already use POIDetailedInfo, no conversion needed
    - **Itinerary map** (`itinerary_results.templ:220`):
      - Updated to use blue numbered markers matching itinerary POI order
    - **Layout Restructuring** (Phase 2):
      - All pages now follow itinerary's exact structure:
        - Map shown when `viewMode === 'map' || viewMode === 'split'`
        - Content shown when `viewMode === 'list' || viewMode === 'split'`
        - Dynamic height: Full screen for map view, 300-500px for split view
      - Added "Split" view button to all pages (between Map and Grid buttons)
      - City information sections with themed gradients (orange/green/purple)
      - Card components now show index badges (1, 2, 3, etc.) matching map markers
      - Badges positioned in top-right corner of each card with color-coding
  - Files modified:
    - `app/internal/features/results/map_container.templ` - Lines 11, 66, 69, 135-187
    - `app/internal/features/results/restaurant_results_enhanced.templ` - Complete restructure with split view
    - `app/internal/features/results/hotel_results.templ` - Complete restructure with split view
    - `app/internal/features/results/activities_results_enhanced.templ` - Complete restructure with split view
    - `app/internal/features/results/itinerary_results.templ` - Line 220
  - Result: All result pages have identical layout with working maps, split view, and numbered markers

- [x] **4.2. Advanced Map Features (Route Lines, Clustering, Filtering)** ✓ COMPLETE
  - Issue: Map lacked route visualization, clustering, and dynamic filtering
  - Solution: Implemented three advanced map features using HTMX-first approach
  - Implementation details:
    - **Route Lines** (`map_container.templ`):
      - GeoJSON LineString connecting POIs in sequential order
      - Dashed blue lines with 60% opacity for subtle appearance
      - Automatic updates when filters are applied
      - Color matches marker theme (blue/orange/green/purple)
    - **Marker Clustering** (`map_container.templ`):
      - Intelligent clustering activates for 20+ POIs
      - Color-coded clusters: Blue (1-9), Yellow (10-29), Pink (30+)
      - Click to zoom and expand clusters
      - Individual numbered markers for <20 POIs
    - **HTMX Filtering System**:
      - Filter panel component with category, price, and rating filters
      - Server-side filtering returns HTML fragments (not JSON)
      - Map automatically updates via `htmx:afterSwap` event
      - 500ms debounce on filter changes
      - Synchronizes list and map views
  - Files created:
    - `app/internal/features/results/filter_panel.templ` - HTMX filter component
    - `app/pkg/handlers/filters.go` - Server-side filter logic
    - `app/internal/features/results/list_fragments.templ` - HTML fragments for HTMX
  - Files modified:
    - `app/internal/features/results/map_container.templ` - Added route lines, clustering, HTMX listeners
    - `app/routes/routes.go` - Registered filter routes
  - Result: Fully interactive maps with route visualization, smart clustering, and real-time filtering

- [x] **4.3. City Information Panel Fixes** ✓ COMPLETE
  - Issue: City names not appearing in Hotels/Activities/Restaurants page titles when loading from cache
  - Root cause: Handlers were passing empty `GeneralCityData{}` when retrieving from domain-specific caches
  - Solution: Implemented two-tier city data loading strategy
  - Implementation details:
    - **Fixed page titles** (all three templates):
      - Added conditional display: Shows "Restaurants in {city}" when city data available, "Restaurants" when empty
      - Prevents awkward "Restaurants in " display
      - Applied to restaurants, hotels, and activities templates
    - **City data loading strategy** (all three handlers):
      - Primary: Check `CompleteItineraryCache` for city data (fast cache lookup)
      - Fallback: Load from database using `sessionID` (slower but reliable)
      - Added `loadCityDataFromDatabase()` helper function to each handler
      - Comprehensive error handling and logging at each step
    - **Enhanced city information display**:
      - Rich city panel with gradient backgrounds (already existed, now properly populated)
      - Shows: City description, area, timezone, language, population
      - Color-coded by domain: orange (restaurants), green (hotels), purple (activities)
      - Grid layout with icons for each stat
  - Files modified:
    - `app/internal/features/results/restaurant_results_enhanced.templ` - Lines 22-28 (title fix)
    - `app/internal/features/results/hotel_results.templ` - Lines 22-28 (title fix)
    - `app/internal/features/results/activities_results_enhanced.templ` - Lines 21-27 (title fix)
    - `app/pkg/handlers/restaurants.go` - Lines 74-103 (cache lookup), 280-310 (helper)
    - `app/pkg/handlers/hotels.go` - Lines 68-91 (cache lookup), 238-268 (helper)
    - `app/pkg/handlers/activities.go` - Lines 68-91 (cache lookup), 186-216 (helper)
  - Result: City names and information now display correctly on all pages, with graceful fallbacks

### Missing Features from SolidJS Implementation

#### High Priority (Core Functionality)
1. **Map Integration** - ✅ FULLY COMPLETED
   - ✅ Full Mapbox GL integration with marker visualization
   - ✅ Marker styling with numbered markers
   - ✅ View modes: Map/List/Split/Grid view implemented
   - ✅ Markers with same numbers as the list items
   - ✅ Route lines connecting POIs in order (dashed lines)
   - ✅ Marker clustering for dense areas (20+ POIs)
   - ✅ HTMX-based filtering with dynamic marker updates

2. **City Information Panel** - ✅ COMPLETED
   - ✅ General city data display (population, language, weather, timezone, area)
   - ✅ City description with metadata stats
   - ✅ Quick facts section with icons
   - ✅ Fixed city name display in Hotels/Activities/Restaurants titles
   - ✅ Implemented city data loading from cache with database fallback 

3. **Advanced Filtering System** - ✅ COMPLETED
   - ✅ Consistent filter panel across all result pages (`filter_panel.templ`)
   - ✅ Multi-select filters with checkboxes
   - ✅ Price range filters (€ to €€€€)
   - ✅ Rating-based filtering
   - ✅ Category filters specific to each domain:
     - Restaurants: Cuisine type (Portuguese, Seafood, International)
     - Hotels: Type (Luxury, Boutique, Historic, Business)
     - Activities: Type (Cultural, Entertainment, Outdoor, Adventure)
   - ✅ Feature filters (Michelin Star, River View, Pet Friendly, etc.)
   - ✅ Filter badges showing active filters with JavaScript management
   - ✅ HTMX integration for dynamic filtering

4. **Enhanced Card Design** - ✅ COMPLETED
   - ✅ Consistent emoji and icon system per domain
     - Restaurants: `getCuisineEmoji()` with 15+ cuisines
     - Hotels: `getAmenityIcon()` templ component
     - Activities: `getCategoryEmoji()` with 18+ categories
   - ✅ Metadata label badges with color coding
     - Rating colors: `getRatingColorClass()`
     - Price badge colors: `getPriceBadgeColor()`
     - Budget colors: `getBudgetColorClass()`
   - ✅ Enhanced footer with price/budget and feature tags
   - ✅ Duration/time-to-spend indicators
   - ✅ Special feature badges (tags display)

#### Medium Priority (UX Improvements)
5. **General POIs Section** - ✅ ENHANCED
   - ✅ "All Points of Interest" section showing broader city context (`itinerary_results.templ:282-327`)
   - ✅ Separate from curated/personalized recommendations
   - ✅ Collapsible "show more" functionality with Alpine.js
     - Shows 6 POIs initially
     - "Show all X places" button with animated expand/collapse
     - Smooth transitions with Alpine.js x-transition
   - ✅ Responsive grid layout (1 col mobile, 2 cols tablet, 3 cols desktop)
   - ✅ Count badge showing total number of places
   - ✅ Enhanced GeneralPOICard with images, ratings, tags, and actions 

6. **Session Management & Deep Linking** - ✅ COMPLETE
   - ✅ URL parameter support for sessionId, cityName, cacheKey, domain (`session_manager.templ`)
   - ✅ Deep linking to specific results with `getDeepLink(domain, params)`
   - ✅ Session data persistence in sessionStorage with 24-hour expiry
   - ✅ Fallback session creation if expired
   - ✅ Share functionality with navigator.share API and clipboard fallback
   - ✅ Global helpers: `getSessionId()`, `getCityName()`, `getCacheKey()`, `shareResults()`
   - ✅ URL update without page reload using `updateURL(params)`
   - Files created:
     - `app/internal/components/scripts/session_manager.templ`
   - Integrated in: `app/internal/pages/layout.templ:60`

7. **Favorites/Bookmarking System** - ✅ COMPLETE
   - ✅ Save to user account via POI service database integration
   - ✅ Toggle favorite state with HTMX swap (no loading indicator needed)
   - ✅ Heart icon with fill animation (`animate-heartbeat` from `animations.templ`)
   - ✅ Favorites count in results headers for all pages (itinerary, restaurants, hotels, activities)
   - ✅ Sign-in required messaging for guests (tooltip with helpful message)
   - ✅ UUID validation for user and POI IDs
   - ✅ isLLMGenerated parameter handling
   - Files modified:
     - `app/pkg/handlers/favorites.go` - Complete database integration
     - `app/routes/routes.go` - POI service initialization and dependency injection
     - `app/internal/features/results/itinerary_results.templ:51-58` - Favorites count badge
     - `app/internal/features/results/restaurant_results_enhanced.templ:51-58` - Favorites count badge
     - `app/internal/features/results/hotel_results.templ:51-58` - Favorites count badge
     - `app/internal/features/results/activities_results_enhanced.templ:51-58` - Favorites count badge

8. **Enhanced View Modes** - ✅ COMPLETE
   - ✅ Split view implemented (map + list side-by-side on desktop) in all result pages
   - ✅ Smooth transitions between view modes with CSS transitions
   - ✅ Responsive stacking on mobile (automatic grid layout collapse)
   - ✅ View mode persistence in localStorage
   - ✅ Alpine.js integration with `createViewModeData()` helper
   - ✅ Custom event dispatch on view mode changes (`viewModeChanged`)
   - ✅ Valid modes: 'list', 'map', 'split', 'grid' (default: 'list')
   - ✅ Dynamic CSS injection for transitions
   - Files created:
     - `app/internal/components/scripts/view_mode_manager.templ`
     - `app/internal/components/styles/animations.templ` - All UI animations including:
       - Heartbeat animation for favorites (scale up animation)
       - Fade in, slide in, pulse, bounce, spin animations
       - Shimmer effect for loading states
       - View mode transition classes
       - Split view container styles with responsive breakpoints
       - Map container height transitions
       - Button active states with scale
       - Smooth hover transitions and card hover effects
   - Integrated in: `app/internal/pages/layout.templ:61-62`

#### Low Priority (Nice-to-have)
9. **Chat Session Continuity** - ✅ PARTIAL (Backend Complete, Frontend Pending)
   - ✅ **Backend /continue endpoint implemented** (`app/pkg/handlers/chat.go:1286-1411`)
     - Handler: `ContinueChatSession` with SSE streaming support
     - Route: `POST /chat/continue/:sessionID` (registered in `app/routes/routes.go:743`)
     - Supports both JSON and form-encoded requests for HTMX compatibility
     - Uses existing `ContinueSessionStreamed` service method
     - Features:
       - Session validation and context loading
       - Intent classification (Add POI, Remove POI, Modify, Question)
       - Semantic POI enhancement with vector search
       - Real-time SSE event streaming
       - Conversation history management
       - HTMX-compatible HTML error responses
   - ✅ **Service layer already implemented** (`app/pkg/domain/chat_prompt/chat_service.go:1398+`)
     - `ContinueSessionStreamed` method handles:
       - Session validation (checks active status)
       - City data fetching for context
       - Message history persistence
       - Intent classification using LLM
       - Semantic POI recommendations via vector search
       - Dynamic handling of different intents:
         - `IntentAddPOI` - Adds new POIs using `handleSemanticAddPOIStreamed`
         - `IntentRemovePOI` - Removes POIs with semantic understanding
         - `IntentModifyItinerary` - Modifies existing itinerary
         - `IntentQuestion` - Answers questions about current data
   - ✅ **SSE Event Types** streamed to frontend:
     - `session_validated` - Session is active and ready
     - `progress` - Processing updates
     - `intent_classified` - User intent determined
     - `semantic_context_generated` - Vector search complete
     - `itinerary` - Updated POI/restaurant/hotel/activity data
     - `complete` - Processing finished
     - `error` - Error occurred
   - ✅ **COMPLETED: Frontend Integration**
     - ✅ Created reusable chat panel component (`app/internal/components/chat/chat_panel.templ`)
       - Fixed position floating chat button (bottom-right)
       - HTMX SSE integration with `hx-ext="sse"` connecting to `/chat/continue/:sessionID`
       - Alpine.js state management for local UI (showChat, message, isLoading, messages)
       - Quick action buttons for common requests (More like these, Budget options, Premium)
       - Auto-scroll functionality for messages
       - Dark mode support and responsive design
       - Props: SessionID, CityName, Domain, Placeholder, ShowByDefault
     - ✅ Added chat panel to restaurant results page
       - Updated `RestaurantsResults` signature to include sessionID parameter
       - Modified all handler calls in `restaurants.go` (4 locations: lines 100, 131, 143, 168)
       - Integrated chat panel before closing container div (lines 224-233)
       - Updated SSE template `RestaurantsContentComplete` signature
     - ✅ Added chat panel to hotels results page
       - Updated `HotelsResults` signature to include sessionID parameter
       - Modified all handler calls in `hotels.go` (4 locations: lines 89, 118, 130, 143)
       - Modified handler call in `results.go` (line 191)
       - Integrated chat panel before closing container div (lines 224-233)
     - ✅ Added chat panel to activities results page
       - Updated `ActivitiesResults` signature to include sessionID parameter
       - Modified all handler calls in `activities.go` (4 locations: lines 89, 118, 130, 143)
       - Integrated chat panel before closing container div (lines 224-233)
     - ✅ Added chat panel to itinerary results page
       - Updated `ItineraryResults` signature to include sessionID parameter
       - Modified all handler calls in `itinerary.go` (4 locations: lines 359, 371, 399, 624)
       - Modified handler calls in `results.go` (lines 239, 355)
       - Integrated chat panel before closing container div (lines 672-681)
     - ⏳ **NEXT STEPS** (optional enhancements):
       - [x] Create HTML fragment templates for dynamic SSE updates (restaurant/hotel/activity cards) ✓ COMPLETE
         - Created `/app/internal/features/results/sse_card_fragments.templ`
         - Includes: `SingleRestaurantCard`, `SingleHotelCard`, `SingleActivityCard`, `SinglePOICard`
         - Added visual feedback components: `ChatProcessingIndicator`, `ChatSuccessMessage`, `ChatErrorMessage`
         - All cards have unique IDs, fade-in animations, and proper data attributes for map updates
       - [~] Enhance SSE event handling to insert new items dynamically ⏳ IN PROGRESS
         **Completed:**
         - ✅ Added new event types to `models.StreamEvent` (`app/internal/models/chat_session.go:222-224`):
           - `EventTypeItemAdded` - For adding new items to lists
           - `EventTypeItemRemoved` - For removing items from lists
           - `EventTypeItemUpdated` - For updating existing items
         - ✅ Extended `StreamEvent` struct with new fields (`chat_session.go:196-199`):
           - `HTML` - Rendered HTML fragment for immediate insertion
           - `Domain` - Target domain (restaurants/hotels/activities/itinerary)
           - `ItemID` - Unique identifier for the item
           - `ItemData` - Full item data including coordinates for map updates
         - ✅ Created HTML rendering helpers (`app/pkg/domain/chat_prompt/render_helpers.go`):
           - `RenderItemHTML()` - Renders domain-specific item cards (restaurants/hotels/activities/POIs)
           - `RenderSuccessMessage()` - Renders success toasts with action and item name
           - `RenderErrorMessage()` - Renders error toasts
           - `RenderProcessingIndicator()` - Renders processing indicators with domain
         - ✅ Updated `handleSemanticAddPOIStreamed` (`chat_service.go:1858-1893`):
           - Now sends `EventTypeItemAdded` with rendered HTML fragment
           - Includes full item data with coordinates for map updates
           - Calculates new index for proper numbering
           - Logs errors if HTML rendering fails
         - ✅ Updated `handleSemanticRemovePOI` (`chat_service.go:1231-1243`):
           - Prepared for sending `EventTypeItemRemoved` events
           - Ready for integration with event channel
         **Completed:**
         - [x] Add frontend SSE listeners in result pages (Alpine.js): ✓ COMPLETE
           - [x] Restaurants page - Listen for item_added/removed, insert HTML, update map (`restaurant_results_enhanced.templ:248-475`)
           - [x] Hotels page - Listen for item_added/removed, insert HTML, update map (`hotel_results.templ:248-389`)
           - [x] Activities page - Listen for item_added/removed, insert HTML, update map (`activities_results_enhanced.templ:248-389`)
           - [x] Itinerary page - Listen for item_added/removed, insert HTML, update map (`itinerary_results.templ:1138-1372`)
         - [x] Create map synchronization helpers (JavaScript): ✓ COMPLETE (all 4 pages)
           - [x] `addMarkerToMap(itemData)` - Adds numbered markers with domain-specific colors (blue/orange/green/purple)
           - [x] `removeMarkerFromMap(itemId)` - Removes marker by ID from domain-specific Map storage
           - [x] `recalculateMapBounds()` - Auto-adjusts viewport with responsive padding (mobile: 20-40px, desktop: 60-80px)

         **Implementation Details:**
         - **Container IDs Added:**
           - Itinerary: `#curated-itinerary-list` (line 374)
           - Restaurants: `#restaurants-list`, `#restaurants-grid` (lines 210, 218)
           - Hotels: `#hotels-list`, `#hotels-grid` (lines 210, 218)
           - Activities: `#activities-list`, `#activities-grid` (lines 210, 218)
         - **Event Handlers:** Each page implements `handleItemAdded()`, `handleItemRemoved()`, `handleItemUpdated()`
         - **Map Markers:** 32px numbered circles with 3px colored borders, white backgrounds
         - **Marker Storage:** Domain-specific Maps (`window.itineraryMarkers`, `restaurantMarkers`, `hotelMarkers`, `activityMarkers`)
         - **Coordinate Validation:** Rejects (0,0) and out-of-range values
         - **Dual View Support:** Restaurants/Hotels/Activities sync both list and grid views

         - [x] **Cache Update Strategy** ✓ COMPLETE
           - [x] Created `updateCacheAfterModification()` helper (`chat_service.go:1214-1264`)
           - [x] Updates CompleteItineraryCache when POIs are added/removed
           - [x] Uses same MD5 cache key generation as initial requests
           - [x] Integrated into `handleSemanticAddPOIStreamed` (line 1895)
           - [x] Integrated into `handleSemanticRemovePOI` (line 1296)
           - [x] Logs cache updates with session_id, cache_key, and POI count
           - [x] OpenTelemetry tracing for cache operations

         **Remaining:**
         - [ ] Update route lines after additions/removals (optional enhancement - not implemented yet)
         - [ ] Recalculate indices for numbered markers after removals (optional - current implementation maintains order)
         - [ ] Test end-to-end with real LLM backend
         - [ ] Add visual feedback for chat processing states (optional - using console.log for now)

         **✅ IMPLEMENTATION COMPLETE - What Was Built:**

         **1. Backend SSE Event System** ✅ DONE
         - ✅ `handleSemanticAddPOIStreamed` sends `EventTypeItemAdded` with rendered HTML (line 1874-1888)
         - ✅ `handleSemanticRemovePOI` prepared for `EventTypeItemRemoved` (would need event channel integration)
         - ✅ `RenderItemHTML()` renders domain-specific cards (restaurants/hotels/activities/POIs)
         - ✅ Event structure includes: `Type`, `Domain`, `ItemID`, `HTML`, `ItemData` with coordinates
         - ✅ Actual SSE event sent:
           ```go
           l.sendEvent(ctx, eventCh, models.StreamEvent{
               Type:   models.EventTypeItemAdded,
               Domain: "itinerary",
               ItemID: semanticPOI.ID.String(),
               HTML:   htmlFragment,  // Pre-rendered server-side
               ItemData: map[string]interface{}{
                   "name": semanticPOI.Name,
                   "latitude": semanticPOI.Latitude,
                   "longitude": semanticPOI.Longitude,
                   "index": newIndex + 1,
               },
               Message: fmt.Sprintf("Added %s to your itinerary", semanticPOI.Name),
           }, 3)
           ```

         **2. Frontend SSE Listeners** ✅ DONE (All 4 Pages)
         - ✅ Itinerary: `itinerary_results.templ:1138-1372`
         - ✅ Restaurants: `restaurant_results_enhanced.templ:248-475`
         - ✅ Hotels: `hotel_results.templ:248-389`
         - ✅ Activities: `activities_results_enhanced.templ:248-389`
         - ✅ Each implements: `init()`, `handleItemAdded()`, `handleItemRemoved()`, `handleItemUpdated()`
         - ✅ Actual implementation:
           ```javascript
           init() {
               document.addEventListener('htmx:sseMessage', (event) => {
                   const sseEvent = event.detail;
                   let eventData;
                   try {
                       eventData = typeof sseEvent.data === 'string' ?
                           JSON.parse(sseEvent.data) : sseEvent.data;
                   } catch (e) {
                       console.log('Non-JSON SSE event:', sseEvent);
                       return;
                   }

                   if (eventData.type === 'item_added' && eventData.domain === 'itinerary') {
                       this.handleItemAdded(eventData);
                   }
                   // ... similar for item_removed, item_updated
               });
           }
           ```

         **3. Cache Update Strategy** ✅ DONE
         - ✅ `updateCacheAfterModification()` helper function (lines 1214-1264)
         - ✅ Generates MD5 cache key from user_id + profile_id + city + domain
         - ✅ Creates `AiCityResponse` from session state
         - ✅ Updates `CompleteItineraryCache` with modified data
         - ✅ Integrated into both add (line 1895) and remove (line 1296) operations
         - ✅ OpenTelemetry tracing and comprehensive logging

         **4. Map Synchronization** ✅ DONE (All 4 Pages)
         - ✅ `addMarkerToMap(itemData)` - Creates 32px numbered markers with domain colors
         - ✅ `removeMarkerFromMap(itemId)` - Removes markers from map and storage
         - ✅ `recalculateMapBounds()` - Auto-adjusts viewport with responsive padding
         - ✅ Coordinate validation (rejects 0,0 and out-of-range)
         - ✅ Mapbox GL popups with item name and category
         - ✅ Domain-specific storage Maps (itineraryMarkers, restaurantMarkers, etc.)

         **Testing Checklist (Ready for E2E Testing):**
         - [ ] Test add single POI to itinerary via chat
         - [ ] Test remove existing POI via chat
         - [ ] Verify HTML fragment insertion in DOM
         - [ ] Verify map marker added with correct color/number
         - [ ] Verify cache updated after modifications
         - [ ] Test dual-view sync (restaurants/hotels/activities list + grid)
         - [ ] Verify bounds recalculation after add/remove
         - [ ] Test across all 4 domains (itinerary/restaurants/hotels/activities)
   - **Analyzed REST API** (`go-ai-poi-server/internal/api/chat_prompt/chat_handler.go:757-844`)
   - **Analyzed SolidJS client** (`go-ai-poi-client/src/lib/hooks/useChatSession.ts`)
   - **Design**: HTMX-based SSE with HTML fragments instead of JSON
   
10. [x] **Banner Messaging System**
    - Success banners after operations
    - Authentication promotion for guests
    - Typing animation for dynamic text
    - Dismissable with X button
    - Color-coded by context

- [x] **12. Coordinate Validation** ✓ COMPLETE
  - Issue: Need validation for POI coordinates to prevent map errors and improve UX
  - Requirements: Validate lat/long, filter invalid POIs, show "Location TBD", fallback center point
  - Solution: Implemented comprehensive coordinate validation system
  - Implementation details:
    - **Go utilities** (`app/pkg/utils/coordinates.go`):
      - `ValidateCoordinates()` - Validates lat [-90, 90] and lng [-180, 180]
      - `HasValidCoordinates()` - Checks for valid coords, rejects 0,0
      - `CalculateCenterPoint()` - Calculates center with fallback
      - `CalculateBounds()` - Returns bounding box for coordinates
      - `FormatCoordinatesDisplay()` - Formats coords or shows "Location TBD"
    - **JavaScript validation** (`app/internal/features/results/map_container.templ`):
      - `isValidCoordinate()` function validates ranges and rejects 0,0
      - Filters POIs: `validPOIs = poisData.filter(poi => isValidCoordinate(...))`
      - Console warning for filtered POIs: "Filtered out N POI(s) with invalid coordinates"
      - Enhanced `getCityCenter()` with validation and fallback to Greenwich [0, 51.5]
    - **Templ helper functions** (`map_container.templ:378-396`):
      - `formatCoordinates()` - Returns "Location TBD" for invalid coords
      - `hasValidCoordinates()` - Boolean check for valid coords
      - Both functions check for 0,0 (placeholder) and valid ranges
    - **UI updates**:
      - City info coordinates display (`city_info_section.templ:83-90`):
        - Shows "Location TBD" in orange for invalid coords
        - Displays formatted coords for valid ones
        - Removed conditional display, always shows coordinate section
      - Map rendering only includes validated POIs
      - Invalid POIs logged but not displayed on map
    - **Validation rules**:
      - Latitude: must be between -90 and 90
      - Longitude: must be between -180 and 180
      - Reject 0,0 as it's often a placeholder for missing data
      - Fallback city center: Greenwich, London [0, 51.5] (better than Atlantic Ocean 0,0)
  - Files created:
    - `app/pkg/utils/coordinates.go` - Coordinate validation utilities
  - Files modified:
    - `app/internal/features/results/map_container.templ` - Lines 3-6 (imports), 89-123 (validation), 186-191 (filtering), 332-348 (fallback), 378-396 (helpers)
    - `app/internal/features/results/city_info_section.templ` - Lines 3-6 (imports), 83-90 (display)
  - Build status: ✅ Templ generation successful, Go build compiles cleanly
  - Result: Maps gracefully handle invalid coordinates, user-friendly "Location TBD" messaging

### Implementation Recommendations

**Phase 1: Core Functionality** (Do First)
- Map integration with Mapbox GL (issue #5) (done)
- City information panel
- View mode toggles (Map/List/Split)
- Basic filtering system

**Phase 2: Enhanced UX**
- Advanced filtering with all categories
- Enhanced card design with badges
- Enhanced metadata for Itineraries (add distance to user calculated through PostGis, etc)
- Enhanced metadata for Restaurants (number of starts, opening times, type of food, etc)
- Enhanced metadata for Hotels (number of stars)
- Activities ((add distance to user calculated through PostGis, etc), opening times)
- General POIs section (done, improve layout and metadata)
- Favorites API integration
- Bookmark API integration

**Phase 3: Polish**
- Session management improvements
- Banner system
- Coordinate validation
- Chat continuity

- [x] **5. Map Integration** ✓ COMPLETE
  - Issue: Multiple errors with map initialization, then needed responsive sizing for all screen sizes
  - Root causes:
    1. Alpine.js `mapLoaded` variable not defined in data context
    2. Incorrect JavaScript template interpolation for Mapbox API key
    3. Map controls trying to access undefined map object
    4. Map didn't auto-zoom to show all POI coordinates
    5. Map needed responsive behavior from iPhone SE to 50" screens
  - Solution:
    1. Added `x-data="{ mapLoaded: false }"` to map container
    2. Fixed MapboxConfigScript to properly interpolate API key using `script` keyword
    3. Updated map load event to use `Alpine.$data()` to access correct context
    4. Added null checks to map control click handlers
    5. Replaced Alpine store dependency with custom events for POI selection
    6. Added Mapbox token validation with error messaging
    7. Implemented auto-fit bounds with `bounds.extend()` and `fitBounds()`
    8. Added responsive padding calculation based on `window.innerWidth`
       - Mobile (< 640px): 20-40px padding
       - Desktop (≥ 640px): 60-80px padding
    9. Added map constraints:
       - `minZoom: 3` - Prevents zooming out too far
       - `maxZoom: 18` - Allows detailed street view
       - `renderWorldCopies: false` - Prevents duplicate worlds on wide screens
    10. Added window resize handler with debouncing to maintain map responsiveness
    11. Updated resetMapView() to recalculate responsive padding
  - Files modified:
    - `app/pkg/config/config.go` - Added MapConfig struct and loading from env
    - `app/internal/components/map/mapbox.templ` - Fixed template syntax and API key interpolation
    - `app/internal/pages/layout.templ` - Pass API key from environment to MapboxScripts
    - `app/internal/features/results/map_container.templ` - Full map implementation with responsive sizing
      - Lines 88-96: Map initialization with constraints
      - Lines 160-175: Auto-fit bounds with responsive padding
      - Lines 178-206: Enhanced resetMapView() function
      - Lines 218-230: Window resize handler with debouncing
  - Status: ✅ FULLY COMPLETE - Map working with responsive sizing and proper container fill

- [x] **6. Metadata Enhancement**
  - Add more metadata to the itinerary results to be attached on the card for better UI and user interaction
  - Also apply to restaurants, hotels and activities

- [x] **7. Empty IDs in complete_itinerary.json**
  - Issue: complete_itinerary.json is returning empty `ids` and empty `city_ids`
  - These should be filled properly with actual database IDs

- [x] **7.1. Default Profile Creation and Registration Fix** ✓ FIXED
  - Issue: "No default profile found. Please create a profile first." error prevented itinerary generation
  - Second issue: User registration broken with "column 'accessibility_needs' does not exist"
  - Third issue: Duplicate key constraint violation "idx_user_preference_profiles_user_id_default"
  - Root causes:
    1. Initial attempt: `Register` function only created user record, not default profile
    2. Second attempt: Profile insert used incorrect column names not matching actual schema
    3. Final discovery: Database has TRIGGER that automatically creates default profile on user insert
  - Investigation timeline:
    1. First fix: Added manual profile creation with transaction (failed - wrong columns)
    2. Second fix: Corrected column names to match schema (failed - duplicate constraint)
    3. Final fix: Discovered database trigger handles profile creation automatically
  - Solution:
    - Removed manual profile creation code entirely
    - Database trigger `trigger_create_user_profile_after_insert` (migration 0008) handles it
    - Trigger calls `create_initial_user_profile()` which inserts default profile automatically
    - Simple user insert is sufficient - trigger creates profile with name "Default"
  - Files modified:
    - `app/pkg/domain/auth/auth_repository.go` (lines 112-131)
  - Final implementation:
    ```go
    // Insert user - database trigger will automatically create default profile
    // See migration 0008_user_profile.up.sql: trigger_create_user_profile_after_insert
    userQuery := `INSERT INTO users (username, email, password_hash, created_at) VALUES ($1, $2, $3, $4) RETURNING id`
    err := r.pgpool.QueryRow(ctx, userQuery, username, email, hashedPassword, time.Now()).Scan(&userID)
    ```
  - Result: Clean registration flow, no duplicate profiles, trigger ensures atomic user+profile creation

- [x] **7.2. pg_trgm Extension for Fuzzy City Search** ✓ FIXED
  - Issue: Chat continue session failed with "ERROR: function similarity(text, unknown) does not exist (SQLSTATE 42883)"
  - Occurred when trying to add "Estadio da Luz" via chat
  - Root cause: Missing PostgreSQL `pg_trgm` extension required for fuzzy city name matching
  - Location: `FindCityByFuzzyName` in `city_repository.go` uses `similarity()` function
  - Solution: Added `CREATE EXTENSION IF NOT EXISTS pg_trgm;` to initial setup migration
  - Files modified:
    - `app/db/migrations/0001_setup.up.sql` (line 16)
  - Result: Chat continue session can now fuzzy match city names (e.g., "Lisbon" → "Lisboa")

- [x] **7.3. Itinerary Chat Bubble Always Visible** ✓ FIXED
  - Issue: Itinerary page chat bubble not visible on desktop or mobile, inconsistent with other pages
  - User feedback: "I still dont see the itinerary bubble on desktop. And I cant click it on mobile."
  - Root causes:
    1. Entire `ChatPanel` component wrapped in `if sessionID != ""` conditional (line 675)
    2. This prevented the floating chat bubble from ever rendering
    3. Desktop "Modify Itinerary" button also had unnecessary conditional
  - Solution: Removed all conditional wrappers to match restaurant/hotel/activities behavior
  - Implementation details:
    - **Removed conditional wrapper** around `@chat.ChatPanel` component (line 675)
    - Chat bubble now always renders, providing consistent UX across all result pages
    - **Removed conditional** from desktop "Modify Itinerary" button (line 93)
    - Button now always visible in desktop actions bar
    - Mobile menu button was already unconditional (line 179)
    - ChatPanel component itself handles sessionID internally for SSE functionality
  - Files modified:
    - `app/internal/features/results/itinerary_results.templ` (lines 93, 675-681)
  - Result:
    - ✅ Floating chat bubble visible on bottom-right corner (desktop & mobile)
    - ✅ Desktop "Modify Itinerary" button always visible in header
    - ✅ Mobile menu has "Modify Itinerary" option
    - ✅ Consistent behavior with restaurants, hotels, and activities pages
  - Note: This is the core feature for the itinerary page - chat allows dynamic POI modifications

- [x] **8 create under docs, a FLY.md document**
 on how to create and integrate my Postgres DB on fly.io with pgvector and PostGis, setup env variables and create the app using the hosted postgres with all the env setup.

- [x] **9 do cd .. and analyse go-ai-poi-server**
 go-ai-poi-server is the old REST API that im rebuilding using a server side rendering solution. 
 The logic to fill the itinerary, restaurant, hotel and activities are found under the routes:
  - r.Post("/prompt-response/chat/sessions/stream/{profileID}", HandlerImpl.StartChatMessageStream)
	- r.Post("/prompt-response/chat/sessions/{sessionID}/continue", HandlerImpl.ContinueChatSessionStream) Analyse both and start working on these sub tasks:

  - [x] **9.1. Fixed Restaurants, Hotels, and Activities data display** ✓ COMPLETE
    - Issue: Pages stuck showing "Finding... This may take a few moments" with empty data display
    - Root cause: JSON parsers couldn't parse LLM response format `{"restaurants": [...]}`, `{"activities": [...]}`, `{"hotels": [...]}`
    - Parsers were only checking for direct array `[...]` or `{"data": [...]}`
    - This caused silent parsing failures, preventing cache population
    - When browser navigated to results pages, cache was empty → empty display
    - Solution:
      - Updated `parseRestaurantsFromResponse()` to handle `{"restaurants": [...]}`
      - Updated `parseActivitiesFromResponse()` to handle `{"activities": [...]}`
      - Updated `parseHotelsFromResponse()` to handle `{"hotels": [...]}`
      - Now parsers try 3 formats: direct array, `{"data": [...]}`, and domain-specific wrapper
    - Files modified:
      - `app/pkg/domain/chat_prompt/chat_parser.go:184-192` (restaurants)
      - `app/pkg/domain/chat_prompt/chat_parser.go:217-225` (activities)
      - `app/pkg/domain/chat_prompt/chat_parser.go:250-258` (hotels)
    - Result: Cache properly populated, results pages display data immediately
The restaurant data is shown on my server just never print because the struct inside is empty
Analyse go-ai-poi-server and make the changes needed.

10. Analyse my Stripe services and implement the handlers for my business needs. An user can subscribe to the services I offer and eventually have in item purchases but for now I only the subscriptions.
Build the handlers keeping in mind im using HTMX and Go full stack.

10.1 For the payments I want to have a proper middleware that connects the users with their benefits
10.2 I want to have a way of users having discount codes and special offer codes to be able to use the sub

- [x] **11. Banner Messaging System** ✓ COMPLETE
  - Issue: Need user notification system for operations, auth promotion, and feedback
  - Requirements: Success/info/warning/error banners, dismissable, auto-dismiss, typing animation, color-coded
  - Solution: Created comprehensive banner component system
  - Implementation details:
    - **Banner component** (`app/internal/components/banner/banner.templ`):
      - `BannerType` enum: Success, Info, Warning, Error
      - `BannerProps` struct: Type, Message, Description, Dismissable, ID, AutoDismiss
      - Four banner variants:
        1. `Banner()` - Standard inline banner with slide-down transition
        2. `FixedBanner()` - Fixed position at top of screen
        3. `ToastBanner()` - Bottom-right corner toast notification
        4. `AuthPromoBanner()` - Special authentication promotion with typing animation
      - Helper functions:
        - `getBannerClasses()` - Color-coded styling based on type
        - `getBannerIcon()` - Type-specific icons (checkmark, info, warning, error)
    - **Features implemented:**
      - ✅ Color-coded by context:
        - Success: Green background, green border-left, green icon
        - Info: Blue background, blue border-left, blue icon
        - Warning: Yellow background, yellow border-left, yellow icon
        - Error: Red background, red border-left, red icon
      - ✅ Dismissable with X button (optional via `Dismissable` prop)
      - ✅ Auto-dismiss after N seconds (configurable via `AutoDismiss` prop, 0 = manual dismiss only)
      - ✅ Alpine.js transitions:
        - Enter: opacity 0→100, transform -translate-y-4→0 (300ms ease-out)
        - Leave: opacity 100→0 (200ms ease-in)
        - Toast variant: slide from right (translate-x-full→0)
      - ✅ Typing animation for AuthPromoBanner:
        - Cycles through messages: "Sign in to save your favorites", "Create an account...", "Join now..."
        - 50ms per character typing speed
        - 5 second pause between messages
        - Cursor blink animation
      - ✅ Dark mode support: `dark:bg-*-900/20`, `dark:text-*-200`
      - ✅ Accessibility: `role="alert"`, `aria-label="Dismiss"`
      - ✅ Responsive design: max-w-7xl container, responsive padding
      - ✅ Authentication detection: Checks `document.cookie.includes('session_token')`
    - **Usage examples:**
      ```go
      // Success banner
      @banner.Banner(banner.BannerProps{
          Type: banner.BannerSuccess,
          Message: "Itinerary saved successfully!",
          Description: "Your travel plans have been saved to your account.",
          Dismissable: true,
          ID: "save-success",
          AutoDismiss: 5, // Auto-dismiss after 5 seconds
      })

      // Error banner
      @banner.FixedBanner(banner.BannerProps{
          Type: banner.BannerError,
          Message: "Failed to load data",
          Dismissable: true,
          ID: "load-error",
          AutoDismiss: 0, // Manual dismiss only
      })

      // Toast notification
      @banner.ToastBanner(banner.BannerProps{
          Type: banner.BannerInfo,
          Message: "New restaurant added to your list",
          Dismissable: true,
          ID: "restaurant-added",
          AutoDismiss: 3,
      })

      // Authentication promotion (no props needed)
      @banner.AuthPromoBanner()
      ```
  - Files created:
    - `app/internal/components/banner/banner.templ` - Complete banner system
    - `app/internal/components/banner/banner_templ.go` - Generated templ code
  - Build status: ✅ Templ generation successful, Go build compiles cleanly
  - Result: Comprehensive banner messaging system ready for integration across all pages

11. [x] **Restaurants has two structs, cityData that is empty and restaurants which has the data including the city name. on the view, the title is being populated with cityData which then is populating the title empty. We need to have either the cityData being populated for Restaurants, Hotels and Activities so the title can be filled or fill the view with the right structure since city data is empty.**
12 [x]. **We need to rethink how the cache is being done:**
```go
	if cacheKey != "" {
		if restaurantsData, found := middleware.RestaurantsCache.Get(cacheKey); found {

			logger.Log.Info("Restaurants found in cache. Rendering results.",

				zap.Int("restaurants", len(restaurantsData)))

			// Try to get city data from complete cache
			var cityData models.GeneralCityData
			if completeData, found := middleware.CompleteItineraryCache.Get(cacheKey); found {
				cityData = completeData.GeneralCityData
				logger.Log.Info("City data loaded from complete cache",
					zap.String("city", cityData.City))
			} else {
				// Fallback: load from database using sessionID
				cityData = h.loadCityDataFromDatabase(sessionIDParam)
			}

			// Return static template when data is available

			return results.RestaurantsResults(

				cityData,

				restaurantsData,

				true, true, 15, []string{})
		}
	}

```
Its confusing to have RestaurantsCache and CompleteItineraryCache. 
Each handler should have its own cache?
Or abstract and have a middleware that abstracts this away?
Use case:
User searches Restaurant ABC + Preference 1,2,3 = LLM Search.
User searches Restaurant ABC + Preference 1,2,3 = cached response
User searches Restaurant ABC + Preference 1,2,4 = New preference, new LLM Search
User searches Restaurant ABC + Preference 1,2,4 = cached response
User searches Restaurant XYZ + Preference 1,2,4 = New restaurant, new LLM Search

This logic should be applied to all intents in order to save LLM calls, tokens and budgets. 
12.1 Think about caching with PGVector to return vector data from DB

- [x] **13. Discover Route with PostGIS Distance Filtering** ✓ COMPLETE
  - Analyzed go-ai-poi-client and go-ai-poi-server discover implementations
  - Implemented location-based discovery with PostGIS ST_Distance and ST_DWithin queries
  - Created HTMX-based distance filter dropdown (1km, 5km, 10km, 15km, 25km, 50km, 75km, 100km)
  - Solution:
    - **Handler Implementation** (`app/pkg/handlers/discover.go:237-302`):
      - `GetNearbyPOIs()` - Accepts lat, lon, radius (km), optional category as query params
      - Converts radius from kilometers to meters for PostGIS compatibility
      - Uses existing `GetPOIsByLocationAndDistance()` and `GetPOIsByLocationAndDistanceWithCategory()` repository methods
      - Returns HTML fragments with POI cards showing distance, rating, category, description
    - **PostGIS Queries** (Already existed in `app/pkg/domain/poi/poi_repository.go:2142-2200`):
      - `ST_DWithin()` - Efficient indexed search within radius
      - `ST_Distance()` - Calculates exact distance in meters using geography type
      - `ST_SetSRID(ST_MakePoint(lon, lat), 4326)` - Creates WGS84 point geometry
      - Results ordered by distance ASC, limited to 50 POIs
    - **Template Implementation** (`app/internal/features/discover/discover_nearby.templ`):
      - Alpine.js state management for lat/lon/radius (default: Lisbon 38.7223, -9.1393)
      - HTMX `hx-get="/discover/nearby"` with automatic triggers on distance change
      - Geolocation API integration with "Use My Location" button
      - Responsive grid layout for results display
    - **Route Registration** (`app/routes/routes.go:758`):
      - `htmxGroup.GET("/discover/nearby", discoverHandlers.GetNearbyPOIs)`
      - Handler initialized with `poiRepo` and logger dependencies
  - Features:
    - Distance radius selector: 1, 5, 10 (default), 15, 25, 50, 75, 100 km
    - Manual lat/lon input with 4 decimal precision
    - Browser geolocation support with fallback
    - Category filtering (optional query parameter)
    - Real-time HTMX updates without page reload
    - Proper error handling and empty state messaging
  - Files modified/created:
    - `app/pkg/handlers/discover.go` - Added GetNearbyPOIs handler, helper functions
    - `app/internal/features/discover/discover_nearby.templ` - New distance filter section
    - `app/routes/routes.go` - Updated handler initialization and route registration
  - Result: Full feature parity with go-ai-poi-server discover functionality using HTMX and PostGIS 

13.0
The chat bubble is not appearing on /itinerary /restaurant /activity / hotels
it should be visible to the user and when the user clicks on it, open a modal with the chat continuity session. (this was working before)

13.1 Discover when clicking for results is returning something like:
```bash
  2025-10-25T10:39:22.102+0100    INFO    handlers/discover.go:57 Discovery search requested      {"port": "8090", "service": "loci-templui", "line": "42", "query": "5 star restaurant", "location": "Berlin", "user": "f04eaf02-e1fa-4bbe-b4f6-506767a4fa8d", "ip": "::1"}
2025/10/25 10:39:22 INFO Calling LLM for discover search query="5 star restaurant" location=Berlin
2025-10-25T10:39:29.647+0100    INFO    routes/routes.go:832    404 - Page not found    {"port": "8090", "service": "loci-templui", "line": "42", "path": "/metrics", "method": "GET", "ip": "127.0.0.1"}
2025-10-25T10:39:29.648+0100    WARN    middleware/middleware.go:76     HTTP Request    {"port": "8090", "service": "loci-templui", "line": "42", "method": "GET", "path": "/metrics", "ip": "127.0.0.1", "status": 404, "latency": 0.000995792, "user_agent": "Prometheus/3.7.2"}
2025/10/25 10:39:31 traces export: parse "http://http:%2F%2Fotel-collector:4318/v1/traces": invalid URL escape "%2F"
2025/10/25 10:39:32 ERROR Failed to parse LLM response error="invalid character '`' looking for beginning of value" response="```json\n{\n    \"results\": [\n        {\n            \"name\": \"Facil\",\n            \"latitude\": 52.5078,\n            \"longitude\": 13.3663,\n            \"category\": \"Restaurant\",\n            \"description\": \"Awarded two Michelin stars, Facil offers modern, innovative cuisine in a sophisticated setting with panoramic views of Berlin. The restaurant is known for its exceptional service and exquisite tasting menus, making it a top choice for a 5-star dining experience.\",\n            \"address\": \"Potsdamer Straße 3, 10785 Berlin, Germany\",\n            \"website\": \"https://www.facil.de/\",\n            \"phone_number\": \"+49 30 590051234\",\n            \"opening_hours\": \"Tuesday-Saturday 6:30 PM - 11:00 PM\",\n            \"price_level\": \"$$$$\",\n            \"rating\": 4.7,\n            \"tags\": [\"Michelin Star\", \"Fine Dining\", \"Modern Cuisine\", \"Luxury\"],\n            \"images\": [],\n            \"cuisine_type\": \"Modern European\",\n            \"star_rating\": null\n        },\n        {\n            \"name\": \"Lorenz Adlon Esszimmer\",\n            \"latitude\": 52.5171,\n            \"longitude\": 13.3788,\n            \"category\": \"Restaurant\",\n            \"description\": \"Located in the Adlon Kempinski Hotel, Lorenz Adlon Esszimmer boasts two Michelin stars and offers refined European cuisine with a focus on seasonal ingredients. The elegant ambiance and impeccable service contribute to a truly luxurious 5-star dining experience. It is known for its classic dishes with a modern twist.\",\n            \"address\": \"Unter den Linden 77, 10117 Berlin, Germany\",\n            \"website\": \"https://www.kempinski.com/en/berlin/hotel-adlon/restaurants-bars/lorenz-adlon-esszimmer/\",\n            \"phone_number\": \"+49 30 22611960\",\n            \"opening_hours\": \"Tuesday-Saturday 6:30 PM - 10:00 PM\",\n            \"price_level\": \"$$$$\",\n            \"rating\": 4.8,\n            \"tags\": [\"Michelin Star\", \"Fine Dining\", \"European Cuisine\", \"Luxury\", \"Hotel Restaurant\"],\n            \"images\": [],\n            \"cuisine_type\": \"European\",\n            \"star_rating\": null\n        },\n        {\n            \"name\": \"Nobelhart & Schmutzig\",\n            \"latitude\": 52.5181,\n            \"longitude\": 13.3932,\n            \"category\": \"Restaurant\",\n            \"description\": \"Nobelhart & Schmutzig, a Michelin-starred restaurant, offers a unique culinary experience focused on regional German ingredients. The tasting menu is a journey through the terroir of Berlin and Brandenburg, presented in a minimalist and sophisticated setting. It's a 5-star experience for adventurous palates seeking authenticity and innovation.\",\n            \"address\": \"Friedrichstraße 218, 10969 Berlin, Germany\",\n            \"website\": \"https://nobelhartundschmutzig.com/en/\",\n            \"phone_number\": \"+49 30 25940610\",\n            \"opening_hours\": \"Tuesday-Saturday 7:00 PM - 12:00 AM\",\n            \"price_level\": \"$$$$\",\n            \"rating\": 4.6,\n            \"tags\": [\"Michelin Star\", \"Fine Dining\", \"Regional Cuisine\", \"Luxury\", \"German Cuisine\"],\n            \"images\": [],\n            \"cuisine_type\": \"German\",\n            \"star_rating\": null\n        },\n        {\n            \"name\": \"Restaurant Tim Raue\",\n            \"latitude\": 52.5081,\n            \"longitude\": 13.3931,\n            \"category\": \"Restaurant\",\n            \"description\": \"Restaurant Tim Raue, a renowned two-Michelin-starred establishment, features Asian-inspired cuisine crafted with precision and artistry. The restaurant's elegant atmosphere and impeccable service contribute to its 5-star rating. It is celebrated for its bold flavors and innovative dishes.\",\n            \"address\": \"Rudi-Dutschke-Straße 26, 10969 Berlin, Germany\",\n            \"website\": \"https://tim-raue.com/en/\",\n            \"phone_number\": \"+49 30 25937930\",\n            \"opening_hours\": \"Tuesday-Saturday 6:30 PM - 11:00 PM\",\n            \"price_level\": \"$$$$\",\n            \"rating\": 4.7,\n            \"tags\": [\"Michelin Star\", \"Fine Dining\", \"Asian Cuisine\", \"Luxury\"],\n            \"images\": [],\n            \"cuisine_type\": \"Asian Fusion\",\n            \"star_rating\": null\n        },\n        {\n            \"name\": \"Skykitchen\",\n            \"latitude\": 52.5105,\n            \"longitude\": 13.4486,\n            \"category\": \"Restaurant\",\n            \"description\": \"Located on the 12th floor of the andel's Hotel, Skykitchen offers modern European cuisine with breathtaking panoramic views of Berlin. The restaurant is known for its creative dishes, stylish ambiance, and attentive service, providing a 5-star dining experience with a unique perspective.\",\n            \"address\": \"Landsberger Allee 106, 10369 Berlin, Germany\",\n            \"website\": \"https://www.skykitchen.berlin/en/\",\n            \"phone_number\": \"+49 30 4530532620\",\n            \"opening_hours\": \"Tuesday-Saturday 6:30 PM - 10:00 PM\",\n            \"price_level\": \"$$$\",\n            \"rating\": 4.5,\n            \"tags\": [\"Fine Dining\", \"Modern Cuisine\", \"European Cuisine\", \"Luxury\", \"Panoramic View\"],\n            \"images\": [],\n            \"cuisine_type\": \"European\",\n            \"star_rating\": null\n        },\n        {\n            \"name\": \"Hugos Restaurant\",\n            \"latitude\": 52.5043,\n            \"longitude\": 13.3647,\n            \"category\": \"Restaurant\",\n            \"description\": \"Hugos Restaurant, nestled atop the InterContinental Berlin, provides an exquisite fine-dining experience. With one Michelin star, it offers refined European cuisine paired with stunning city views. The elegant ambiance and impeccable service solidify its 5-star reputation.\",\n            \"address\": \"Budapester Straße 2, 10787 Berlin, Germany\",\n            \"website\": \"https://www.hugos-restaurant.de/en/\",\n            \"phone_number\": \"+49 30 26021263\",\n            \"opening_hours\": \"Tuesday-Saturday 6:30 PM - 10:00 PM\",\n            \"price_level\": \"$$$$\",\n            \"rating\": 4.6,\n            \"tags\": [\"Michelin Star\", \"Fine Dining\", \"European Cuisine\", \"Luxury\", \"Panoramic View\"],\n            \"images\": [],\n            \"cuisine_type\": \"European\",\n            \"star_rating\": null\n        }\n    ]\n}\n```"
2025-10-25T10:39:32.224+0100    ERROR   middleware/middleware.go:74     HTTP Request    {"port": "8090", "service": "loci-templui", "line": "42", "method": "POST", "path": "/discover/search", "ip": "::1", "status": 500, "latency": 10.122053, "user_agent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/141.0.0.0 Safari/537.36"}
2025/10/25 10:39:36 traces export: parse "http://http:%2F%2Fotel-collector:4318/v1/traces": invalid URL escape "%2F"
2025-10-25T10:39:37.798+0100    INFO    routes/routes.go:832    404 - Page not found    {"port": "8090", "service": "loci-templui", "line": "42", "path": "/chat/continue/e9bb9201-e24f-444b-b8ee-423cd4a3b6d2", "method": "GET", "ip": "::1"}
2025-10-25T10:39:37.799+0100    WARN    middleware/middleware.go:76     HTTP Request    {"port": "8090", "service": "loci-templui", "line": "42", "method": "GET", "path": "/chat/continue/e9bb9201-e24f-444b-b8ee-423cd4a3b6d2", "ip": "::1", "status": 404, "latency": 0.000345458, "user_agent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/141.0.0.0 Safari/537.36"}
2025/10/25 10:39:41 traces export: parse "http://http:%2F%2Fotel-collector:4318/v1/traces": invalid URL escape "%2F" 
```

13.2 I need you to go to go-ai-poi-server (cd ..) and analyse the routes:
```go
// Unified chat endpoints - more specific routes first
	r.Post("/prompt-response/chat/sessions/stream/{profileID}", HandlerImpl.StartChatMessageStream)
	r.Post("/prompt-response/chat/sessions/{sessionID}/continue", HandlerImpl.ContinueChatSessionStream) // POST http://localhost:8000/api/v1/llm/prompt-response/chat/sessions/{sessionID}/continue
```
on go-ai-poi-server we are storing LLM data either by LLM_interactions or LLM_Siggested_pois
We should do the same because in the end we want to have a query that makes a search first on the database. 
Either search first with PGVector and see if we find the data first on the DB, and if not, return from LLM.
User searches Itinerary ABC + Preference 1,2,3 = LLM Search, Save on DB (no need to block the search, can save on the background with a go routine).
User searches Itinerary ABC + Preference 1,2,3 = Checks DB, exists, returns from DB. 
User searches Itinerary ABC + Preference 1,2,4 = New preference, New Itinerary, Check DB, wont exist, Save on DB (no need to block the search, can save on the background with a go routine).
Repeat

13.3 I need you to go to go-ai-poi-server (cd ..) and analyse the routes:
```go
// Unified chat endpoints - more specific routes first
	r.Post("/prompt-response/chat/sessions/stream/{profileID}", HandlerImpl.StartChatMessageStream)
	r.Post("/prompt-response/chat/sessions/{sessionID}/continue", HandlerImpl.ContinueChatSessionStream) // POST http://localhost:8000/api/v1/llm/prompt-response/chat/sessions/{sessionID}/continue
```
I want you to analyse the old API r.Post("/prompt-response/chat/sessions/{sessionID}/continue" and check how its working.
When I continue this chat I'm having this error:
```bash
Lisbon city does not exist
```
This could be due to the 13.1 error or the chat not having the chat sessionID but go-ai-poi-server is working properly and so should this. And on the chat we should have proper text and not json data even if there is an error


14. Similar to Discover, I have a new view called nearby. The nearby should update new views when user locations changes without the user needing to update this. 
Use websockets, htmx and whatever needed. The idea is the user to be able to walk, and when the coordinates of the user changes, the AI generates new points of interest around the user dynamically.
Elaborate on the effort for this and build the UI and Service logic. 

15. The Recents page should have all the recents itineraries, restaurants, hotels, activities or chats built and should be able to sort by date
All this data should be available on the Database so for recents only queries and displays should be needed.

16. The chat view should work as it does under go-ai-poi-client and go-ai-poi-server but improved. The chat should be able to return an intent data (restaurant, itinerary, hotel, activity) and upon writing "Add" or "Remove" should add to the current context more points and writing "New" on chat creates a new conversation with the Agent.
17. Favourites tab should contain all the favourited items from the database. Search and filter by type should work and be implemented. Pagination as well.  
18. Bookmarks tab should contain all the bookmarked itineraries from the database. Search and filter by type should work and be implemented. Pagination as well. 
19. Lists should also work as it does under go-ai-poi-client and go-ai-poi-server. On a list an user should be able to create a new one and be able to add diverse activities already searched from the database. 
To discuss: A list should be a mix of itineraries, hotels, activities and restaurants or only itineraries?
20. The settings page should be working like go-ai-poi-client and go-ai-poi-server saving on the database the data from the tabs:
    20.1 Account Settings
    20.2 Travel Preferences
    20.3 Privacy and Security
    20.4 Notifications
21. On Pricing, add an "Enterprise column" where companies and travel agencies should be able to contact me for a one time purchase of the whole pack. 
22. Adapt the footer for Loci instead of the current hardcodded TemplUI
23. Currently the Auth is too simplistic. We need to add email verification and sms verification when an user Signs up instead of redirecting right away to the app. The login is fine. 
24. 2025-10-24T11:36:08.953+0100    ERROR   auth/handlers.go:164    Failed to register user {"port": "8090", "service": "loci-templui", "line": "42", "error": "registration failed: email or username already exists: item already exists or conflict"} On error, the user should see an error message on the form instead of the constant loading button. Also when creating account, only one message should be on the button. "Creating account" with the loader. The state still shows the loader indicator with the label.
25. On the dashboard there is a settings icon. The settings should either be:
    25.1 A popup with the same settings for /settings for the user to be able to modify fast the travel preferences
    25.2 (Preferences) a side bar where the user gets the same views of the Travel Preferences in /settings to be able to modify the settings fast before a search.
26. On Dashboard there is hardcodded data for Saved Places, Itineraries, Cities Explored and Discoveries.
    26.1 Saved places should be the count of favourited items (itineraries, hotels, activities and )
    26.2 Itineraries should be the count of bookmarked itineraries
    26.3 Should be the count of cities explored
    26.4 Discoveries should be the number of times user finds place under Discover tab
If the queries for each item doesn't exist, create them. 
27.   When I make a search for itinerary, everything works fine.
      But when I make a retrocess on the page, we still have the state of the previous page.
      Upon coming back to the main page, we should have a new state instead of having
      "Redirecting to results..." under the input and have the terminal under an eternal search. 
28. Clicking on each favorite item on the restaurant, hotel and activity card should add the item to the favorite table on the DB with a visual indicator. 
29. When clicking on "My itineraries" on the dashboard, it redirects to /itinerary which is a travel planner. 
The travel planner is a good idea and should be its own tab on nav. After that, "My itineraries" should just go to a new view with all the bookmarked itineraries on the app.
After that we should make the logic to work on the travel planner. It should follow the logic of the rest of the LLM calls of this app.
30. [x] **Fixed Missing Default Profile on User Registration** ✓ COMPLETE
   - Issue: Users couldn't generate itineraries due to "No default profile found. Please create a profile first." error
   - Root cause: User registration (`Register` function) only created user record without creating a default search profile
   - Solution: Modified `Register` function in `auth_repository.go` to use database transaction
   - Implementation details:
     - Uses database transaction to ensure atomicity (both user and profile created or neither)
     - Creates user and default profile in same transaction
     - Default profile settings:
       - Name: "Default Profile"
       - Budget level: "medium"
       - Accessibility needs: empty array `{}`
       - Dietary restrictions: empty array `{}`
       - is_default: true
     - Automatic rollback on failure ensures data consistency
     - Enhanced error logging and tracing
   - Files modified:
     - `app/pkg/domain/auth/auth_repository.go` (lines 112-172)
   - Result: New users automatically get a default profile, allowing immediate itinerary generation without manual profile creation
31. [x] **Fixed PostgreSQL pg_trgm Extension Error** ✓ COMPLETE
   - Issue: Chat continue session failing with error: `ERROR: function similarity(text, unknown) does not exist (SQLSTATE 42883)`
   - Root cause: Missing `pg_trgm` extension in PostgreSQL database required for fuzzy city name matching
   - Error occurred in: `FindCityByFuzzyName` function in `city_repository.go` (lines 166-167)
   - Solution: Added `CREATE EXTENSION IF NOT EXISTS pg_trgm;` to initial database setup migration
   - Implementation:
     - Modified `app/db/migrations/0001_setup.up.sql` (line 16)
     - Added pg_trgm extension alongside other required extensions (postgis, vector, etc.)
     - Extension provides `similarity()` function for trigram-based fuzzy text matching
   - Files modified:
     - `app/db/migrations/0001_setup.up.sql` (line 16)
   - Result: City lookup by fuzzy name now works, allowing chat to find cities like "Lisbon" and continue sessions
   - Action required: Run migrations or manually execute `CREATE EXTENSION IF NOT EXISTS pg_trgm;` on existing database

32. Im using pprof with Gin. Check main.go and create a setup to be able to use graphviz with pprof locally and in prod. Add the commands to makefile if necessary and documentation under docs if necessary as well.
32. Review all my observability stack. Right now I use everything locally through my compose and want to test locally. In the future, I want to have my observability stack also working in prod hosted on fly.io or on hetzner. Make changes if needed (my services run through docker compose but my app runs locally through air, keep that in mind) and write docs if necessary to document how I will host my observability stack on Fly, DO or Hetzner. 
33. Review my project structure and work done and check where Gorilla Websockets could be used in this project. Write a document under docs to have that documented before the implementation.
34. Add recover system and retries on LLM calls in case something happens with the API call so the user has more uptime and security to avoid cases where the LLM makes the call but network problems or other issues make thing like:
```bash
5,\n    \"longitude\": -9.1406,\n    \"category\": \"Pastry Shop\",\n    \"description\": \"Another excellent place to try Pastéis de Nata, with a focus on quality and freshness. Watch the pastries being made in the open kitchen.\",\n    \"address\": \"Rua do Loreto 2, 1200-242 Lisboa, Portugal\",\n    \"website\": \"https://www.manteigaria.com/\",\n    \"phone_number\": null,\n    \"opening_hours\": \"Mon-Sun 08:00-23:00\",\n    \"price_level\": \"$\",\n    \"cuisine_type\": \"Pastries\",\n    \"tags\": [\n      \"Pastries\",\n      \"Portuguese\",\n      \"Fresh\",\n      \"Open Kitchen\"\n    ],\n    \"images\": [],\n    \"rating\": 4.7,\n    \"llm_interaction_id\": \"00000000-0000-0000-0000-000000000000\"\n  }\n]"
2025/10/25 09:20:10 INFO Caching restaurant data sessionID=0feae1cb-1511-44f6-a591-4c1e103e33a3 cacheKey=08ccf4fb6fcc7ea824ab55382b9dfe00 restaurantsCount=10
panic: runtime error: invalid memory address or nil pointer dereference
[signal SIGSEGV: segmentation violation code=0x2 addr=0x0 pc=0x105c06480]

goroutine 1505 [running]:
github.com/FACorreiaa/go-templui/app/pkg/domain/chat_prompt.(*ServiceImpl).parseCompleteResponseFromParts(0x140000f69a0, 0x14000628570, {0xf, 0xea, 0xe1, 0xcb, 0x15, 0x11, 0x44, 0xf6, ...})
        /Users/fernando_idwell/Projects/Loci/go-templui/app/pkg/domain/chat_prompt/chat_parser.go:122 +0x900
github.com/FACorreiaa/go-templui/app/pkg/domain/chat_prompt.(*ServiceImpl).cacheResultsIfAvailable(0x140000f69a0, {0x10666a050, 0x14000783560}, {0xf, 0xea, 0xe1, 0xcb, 0x15, 0x11, 0x44, ...}, ...)
        /Users/fernando_idwell/Projects/Loci/go-templui/app/pkg/domain/chat_prompt/chat_service.go:2767 +0x1bb0
github.com/FACorreiaa/go-templui/app/pkg/domain/chat_prompt.(*ServiceImpl).ProcessUnifiedChatMessageStream.func11()
        /Users/fernando_idwell/Projects/Loci/go-templui/app/pkg/domain/chat_prompt/chat_service.go:2194 +0x1f0
created by github.com/FACorreiaa/go-templui/app/pkg/domain/chat_prompt.(*ServiceImpl).ProcessUnifiedChatMessageStream in goroutine 1495
        /Users/fernando_idwell/Projects/Loci/go-templui/app/pkg/domain/chat_prompt/chat_service.go:2168 +0x1c74
exit status 2
```
35. 