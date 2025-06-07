package testutil

import (
	"strings"
	"testing"

	"github.com/barisgit/goflux/internal/static"
)

// StaticTestCase represents a test case for static file serving
type StaticTestCase struct {
	Name                string
	Path                string
	ExpectedStatus      int
	ExpectedBodyContent string
	ExpectedContentType string
	ExpectCacheControl  bool
}

// GetBasicFileServingTests returns common test cases for file serving
func GetBasicFileServingTests() []StaticTestCase {
	return []StaticTestCase{
		{
			Name:                "serve index.html",
			Path:                "/index.html",
			ExpectedStatus:      200,
			ExpectedBodyContent: "Test Static App",
			ExpectedContentType: "text/html",
			ExpectCacheControl:  true,
		},
		{
			Name:                "serve root path as index.html",
			Path:                "/",
			ExpectedStatus:      200,
			ExpectedBodyContent: "Test Static App",
			ExpectedContentType: "text/html",
			ExpectCacheControl:  true,
		},
		{
			Name:                "serve JavaScript file",
			Path:                "/app.js",
			ExpectedStatus:      200,
			ExpectedBodyContent: "Hello from test app.js",
			ExpectedContentType: "application/javascript",
			ExpectCacheControl:  true,
		},
		{
			Name:                "serve CSS file",
			Path:                "/styles.css",
			ExpectedStatus:      200,
			ExpectedBodyContent: "font-family: Arial",
			ExpectedContentType: "text/css",
			ExpectCacheControl:  true,
		},
		{
			Name:                "serve JSON file",
			Path:                "/data.json",
			ExpectedStatus:      200,
			ExpectedBodyContent: "Test Data",
			ExpectedContentType: "application/json",
			ExpectCacheControl:  true,
		},
		{
			Name:                "serve SVG file",
			Path:                "/images/icon.svg",
			ExpectedStatus:      200,
			ExpectedBodyContent: "<svg xmlns",
			ExpectedContentType: "image/svg+xml",
			ExpectCacheControl:  true,
		},
	}
}

// GetTestConfigs returns common test configurations
func GetTestConfigs() []struct {
	Name   string
	Config static.StaticConfig
} {
	return []struct {
		Name   string
		Config static.StaticConfig
	}{
		{
			Name: "basic_config",
			Config: static.StaticConfig{
				AssetsDir: "",
				SPAMode:   false,
				DevMode:   false,
				APIPrefix: "/api/",
			},
		},
		{
			Name: "spa_mode",
			Config: static.StaticConfig{
				AssetsDir: "",
				SPAMode:   true,
				DevMode:   false,
				APIPrefix: "/api/",
			},
		},
		{
			Name: "dev_mode",
			Config: static.StaticConfig{
				AssetsDir: "",
				SPAMode:   false,
				DevMode:   true,
				APIPrefix: "/api/",
			},
		},
		{
			Name: "custom_api_prefix",
			Config: static.StaticConfig{
				AssetsDir: "",
				SPAMode:   false,
				DevMode:   false,
				APIPrefix: "/api/v1/",
			},
		},
	}
}

// ValidateStaticResponse validates common aspects of static file responses
func ValidateStaticResponse(t *testing.T, testCase StaticTestCase, statusCode int, contentType, cacheControl, body string) {
	t.Helper()

	// Test status code
	if statusCode != testCase.ExpectedStatus {
		t.Errorf("Expected status %d, got %d", testCase.ExpectedStatus, statusCode)
	} else {
		t.Logf("✅ Status code: %d", statusCode)
	}

	// Test Content-Type header
	if !strings.Contains(contentType, testCase.ExpectedContentType) {
		t.Errorf("Expected Content-Type to contain '%s', got '%s'", testCase.ExpectedContentType, contentType)
	} else {
		t.Logf("✅ Content-Type: %s", contentType)
	}

	// Test Cache-Control header
	if testCase.ExpectCacheControl {
		if cacheControl == "" {
			t.Errorf("Expected Cache-Control header to be set, got empty")
		} else {
			t.Logf("✅ Cache-Control: %s", cacheControl)
		}
	}

	// Test response body
	if !strings.Contains(body, testCase.ExpectedBodyContent) {
		t.Errorf("Expected body to contain '%s', got '%s'", testCase.ExpectedBodyContent, body)
	} else {
		t.Logf("✅ Body contains expected content")
	}

	// Verify body is not empty
	if len(body) == 0 {
		t.Errorf("Expected non-empty body, got empty body")
	} else {
		t.Logf("✅ Body length: %d bytes", len(body))
	}
}

// GetHTTPMethods returns common HTTP methods for testing
func GetHTTPMethods() []string {
	return []string{"GET", "POST", "PUT", "DELETE", "HEAD", "OPTIONS"}
}
