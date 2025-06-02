package processor

import (
	"fmt"
	"sort"
	"strings"

	"github.com/barisgit/goflux/cli/internal/typegen/config"
	"github.com/barisgit/goflux/cli/internal/typegen/types"
)

// TypeProcessor handles common type processing operations
type TypeProcessor struct {
	converter *config.CaseConverter
}

// NewTypeProcessor creates a new type processor with the given configuration
func NewTypeProcessor(casingConfig *config.CasingConfig) *TypeProcessor {
	return &TypeProcessor{
		converter: config.NewCaseConverter(casingConfig),
	}
}

// ProcessTypeName sanitizes and converts type names according to configuration
func (p *TypeProcessor) ProcessTypeName(name string) string {
	if name == "" {
		return name
	}
	return p.converter.ConvertTypeName(name)
}

// ProcessFieldName sanitizes and converts field names according to configuration
func (p *TypeProcessor) ProcessFieldName(name string) string {
	if name == "" {
		return name
	}
	return p.converter.ConvertFieldName(name)
}

// ProcessMethodName sanitizes and converts method names according to configuration
func (p *TypeProcessor) ProcessMethodName(name string) string {
	if name == "" {
		return name
	}
	return p.converter.ConvertMethodName(name)
}

// ExtractTypeFromRef extracts type name from OpenAPI $ref
func (p *TypeProcessor) ExtractTypeFromRef(ref string) string {
	if ref == "" {
		return ""
	}

	// Extract from $ref like "#/components/schemas/User"
	parts := strings.Split(ref, "/")
	if len(parts) > 0 {
		typeName := parts[len(parts)-1]
		return p.ProcessTypeName(typeName)
	}

	return ""
}

