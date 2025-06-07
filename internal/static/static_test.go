package static

import (
	"strings"
	"testing"

	"github.com/barisgit/goflux/internal/testassets"
)

func TestServeStaticFile_BasicFileServing(t *testing.T) {
	config := StaticConfig{
		AssetsDir: "",
		SPAMode:   false,
		DevMode:   false,
		APIPrefix: "/api/",
	}

	tests := []struct {
		name                string
		path                string
		expectedStatus      int
		expectedContentType string
		expectedNotFound    bool
		expectBody          bool
	}{
		{
			name:                "serve index.html",
			path:                "/index.html",
			expectedStatus:      200,
			expectedContentType: "text/html; charset=utf-8",
			expectedNotFound:    false,
			expectBody:          true,
		},
		{
			name:                "serve root path as index.html",
			path:                "/",
			expectedStatus:      200,
			expectedContentType: "text/html; charset=utf-8",
			expectedNotFound:    false,
			expectBody:          true,
		},
		{
			name:                "serve JavaScript file",
			path:                "/app.js",
			expectedStatus:      200,
			expectedContentType: "application/javascript; charset=utf-8",
			expectedNotFound:    false,
			expectBody:          true,
		},
		{
			name:                "serve CSS file",
			path:                "/styles.css",
			expectedStatus:      200,
			expectedContentType: "text/css; charset=utf-8",
			expectedNotFound:    false,
			expectBody:          true,
		},
		{
			name:                "serve JSON file",
			path:                "/data.json",
			expectedStatus:      200,
			expectedContentType: "application/json; charset=utf-8",
			expectedNotFound:    false,
			expectBody:          true,
		},
		{
			name:                "serve SVG file",
			path:                "/images/icon.svg",
			expectedStatus:      200,
			expectedContentType: "image/svg+xml",
			expectedNotFound:    false,
			expectBody:          true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := ServeStaticFile(testassets.TestFS, config, tt.path)

			if response.StatusCode != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, response.StatusCode)
			}

			if response.ContentType != tt.expectedContentType {
				t.Errorf("Expected content type %s, got %s", tt.expectedContentType, response.ContentType)
			}

			if response.NotFound != tt.expectedNotFound {
				t.Errorf("Expected NotFound %v, got %v", tt.expectedNotFound, response.NotFound)
			}

			if tt.expectBody && len(response.Body) == 0 {
				t.Error("Expected non-empty body")
			}

			if response.CacheControl == "" {
				t.Error("Expected Cache-Control header to be set")
			}

			t.Logf("✅ %s: Status=%d, ContentType=%s, CacheControl=%s, BodySize=%d",
				tt.name, response.StatusCode, response.ContentType, response.CacheControl, len(response.Body))
		})
	}
}

func TestServeStaticFile_DevMode(t *testing.T) {
	config := StaticConfig{
		AssetsDir: "",
		SPAMode:   false,
		DevMode:   true,
		APIPrefix: "/api/",
	}

	response := ServeStaticFile(testassets.TestFS, config, "/index.html")

	if response.StatusCode != 404 {
		t.Errorf("Expected status 404 in dev mode, got %d", response.StatusCode)
	}

	if !response.NotFound {
		t.Error("Expected NotFound to be true in dev mode")
	}

	t.Log("✅ Dev mode returns 404 as expected")
}

func TestServeStaticFile_SPAMode(t *testing.T) {
	config := StaticConfig{
		AssetsDir: "",
		SPAMode:   true,
		DevMode:   false,
		APIPrefix: "/api/",
	}

	tests := []struct {
		name             string
		path             string
		expectedStatus   int
		shouldServeIndex bool
	}{
		{
			name:             "SPA route without extension",
			path:             "/users/123",
			expectedStatus:   200,
			shouldServeIndex: true,
		},
		{
			name:             "SPA nested route",
			path:             "/dashboard/settings/profile",
			expectedStatus:   200,
			shouldServeIndex: true,
		},
		{
			name:             "Existing file should serve normally",
			path:             "/app.js",
			expectedStatus:   200,
			shouldServeIndex: false,
		},
		{
			name:             "Non-existent file with extension should 404",
			path:             "/nonexistent.css",
			expectedStatus:   404,
			shouldServeIndex: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := ServeStaticFile(testassets.TestFS, config, tt.path)

			if response.StatusCode != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, response.StatusCode)
			}

			if tt.shouldServeIndex {
				if response.ContentType != "text/html; charset=utf-8" {
					t.Errorf("Expected HTML content type for SPA route, got %s", response.ContentType)
				}
				if len(response.Body) == 0 {
					t.Error("Expected non-empty body for SPA route")
				}
			}

			t.Logf("✅ %s: Status=%d, ShouldServeIndex=%v", tt.name, response.StatusCode, tt.shouldServeIndex)
		})
	}
}

