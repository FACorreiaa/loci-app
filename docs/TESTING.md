# Test Suite Documentation

This document describes the comprehensive test suite created for the SSE data and handlers refactoring.

## Overview

The test suite validates the unified data source architecture and domain-specific filtering functionality implemented for Activities, Hotels, and Restaurants handlers.

## Test Structure

### 1. Core Functionality Tests (`core_functions_test.go`)

**Purpose**: Tests the fundamental filtering and conversion logic that enables the unified data source architecture.

#### Filtering Function Tests (`TestFilteringFunctions`)
- **Domain-Specific Filtering**: Validates that each domain handler correctly filters POIs from a unified dataset
  - `filterPOIsForActivities`: Tests filtering for museums, parks, theaters, galleries, sports, adventure, cultural, entertainment, outdoor, recreation
  - `filterPOIsForHotels`: Tests filtering for hotels, hostels, resorts, guesthouses, apartments, villas, motels, inns, B&Bs, accommodation, lodging
  - `filterPOIsForRestaurants`: Tests filtering for restaurants, cafes, coffee shops, bars, pubs, bistros, brasseries, pizzerias, bakeries, markets, food courts, fast food, takeaways, dining, food

- **Case Insensitivity**: Ensures filters work with mixed-case category names
- **Empty Input Handling**: Verifies graceful handling of empty POI arrays
- **No Overlap Verification**: Confirms that POIs are categorized into exactly one domain (critical for unified data source integrity)

#### Conversion Function Tests (`TestConversionFunctions`)
- **Type Conversion Accuracy**: Tests conversion from generic `POIDetailedInfo` to domain-specific types:
  - `convertPOIToHotel`: Converts to `HotelDetailedInfo` with proper pointer field handling
  - `convertPOIToRestaurant`: Converts to `RestaurantDetailedInfo` with proper pointer field handling

- **Field Mapping Verification**: Ensures all fields are correctly mapped between types
- **Opening Hours Conversion**: Tests conversion of map data to string format for domain-specific types
- **Null/Empty Field Handling**: Verifies correct handling of optional fields

### 2. Basic Integration Tests (`handlers_basic_test.go`)

**Purpose**: Tests handler instantiation and basic functionality without requiring database connections.

#### Handler Instantiation Tests (`TestHandlers_BasicInstantiation`)
- **Graceful Degradation**: Ensures handlers work even with minimal dependencies
- **Query Parameter Handling**: Verifies handlers accept and process query parameters
- **Component Generation**: Confirms all handlers return valid template components

#### Unified Data Source Concept Tests (`TestUnifiedDataSourceConcept`)
- **Data Source Unity**: Demonstrates that all domain handlers work from the same data source
- **Cache Integration**: Tests integration with the actual cache data structures (`AiCityResponse`)
- **Complete Data Flow**: Validates the entire flow from unified data through domain filtering

### 3. End-to-End Integration Tests (`sse_integration_test.go`)

**Purpose**: Comprehensive integration tests for SSE data flow (requires integration build tag).

#### SSE Data Flow Tests (`TestSSE_EndToEndDataFlow`)
- **Complete SSE Pipeline**: Tests SSE stream initiation through domain handler consumption
- **Session Management**: Validates session ID extraction and tracking
- **Cache Population**: Ensures SSE streams populate cache correctly
- **Domain Handler Integration**: Verifies domain handlers can consume SSE-generated data

#### SSE Streaming Tests (`TestSSE_StreamingEvents`)
- **Stream Event Validation**: Tests proper SSE event formatting and content
- **Header Verification**: Ensures correct SSE headers are set
- **Event Type Coverage**: Validates different SSE event types (progress, data, complete, error)

#### Cache Integration Tests (`TestSSE_CacheIntegration`)
- **Cache Population Timing**: Tests cache is populated during SSE streaming
- **Data Accessibility**: Ensures cached data is accessible to domain handlers
- **Data Consistency**: Verifies data integrity throughout the pipeline

### 4. Unified Data Source Architecture Tests (`unified_data_integration_test.go`)

