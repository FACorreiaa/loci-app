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

- [ ] **3. Infinite Loop in Itinerary Search**
  - Fix looping behavior in itinerary search
  - Issues:
    - Restaurants: breaking with no data
    - Activities: stuck on "Finding activities... This may take a few moments"
    - Hotels: stuck on loading state
  - This used to work - regression investigation needed

- 3.1 Optimise the services that are using 
## Feature Parity with SolidJS Project
- [ ] **4. Compare with go-ai-poi-client**
  - Navigate to `../go-ai-poi-client`
  - Compare `/itinerary`, `/restaurant`, `/activity`, `/hotel` implementations
  - Document missing features/UI differences
  - Implement missing features in current project

- [ ] **5. Map Integration**
  - Implement working map functionality
  - Match map behavior from SolidJS project
  - Note: Env variables will be fixed separately

## Verification Steps
After fixing each issue:
1. Test the specific endpoint/feature
2. Verify cache behavior with same/different parameters
3. Check browser console for errors
4. Verify data flow end-to-end
5. Test edge cases (no results, errors, etc.)
