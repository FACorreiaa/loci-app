# **Loci** – Personalized City Discovery 🗺️✨

Loci is a smart, mobile-first web application delivering hyper-personalized city exploration recommendations based on user interests, time, location, and an evolving AI engine. It starts with an HTTP/REST API, utilizing WebSockets/SSE for real-time features.

## 🚀 Elevator Pitch & Core Features

Tired of generic city guides? loci learns your preferences (history, food, art, etc.) and combines them with your available time and location to suggest the perfect spots.

- **🧠 AI-Powered Personalization:** Recommendations adapt to explicit preferences and learned behavior.
- **🔍 Contextual Filtering:** Filter by distance, time, opening hours, interests, and soon, budget.
- **🗺 Interactive Map Integration:** Visualize recommendations and routes.
- **📌 Save & Organize:** Bookmark favorites and create lists/itineraries (enhanced in Premium).
- **📱 Mobile-First Design:** Optimized for on-the-go web browsing.

## 💰 Business Model & Monetization

Loci uses a **Freemium Model**:

- **Free Tier:** Core recommendations, basic filters, limited saves, non-intrusive ads.
- **Premium Tier (Subscription):** Enhanced/Advanced AI recommendations & filters (niche tags, cuisine, accessibility), unlimited saves, offline access, exclusive content, ad-free.

**Monetization Avenues:**

- Premium Subscriptions
- **Partnerships & Commissions:** Booking referrals (GetYourGuide, Booking.com, OpenTable), transparent featured listings, exclusive deals.
- **Future:** One-time purchases (guides), aggregated anonymized trend data.

## 🛠 Technology Stack & Design Choices

The stack prioritizes performance, personalization, and developer experience.

- **Backend:** **Go (Golang)** with **Gin**, **PostgreSQL + PostGIS** (for geospatial queries), `pgx` or `sqlc`.
  - _Rationale:_ Go for performance and concurrency; PostGIS for essential location features.
- **Frontend:** **Templ** (type-safe Go templates), **HTMX** (dynamic interactions), **Alpine.js** (lightweight reactivity), **Tailwind CSS**.
  - _Rationale:_ Server-side rendering with minimal JavaScript for performance and simplicity.
- **AI / Recommendation Engine:**

Direct Google Gemini API integration via `google/generative-ai-go` SDK.** \* _Rationale:_ Leverage latest models (e.g., Gemini 1.5 Pro) for deep personalization via rich prompts and function calling to access PostgreSQL data (e.g., nearby POIs from PostGIS). \* **Vector Embeddings:\*\* PostgreSQL with `pgvector` extension for semantic search and advanced recommendations.

- **API Layer:** Primary **HTTP/REST API**.
  - _Rationale:_ Simplicity for frontend integration and broad compatibility. gRPC considered for future backend-to-backend needs.
- **Authentication:** Standard JWT + `Goth` package for social logins.
- **Infrastructure:** Docker, Docker Compose; Cloud (AWS/GCP/Azure for managed services like Postgres, Kubernetes/Fargate/Cloud Run); CI/CD (GitHub Actions/GitLab CI).

## 🗺️ Roadmap Highlights

- **Phase 1 (MVP):** Core recommendation engine (Gemini-powered), user accounts, map view, itinerary personalisation.
- **Phase 2:** Premium tier, enhanced AI (embeddings, `pgvector`), add more gemini features like

* speech to text
* itinerary download to different formats (pdf/markdown)
* itinerary uploads
* 24/7 agent more personalised agent

reviews/ratings, booking partnerships.

- **Phase 3:** Multi-city expansion, curated content, native app exploration.

## 🚀 Elevator Pitch

Tired of generic city guides? **WanderWise** learns what you love—be it history, food, art, nightlife, or hidden gems—and combines it with your available time and location to suggest the perfect spots, activities, and restaurants.

Whether you're a tourist on a tight schedule or a local looking for something new, discover your city like never before with hyper-personalized, intelligent recommendations.

---

## 🌟 Core Features

- **🧠 AI-Powered Personalization**
  Recommendations adapt based on explicit user preferences and learned behavior over time.

- **🔍 Contextual Filtering**
  Filters results by:
  - Distance / Location
  - Available Time (e.g., “things to do in the next 2 hours”)
  - Opening Hours
  - User Interests (e.g., "art", "foodie", "outdoors", "history")
  - Budget (coming soon)

- **🗺 Interactive Map Integration**
  Visualize recommendations, your location, and potential routes.

