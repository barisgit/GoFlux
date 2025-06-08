package features

import (
	"context"
	"net/http"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humago"
)

func createTestAPI() huma.API {
	mux := http.NewServeMux()
	config := huma.DefaultConfig("Test API", "1.0.0")
	return humago.New(mux, config)
}

func TestGreet(t *testing.T) {
	api := createTestAPI()

	// Test with default options
	options := GreetOptions{
		ServiceName: "Test Service",
		Version:     "1.0.0",
		Host:        "localhost",
		Port:        8080,
		DevMode:     false,
	}

	// This function outputs to stdout, so we can't easily test the output
	// but we can ensure it doesn't panic
	Greet(api, options)
}

func TestGreetWithCustomOptions(t *testing.T) {
	api := createTestAPI()

	// Test with custom options
	options := GreetOptions{
		ServiceName: "Custom Service",
		Version:     "2.0.0",
		Host:        "0.0.0.0",
		Port:        3000,
		ProxyPort:   8080,
		DevMode:     true,
		DocsPath:    "/api/docs",
		OpenAPIPath: "/api/openapi",
	}

	// This function outputs to stdout, so we can't easily test the output
	// but we can ensure it doesn't panic
	Greet(api, options)
}

func TestGreetWithMinimalOptions(t *testing.T) {
	api := createTestAPI()

	// Test with minimal options
	options := GreetOptions{}

	// Should not panic even with empty options
	Greet(api, options)
}

func TestQuickGreet(t *testing.T) {
	// Test QuickGreet with all parameters
	QuickGreet("Test Service", "1.0.0", "localhost", 8080)

	// Test with empty values
	QuickGreet("", "", "", 0)

	// Test with partial values
	QuickGreet("Service Only", "", "", 0)
	QuickGreet("", "1.0.0", "localhost", 8080)
}

func TestGreetOptionsDefaults(t *testing.T) {
	// Test that GreetOptions has sensible zero values
	var options GreetOptions

	if options.ServiceName != "" {
		t.Errorf("Expected empty service name by default, got '%s'", options.ServiceName)
	}

	if options.Version != "" {
		t.Errorf("Expected empty version by default, got '%s'", options.Version)
	}

	if options.Host != "" {
		t.Errorf("Expected empty host by default, got '%s'", options.Host)
	}

	if options.Port != 0 {
		t.Errorf("Expected zero port by default, got %d", options.Port)
	}

	if options.ProxyPort != 0 {
		t.Errorf("Expected zero proxy port by default, got %d", options.ProxyPort)
	}

	if options.DevMode {
		t.Error("Expected dev mode to be false by default")
	}

	if options.DocsPath != "" {
		t.Errorf("Expected empty docs path by default, got '%s'", options.DocsPath)
	}

	if options.OpenAPIPath != "" {
		t.Errorf("Expected empty OpenAPI path by default, got '%s'", options.OpenAPIPath)
	}
}

func TestGreetWithDevelopmentMode(t *testing.T) {
	api := createTestAPI()

	options := GreetOptions{
		ServiceName: "Dev Service",
		Version:     "1.0.0-dev",
		Host:        "localhost",
		Port:        3000,
		ProxyPort:   8080,
		DevMode:     true,
		DocsPath:    "/docs",
		OpenAPIPath: "/openapi",
	}

	// Should not panic in dev mode
	Greet(api, options)
}

func TestGreetWithProductionMode(t *testing.T) {
	api := createTestAPI()

	options := GreetOptions{
		ServiceName: "Prod Service",
		Version:     "1.0.0",
		Host:        "0.0.0.0",
		Port:        80,
		DevMode:     false,
		DocsPath:    "/api/docs",
		OpenAPIPath: "/api/openapi",
	}

	// Should not panic in production mode
	Greet(api, options)
}

func TestAddHealthCheck(t *testing.T) {
	api := createTestAPI()

	// Test with custom path
	AddHealthCheck(api, "/health", "Test Service", "1.0.0")

	// Verify the health endpoint was registered
	spec := api.OpenAPI()
	if spec.Paths == nil {
		t.Fatal("Expected paths to be defined in OpenAPI spec")
	}

	pathItem := spec.Paths["/health"]
	if pathItem == nil {
		t.Fatal("Expected /health path to be registered")
	}

	if pathItem.Get == nil {
		t.Error("Expected GET operation to be registered")
	}

	if pathItem.Get.OperationID != "health-check" {
		t.Errorf("Expected operation ID 'health-check', got '%s'", pathItem.Get.OperationID)
	}

	if pathItem.Get.Summary != "Health Check" {
		t.Errorf("Expected summary 'Health Check', got '%s'", pathItem.Get.Summary)
	}
}

func TestAddHealthCheckWithDefaultPath(t *testing.T) {
	api := createTestAPI()

	// Test with empty path (should use default)
	AddHealthCheck(api, "", "Test Service", "1.0.0")

	// Verify the default health endpoint was registered
	spec := api.OpenAPI()
	pathItem := spec.Paths["/api/health"]
	if pathItem == nil {
		t.Fatal("Expected /api/health path to be registered as default")
	}

	if pathItem.Get == nil {
		t.Error("Expected GET operation to be registered")
	}
}

func TestCustomHealthCheck(t *testing.T) {
	api := createTestAPI()

	// Custom health function
	healthFunc := func(ctx context.Context) (*HealthResponse, error) {
		resp := &HealthResponse{}
		resp.Body.Status = "custom"
		resp.Body.Message = "Custom health check"
		return resp, nil
	}

	CustomHealthCheck(api, "/custom-health", healthFunc)

	// Verify the custom health endpoint was registered
	spec := api.OpenAPI()
	pathItem := spec.Paths["/custom-health"]
	if pathItem == nil {
		t.Fatal("Expected /custom-health path to be registered")
	}

	if pathItem.Get == nil {
		t.Error("Expected GET operation to be registered")
	}

	if pathItem.Get.OperationID != "health-check" {
		t.Errorf("Expected operation ID 'health-check', got '%s'", pathItem.Get.OperationID)
	}
}

func TestCustomHealthCheckWithDefaultPath(t *testing.T) {
	api := createTestAPI()

	healthFunc := func(ctx context.Context) (*HealthResponse, error) {
		resp := &HealthResponse{}
		resp.Body.Status = "ok"
		return resp, nil
	}

	// Test with empty path (should use default)
	CustomHealthCheck(api, "", healthFunc)

	// Verify the default health endpoint was registered
	spec := api.OpenAPI()
	pathItem := spec.Paths["/api/health"]
	if pathItem == nil {
		t.Fatal("Expected /api/health path to be registered as default")
	}
}

func TestHealthResponse(t *testing.T) {
	// Test that HealthResponse can be created and used
	resp := &HealthResponse{}
	resp.Body.Status = "ok"
	resp.Body.Message = "Service is healthy"
	resp.Body.Version = "1.0.0"

	if resp.Body.Status != "ok" {
		t.Errorf("Expected status 'ok', got '%s'", resp.Body.Status)
	}

	if resp.Body.Message != "Service is healthy" {
		t.Errorf("Expected message 'Service is healthy', got '%s'", resp.Body.Message)
	}

	if resp.Body.Version != "1.0.0" {
		t.Errorf("Expected version '1.0.0', got '%s'", resp.Body.Version)
	}
}
