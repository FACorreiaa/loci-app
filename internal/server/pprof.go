package server

import (
	"log"

	"github.com/gin-contrib/pprof"
	"github.com/gin-gonic/gin"
)

// StartPprofServer starts the pprof server on a separate port
// This should only be accessible internally or via SSH tunnel
func StartPprofServer(port string) {
	pprofRouter := gin.New()
	pprof.Register(pprofRouter)

	go func() {
		log.Printf("Starting pprof server on %s", port)
		if err := pprofRouter.Run(port); err != nil {
			log.Fatalf("Failed to start pprof server: %v", err)
		}
	}()
}
