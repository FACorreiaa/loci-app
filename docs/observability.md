# Loci TemplUI - HTTP-Only Observability Stack

## Overview
Complete observability stack using HTTP-only protocols with OpenTelemetry, Prometheus, Grafana, Loki, and Tempo.

## Architecture

```
┌─────────────────┐    HTTP/4318     ┌──────────────────┐
│  Loci TemplUI  │ ───────────────> │ OTEL Collector  │
│     App         │                  │                  │
└─────────────────┘                  └──────────────────┘
                                               │
                                               │ HTTP
                                               ▼
┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐
│ Prometheus  │  │    Loki     │  │   Tempo     │  │   Grafana   │
│   :9090     │  │   :3100     │  │   :3200     │  │   :3000     │
└─────────────┘  └─────────────┘  └─────────────┘  └─────────────┘
```

## Components

### 1. Application (Loci TemplUI)
- **Port**: 8091
- **Metrics endpoint**: `/metrics` 
- **OTEL Middleware**: Gin integration with automatic tracing
- **Configuration**: Environment variables for OTEL

### 2. OTEL Collector  
- **Port**: 4318 (HTTP OTLP receiver)
- **Metrics**: 8889 (Prometheus format)
- **Protocol**: HTTP only (no gRPC)
- **Exports to**: Prometheus, Tempo, Loki

### 3. Prometheus
- **Port**: 9090
- **Scrapes**: App metrics (8091), OTEL Collector (8889)
- **Config**: Fixed scrape intervals and timeouts

### 4. Grafana
- **Port**: 3000  
- **Credentials**: admin/admin
- **Data sources**: Prometheus, Loki, Tempo

### 5. Loki (Logs)
- **Port**: 3100
- **Schema**: v13 with tsdb
- **Storage**: Local filesystem

### 6. Tempo (Traces)  
- **Port**: 3200
- **Protocol**: HTTP OTLP receiver
- **Storage**: Local filesystem

## Environment Variables

Required environment variables in `.env`:

```bash
# OpenTelemetry Configuration  
OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318
OTEL_EXPORTER_OTLP_PROTOCOL=http/protobuf
OTEL_SERVICE_NAME=loci-templui
OTEL_RESOURCE_ATTRIBUTES=service.name=loci-templui,service.version=1.0.0
OTEL_LOG_LEVEL=info
OTEL_SDK_DISABLED=false

# Server
SERVER_PORT=8091
METRICS_PORT=9090
```

## Usage

### Start Observability Stack
```bash
docker-compose up -d
```

### Start Application
```bash
# Loads .env automatically
./bin/loci-app
```

### Test Observability
```bash
./scripts/test-observability.sh
```

### Access Points
- **Grafana Dashboard**: http://localhost:3000 (admin/admin)
- **Prometheus**: http://localhost:9090  
- **Application**: http://localhost:8091
- **App Metrics**: http://localhost:8091/metrics
- **OTEL Internal Metrics**: http://localhost:9092/metrics
- **OTEL Collector**: http://localhost:8889/metrics

## Gin OTEL Middleware

The application uses official OpenTelemetry Gin middleware:

```go
// In main.go
r.Use(middleware.OTELGinMiddleware("loci-templui"))

// In app/pkg/middleware/otel.go  
func OTELGinMiddleware(serviceName string) gin.HandlerFunc {
    return otelgin.Middleware(serviceName)
}
```

This automatically:
- Creates spans for each HTTP request
- Adds tracing headers
- Records HTTP metrics  
- Integrates with OpenTelemetry context propagation

## Fixed Issues

1. **OTEL Collector**: Added missing `check_interval` for memory_limiter
2. **Prometheus**: Fixed scrape timeout/interval mismatch  
3. **Loki**: Updated to v13 schema with tsdb, removed deprecated `shared_store`
4. **Docker Volumes**: Fixed mount paths for all services
5. **Gin Integration**: Proper OTEL middleware instead of custom implementation
6. **Schema Conflict**: Fixed OpenTelemetry schema version conflict by updating semconv to v1.37.0 and simplifying resource creation

## Benefits

- ✅ **HTTP-Only**: No gRPC dependencies, simpler network setup
- ✅ **Automatic Tracing**: Every HTTP request traced automatically
- ✅ **Comprehensive Metrics**: HTTP requests, database queries, custom app metrics
- ✅ **Centralized Logs**: All logs aggregated in Loki
- ✅ **Unified Dashboard**: Single Grafana interface for all observability data
- ✅ **Production Ready**: Proper error handling and configuration