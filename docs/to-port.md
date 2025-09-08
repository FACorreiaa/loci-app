# UI Components Architecture - Port Guide

This document details the UI component structure for restaurants, itineraries, activities, and hotels pages to enable 1:1 porting to another project.

## Table of Contents
- [Technology Stack](#technology-stack)
- [Route Structure](#route-structure)
- [Core Result Components](#core-result-components)
- [Shared UI Components](#shared-ui-components)
- [Hooks and Utilities](#hooks-and-utilities)
- [External Dependencies](#external-dependencies)
- [Styling System](#styling-system)

## Technology Stack

### Framework & Libraries
- **SolidJS** (`solid-js`) - Reactive framework
- **SolidJS Router** (`@solidjs/router`) - Client-side routing
- **Lucide Solid** (`lucide-solid`) - Icon library
- **Mapbox GL JS** (`mapbox-gl`) - Interactive maps
- **TailwindCSS** - Utility-first CSS framework

### State Management
- SolidJS signals (`createSignal`)
- SolidJS effects (`createEffect`)
- Batch updates (`batch`)

## Route Structure

### File Organization
```
src/routes/
├── restaurants/index.tsx
├── itinerary/index.tsx
├── activities/index.tsx
└── hotels/index.tsx

src/components/results/
├── RestaurantResults.tsx
├── ItineraryResults.tsx
├── ActivityResults.tsx
└── HotelResults.tsx
```

## Core Result Components

### 1. RestaurantResults.tsx (`src/components/results/RestaurantResults.tsx`)

**Purpose:** Displays restaurant listings with ratings, cuisine types, and price ranges.

**Key Imports:**
```tsx
import {
  ChevronDown, ChevronUp, Clock, DollarSign, 
  Heart, MapPin, Share2, Star
} from "lucide-solid";
import { For, Show, createSignal } from "solid-js";
import AddToListButton from "~/components/lists/AddToListButton";
```

**Interface Definition:**
```tsx
interface Restaurant {
  name: string;
  latitude?: number;
  longitude?: number;
  category?: string;
  description_poi?: string;
  address?: string;
  website?: string;
  opening_hours?: string;
  rating?: number;
  price_range?: string;
  cuisine_type?: string;
  distance?: number;
}

interface RestaurantResultsProps {
  restaurants: Restaurant[];
  compact?: boolean;
  limit?: number;
  showToggle?: boolean;
  initialLimit?: number;
  onItemClick?: (restaurant: Restaurant) => void;
  onFavoriteClick?: (restaurant: Restaurant) => void;
  onToggleFavorite?: (restaurantId: string, restaurant: Restaurant) => void;
  onShareClick?: (restaurant: Restaurant) => void;
  favorites?: string[];
  showAuthMessage?: boolean;
  isLoadingFavorites?: boolean;
}
```

**Key Features:**
- Show/hide toggle functionality for long lists
- Rating color coding (green ≥4.5, blue ≥4.0, yellow ≥3.5)
- Price range color coding with euro/dollar symbols
- Cuisine type emoji mapping
- Favorite/heart button with loading states
- Add to list functionality
- Share functionality
- Website links
- Responsive design (compact vs full mode)

**Utility Functions:**
- `getRatingColor(rating)` - Returns color classes based on rating
- `getPriceColor(priceRange)` - Returns color classes for price range
- `getCuisineEmoji(cuisine)` - Maps cuisine types to emojis

---

### 2. ItineraryResults.tsx (`src/components/results/ItineraryResults.tsx`)

**Purpose:** Displays itinerary with multiple POIs in ordered sequence with priority indicators.

**Key Imports:**
```tsx
import {
  Calendar, ChevronDown, ChevronRight, ChevronUp, 
  Clock, MapPin, Share2, Star
} from "lucide-solid";
import { For, Show, createSignal } from "solid-js";
import AddToListButton from "~/components/lists/AddToListButton";
```

**Interface Definition:**
```tsx
interface POI {
  id?: string;
  name: string;
  latitude?: number;
  longitude?: number;
  category?: string;
  description_poi?: string;
  address?: string;
  website?: string;
  opening_hours?: string;
  rating?: number;
  priority?: number;
  distance?: number;
  timeToSpend?: string;
  budget?: string;
  llm_interaction_id?: string;
}

interface ItineraryData {
  itinerary_name?: string;
  overall_description?: string;
  points_of_interest?: POI[];
}
```

**Key Features:**
- Priority-based ordering and color coding
- Numbered sequence markers (priority or index-based)
- Itinerary name parsing (handles JSON strings)
- Overall description display
- Time to spend and budget indicators
- Category emoji mapping
- Star-based favorites (yellow theme)
- Add entire itinerary to list option

**Utility Functions:**
- `getPriorityColor(priority)` - Color codes priority levels
- `getCategoryEmoji(category)` - Maps categories to emojis
- `itineraryName()` - Parses itinerary name from various formats

---

### 3. ActivityResults.tsx (`src/components/results/ActivityResults.tsx`)

**Purpose:** Displays activity listings with duration, budget, and category information.

**Key Imports:**
```tsx
import {
  Calendar, ChevronDown, ChevronUp, Clock, 
  DollarSign, MapPin, Star
} from "lucide-solid";
import { For, Show, createSignal } from "solid-js";
import AddToListButton from "~/components/lists/AddToListButton";
```

**Interface Definition:**
```tsx
interface Activity {
  id?: string;
  name: string;
  latitude?: number;
  longitude?: number;
  category?: string;
  description_poi?: string;
  address?: string;
  website?: string;
  opening_hours?: string;
  rating?: number;
  budget?: string;
  price_range?: string;
  duration?: string;
  distance?: number;
  llm_interaction_id?: string;
}
```

**Key Features:**
- Duration display with calendar icon
- Budget/price range with color coding
- Category-specific emoji mapping (includes activity-specific ones)
- No favorite functionality (simpler than restaurants)
- Distance indicators
- Opening hours display

**Utility Functions:**
- `getBudgetColor(budget)` - Color codes budget levels
- `getCategoryEmoji(category)` - Activity-focused emoji mapping

---

### 4. HotelResults.tsx (`src/components/results/HotelResults.tsx`)

**Purpose:** Displays hotel listings with amenities, ratings, and pricing.

**Key Imports:**
```tsx
import {
  Car, ChevronDown, ChevronUp, Coffee, Heart, 
  MapPin, Share2, Star, Utensils, Wifi
} from "lucide-solid";
import { For, Show, createSignal } from "solid-js";
import AddToListButton from "~/components/lists/AddToListButton";
```

**Interface Definition:**
```tsx
interface Hotel {
  id?: string;
  name: string;
  latitude?: number;
  longitude?: number;
  category?: string;
  description_poi?: string;
  address?: string;
  website?: string;
  opening_hours?: string;
  rating?: number;
  price_range?: string;
  amenities?: string[];
  distance?: number;
  llm_interaction_id?: string;
}
```

**Key Features:**
- Amenities display with icons (WiFi, parking, breakfast, restaurant)
- Price range badges with blue theme
- Heart-based favorites
- Amenity overflow handling (shows +X more)
- Hotel-specific styling

**Utility Functions:**
- `getAmenityIcon(amenity)` - Maps amenity strings to icons

## Shared UI Components

### 1. AddToListButton.tsx (`src/components/lists/AddToListButton.tsx`)

**Purpose:** Universal component for adding any content type to user lists.

**Key Imports:**
```tsx
import { FolderPlus, Plus, X } from "lucide-solid";
import { createSignal, For, Show } from "solid-js";
import {
  useAddToListMutation, useCreateListMutation, useLists
} from "~/lib/api/lists";
import { useBookmarkedItineraries } from "~/lib/api/itineraries";
```

**Interface Definition:**
```tsx
interface AddToListButtonProps {
  itemId: string;
  contentType: "poi" | "restaurant" | "hotel" | "itinerary";
  itemName: string;
  className?: string;
  size?: "sm" | "md" | "lg";
  variant?: "icon" | "button" | "minimal";
  sourceInteractionId?: string;
  aiDescription?: string;
}
```

**Key Features:**
- Three display variants (icon, button, minimal)
- Size variants (sm, md, lg)
- Modal interface for list selection
- Create new list functionality
- Integration with bookmarked itineraries
- Mutation-based API integration

---

### 2. ChatInterface.tsx (`src/components/ui/ChatInterface.tsx`)

**Purpose:** Reusable chat interface component with floating button.

**Key Features:**
- Floating chat button
- Expandable chat window
- Message history display
- Typing indicators
- Customizable theming
- SSE streaming support integration

---

### 3. Map.tsx (`src/components/features/Map/Map.tsx`)

**Purpose:** Interactive Mapbox GL JS map with POI markers and route lines.

**Key Imports:**
```tsx
import { onMount, onCleanup, createEffect } from "solid-js";
import mapboxgl from "mapbox-gl";
```

**Key Features:**
- Mapbox GL JS integration
- Custom numbered markers with priority colors
- Route optimization and line drawing
- Responsive marker sizing
- Popup information windows
- Coordinate validation and error handling
- Bounds fitting

**Configuration:**
- Requires `VITE_MAPBOX_API_KEY` environment variable
- Uses Standard map style by default
- Supports custom styling and zoom levels

---

### 4. TypingAnimation.tsx (`src/components/TypingAnimation.tsx`)

**Purpose:** Text streaming animation component.

**Dependencies:**
- Custom hook: `useStreamingText`

## Hooks and Utilities

### 1. useChatSession.ts (`src/lib/hooks/useChatSession.ts`)

**Purpose:** Comprehensive chat session management with SSE streaming.

**Key Features:**
- SSE (Server-Sent Events) streaming
- Session persistence and recovery
- Message history management
- Error handling and retry logic
- Navigation integration
- Itinerary update handling

**Interface Definition:**
```tsx
interface ChatMessage {
  type: 'user' | 'assistant' | 'error';
  content: string;
  timestamp: Date;
}

interface UseChatSessionOptions {
  sessionIdPrefix?: string;
  onStreamingComplete?: (data: any) => void;
  onError?: (error: Error) => void;
  onUpdateStart?: () => void;
  onUpdateComplete?: () => void;
  getStreamingData?: () => any;
  setStreamingData?: (data: any) => void;
  mapDisabled?: boolean;
  setMapDisabled?: (disabled: boolean) => void;
  poisUpdateTrigger?: () => void;
  setPoisUpdateTrigger?: (fn: (prev: number) => number) => void;
  enableNavigation?: boolean;
  onNavigationData?: (navigation: any) => void;
}
```

## External Dependencies

### Required npm packages:
```json
{
  "solid-js": "^1.x.x",
  "@solidjs/router": "^0.x.x",
  "lucide-solid": "^0.x.x",
  "mapbox-gl": "^2.x.x"
}
```

### Environment Variables:
```env
VITE_MAPBOX_API_KEY=your_mapbox_access_token
VITE_API_BASE_URL=your_api_base_url
```

## Styling System

### TailwindCSS Classes Used:

**Layout & Spacing:**
- `space-y-3`, `space-y-4`, `gap-2`, `gap-4`
- `p-3`, `p-4`, `px-4`, `py-2`
- `mb-2`, `mb-3`, `mb-4`, `mt-3`

**Colors & Themes:**
- **Primary:** `blue-600`, `blue-700`, `blue-50`
- **Success:** `green-600`, `green-400`
- **Warning:** `yellow-600`, `yellow-400`
- **Error:** `red-600`, `red-400`
- **Orange:** `orange-500`, `orange-600`
- **Purple:** `purple-500`, `purple-600`

**Dark Mode Support:**
- All components include `dark:` variants
- Primary dark classes: `dark:bg-gray-800`, `dark:text-white`

**Responsive Design:**
- Mobile-first approach
- `sm:`, `md:`, `lg:` breakpoints used throughout

**Component States:**
- `hover:`, `focus:`, `disabled:` pseudo-classes
- `transition-colors`, `transition-shadow`
- Loading states with `animate-spin`

## Porting Checklist

When porting to a new project:

1. **Framework Migration:**
   - [ ] Convert SolidJS signals to your framework's state management
   - [ ] Convert `For` and `Show` components to equivalent loops/conditionals
   - [ ] Convert `createEffect` to your framework's effect system

2. **Dependencies:**
   - [ ] Install required packages
   - [ ] Set up environment variables
   - [ ] Configure API endpoints

3. **Styling:**
   - [ ] Ensure TailwindCSS is configured
   - [ ] Copy custom CSS classes if any
   - [ ] Test dark mode functionality

4. **API Integration:**
   - [ ] Replace API hooks with your data fetching solution
   - [ ] Update endpoint URLs
   - [ ] Handle authentication tokens

5. **Icons:**
   - [ ] Install lucide-react (for React) or equivalent icon library
   - [ ] Update icon imports if changing libraries

6. **Testing:**
   - [ ] Test all interactive features
   - [ ] Verify responsive design
   - [ ] Test favorite/share functionality
   - [ ] Verify map integration
   - [ ] Test add-to-list functionality

This guide provides all the information needed to recreate the UI components in any modern framework while maintaining the same functionality and user experience.