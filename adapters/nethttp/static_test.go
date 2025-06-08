package nethttp

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/barisgit/goflux"
	"github.com/barisgit/goflux/internal/testassets"
	"github.com/barisgit/goflux/internal/testutil"
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

func TestStaticHandlerFunc_BasicCreation(t *testing.T) {
	config := goflux.StaticConfig{
		AssetsDir: "",
		SPAMode:   false,
		DevMode:   false,
		APIPrefix: "/api/",
	}

	handlerFunc := StaticHandlerFunc(testassets.EmptyFS, config)
	if handlerFunc == nil {
		t.Fatal("Expected handler func to be created")
	}

	t.Log("✅ StaticHandlerFunc created successfully")
}

func TestStaticHandler_RealFileServing(t *testing.T) {
	config := goflux.StaticConfig{
		AssetsDir: "",
		SPAMode:   false,
		DevMode:   false,
		APIPrefix: "/api/",
	}

	mux := http.NewServeMux()
	handler := StaticHandler(testassets.TestFS, config)
	mux.Handle("/", handler)

	tests := testutil.GetBasicFileServingTests()

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			t.Logf("🧪 Testing %s", tt.Path)

			req := httptest.NewRequest("GET", tt.Path, nil)
			w := httptest.NewRecorder()

			mux.ServeHTTP(w, req)

			testutil.ValidateStaticResponse(t, tt, w.Code,
				w.Header().Get("Content-Type"),
				w.Header().Get("Cache-Control"),
				w.Body.String())
		})
	}
}

