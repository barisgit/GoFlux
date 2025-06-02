package openapi

import (
	"fmt"
	"net/http"
	"reflect"
	"strings"

	"github.com/barisgit/goflux/internal/core"
	"github.com/danielgtaylor/huma/v2"
)

// SchemaProcessor handles OpenAPI schema generation with flexible multipart form support
//
// Supported tags for multipart form control:
//
// Field-level tags:
//   - `upload:"file|binary"` - Force field to be treated as binary file upload
//   - `upload:"text"`        - Force field to be treated as text input in multipart form
//   - `format:"binary"`      - Mark field as binary (same as upload:"file")
//   - `mime:"type"`          - Specify MIME type (binary if not text/json/xml)
//   - `required:"true"`      - Mark field as required
//   - `optional:"true"`      - Mark field as optional (overrides auto-detection)
//   - `doc:"description"`    - Field description for OpenAPI
//
// Body-level tags:
//   - `contentType:"multipart/form-data"` - Force multipart form (overrides auto-detection)
//   - `contentType:"application/json"`    - Force JSON (disables auto-detection)
//
// Auto-detection rules (when no explicit contentType is set):
//  1. Explicit upload/format tags take precedence
//  2. MIME type tags indicating non-text content
//  3. Common file field names (file, upload, attachment, image, etc.) with string/[]byte types
//  4. []byte fields are always treated as binary in multipart forms
//
// Examples:
//
//	type FileUpload struct {
//	  Body struct {
//	    File        string `json:"file" format:"binary"`           // Binary file
//	    Document    string `json:"doc" upload:"file"`              // Binary file
//	    Description string `json:"desc" upload:"text"`             // Text field
//	    Avatar      string `json:"avatar" mime:"image/png"`        // Binary by MIME
//	    Resume      []byte `json:"resume"`                         // Auto-detected binary
//	    Optional    string `json:"tags" optional:"true"`           // Optional field
//	  } `contentType:"multipart/form-data"`                        // Explicit multipart
//	}
type SchemaProcessor struct{}

// NewSchemaProcessor creates a new schema processor
func NewSchemaProcessor() *SchemaProcessor {
	return &SchemaProcessor{}
}

// ProcessOperation processes an operation like huma.Register does
func (p *SchemaProcessor) ProcessOperation(operation *huma.Operation, api huma.API, inputType, outputType reflect.Type, deps []*core.DependencyCore) error {
	oapi := api.OpenAPI()
	registry := oapi.Components.Schemas

	// Initialize responses if needed (like huma.Register does)
	if operation.Responses == nil {
		operation.Responses = map[string]*huma.Response{}
	}

	// Process input type for parameters and request body (like huma.Register does)
	if err := p.processInputType(operation, registry, inputType); err != nil {
		return fmt.Errorf("error processing input type: %w", err)
	}

	// Process dependency input fields for additional parameters
	if err := p.processDependencyInputFields(operation, registry, deps); err != nil {
		return fmt.Errorf("error processing dependency input fields: %w", err)
	}

	// Process output type for response schemas (like huma.Register does)
	if err := p.processOutputType(operation, registry, outputType); err != nil {
		return fmt.Errorf("error processing output type: %w", err)
	}

	// Set up error responses (like huma.Register does)
	if err := p.setupErrorResponses(operation, registry); err != nil {
		return fmt.Errorf("error setting up error responses: %w", err)
	}

	// Ensure all validation schemas are set up properly (like huma.Register does)
	if operation.RequestBody != nil {
		for _, mediatype := range operation.RequestBody.Content {
			if mediatype.Schema != nil {
				mediatype.Schema.PrecomputeMessages()
			}
		}
	}

	return nil
}

