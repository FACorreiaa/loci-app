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

- [ ] **2. Cache System Verification**
  - Verify cache implementation for endpoints
  - Cache key should be: `city + user_preference`
  - Rules:
    - Same city + different preference = NEW request (no cache)
    - Same city + same preference = CACHED request
  - Current implementation: go-cache
  - Endpoints to check: `/itinerary`, `/restaurant`, `/activity`, `/hotel`

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

### Missing Features from SolidJS Implementation

#### High Priority (Core Functionality)
1. **Map Integration** - CRITICAL (some already implemented)
   - Full Mapbox GL integration with route visualization
   - Marker clustering and styling
   - Route optimization algorithm connecting POIs
   - Dynamic marker updates based on filters
   - View modes: Map/List/Split view
   - Markers with same number as the numbers on the list

2. **City Information Panel**
   - General city data display (population, language, weather, timezone, area)
   - City description with metadata stats
   - Quick facts section with icons

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

## Verification Steps
After fixing each issue:
1. Test the specific endpoint/feature
2. Verify cache behavior with same/different parameters
3. Check browser console for errors
4. Verify data flow end-to-end
5. Test edge cases (no results, errors, etc.)
