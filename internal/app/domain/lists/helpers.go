package lists

import (
	"fmt"
	"time"

	"github.com/FACorreiaa/go-templui/internal/app/models"
)

// calculateTotalItems sums up all item counts from the user's lists
func calculateTotalItems(lists []*models.List) int {
	total := 0
	for _, list := range lists {
		if list != nil {
			total += list.ItemCount
		}
	}
	return total
}

// countPublicLists counts how many lists are public
func countPublicLists(lists []*models.List) int {
	count := 0
	for _, list := range lists {
		if list != nil && list.IsPublic {
			count++
		}
	}
	return count
}

// formatRelativeTime formats a time.Time as a relative time string (e.g., "2 days ago", "just now")
func formatRelativeTime(t time.Time) string {
	now := time.Now()
	diff := now.Sub(t)

	seconds := int(diff.Seconds())
	minutes := int(diff.Minutes())
	hours := int(diff.Hours())
	days := int(hours / 24)
	weeks := int(days / 7)
	months := int(days / 30)
	years := int(days / 365)

	switch {
	case seconds < 60:
		return "just now"
	case minutes == 1:
		return "1 minute ago"
	case minutes < 60:
		return fmt.Sprintf("%d minutes ago", minutes)
	case hours == 1:
		return "1 hour ago"
	case hours < 24:
		return fmt.Sprintf("%d hours ago", hours)
	case days == 1:
		return "1 day ago"
	case days < 7:
		return fmt.Sprintf("%d days ago", days)
	case weeks == 1:
		return "1 week ago"
	case weeks < 4:
		return fmt.Sprintf("%d weeks ago", weeks)
	case months == 1:
		return "1 month ago"
	case months < 12:
		return fmt.Sprintf("%d months ago", months)
	case years == 1:
		return "1 year ago"
	default:
		return fmt.Sprintf("%d years ago", years)
	}
}
