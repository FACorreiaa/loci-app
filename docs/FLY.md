# Fly.io PostgreSQL Setup Guide with pgvector and PostGIS

This guide walks you through creating and integrating a PostgreSQL database on Fly.io with pgvector and PostGIS extensions for your Go application.

## Table of Contents
1. [Prerequisites](#prerequisites)
2. [Create PostgreSQL Cluster](#create-postgresql-cluster)
3. [Enable Extensions](#enable-extensions)
4. [Connect to Database](#connect-to-database)
5. [Environment Variables Setup](#environment-variables-setup)
6. [Deploy Application](#deploy-application)
7. [Troubleshooting](#troubleshooting)

## Prerequisites

- [Fly.io CLI](https://fly.io/docs/hands-on/install-flyctl/) installed
- Fly.io account created and authenticated (`flyctl auth login`)
- Go application ready to deploy

## Create PostgreSQL Cluster

### 1. Create a new Postgres app

```bash
flyctl postgres create
```

You'll be prompted for:
- **App Name**: Choose a name (e.g., `loci-db`, `templui-postgres`)
- **Organization**: Select your organization
- **Region**: Choose closest to your users (e.g., `iad` for US East)
- **Configuration**: Select based on needs:
  - `Development - Single node, 1x shared CPU, 256MB RAM, 1GB disk`
  - `Production - Highly available, 1x shared CPU, 256MB RAM, 10GB disk`
  - `Production - Highly available, 1x shared CPU, 512MB RAM, 10GB disk`
  - Custom configuration

Example output:
```
? Choose an app name (leave blank to generate one): loci-db
automatically selected personal organization: Your Name
? Select region: iad (Ashburn, Virginia (US))
? Select configuration: Development - Single node, 1x shared CPU, 256MB RAM, 1GB disk
Creating postgres cluster in organization personal
Creating app...
Setting secrets on app loci-db...
Provisioning 1 of 1 machines with image flyio/postgres:15.3
Waiting for machine to start...
Machine <machine-id> is created
==> Monitoring health checks
```

### 2. Note the connection details

Save these from the output:
```
Username:    postgres
Password:    <generated-password>
Hostname:    loci-db.internal
Flycast:     fdaa:X:XXXX:X:X:X:X:X
Proxy port:  5432
Postgres port: 5433
Connection string: postgres://postgres:<password>@loci-db.internal:5432
```

**IMPORTANT**: Save the password immediately - you won't see it again!

### 3. Check cluster status

```bash
flyctl postgres list
flyctl status -a loci-db
```

## Enable Extensions

### 1. Connect to the Postgres instance

```bash
flyctl postgres connect -a loci-db
```

This opens a `psql` session.

### 2. Enable pgvector extension

pgvector enables vector similarity search for embeddings (useful for AI/ML features).

```sql
-- Connect to your database
\c <your-database-name>

-- Enable pgvector extension
CREATE EXTENSION IF NOT EXISTS vector;

-- Verify installation
\dx vector
```

### 3. Enable PostGIS extension

PostGIS adds geographic object support for location-based queries.

```sql
-- Enable PostGIS
CREATE EXTENSION IF NOT EXISTS postgis;

-- Enable PostGIS topology
CREATE EXTENSION IF NOT EXISTS postgis_topology;

-- Enable PostGIS raster (optional, for raster data)
CREATE EXTENSION IF NOT EXISTS postgis_raster;

-- Verify installations
\dx postgis*
```

### 4. Verify extensions

```sql
SELECT
    extname AS "Extension",
    extversion AS "Version"
FROM pg_extension
WHERE extname IN ('vector', 'postgis', 'postgis_topology', 'postgis_raster');
```

Expected output:
```
     Extension      | Version
--------------------+---------
 vector             | 0.5.1
 postgis            | 3.4.0
 postgis_topology   | 3.4.0
 postgis_raster     | 3.4.0
```

### 5. Exit psql

```sql
\q
```

## Connect to Database

### Create application database

If you haven't created your app-specific database:

```bash
flyctl postgres connect -a loci-db
```

```sql
CREATE DATABASE templui_prod;
\c templui_prod

-- Enable extensions in the new database
CREATE EXTENSION IF NOT EXISTS vector;
CREATE EXTENSION IF NOT EXISTS postgis;
CREATE EXTENSION IF NOT EXISTS postgis_topology;

\q
```

## Environment Variables Setup

### 1. Attach database to your application

First, ensure you have a fly.toml for your app:

```bash
flyctl launch --no-deploy
```

Then attach the Postgres cluster:

```bash
flyctl postgres attach loci-db -a <your-app-name>
```

This automatically creates a `DATABASE_URL` secret in your app.

### 2. Verify the secret was created

```bash
flyctl secrets list -a <your-app-name>
```

You should see:
```
NAME          DIGEST                  CREATED AT
DATABASE_URL  <digest>                <timestamp>
```

### 3. Set additional environment variables

Add other required secrets:

```bash
# Database configuration
flyctl secrets set \
  DB_HOST=loci-db.internal \
  DB_PORT=5432 \
  DB_NAME=templui_prod \
  DB_USER=postgres \
  DB_PASSWORD=<your-postgres-password> \
  DB_SSLMODE=disable \
  -a <your-app-name>

# API keys and other config
flyctl secrets set \
  OPENAI_API_KEY=<your-openai-key> \
  GEMINI_API_KEY=<your-gemini-key> \
  MAPBOX_ACCESS_TOKEN=<your-mapbox-token> \
  SESSION_SECRET=<generate-random-string> \
  -a <your-app-name>

# Server configuration
flyctl secrets set \
  SERVER_PORT=8080 \
  GIN_MODE=release \
  -a <your-app-name>
```

### 4. Update fly.toml

Add environment variables section to your `fly.toml`:

```toml
app = "your-app-name"
primary_region = "iad"

[build]
  [build.args]
    GO_VERSION = "1.21"

[env]
  SERVER_PORT = "8080"
  GIN_MODE = "release"

[http_service]
  internal_port = 8080
  force_https = true
  auto_stop_machines = true
  auto_start_machines = true
  min_machines_running = 0
  processes = ["app"]

[[vm]]
  cpu_kind = "shared"
  cpus = 1
  memory_mb = 256
```

### 5. Local development .env file

For local development, create a `.env` file:

```bash
# Database
DB_HOST=localhost
DB_PORT=5432
DB_NAME=templui_dev
DB_USER=postgres
DB_PASSWORD=postgres
DB_SSLMODE=disable
DATABASE_URL=postgres://postgres:postgres@localhost:5432/templui_dev?sslmode=disable

# Server
SERVER_PORT=8080
GIN_MODE=debug

# API Keys
OPENAI_API_KEY=your_openai_key_here
GEMINI_API_KEY=your_gemini_key_here
MAPBOX_ACCESS_TOKEN=your_mapbox_token_here

# Session
SESSION_SECRET=your_session_secret_here
```

**IMPORTANT**: Add `.env` to `.gitignore`!

## Deploy Application

### 1. Ensure your config package loads environment variables

Check `app/pkg/config/config.go`:

```go
package config

import (
    "os"
    "github.com/joho/godotenv"
)

type Config struct {
    Database DatabaseConfig
    Server   ServerConfig
    OpenAI   OpenAIConfig
    Gemini   GeminiConfig
    Map      MapConfig
    Session  SessionConfig
}

type DatabaseConfig struct {
    Host     string
    Port     string
    Name     string
    User     string
    Password string
    SSLMode  string
}

func Load() (*Config, error) {
    // Load .env only in development
    _ = godotenv.Load()

    return &Config{
        Database: DatabaseConfig{
            Host:     os.Getenv("DB_HOST"),
            Port:     os.Getenv("DB_PORT"),
            Name:     os.Getenv("DB_NAME"),
            User:     os.Getenv("DB_USER"),
            Password: os.Getenv("DB_PASSWORD"),
            SSLMode:  os.Getenv("DB_SSLMODE"),
        },
        // ... other configs
    }, nil
}

func (c *DatabaseConfig) DSN() string {
    return fmt.Sprintf(
        "host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
        c.Host, c.Port, c.User, c.Password, c.Name, c.SSLMode,
    )
}
```

### 2. Create a Dockerfile

```dockerfile
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Build
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/bin/templui ./cmd/server

FROM alpine:latest

WORKDIR /app

# Install runtime dependencies
RUN apk --no-cache add ca-certificates tzdata

# Copy binary
COPY --from=builder /app/bin/templui .

# Expose port
EXPOSE 8080

# Run
CMD ["./templui"]
```

### 3. Deploy to Fly.io

```bash
# Deploy
flyctl deploy

# Monitor logs
flyctl logs -a <your-app-name>

# Check app status
flyctl status -a <your-app-name>

# Open in browser
flyctl open -a <your-app-name>
```

### 4. Run database migrations

If you have migrations:

```bash
# SSH into your app
flyctl ssh console -a <your-app-name>

# Run migrations
./templui migrate up

# Or connect to DB and run SQL
flyctl postgres connect -a loci-db
```

## Troubleshooting

### Connection Issues

**Issue**: App can't connect to database

```bash
# Check if database is attached
flyctl postgres list
flyctl postgres users list -a loci-db

# Verify DATABASE_URL secret
flyctl secrets list -a <your-app-name>

# Test connection from app machine
flyctl ssh console -a <your-app-name>
nc -zv loci-db.internal 5432
```

### Extension Not Found

**Issue**: `ERROR: extension "vector" is not available`

```bash
# Check Postgres version (pgvector requires 11+)
flyctl postgres connect -a loci-db
SELECT version();

# Verify extension files exist
\dx
```

### Performance Issues

**Issue**: Database is slow

```bash
# Scale up database resources
flyctl scale vm shared-cpu-1x --memory 512 -a loci-db

# Check database metrics
flyctl dashboard -a loci-db
```

### View logs

```bash
# Application logs
flyctl logs -a <your-app-name>

# Database logs
flyctl logs -a loci-db

# Real-time logs
flyctl logs -a <your-app-name> -f
```

### Reset database password

```bash
flyctl postgres connect -a loci-db

ALTER USER postgres WITH PASSWORD 'new_secure_password';
\q

# Update secret in your app
flyctl secrets set DB_PASSWORD=new_secure_password -a <your-app-name>
```

## Useful Commands Reference

```bash
# Postgres Management
flyctl postgres list                          # List all Postgres apps
flyctl postgres connect -a <db-name>         # Connect via psql
flyctl postgres db list -a <db-name>         # List databases
flyctl postgres users list -a <db-name>      # List users

# App Management
flyctl apps list                             # List all apps
flyctl status -a <app-name>                  # App status
flyctl logs -a <app-name>                    # View logs
flyctl ssh console -a <app-name>             # SSH into app

# Secrets Management
flyctl secrets list -a <app-name>            # List secrets
flyctl secrets set KEY=value -a <app-name>   # Set secret
flyctl secrets unset KEY -a <app-name>       # Remove secret

# Scaling
flyctl scale show -a <app-name>              # Show current scale
flyctl scale count 2 -a <app-name>           # Scale to 2 instances
flyctl scale vm shared-cpu-1x --memory 512 -a <app-name>  # Change resources

# Monitoring
flyctl dashboard -a <app-name>               # Open dashboard
flyctl metrics -a <app-name>                 # View metrics
```

## Additional Resources

- [Fly.io Postgres Documentation](https://fly.io/docs/postgres/)
- [pgvector GitHub](https://github.com/pgvector/pgvector)
- [PostGIS Documentation](https://postgis.net/documentation/)
- [Fly.io Go Guide](https://fly.io/docs/languages-and-frameworks/golang/)

## Security Best Practices

1. **Never commit secrets**: Add `.env` to `.gitignore`
2. **Use strong passwords**: Generate random passwords for production
3. **Enable SSL**: Set `DB_SSLMODE=require` in production
4. **Rotate secrets regularly**: Change passwords and API keys periodically
5. **Limit access**: Use firewall rules and private networking
6. **Monitor logs**: Regularly check for suspicious activity

---

**Last Updated**: 2025-10-23
**Maintainer**: Loci Team
