package openapi

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/danielgtaylor/huma/v2"
)

// GenerateSpecToFile generates an OpenAPI spec from a Huma API and saves it to a file
func GenerateSpecToFile(api huma.API, outputPath string) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// Generate OpenAPI JSON
	spec, err := api.OpenAPI().MarshalJSON()
	if err != nil {
		return fmt.Errorf("failed to generate OpenAPI JSON: %w", err)
	}

	// Save to file
	if err := os.WriteFile(outputPath, spec, 0644); err != nil {
		return fmt.Errorf("failed to save OpenAPI spec to %s: %w", outputPath, err)
	}

	return nil
}

// GenerateSpec generates an OpenAPI spec from a Huma API and returns it as bytes
func GenerateSpec(api huma.API) ([]byte, error) {
	return api.OpenAPI().MarshalJSON()
}

// GenerateSpecYAML generates an OpenAPI spec in YAML format
func GenerateSpecYAML(api huma.API) ([]byte, error) {
	return api.OpenAPI().YAML()
}

// GetRouteCount returns the number of routes in the API
func GetRouteCount(api huma.API) int {
	openAPISpec := api.OpenAPI()
	if openAPISpec == nil || openAPISpec.Paths == nil {
		return 0
	}

	routeCount := 0
	for path := range openAPISpec.Paths {
		if pathItem := openAPISpec.Paths[path]; pathItem != nil {
			if pathItem.Get != nil {
				routeCount++
			}
			if pathItem.Post != nil {
				routeCount++
			}
			if pathItem.Put != nil {
				routeCount++
			}
			if pathItem.Delete != nil {
				routeCount++
			}
			if pathItem.Patch != nil {
				routeCount++
			}
			if pathItem.Head != nil {
				routeCount++
			}
			if pathItem.Options != nil {
				routeCount++
			}
		}
	}
	return routeCount
}
