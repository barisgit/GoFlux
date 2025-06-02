package generator

import (
	"fmt"
	"path/filepath"

	"github.com/barisgit/goflux/cli/internal/typegen/types"
	"github.com/barisgit/goflux/config"
)

// detectAuthRequirements analyzes routes to determine if authentication is needed
func detectAuthRequirements(routes []types.APIRoute) (bool, string) {
	for _, route := range routes {
		if route.RequiresAuth {
			return true, route.AuthType
		}
	}
	return false, ""
}

// GenerateAPIClient generates the API client based on configuration
func GenerateAPIClient(routes []types.APIRoute, typeDefs []types.TypeDefinition, config *config.APIClientConfig) error {
	libDir := filepath.Join("frontend", "src", "lib")

	// Use configured output file name or default
	outputFile := config.OutputFile
	if outputFile == "" {
		if config.Generator == "basic" {
			outputFile = "api-client.js"
		} else {
			outputFile = "api-client.ts"
		}
	}

	// Generate based on selected generator type
	switch config.Generator {
	case "basic":
		return generateBasicJSClient(routes, typeDefs, config, libDir, outputFile)
	case "basic-ts":
		return generateBasicTSClient(routes, typeDefs, config, libDir, outputFile)
	case "axios":
		return generateAxiosClient(routes, typeDefs, config, libDir, outputFile)
	case "trpc-like":
		return generateTRPCLikeClient(routes, typeDefs, config, libDir, outputFile)
	default:
		return fmt.Errorf("unsupported generator type: %s", config.Generator)
	}
}

// generateBasicJSClient generates the new basic JavaScript API client
func generateBasicJSClient(routes []types.APIRoute, typeDefs []types.TypeDefinition, config *config.APIClientConfig, libDir, outputFile string) error {
	apiObject := generateAPIObjectString(routes, "basic")

	data := ClientTemplateData{
		UsedTypes:   []string{}, // No types needed for JavaScript
		TypesImport: "",         // No types import for JavaScript
		APIObject:   apiObject,
	}

	return generateFromTemplate(basicClientTemplate, data, filepath.Join(libDir, outputFile))
}

// generateBasicTSClient generates the original basic TypeScript API client
func generateBasicTSClient(routes []types.APIRoute, typeDefs []types.TypeDefinition, config *config.APIClientConfig, libDir, outputFile string) error {
	usedTypes := collectUsedTypes(routes, typeDefs)
	apiObject := generateAPIObjectString(routes, "basic-ts")
	requiresAuth, authType := detectAuthRequirements(routes)

	data := ClientTemplateData{
		UsedTypes:    usedTypes,
		TypesImport:  config.TypesImport,
		APIObject:    apiObject,
		RequiresAuth: requiresAuth,
		AuthType:     authType,
	}

	return generateFromTemplate(basicTSClientTemplate, data, filepath.Join(libDir, outputFile))
}

// generateAxiosClient generates an Axios-based API client
func generateAxiosClient(routes []types.APIRoute, typeDefs []types.TypeDefinition, config *config.APIClientConfig, libDir, outputFile string) error {
	usedTypes := collectUsedTypes(routes, typeDefs)
	apiObject := generateAPIObjectString(routes, "axios")
	requiresAuth, authType := detectAuthRequirements(routes)

	data := ClientTemplateData{
		UsedTypes:    usedTypes,
		TypesImport:  config.TypesImport,
		APIObject:    apiObject,
		RequiresAuth: requiresAuth,
		AuthType:     authType,
	}

	return generateFromTemplate(axiosClientTemplate, data, filepath.Join(libDir, outputFile))
}

// generateTRPCLikeClient generates a tRPC-like API client with React Query integration
func generateTRPCLikeClient(routes []types.APIRoute, typeDefs []types.TypeDefinition, config *config.APIClientConfig, libDir, outputFile string) error {
	usedTypes := collectUsedTypes(routes, typeDefs)
	apiObject := generateTRPCAPIObjectString(routes, config)
	requiresAuth, authType := detectAuthRequirements(routes)

	var queryKeys string
	if config.ReactQuery.QueryKeys {
		queryKeys = generateQueryKeys(routes)
	}

	data := ClientTemplateData{
		UsedTypes:         usedTypes,
		TypesImport:       config.TypesImport,
		APIObject:         apiObject,
		ReactQueryEnabled: config.ReactQuery.Enabled,
		QueryKeysEnabled:  config.ReactQuery.QueryKeys,
		QueryKeys:         queryKeys,
		RequiresAuth:      requiresAuth,
		AuthType:          authType,
	}

	return generateFromTemplate(trpcLikeClientTemplate, data, filepath.Join(libDir, outputFile))
}
