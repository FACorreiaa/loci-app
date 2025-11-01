package main // Same as main.go

import (
	"embed"
	"io/fs"
	"net/http"

	"github.com/gin-gonic/gin"
)

//go:embed assets/css/* assets/js/* assets/static/*
var Assets embed.FS

func SetupAssets(r *gin.Engine) error { // Renamed from SetupStatic for consistency with your code
	staticFiles, err := fs.Sub(Assets, "assets")
	if err != nil {
		return err // Return err instead of panic
	}
	r.StaticFS("/assets", http.FS(staticFiles))
	return nil
}
