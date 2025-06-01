package goflux

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"strings"

	"github.com/danielgtaylor/huma/v2"
)

// Dependency represents something that can be injected
type Dependency struct {
	Name        string
	LoadFn      func(ctx context.Context, input interface{}) (interface{}, error)
	TypeFn      func() reflect.Type
	InputFields reflect.Type // Optional: additional input fields this dependency needs
}

// Load executes the dependency's load function
func (d *Dependency) Load(ctx context.Context, input interface{}) (interface{}, error) {
	return d.LoadFn(ctx, input)
}

// Type returns the type this dependency provides
func (d *Dependency) Type() reflect.Type {
	return d.TypeFn()
}

// NewDependency creates a new dependency with automatic type inference
// Works with both value types (T) and pointer types (*T)
func NewDependency[T any](name string, loadFn func(context.Context, interface{}) (T, error)) Dependency {
	return Dependency{
		Name: name,
		LoadFn: func(ctx context.Context, input interface{}) (interface{}, error) {
			return loadFn(ctx, input)
		},
		TypeFn: func() reflect.Type {
			// Get the return type directly from the function signature
			fnType := reflect.TypeOf(loadFn)
			return fnType.Out(0) // First return value (T or *T)
		},
		InputFields: nil, // No additional input fields by default
	}
}

// NewDependencyWithInput creates a dependency that requires additional input fields
// I is the input fields type, T is the dependency output type
func NewDependencyWithInput[I, T any](name string, loadFn func(context.Context, interface{}) (T, error)) Dependency {
	return Dependency{
		Name: name,
		LoadFn: func(ctx context.Context, input interface{}) (interface{}, error) {
			return loadFn(ctx, input)
		},
		TypeFn: func() reflect.Type {
			fnType := reflect.TypeOf(loadFn)
			return fnType.Out(0)
		},
		InputFields: reflect.TypeOf((*I)(nil)).Elem(),
	}
}

// Middleware can modify context or halt execution (works with huma.Context)
type Middleware func(ctx huma.Context, next func(huma.Context))

// Procedure represents a fluent builder for dependency injection
type Procedure struct {
	deps        []Dependency
	middlewares []Middleware
	security    []map[string][]string
}

// NewProcedure creates a new procedure builder
func NewProcedure() *Procedure {
	return &Procedure{
		deps:        make([]Dependency, 0),
		middlewares: make([]Middleware, 0),
		security:    make([]map[string][]string, 0),
	}
}

// InjectDeps adds dependencies to the procedure
func InjectDeps(deps ...Dependency) *Procedure {
	return &Procedure{
		deps:        deps,
		middlewares: make([]Middleware, 0),
		security:    make([]map[string][]string, 0),
	}
}

// Use adds middleware to the procedure
func (p *Procedure) Use(middleware Middleware) *Procedure {
	return &Procedure{
		deps:        p.deps,
		middlewares: append(p.middlewares, middleware),
		security:    p.security,
	}
}

// Inject adds additional dependencies
func (p *Procedure) Inject(deps ...Dependency) *Procedure {
	return &Procedure{
		deps:        append(p.deps, deps...),
		middlewares: p.middlewares,
		security:    p.security,
	}
}

// WithSecurity adds security requirements to the procedure
func (p *Procedure) WithSecurity(security ...map[string][]string) *Procedure {
	return &Procedure{
		deps:        p.deps,
		middlewares: p.middlewares,
		security:    append(p.security, security...),
	}
}

