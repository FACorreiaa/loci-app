package main

import (
	"net/http"

	"github.com/FACorreiaa/go-templui/app/pages"
	"github.com/FACorreiaa/go-templui/assets"
	"github.com/a-h/templ"
)

func main() {
	mux := http.NewServeMux()
	SetupAssetsRoutes(mux)

	// Route handlers
	mux.Handle("GET /", templ.Handler(pages.Landing()))
	mux.Handle("GET /about", templ.Handler(pages.About()))
	mux.Handle("GET /projects", templ.Handler(pages.Projects()))

	// Start server
	http.ListenAndServe(":8090", mux)
}

func SetupAssetsRoutes(mux *http.ServeMux) {
	assetHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Always use embedded assets
		fs := http.FileServer(http.FS(assets.Assets))
		fs.ServeHTTP(w, r)
	})

	mux.Handle("GET /assets/", http.StripPrefix("/assets/", assetHandler))
}