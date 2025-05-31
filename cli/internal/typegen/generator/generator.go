package generator

import (
	"fmt"
	"os"

	"github.com/barisgit/goflux/config"
	"github.com/barisgit/goflux/cli/internal/typegen/types"
)

// Generate generates all API client artifacts based on configuration
// This is the main entry point for the type generation system
func Generate(routes []types.APIRoute, typeDefs []types.TypeDefinition, config *config.APIClientConfig) error {
	// Ensure output directories exist
	if err := ensureOutputDirectories(); err != nil {
		return fmt.Errorf("failed to create output directories: %w", err)
	}

	// Generate TypeScript types if needed
	if ShouldGenerateTypeScriptTypes(config.Generator) {
		if err := GenerateTypeScriptTypes(typeDefs); err != nil {
			return fmt.Errorf("failed to generate TypeScript types: %w", err)
		}
	}

	// Generate API client
	if err := GenerateAPIClient(routes, typeDefs, config); err != nil {
		return fmt.Errorf("failed to generate API client: %w", err)
	}

	// Generate route manifest
	if err := GenerateRouteManifest(routes); err != nil {
		return fmt.Errorf("failed to generate route manifest: %w", err)
	}

	return nil
}

// ensureOutputDirectories creates necessary output directories
func ensureOutputDirectories() error {
	directories := []string{
		"frontend/src/types",
		"frontend/src/lib",
		"internal/static",
	}

	for _, dir := range directories {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	return nil
}

// GetSupportedGenerators returns a list of supported generator types
func GetSupportedGenerators() []string {
	return []string{
		"basic",     // Basic JavaScript client
		"basic-ts",  // Basic TypeScript client
		"axios",     // Axios-based TypeScript client
		"trpc-like", // tRPC-like client with React Query integration
	}
}

// ValidateGeneratorType validates if the generator type is supported
func ValidateGeneratorType(generatorType string) error {
	supported := GetSupportedGenerators()
	for _, gen := range supported {
		if gen == generatorType {
			return nil
		}
	}
	return fmt.Errorf("unsupported generator type '%s', supported types: %v", generatorType, supported)
}