// processInputType processes the input type to generate OpenAPI parameters and request body
func (p *SchemaProcessor) processInputType(operation *huma.Operation, registry huma.Registry, inputType reflect.Type) error {
	// Initialize parameters slice if needed
	if operation.Parameters == nil {
		operation.Parameters = []*huma.Param{}
	}

	// Process each field in the input struct
	for i := 0; i < inputType.NumField(); i++ {
		field := inputType.Field(i)
		if !field.IsExported() {
			continue
		}

		// Check for parameter tags (path, query, header, cookie)
		if p.processFieldAsParameter(operation, registry, field) {
			continue
		}

		// Check for body field
		if field.Name == "Body" {
			if err := p.processFieldAsRequestBody(operation, registry, field, inputType); err != nil {
				return err
			}
		}

		// Check for RawBody field - enhanced with dynamic schema generation
		if field.Name == "RawBody" {
			if err := p.processRawBodyField(operation, registry, field, inputType); err != nil {
				return err
			}
		}
	}

	return nil
}

// processFieldAsParameter processes a field as a path/query/header/cookie parameter
func (p *SchemaProcessor) processFieldAsParameter(operation *huma.Operation, registry huma.Registry, field reflect.StructField) bool {
	var paramName, paramIn string

	// Check parameter tags
	if pathParam := field.Tag.Get("path"); pathParam != "" {
		paramName = pathParam
		paramIn = "path"
	} else if queryParam := field.Tag.Get("query"); queryParam != "" {
		paramName = strings.Split(queryParam, ",")[0] // Handle "name,explode" format
		paramIn = "query"
	} else if headerParam := field.Tag.Get("header"); headerParam != "" {
		paramName = headerParam
		paramIn = "header"
	} else if cookieParam := field.Tag.Get("cookie"); cookieParam != "" {
		paramName = cookieParam
		paramIn = "cookie"
	} else {
		return false // Not a parameter field
	}

	// Create parameter schema
	schema := huma.SchemaFromField(registry, field, "")

	// Determine if parameter is required
	required := paramIn == "path" || field.Tag.Get("required") == "true"

	// Create parameter
	param := &huma.Param{
		Name:     paramName,
		In:       paramIn,
		Required: required,
		Schema:   schema,
	}

	// Add description from schema if available
	if schema != nil && schema.Description != "" {
		param.Description = schema.Description
	}

	// Add example if available
	if example := field.Tag.Get("example"); example != "" {
		param.Example = p.jsonTagValue(registry, field.Type.Name(), schema, example)
	}

	operation.Parameters = append(operation.Parameters, param)
	return true
}

// processFieldAsRequestBody processes a field as request body
func (p *SchemaProcessor) processFieldAsRequestBody(operation *huma.Operation, registry huma.Registry, field reflect.StructField, parentType reflect.Type) error {
	// Initialize request body if needed
	if operation.RequestBody == nil {
		operation.RequestBody = &huma.RequestBody{
			Content: map[string]*huma.MediaType{},
		}
	}

	// Determine content type with explicit tag support
	contentType := "application/json"
	if ct := field.Tag.Get("contentType"); ct != "" {
		contentType = ct
	} else {
		// Auto-detect file upload only if no explicit contentType is set
		if p.detectFileUpload(field.Type) {
			contentType = "multipart/form-data"
		}
	}

	// Determine if required
	required := field.Tag.Get("required") == "true" || (field.Type.Kind() != reflect.Pointer && field.Type.Kind() != reflect.Interface)
	operation.RequestBody.Required = required

	if contentType == "multipart/form-data" {
		// Generate multipart/form-data schema for file uploads
		schema := p.generateMultipartSchema(registry, field.Type)
		operation.RequestBody.Content[contentType] = &huma.MediaType{
			Schema: schema,
		}
	} else {
		// Generate regular schema (JSON, XML, etc.)
		hint := p.getHint(parentType, field.Name, operation.OperationID+"Request")
		if nameHint := field.Tag.Get("nameHint"); nameHint != "" {
			hint = nameHint
		}
		schema := huma.SchemaFromField(registry, field, hint)

		// Add to request body
		operation.RequestBody.Content[contentType] = &huma.MediaType{
			Schema: schema,
		}
	}

	return nil
}

