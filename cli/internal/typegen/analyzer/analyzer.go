package analyzer

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/barisgit/goflux/cli/internal/typegen/types"
)

// OpenAPISpec represents the OpenAPI 3.x specification structure
type OpenAPISpec struct {
	OpenAPI    string                 `json:"openapi"`
	Info       map[string]interface{} `json:"info"`
	Paths      map[string]PathItem    `json:"paths"`
	Components *Components            `json:"components,omitempty"`
}

type Components struct {
	Schemas map[string]Schema `json:"schemas,omitempty"`
}

type Schema struct {
	Type                 interface{}       `json:"type,omitempty"` // Can be string or array
	Properties           map[string]Schema `json:"properties,omitempty"`
	Items                *Schema           `json:"items,omitempty"`
	Required             []string          `json:"required,omitempty"`
	Description          string            `json:"description,omitempty"`
	Example              interface{}       `json:"example,omitempty"`
	Enum                 []interface{}     `json:"enum,omitempty"`
	AdditionalProperties interface{}       `json:"additionalProperties,omitempty"`
	Ref                  string            `json:"$ref,omitempty"`
	Format               string            `json:"format,omitempty"`
	AllOf                []Schema          `json:"allOf,omitempty"`
	OneOf                []Schema          `json:"oneOf,omitempty"`
	AnyOf                []Schema          `json:"anyOf,omitempty"`
}

type PathItem struct {
	Get    *Operation `json:"get,omitempty"`
	Post   *Operation `json:"post,omitempty"`
	Put    *Operation `json:"put,omitempty"`
	Delete *Operation `json:"delete,omitempty"`
	Patch  *Operation `json:"patch,omitempty"`
}

type Operation struct {
	OperationID string                    `json:"operationId,omitempty"`
	Summary     string                    `json:"summary,omitempty"`
	Description string                    `json:"description,omitempty"`
	Tags        []string                  `json:"tags,omitempty"`
	Parameters  []Parameter               `json:"parameters,omitempty"`
	RequestBody *RequestBody              `json:"requestBody,omitempty"`
	Responses   map[string]ResponseObject `json:"responses,omitempty"`
}

type Parameter struct {
	Name        string  `json:"name"`
	In          string  `json:"in"`
	Required    bool    `json:"required"`
	Description string  `json:"description,omitempty"`
	Schema      *Schema `json:"schema,omitempty"`
}

type RequestBody struct {
	Description string                     `json:"description,omitempty"`
	Required    bool                       `json:"required"`
	Content     map[string]MediaTypeObject `json:"content"`
}

type ResponseObject struct {
	Description string                     `json:"description"`
	Content     map[string]MediaTypeObject `json:"content,omitempty"`
}

type MediaTypeObject struct {
	Schema *Schema `json:"schema,omitempty"`
}

// AnalyzeProject performs OpenAPI-based analysis of a project
func AnalyzeProject(projectPath string, debug bool) (*types.APIAnalysis, error) {
	if debug {
		fmt.Printf("Analyzing project at: %s\n", projectPath)
	}

	// Look for OpenAPI spec in the project
	spec, err := loadOpenAPISpec(projectPath, debug)
	if err != nil {
		return nil, fmt.Errorf("failed to load OpenAPI spec: %w", err)
	}

	// Parse the OpenAPI spec to extract routes and types
	analysis := parseOpenAPISpec(spec, debug)

	if debug {
		fmt.Printf("Extracted %d routes and %d types from OpenAPI spec\n",
			len(analysis.Routes), len(analysis.TypeDefs))
	}

	return analysis, nil
}

// loadOpenAPISpec loads the OpenAPI specification from available sources
func loadOpenAPISpec(projectPath string, debug bool) (*OpenAPISpec, error) {
	// Try to find OpenAPI spec file
	possiblePaths := []string{
		filepath.Join(projectPath, "build", "openapi.json"),
		filepath.Join(projectPath, "openapi.json"),
		filepath.Join(projectPath, "docs", "openapi.json"),
		filepath.Join(projectPath, "api", "openapi.json"),
	}

	for _, path := range possiblePaths {
		if debug {
			fmt.Printf("Checking for OpenAPI spec at: %s\n", path)
		}

		if _, err := os.Stat(path); err == nil {
			data, err := os.ReadFile(path)
			if err != nil {
				if debug {
					fmt.Printf("Failed to read %s: %v\n", path, err)
				}
				continue
			}

			var spec OpenAPISpec
			err = json.Unmarshal(data, &spec)
			if err != nil {
				if debug {
					fmt.Printf("Failed to parse %s: %v\n", path, err)
				}
				continue
			}

			if debug {
				fmt.Printf("Successfully loaded OpenAPI spec from: %s\n", path)
			}
			return &spec, nil
		}
	}

	return nil, fmt.Errorf("no OpenAPI specification found in paths: %v", possiblePaths)
}

