package debugger

import (
	"bytes"
	"encoding/json"

	"go.uber.org/zap"
)

// DebugPrintEvents logs the event data for debugging purposes. It can be called
// in non-production environments to print events to the console or logs.
func DebugPrintEvents(logger *zap.SugaredLogger, eventData []byte) {
	// Log the raw event data
	logger.Info("Debug event data", "event", string(eventData))

	// Optionally, attempt to pretty-print if it's JSON
	var prettyJSON bytes.Buffer
	if err := json.Indent(&prettyJSON, eventData, "", "  "); err == nil {
		logger.Info("Debug pretty-printed event JSON", "event", prettyJSON.String())
	} else {
		logger.Warn("Failed to pretty-print event as JSON", zap.Error(err))
	}
}
