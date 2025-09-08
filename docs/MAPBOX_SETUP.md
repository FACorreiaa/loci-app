# Mapbox Integration Setup

This application includes Mapbox GL JS integration for interactive maps in both the Nearby and Itinerary features.

## Setup Instructions

### 1. Get a Mapbox Access Token

1. Go to [https://account.mapbox.com/](https://account.mapbox.com/)
2. Sign up for a free account or sign in
3. Go to your [Access Tokens](https://account.mapbox.com/access-tokens/) page
4. Copy your default public token

### 2. Configure the Access Token

You have several options to set your Mapbox access token:

#### Option A: Environment Variable (Recommended for production)
```bash
export MAPBOX_ACCESS_TOKEN="pk.eyJ1IjoieW91ci11c2VybmFtZSIsImEiOiJjbGF1ZGUtY29kZSJ9.your-actual-token"
```

#### Option B: Update the configuration file
Edit `/app/lib/components/map/mapbox.templ` and replace the placeholder token:
```javascript
window.MAPBOX_ACCESS_TOKEN = 'your-actual-mapbox-token-here';
```

### 3. Features Included

#### Nearby Feature (`/nearby`)
- **Location-based POI search** with geolocation support
- **Interactive map** with custom markers for different POI categories
- **Multiple view modes**: Cards, Map, Split-view
- **Real-time filtering** by distance, rating, and price
- **Category-based tabs**: Restaurants, Activities, Hotels, Attractions
- **Click-to-add** functionality for favorites and itineraries

#### Itinerary Feature (`/itinerary`)
- **Split-screen layout** with chat interface and interactive map
- **AI-powered trip planning** with contextual responses
- **Destination search** with autocomplete suggestions
- **Dynamic map updates** based on selected destination
- **Route visualization** between selected POIs
- **Layer controls** for different POI types (attractions, restaurants, hotels, transport)

### 4. Map Features

- **Custom markers** with category-specific icons (Font Awesome)
- **Interactive popups** with POI details, ratings, and action buttons
- **Smooth animations** and transitions
- **Responsive design** that works on all screen sizes
- **CSP compliance** with proper security headers

### 5. Customization

#### Map Styles
You can change the map style by modifying the default style in `/app/lib/components/map/config.go`:
```go
DefaultStyle: "mapbox://styles/mapbox/streets-v12"
```

Available styles:
- `mapbox://styles/mapbox/streets-v12` (default)
- `mapbox://styles/mapbox/outdoors-v12`
- `mapbox://styles/mapbox/light-v11`
- `mapbox://styles/mapbox/dark-v11`
- `mapbox://styles/mapbox/satellite-v9`
- `mapbox://styles/mapbox/satellite-streets-v12`

#### Marker Icons
Customize marker icons by editing the `getCategoryIcon` function in `/app/lib/components/map/mapbox.templ`.

### 6. Development Notes

- The application includes a **fallback token** for development
- **CSP headers** are configured to allow Mapbox resources
- **Font Awesome icons** are included for marker categories
- Maps are **responsive** and work in both full-screen and split-view modes

### 7. Production Considerations

1. **Set a real Mapbox token** via environment variables
2. **Configure rate limiting** for map API calls
3. **Enable proper error handling** for network issues
4. **Consider caching** geocoding results
5. **Monitor Mapbox usage** to stay within free tier limits

The integration follows Mapbox best practices and includes proper error handling, accessibility features, and performance optimizations.