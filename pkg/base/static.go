package base

import (
	"embed"
	"io/fs"
	"net/http"
	"path/filepath"
	"strings"
)

// StaticConfig configures static file serving behavior
type StaticConfig struct {
	// AssetsDir is the subdirectory within the embedded FS (e.g., "assets", "dist")
	AssetsDir string
	// SPAMode enables Single Page Application routing (serves index.html for non-file routes)
	SPAMode bool
	// DevMode disables static serving (for development when using dev server)
	DevMode bool
	// APIPrefix excludes paths starting with this prefix from static serving
	APIPrefix string
}

// StaticHandler creates a configurable HTTP handler for serving embedded static files
func StaticHandler(assets embed.FS, config StaticConfig) http.Handler {
	// Set defaults
	if config.APIPrefix == "" {
		config.APIPrefix = "/api/"
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// In development mode, don't serve static files (they should be proxied)
		if config.DevMode {
			http.NotFound(w, r)
			return
		}

		// Don't serve static files for API endpoints
		if strings.HasPrefix(r.URL.Path, config.APIPrefix) {
			http.NotFound(w, r)
			return
		}

		// Extract and clean the path
		path := strings.TrimPrefix(r.URL.Path, "/")
		if path == "" {
			path = "index.html"
		}

		// Create sub-filesystem if assets are in a subdirectory
		var assetsFS fs.FS = assets
		if config.AssetsDir != "" {
			var err error
			assetsFS, err = fs.Sub(assets, config.AssetsDir)
			if err != nil {
				http.Error(w, "Assets not available", http.StatusNotFound)
				return
			}
		}

		// Check if file exists
		if _, err := assetsFS.Open(path); err != nil {
			// SPA mode: serve index.html for routes that don't look like files
			if config.SPAMode && !strings.Contains(path, ".") {
				path = "index.html"
				if _, err := assetsFS.Open(path); err != nil {
					http.NotFound(w, r)
					return
				}
			} else {
				http.NotFound(w, r)
				return
			}
		}

		// Set content type and cache headers
		setContentType(w, path)
		setCacheHeaders(w, path)

		// Serve the file
		http.FileServer(http.FS(assetsFS)).ServeHTTP(w, r)
	})
}

// setContentType sets appropriate content type based on file extension
func setContentType(w http.ResponseWriter, path string) {
	ext := filepath.Ext(path)
	contentTypes := map[string]string{
		".html":  "text/html; charset=utf-8",
		".css":   "text/css; charset=utf-8",
		".js":    "application/javascript; charset=utf-8",
		".json":  "application/json; charset=utf-8",
		".png":   "image/png",
		".jpg":   "image/jpeg",
		".jpeg":  "image/jpeg",
		".svg":   "image/svg+xml",
		".ico":   "image/x-icon",
		".woff":  "font/woff",
		".woff2": "font/woff",
		".ttf":   "font/ttf",
	}

	if contentType, ok := contentTypes[ext]; ok {
		w.Header().Set("Content-Type", contentType)
	}
}

// setCacheHeaders sets appropriate cache headers
func setCacheHeaders(w http.ResponseWriter, path string) {
	ext := filepath.Ext(path)

	// Long cache for assets, short cache for HTML
	if strings.Contains(path, "assets/") || ext == ".css" || ext == ".js" {
		w.Header().Set("Cache-Control", "public, max-age=31536000") // 1 year
	} else {
		w.Header().Set("Cache-Control", "public, max-age=300") // 5 minutes
	}
}
