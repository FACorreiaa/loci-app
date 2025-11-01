package metrics

import (
	"log"
	"sync"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
)

// AppMetrics holds the application's metric instruments.
// Make fields public so they can be accessed from other packages.
type AppMetrics struct {
	HTTPRequestsTotal      metric.Int64Counter
	HTTPRequestDuration    metric.Float64Histogram
	AuthRequestsTotal      metric.Int64Counter
	SearchRequestsTotal    metric.Int64Counter
	DBQueryDurationSeconds metric.Float64Histogram
	DBQueryErrorsTotal     metric.Int64Counter
	ActiveUsersGauge       metric.Int64Gauge
	TemplateRenderDuration metric.Float64Histogram
}

var (
	// Global instance of AppMetrics (initialized once)
	appMetrics *AppMetrics
	once       sync.Once
)

// InitAppMetrics initializes the global metrics instruments ONLY ONCE.
// It gets the Meter from the globally configured MeterProvider.
func InitAppMetrics() {
	once.Do(func() { // Ensure this only runs once
		meter := otel.GetMeterProvider().Meter("loci-templui") // Get meter from global provider
		var err error
		m := &AppMetrics{}

		m.HTTPRequestsTotal, err = meter.Int64Counter(
			"http_requests_total",
			metric.WithDescription("Total number of HTTP requests completed"),
			metric.WithUnit("{request}"),
		)
		if err != nil {
			log.Fatalf("Metrics: Failed to create http_requests_total: %v", err)
		}

		m.HTTPRequestDuration, err = meter.Float64Histogram(
			"http_request_duration_seconds",
			metric.WithDescription("Duration of HTTP requests in seconds"),
			metric.WithUnit("s"),
		)
		if err != nil {
			log.Fatalf("Metrics: Failed to create http_request_duration_seconds: %v", err)
		}

		m.AuthRequestsTotal, err = meter.Int64Counter(
			"auth_requests_total",
			metric.WithDescription("Total number of authentication requests"),
			metric.WithUnit("{request}"),
		)
		if err != nil {
			log.Fatalf("Metrics: Failed to create auth_requests_total: %v", err)
		}

		m.SearchRequestsTotal, err = meter.Int64Counter(
			"search_requests_total",
			metric.WithDescription("Total number of search/discover requests"),
			metric.WithUnit("{request}"),
		)
		if err != nil {
			log.Fatalf("Metrics: Failed to create search_requests_total: %v", err)
		}

		m.ActiveUsersGauge, err = meter.Int64Gauge(
			"active_users_current",
			metric.WithDescription("Current number of active users"),
			metric.WithUnit("{user}"),
		)
		if err != nil {
			log.Fatalf("Metrics: Failed to create active_users_current: %v", err)
		}

		m.TemplateRenderDuration, err = meter.Float64Histogram(
			"template_render_duration_seconds",
			metric.WithDescription("Duration of template rendering in seconds"),
			metric.WithUnit("s"),
		)
		if err != nil {
			log.Fatalf("Metrics: Failed to create template_render_duration_seconds: %v", err)
		}

		m.DBQueryDurationSeconds, err = meter.Float64Histogram(
			"db_query_duration_seconds",
			metric.WithDescription("Duration of database queries in seconds"),
			metric.WithUnit("s"),
		)
		if err != nil {
			log.Fatalf("Metrics: Failed to create db_query_duration_seconds: %v", err)
		}

		m.DBQueryErrorsTotal, err = meter.Int64Counter(
			"db_query_errors_total",
			metric.WithDescription("Total number of database query errors"),
			metric.WithUnit("{error}"),
		)
		if err != nil {
			log.Fatalf("Metrics: Failed to create db_query_errors_total: %v", err)
		}

		log.Println("Application metrics instruments initialized.")
		appMetrics = m // Assign to global variable
	})
}

// Get returns the globally initialized AppMetrics instance.
// Panics if InitAppMetrics was not called first.
func Get() *AppMetrics {
	if appMetrics == nil {
		// This indicates a programming error - InitAppMetrics must be called at startup.
		panic("metrics instruments not initialized. Call metrics.InitAppMetrics() first.")
	}
	return appMetrics
}
