# Fix for LLM POI Favorites Error

## Problem
When trying to add an LLM-generated POI to favorites, you're getting a foreign key constraint violation:
```
ERROR: insert or update on table "user_favorite_llm_pois" violates foreign key constraint "user_favorite_llm_pois_llm_poi_id_fkey"
```

## Root Cause
The LLM POIs displayed in the chat are not being saved to the `llm_suggested_pois` table before attempting to add them to favorites. The `user_favorite_llm_pois` table has a foreign key constraint that requires the POI to exist in `llm_suggested_pois` first.

## Solution Options

### Option 1: Save LLM POIs when chat response is generated (Recommended)
When the chat service generates POIs, immediately save them to `llm_suggested_pois` table.

**Pros:**
- POIs are persisted and can be tracked
- Maintains referential integrity
- Can track which POIs were actually shown to users

**Cons:**
- More database writes
- Need to handle duplicate POIs

### Option 2: Save LLM POI on favorite action
Only save the POI to `llm_suggested_pois` when the user clicks the favorite button.

**Pros:**
- Fewer database writes
- Only saves POIs users care about

**Cons:**
- POI might not have all required data at favorite time
- Need to reconstruct POI data from the UI

### Option 3: Use a different approach for LLM favorites
Instead of using a separate `llm_suggested_pois` table, save LLM POIs directly to the main `points_of_interest` table with a flag indicating they're LLM-generated.

**Pros:**
- Simpler data model
- Can use same favorite mechanism for all POIs
- No foreign key issues

**Cons:**
- Might clutter main POI table
- Harder to distinguish LLM vs real POIs

## Recommended Implementation (Option 1)

### Step 1: Add method to save LLM POI to repository

Add to `poi_repository.go`:

```go
func (r *RepositoryImpl) SaveLLMPOI(ctx context.Context, poi models.LLMSuggestedPOI) (uuid.UUID, error) {
    query := `
        INSERT INTO llm_suggested_pois (
            id,
            user_id,
            llm_interaction_id,
            city_id,
            city_name,
            latitude,
            longitude,
            name,
            description,
            category,
            phone_number,
            website,
            opening_hours,
            created_at
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, NOW())
        ON CONFLICT (id) DO NOTHING
        RETURNING id
    `

    var id uuid.UUID
    err := r.pgpool.QueryRow(ctx, query,
        poi.ID,
        poi.UserID,
        poi.LLMInteractionID,
        poi.CityID,
        poi.CityName,
        poi.Latitude,
        poi.Longitude,
        poi.Name,
        poi.Description,
        poi.Category,
        poi.PhoneNumber,
        poi.Website,
        poi.OpeningHours,
    ).Scan(&id)

    if err != nil {
        if errors.Is(err, pgx.ErrNoRows) {
            // Already exists, return the provided ID
            return poi.ID, nil
        }
        return uuid.Nil, fmt.Errorf("failed to save LLM POI: %w", err)
    }

    return id, nil
}
```

### Step 2: Add model for LLM Suggested POI

Add to `models/poi.go`:

```go
type LLMSuggestedPOI struct {
    ID                uuid.UUID  `json:"id"`
    UserID            uuid.UUID  `json:"user_id"`
    LLMInteractionID  uuid.UUID  `json:"llm_interaction_id"`
    CityID            *uuid.UUID `json:"city_id,omitempty"`
    CityName          string     `json:"city_name"`
    Latitude          float64    `json:"latitude"`
    Longitude         float64    `json:"longitude"`
    Name              string     `json:"name"`
    Description       string     `json:"description"`
    Category          string     `json:"category"`
    PhoneNumber       *string    `json:"phone_number,omitempty"`
    Website           *string    `json:"website,omitempty"`
    OpeningHours      *string    `json:"opening_hours,omitempty"`
    CreatedAt         time.Time  `json:"created_at"`
}
```

### Step 3: Modify chat service to save POIs

In your chat service, after generating the response with POIs, save each POI:

```go
// After getting the response from LLM
for _, poi := range response.POIs {
    llmPOI := models.LLMSuggestedPOI{
        ID:               uuid.New(),
        UserID:           userID,
        LLMInteractionID: interactionID,
        CityID:           cityID,
        CityName:         cityName,
        Latitude:         poi.Latitude,
        Longitude:        poi.Longitude,
        Name:             poi.Name,
        Description:      poi.Description,
        Category:         poi.Category,
        PhoneNumber:      poi.PhoneNumber,
        Website:          poi.Website,
        OpeningHours:     poi.OpeningHours,
    }

    savedID, err := h.poiRepo.SaveLLMPOI(ctx, llmPOI)
    if err != nil {
        h.logger.Warn("Failed to save LLM POI", zap.Error(err))
        // Continue anyway, just log the error
    } else {
        // Update the POI ID in the response to use the saved ID
        poi.ID = savedID.String()
    }
}
```

### Step 4: Update AddPoiToFavourites to ensure POI exists

Modify `poi_service.go`:

```go
func (s *ServiceImpl) AddPoiToFavourites(ctx context.Context, userID, poiID uuid.UUID, isLLMGenerated bool) (uuid.UUID, error) {
    if isLLMGenerated {
        // For LLM POIs, check if it exists in llm_suggested_pois
        // If not, you might want to return an error or handle it differently
        return s.repo.AddLLMPoiToFavourite(ctx, userID, poiID)
    }
    return s.repo.AddPoiToFavourites(ctx, userID, poiID)
}
```

## Quick Fix (If you just need it working now)

If you need a quick fix without restructuring, you could:

1. Remove the foreign key constraint temporarily
2. Add ON CONFLICT DO NOTHING to the favorite insert

**Migration to remove constraint:**

```sql
-- +goose Up
ALTER TABLE user_favorite_llm_pois
DROP CONSTRAINT IF EXISTS user_favorite_llm_pois_llm_poi_id_fkey;

-- +goose Down
ALTER TABLE user_favorite_llm_pois
ADD CONSTRAINT user_favorite_llm_pois_llm_poi_id_fkey
FOREIGN KEY (llm_poi_id) REFERENCES llm_suggested_pois (id) ON DELETE CASCADE;
```

**Warning:** This is not recommended for production as it breaks referential integrity.

## Testing

After implementing the fix:

1. Generate a chat response with POIs
2. Verify POIs are saved to `llm_suggested_pois` table:
   ```sql
   SELECT * FROM llm_suggested_pois WHERE user_id = 'your-user-id' ORDER BY created_at DESC LIMIT 10;
   ```
3. Try adding a POI to favorites
4. Verify it's in the favorites table:
   ```sql
   SELECT * FROM user_favorite_llm_pois WHERE user_id = 'your-user-id';
   ```
