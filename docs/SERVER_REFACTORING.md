# Server Refactoring - Before and After

## Overview

This document shows the before and after comparison of the server refactoring to help understand the improvements made.

## Main.go Comparison

### Before (179 lines)

```go
func main() {
    // Load .env
    err := godotenv.Load()
    if err != nil {
        log.Fatal("Error loading .env file")
    }

    // Create logger inline
    logger, err := zap.NewProduction()
    if err != nil {
        log.Fatal("Error initializing zap logger")
    }
    defer logger.Sync()

    // Load config inline
    cfg, err := config.Load()
    if err != nil {
        logger.Fatal("Failed to load configuration", zap.Error(err))
    }

    // Initialize observability inline
    otelShutdown, err := tracer.InitOtelProviders("loci-templui", ":9092")
    if err != nil {
        logger.Fatal("Failed to initialize OpenTelemetry", zap.Error(err))
    }
    defer func() {
        if err := otelShutdown(context.Background()); err != nil {
            logger.Error("Failed to shutdown OpenTelemetry", zap.Error(err))
        }
    }()

    // Initialize metrics inline
    metrics.InitAppMetrics()
    logger.Info("Observability initialized", zap.String("metrics_endpoint", ":9092/metrics"))

    // Setup database inline with lots of code
    ctx := context.Background()
    dbPool, err := setupDatabase(ctx, cfg, logger)
    if err != nil {
        logger.Fatal("Failed to setup database", zap.Error(err))
    }
    defer dbPool.Close()

    // Configure Gin
    gin.SetMode(gin.ReleaseMode)
    r := gin.New()

    // Setup assets inline
    if err := SetupAssets(r); err != nil {
        logger.Fatal("Failed to setup assets", zap.Error(err))
        panic(err)
    }

    // Configure ALL middleware inline (30+ lines)
    r.Use(ginzap.GinzapWithConfig(logger, &ginzap.Config{
        // ... lots of config
    }))
    r.Use(ginzap.RecoveryWithZap(logger, true))
    r.Use(middleware.OTELGinMiddleware("loci-templui"))
    r.Use(gin.Recovery())
    r.Use(middleware.CORSMiddleware())
    r.Use(middleware.SecurityMiddleware())
    r.Use(func(c *gin.Context) {
        c.Set("db", dbPool)
        c.Next()
    })

    // Setup routes
    routes.Setup(r, dbPool, logger)

    // Setup pprof inline
    pprofRouter := gin.New()
    pprof.Register(pprofRouter)
    go func() {
        log.Println("Starting pprof server on :6060")
        if err := pprofRouter.Run(":6060"); err != nil {
            log.Fatalf("Failed to start pprof server: %v", err)
        }
    }()

    // Start server - no graceful shutdown
    serverPort := ":" + cfg.ServerPort
    logger.Info("Server starting", zap.String("port", cfg.ServerPort))
    if err := r.Run(serverPort); err != nil {
        logger.Fatal("Failed to start server", zap.Error(err))
    }
}

func setupDatabase(ctx context.Context, cfg *config.Config, logger *zap.Logger) (*pgxpool.Pool, error) {
    // 25+ lines of database setup code
}
```

**Issues**:
- Too much responsibility in one function
- Hard to test
- No graceful shutdown
- Middleware configuration mixed with setup
- Database setup code in main package

### After (90 lines)

```go
func main() {
    if err := run(); err != nil {
        log.Fatal(err)
    }
}

func run() error {
    // Load environment variables
    if err := godotenv.Load(); err != nil {
        log.Println("Warning: Error loading .env file, using environment variables")
    }

    // Initialize logger
    logger, err := zap.NewProduction()
    if err != nil {
        return err
    }
    defer logger.Sync()

    // Load configuration
    cfg, err := config.Load()
    if err != nil {
        return err
    }

    // Initialize observability
    otelShutdown, err := server.InitObservability("loci-templui", ":9092", logger)
    if err != nil {
        return err
    }
    defer func() {
        if err := otelShutdown(context.Background()); err != nil {
            logger.Error("Failed to shutdown OpenTelemetry", zap.Error(err))
        }
    }()

    // Create server
    srv, err := server.New(cfg, logger)
    if err != nil {
        return err
    }
    defer srv.Close()

    // Setup router
    router := server.SetupRouter(srv.GetDBPool(), logger)

    // Setup assets
    if err := server.SetupAssets(router); err != nil {
        logger.Error("Failed to setup assets", zap.Error(err))
        return err
    }

    // Set the router on the server
    srv.SetRouter(router)

    // Start pprof server (on separate port, not exposed publicly)
    server.StartPprofServer(":6060")

    // Create HTTP server
    httpServer := srv.HTTPServer()

    // Setup graceful shutdown
    done := make(chan bool, 1)
    go server.GracefulShutdown(httpServer, logger, done)

    // Start server
    logger.Info("Server starting", zap.String("port", cfg.ServerPort))
    if err := httpServer.ListenAndServe(); err != nil {
        logger.Error("Server error", zap.Error(err))
    }

    // Wait for graceful shutdown to complete
    <-done
    logger.Info("Graceful shutdown complete")

    return nil
}
```

