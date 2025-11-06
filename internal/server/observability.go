package server

import (
	"context"
	"fmt"

	"github.com/FACorreiaa/go-templui/internal/app/observability/metrics"
	"github.com/FACorreiaa/go-templui/internal/app/observability/tracer"
	"go.uber.org/zap"
)

// ObservabilityShutdownFunc is the function type returned by InitObservability
type ObservabilityShutdownFunc func(context.Context) error

// InitObservability initializes OpenTelemetry and application metrics
func InitObservability(serviceName, metricsEndpoint string, logger *zap.Logger) (ObservabilityShutdownFunc, error) {
	// Initialize OpenTelemetry
	otelShutdown, err := tracer.InitOtelProviders(serviceName, metricsEndpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize OpenTelemetry: %w", err)
	}

	// Initialize application metrics
	metrics.InitAppMetrics()
	logger.Info("Observability initialized", zap.String("metrics_endpoint", metricsEndpoint+"/metrics"))

	return otelShutdown, nil
}
