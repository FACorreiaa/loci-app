package main

import (
	"context"
	"log"

	"github.com/joho/godotenv"
	"go.uber.org/zap"

	"github.com/FACorreiaa/go-templui/internal/pkg/config"
	"github.com/FACorreiaa/go-templui/internal/server"
)

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