func TestServeStaticFile_APIPrefix(t *testing.T) {
	tests := []struct {
		name        string
		apiPrefix   string
		path        string
		shouldServe bool
	}{
		{
			name:        "default API prefix blocks /api/",
			apiPrefix:   "/api/",
			path:        "/api/users",
			shouldServe: false,
		},
		{
			name:        "custom API prefix blocks custom path",
			apiPrefix:   "/v1/api/",
			path:        "/v1/api/users",
			shouldServe: false,
		},
		{
			name:        "none prefix serves all paths",
			apiPrefix:   "none",
			path:        "/api/users",
			shouldServe: true,
		},
		{
			name:        "empty prefix defaults to /api/",
			apiPrefix:   "",
			path:        "/api/users",
			shouldServe: false,
		},
		{
			name:        "non-API path should serve",
			apiPrefix:   "/api/",
			path:        "/index.html",
			shouldServe: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := StaticConfig{
				AssetsDir: "",
				SPAMode:   false,
				DevMode:   false,
				APIPrefix: tt.apiPrefix,
			}

			response := ServeStaticFile(testassets.TestFS, config, tt.path)

			if tt.shouldServe {
				// Should attempt to serve (may still 404 if file doesn't exist)
				if tt.path == "/index.html" && response.StatusCode != 200 {
					t.Errorf("Expected to serve file, got status %d", response.StatusCode)
				}
			} else {
				// Should not serve, return 404
				if response.StatusCode != 404 || !response.NotFound {
					t.Errorf("Expected API path to be blocked, got status %d, NotFound=%v", response.StatusCode, response.NotFound)
				}
			}

			t.Logf("✅ %s: ShouldServe=%v, Status=%d", tt.name, tt.shouldServe, response.StatusCode)
		})
	}
}

func TestServeStaticFile_AssetsDir(t *testing.T) {
	// Test with different AssetsDir configurations
	tests := []struct {
		name          string
		assetsDir     string
		path          string
		expectSuccess bool
	}{
		{
			name:          "empty assets dir uses default",
			assetsDir:     "",
			path:          "/index.html",
			expectSuccess: true,
		},
		{
			name:          "custom assets dir",
			assetsDir:     "dist",
			path:          "/index.html",
			expectSuccess: false, // Will fail because our test assets are in assets/
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := StaticConfig{
				AssetsDir: tt.assetsDir,
				SPAMode:   false,
				DevMode:   false,
				APIPrefix: "/api/",
			}

			response := ServeStaticFile(testassets.TestFS, config, tt.path)

			if tt.expectSuccess {
				if response.StatusCode != 200 {
					t.Errorf("Expected success, got status %d", response.StatusCode)
				}
			} else {
				if response.StatusCode == 200 {
					t.Errorf("Expected failure, got status %d", response.StatusCode)
				}
			}

			t.Logf("✅ %s: ExpectSuccess=%v, Status=%d", tt.name, tt.expectSuccess, response.StatusCode)
		})
	}
}

func TestServeStaticFile_NotFound(t *testing.T) {
	config := StaticConfig{
		AssetsDir: "",
		SPAMode:   false,
		DevMode:   false,
		APIPrefix: "/api/",
	}

	response := ServeStaticFile(testassets.TestFS, config, "/nonexistent.txt")

	if response.StatusCode != 404 {
		t.Errorf("Expected status 404 for non-existent file, got %d", response.StatusCode)
	}

	if !response.NotFound {
		t.Error("Expected NotFound to be true for non-existent file")
	}

	t.Log("✅ Non-existent file returns 404")
}

func TestServeStaticFile_EmptyFilesystem(t *testing.T) {
	config := StaticConfig{
		AssetsDir: "",
		SPAMode:   false,
		DevMode:   false,
		APIPrefix: "/api/",
	}

	response := ServeStaticFile(testassets.EmptyFS, config, "/index.html")

	if response.StatusCode != 404 {
		t.Errorf("Expected status 404 for empty filesystem, got %d", response.StatusCode)
	}

	if !response.NotFound {
		t.Error("Expected NotFound to be true for empty filesystem")
	}

	t.Log("✅ Empty filesystem returns 404")
}

