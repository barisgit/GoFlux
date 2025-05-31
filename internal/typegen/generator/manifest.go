package generator

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/barisgit/goflux/internal/typegen/types"
)

// GenerateRouteManifest generates a JSON manifest of all routes
func GenerateRouteManifest(routes []types.APIRoute) error {
	manifestDir := filepath.Join("internal", "static")
	if err := os.MkdirAll(manifestDir, 0755); err != nil {
		return fmt.Errorf("failed to create manifest directory: %w", err)
	}

	manifestFile := filepath.Join(manifestDir, "routes.json")

	data, err := json.MarshalIndent(routes, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal routes: %w", err)
	}

	if err := os.WriteFile(manifestFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write manifest file: %w", err)
	}

	return nil
}