func TestStaticHandlerFunc_RealFileServing(t *testing.T) {
	config := goflux.StaticConfig{
		AssetsDir: "",
		SPAMode:   false,
		DevMode:   false,
		APIPrefix: "/api/",
	}

	mux := http.NewServeMux()
	handlerFunc := StaticHandlerFunc(testassets.TestFS, config)
	mux.HandleFunc("/", handlerFunc)

	tests := testutil.GetBasicFileServingTests()

	for _, tt := range tests {
		t.Run(tt.Name+"_func", func(t *testing.T) {
			t.Logf("🧪 Testing %s with HandlerFunc", tt.Path)

			req := httptest.NewRequest("GET", tt.Path, nil)
			w := httptest.NewRecorder()

			mux.ServeHTTP(w, req)

			testutil.ValidateStaticResponse(t, tt, w.Code,
				w.Header().Get("Content-Type"),
				w.Header().Get("Cache-Control"),
				w.Body.String())
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

	mux := http.NewServeMux()
	handler := StaticHandler(testassets.TestFS, config)
	mux.Handle("/", handler)

	req := httptest.NewRequest("GET", "/users/123/profile", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	t.Logf("🧪 Testing SPA fallback for /users/123/profile")

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 for SPA fallback, got %d", w.Code)
	} else {
		t.Logf("✅ SPA fallback status: %d", w.Code)
	}

	if !strings.Contains(w.Body.String(), "Test Static App") {
		t.Errorf("Expected SPA fallback to serve index.html content, got '%s'", w.Body.String())
	} else {
		t.Logf("✅ SPA fallback serves index.html")
	}

	contentType := w.Header().Get("Content-Type")
	if !strings.Contains(contentType, "text/html") {
		t.Errorf("Expected HTML content type for SPA fallback, got '%s'", contentType)
	} else {
		t.Logf("✅ SPA Content-Type: %s", contentType)
	}
}

func TestStaticHandlerFunc_SPAMode(t *testing.T) {
	config := goflux.StaticConfig{
		AssetsDir: "",
		SPAMode:   true,
		DevMode:   false,
		APIPrefix: "/api/",
	}

	mux := http.NewServeMux()
	handlerFunc := StaticHandlerFunc(testassets.TestFS, config)
	mux.HandleFunc("/", handlerFunc)

	req := httptest.NewRequest("GET", "/users/123/profile", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	t.Logf("🧪 Testing SPA fallback for /users/123/profile with HandlerFunc")

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 for SPA fallback, got %d", w.Code)
	} else {
		t.Logf("✅ SPA fallback status: %d", w.Code)
	}

	if !strings.Contains(w.Body.String(), "Test Static App") {
		t.Errorf("Expected SPA fallback to serve index.html content, got '%s'", w.Body.String())
	} else {
		t.Logf("✅ SPA fallback serves index.html")
	}
}

func TestStaticHandler_APIRouting(t *testing.T) {
	config := goflux.StaticConfig{
		AssetsDir: "",
		SPAMode:   false,
		DevMode:   false,
		APIPrefix: "/api/",
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/api/test", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message": "api works", "status": "ok"}`))
	})

	handler := StaticHandler(testassets.TestFS, config)
	mux.Handle("/", handler)

	t.Run("api_route_not_intercepted", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/test", nil)
		w := httptest.NewRecorder()

		mux.ServeHTTP(w, req)

		t.Logf("🧪 Testing API route /api/test")

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200 for API route, got %d", w.Code)
		} else {
			t.Logf("✅ API route status: %d", w.Code)
		}

		if !strings.Contains(w.Body.String(), "api works") {
			t.Errorf("Expected API response, got '%s'", w.Body.String())
		} else {
			t.Logf("✅ API response correct")
		}

		contentType := w.Header().Get("Content-Type")
		if !strings.Contains(contentType, "application/json") {
			t.Errorf("Expected JSON content type for API, got '%s'", contentType)
		} else {
			t.Logf("✅ API Content-Type: %s", contentType)
		}
	})

	t.Run("static_route_works", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/index.html", nil)
		w := httptest.NewRecorder()

		mux.ServeHTTP(w, req)

		t.Logf("🧪 Testing static route /index.html")

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200 for static route, got %d", w.Code)
		} else {
			t.Logf("✅ Static route status: %d", w.Code)
		}

		if !strings.Contains(w.Body.String(), "Test Static App") {
			t.Errorf("Expected static content, got '%s'", w.Body.String())
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

	mux := http.NewServeMux()
	handler := StaticHandler(testassets.TestFS, config)
	mux.Handle("/", handler)

	req := httptest.NewRequest("GET", "/index.html", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	t.Logf("🧪 Testing dev mode for /index.html")

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404 in dev mode, got %d", w.Code)
	} else {
		t.Logf("✅ Dev mode returns 404 as expected")
	}
}

func TestStaticHandlerFunc_DevMode(t *testing.T) {
	config := goflux.StaticConfig{
		AssetsDir: "",
		SPAMode:   false,
		DevMode:   true,
		APIPrefix: "/api/",
	}

	mux := http.NewServeMux()
	handlerFunc := StaticHandlerFunc(testassets.TestFS, config)
	mux.HandleFunc("/", handlerFunc)

	req := httptest.NewRequest("GET", "/index.html", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	t.Logf("🧪 Testing dev mode for /index.html with HandlerFunc")

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404 in dev mode, got %d", w.Code)
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

	mux := http.NewServeMux()
	handler := StaticHandler(testassets.TestFS, config)
	mux.Handle("/", handler)

	req := httptest.NewRequest("GET", "/nonexistent.txt", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	t.Logf("🧪 Testing 404 for /nonexistent.txt")

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404 for non-existent file, got %d", w.Code)
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

	mux := http.NewServeMux()
	handler := StaticHandler(testassets.TestFS, config)
	mux.Handle("/", handler)

	methods := testutil.GetHTTPMethods()

	for _, method := range methods {
		t.Run("method_"+method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/index.html", nil)
			w := httptest.NewRecorder()

			mux.ServeHTTP(w, req)

			t.Logf("🧪 Testing %s method", method)

			if w.Code == 0 {
				t.Errorf("Expected some status code for %s method, got 0", method)
			} else {
				t.Logf("✅ %s method status: %d", method, w.Code)
			}
		})
	}
}

func TestStaticHandlerFunc_HTTPMethods(t *testing.T) {
	config := goflux.StaticConfig{
		AssetsDir: "",
		SPAMode:   false,
		DevMode:   false,
		APIPrefix: "/api/",
	}

	mux := http.NewServeMux()
	handlerFunc := StaticHandlerFunc(testassets.TestFS, config)
	mux.HandleFunc("/", handlerFunc)

	methods := testutil.GetHTTPMethods()

	for _, method := range methods {
		t.Run("method_"+method+"_func", func(t *testing.T) {
			req := httptest.NewRequest(method, "/index.html", nil)
			w := httptest.NewRecorder()

			mux.ServeHTTP(w, req)

			t.Logf("🧪 Testing %s method with HandlerFunc", method)

			if w.Code == 0 {
				t.Errorf("Expected some status code for %s method, got 0", method)
			} else {
				t.Logf("✅ %s method status: %d", method, w.Code)
			}
		})
	}
}

func TestStaticHandler_ConfigVariations(t *testing.T) {
	configs := testutil.GetTestConfigs()

	for _, tc := range configs {
		t.Run(tc.Name, func(t *testing.T) {
			mux := http.NewServeMux()
			handler := StaticHandler(testassets.TestFS, tc.Config)
			mux.Handle("/", handler)

			req := httptest.NewRequest("GET", "/index.html", nil)
			w := httptest.NewRecorder()

			mux.ServeHTTP(w, req)

			t.Logf("🧪 Testing config: %s", tc.Name)

			if w.Code == 0 {
				t.Error("Expected some HTTP status code, got 0")
			} else {
				t.Logf("✅ Config %s works, status: %d", tc.Name, w.Code)
			}
		})
	}
}

func TestStaticHandlerFunc_ConfigVariations(t *testing.T) {
	configs := testutil.GetTestConfigs()

	for _, tc := range configs {
		t.Run(tc.Name+"_func", func(t *testing.T) {
			mux := http.NewServeMux()
			handlerFunc := StaticHandlerFunc(testassets.TestFS, tc.Config)
			mux.HandleFunc("/", handlerFunc)

			req := httptest.NewRequest("GET", "/index.html", nil)
			w := httptest.NewRecorder()

			mux.ServeHTTP(w, req)

			t.Logf("🧪 Testing config: %s with HandlerFunc", tc.Name)

			if w.Code == 0 {
				t.Error("Expected some HTTP status code, got 0")
			} else {
				t.Logf("✅ Config %s works, status: %d", tc.Name, w.Code)
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

	mux := http.NewServeMux()
	handler := StaticHandler(testassets.EmptyFS, config)
	mux.Handle("/", handler)

	req := httptest.NewRequest("GET", "/test.html", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	t.Logf("🧪 Testing empty filesystem")

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404 for empty filesystem, got %d", w.Code)
	} else {
		t.Logf("✅ Empty filesystem returns 404")
	}
}

func TestStaticHandlerFunc_EmptyFilesystem(t *testing.T) {
	config := goflux.StaticConfig{
		AssetsDir: "",
		SPAMode:   false,
		DevMode:   false,
		APIPrefix: "/api/",
	}

	mux := http.NewServeMux()
	handlerFunc := StaticHandlerFunc(testassets.EmptyFS, config)
	mux.HandleFunc("/", handlerFunc)

	req := httptest.NewRequest("GET", "/test.html", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	t.Logf("🧪 Testing empty filesystem with HandlerFunc")

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404 for empty filesystem, got %d", w.Code)
	} else {
		t.Logf("✅ Empty filesystem returns 404")
	}
}