**Purpose**: Comprehensive testing of the unified data source pattern (requires integration build tag).

#### Unified Data Source Tests (`TestUnifiedDataSource_Integration`)
- **Single Source, Multiple Domains**: Validates all handlers use the same data source
- **Domain-Specific Filtering**: Tests filtering works correctly across all domains
- **Data Consistency**: Ensures consistent city and metadata across domains
- **Type Conversion Integration**: Validates conversion functions work in real scenarios

#### Database Fallback Tests (`TestUnifiedDataSource_DatabaseFallback`)
- **Cache Miss Handling**: Tests database fallback when cache is empty
- **Data Parsing**: Validates parsing of stored database responses
- **Error Handling**: Ensures graceful handling of database errors

#### Legacy Cache Support (`TestUnifiedDataSource_LegacyCacheFallback`)
- **Backward Compatibility**: Tests fallback to legacy cache structures
- **Migration Support**: Ensures old data can still be accessed

## Test Data Strategy

### Comprehensive Test Data Sets
- **40+ POIs**: Tests use comprehensive datasets covering all category types
- **Realistic Data**: Test POIs include real-world examples (Louvre Museum, Le Bernardin, etc.)
- **Edge Cases**: Include mixed-case categories, empty fields, and boundary conditions

### Category Coverage
- **Activities**: 10 categories (museum, park, theater, gallery, sports, adventure, cultural, entertainment, outdoor, recreation)
- **Hotels**: 12 categories (hotel, hostel, resort, guesthouse, apartment, villa, motel, inn, b&b, accommodation, lodging, bnb)  
- **Restaurants**: 15 categories (restaurant, cafe, coffee, bar, pub, bistro, brasserie, pizzeria, bakery, market, foodcourt, fastfood, takeaway, dining, food)

## Key Testing Principles

### 1. Unified Data Source Validation
Every test reinforces that Activities, Hotels, and Restaurants handlers:
- Use the same data source (cache/database)
- Apply domain-specific filtering
- Never return PageNotFound for valid sessions
- Maintain data consistency across domains

### 2. Type Safety
All conversion functions are thoroughly tested to ensure:
- Correct field mapping
- Proper pointer handling
- Safe null/empty value processing
- No data loss during conversion

### 3. Real-World Scenarios
Tests simulate actual usage patterns:
- SSE stream → cache population → domain filtering
- Cache hits and misses
- Database fallback scenarios
- Session ID validation

## Running the Tests

### Unit Tests (No External Dependencies)
```bash
go test ./app/pkg/handlers -v
```

### Integration Tests (Requires Test Database)
```bash
go test ./app/pkg/handlers -v -tags=integration
```

### Specific Test Suites
```bash
# Core functionality only
go test ./app/pkg/handlers -v -run="TestFilteringFunctions|TestConversionFunctions"

# Basic integration only  
go test ./app/pkg/handlers -v -run="TestHandlers_BasicInstantiation|TestUnifiedDataSourceConcept"
```

## Test Coverage

The test suite covers:
- ✅ Domain-specific filtering logic (100% of filtering functions)
- ✅ Type conversion logic (100% of conversion functions)  
- ✅ Handler instantiation and basic routing
- ✅ Cache integration and data flow
- ✅ SSE stream processing (integration tests)
- ✅ Database fallback mechanisms (integration tests)
- ✅ Error handling and edge cases
- ✅ Unified data source architecture validation

## Architecture Validation

The tests specifically validate the key architectural decision:
> **"All data comes from the same place as the Itinerary one, so it should never return a PageNotFound. The service splits the structure of the response based on the intent."**

This is tested through:
1. Unified data source tests demonstrating single data origin
2. Domain filtering tests showing proper categorization
3. Integration tests validating no PageNotFound for valid sessions
4. End-to-end tests confirming complete data flow

## Future Test Enhancements

Potential areas for additional testing:
- Performance tests for large datasets
- Concurrent access testing
- Cache expiration and cleanup testing  
- Network failure simulation
- Template rendering validation