// processOutputType processes the output type to generate response schemas
func (p *SchemaProcessor) processOutputType(operation *huma.Operation, registry huma.Registry, outputType reflect.Type) error {
	// Default status
	status := operation.DefaultStatus
	if status == 0 {
		status = http.StatusOK
	}
	statusStr := fmt.Sprintf("%d", status)

	// Initialize response if needed
	if operation.Responses[statusStr] == nil {
		operation.Responses[statusStr] = &huma.Response{
			Description: http.StatusText(status),
			Headers:     map[string]*huma.Param{},
		}
	}

	response := operation.Responses[statusStr]

	// Process output fields
	for i := 0; i < outputType.NumField(); i++ {
		field := outputType.Field(i)
		if !field.IsExported() {
			continue
		}

		switch field.Name {
		case "Status":
			// Status field doesn't affect OpenAPI schema directly
			continue
		case "Body":
			// Process body field for response schema
			if err := p.processOutputBodyField(response, registry, field, outputType, operation.OperationID); err != nil {
				return err
			}
		default:
			// Check if it's a header field
			if headerName := p.getHeaderName(field); headerName != "" {
				if err := p.processOutputHeaderField(response, registry, field, headerName, outputType, operation.OperationID); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// processOutputBodyField processes the body field of output type
func (p *SchemaProcessor) processOutputBodyField(response *huma.Response, registry huma.Registry, field reflect.StructField, parentType reflect.Type, operationID string) error {
	// Initialize content if needed
	if response.Content == nil {
		response.Content = map[string]*huma.MediaType{}
	}

	// Determine content type
	contentType := "application/json"

	// Check if the field's type implements ContentTypeFilter
	if reflect.PointerTo(field.Type).Implements(reflect.TypeFor[huma.ContentTypeFilter]()) {
		instance := reflect.New(field.Type).Interface().(huma.ContentTypeFilter)
		contentType = instance.ContentType(contentType)
	}

	// Generate schema
	hint := p.getHint(parentType, field.Name, operationID+"Response")
	if nameHint := field.Tag.Get("nameHint"); nameHint != "" {
		hint = nameHint
	}
	schema := huma.SchemaFromField(registry, field, hint)

	// Add to response content
	response.Content[contentType] = &huma.MediaType{
		Schema: schema,
	}

	return nil
}

// processOutputHeaderField processes header fields in output type
func (p *SchemaProcessor) processOutputHeaderField(response *huma.Response, registry huma.Registry, field reflect.StructField, headerName string, parentType reflect.Type, operationID string) error {
	// Generate schema for header
	hint := p.getHint(parentType, field.Name, operationID+fmt.Sprintf("%d", http.StatusOK)+headerName)
	schema := huma.SchemaFromField(registry, field, hint)

	// Handle slice types (multiple header values)
	if field.Type.Kind() == reflect.Slice {
		schema = huma.SchemaFromField(registry, reflect.StructField{
			Type: field.Type.Elem(),
			Tag:  field.Tag,
		}, hint)
	}

	// Create header parameter
	response.Headers[headerName] = &huma.Param{
		Schema: schema,
	}

	return nil
}

// processDependencyInputFields processes input fields from dependencies that have InputFields defined
func (p *SchemaProcessor) processDependencyInputFields(operation *huma.Operation, registry huma.Registry, deps []*core.DependencyCore) error {
	// Iterate over each dependency
	for _, dep := range deps {
		// Check if the dependency has InputFields defined
		if dep.InputFields != nil {
			// Process each field in the dependency's InputFields
			for i := 0; i < dep.InputFields.NumField(); i++ {
				field := dep.InputFields.Field(i)
				if !field.IsExported() {
					continue
				}

				// Check for parameter tags (path, query, header, cookie)
				if p.processFieldAsParameter(operation, registry, field) {
					continue
				}

				// Check for body field
				if field.Name == "Body" {
					if err := p.processFieldAsRequestBody(operation, registry, field, dep.InputFields); err != nil {
						return err
					}
				}
			}
		}
	}

	return nil
}

// setupErrorResponses sets up standard error responses
func (p *SchemaProcessor) setupErrorResponses(operation *huma.Operation, registry huma.Registry) error {
	// Create example error for schema
	exampleErr := huma.NewError(0, "")
	errContentType := "application/json"
	if ctf, ok := exampleErr.(huma.ContentTypeFilter); ok {
		errContentType = ctf.ContentType(errContentType)
	}

	errType := p.deref(reflect.TypeOf(exampleErr))
	errSchema := registry.Schema(errType, true, p.getHint(errType, "", "Error"))

	// Add common error responses
	errorCodes := []int{http.StatusBadRequest, http.StatusUnprocessableEntity, http.StatusInternalServerError}

	// Add authentication/authorization errors if the operation has security requirements
	if len(operation.Security) > 0 {
		errorCodes = append([]int{http.StatusUnauthorized, http.StatusForbidden}, errorCodes...)
	}

	for _, code := range errorCodes {
		codeStr := fmt.Sprintf("%d", code)
		if operation.Responses[codeStr] == nil {
			operation.Responses[codeStr] = &huma.Response{
				Description: http.StatusText(code),
				Content: map[string]*huma.MediaType{
					errContentType: {
						Schema: errSchema,
					},
				},
			}
		}
	}

	// Add default error response if no specific errors defined
	if len(operation.Responses) <= 1 {
		operation.Responses["default"] = &huma.Response{
			Description: "Error",
			Content: map[string]*huma.MediaType{
				errContentType: {
					Schema: errSchema,
				},
			},
		}
	}

	return nil
}

// getHeaderName extracts header name from struct field
func (p *SchemaProcessor) getHeaderName(field reflect.StructField) string {
	if header := field.Tag.Get("header"); header != "" {
		return header
	}
	// Default to field name for headers
	if field.Name != "Body" && field.Name != "Status" {
		return field.Name
	}
	return ""
}

// Helper functions
func (p *SchemaProcessor) getHint(parent reflect.Type, name string, other string) string {
	if parent.Name() != "" {
		return parent.Name() + name
	}
	return p.sanitizeTypeScriptTypeName(other)
}

// sanitizeTypeScriptTypeName converts operation IDs to valid TypeScript type names
// using PascalCase convention
func (p *SchemaProcessor) sanitizeTypeScriptTypeName(name string) string {
	return toPascalCase(name)
}

// toPascalCase converts strings to PascalCase for TypeScript type names
// Examples: "get-api-validate-by-value" -> "GetApiValidateByValue"
//
//	"user_profile" -> "UserProfile"
//	"create-user-request" -> "CreateUserRequest"
func toPascalCase(s string) string {
	if len(s) == 0 {
		return s
	}

	// Split on common separators: hyphens, underscores, spaces, dots
	parts := strings.FieldsFunc(s, func(c rune) bool {
		return c == '-' || c == '_' || c == ' ' || c == '.'
	})

	var result strings.Builder
	for _, part := range parts {
		if len(part) > 0 {
			// Capitalize first letter and make rest lowercase
			result.WriteString(strings.ToUpper(part[:1]) + strings.ToLower(part[1:]))
		}
	}

	return result.String()
}

func (p *SchemaProcessor) deref(t reflect.Type) reflect.Type {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	return t
}

func (p *SchemaProcessor) jsonTagValue(registry huma.Registry, typeName string, schema *huma.Schema, value string) interface{} {
	// Simple implementation - just return the string value
	// Huma has a more complex implementation that converts based on schema type
	return value
}

func (p *SchemaProcessor) detectFileUpload(t reflect.Type) bool {
	// Dereference pointer types
	t = p.deref(t)

	// Only process struct types
	if t.Kind() != reflect.Struct {
		return false
	}

	// Check each field in the struct for file upload indicators
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if !field.IsExported() {
			continue
		}

		// Priority 1: Explicit upload tags
		if upload := field.Tag.Get("upload"); upload != "" {
			switch upload {
			case "file", "binary", "multipart":
				return true
			}
		}

		// Priority 2: Format tags indicating binary data
		if format := field.Tag.Get("format"); format == "binary" {
			return true
		}

		// Priority 3: MIME type tags indicating file uploads
		if mime := field.Tag.Get("mime"); mime != "" {
			// Any MIME type that's not text/json/xml indicates potential file upload
			if !strings.HasPrefix(mime, "text/") &&
				!strings.HasPrefix(mime, "application/json") &&
				!strings.HasPrefix(mime, "application/xml") {
				return true
			}
		}

		// Priority 4: Common field names and types (less restrictive)
		fieldName := strings.ToLower(field.Name)
		if p.isLikelyFileField(fieldName, field.Type) {
			return true
		}
	}

	return false
}

// isLikelyFileField checks if a field name and type combination suggests a file upload
func (p *SchemaProcessor) isLikelyFileField(fieldName string, fieldType reflect.Type) bool {
	// Common file field names
	fileFieldNames := []string{
		"file", "upload", "attachment", "document", "image",
		"photo", "video", "audio", "media", "avatar", "logo",
		"csv", "pdf", "zip", "archive", "backup",
	}

	for _, name := range fileFieldNames {
		if fieldName == name || strings.Contains(fieldName, name) {
			// If it's a byte slice, definitely a file
			if fieldType.Kind() == reflect.Slice && fieldType.Elem().Kind() == reflect.Uint8 {
				return true
			}
			// If it's a string, likely a file reference/content
			if fieldType.Kind() == reflect.String {
				return true
			}
		}
	}

	return false
}

func (p *SchemaProcessor) generateMultipartSchema(registry huma.Registry, t reflect.Type) *huma.Schema {
	// Dereference pointer types
	t = p.deref(t)

	schema := &huma.Schema{
		Type:       "object",
		Properties: make(map[string]*huma.Schema),
		Required:   []string{},
	}

	// Process each field in the struct
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if !field.IsExported() {
			continue
		}

		// Get field name from JSON tag or field name
		fieldName := field.Name
		if jsonTag := field.Tag.Get("json"); jsonTag != "" {
			fieldName = strings.Split(jsonTag, ",")[0]
		}

		// Skip empty field names
		if fieldName == "" || fieldName == "-" {
			continue
		}

		// Determine field type with explicit tag overrides
		fieldSchema := p.determineMultipartFieldSchema(field, registry)
		schema.Properties[fieldName] = fieldSchema

		// Check if field is required
		if p.isFieldRequired(field) {
			schema.Required = append(schema.Required, fieldName)
		}
	}

	return schema
}

// determineMultipartFieldSchema determines the appropriate schema for a multipart field
func (p *SchemaProcessor) determineMultipartFieldSchema(field reflect.StructField, registry huma.Registry) *huma.Schema {
	// Priority 1: Explicit override tags
	if upload := field.Tag.Get("upload"); upload != "" {
		switch upload {
		case "file", "binary":
			return &huma.Schema{
				Type:        "string",
				Format:      "binary",
				Description: field.Tag.Get("doc"),
			}
		case "text":
			return &huma.Schema{
				Type:        "string",
				Description: field.Tag.Get("doc"),
			}
		}
	}

	// Priority 2: Format tag
	if format := field.Tag.Get("format"); format == "binary" {
		return &huma.Schema{
			Type:        "string",
			Format:      "binary",
			Description: field.Tag.Get("doc"),
		}
	}

	// Priority 3: MIME type tag
	if mime := field.Tag.Get("mime"); mime != "" {
		if strings.HasPrefix(mime, "text/") ||
			strings.HasPrefix(mime, "application/json") ||
			strings.HasPrefix(mime, "application/xml") {
			return &huma.Schema{
				Type:        "string",
				Description: field.Tag.Get("doc"),
			}
		} else {
			return &huma.Schema{
				Type:        "string",
				Format:      "binary",
				Description: field.Tag.Get("doc"),
			}
		}
	}

	// Priority 4: Auto-detect based on field name and type
	fieldName := strings.ToLower(field.Name)
	if p.isLikelyFileField(fieldName, field.Type) {
		return &huma.Schema{
			Type:        "string",
			Format:      "binary",
			Description: field.Tag.Get("doc"),
		}
	}

	// Priority 5: Default to regular field schema
	fieldSchema := huma.SchemaFromField(registry, field, "")
	return fieldSchema
}

// isFieldRequired determines if a field should be marked as required
func (p *SchemaProcessor) isFieldRequired(field reflect.StructField) bool {
	// Explicit required tag
	if required := field.Tag.Get("required"); required == "true" {
		return true
	}

	// Explicit optional tag overrides default behavior
	if optional := field.Tag.Get("optional"); optional == "true" {
		return false
	}

	// Non-pointer types are typically required unless explicitly marked optional
	return field.Type.Kind() != reflect.Pointer
}

// processRawBodyField processes RawBody fields, generating dynamic schemas from tags
func (p *SchemaProcessor) processRawBodyField(operation *huma.Operation, registry huma.Registry, field reflect.StructField, parentType reflect.Type) error {
	// Initialize request body if needed
	if operation.RequestBody == nil {
		operation.RequestBody = &huma.RequestBody{
			Content: map[string]*huma.MediaType{},
		}
	}

	// Get content type from tag or default to multipart/form-data for multipart types
	contentType := "multipart/form-data"
	if ct := field.Tag.Get("contentType"); ct != "" {
		contentType = ct
	}

	// Check if this is a multipart.Form type
	if field.Type.String() == "multipart.Form" {
		// Generate schema dynamically from the schema tag
		schema := p.generateDynamicMultipartSchema(field.Tag.Get("schema"))
		operation.RequestBody.Content[contentType] = &huma.MediaType{
			Schema: schema,
		}
		operation.RequestBody.Required = true
	} else if strings.HasPrefix(field.Type.Name(), "MultipartFormFiles") {
		// Handle huma.MultipartFormFiles[T] types like Huma does
		// Get the generic type parameter (T) from MultipartFormFiles[T]
		if field.Type.NumField() > 0 {
			// Look for the 'data' field which contains the struct definition
			dataField, found := field.Type.FieldByName("data")
			if found && dataField.Type.Kind() == reflect.Pointer {
				// Generate schema from the struct type inside MultipartFormFiles
				structType := dataField.Type.Elem()
				schema := p.generateMultipartFormFileSchema(registry, structType)
				operation.RequestBody.Content[contentType] = &huma.MediaType{
					Schema:   schema,
					Encoding: p.generateMultipartEncoding(structType),
				}
				operation.RequestBody.Required = false // MultipartFormFiles are typically optional
			} else {
				// Fallback to basic multipart schema
				schema := p.generateDynamicMultipartSchema("")
				operation.RequestBody.Content[contentType] = &huma.MediaType{
					Schema: schema,
				}
				operation.RequestBody.Required = true
			}
		} else {
			// Fallback to basic multipart schema
			schema := p.generateDynamicMultipartSchema("")
			operation.RequestBody.Content[contentType] = &huma.MediaType{
				Schema: schema,
			}
			operation.RequestBody.Required = true
		}
	} else {
		// Handle other RawBody types (binary, etc.)
		schema := &huma.Schema{
			Type:   "string",
			Format: "binary",
		}
		operation.RequestBody.Content[contentType] = &huma.MediaType{
			Schema: schema,
		}
		operation.RequestBody.Required = true
	}

	return nil
}

// generateMultipartFormFileSchema generates schema for huma.MultipartFormFiles[T] generic types
func (p *SchemaProcessor) generateMultipartFormFileSchema(registry huma.Registry, structType reflect.Type) *huma.Schema {
	schema := &huma.Schema{
		Type:       "object",
		Properties: make(map[string]*huma.Schema),
		Required:   []string{},
	}

	// Process each field in the struct
	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)
		if !field.IsExported() {
			continue
		}

		// Get field name from form tag
		formTag := field.Tag.Get("form")
		if formTag == "" {
			// Skip fields without form tags in MultipartFormFiles
			continue
		}

		fieldName := formTag

		// Generate schema based on field type and tags
		var fieldSchema *huma.Schema

		// Check if this is a huma.FormFile type (file upload field)
		if field.Type.Name() == "FormFile" {
			fieldSchema = &huma.Schema{
				Type:        "string",
				Format:      "binary",
				Description: field.Tag.Get("doc"),
			}
		} else {
			// Regular form field (string, int, etc.)
			fieldSchema = huma.SchemaFromField(registry, field, "")
			if fieldSchema.Description == "" {
				fieldSchema.Description = field.Tag.Get("doc")
			}
		}

		schema.Properties[fieldName] = fieldSchema

		// Check if field is required (form fields are typically optional unless explicitly marked)
		if field.Tag.Get("required") == "true" {
			schema.Required = append(schema.Required, fieldName)
		}
	}

	return schema
}

