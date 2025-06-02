package analyzer

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/barisgit/goflux/cli/internal/typegen/config"
	"github.com/barisgit/goflux/cli/internal/typegen/processor"
	"github.com/barisgit/goflux/cli/internal/typegen/types"
)

// Analyzer handles OpenAPI-based analysis of projects
type Analyzer struct {
	processor *processor.TypeProcessor
	debug     bool
}

// NewAnalyzer creates a new analyzer with the given configuration
func NewAnalyzer(casingConfig *config.CasingConfig, debug bool) *Analyzer {
	return &Analyzer{
		processor: processor.NewTypeProcessor(casingConfig),
		debug:     debug,
	}
}

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
	Security    []map[string][]string     `json:"security,omitempty"`
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
func (a *Analyzer) AnalyzeProject(projectPath string) (*types.APIAnalysis, error) {
	if a.debug {
		fmt.Printf("Analyzing project at: %s\n", projectPath)
	}

	// Look for OpenAPI spec in the project
	spec, err := a.loadOpenAPISpec(projectPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load OpenAPI spec: %w", err)
	}

	// Parse the OpenAPI spec to extract routes and types
	analysis := a.parseOpenAPISpec(spec)

	if a.debug {
		fmt.Printf("Extracted %d routes and %d types from OpenAPI spec\n",
			len(analysis.Routes), len(analysis.TypeDefs))
	}

	return analysis, nil
}

// loadOpenAPISpec loads the OpenAPI specification from available sources
func (a *Analyzer) loadOpenAPISpec(projectPath string) (*OpenAPISpec, error) {
	// Try to find OpenAPI spec file
	possiblePaths := []string{
		filepath.Join(projectPath, "build", "openapi.json"),
		filepath.Join(projectPath, "openapi.json"),
		filepath.Join(projectPath, "docs", "openapi.json"),
		filepath.Join(projectPath, "api", "openapi.json"),
	}

	for _, path := range possiblePaths {
		if a.debug {
			fmt.Printf("Checking for OpenAPI spec at: %s\n", path)
		}

		if _, err := os.Stat(path); err == nil {
			data, err := os.ReadFile(path)
			if err != nil {
				if a.debug {
					fmt.Printf("Failed to read %s: %v\n", path, err)
				}
				continue
			}

			var spec OpenAPISpec
			err = json.Unmarshal(data, &spec)
			if err != nil {
				if a.debug {
					fmt.Printf("Failed to parse %s: %v\n", path, err)
				}
				continue
			}

			if a.debug {
				fmt.Printf("Successfully loaded OpenAPI spec from: %s\n", path)
			}
			return &spec, nil
		}
	}

	return nil, fmt.Errorf("no OpenAPI specification found in paths: %v", possiblePaths)
}

// parseOpenAPISpec converts OpenAPI spec to our internal format
func (a *Analyzer) parseOpenAPISpec(spec *OpenAPISpec) *types.APIAnalysis {
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
				Description: a.buildRouteDescription(operation),
			}

			// Extract request type from requestBody
			if operation.RequestBody != nil {
				route.RequestType = a.extractTypeFromRequestBody(operation.RequestBody)
			}

			// Extract response type from responses
			route.ResponseType = a.extractTypeFromResponses(operation.Responses)

			// Extract query parameters
			if operation.Parameters != nil {
				route.QueryParameters = a.extractQueryParameters(operation.Parameters)
			}

			// Extract security requirements
			if len(operation.Security) > 0 {
				route.RequiresAuth = true
				route.SecuritySchemes = operation.Security
				route.AuthType = a.extractAuthType(operation.Security)
			}

			analysis.Routes = append(analysis.Routes, route)
		}
	}

	// Extract type definitions from components/schemas
	if spec.Components != nil && spec.Components.Schemas != nil {
		analysis.TypeDefs = a.extractTypeDefinitions(spec.Components.Schemas)
	}

	return analysis
}

// extractTypeFromRequestBody extracts the type name from request body schema
func (a *Analyzer) extractTypeFromRequestBody(requestBody *RequestBody) string {
	for _, mediaType := range requestBody.Content {
		if mediaType.Schema != nil {
			return a.extractTypeName(mediaType.Schema)
		}
	}
	return ""
}

// extractTypeFromResponses extracts the type name from successful response schemas
func (a *Analyzer) extractTypeFromResponses(responses map[string]ResponseObject) string {
	// Look for 200, 201, or other success responses
	successCodes := []string{"200", "201", "202"}
	for _, code := range successCodes {
		if response, exists := responses[code]; exists {
			for _, mediaType := range response.Content {
				if mediaType.Schema != nil {
					return a.extractTypeName(mediaType.Schema)
				}
			}
		}
	}
	return ""
}

// extractTypeName extracts the type name from a schema
func (a *Analyzer) extractTypeName(schema *Schema) string {
	if schema.Ref != "" {
		return a.processor.ExtractTypeFromRef(schema.Ref)
	}

	// Handle type field which can be string or array in OpenAPI 3.1
	typeStr := a.processor.NormalizeTypeString(schema.Type)

	if typeStr == "array" && schema.Items != nil {
		itemType := a.extractTypeName(schema.Items)
		if itemType != "" {
			return itemType + "[]"
		}
	}

	// For simple types, return the TypeScript equivalent
	return a.processor.ConvertOpenAPITypeToTypeScript(typeStr)
}