func TestGetContentType(t *testing.T) {
	tests := []struct {
		path     string
		expected string
	}{
		{"index.html", "text/html; charset=utf-8"},
		{"styles.css", "text/css; charset=utf-8"},
		{"app.js", "application/javascript; charset=utf-8"},
		{"data.json", "application/json; charset=utf-8"},
		{"image.png", "image/png"},
		{"photo.jpg", "image/jpeg"},
		{"picture.jpeg", "image/jpeg"},
		{"icon.svg", "image/svg+xml"},
		{"favicon.ico", "image/x-icon"},
		{"font.woff", "font/woff"},
		{"font.woff2", "font/woff2"},
		{"font.ttf", "font/ttf"},
		{"unknown.xyz", "application/octet-stream"},
		{"noextension", "application/octet-stream"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := getContentType(tt.path)
			if result != tt.expected {
				t.Errorf("Expected content type %s for %s, got %s", tt.expected, tt.path, result)
			}
			t.Logf("✅ %s -> %s", tt.path, result)
		})
	}
}

func TestGetCacheControl(t *testing.T) {
	tests := []struct {
		path     string
		expected string
	}{
		{"assets/app.js", "public, max-age=31536000"},
		{"assets/styles.css", "public, max-age=31536000"},
		{"app.js", "public, max-age=31536000"},
		{"styles.css", "public, max-age=31536000"},
		{"index.html", "public, max-age=300"},
		{"data.json", "public, max-age=300"},
		{"image.png", "public, max-age=300"},
		{"noextension", "public, max-age=300"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := getCacheControl(tt.path)
			if result != tt.expected {
				t.Errorf("Expected cache control %s for %s, got %s", tt.expected, tt.path, result)
			}
			t.Logf("✅ %s -> %s", tt.path, result)
		})
	}
}

func TestServeStaticFile_EdgeCases(t *testing.T) {
	config := StaticConfig{
		AssetsDir: "",
		SPAMode:   false,
		DevMode:   false,
		APIPrefix: "/api/",
	}

	tests := []struct {
		name           string
		path           string
		expectedStatus int
	}{
		{
			name:           "empty path",
			path:           "",
			expectedStatus: 200, // Should serve index.html
		},
		{
			name:           "root path",
			path:           "/",
			expectedStatus: 200, // Should serve index.html
		},
		{
			name:           "path with multiple slashes",
			path:           "//index.html",
			expectedStatus: 200,
		},
		{
			name:           "path without leading slash",
			path:           "index.html",
			expectedStatus: 200,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := ServeStaticFile(testassets.TestFS, config, tt.path)

			if response.StatusCode != tt.expectedStatus {
				t.Errorf("Expected status %d for %s, got %d", tt.expectedStatus, tt.path, response.StatusCode)
			}

			t.Logf("✅ %s: Status=%d", tt.name, response.StatusCode)
		})
	}
}

func TestServeStaticFile_PathTraversal(t *testing.T) {
	config := StaticConfig{
		AssetsDir: "",
		SPAMode:   false,
		DevMode:   false,
		APIPrefix: "/api/",
	}

	// Test path traversal attempts
	maliciousPaths := []string{
		"../../../etc/passwd",
		"..\\..\\..\\windows\\system32\\config\\sam",
		"....//....//....//etc/passwd",
		"%2e%2e%2f%2e%2e%2f%2e%2e%2fetc%2fpasswd",
	}

	for _, path := range maliciousPaths {
		t.Run("path_traversal_"+path, func(t *testing.T) {
			response := ServeStaticFile(testassets.TestFS, config, path)

			// Should return 404 or safely handle the request
			if response.StatusCode == 200 {
				// If it returns 200, make sure it's serving a legitimate file
				if len(response.Body) > 0 {
					bodyStr := string(response.Body)
					// Check that we're not serving system files
					if strings.Contains(bodyStr, "root:") || strings.Contains(bodyStr, "Administrator") {
						t.Errorf("Possible path traversal vulnerability with path: %s", path)
					}
				}
			}

			t.Logf("✅ Path traversal attempt %s handled safely: Status=%d", path, response.StatusCode)
		})
	}
}

func TestServeStaticFile_InvalidAssetsDir(t *testing.T) {
	config := StaticConfig{
		AssetsDir: "nonexistent",
		SPAMode:   false,
		DevMode:   false,
		APIPrefix: "/api/",
	}

	response := ServeStaticFile(testassets.TestFS, config, "/index.html")

	if response.StatusCode != 404 {
		t.Errorf("Expected status 404 for invalid AssetsDir, got %d", response.StatusCode)
	}

	if !response.NotFound {
		t.Error("Expected NotFound to be true for invalid AssetsDir")
	}

	t.Log("✅ Invalid AssetsDir returns 404")
}

