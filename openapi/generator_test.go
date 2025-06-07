package openapi

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humago"
)

// Test input/output types
type EmptyInput struct{}
type EmptyOutput struct{}

// createTestAPI creates a test API for OpenAPI testing
func createTestAPI() huma.API {
	mux := http.NewServeMux()
	config := huma.DefaultConfig("Test API", "1.0.0")
	return humago.New(mux, config)
}

func TestGenerateSpec(t *testing.T) {
	api := createTestAPI()

	// Add a simple endpoint to generate spec from
	huma.Register(api, huma.Operation{
		OperationID: "test-operation",
		Method:      http.MethodGet,
		Path:        "/test",
		Summary:     "Test endpoint",
	}, func(ctx context.Context, input *EmptyInput) (*EmptyOutput, error) {
		return &EmptyOutput{}, nil
	})

	spec, err := GenerateSpec(api)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(spec) == 0 {
		t.Error("Expected non-empty spec")
	}

	// Check if it contains basic OpenAPI structure
	specStr := string(spec)
	if !contains(specStr, "openapi") {
		t.Error("Expected spec to contain 'openapi' field")
	}

	if !contains(specStr, "Test API") {
		t.Error("Expected spec to contain API title")
	}
}

func TestGenerateSpecYAML(t *testing.T) {
	api := createTestAPI()

	// Add a simple endpoint
	huma.Register(api, huma.Operation{
		OperationID: "yaml-test",
		Method:      http.MethodPost,
		Path:        "/yaml-test",
		Summary:     "YAML test endpoint",
	}, func(ctx context.Context, input *EmptyInput) (*EmptyOutput, error) {
		return &EmptyOutput{}, nil
	})

	spec, err := GenerateSpecYAML(api)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(spec) == 0 {
		t.Error("Expected non-empty YAML spec")
	}

	// Check if it's YAML format (different from JSON)
	specStr := string(spec)
	if !contains(specStr, "openapi:") {
		t.Error("Expected YAML spec to contain 'openapi:' field")
	}
}

func TestGenerateSpecToFile(t *testing.T) {
	api := createTestAPI()

	// Add a test endpoint
	huma.Register(api, huma.Operation{
		OperationID: "file-test",
		Method:      http.MethodPut,
		Path:        "/file-test",
		Summary:     "File test endpoint",
	}, func(ctx context.Context, input *EmptyInput) (*EmptyOutput, error) {
		return &EmptyOutput{}, nil
	})

	// Create a temporary file path
	tempDir := t.TempDir()
	outputPath := filepath.Join(tempDir, "test-spec.json")

	err := GenerateSpecToFile(api, outputPath)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Check if file exists
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Error("Expected output file to exist")
	}

	// Read and verify file content
	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	if len(content) == 0 {
		t.Error("Expected non-empty file content")
	}

	contentStr := string(content)
	if !contains(contentStr, "file-test") {
		t.Error("Expected file content to contain operation ID")
	}
}

func TestGenerateSpecToFileCreatesDirectory(t *testing.T) {
	api := createTestAPI()

	tempDir := t.TempDir()
	outputPath := filepath.Join(tempDir, "nested", "dir", "spec.json")

	err := GenerateSpecToFile(api, outputPath)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Check if nested directory was created
	if _, err := os.Stat(filepath.Dir(outputPath)); os.IsNotExist(err) {
		t.Error("Expected nested directory to be created")
	}

	// Check if file exists
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Error("Expected output file to exist")
	}
}

func TestGetRouteCount(t *testing.T) {
	api := createTestAPI()

	// Initially should have 0 routes
	count := GetRouteCount(api)
	if count != 0 {
		t.Errorf("Expected 0 routes initially, got %d", count)
	}

	// Add some endpoints
	huma.Register(api, huma.Operation{
		OperationID: "get-users",
		Method:      http.MethodGet,
		Path:        "/users",
	}, func(ctx context.Context, input *EmptyInput) (*EmptyOutput, error) {
		return &EmptyOutput{}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "create-user",
		Method:      http.MethodPost,
		Path:        "/users",
	}, func(ctx context.Context, input *EmptyInput) (*EmptyOutput, error) {
		return &EmptyOutput{}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "get-user",
		Method:      http.MethodGet,
		Path:        "/users/{id}",
	}, func(ctx context.Context, input *EmptyInput) (*EmptyOutput, error) {
		return &EmptyOutput{}, nil
	})

	count = GetRouteCount(api)
	if count != 3 {
		t.Errorf("Expected 3 routes, got %d", count)
	}
}

func TestGetRouteCountWithAllHTTPMethods(t *testing.T) {
	api := createTestAPI()

	handler := func(ctx context.Context, input *EmptyInput) (*EmptyOutput, error) {
		return &EmptyOutput{}, nil
	}

	// Add endpoints for all HTTP methods on the same path
	huma.Register(api, huma.Operation{Method: http.MethodGet, Path: "/test"}, handler)
	huma.Register(api, huma.Operation{Method: http.MethodPost, Path: "/test"}, handler)
	huma.Register(api, huma.Operation{Method: http.MethodPut, Path: "/test"}, handler)
	huma.Register(api, huma.Operation{Method: http.MethodDelete, Path: "/test"}, handler)
	huma.Register(api, huma.Operation{Method: http.MethodPatch, Path: "/test"}, handler)
	huma.Register(api, huma.Operation{Method: http.MethodHead, Path: "/test"}, handler)
	huma.Register(api, huma.Operation{Method: http.MethodOptions, Path: "/test"}, handler)

	count := GetRouteCount(api)
	if count != 7 {
		t.Errorf("Expected 7 routes (all HTTP methods), got %d", count)
	}
}

func TestGetRouteCountWithEmptyAPI(t *testing.T) {
	api := createTestAPI()

	count := GetRouteCount(api)
	if count != 0 {
		t.Errorf("Expected 0 routes for empty API, got %d", count)
	}
}

func TestGenerateSpecError(t *testing.T) {
	// This test would need a way to force an error in spec generation
	// For now, we'll test with a valid API and expect no error
	api := createTestAPI()

	_, err := GenerateSpec(api)
	if err != nil {
		t.Errorf("Expected no error for valid API, got %v", err)
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > len(substr) && (s[:len(substr)] == substr ||
			s[len(s)-len(substr):] == substr ||
			findSubstring(s, substr))))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
