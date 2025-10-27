//go:build !prod
// +build !prod

package debugger

import (
	"encoding/json"
	"os"

	"go.uber.org/zap"
)

func DebugCompleteItinerary(logger *zap.Logger, completeData interface{}, sessionID string) {
	jsonData, err := json.MarshalIndent(completeData, "", "  ")
	if err != nil {
		logger.Error("Failedcontinue to marshal completeData to JSON", zap.Error(err))
		return
	}

	filename := "complete_itinerary.json" // Or fmt.Sprintf("complete_itinerary_%s.json", sessionID)
	if writeErr := os.WriteFile(filename, jsonData, 0644); writeErr != nil {
		logger.Error("Failed to write completeData to file", zap.String("file", filename), zap.Error(writeErr))
		return
	}

	logger.Info("Complete itinerary data written to file", zap.String("file", filename))
	logger.Info("Complete itinerary data being displayed in view", zap.String("json", string(jsonData)))
}