// NormalizeTypeString safely extracts type as string (handles OpenAPI 3.1 array format)
func (p *TypeProcessor) NormalizeTypeString(typeField interface{}) string {
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

// ConvertOpenAPITypeToTypeScript converts OpenAPI types to TypeScript equivalents
func (p *TypeProcessor) ConvertOpenAPITypeToTypeScript(typeStr string) string {
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

// ProcessRouteMethodName generates a method name from HTTP method and context
func (p *TypeProcessor) ProcessRouteMethodName(httpMethod string, hasIDParam bool, isNested bool) string {
	switch httpMethod {
	case "GET":
		if hasIDParam {
			return "get"
		}
		return "list"
	case "POST":
		return "create"
	case "PUT", "PATCH":
		return "update"
	case "DELETE":
		return "delete"
	default:
		return strings.ToLower(httpMethod)
	}
}

// SortTypeDefinitions sorts type definitions consistently
func (p *TypeProcessor) SortTypeDefinitions(typeDefs []types.TypeDefinition) {
	sort.Slice(typeDefs, func(i, j int) bool {
		return typeDefs[i].Name < typeDefs[j].Name
	})
}

// SortFieldInfos sorts field infos consistently
func (p *TypeProcessor) SortFieldInfos(fields []types.FieldInfo) {
	sort.Slice(fields, func(i, j int) bool {
		return fields[i].Name < fields[j].Name
	})
}

// BuildQueryParamsType builds a TypeScript type for query parameters
func (p *TypeProcessor) BuildQueryParamsType(queryParams []types.QueryParameter) string {
	if len(queryParams) == 0 {
		return ""
	}

	var fields []string
	for _, param := range queryParams {
		fieldType := param.Type
		if len(param.Enum) > 0 {
			// Use union type for enums
			enumValues := make([]string, len(param.Enum))
			for i, val := range param.Enum {
				enumValues[i] = fmt.Sprintf("'%s'", val)
			}
			fieldType = strings.Join(enumValues, " | ")
		}

		optional := ""
		if !param.Required {
			optional = "?"
		}

		fields = append(fields, fmt.Sprintf("%s%s: %s", param.Name, optional, fieldType))
	}

	return fmt.Sprintf("{ %s }", strings.Join(fields, "; "))
}

// ExtractUsedTypes extracts all types used in API routes
func (p *TypeProcessor) ExtractUsedTypes(routes []types.APIRoute, typeDefs []types.TypeDefinition) []string {
	usedTypes := make(map[string]bool)

	for _, route := range routes {
		if route.RequestType != "" {
			// Extract base type from wrapper types
			baseType := p.extractBaseType(route.RequestType)
			if baseType != "" {
				usedTypes[baseType] = true
			}
		}
		if route.ResponseType != "" {
			responseType := route.ResponseType
			// Handle array types like "User[]"
			if strings.HasSuffix(responseType, "[]") {
				responseType = strings.TrimSuffix(responseType, "[]")
			}
			usedTypes[responseType] = true
		}
	}

	// Filter to only include types that exist in our generated types
	var typeNames []string
	typeDefsMap := make(map[string]bool)
	for _, t := range typeDefs {
		typeDefsMap[t.Name] = true
	}

	for typeName := range usedTypes {
		if typeDefsMap[typeName] {
			typeNames = append(typeNames, typeName)
		}
	}

	sort.Strings(typeNames)
	return typeNames
}

// extractBaseType extracts the base type from wrapper types like Omit<Type, 'id'> or Partial<Type>
func (p *TypeProcessor) extractBaseType(typeStr string) string {
	if strings.Contains(typeStr, "Omit<") {
		start := strings.Index(typeStr, "Omit<") + 5
		end := strings.Index(typeStr[start:], ",")
		if end > 0 {
			return typeStr[start : start+end]
		}
	} else if strings.Contains(typeStr, "Partial<") {
		start := strings.Index(typeStr, "Partial<") + 8
		end := strings.Index(typeStr[start:], ">")
		if end > 0 {
			return typeStr[start : start+end]
		}
	}
	return typeStr
}

// ProcessPathParameters processes path parameters and returns clean resource parts
func (p *TypeProcessor) ProcessPathParameters(path string) ([]string, bool) {
	path = strings.Replace(path, "/api/", "", 1)
	path = strings.TrimPrefix(path, "/")
	pathParts := strings.Split(path, "/")

	var resourceParts []string
	hasIDParam := false

	for _, part := range pathParts {
		if strings.Contains(part, ":") || strings.HasPrefix(part, "{") {
			hasIDParam = true
			// Don't add parameter parts to resourceParts
		} else if part != "" {
			// Clean up invalid characters
			cleanPart := strings.ReplaceAll(part, "-", "")
			cleanPart = strings.ReplaceAll(cleanPart, ":", "")
			resourceParts = append(resourceParts, cleanPart)
		}
	}

	return resourceParts, hasIDParam
}

// BuildRequestPath builds a request path with parameter substitution
func (p *TypeProcessor) BuildRequestPath(path string, hasIDParam bool) string {
	requestPath := path
	// Remove "/api" prefix if it exists
	if strings.HasPrefix(requestPath, "/api") {
		requestPath = requestPath[4:]
	}

	if hasIDParam {
		// Replace common path parameter patterns with template variables
		requestPath = strings.ReplaceAll(requestPath, "/:id", "/${id}")
		requestPath = strings.ReplaceAll(requestPath, "/{id}", "/${id}")

		// Handle other parameter patterns
		if strings.Contains(requestPath, ":") {
			lastSlash := strings.LastIndex(requestPath, "/:")
			if lastSlash != -1 {
				nextSlash := strings.Index(requestPath[lastSlash+1:], "/")
				if nextSlash == -1 {
					requestPath = requestPath[:lastSlash] + "/${id}"
				} else {
					requestPath = requestPath[:lastSlash] + "/${id}" + requestPath[lastSlash+1+nextSlash:]
				}
			}
		} else if strings.Contains(requestPath, "{") {
			lastSlash := strings.LastIndex(requestPath, "/{")
			if lastSlash != -1 {
				nextSlash := strings.Index(requestPath[lastSlash+1:], "}")
				if nextSlash != -1 {
					end := lastSlash + 1 + nextSlash + 1
					if end >= len(requestPath) {
						requestPath = requestPath[:lastSlash] + "/${id}"
					} else {
						requestPath = requestPath[:lastSlash] + "/${id}" + requestPath[end:]
					}
				}
			}
		}
	}
	return requestPath
}

// ContainsString checks if a slice contains a string
func (p *TypeProcessor) ContainsString(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// CleanDescription extracts only the main description, removing parameter information
func (p *TypeProcessor) CleanDescription(description string) string {
	if description == "" {
		return ""
	}

	// Split by double newlines to separate sections
	sections := strings.Split(description, "\n\n")

	// Return only the first section (main description)
	if len(sections) > 0 {
		return strings.TrimSpace(sections[0])
	}

	return strings.TrimSpace(description)
}

// IsValidJSIdentifier checks if a string is a valid JavaScript identifier
func (p *TypeProcessor) IsValidJSIdentifier(s string) bool {
	return p.converter.IsValidJSIdentifier(s)
}

// Singularize converts plural words to singular
func (p *TypeProcessor) Singularize(s string) string {
	return p.converter.Singularize(s)
}
