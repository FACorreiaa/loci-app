#!/bin/bash

# Test Observability Stack
echo "🔧 Testing Loci TemplUI Observability Stack..."

# Check if Docker Compose is running
echo "📊 Starting observability services..."
docker-compose up -d

echo "⏳ Waiting for services to start..."
sleep 10

# Test endpoints
echo "🧪 Testing service endpoints..."

echo "- Prometheus (metrics): http://localhost:9090"
curl -s -o /dev/null -w "Status: %{http_code}\n" http://localhost:9090

echo "- Grafana (visualization): http://localhost:3000"
curl -s -o /dev/null -w "Status: %{http_code}\n" http://localhost:3000

echo "- Loki (logs): http://localhost:3100/ready"
curl -s -o /dev/null -w "Status: %{http_code}\n" http://localhost:3100/ready

echo "- Tempo (traces): http://localhost:3200/ready"
curl -s -o /dev/null -w "Status: %{http_code}\n" http://localhost:3200/ready

echo "- OTEL Collector (metrics): http://localhost:8889/metrics"
curl -s -o /dev/null -w "Status: %{http_code}\n" http://localhost:8889/metrics

echo ""
echo "✅ Observability stack test complete!"
echo ""
echo "📈 Access points:"
echo "  - Grafana Dashboard: http://localhost:3000 (admin/admin)"
echo "  - Prometheus: http://localhost:9090"
echo "  - Application Metrics: http://localhost:8091/metrics"
echo ""
echo "🚀 Start your app with: ./bin/loci-app"