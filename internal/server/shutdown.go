package server

import (
	"context"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"
)

// GracefulShutdown handles graceful shutdown of the HTTP server
func GracefulShutdown(srv *http.Server, logger *zap.Logger, done chan bool) {
	// Create context that listens for the interrupt signal from the OS
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Listen for the interrupt signal
	<-ctx.Done()
	logger.Info("Shutting down gracefully, press Ctrl+C again to force")

	stop() // Allow Ctrl+C to force shutdown

	// The context is used to inform the server it has 5 seconds to finish
	// the request it is currently handling
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("Server forced to shutdown", zap.Error(err))
	}

	logger.Info("Server exiting")

	// Notify the main goroutine that the shutdown is complete
	done <- true
}
