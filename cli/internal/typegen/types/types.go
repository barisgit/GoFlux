package types

// QueryParameter represents a query parameter from the OpenAPI spec
type QueryParameter struct {
	Name        string      `json:"name"`
	Type        string      `json:"type"`
	Required    bool        `json:"required"`
	Description string      `json:"description,omitempty"`
	Default     interface{} `json:"default,omitempty"`
	Enum        []string    `json:"enum,omitempty"`
}

// APIRoute represents a single API endpoint
type APIRoute struct {
	Method          string                `json:"method"`
	Path            string                `json:"path"`
	Handler         string                `json:"handler"`
	RequestType     string                `json:"request_type,omitempty"`
	ResponseType    string                `json:"response_type,omitempty"`
	Description     string                `json:"description,omitempty"`
	QueryParameters []QueryParameter      `json:"query_parameters,omitempty"`
	RequiresAuth    bool                  `json:"requires_auth"`
	AuthType        string                `json:"auth_type,omitempty"` // "Bearer", "Basic", "ApiKey", etc.
	SecuritySchemes []map[string][]string `json:"security_schemes,omitempty"`
}

// TypeDefinition represents a Go struct converted to TypeScript
type TypeDefinition struct {
	Name        string      `json:"name"`
	Fields      []FieldInfo `json:"fields"`
	PackageName string      `json:"packageName"`
	IsEnum      bool        `json:"isEnum"`
	EnumValues  []string    `json:"enumValues,omitempty"`
}

// FieldInfo represents a field in a struct
type FieldInfo struct {
	Name        string `json:"name"`
	TypeName    string `json:"type"`
	JSONTag     string `json:"jsonTag"`
	Optional    bool   `json:"optional"`
	IsArray     bool   `json:"isArray"`
	Description string `json:"description,omitempty"`
}

// APIAnalysis contains the complete analysis results
type APIAnalysis struct {
	Routes           []APIRoute
	UsedTypes        map[string]interface{} // Simplified for OpenAPI-based analysis
	TypeDefs         []TypeDefinition
	HandlerFuncs     map[string]interface{} // Simplified for OpenAPI-based analysis
	ImportNamespaces map[string]bool
	EnumTypes        map[string]TypeDefinition
}

// APIMethod represents a generated API method
type APIMethod struct {
	Route          APIRoute
	MethodName     string
	HasIDParam     bool
	HasBodyData    bool
	HasQueryParams bool
}

// NestedAPI represents the nested API structure
type NestedAPI map[string]interface{} // Can contain either another NestedAPI or APIMethod
