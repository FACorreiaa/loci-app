package tracer

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/prometheus"

	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"
)

// InitOtelProviders initializes OpenTelemetry tracing and metrics providers.
// Returns a shutdown function.
func InitOtelProviders(serviceName string, metricsAddr string) (func(context.Context) error, error) {
	// --- Common Resource ---
	res := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceName(serviceName),
		semconv.ServiceVersion("1.0.0"),
	)

	// --- Tracer Provider Setup ---
	var tp *sdktrace.TracerProvider

	// Configure OTLP exporter for traces to send to collector via HTTP
	traceExporter, err := otlptracehttp.New(context.Background(),
		otlptracehttp.WithEndpoint("http://otel-collector:4318"), // Send to OTel Collector via HTTP
		otlptracehttp.WithInsecure(),                             // OK for local docker network
	)
	if err != nil {
		// Fallback to NoOp if OTLP export fails (for local development)
		log.Printf("Warning: Failed to create OTLP trace exporter, using NoOp: %v", err)
		tp = sdktrace.NewTracerProvider(sdktrace.WithResource(res))
		log.Println("Set up OpenTelemetry Tracer Provider (NoOp Exporter)")
	} else {
		bsp := sdktrace.NewBatchSpanProcessor(traceExporter)
		tp = sdktrace.NewTracerProvider(
			sdktrace.WithResource(res),
			sdktrace.WithSpanProcessor(bsp),
		)
		log.Println("Set up OpenTelemetry Tracer Provider with OTLP Exporter")
	}

	otel.SetTracerProvider(tp)
	// otel.SetTextMapPropagator(...) // Setup propagator if needed

	// --- Metrics Provider Setup (Prometheus) ---
	promExporter, err := prometheus.New()
	if err != nil {
		return nil, fmt.Errorf("failed to create Prometheus exporter: %w", err)
	}
	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(res),
		sdkmetric.WithReader(promExporter),
	)
	otel.SetMeterProvider(mp) // Set the global MeterProvider
	log.Println("Set up OpenTelemetry Meter Provider with Prometheus Exporter")

	// --- Start Prometheus Metrics Endpoint ---
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	metricsServer := &http.Server{Addr: metricsAddr, Handler: mux}
	go func() {
		log.Printf("Starting Prometheus metrics server on %s", metricsAddr)
		if err := metricsServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("Metrics server error: %v", err)
		}
	}()

	// --- Return Shutdown Function ---
	shutdown := func(ctx context.Context) error {
		var shutdownErr error
		log.Println("Shutting down OpenTelemetry providers and metrics server...")
		// Shutdown metrics server first
		if err := metricsServer.Shutdown(ctx); err != nil {
			shutdownErr = errors.Join(shutdownErr, fmt.Errorf("metrics server shutdown error: %w", err))
		} else {
			log.Println("Metrics server stopped.")
		}
		// Shutdown MeterProvider
		if err := mp.Shutdown(ctx); err != nil {
			shutdownErr = errors.Join(shutdownErr, fmt.Errorf("OTel Meter Provider shutdown error: %w", err))
		} else {
			log.Println("OTel Meter Provider stopped.")
		}
		// Shutdown TracerProvider
		if err := tp.Shutdown(ctx); err != nil {
			shutdownErr = errors.Join(shutdownErr, fmt.Errorf("OTel Tracer Provider shutdown error: %w", err))
		} else {
			log.Println("OTel Tracer Provider stopped.")
		}
		return shutdownErr
	}

	return shutdown, nil
}