// RegisterWithDI automatically determines input and output types from the handler signature
// This works with any handler signature: func(context.Context, *InputType, ...deps) (*OutputType, error)
func RegisterWithDI(
	api huma.API,
	operation huma.Operation,
	procedure *Procedure,
	handler interface{},
) {
	// Capture caller information for better error reporting
	_, callerFile, callerLine, _ := runtime.Caller(1)

	// Extract just the filename and relative path to avoid exposing build machine paths
	relativeFile := filepath.Base(callerFile)
	if strings.Contains(callerFile, "/") {
		// Try to get a more useful relative path (last 2-3 directories)
		parts := strings.Split(callerFile, "/")
		if len(parts) >= 3 {
			relativeFile = strings.Join(parts[len(parts)-3:], "/")
		} else if len(parts) >= 2 {
			relativeFile = strings.Join(parts[len(parts)-2:], "/")
		}
	}

	handlerValue := reflect.ValueOf(handler)
	handlerType := handlerValue.Type()

	// Validate handler signature
	if handlerType.Kind() != reflect.Func {
		panic(fmt.Sprintf("handler must be a function, got %T", handler))
	}

	if handlerType.NumIn() < 2 {
		panic("handler must have at least 2 parameters: (context.Context, *InputType)")
	}

	if handlerType.NumOut() != 2 {
		panic("handler must have exactly 2 return values: (*OutputType, error)")
	}

	// Extract input and output types from handler signature
	inputParamType := handlerType.In(1)    // Second parameter (*InputType)
	outputReturnType := handlerType.Out(0) // First return value (*OutputType)

	// Validate that input param is a pointer
	if inputParamType.Kind() != reflect.Pointer {
		panic("handler's input parameter must be a pointer type (*InputType)")
	}

	// Validate that output return is a pointer
	if outputReturnType.Kind() != reflect.Pointer {
		panic("handler's output return must be a pointer type (*OutputType)")
	}

	// Get the actual types (dereference pointers)
	inputType := inputParamType.Elem()    // InputType
	outputType := outputReturnType.Elem() // OutputType

	// Build dependency mapping and validate
	depsByType := buildDependencyMappingFromHandler(procedure, operation.OperationID, handlerType, relativeFile, callerLine)

	// Apply middlewares and security to operation first
	applyMiddlewaresAndSecurity(&operation, procedure)

	// Process the operation like huma.Register does - this is the key part we were missing!
	processOperationLikeHuma(&operation, api, inputType, outputType, procedure)

	// Create a dependency injection wrapper that will be registered as the actual handler
	diWrapper := func(ctx huma.Context) {
		// Create an instance of the input type
		inputPtr := reflect.New(inputType)
		input := inputPtr.Interface()

		// Parse the input from the request (path params, query params, body, etc.)
		if err := parseHumaInput(api, ctx, inputPtr, inputType); err != nil {
			huma.WriteErr(api, ctx, http.StatusBadRequest, "Failed to parse input", err)
			return
		}

		// Prepare handler arguments
		handlerArgs := []reflect.Value{
			reflect.ValueOf(ctx.Context()),
			inputPtr,
		}

		// Resolve and inject dependencies
		for i := 2; i < handlerType.NumIn(); i++ {
			paramType := handlerType.In(i)

			if dep, exists := depsByType[paramType]; exists {
				resolved, err := dep.Load(ctx.Context(), input)
				if err != nil {
					huma.WriteErr(api, ctx, http.StatusInternalServerError, "Failed to resolve dependency", err)
					return
				}
				handlerArgs = append(handlerArgs, reflect.ValueOf(resolved))
			} else {
				err := fmt.Errorf("no dependency found for parameter %d of type %v", i-2, paramType)
				huma.WriteErr(api, ctx, http.StatusInternalServerError, "Missing dependency", err)
				return
			}
		}

		// Call the original handler
		results := handlerValue.Call(handlerArgs)

		// Handle the response
		if len(results) != 2 {
			huma.WriteErr(api, ctx, http.StatusInternalServerError, "Handler returned wrong number of values", fmt.Errorf("expected 2, got %d", len(results)))
			return
		}

		// Check for error (second return value)
		if !results[1].IsNil() {
			err := results[1].Interface().(error)
			// Handle different error types appropriately
			var se huma.StatusError
			if errors.As(err, &se) {
				huma.WriteErr(api, ctx, se.GetStatus(), se.Error())
			} else {
				huma.WriteErr(api, ctx, http.StatusInternalServerError, "Handler error", err)
			}
			return
		}

		// Handle successful response (first return value)
		output := results[0].Interface()
		if output != nil {
			// Use Huma's exact response pipeline: Transform -> Marshal
			writeHumaOutput(api, ctx, output, outputType, operation)
		} else {
			ctx.SetStatus(operation.DefaultStatus)
		}
	}

	// Register with the adapter
	adapter := api.Adapter()
	adapter.Handle(&operation, api.Middlewares().Handler(operation.Middlewares.Handler(diWrapper)))

	// Add to OpenAPI if not hidden (like huma.Register does)
	if !operation.Hidden {
		api.OpenAPI().AddOperation(&operation)
	}
}