// parseOpenAPISpec converts OpenAPI spec to our internal format
func parseOpenAPISpec(spec *OpenAPISpec, debug bool) *types.APIAnalysis {
	analysis := &types.APIAnalysis{
		Routes:           []types.APIRoute{},
		TypeDefs:         []types.TypeDefinition{},
		UsedTypes:        make(map[string]interface{}),
		HandlerFuncs:     make(map[string]interface{}),
		ImportNamespaces: make(map[string]bool),
		EnumTypes:        make(map[string]types.TypeDefinition),
	}

	// Extract routes from paths
	for path, pathItem := range spec.Paths {
		operations := map[string]*Operation{
			"GET":    pathItem.Get,
			"POST":   pathItem.Post,
			"PUT":    pathItem.Put,
			"DELETE": pathItem.Delete,
			"PATCH":  pathItem.Patch,
		}

		for method, operation := range operations {
			if operation == nil {
				continue
			}

			route := types.APIRoute{
				Method:      method,
				Path:        path,
				Handler:     operation.OperationID,
				Description: buildRouteDescription(operation),
			}

			// Extract request type from requestBody
			if operation.RequestBody != nil {
				route.RequestType = extractTypeFromRequestBody(operation.RequestBody)
			}

			// Extract response type from responses
			route.ResponseType = extractTypeFromResponses(operation.Responses)

			analysis.Routes = append(analysis.Routes, route)
		}
	}

	// Extract type definitions from components/schemas
	if spec.Components != nil && spec.Components.Schemas != nil {
		analysis.TypeDefs = extractTypeDefinitions(spec.Components.Schemas, debug)
	}

	return analysis
}

// extractTypeFromRequestBody extracts the type name from request body schema
func extractTypeFromRequestBody(requestBody *RequestBody) string {
	for _, mediaType := range requestBody.Content {
		if mediaType.Schema != nil {
			return extractTypeName(mediaType.Schema)
		}
	}
	return ""
}

// extractTypeFromResponses extracts the type name from successful response schemas
func extractTypeFromResponses(responses map[string]ResponseObject) string {
	// Look for 200, 201, or other success responses
	successCodes := []string{"200", "201", "202"}
	for _, code := range successCodes {
		if response, exists := responses[code]; exists {
			for _, mediaType := range response.Content {
				if mediaType.Schema != nil {
					return extractTypeName(mediaType.Schema)
				}
			}
		}
	}
	return ""
}

// extractTypeName extracts the type name from a schema
func extractTypeName(schema *Schema) string {
	if schema.Ref != "" {
		// Extract from $ref like "#/components/schemas/User"
		parts := strings.Split(schema.Ref, "/")
		if len(parts) > 0 {
			return parts[len(parts)-1]
		}
	}

	// Handle type field which can be string or array in OpenAPI 3.1
	typeStr := getTypeString(schema.Type)

	if typeStr == "array" && schema.Items != nil {
		itemType := extractTypeName(schema.Items)
		if itemType != "" {
			return itemType + "[]"
		}
	}

	// For simple types, return the TypeScript equivalent
	switch typeStr {
	case "string":
		return "string"
	case "number", "integer":
		return "number"
	case "boolean":
		return "boolean"
	case "object":
		return "unknown"
	default:
		return "unknown"
	}
}

// getTypeString safely extracts type as string (handles OpenAPI 3.1 array format)
func getTypeString(typeField interface{}) string {
	if typeField == nil {
		return ""
	}

	switch v := typeField.(type) {
	case string:
		return v
	case []interface{}:
		// OpenAPI 3.1 allows type to be an array like ["string", "null"]
		if len(v) > 0 {
			if str, ok := v[0].(string); ok {
				return str
			}
		}
	}
	return ""
}

