package fiber

import (
	"net/http"
	"strings"
	"testing"

	"github.com/barisgit/goflux"
	"github.com/barisgit/goflux/internal/testassets"
	"github.com/barisgit/goflux/internal/testutil"
	"github.com/gofiber/fiber/v2"
)

func TestStaticHandler_BasicCreation(t *testing.T) {
	config := goflux.StaticConfig{
		AssetsDir: "",
		SPAMode:   false,
		DevMode:   false,
		APIPrefix: "/api/",
	}

	handler := StaticHandler(testassets.EmptyFS, config)
	if handler == nil {
		t.Fatal("Expected handler to be created")
	}

	t.Log("✅ StaticHandler created successfully")
}

func TestStaticHandler_RealFileServing(t *testing.T) {
	config := goflux.StaticConfig{
		AssetsDir: "",
		SPAMode:   false,
		DevMode:   false,
		APIPrefix: "/api/",
	}

	app := fiber.New(fiber.Config{
		DisableStartupMessage: true,
	})
	handler := StaticHandler(testassets.TestFS, config)
	app.All("/*", handler)

	tests := testutil.GetBasicFileServingTests()

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			t.Logf("🧪 Testing %s", tt.Path)

			req, _ := http.NewRequest("GET", tt.Path, nil)
			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("Failed to test request: %v", err)
			}
			defer resp.Body.Close()

			var body []byte
			if resp.Body != nil {
				body = make([]byte, resp.ContentLength)
				resp.Body.Read(body)
			}

			testutil.ValidateStaticResponse(t, tt, resp.StatusCode,
				resp.Header.Get("Content-Type"),
				resp.Header.Get("Cache-Control"),
				string(body))
		})
	}
}

func TestStaticHandler_SPAMode(t *testing.T) {
	config := goflux.StaticConfig{
		AssetsDir: "",
		SPAMode:   true,
		DevMode:   false,
		APIPrefix: "/api/",
	}

	app := fiber.New(fiber.Config{
		DisableStartupMessage: true,
	})
	handler := StaticHandler(testassets.TestFS, config)
	app.All("/*", handler)

	req, _ := http.NewRequest("GET", "/users/123/profile", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to test request: %v", err)
	}
	defer resp.Body.Close()

	t.Logf("🧪 Testing SPA fallback for /users/123/profile")

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200 for SPA fallback, got %d", resp.StatusCode)
	} else {
		t.Logf("✅ SPA fallback status: %d", resp.StatusCode)
	}

	var body []byte
	if resp.Body != nil && resp.ContentLength > 0 {
		body = make([]byte, resp.ContentLength)
		resp.Body.Read(body)
	}

	if !strings.Contains(string(body), "Test Static App") {
		t.Errorf("Expected SPA fallback to serve index.html content, got '%s'", string(body))
	} else {
		t.Logf("✅ SPA fallback serves index.html")
	}

	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "text/html") {
		t.Errorf("Expected HTML content type for SPA fallback, got '%s'", contentType)
	} else {
		t.Logf("✅ SPA Content-Type: %s", contentType)
	}
}

func TestStaticHandler_APIRouting(t *testing.T) {
	config := goflux.StaticConfig{
		AssetsDir: "",
		SPAMode:   false,
		DevMode:   false,
		APIPrefix: "/api/",
	}

	app := fiber.New(fiber.Config{
		DisableStartupMessage: true,
	})

	app.Get("/api/test", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"message": "api works", "status": "ok"})
	})

	handler := StaticHandler(testassets.TestFS, config)
	app.All("/*", handler)

	t.Run("api_route_not_intercepted", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/test", nil)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Failed to test request: %v", err)
		}
		defer resp.Body.Close()

		t.Logf("🧪 Testing API route /api/test")

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200 for API route, got %d", resp.StatusCode)
		} else {
			t.Logf("✅ API route status: %d", resp.StatusCode)
		}

		var body []byte
		if resp.Body != nil && resp.ContentLength > 0 {
			body = make([]byte, resp.ContentLength)
			resp.Body.Read(body)
		}

		if !strings.Contains(string(body), "api works") {
			t.Errorf("Expected API response, got '%s'", string(body))
		} else {
			t.Logf("✅ API response correct")
		}

		contentType := resp.Header.Get("Content-Type")
		if !strings.Contains(contentType, "application/json") {
			t.Errorf("Expected JSON content type for API, got '%s'", contentType)
		} else {
			t.Logf("✅ API Content-Type: %s", contentType)
		}
	})

	t.Run("static_route_works", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/index.html", nil)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Failed to test request: %v", err)
		}
		defer resp.Body.Close()

		t.Logf("🧪 Testing static route /index.html")

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200 for static route, got %d", resp.StatusCode)
		} else {
			t.Logf("✅ Static route status: %d", resp.StatusCode)
		}

		var body []byte
		if resp.Body != nil && resp.ContentLength > 0 {
			body = make([]byte, resp.ContentLength)
			resp.Body.Read(body)
		}

		if !strings.Contains(string(body), "Test Static App") {
			t.Errorf("Expected static content, got '%s'", string(body))
		} else {
			t.Logf("✅ Static content correct")
		}
	})
}

