# Nearby POI Feature - Implementation Summary

## Overview
Real-time location-based point of interest discovery using WebSockets and AI-powered recommendations.

## Features
- ✅ Real-time location tracking via browser Geolocation API
- ✅ WebSocket-based bidirectional communication
- ✅ AI-powered POI recommendations using Google Gemini
- ✅ Distance-based update throttling (50 meters)
- ✅ Automatic reconnection with exponential backoff
- ✅ Configurable search radius (0.5km - 10km)
- ✅ Dynamic POI grid display with categories and ratings

## Architecture

### Frontend (`/app/internal/features/nearby/nearby.templ`)
- **Framework**: Alpine.js for reactive UI
- **WebSocket Client**: Connects to `/ws/nearby`
- **Geolocation**: Browser Geolocation API with `watchPosition`
- **Features**:
  - Connection status indicator
  - Live tracking toggle
  - Location display (lat/lon)
  - Search radius selector
  - POI grid with dynamic updates

### Backend (`/app/pkg/handlers/nearby.go`)
- **WebSocket Handler**: Manages connections and message routing
- **Connection Pool**: Thread-safe with `sync.RWMutex`
- **AI Integration**: Uses chat service to generate POI recommendations
- **Response Types**:
  - `LocationUpdate`: Client → Server (coordinates + radius)
  - `WebSocketMessage`: Server → Client (POIs or error)

### AI Service (`/app/pkg/domain/llmChat/chat_service.go`)
- **Method**: `GenerateNearbyPOIs`
- **Model**: Google Gemini 2.0 Flash
- **Output**: JSON array of 5-10 diverse POIs with:
  - ID, name, category, description
  - Emoji, rating, coordinates
  - Distance from user location

## API Endpoints

### HTTP Routes
- `GET /nearby` - Main nearby page
- `GET /ws/nearby` - WebSocket endpoint

### WebSocket Protocol

#### Client → Server
```json
{
  "latitude": 38.7223,
  "longitude": -9.1393,
  "radius": 5.0
}
```

#### Server → Client (Success)
```json
{
  "type": "pois",
  "pois": [
    {
      "id": "poi_123",
      "name": "Example Restaurant",
      "category": "restaurant",
      "description": "Amazing local cuisine",
      "emoji": "🍽️",
      "rating": 4.5,
      "distance": 0.35,
      "latitude": 38.7225,
      "longitude": -9.1395
    }
  ]
}
```

#### Server → Client (Error)
```json
{
  "type": "error",
  "message": "Failed to get nearby places"
}
```

## Testing

### Manual Testing Steps
1. Start the server: `go run main.go`
2. Navigate to `http://localhost:PORT/nearby`
3. Click "Start Live Tracking"
4. Allow browser location permissions
5. Verify:
   - WebSocket connection status shows "Live"
   - Current coordinates are displayed
   - POIs are loaded automatically
   - Changing radius updates results
   - Moving 50+ meters triggers new POI fetch

### Expected Behavior
- Initial connection establishes WebSocket
- First location update triggers POI generation
- POIs display in grid with emoji, name, category, rating, distance
- Subsequent location changes (>50m) trigger updates
- Connection status indicator reflects WebSocket state
- Automatic reconnection on disconnect (max 5 attempts)

## Implementation Details

### Distance Calculation
Uses Haversine formula to calculate distance between GPS coordinates:
```javascript
calculateDistance(lat1, lon1, lat2, lon2) {
  const R = 6371; // Earth's radius in km
  const dLat = (lat2 - lat1) * Math.PI / 180;
  const dLon = (lon2 - lon1) * Math.PI / 180;
  const a = Math.sin(dLat/2) * Math.sin(dLat/2) +
    Math.cos(lat1 * Math.PI / 180) * Math.cos(lat2 * Math.PI / 180) *
    Math.sin(dLon/2) * Math.sin(dLon/2);
  const c = 2 * Math.atan2(Math.sqrt(a), Math.sqrt(1-a));
  return R * c;
}
```

### Location Update Throttling
```javascript
// Only update if moved more than 50 meters
const distance = this.calculateDistance(this.lastLat, this.lastLon, newLat, newLon);
if (distance > 0.05) { // 50 meters
  this.currentLat = newLat;
  this.currentLon = newLon;
  this.lastLat = newLat;
  this.lastLon = newLon;
  this.sendLocationUpdate();
}
```

### Reconnection Logic
```javascript
// Exponential backoff with max 5 attempts
if (this.reconnectAttempts < this.maxReconnectAttempts) {
  this.reconnectAttempts++;
  const delay = Math.min(1000 * Math.pow(2, this.reconnectAttempts), 30000);
  setTimeout(() => this.connectWebSocket(), delay);
}
```

## Files Modified/Created

### Created
- `/app/pkg/handlers/nearby.go` - WebSocket handler
- `/app/pkg/domain/llmChat/chat_service.go` - Added `GenerateNearbyPOIs` method

### Modified
- `/app/internal/features/nearby/nearby.templ` - Complete rewrite with WebSocket support
- `/app/routes/routes.go` - Added WebSocket route and handler initialization
- `/app/pkg/domain/poi/poi_repository.go` - Added `GetFavouritesFiltered` stub method

## Dependencies
- `github.com/gorilla/websocket` - WebSocket implementation
- `google.golang.org/genai` - Google Gemini AI
- Alpine.js (CDN) - Reactive UI framework

## Configuration
- **Default Location**: Lisbon (38.7223, -9.1393)
- **Update Threshold**: 50 meters
- **Reconnect Attempts**: 5 max
- **WebSocket Buffer**: 1024 bytes read/write
- **Default Radius**: 5 km
- **Radius Options**: 0.5, 1, 2, 5, 10 km

## Security Considerations
- ✅ WebSocket origin check (currently allows all - should be restricted in production)
- ✅ Context-aware request handling
- ✅ Thread-safe connection management
- ✅ User location history tracking
- ✅ POI interaction analytics
- ⚠️ TODO: Add rate limiting for location updates
- ⚠️ TODO: Add authentication for WebSocket connections (optional - can use middleware)
- ⚠️ TODO: Validate coordinate bounds

## Performance Optimizations
- ✅ Distance-based throttling reduces API calls
- ✅ Connection pooling with efficient cleanup
- ✅ JSON streaming for AI responses
- ✅ Fallback JSON extraction for malformed responses
- ⚠️ TODO: Add caching for frequently requested locations
- ⚠️ TODO: Implement POI result pagination

## Database Schema

### location_history table
- Stores user location updates
- Indexed by user_id and timestamp
- Includes latitude, longitude, radius, and timestamp

### poi_interactions table
- Tracks user interactions with POIs (view, click, favorite)
- Stores user and POI coordinates for distance analysis
- Indexed by user_id, poi_id, category, and interaction_type
- Enables analytics on user preferences and behavior

## Known Issues / TODOs
1. WebSocket CORS configuration should be environment-specific
2. No authentication/authorization on WebSocket endpoint (can use middleware)
3. No rate limiting on location updates (optional - can be added if needed)
4. Backend API endpoint in go-ai-poi-server not yet implemented
5. ✅ User location history now stored in database
6. No offline support
7. ✅ POI interaction analytics implemented

## Next Steps
1. Add rate limiting middleware (optional)
2. Add POI result caching with Redis (performance optimization)
3. Create backend API endpoint in go-ai-poi-server
4. Add comprehensive error handling and logging
5. Implement E2E tests with WebSocket testing
6. Add analytics dashboard for visualizing location and interaction data
7. Implement geospatial queries for finding nearby users
