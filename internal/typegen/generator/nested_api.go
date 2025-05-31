package generator

import (
	"fmt"
	"strings"

	"github.com/barisgit/goflux/internal/config"
	"github.com/barisgit/goflux/internal/typegen/types"
)

// buildNestedAPIStructure builds a nested API structure from routes
func buildNestedAPIStructure(routes []types.APIRoute) types.NestedAPI {
	nested := make(types.NestedAPI)

	for _, route := range routes {
		if generateMethodName(route) == "" {
			continue // Skip SSR routes
		}

		path := strings.Replace(route.Path, "/api/", "", 1)
		path = strings.TrimPrefix(path, "/")
		pathParts := strings.Split(path, "/")

		// Filter out parameter parts and build resource hierarchy
		var resourceParts []string
		hasIDParam := false

		for _, part := range pathParts {
			if strings.Contains(part, ":") || strings.HasPrefix(part, "{") {
				hasIDParam = true
				// Don't add parameter parts to resourceParts
			} else if part != "" {
				resourceParts = append(resourceParts, part)
			}
		}

		if len(resourceParts) == 0 {
			continue
		}

		// Determine method name based on HTTP method and context
		methodName := getMethodNameForHTTPMethod(route.Method, hasIDParam, len(resourceParts) > 1)

		method := types.APIMethod{
			Route:       route,
			MethodName:  methodName,
			HasIDParam:  hasIDParam,
			HasBodyData: route.RequestType != "",
		}

		// Build nested structure - but only use the resource parts, not parameters
		current := nested
		for i, part := range resourceParts {
			if i == len(resourceParts)-1 {
				// Last part - this is where we place the method
				if current[part] == nil {
					current[part] = make(types.NestedAPI)
				}
				if nestedPart, ok := current[part].(types.NestedAPI); ok {
					nestedPart[methodName] = method
				}
			} else {
				// Intermediate part - create nested structure
				if current[part] == nil {
					current[part] = make(types.NestedAPI)
				}
				if nestedPart, ok := current[part].(types.NestedAPI); ok {
					current = nestedPart
				}
			}
		}
	}

	return nested
}