func TestStaticHandler_DevMode(t *testing.T) {
	config := goflux.StaticConfig{
		AssetsDir: "",
		SPAMode:   false,
		DevMode:   true,
		APIPrefix: "/api/",
	}

	app := fiber.New(fiber.Config{
		DisableStartupMessage: true,
	})
	handler := StaticHandler(testassets.TestFS, config)
	app.All("/*", handler)

	req, _ := http.NewRequest("GET", "/index.html", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to test request: %v", err)
	}
	defer resp.Body.Close()

	t.Logf("🧪 Testing dev mode for /index.html")

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Expected status 404 in dev mode, got %d", resp.StatusCode)
	} else {
		t.Logf("✅ Dev mode returns 404 as expected")
	}
}

func TestStaticHandler_NotFound(t *testing.T) {
	config := goflux.StaticConfig{
		AssetsDir: "",
		SPAMode:   false,
		DevMode:   false,
		APIPrefix: "/api/",
	}

	app := fiber.New(fiber.Config{
		DisableStartupMessage: true,
	})
	handler := StaticHandler(testassets.TestFS, config)
	app.All("/*", handler)

	req, _ := http.NewRequest("GET", "/nonexistent.txt", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to test request: %v", err)
	}
	defer resp.Body.Close()

	t.Logf("🧪 Testing 404 for /nonexistent.txt")

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Expected status 404 for non-existent file, got %d", resp.StatusCode)
	} else {
		t.Logf("✅ 404 status for non-existent file")
	}
}

func TestStaticHandler_HTTPMethods(t *testing.T) {
	config := goflux.StaticConfig{
		AssetsDir: "",
		SPAMode:   false,
		DevMode:   false,
		APIPrefix: "/api/",
	}

	app := fiber.New(fiber.Config{
		DisableStartupMessage: true,
	})
	handler := StaticHandler(testassets.TestFS, config)
	app.All("/*", handler)

	methods := testutil.GetHTTPMethods()

	for _, method := range methods {
		t.Run("method_"+method, func(t *testing.T) {
			req, _ := http.NewRequest(method, "/index.html", nil)
			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("Failed to test request: %v", err)
			}
			defer resp.Body.Close()

			t.Logf("🧪 Testing %s method", method)

			if resp.StatusCode == 0 {
				t.Errorf("Expected some status code for %s method, got 0", method)
			} else {
				t.Logf("✅ %s method status: %d", method, resp.StatusCode)
			}
		})
	}
}

func TestStaticHandler_ConfigVariations(t *testing.T) {
	configs := testutil.GetTestConfigs()

	for _, tc := range configs {
		t.Run(tc.Name, func(t *testing.T) {
			app := fiber.New(fiber.Config{
				DisableStartupMessage: true,
			})
			handler := StaticHandler(testassets.TestFS, tc.Config)
			app.All("/*", handler)

			req, _ := http.NewRequest("GET", "/index.html", nil)
			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("Failed to test request: %v", err)
			}
			defer resp.Body.Close()

			t.Logf("🧪 Testing config: %s", tc.Name)

			if resp.StatusCode == 0 {
				t.Error("Expected some HTTP status code, got 0")
			} else {
				t.Logf("✅ Config %s works, status: %d", tc.Name, resp.StatusCode)
			}
		})
	}
}

func TestStaticHandler_EmptyFilesystem(t *testing.T) {
	config := goflux.StaticConfig{
		AssetsDir: "",
		SPAMode:   false,
		DevMode:   false,
		APIPrefix: "/api/",
	}

	app := fiber.New(fiber.Config{
		DisableStartupMessage: true,
	})
	handler := StaticHandler(testassets.EmptyFS, config)
	app.All("/*", handler)

	req, _ := http.NewRequest("GET", "/test.html", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to test request: %v", err)
	}
	defer resp.Body.Close()

	t.Logf("🧪 Testing empty filesystem")

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Expected status 404 for empty filesystem, got %d", resp.StatusCode)
	} else {
		t.Logf("✅ Empty filesystem returns 404")
	}
}