// processOperationLikeHuma does the same OpenAPI processing that huma.Register does
func processOperationLikeHuma(operation *huma.Operation, api huma.API, inputType, outputType reflect.Type, procedure *Procedure) {
	oapi := api.OpenAPI()
	registry := oapi.Components.Schemas

	// Initialize responses if needed (like huma.Register does)
	if operation.Responses == nil {
		operation.Responses = map[string]*huma.Response{}
	}

	// Process input type for parameters and request body (like huma.Register does)
	processInputTypeForOpenAPI(operation, registry, inputType)

	// Process dependency input fields for additional parameters
	processDependencyInputFields(operation, registry, procedure)

	// Process output type for response schemas (like huma.Register does)
	processOutputTypeForOpenAPI(operation, registry, outputType)

	// Set up error responses (like huma.Register does)
	setupErrorResponses(operation, registry)

	// Ensure all validation schemas are set up properly (like huma.Register does)
	if operation.RequestBody != nil {
		for _, mediatype := range operation.RequestBody.Content {
			if mediatype.Schema != nil {
				mediatype.Schema.PrecomputeMessages()
			}
		}
	}
}

// parseHumaInput parses the incoming request into the input struct
func parseHumaInput(api huma.API, ctx huma.Context, inputPtr reflect.Value, inputType reflect.Type) error {
	input := inputPtr.Elem()

	// Parse each field in the input struct
	for i := 0; i < inputType.NumField(); i++ {
		field := inputType.Field(i)
		if !field.IsExported() {
			continue
		}

		fieldValue := input.Field(i)

		// Handle different parameter types
		if err := parseInputField(ctx, field, fieldValue); err != nil {
			return fmt.Errorf("failed to parse field %s: %w", field.Name, err)
		}

		// Handle body field
		if field.Name == "Body" {
			if err := parseBodyField(api, ctx, fieldValue); err != nil {
				return fmt.Errorf("failed to parse body: %w", err)
			}
		}
	}

	return nil
}

// parseInputField parses a single input field based on its tags
func parseInputField(ctx huma.Context, field reflect.StructField, fieldValue reflect.Value) error {
	// Handle path parameters
	if pathParam := field.Tag.Get("path"); pathParam != "" {
		value := ctx.Param(pathParam)
		return setFieldValue(fieldValue, value, field.Type)
	}

	// Handle query parameters
	if queryParam := field.Tag.Get("query"); queryParam != "" {
		paramName := strings.Split(queryParam, ",")[0] // Handle "name,explode" format
		value := ctx.Query(paramName)
		return setFieldValue(fieldValue, value, field.Type)
	}

	// Handle header parameters
	if headerParam := field.Tag.Get("header"); headerParam != "" {
		value := ctx.Header(headerParam)
		return setFieldValue(fieldValue, value, field.Type)
	}

	// Handle cookie parameters
	if cookieParam := field.Tag.Get("cookie"); cookieParam != "" {
		// Get cookie value from request headers
		cookieHeader := ctx.Header("Cookie")
		if cookieHeader != "" {
			// Parse cookies manually since huma.Context doesn't provide Cookie method
			cookies := parseCookies(cookieHeader)
			if value, exists := cookies[cookieParam]; exists {
				return setFieldValue(fieldValue, value, field.Type)
			}
		}
	}

	return nil
}

