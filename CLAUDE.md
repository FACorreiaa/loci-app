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

3. **Advanced Filtering System**
   - Consistent filter panel across all result pages
   - Multi-select filters with checkboxes
   - Price range filters (€ to €€€€)
   - Rating-based filtering
   - Category filters specific to each domain:
     - Restaurants: Cuisine type (Portuguese, Seafood, International)
     - Hotels: Type (Luxury, Boutique, Historic, Business)
     - Activities: Type (Cultural, Entertainment, Outdoor, Adventure)
   - Feature filters (Michelin Star, River View, Pet Friendly, etc.)
   - Filter badges showing active filters

4. **Enhanced Card Design**
   - Consistent emoji and icon system per domain
   - Metadata label badges with color coding
   - Priority indicators (Must Visit, Recommended, Popular)
   - Enhanced footer with price/budget and feature tags
   - Duration/time-to-spend indicators
   - Special feature badges (Vegetarian, Terrace, Family-friendly)

#### Medium Priority (UX Improvements)
5. **General POIs Section**
   - "All Points of Interest" section showing broader city context
   - Separate from curated/personalized recommendations
   - Collapsible "show all" with pagination
   - Grid layout with responsive columns

6. **Session Management & Deep Linking**
   - URL parameter support for sessionId and cityName
   - Deep linking to specific results
   - Session data persistence in sessionStorage
   - Fallback session creation if expired

7. **Favorites/Bookmarking System**
   - Save to user account via API
   - Toggle favorite state with loading indicator
   - Heart icon with fill animation
   - Favorites count in results header
   - Sign-in required messaging for guests

8. **Enhanced View Modes**
   - Split view (map + list side-by-side on desktop)
   - Smooth transitions between view modes
   - Responsive stacking on mobile
   - View mode persistence

#### Low Priority (Nice-to-have)
9. **Chat Session Continuity**
   - Persistent session IDs across page navigation
   - Multi-location session ID extraction
   - Continue conversation after returning to page

10. **Banner Messaging System**
    - Success banners after operations
    - Authentication promotion for guests
    - Typing animation for dynamic text
    - Dismissable with X button
    - Color-coded by context

11. **Coordinate Validation**
    - Validate lat/long for map display
    - Filter invalid POIs from map
    - Show "Location TBD" for cards without coordinates
    - Fallback center point for invalid data

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

- [] 3.2 When I make a search for itinerary, everything works fine. 
But when I make a retrocess on the page, we still have the state of the previous page. 
Upon coming back to the main page, we should have a new state instead of having 
"Redirecting to results..." under the input and have the terminal under an eternal search. 

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

- [ ] **6. Metadata Enhancement**
  - Add more metadata to the itinerary results to be attached on the card for better UI and user interaction
  - Also apply to restaurants, hotels and activities

- [x] **7. Empty IDs in complete_itinerary.json**
  - Issue: complete_itinerary.json is returning empty `ids` and empty `city_ids`
  - These should be filled properly with actual database IDs 

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

  9.2. [] I want you to start preparing the /continue transcation that should be used on this server side rendering with HTMX. The /continue has the same functionality as the REST API: 
  - On full pages allows to "Add" or "Remove" items from the page refreshing the page 
  - On Chats it allows the user to do the same with the LLM replying with the new data and the previous data. If necessary for this task do cd .. and also analise go-ai-poi-client

10. Analyse my Stripe services and implement the handlers for my business needs. An user can subscribe to the services I offer and eventually have in item purchases but for now I only the subscriptions. 
Build the handlers keeping in mind im using HTMX and Go full stack.

10.1 For the payments I want to have a proper middleware that connects the users with their benefits
10.2 I want to have a way of users having discount codes and special offer codes to be able to use the sub

11. Restaurants has two structs, cityData that is empty and restaurants which has the data including the city name. on the view, the title is being populated with cityData which then is populating the title empty. We need to have either the cityData being populated for Restaurants, Hotels and Activities so the title can be filled or fill the view with the right structure since city data is empty.
12. 