func TestServeStaticFile_MalformedPaths(t *testing.T) {
	config := StaticConfig{
		AssetsDir: "",
		SPAMode:   false,
		DevMode:   false,
		APIPrefix: "/api/",
	}

	tests := []struct {
		name           string
		path           string
		expectedStatus int
	}{
		{
			name:           "path with Unicode",
			path:           "/测试.html",
			expectedStatus: 404,
		},
		{
			name:           "path with special characters",
			path:           "/!@#$%^&*()",
			expectedStatus: 404,
		},
		{
			name:           "path with spaces",
			path:           "/file with spaces.html",
			expectedStatus: 404,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := ServeStaticFile(testassets.TestFS, config, tt.path)

			if response.StatusCode != tt.expectedStatus {
				t.Errorf("Expected status %d for %s, got %d", tt.expectedStatus, tt.path, response.StatusCode)
			}

			t.Logf("✅ %s: Status=%d", tt.name, response.StatusCode)
		})
	}
}

func TestServeStaticFile_EmptyAssets(t *testing.T) {
	config := StaticConfig{
		AssetsDir: "",
		SPAMode:   false,
		DevMode:   false,
		APIPrefix: "/api/",
	}

	// Test with empty embedded filesystem
	response := ServeStaticFile(testassets.EmptyFS, config, "/index.html")
	if response.StatusCode != 404 {
		t.Errorf("Expected status 404 for empty filesystem, got %d", response.StatusCode)
	}

	// Test with non-existent file in non-empty filesystem
	response = ServeStaticFile(testassets.TestFS, config, "/nonexistent.html")
	if response.StatusCode != 404 {
		t.Errorf("Expected status 404 for non-existent file, got %d", response.StatusCode)
	}

	t.Log("✅ Empty assets and non-existent files handled correctly")
}

func TestServeStaticFile_ContentTypes(t *testing.T) {
	tests := []struct {
		name                string
		path                string
		expectedStatus      int
		expectedContentType string
	}{
		{"HTML file", "/index.html", 200, "text/html; charset=utf-8"},
		{"CSS file", "/styles.css", 200, "text/css; charset=utf-8"},
		{"JS file", "/app.js", 200, "application/javascript; charset=utf-8"},
		{"JSON file", "/data.json", 200, "application/json; charset=utf-8"},
		{"SVG image", "/images/icon.svg", 200, "image/svg+xml"},
		// These files do not exist in testassets, so expect 404,
		// but we can still check the content type that would be inferred.
		{"PNG image (non-existent)", "/image.png", 404, "image/png"},
		{"JPEG image (non-existent)", "/photo.jpg", 404, "image/jpeg"},
		{"ICO file (non-existent)", "/favicon.ico", 404, "image/x-icon"},
		{"WOFF font (non-existent)", "/font.woff", 404, "font/woff"},
		{"WOFF2 font (non-existent)", "/font.woff2", 404, "font/woff2"},
		{"TTF font (non-existent)", "/font.ttf", 404, "font/ttf"},
		{"Unknown type (non-existent)", "/unknown.xyz", 404, "application/octet-stream"},
	}

	config := StaticConfig{
		AssetsDir: "", // Default assets directory is 'assets' within the embed FS
		SPAMode:   false,
		DevMode:   false,
		APIPrefix: "/api/",
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := ServeStaticFile(testassets.TestFS, config, tt.path)
			if response.StatusCode != tt.expectedStatus {
				t.Errorf("Expected status %d for %s, got %d", tt.expectedStatus, tt.path, response.StatusCode)
			}
			// For 404s, the content type might not be explicitly set by the main serving logic
			// but getContentType would have been called internally if the file was found.
			// We are essentially testing getContentType indirectly for non-existent files.
			if response.StatusCode == 200 {
				if response.ContentType != tt.expectedContentType {
					t.Errorf("Expected content type %s for %s, got %s",
						tt.expectedContentType, tt.path, response.ContentType)
				}
			} else {
				// For 404s, we can still verify what getContentType *would* return
				// This makes the test more about getContentType's behavior for these extensions
				inferredContentType := getContentType(tt.path)
				if inferredContentType != tt.expectedContentType {
					t.Errorf("Expected inferred content type %s for %s (on 404), got %s",
						tt.expectedContentType, tt.path, inferredContentType)
				}
			}
		})
	}
}