// parseCookies parses the Cookie header string and returns a map of cookie name to value
func parseCookies(cookieHeader string) map[string]string {
	cookies := make(map[string]string)

	// Split by semicolon and parse each cookie
	for _, cookie := range strings.Split(cookieHeader, ";") {
		cookie = strings.TrimSpace(cookie)
		if cookie == "" {
			continue
		}

		// Split by = to get name and value
		parts := strings.SplitN(cookie, "=", 2)
		if len(parts) == 2 {
			name := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			cookies[name] = value
		}
	}

	return cookies
}

// parseBodyField parses the request body into the field
func parseBodyField(api huma.API, ctx huma.Context, fieldValue reflect.Value) error {
	bodyReader := ctx.BodyReader()
	if bodyReader == nil {
		return nil
	}

	var bodyBytes []byte
	buf := make([]byte, 1024)
	for {
		n, err := bodyReader.Read(buf)
		if n > 0 {
			bodyBytes = append(bodyBytes, buf[:n]...)
		}
		if err != nil {
			if err != io.EOF {
				return err
			}
			break
		}
	}

	if len(bodyBytes) > 0 {
		contentType := ctx.Header("Content-Type")
		return api.Unmarshal(contentType, bodyBytes, fieldValue.Addr().Interface())
	}

	return nil
}