// extractTypeDefinitions converts OpenAPI schemas to TypeScript type definitions
func (a *Analyzer) extractTypeDefinitions(schemas map[string]Schema) []types.TypeDefinition {
	var typeDefs []types.TypeDefinition

	for name, schema := range schemas {
		// Skip internal OpenAPI types (those with $ in the name)
		if strings.Contains(name, "$") {
			continue
		}

		typeDef := a.convertSchemaToTypeDef(name, schema)
		if typeDef.Name != "" {
			typeDefs = append(typeDefs, typeDef)
		}
	}

	// Sort for consistent output
	a.processor.SortTypeDefinitions(typeDefs)

	return typeDefs
}

// convertSchemaToTypeDef converts an OpenAPI schema to a TypeScript type definition
func (a *Analyzer) convertSchemaToTypeDef(name string, schema Schema) types.TypeDefinition {
	// Sanitize the type name for TypeScript compatibility
	sanitizedName := a.processor.ProcessTypeName(name)

	if len(schema.Enum) > 0 {
		// Handle enum types
		enumValues := make([]string, len(schema.Enum))
		for i, val := range schema.Enum {
			enumValues[i] = fmt.Sprintf(`"%v"`, val)
		}
		return types.TypeDefinition{
			Name:       sanitizedName,
			IsEnum:     true,
			EnumValues: enumValues,
		}
	}

	typeStr := a.processor.NormalizeTypeString(schema.Type)
	if typeStr == "object" && schema.Properties != nil {
		// Handle object types
		var fields []types.FieldInfo

		for propName, propSchema := range schema.Properties {
			// Skip $schema fields
			if propName == "$schema" {
				continue
			}

			fieldType := a.schemaToTypeScriptType(propSchema)
			isRequired := a.processor.ContainsString(schema.Required, propName)

			field := types.FieldInfo{
				Name:        propName,
				TypeName:    fieldType,
				JSONTag:     propName,
				Optional:    !isRequired,
				IsArray:     a.processor.NormalizeTypeString(propSchema.Type) == "array",
				Description: propSchema.Description,
			}
			fields = append(fields, field)
		}

		// Sort fields for consistent output
		a.processor.SortFieldInfos(fields)

		return types.TypeDefinition{
			Name:   sanitizedName,
			Fields: fields,
		}
	}

	return types.TypeDefinition{} // Empty definition for unsupported schemas
}

// schemaToTypeScriptType converts an OpenAPI schema to a TypeScript type string
func (a *Analyzer) schemaToTypeScriptType(schema Schema) string {
	if schema.Ref != "" {
		return a.processor.ExtractTypeFromRef(schema.Ref)
	}

	typeStr := a.processor.NormalizeTypeString(schema.Type)
	switch typeStr {
	case "string":
		return "string"
	case "number", "integer":
		return "number"
	case "boolean":
		return "boolean"
	case "array":
		if schema.Items != nil {
			itemType := a.schemaToTypeScriptType(*schema.Items)
			return itemType + "[]"
		}
		return "unknown[]"
	case "object":
		return "unknown"
	default:
		return "unknown"
	}
}

// buildRouteDescription builds a comprehensive route description using OpenAPI details
func (a *Analyzer) buildRouteDescription(operation *Operation) string {
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

// extractQueryParameters extracts query parameters from operation parameters
func (a *Analyzer) extractQueryParameters(parameters []Parameter) []types.QueryParameter {
	var queryParams []types.QueryParameter

	for _, param := range parameters {
		if param.In == "query" && param.Schema != nil {
			queryParam := types.QueryParameter{
				Name:        param.Name,
				Type:        a.convertSchemaToTSType(param.Schema),
				Required:    param.Required,
				Description: param.Description,
			}

			// Extract default value if present
			if param.Schema.Example != nil {
				queryParam.Default = param.Schema.Example
			}

			// Extract enum values if present
			if len(param.Schema.Enum) > 0 {
				enumValues := make([]string, len(param.Schema.Enum))
				for i, val := range param.Schema.Enum {
					enumValues[i] = fmt.Sprintf("%v", val)
				}
				queryParam.Enum = enumValues
			}

			queryParams = append(queryParams, queryParam)
		}
	}

	return queryParams
}

// convertSchemaToTSType converts an OpenAPI schema to TypeScript type string
func (a *Analyzer) convertSchemaToTSType(schema *Schema) string {
	if schema == nil {
		return "unknown"
	}

	if schema.Ref != "" {
		return a.processor.ExtractTypeFromRef(schema.Ref)
	}

	typeStr := a.processor.NormalizeTypeString(schema.Type)
	switch typeStr {
	case "string":
		return "string"
	case "number", "integer":
		return "number"
	case "boolean":
		return "boolean"
	case "array":
		if schema.Items != nil {
			itemType := a.convertSchemaToTSType(schema.Items)
			return itemType + "[]"
		}
		return "unknown[]"
	case "object":
		return "Record<string, unknown>"
	default:
		return "unknown"
	}
}

// extractAuthType determines the authentication type from security schemes
func (a *Analyzer) extractAuthType(security []map[string][]string) string {
	if len(security) == 0 {
		return ""
	}

	// Check the first security requirement
	for authType := range security[0] {
		switch authType {
		case "Bearer", "bearer":
			return "Bearer"
		case "Basic", "basic":
			return "Basic"
		case "ApiKey", "apiKey", "api_key":
			return "ApiKey"
		default:
			return authType // Return as-is for custom schemes
		}
	}

	return ""
}

// AnalyzeProject is the main entry point (backwards compatibility)
func AnalyzeProject(projectPath string, debug bool) (*types.APIAnalysis, error) {
	analyzer := NewAnalyzer(config.DefaultCasingConfig(), debug)
	return analyzer.AnalyzeProject(projectPath)
}