// extractTypeDefinitions converts OpenAPI schemas to TypeScript type definitions
func extractTypeDefinitions(schemas map[string]Schema, debug bool) []types.TypeDefinition {
	var typeDefs []types.TypeDefinition

	for name, schema := range schemas {
		// Skip internal OpenAPI types (those with $ in the name)
		if strings.Contains(name, "$") {
			continue
		}

		typeDef := convertSchemaToTypeDef(name, schema, debug)
		if typeDef.Name != "" {
			typeDefs = append(typeDefs, typeDef)
		}
	}

	// Sort for consistent output
	sort.Slice(typeDefs, func(i, j int) bool {
		return typeDefs[i].Name < typeDefs[j].Name
	})

	return typeDefs
}

// convertSchemaToTypeDef converts an OpenAPI schema to a TypeScript type definition
func convertSchemaToTypeDef(name string, schema Schema, debug bool) types.TypeDefinition {
	if len(schema.Enum) > 0 {
		// Handle enum types
		enumValues := make([]string, len(schema.Enum))
		for i, val := range schema.Enum {
			enumValues[i] = fmt.Sprintf(`"%v"`, val)
		}
		return types.TypeDefinition{
			Name:       name,
			IsEnum:     true,
			EnumValues: enumValues,
		}
	}

	typeStr := getTypeString(schema.Type)
	if typeStr == "object" && schema.Properties != nil {
		// Handle object types
		var fields []types.FieldInfo

		for propName, propSchema := range schema.Properties {
			// Skip $schema fields
			if propName == "$schema" {
				continue
			}

			fieldType := schemaToTypeScriptType(propSchema)
			isRequired := contains(schema.Required, propName)

			field := types.FieldInfo{
				Name:        propName,
				TypeName:    fieldType,
				JSONTag:     propName,
				Optional:    !isRequired,
				IsArray:     getTypeString(propSchema.Type) == "array",
				Description: propSchema.Description,
			}
			fields = append(fields, field)
		}

		// Sort fields for consistent output
		sort.Slice(fields, func(i, j int) bool {
			return fields[i].Name < fields[j].Name
		})

		return types.TypeDefinition{
			Name:   name,
			Fields: fields,
		}
	}

	return types.TypeDefinition{} // Empty definition for unsupported schemas
}

// schemaToTypeScriptType converts an OpenAPI schema to a TypeScript type string
func schemaToTypeScriptType(schema Schema) string {
	if schema.Ref != "" {
		parts := strings.Split(schema.Ref, "/")
		if len(parts) > 0 {
			return parts[len(parts)-1]
		}
	}

	typeStr := getTypeString(schema.Type)
	switch typeStr {
	case "string":
		return "string"
	case "number", "integer":
		return "number"
	case "boolean":
		return "boolean"
	case "array":
		if schema.Items != nil {
			itemType := schemaToTypeScriptType(*schema.Items)
			return itemType + "[]"
		}
		return "unknown[]"
	case "object":
		return "unknown"
	default:
		return "unknown"
	}
}

// contains checks if a slice contains a string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// buildRouteDescription builds a comprehensive route description using OpenAPI details
func buildRouteDescription(operation *Operation) string {
	var description strings.Builder

	// Use the primary description or summary
	if operation.Description != "" {
		description.WriteString(operation.Description)
	} else if operation.Summary != "" {
		description.WriteString(operation.Summary)
	}

	// Add parameter descriptions
	if len(operation.Parameters) > 0 {
		var paramDescs []string
		for _, param := range operation.Parameters {
			if param.Description != "" {
				paramDescs = append(paramDescs, fmt.Sprintf("%s: %s", param.Name, param.Description))
			}
		}
		if len(paramDescs) > 0 {
			if description.Len() > 0 {
				description.WriteString("\n\n")
			}
			description.WriteString("Parameters:\n")
			for _, desc := range paramDescs {
				description.WriteString(fmt.Sprintf("- %s\n", desc))
			}
		}
	}

	// Add request body description
	if operation.RequestBody != nil && operation.RequestBody.Description != "" {
		if description.Len() > 0 {
			description.WriteString("\n\n")
		}
		description.WriteString(fmt.Sprintf("Request: %s", operation.RequestBody.Description))
	}

	return strings.TrimSpace(description.String())
}