// setFieldValue sets a field value from a string, handling type conversion
func setFieldValue(fieldValue reflect.Value, value string, fieldType reflect.Type) error {
	if value == "" {
		return nil
	}

	// Handle pointer types
	if fieldType.Kind() == reflect.Pointer {
		if fieldValue.IsNil() {
			fieldValue.Set(reflect.New(fieldType.Elem()))
		}
		return setFieldValue(fieldValue.Elem(), value, fieldType.Elem())
	}

	// Handle different types
	switch fieldType.Kind() {
	case reflect.String:
		fieldValue.SetString(value)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if intVal, err := strconv.ParseInt(value, 10, 64); err == nil {
			fieldValue.SetInt(intVal)
		} else {
			return fmt.Errorf("invalid integer value: %s", value)
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if uintVal, err := strconv.ParseUint(value, 10, 64); err == nil {
			fieldValue.SetUint(uintVal)
		} else {
			return fmt.Errorf("invalid unsigned integer value: %s", value)
		}
	case reflect.Float32, reflect.Float64:
		if floatVal, err := strconv.ParseFloat(value, 64); err == nil {
			fieldValue.SetFloat(floatVal)
		} else {
			return fmt.Errorf("invalid float value: %s", value)
		}
	case reflect.Bool:
		if boolVal, err := strconv.ParseBool(value); err == nil {
			fieldValue.SetBool(boolVal)
		} else {
			return fmt.Errorf("invalid boolean value: %s", value)
		}
	case reflect.Slice:
		// Handle slice types (e.g., multiple query parameters)
		return setSliceValue(fieldValue, value, fieldType)
	default:
		return fmt.Errorf("unsupported field type: %v", fieldType)
	}

	return nil
}

// setSliceValue handles setting slice values from comma-separated strings
func setSliceValue(fieldValue reflect.Value, value string, fieldType reflect.Type) error {
	elemType := fieldType.Elem()
	values := strings.Split(value, ",")

	slice := reflect.MakeSlice(fieldType, len(values), len(values))

	for i, v := range values {
		elemValue := slice.Index(i)
		if err := setFieldValue(elemValue, strings.TrimSpace(v), elemType); err != nil {
			return err
		}
	}

	fieldValue.Set(slice)
	return nil
}

// writeHumaOutput writes the output response using Huma's exact pipeline
func writeHumaOutput(api huma.API, ctx huma.Context, output interface{}, outputType reflect.Type, operation huma.Operation) {
	outputValue := reflect.ValueOf(output)
	if outputValue.Kind() == reflect.Pointer {
		outputValue = outputValue.Elem()
	}

	// Set the default status if not already set
	status := operation.DefaultStatus
	if status == 0 {
		status = http.StatusOK
	}

	// Check for Status field in output
	if statusField := outputValue.FieldByName("Status"); statusField.IsValid() && statusField.Kind() == reflect.Int {
		if statusField.Int() != 0 {
			status = int(statusField.Int())
		}
	}

	// Handle response headers (like Huma does)
	ct := ""
	for i := 0; i < outputValue.NumField(); i++ {
		field := outputType.Field(i)
		if !field.IsExported() {
			continue
		}

		if headerName := getHeaderName(field); headerName != "" {
			headerValue := outputValue.Field(i)
			if headerValue.IsValid() && !headerValue.IsZero() {
				if headerValue.Kind() == reflect.String && headerName == "Content-Type" {
					// Track custom content type (like Huma does)
					ct = headerValue.String()
				}
				ctx.SetHeader(headerName, fmt.Sprintf("%v", headerValue.Interface()))
			}
		}
	}

	// Handle response body (exactly like Huma's Register function)
	if bodyField := outputValue.FieldByName("Body"); bodyField.IsValid() {
		body := bodyField.Interface()

		// Handle byte slice special case (like Huma does)
		if b, ok := body.([]byte); ok {
			ctx.SetStatus(status)
			ctx.BodyWriter().Write(b)
			return
		}

		// Use Huma's exact response pipeline: Transform -> Marshal
		if ct == "" {
			// Content negotiation (like Huma does)
			var err error
			ct, err = api.Negotiate(ctx.Header("Accept"))
			if err != nil {
				huma.WriteErr(api, ctx, http.StatusNotAcceptable, "unable to marshal response", err)
				return
			}

			if ctf, ok := body.(huma.ContentTypeFilter); ok {
				ct = ctf.ContentType(ct)
			}

			ctx.SetHeader("Content-Type", ct)
		}

		// Transform the response body (like Huma does)
		tval, terr := api.Transform(ctx, strconv.Itoa(status), body)
		if terr != nil {
			huma.WriteErr(api, ctx, http.StatusInternalServerError, "error transforming response", terr)
			return
		}

		ctx.SetStatus(status)

		// Marshal and write the response (like Huma does)
		if status != http.StatusNoContent && status != http.StatusNotModified {
			if merr := api.Marshal(ctx.BodyWriter(), ct, tval); merr != nil {
				huma.WriteErr(api, ctx, http.StatusInternalServerError, "error marshaling response", merr)
				return
			}
		}
	} else {
		// No body field, just set status
		ctx.SetStatus(status)
	}
}

// processInputTypeForOpenAPI processes the input type to generate OpenAPI parameters and request body
func processInputTypeForOpenAPI(operation *huma.Operation, registry huma.Registry, inputType reflect.Type) {
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
		if processFieldAsParameter(operation, registry, field) {
			continue
		}

		// Check for body field
		if field.Name == "Body" {
			processFieldAsRequestBody(operation, registry, field, inputType)
		}
	}
}

// processFieldAsParameter processes a field as a path/query/header/cookie parameter
func processFieldAsParameter(operation *huma.Operation, registry huma.Registry, field reflect.StructField) bool {
	var paramName, paramIn string

	// Check parameter tags
	if p := field.Tag.Get("path"); p != "" {
		paramName = p
		paramIn = "path"
	} else if q := field.Tag.Get("query"); q != "" {
		paramName = strings.Split(q, ",")[0] // Handle "name,explode" format
		paramIn = "query"
	} else if h := field.Tag.Get("header"); h != "" {
		paramName = h
		paramIn = "header"
	} else if c := field.Tag.Get("cookie"); c != "" {
		paramName = c
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
		param.Example = jsonTagValue(registry, field.Type.Name(), schema, example)
	}

	operation.Parameters = append(operation.Parameters, param)
	return true
}

