package server

import (
	"io/fs"
	"net/http"

	"github.com/FACorreiaa/go-templui/assets"
	"github.com/gin-gonic/gin"
)

// SetupAssets configures static asset serving for the Gin router
func SetupAssets(r *gin.Engine) error {
	staticFiles, err := fs.Sub(assets.Assets, ".")
	if err != nil {
		return err
	}
	r.StaticFS("/assets", http.FS(staticFiles))
	return nil
}
