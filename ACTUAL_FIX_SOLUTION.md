# Actual Fix for LLM POI Favorites Error

## Root Cause Analysis

The error occurs because:

1. When an LLM-generated POI is displayed in chat, it's saved to `llm_suggested_pois` via `SaveSinglePOI()`
2. However, `SaveSinglePOI()` has `ON CONFLICT (name, latitude, longitude) DO NOTHING`
3. When a POI with the same name and coordinates already exists, the insert is skipped
4. Because of `DO NOTHING`, no ID is returned (`RETURNING id` returns nothing)
5. This causes `pgx.ErrNoRows` error which is treated as a failure
6. The POI's ID in the response becomes `uuid.Nil`
7. When you try to favorite this POI, it uses `uuid.Nil` which doesn't exist in `llm_suggested_pois`
8. This violates the foreign key constraint

## The Fix

Modify `SaveSinglePOI` in `/Users/fernando_idwell/Projects/Loci/go-templui/internal/app/domain/chat_prompt/chat_repository.go`

### Current Code (lines 1536-1575):

```go
query := `
    INSERT INTO llm_suggested_pois (
        id, user_id, city_id, llm_interaction_id, name,
        latitude, longitude, "location",
        category, description_poi
    ) VALUES (
        $1, $2, $3, $4, $5,
        $6, $7, ST_SetSRID(ST_MakePoint($7, $6), 4326),
        $8, $9
    )
    ON CONFLICT (name, latitude, longitude) DO NOTHING
    RETURNING id
`

var returnedID uuid.UUID
err = tx.QueryRow(ctx, query,
    recordID,           // $1: id
    userID,             // $2: user_id
    cityID,             // $3: city_id
    llmInteractionID,   // $4: llm_interaction_id
    poi.Name,           // $5: name
    poi.Latitude,       // $6: latitude column value
    poi.Longitude,      // $7: longitude column value
    poi.Category,       // $8: category
    poi.DescriptionPOI, // $9: description_poi
).Scan(&returnedID)

if err != nil {
    r.logger.Error("Failed to insert llm_suggested_poi", zap.Any("error", err), zap.String("query", query), zap.String("name", poi.Name))
    span.RecordError(err)
    return uuid.Nil, fmt.Errorf("failed to save llm_suggested_poi: %w", err)
}
```

### Fixed Code:

```go
query := `
    INSERT INTO llm_suggested_pois (
        id, user_id, city_id, llm_interaction_id, name,
        latitude, longitude, "location",
        category, description_poi
    ) VALUES (
        $1, $2, $3, $4, $5,
        $6, $7, ST_SetSRID(ST_MakePoint($7, $6), 4326),
        $8, $9
    )
    ON CONFLICT (name, latitude, longitude) DO NOTHING
    RETURNING id
`

var returnedID uuid.UUID
err = tx.QueryRow(ctx, query,
    recordID,           // $1: id
    userID,             // $2: user_id
    cityID,             // $3: city_id
    llmInteractionID,   // $4: llm_interaction_id
    poi.Name,           // $5: name
    poi.Latitude,       // $6: latitude column value
    poi.Longitude,      // $7: longitude column value
    poi.Category,       // $8: category
    poi.DescriptionPOI, // $9: description_poi
).Scan(&returnedID)

if err != nil {
    // Check if it's a "no rows" error, which means the POI already exists
    if errors.Is(err, pgx.ErrNoRows) {
        r.logger.Info("POI already exists, fetching existing ID",
            zap.String("name", poi.Name),
            zap.Float64("latitude", poi.Latitude),
            zap.Float64("longitude", poi.Longitude))

        // Query for the existing POI's ID
        selectQuery := `
            SELECT id FROM llm_suggested_pois
            WHERE name = $1 AND latitude = $2 AND longitude = $3
            LIMIT 1
        `
        err = tx.QueryRow(ctx, selectQuery, poi.Name, poi.Latitude, poi.Longitude).Scan(&returnedID)
        if err != nil {
            r.logger.Error("Failed to fetch existing POI ID", zap.Any("error", err))
            span.RecordError(err)
            return uuid.Nil, fmt.Errorf("failed to fetch existing llm_suggested_poi: %w", err)
        }

        r.logger.Info("Found existing POI", zap.String("id", returnedID.String()))
    } else {
        r.logger.Error("Failed to insert llm_suggested_poi", zap.Any("error", err), zap.String("query", query), zap.String("name", poi.Name))
        span.RecordError(err)
        return uuid.Nil, fmt.Errorf("failed to save llm_suggested_poi: %w", err)
    }
}
```

## Implementation Steps

1. Import the `errors` package at the top of the file if not already imported:
   ```go
   import "errors"
   ```

2. Replace the error handling section (lines 1563-1566) with the fixed code above

3. Test by:
   - Generating a chat response with POIs
   - Adding one of the POIs to favorites
   - Generating another chat response (same city) which might include the same POI
   - Try to favorite it again - should work now

## Why This Works

Instead of failing when a POI already exists:
1. We detect the `pgx.ErrNoRows` error (which happens when `ON CONFLICT DO NOTHING` skips the insert)
2. We query for the existing POI's ID using the same conflict keys (name, latitude, longitude)
3. We return that existing ID instead of `uuid.Nil`
4. Now when the favorite button is clicked, it has a valid POI ID that exists in `llm_suggested_pois`
5. The foreign key constraint is satisfied

## Alternative Solution (if you want to avoid the second query)

Change the `ON CONFLICT` clause to use `DO UPDATE` instead:

```go
query := `
    INSERT INTO llm_suggested_pois (
        id, user_id, city_id, llm_interaction_id, name,
        latitude, longitude, "location",
        category, description_poi
    ) VALUES (
        $1, $2, $3, $4, $5,
        $6, $7, ST_SetSRID(ST_MakePoint($7, $6), 4326),
        $8, $9
    )
    ON CONFLICT (name, latitude, longitude) DO UPDATE SET
        llm_interaction_id = EXCLUDED.llm_interaction_id,
        category = EXCLUDED.category,
        description_poi = EXCLUDED.description_poi
    RETURNING id
`
```

This will:
- Insert if the POI doesn't exist
- Update the existing POI if it does exist
- **Always return an ID** (either the new one or the existing one)

However, this approach updates the existing POI, which might not be desired if you want to preserve the original version.

## Recommended Approach

I recommend the first solution (fetching the existing ID) because:
- It preserves the original POI data
- It's clearer about what's happening
- It logs when a duplicate is found
- It's safer and more predictable