// processFieldAsRequestBody processes a field as request body
func processFieldAsRequestBody(operation *huma.Operation, registry huma.Registry, field reflect.StructField, parentType reflect.Type) {
	// Initialize request body if needed
	if operation.RequestBody == nil {
		operation.RequestBody = &huma.RequestBody{
			Content: map[string]*huma.MediaType{},
		}
	}

	// Determine content type
	contentType := "application/json"
	if ct := field.Tag.Get("contentType"); ct != "" {
		contentType = ct
	}

	// Determine if required
	required := field.Tag.Get("required") == "true" || (field.Type.Kind() != reflect.Pointer && field.Type.Kind() != reflect.Interface)
	operation.RequestBody.Required = required

	// Generate schema for the body field
	hint := getHint(parentType, field.Name, operation.OperationID+"Request")
	if nameHint := field.Tag.Get("nameHint"); nameHint != "" {
		hint = nameHint
	}
	schema := huma.SchemaFromField(registry, field, hint)

	// Add to request body
	operation.RequestBody.Content[contentType] = &huma.MediaType{
		Schema: schema,
	}
}

// processOutputTypeForOpenAPI processes the output type to generate response schemas
func processOutputTypeForOpenAPI(operation *huma.Operation, registry huma.Registry, outputType reflect.Type) {
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
			processOutputBodyField(response, registry, field, outputType, operation.OperationID)
		default:
			// Check if it's a header field
			if headerName := getHeaderName(field); headerName != "" {
				processOutputHeaderField(response, registry, field, headerName, outputType, operation.OperationID)
			}
		}
	}
}

// processOutputBodyField processes the body field of output type
func processOutputBodyField(response *huma.Response, registry huma.Registry, field reflect.StructField, parentType reflect.Type, operationID string) {
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
	hint := getHint(parentType, field.Name, operationID+"Response")
	if nameHint := field.Tag.Get("nameHint"); nameHint != "" {
		hint = nameHint
	}
	schema := huma.SchemaFromField(registry, field, hint)

	// Add to response content
	response.Content[contentType] = &huma.MediaType{
		Schema: schema,
	}
}

// processOutputHeaderField processes header fields in output type
func processOutputHeaderField(response *huma.Response, registry huma.Registry, field reflect.StructField, headerName string, parentType reflect.Type, operationID string) {
	// Generate schema for header
	hint := getHint(parentType, field.Name, operationID+fmt.Sprintf("%d", http.StatusOK)+headerName)
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
}

// getHeaderName extracts header name from struct field
func getHeaderName(field reflect.StructField) string {
	if header := field.Tag.Get("header"); header != "" {
		return header
	}
	// Default to field name for headers
	if field.Name != "Body" && field.Name != "Status" {
		return field.Name
	}
	return ""
}

// setupErrorResponses sets up standard error responses
func setupErrorResponses(operation *huma.Operation, registry huma.Registry) {
	// Create example error for schema
	exampleErr := huma.NewError(0, "")
	errContentType := "application/json"
	if ctf, ok := exampleErr.(huma.ContentTypeFilter); ok {
		errContentType = ctf.ContentType(errContentType)
	}

	errType := deref(reflect.TypeOf(exampleErr))
	errSchema := registry.Schema(errType, true, getHint(errType, "", "Error"))

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
}

// Helper functions
func getHint(parent reflect.Type, name string, other string) string {
	if parent.Name() != "" {
		return parent.Name() + name
	} else {
		return other
	}
}

func deref(t reflect.Type) reflect.Type {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	return t
}

func jsonTagValue(registry huma.Registry, typeName string, schema *huma.Schema, value string) interface{} {
	// Simple implementation - just return the string value
	// Huma has a more complex implementation that converts based on schema type
	return value
}

