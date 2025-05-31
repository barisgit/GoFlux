package base

import (
	"embed"
	"io/fs"
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

// StaticResponse contains the result of static file processing
type StaticResponse struct {
	StatusCode   int
	ContentType  string
	CacheControl string
	Body         []byte
	NotFound     bool
}

// ServeStaticFile is the core logic for serving static files, router-agnostic
func ServeStaticFile(assets embed.FS, config StaticConfig, path string) StaticResponse {
	// Set defaults
	if config.APIPrefix == "" {
		config.APIPrefix = "/api/"
	}

	// In development mode, don't serve static files
	if config.DevMode {
		return StaticResponse{StatusCode: 404, NotFound: true}
	}

	switch config.APIPrefix {
	case "none":
		// Server static files for all paths
		break
	default:
		// Don't serve static files for API endpoints
		if strings.HasPrefix(path, config.APIPrefix) {
			return StaticResponse{StatusCode: 404, NotFound: true}
		}
	}

	// Extract and clean the path
	cleanPath := strings.TrimPrefix(path, "/")
	if cleanPath == "" {
		cleanPath = "index.html"
	}

	// Create sub-filesystem if assets are in a subdirectory
	var assetsFS fs.FS = assets

	// Assets will always be embedded in assets directory we need to move them to the root because we embed as assets/*
	{
		var err error
		assetsFS, err = fs.Sub(assets, "assets")
		if err != nil {
			return StaticResponse{StatusCode: 404, NotFound: true}
		}
	}

	// Then if the user has specified a different assets directory, we need to move them up again
	if config.AssetsDir != "" {
		var err error
		assetsFS, err = fs.Sub(assets, config.AssetsDir)
		if err != nil {
			return StaticResponse{StatusCode: 404, NotFound: true}
		}
	}

	// Check if file exists
	file, err := assetsFS.Open(cleanPath)
	if err != nil {
		// SPA mode: serve index.html for routes that don't look like files
		if config.SPAMode && !strings.Contains(cleanPath, ".") {
			cleanPath = "index.html"
			file, err = assetsFS.Open(cleanPath)
			if err != nil {
				return StaticResponse{StatusCode: 404, NotFound: true}
			}
		} else {
			return StaticResponse{StatusCode: 404, NotFound: true}
		}
	}
	defer file.Close()

	// Read file content
	stat, err := file.Stat()
	if err != nil {
		return StaticResponse{StatusCode: 500, NotFound: true}
	}

	body := make([]byte, stat.Size())
	_, err = file.Read(body)
	if err != nil {
		return StaticResponse{StatusCode: 500, NotFound: true}
	}

	return StaticResponse{
		StatusCode:   200,
		ContentType:  getContentType(cleanPath),
		CacheControl: getCacheControl(cleanPath),
		Body:         body,
		NotFound:     false,
	}
}

// Helper functions for content type and cache control
func getContentType(path string) string {
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
		".woff2": "font/woff2",
		".ttf":   "font/ttf",
	}

	if contentType, ok := contentTypes[ext]; ok {
		return contentType
	}
	return "application/octet-stream"
}

func getCacheControl(path string) string {
	ext := filepath.Ext(path)
	// Long cache for assets, short cache for HTML
	if strings.Contains(path, "assets/") || ext == ".css" || ext == ".js" {
		return "public, max-age=31536000" // 1 year
	}
	return "public, max-age=300" // 5 minutes
}
