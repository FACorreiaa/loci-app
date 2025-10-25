package llmchat

import (
	"bytes"
	"context"
	"fmt"

	"github.com/FACorreiaa/go-templui/app/internal/features/results"
	"github.com/FACorreiaa/go-templui/app/internal/models"
)

// RenderItemHTML renders an HTML fragment for a given item based on domain
func (l *ServiceImpl) RenderItemHTML(ctx context.Context, domain string, item interface{}, index int) (string, error) {
	var buf bytes.Buffer

	switch domain {
	case "restaurants":
		restaurant, ok := item.(models.RestaurantDetailedInfo)
		if !ok {
			return "", fmt.Errorf("invalid restaurant data type")
		}
		component := results.SingleRestaurantCard(restaurant, index)
		if err := component.Render(ctx, &buf); err != nil {
			return "", fmt.Errorf("failed to render restaurant card: %w", err)
		}

	case "hotels":
		hotel, ok := item.(models.HotelDetailedInfo)
		if !ok {
			return "", fmt.Errorf("invalid hotel data type")
		}
		component := results.SingleHotelCard(hotel, index)
		if err := component.Render(ctx, &buf); err != nil {
			return "", fmt.Errorf("failed to render hotel card: %w", err)
		}

	case "activities":
		activity, ok := item.(models.POIDetailedInfo)
		if !ok {
			return "", fmt.Errorf("invalid activity data type")
		}
		component := results.SingleActivityCard(activity, index)
		if err := component.Render(ctx, &buf); err != nil {
			return "", fmt.Errorf("failed to render activity card: %w", err)
		}

	case "itinerary":
		poi, ok := item.(models.POIDetailedInfo)
		if !ok {
			return "", fmt.Errorf("invalid POI data type")
		}
		component := results.SinglePOICard(poi, index)
		if err := component.Render(ctx, &buf); err != nil {
			return "", fmt.Errorf("failed to render POI card: %w", err)
		}

	default:
		return "", fmt.Errorf("unsupported domain: %s", domain)
	}

	return buf.String(), nil
}

// RenderSuccessMessage renders a success message for chat operations
func (l *ServiceImpl) RenderSuccessMessage(ctx context.Context, action string, itemName string) (string, error) {
	var buf bytes.Buffer
	component := results.ChatSuccessMessage(action, itemName)
	if err := component.Render(ctx, &buf); err != nil {
		return "", fmt.Errorf("failed to render success message: %w", err)
	}
	return buf.String(), nil
}

// RenderErrorMessage renders an error message for chat operations
func (l *ServiceImpl) RenderErrorMessage(ctx context.Context, message string) (string, error) {
	var buf bytes.Buffer
	component := results.ChatErrorMessage(message)
	if err := component.Render(ctx, &buf); err != nil {
		return "", fmt.Errorf("failed to render error message: %w", err)
	}
	return buf.String(), nil
}

// RenderProcessingIndicator renders a processing indicator for chat operations
func (l *ServiceImpl) RenderProcessingIndicator(ctx context.Context, domain string) (string, error) {
	var buf bytes.Buffer
	component := results.ChatProcessingIndicator(domain)
	if err := component.Render(ctx, &buf); err != nil {
		return "", fmt.Errorf("failed to render processing indicator: %w", err)
	}
	return buf.String(), nil
}