// buildDependencyMappingFromHandler creates dependency mapping by analyzing handler signature with enhanced error reporting
func buildDependencyMappingFromHandler(procedure *Procedure, operationID string, handlerType reflect.Type, relativeFile string, callerLine int) map[reflect.Type]*Dependency {
	// Build set of types that the handler actually needs
	requiredTypes := make(map[reflect.Type]bool)
	for i := 2; i < handlerType.NumIn(); i++ {
		paramType := handlerType.In(i)
		requiredTypes[paramType] = true
	}

	// Create type-to-dependency mapping and check for conflicts
	depsByType := make(map[reflect.Type]*Dependency)
	unusedDeps := make([]*Dependency, 0)

	for i := range procedure.deps {
		dep := &procedure.deps[i]
		depType := dep.Type()

		if existing, exists := depsByType[depType]; exists {
			FormatDuplicateDependenciesError(operationID, relativeFile, callerLine, DuplicateDependencies{
				ConflictingType: depType,
				ExistingDep:     existing,
				NewDep:          dep,
			})
			panic(fmt.Sprintf("duplicate dependency types for operation '%s' - see error details above", operationID))
		}

		// Check if this dependency is actually needed
		if requiredTypes[depType] {
			depsByType[depType] = dep
		} else {
			unusedDeps = append(unusedDeps, dep)
		}
	}

	// Log warnings for unused dependencies with line information
	if len(unusedDeps) > 0 {
		FormatUnusedDependenciesWarning(operationID, relativeFile, callerLine, unusedDeps)
	}

	// Validate that all handler parameters have corresponding dependencies
	missingDeps := make([]reflect.Type, 0)
	for i := 2; i < handlerType.NumIn(); i++ {
		paramType := handlerType.In(i)
		if _, exists := depsByType[paramType]; !exists {
			missingDeps = append(missingDeps, paramType)
		}
	}

	if len(missingDeps) > 0 {
		FormatMissingDependenciesError(operationID, relativeFile, callerLine, MissingDependencies{
			MissingTypes:  missingDeps,
			AvailableDeps: depsByType,
		})
		panic(fmt.Sprintf("missing dependencies for operation '%s' - see error details above", operationID))
	}

	return depsByType
}

// applyMiddlewaresAndSecurity applies middlewares and security to the operation
func applyMiddlewaresAndSecurity(operation *huma.Operation, procedure *Procedure) {
	// Add middlewares
	for _, middleware := range procedure.middlewares {
		operation.Middlewares = append(operation.Middlewares, middleware)
	}

	// Apply security
	if len(procedure.security) > 0 && len(operation.Security) == 0 {
		operation.Security = procedure.security
	}
}

// Register is the simple version without DI (maintains backward compatibility)
func Register[I, O any](api huma.API, operation huma.Operation, handler func(context.Context, *I) (*O, error)) {
	huma.Register(api, operation, handler)
}

// PublicProcedure is for public endpoints (no auth required)
func PublicProcedure(deps ...Dependency) *Procedure {
	return NewProcedure().Inject(deps...)
}

// AuthenticatedProcedure is pre-configured with auth middleware
// Note: AuthMiddleware would need to be imported from procedures package
func AuthenticatedProcedure(procedure *Procedure, authMiddleware Middleware, security map[string][]string) *Procedure {
	return procedure.Use(authMiddleware).WithSecurity(security)
}

// AdminProcedure is pre-configured with auth + admin role check
func AdminProcedure(authProcedure *Procedure, adminMiddleware Middleware) *Procedure {
	return authProcedure.Use(adminMiddleware)
}

// processDependencyInputFields processes input fields from dependencies that have InputFields defined
func processDependencyInputFields(operation *huma.Operation, registry huma.Registry, procedure *Procedure) {
	// Iterate over each dependency in the procedure
	for _, dep := range procedure.deps {
		// Check if the dependency has InputFields defined
		if dep.InputFields != nil {
			// Process each field in the dependency's InputFields
			for i := 0; i < dep.InputFields.NumField(); i++ {
				field := dep.InputFields.Field(i)
				if !field.IsExported() {
					continue
				}

				// Check for parameter tags (path, query, header, cookie)
				if processFieldAsParameter(operation, registry, field) {
					continue
				}

				// Check for body field
				if field.Name == "Body" {
					processFieldAsRequestBody(operation, registry, field, dep.InputFields)
				}
			}
		}
	}
}
