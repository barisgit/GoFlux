package echo

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/barisgit/goflux"
	"github.com/barisgit/goflux/internal/testassets"
	"github.com/barisgit/goflux/internal/testutil"
	"github.com/labstack/echo/v4"
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

	e := echo.New()
	handler := StaticHandler(testassets.TestFS, config)
	e.Any("/*", handler)

	tests := testutil.GetBasicFileServingTests()

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			t.Logf("🧪 Testing %s", tt.Path)

			req := httptest.NewRequest("GET", tt.Path, nil)
			rec := httptest.NewRecorder()

			e.ServeHTTP(rec, req)

			testutil.ValidateStaticResponse(t, tt, rec.Code,
				rec.Header().Get("Content-Type"),
				rec.Header().Get("Cache-Control"),
				rec.Body.String())
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

	e := echo.New()
	handler := StaticHandler(testassets.TestFS, config)
	e.Any("/*", handler)

	req := httptest.NewRequest("GET", "/users/123/profile", nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	t.Logf("🧪 Testing SPA fallback for /users/123/profile")

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200 for SPA fallback, got %d", rec.Code)
	} else {
		t.Logf("✅ SPA fallback status: %d", rec.Code)
	}

	if !strings.Contains(rec.Body.String(), "Test Static App") {
		t.Errorf("Expected SPA fallback to serve index.html content, got '%s'", rec.Body.String())
	} else {
		t.Logf("✅ SPA fallback serves index.html")
	}

	contentType := rec.Header().Get("Content-Type")
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

	e := echo.New()

	e.GET("/api/test", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"message": "api works", "status": "ok"})
	})

	handler := StaticHandler(testassets.TestFS, config)
	e.Any("/*", handler)

	t.Run("api_route_not_intercepted", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/test", nil)
		rec := httptest.NewRecorder()

		e.ServeHTTP(rec, req)

		t.Logf("🧪 Testing API route /api/test")

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200 for API route, got %d", rec.Code)
		} else {
			t.Logf("✅ API route status: %d", rec.Code)
		}

		if !strings.Contains(rec.Body.String(), "api works") {
			t.Errorf("Expected API response, got '%s'", rec.Body.String())
		} else {
			t.Logf("✅ API response correct")
		}

		contentType := rec.Header().Get("Content-Type")
		if !strings.Contains(contentType, "application/json") {
			t.Errorf("Expected JSON content type for API, got '%s'", contentType)
		} else {
			t.Logf("✅ API Content-Type: %s", contentType)
		}
	})

	t.Run("static_route_works", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/index.html", nil)
		rec := httptest.NewRecorder()

		e.ServeHTTP(rec, req)

		t.Logf("🧪 Testing static route /index.html")

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200 for static route, got %d", rec.Code)
		} else {
			t.Logf("✅ Static route status: %d", rec.Code)
		}

		if !strings.Contains(rec.Body.String(), "Test Static App") {
			t.Errorf("Expected static content, got '%s'", rec.Body.String())
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

	e := echo.New()
	handler := StaticHandler(testassets.TestFS, config)
	e.Any("/*", handler)

	req := httptest.NewRequest("GET", "/index.html", nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	t.Logf("🧪 Testing dev mode for /index.html")

	if rec.Code != http.StatusNotFound {
		t.Errorf("Expected status 404 in dev mode, got %d", rec.Code)
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

	e := echo.New()
	handler := StaticHandler(testassets.TestFS, config)
	e.Any("/*", handler)

	req := httptest.NewRequest("GET", "/nonexistent.txt", nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	t.Logf("🧪 Testing 404 for /nonexistent.txt")

	if rec.Code != http.StatusNotFound {
		t.Errorf("Expected status 404 for non-existent file, got %d", rec.Code)
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

	e := echo.New()
	handler := StaticHandler(testassets.TestFS, config)
	e.Any("/*", handler)

	methods := testutil.GetHTTPMethods()

	for _, method := range methods {
		t.Run("method_"+method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/index.html", nil)
			rec := httptest.NewRecorder()

			e.ServeHTTP(rec, req)

			t.Logf("🧪 Testing %s method", method)

			if rec.Code == 0 {
				t.Errorf("Expected some status code for %s method, got 0", method)
			} else {
				t.Logf("✅ %s method status: %d", method, rec.Code)
			}
		})
	}
}

func TestStaticHandler_ConfigVariations(t *testing.T) {
	configs := testutil.GetTestConfigs()

	for _, tc := range configs {
		t.Run(tc.Name, func(t *testing.T) {
			e := echo.New()
			handler := StaticHandler(testassets.TestFS, tc.Config)
			e.Any("/*", handler)

			req := httptest.NewRequest("GET", "/index.html", nil)
			rec := httptest.NewRecorder()

			e.ServeHTTP(rec, req)

			t.Logf("🧪 Testing config: %s", tc.Name)

			if rec.Code == 0 {
				t.Error("Expected some HTTP status code, got 0")
			} else {
				t.Logf("✅ Config %s works, status: %d", tc.Name, rec.Code)
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

	e := echo.New()
	handler := StaticHandler(testassets.EmptyFS, config)
	e.Any("/*", handler)

	req := httptest.NewRequest("GET", "/test.html", nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	t.Logf("🧪 Testing empty filesystem")

	if rec.Code != http.StatusNotFound {
		t.Errorf("Expected status 404 for empty filesystem, got %d", rec.Code)
	} else {
		t.Logf("✅ Empty filesystem returns 404")
	}
}