// generateMultipartEncoding generates encoding information for multipart forms
func (p *SchemaProcessor) generateMultipartEncoding(structType reflect.Type) map[string]*huma.Encoding {
	encoding := make(map[string]*huma.Encoding)

	// Process each field to set up proper encoding
	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)
		if !field.IsExported() {
			continue
		}

		formTag := field.Tag.Get("form")
		if formTag == "" {
			continue
		}

		// Set encoding for file fields
		if field.Type.Name() == "FormFile" {
			encoding[formTag] = &huma.Encoding{
				ContentType: "application/octet-stream",
			}
		}
	}

	return encoding
}

// generateDynamicMultipartSchema generates a schema from a custom schema definition
// Format: "field:type:modifiers,field2:type:modifiers"
// Types: string, binary, integer, boolean, array
// Modifiers: required, enum:val1|val2|val3
func (p *SchemaProcessor) generateDynamicMultipartSchema(schemaDef string) *huma.Schema {
	schema := &huma.Schema{
		Type:       "object",
		Properties: make(map[string]*huma.Schema),
		Required:   []string{},
	}

	if schemaDef == "" {
		// Fallback to basic file upload schema
		schema.Properties["file"] = &huma.Schema{
			Type:        "string",
			Format:      "binary",
			Description: "File to upload",
		}
		schema.Required = []string{"file"}
		return schema
	}

	// Parse schema definition
	fields := strings.Split(schemaDef, ",")
	for _, fieldDef := range fields {
		parts := strings.Split(strings.TrimSpace(fieldDef), ":")
		if len(parts) < 2 {
			continue
		}

		fieldName := strings.TrimSpace(parts[0])

		fieldSchema := &huma.Schema{}
		isRequired := false

		// Process type and modifiers
		for i := 1; i < len(parts); i++ {
			part := strings.TrimSpace(parts[i])

			switch part {
			case "string":
				fieldSchema.Type = "string"
			case "binary":
				fieldSchema.Type = "string"
				fieldSchema.Format = "binary"
			case "integer":
				fieldSchema.Type = "integer"
			case "boolean":
				fieldSchema.Type = "boolean"
			case "array":
				if fieldSchema.Type == "string" && fieldSchema.Format == "binary" {
					fieldSchema.Type = "array"
					fieldSchema.Items = &huma.Schema{
						Type:   "string",
						Format: "binary",
					}
				}
			case "required":
				isRequired = true
			default:
				// Check for enum
				if strings.HasPrefix(part, "enum:") {
					enumValues := strings.Split(strings.TrimPrefix(part, "enum:"), "|")
					fieldSchema.Enum = make([]interface{}, len(enumValues))
					for i, val := range enumValues {
						fieldSchema.Enum[i] = strings.TrimSpace(val)
					}
				}
			}
		}

		// Set default description
		switch {
		case fieldSchema.Format == "binary":
			fieldSchema.Description = fmt.Sprintf("%s file upload", strings.Title(fieldName))
		case fieldName == "name":
			fieldSchema.Description = "Name or identifier"
		case fieldName == "description":
			fieldSchema.Description = "Optional description"
		case fieldName == "metadata":
			fieldSchema.Description = "Additional metadata"
		default:
			fieldSchema.Description = fmt.Sprintf("%s field", strings.Title(fieldName))
		}

		schema.Properties[fieldName] = fieldSchema

		if isRequired {
			schema.Required = append(schema.Required, fieldName)
		}
	}

	return schema
}