- **📌 Save & Organize**
  Bookmark favorites, create custom lists or simple itineraries (enhanced in Premium).

- **📱 Mobile-First Design**
  Optimized for on-the-go browsing via web browser.

---

## 💰 Business Model & Monetization

### Freemium Model

- **Free Tier**:
  - Access to core recommendation engine
  - Basic preference filters
  - Limited saves/lists
  - Non-intrusive contextual ads

- **Premium Tier (Monthly/Annual Subscription)**:
  - Enhanced AI recommendations
  - Advanced filters (cuisine, accessibility, niche tags, specific hours)
  - Unlimited saves & lists
  - Offline access
  - Exclusive curated content & themed tours
  - Ad-free experience

### Partnerships & Commissions

- **Booking Referrals**
  Earn commission via integrations with platforms like GetYourGuide, Booking.com, OpenTable, etc.

- **Featured Listings (Transparent)**
  Local businesses can pay for premium visibility in relevant results.

- **Exclusive Deals**
  Offer users special discounts via business partnerships (potentially Premium-only).

### Future Monetization Options

- One-time in-app purchases (premium guides, city packs)
- Aggregated anonymized trend data (for tourism boards, researchers)

## 🧪 Getting Started

> 🔧 _Instructions for local setup coming soon._

## 🤝 Contributing

> 🛠 _Contribution guidelines and code of conduct coming soon._

## 📄 License

> 📃 _License type to be defined (MIT, Apache 2.0, or Proprietary)._

## 🔄 Real-time Streaming with Gin + Templ + HTMX

For interactive streaming results on `/discover`, `/itinerary`, and `/restaurants` pages:

### Backend (SSE with Gin)
Building on your existing SSE/gRPC streaming experience:

```go
// SSE Headers middleware
func SSEHeaders() gin.HandlerFunc {
    return func(c *gin.Context) {
        c.Header("Content-Type", "text/event-stream")
        c.Header("Cache-Control", "no-cache")
        c.Header("Connection", "keep-alive")
        c.Header("Access-Control-Allow-Origin", "*")
        c.Header("Access-Control-Allow-Headers", "Cache-Control")
        c.Next()
    }
}

// Stream recommendations using Gin's c.Stream()
func (h *Handler) StreamRecommendations(c *gin.Context) {
    clientChan := make(chan string)
    defer close(clientChan)

    // Start Gemini streaming
    stream, err := h.geminiClient.GenerateContentStream(c.Request.Context(), prompt)
    if err != nil {
        c.SSEvent("error", "Failed to start stream")
        return
    }

    c.Stream(func(w io.Writer) bool {
        select {
        case response := <-geminiResponseChan:
            if response.Error != nil {
                c.SSEvent("end", "Stream complete")
                return false
            }
            c.SSEvent("data", response.Text)
            return true
        case <-c.Request.Context().Done():
            return false
        }
    })
}
```

### Frontend (Templ + HTMX + Alpine)
```go
// recommendations.templ
templ RecommendationsPage() {
    <div x-data="{ content: '', loading: true }" 
         hx-ext="sse" 
         sse-connect="/api/recommendations/stream">
        
        <div x-show="loading" class="loading-spinner">
            Generating recommendations...
        </div>
        
        <div sse-swap="data" 
             x-on:sse-data="content += $event.detail.data; loading = false"
             x-text="content"
             class="recommendations-stream">
        </div>
        
        <div sse-swap="end" x-on:sse-end="loading = false"></div>
    </div>
}
```

This leverages Gin's native `c.Stream()` and `c.SSEvent()` methods with Templ templates and HTMX's SSE extension for seamless real-time updates.

## 💡 Monetization Strategy

### Revenue Optimization
- **Affiliate Commissions**: Partner with GetYourGuide, Booking.com, OpenTable for booking referrals
- **Featured Listings**: Transparent premium visibility for local businesses in recommendations
- **Premium Subscriptions**: Enhanced AI features, offline access, unlimited saves
- **Cost Management**: Use Gemini 1.5 Flash for basic queries, cache results with pgvector, limit free tier usage

### Conversion Tactics
- **Free Trial**: 7-day Premium trial to reduce purchase friction
- **Behavioral Targeting**: Personalized upgrade prompts based on usage patterns
- **Tiered Pricing**: Basic Premium ($3.99/month) and Advanced Premium ($9.99/month)
- **Target Markets**: Focus on foodies, solo travelers, and business travelers for higher conversion rates