**Benefits**:
- Clear, linear flow
- Proper error handling
- Testable `run()` function
- Graceful shutdown included
- Each concern delegated to appropriate package

## Architecture Comparison

### Before

```
main.go (179 lines)
├── Direct logger creation
├── Direct config loading
├── Direct observability init
├── Direct database setup
├── Direct Gin setup
├── Direct middleware config (inline)
├── Direct route setup
├── Direct pprof setup
└── Direct server start (no graceful shutdown)
```

### After

```
main.go (90 lines)
└── run()
    ├── Logger creation
    ├── Config loading
    ├── server.InitObservability()
    ├── server.New()
    │   ├── Database setup
    │   └── Dependencies injection
    ├── server.SetupRouter()
    │   ├── Middleware config
    │   └── Route setup
    ├── server.SetupAssets()
    ├── server.StartPprofServer()
    ├── server.HTTPServer()
    └── server.GracefulShutdown()
```

## Dependency Injection Comparison

### Before

```go
// Dependencies scattered everywhere
func main() {
    logger := /* ... */
    cfg := /* ... */
    dbPool := /* ... */
    r := gin.New()
    // Each part accesses what it needs directly
}
```

### After

```go
// Server struct holds all dependencies
type Server struct {
    cfg    *config.Config
    logger *zap.Logger
    dbPool *pgxpool.Pool
    router http.Handler
}

// Everything injected at creation
srv, err := server.New(cfg, logger)
```

## Testing Comparison

### Before

```go
// Hard to test - everything in main()
// Would need to test entire application startup
```

### After

```go
// Can test run() function
func TestRun(t *testing.T) {
    // Set up test environment
    // Call run()
    // Verify behavior
}

// Can test server creation
func TestServerNew(t *testing.T) {
    cfg := &config.Config{/* ... */}
    logger := zap.NewNop()
    srv, err := server.New(cfg, logger)
    // Assertions
}

// Can test router setup
func TestSetupRouter(t *testing.T) {
    dbPool := /* mock */
    logger := zap.NewNop()
    router := server.SetupRouter(dbPool, logger)
    // Assertions
}
```

## Middleware Setup Comparison

### Before (in main.go)

```go
r.Use(ginzap.GinzapWithConfig(logger, &ginzap.Config{
    UTC:        true,
    TimeFormat: time.RFC3339,
    Context: ginzap.Fn(func(c *gin.Context) []zapcore.Field {
        // 30+ lines of inline function
    }),
}))
r.Use(ginzap.RecoveryWithZap(logger, true))
r.Use(middleware.OTELGinMiddleware("loci-templui"))
r.Use(gin.Recovery())
r.Use(middleware.CORSMiddleware())
r.Use(middleware.SecurityMiddleware())
r.Use(func(c *gin.Context) {
    c.Set("db", dbPool)
    c.Next()
})
```

### After (in internal/server/router.go)

```go
func SetupRouter(dbPool *pgxpool.Pool, logger *zap.Logger) *gin.Engine {
    r := gin.New()

    r.Use(ginzap.GinzapWithConfig(logger, &ginzap.Config{
        UTC:        true,
        TimeFormat: time.RFC3339,
        Context:    zapContextFunc(), // Extracted to separate function
    }))
    r.Use(ginzap.RecoveryWithZap(logger, true))
    r.Use(middleware.OTELGinMiddleware("loci-templui"))
    r.Use(gin.Recovery())
    r.Use(middleware.CORSMiddleware())
    r.Use(middleware.SecurityMiddleware())
    r.Use(func(c *gin.Context) {
        c.Set("db", dbPool)
        c.Next()
    })

    routes.Setup(r, dbPool, logger)
    return r
}

func zapContextFunc() ginzap.Fn {
    // 30+ lines in separate, testable function
}
```

## Key Improvements Summary

1. **Separation of Concerns**
   - Main: Application entry point
   - Server: Server lifecycle and dependencies
   - Router: HTTP routing and middleware
   - Each concern in its own file

2. **Error Handling**
   - Before: `log.Fatal()` scattered everywhere
   - After: Errors propagate through `run()`, handled in one place

3. **Testability**
   - Before: Everything in main, hard to test
   - After: Testable functions in server package

4. **Graceful Shutdown**
   - Before: None
   - After: Proper signal handling with 5-second timeout

5. **Code Organization**
   - Before: 179 lines in main.go
   - After: 90 lines in main.go, organized code in internal/server

6. **Dependency Management**
   - Before: Dependencies created and used inline
   - After: Dependencies injected through Server struct

7. **Maintainability**
   - Before: Changes require modifying main.go
   - After: Changes isolated to appropriate packages