// getMethodNameForHTTPMethod returns the method name based on HTTP method
func getMethodNameForHTTPMethod(httpMethod string, hasIDParam bool, isNested bool) string {
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

// generateMethodName generates a unique method name from a route
func generateMethodName(route types.APIRoute) string {
	path := strings.Replace(route.Path, "/api/", "", 1)
	path = strings.TrimPrefix(path, "/")

	// Skip SSR routes
	if strings.Contains(path, "ssr-data") {
		return ""
	}

	// Handle nested routes like /users/:id/profile
	pathParts := strings.Split(path, "/")
	var resourceParts []string
	var hasIDParam bool

	for _, part := range pathParts {
		if strings.Contains(part, ":") {
			hasIDParam = true
		} else if part != "" {
			// Clean up invalid characters
			cleanPart := strings.ReplaceAll(part, "-", "")
			cleanPart = strings.ReplaceAll(cleanPart, ":", "")
			resourceParts = append(resourceParts, cleanPart)
		}
	}

	if len(resourceParts) == 0 {
		return ""
	}

	// Build method name based on route structure
	var methodName string
	switch route.Method {
	case "GET":
		if hasIDParam {
			// For nested routes like /users/:id/profile -> getUserProfile
			if len(resourceParts) > 1 {
				methodName = "get" + capitalize(singularize(resourceParts[0])) + capitalize(resourceParts[1])
			} else {
				methodName = "get" + capitalize(singularize(resourceParts[0]))
			}
		} else {
			// For list routes like /users -> getUsers
			if len(resourceParts) > 1 {
				methodName = "get" + capitalize(resourceParts[0]) + capitalize(resourceParts[1])
			} else {
				methodName = "get" + capitalize(resourceParts[0])
			}
		}
	case "POST":
		if len(resourceParts) > 1 {
			methodName = "create" + capitalize(singularize(resourceParts[0])) + capitalize(resourceParts[1])
		} else {
			methodName = "create" + capitalize(singularize(resourceParts[0]))
		}
	case "PUT", "PATCH":
		if len(resourceParts) > 1 {
			methodName = "update" + capitalize(singularize(resourceParts[0])) + capitalize(resourceParts[1])
		} else {
			methodName = "update" + capitalize(singularize(resourceParts[0]))
		}
	case "DELETE":
		if len(resourceParts) > 1 {
			methodName = "delete" + capitalize(singularize(resourceParts[0])) + capitalize(resourceParts[1])
		} else {
			methodName = "delete" + capitalize(singularize(resourceParts[0]))
		}
	default:
		methodName = strings.ToLower(route.Method) + capitalize(strings.Join(resourceParts, ""))
	}

	return methodName
}

// generateAPIObjectString generates the API object as a string
func generateAPIObjectString(routes []types.APIRoute, generatorType string) string {
	var content strings.Builder
	nestedAPI := buildNestedAPIStructure(routes)

	content.WriteString("export const api = {\n")

	switch generatorType {
	case "basic":
		generateBasicJSNestedObject(&content, nestedAPI, 1)
	case "basic-ts":
		generateBasicTSNestedObject(&content, nestedAPI, 1)
	case "axios":
		generateAxiosNestedObject(&content, nestedAPI, 1)
	case "trpc-like":
		// Will be handled separately for tRPC-like
		generateBasicTSNestedObject(&content, nestedAPI, 1)
	default:
		generateBasicTSNestedObject(&content, nestedAPI, 1)
	}

	content.WriteString("}")
	return content.String()
}

// generateTRPCAPIObjectString generates tRPC-like API object with React Query hooks
func generateTRPCAPIObjectString(routes []types.APIRoute, config *config.APIClientConfig) string {
	var content strings.Builder
	nestedAPI := buildNestedAPIStructure(routes)

	content.WriteString("export const api = {\n")
	generateTRPCNestedObject(&content, nestedAPI, config, 1)
	content.WriteString("}")
	return content.String()
}

// generateQueryKeys generates query key factory functions
func generateQueryKeys(routes []types.APIRoute) string {
	var content strings.Builder
	resources := make(map[string]bool)

	// Extract unique resources
	for _, route := range routes {
		if route.Method == "GET" {
			path := strings.Replace(route.Path, "/api/", "", 1)
			pathParts := strings.Split(path, "/")
			if len(pathParts) > 0 && pathParts[0] != "" {
				resources[pathParts[0]] = true
			}
		}
	}

	// Generate key factory for each resource
	indent := "  "
	for resource := range resources {
		content.WriteString(fmt.Sprintf("%s%s: {\n", indent, resource))
		content.WriteString(fmt.Sprintf("%s  all: () => ['%s'] as const,\n", indent, resource))
		content.WriteString(fmt.Sprintf("%s  list: () => [...queryKeys.%s.all(), 'list'] as const,\n", indent, resource))
		content.WriteString(fmt.Sprintf("%s  detail: (id: string | number) => [...queryKeys.%s.all(), 'detail', id] as const,\n", indent, resource))
		content.WriteString(fmt.Sprintf("%s},\n", indent))
	}

	return content.String()
}

// createMethodTemplateData creates template data for individual method templates
func createMethodTemplateData(method types.APIMethod, generatorType string, reactQueryEnabled bool) MethodTemplateData {
	route := method.Route

	// Build parameter list
	var params []string
	if method.HasIDParam {
		params = append(params, "id: number")
	}
	if method.HasBodyData {
		requestType := route.RequestType
		if route.Method == "POST" && !strings.Contains(requestType, "Omit") {
			requestType = fmt.Sprintf("Omit<%s, 'id'>", requestType)
		} else if (route.Method == "PUT" || route.Method == "PATCH") && !strings.Contains(requestType, "Partial") {
			requestType = fmt.Sprintf("Partial<%s>", requestType)
		}
		params = append(params, "data: "+requestType)
	}

	responseType := route.ResponseType
	if responseType == "" {
		responseType = "unknown"
	}

	// Build request path with parameter substitution - pass full route.Path
	requestPath := buildRequestPath(route.Path, method.HasIDParam)

	// Determine data parameter name based on generator type
	dataParameter := "data"

	// Build mutation variable type for tRPC
	var mutationVariableType string
	if method.HasIDParam && method.HasBodyData {
		requestType := route.RequestType
		if route.Method == "POST" && !strings.Contains(requestType, "Omit") {
			requestType = fmt.Sprintf("Omit<%s, 'id'>", requestType)
		} else if (route.Method == "PUT" || route.Method == "PATCH") && !strings.Contains(requestType, "Partial") {
			requestType = fmt.Sprintf("Partial<%s>", requestType)
		}
		mutationVariableType = fmt.Sprintf("{ id: number; data: %s }", requestType)
	} else if method.HasIDParam {
		mutationVariableType = "number"
	} else if method.HasBodyData {
		requestType := route.RequestType
		if route.Method == "POST" && !strings.Contains(requestType, "Omit") {
			requestType = fmt.Sprintf("Omit<%s, 'id'>", requestType)
		} else if (route.Method == "PUT" || route.Method == "PATCH") && !strings.Contains(requestType, "Partial") {
			requestType = fmt.Sprintf("Partial<%s>", requestType)
		}
		mutationVariableType = requestType
	} else {
		mutationVariableType = "void"
	}

	// Build query parameter signatures for tRPC
	var queryParamSig, queryOptionsParamSig string
	if method.HasIDParam {
		queryParamSig = "id: number, "
		queryOptionsParamSig = "id: number"
	}

	// Build request path for mutations (React Query uses different variable patterns)
	requestPathForMutation := buildRequestPathForMutation(route.Path, method.HasIDParam, method.HasBodyData)

	return MethodTemplateData{
		Description:                    route.Description,
		Method:                         route.Method,
		MethodLower:                    strings.ToLower(route.Method),
		ParameterSignature:             strings.Join(params, ", "),
		ParameterSignatureJS:           strings.Join(params, ", "),
		QueryParameterSignature:        queryParamSig,
		QueryOptionsParameterSignature: queryOptionsParamSig,
		ResponseType:                   responseType,
		RequestPath:                    requestPath,
		RequestPathForMutation:         requestPathForMutation,
		HasIDParam:                     method.HasIDParam,
		HasBodyData:                    method.HasBodyData,
		DataParameter:                  dataParameter,
		QueryKey:                       strings.TrimPrefix(route.Path, "/api/"),
		MutationVariableType:           mutationVariableType,
		ReactQueryEnabled:              reactQueryEnabled,
	}
}

// createMethodTemplateDataJS creates template data for JavaScript method templates with destructured parameters
func createMethodTemplateDataJS(method types.APIMethod) MethodTemplateData {
	route := method.Route

	// Build JavaScript parameter list with destructuring
	var jsParams []string
	if method.HasIDParam && method.HasBodyData {
		// Methods like PUT /users/:id with body data
		jsParams = append(jsParams, "{ id, data }")
	} else if method.HasIDParam {
		// Methods like GET /users/:id or DELETE /users/:id
		jsParams = append(jsParams, "{ id }")
	} else if method.HasBodyData {
		// Methods like POST /users with body data
		jsParams = append(jsParams, "{ data }")
	} else {
		// Methods like GET /users (no parameters)
		// Leave empty - no parameters needed
	}

	// Build request path with parameter substitution - pass full route.Path
	requestPath := buildRequestPath(route.Path, method.HasIDParam)

	// Extract clean description (remove parameter information since we'll use JSDoc @param tags)
	cleanDescription := extractCleanDescription(route.Description)

	return MethodTemplateData{
		Description:          cleanDescription,
		Method:               route.Method,
		MethodLower:          strings.ToLower(route.Method),
		ParameterSignatureJS: strings.Join(jsParams, ", "),
		RequestPath:          requestPath,
		HasIDParam:           method.HasIDParam,
		HasBodyData:          method.HasBodyData,
		DataParameter:        "data",
	}
}
