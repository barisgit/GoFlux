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

/*
Package goflux provides a beautiful dependency injection framework built on top of Huma.

# Basic Usage (Huma-Compatible)

Simple endpoints without dependencies use GoFlux exactly like Huma:

	// Auto-generates operation ID and summary
	goflux.Get(api, "/health", func(ctx context.Context, input *struct{}) (*HealthOutput, error) {
		return &HealthOutput{Body: HealthData{Status: "ok"}}, nil
	})

	// With custom operation details
	goflux.Post(api, "/users", handler, func(o *huma.Operation) {
		o.OperationID = "create-user"
		o.Summary = "Create a new user"
		o.Tags = []string{"users"}
	})

	// Full manual control
	goflux.Register(api, huma.Operation{
		OperationID: "get-health",
		Method:      http.MethodGet,
		Path:        "/health",
	}, handler)

# Dependency Injection

Create dependencies without generics - types are inferred automatically:

	// Create dependencies using reflection-based type inference
	dbDep := goflux.NewDependency("database", func(ctx context.Context, input interface{}) (*sql.DB, error) {
		return sql.Open("postgres", "connection-string")
	})

	userServiceDep := goflux.NewDependency("userService", func(ctx context.Context, input interface{}) (*UserService, error) {
		// Access database from context if needed, or create new instance
		return &UserService{}, nil
	})

	// Dependencies with additional input fields (e.g., pagination)
	type PaginationParams struct {
		Page     int `query:"page" minimum:"1" default:"1"`
		PageSize int `query:"page_size" minimum:"1" maximum:"100" default:"20"`
	}

	paginatedDep := goflux.NewDependency("pagination", func(ctx context.Context, input interface{}) (*PaginationService, error) {
		// The input will contain the parsed PaginationParams
		return &PaginationService{}, nil
	}).WithInputFields(PaginationParams{})

	// Or use the convenience function
	paginatedDep2 := goflux.NewDependencyWithInput("pagination", func(ctx context.Context, input interface{}) (*PaginationService, error) {
		return &PaginationService{}, nil
	}, PaginationParams{})

# Creating Procedures

	// Create a procedure with dependencies
	procedure := goflux.PublicProcedure(dbDep, userServiceDep)

	// Register endpoints with automatic dependency injection and operation generation
	procedure.Get(api, "/users", func(ctx context.Context, input *GetUsersInput, db *sql.DB, userSvc *UserService) (*GetUsersOutput, error) {
		// db and userSvc are automatically injected
		// operation ID and summary auto-generated
		users, err := getUsersFromDB(db, input.Page, input.PageSize)
		return &GetUsersOutput{Body: users}, err
	})

	// With custom operation details
	procedure.Post(api, "/users", handler, func(o *huma.Operation) {
		o.Tags = []string{"users", "admin"}
		o.Summary = "Create user with advanced validation"
	})

# Middleware

Middleware uses standard Huma signatures with optional tRPC-style context extensions:

	// Traditional approach - clean and simple
	func AuthMiddleware(ctx huma.Context, next func(huma.Context)) {
		token := ctx.Header("Authorization")
		if token == "" {
			goflux.WriteErr(ctx, 401, "Authentication required")
			return
		}
		next(ctx)
	}

	// tRPC-style approach - wrap context for method calls
	func AuthMiddleware(ctx huma.Context, next func(huma.Context)) {
		fluxCtx := goflux.Wrap(ctx) // Add convenience methods

		token := ctx.Header("Authorization")
		if token == "" {
			fluxCtx.Unauthorized("Authentication required")
			return
		}

		if !isValidToken(token) {
			fluxCtx.Forbidden("Invalid token")
			return
		}

		next(ctx)
	}

	// Available methods on FluxContext:
	// Error responses: WriteErr, BadRequest, Unauthorized, Forbidden, NotFound, InternalServerError
	// Success responses: OK, Created, NoContent, WriteResponse
	// Example: fluxCtx.BadRequest("Invalid input", err)

	// Use middleware in procedures
	authProcedure := goflux.PublicProcedure(dbDep).Use(AuthMiddleware)

# Advanced Procedures

	// Authenticated procedure with middleware and security
	authProcedure := goflux.AuthenticatedProcedure(
		goflux.PublicProcedure(dbDep, userServiceDep),
		authMiddleware,
		map[string][]string{"bearer": {}},
	)

	authProcedure.Post(api, "/admin/users", func(ctx context.Context, input *CreateUserInput, db *sql.DB, userSvc *UserService) (*CreateUserOutput, error) {
		// Both db and userSvc are automatically injected
		// authMiddleware runs automatically
		// operation ID/summary auto-generated
		user, err := userSvc.CreateUser(ctx, db, input.Body)
		return &CreateUserOutput{Body: user}, err
	}, func(o *huma.Operation) {
		o.Tags = []string{"admin"}
		o.Description = "Admin-only user creation endpoint"
	})

# Migration from Huma

GoFlux is designed to be a drop-in replacement for Huma:

	// Before (Huma)
	huma.Get(api, "/users", handler)

	// After (GoFlux - exactly the same!)
	goflux.Get(api, "/users", handler)

	// With dependencies (GoFlux advantage)
	procedure := goflux.PublicProcedure(dbDep)
	procedure.Get(api, "/users", handlerWithDB)

*/

// Dependency represents something that can be injected
type Dependency struct {
	Name               string
	LoadFn             func(ctx context.Context, input interface{}) (interface{}, error)
	TypeFn             func() reflect.Type
	InputFields        reflect.Type // Optional: additional input fields this dependency needs
	RequiredMiddleware []Middleware // Middleware required for this dependency to function
}

// Load executes the dependency's load function
func (d *Dependency) Load(ctx context.Context, input interface{}) (interface{}, error) {
	return d.LoadFn(ctx, input)
}

// Type returns the type this dependency provides
func (d *Dependency) Type() reflect.Type {
	return d.TypeFn()
}

// RequiresMiddleware adds middleware requirements to this dependency
// Dependencies can declare what middleware they need to function properly
// Example: CurrentUserDep.RequiresMiddleware(AuthMiddleware)
func (d Dependency) RequiresMiddleware(middleware ...Middleware) Dependency {
	newDep := d // Copy the dependency
	newDep.RequiredMiddleware = append(newDep.RequiredMiddleware, middleware...)
	return newDep
}

// NewDependency creates a new dependency with automatic type inference
// The loadFn must have signature: func(context.Context, interface{}) (T, error)
// where T is the type this dependency provides
// TODO: consider removing name parameter, as it is only used for error messages
func NewDependency(name string, loadFn interface{}) Dependency {
	fnType := reflect.TypeOf(loadFn)

	// Validate it's a function
	if fnType.Kind() != reflect.Func {
		panic("loadFn must be a function")
	}

	// Validate signature: func(context.Context, interface{}) (T, error)
	if fnType.NumIn() != 2 || fnType.NumOut() != 2 {
		panic("loadFn must have signature func(context.Context, interface{}) (T, error)")
	}

	// Check first parameter is context.Context
	contextType := reflect.TypeOf((*context.Context)(nil)).Elem()
	if !fnType.In(0).Implements(contextType) {
		panic("first parameter must be context.Context")
	}

	// Check second parameter is interface{}
	if fnType.In(1) != reflect.TypeOf((*interface{})(nil)).Elem() {
		panic("second parameter must be interface{}")
	}

	// Check last return is error
	errorType := reflect.TypeOf((*error)(nil)).Elem()
	if !fnType.Out(1).Implements(errorType) {
		panic("last return value must be error")
	}

	returnType := fnType.Out(0) // The T type
	fnValue := reflect.ValueOf(loadFn)

	return Dependency{
		Name: name,
		LoadFn: func(ctx context.Context, input interface{}) (interface{}, error) {
			// Call the function using reflection
			args := []reflect.Value{
				reflect.ValueOf(ctx),
			}

			// Handle nil input properly for reflection
			if input == nil {
				// Create a zero value of interface{} type for nil input
				args = append(args, reflect.Zero(reflect.TypeOf((*interface{})(nil)).Elem()))
			} else {
				args = append(args, reflect.ValueOf(input))
			}

			results := fnValue.Call(args)

			// Extract return values
			result := results[0].Interface()
			errInterface := results[1].Interface()

			if errInterface != nil {
				return nil, errInterface.(error)
			}

			return result, nil
		},
		TypeFn: func() reflect.Type {
			return returnType
		},
		InputFields: nil, // No additional input fields by default
	}
}

// WithInputFields adds input field requirements to a dependency
// The inputExample should be a zero value of the input struct type
// Example: dep.WithInputFields(PaginationParams{})
func (d Dependency) WithInputFields(inputExample interface{}) Dependency {
	inputType := reflect.TypeOf(inputExample)
	if inputType.Kind() == reflect.Ptr {
		inputType = inputType.Elem()
	}

	newDep := d // Copy the dependency
	newDep.InputFields = inputType
	return newDep
}

// NewDependencyWithInput creates a dependency that requires additional input fields
// This is a convenience function that combines NewDependency and WithInputFields
// The inputExample should be a zero value of the input struct type
func NewDependencyWithInput(name string, inputExample interface{}, loadFn interface{}) Dependency {
	return NewDependency(name, loadFn).WithInputFields(inputExample)
}

// Middleware can modify context or halt execution (standard Huma signature with API available in context)
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

// Use adds middleware to the procedure with automatic deduplication
// Duplicate middleware (identified by function pointer) are automatically filtered out
func (p *Procedure) Use(middleware ...Middleware) *Procedure {
	// Combine existing middleware with new ones, then deduplicate
	combined := append(p.middlewares, middleware...)
	return &Procedure{
		deps:        p.deps,
		middlewares: deduplicateMiddleware(combined),
		security:    p.security,
	}
}

// Inject adds additional dependencies with automatic middleware collection and deduplication
// Also collects any middleware required by the new dependencies
func (p *Procedure) Inject(deps ...Dependency) *Procedure {
	// Collect all middleware from dependencies
	var allMiddleware []Middleware
	allMiddleware = append(allMiddleware, p.middlewares...)

	// Add middleware required by existing dependencies
	for _, dep := range p.deps {
		allMiddleware = append(allMiddleware, dep.RequiredMiddleware...)
	}

	// Add middleware required by new dependencies
	for _, dep := range deps {
		allMiddleware = append(allMiddleware, dep.RequiredMiddleware...)
	}

	return &Procedure{
		deps:        append(p.deps, deps...),
		middlewares: deduplicateMiddleware(allMiddleware),
		security:    p.security,
	}
}

// getMiddlewarePointer returns the function pointer value for middleware deduplication
func getMiddlewarePointer(middleware Middleware) uintptr {
	return reflect.ValueOf(middleware).Pointer()
}

// deduplicateMiddleware removes duplicate middleware while preserving order
// Uses function pointers to identify unique middleware functions
func deduplicateMiddleware(middlewares []Middleware) []Middleware {
	seen := make(map[uintptr]bool)
	var result []Middleware

	for _, middleware := range middlewares {
		ptr := getMiddlewarePointer(middleware)
		if !seen[ptr] {
			seen[ptr] = true
			result = append(result, middleware)
		}
	}

	return result
}

// removeMiddleware removes specific middleware from a slice
func removeMiddleware(middlewares []Middleware, toRemove ...Middleware) []Middleware {
	removeSet := make(map[uintptr]bool)
	for _, middleware := range toRemove {
		removeSet[getMiddlewarePointer(middleware)] = true
	}

	var result []Middleware
	for _, middleware := range middlewares {
		if !removeSet[getMiddlewarePointer(middleware)] {
			result = append(result, middleware)
		}
	}

	return result
}

// WithSecurity adds security requirements to the procedure
func (p *Procedure) WithSecurity(security ...map[string][]string) *Procedure {
	return &Procedure{
		deps:        p.deps,
		middlewares: p.middlewares,
		security:    append(p.security, security...),
	}
}

// findUserCodeLocation walks up the call stack to find the first non-framework code location
// This ensures error messages point to the user's code, not the framework code
func findUserCodeLocation() (string, int) {
	for i := 1; i < 10; i++ { // Check up to 10 stack frames
		_, callerFile, callerLine, ok := runtime.Caller(i)
		if !ok {
			break
		}

		// Skip framework code - look for user code
		if !strings.Contains(callerFile, "/goflux/goflux/") &&
			!strings.Contains(callerFile, "\\goflux\\goflux\\") {
			// Extract relative path for better error reporting
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
			return relativeFile, callerLine
		}
	}

	// Fallback if we can't find user code
	_, callerFile, callerLine, _ := runtime.Caller(1)
	return filepath.Base(callerFile), callerLine
}

// Register method for Procedure - procedures can now register themselves!
func (p *Procedure) Register(
	api huma.API,
	operation huma.Operation,
	handler interface{},
) {
	// Find the actual user code location (skip framework code)
	relativeFile, callerLine := findUserCodeLocation()

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
	depsByType := buildDependencyMappingFromHandler(p, operation.OperationID, handlerType, relativeFile, callerLine)

	// Apply middlewares and security to operation first
	applyMiddlewaresAndSecurity(&operation, p, api)

	// Process the operation like huma.Register does - this is the key part we were missing!
	processOperationLikeHuma(&operation, api, inputType, outputType, p)

	// Create a dependency injection wrapper that will be registered as the actual handler
	diWrapper := func(ctx huma.Context) {
		// Check if response has already been written (for safety)
		defer func() {
			if r := recover(); r != nil {
				// Don't write error if response was already started
				if ctx.Status() == 0 {
					huma.WriteErr(api, ctx, http.StatusInternalServerError, "Internal server error", fmt.Errorf("%v", r))
				}
			}
		}()

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
				// Parse dependency-specific input if the dependency has InputFields
				var depInput interface{}
				if dep.InputFields != nil {
					// Create an instance of the dependency's input type
					depInputPtr := reflect.New(dep.InputFields)
					depInputValue := depInputPtr.Elem()

					// Parse dependency input fields from the request
					for j := 0; j < dep.InputFields.NumField(); j++ {
						field := dep.InputFields.Field(j)
						if !field.IsExported() {
							continue
						}

						fieldValue := depInputValue.Field(j)
						if err := parseInputFieldWithDefaults(ctx, field, fieldValue); err != nil {
							huma.WriteErr(api, ctx, http.StatusBadRequest, "Failed to parse dependency input", err)
							return
						}
					}

					depInput = depInputPtr.Interface()
				} else {
					// No specific input fields, pass the main input
					depInput = input
				}

				resolved, err := dep.Load(ctx.Context(), depInput)
				if err != nil {
					// Don't write error if response was already started
					if ctx.Status() == 0 {
						huma.WriteErr(api, ctx, http.StatusInternalServerError, "Failed to resolve dependency", err)
					}
					return
				}
				handlerArgs = append(handlerArgs, reflect.ValueOf(resolved))
			} else {
				err := fmt.Errorf("no dependency found for parameter %d of type %v", i-2, paramType)
				// Don't write error if response was already started
				if ctx.Status() == 0 {
					huma.WriteErr(api, ctx, http.StatusInternalServerError, "Missing dependency", err)
				}
				return
			}
		}

		// Call the original handler
		results := handlerValue.Call(handlerArgs)

		// Handle the response
		if len(results) != 2 {
			// Don't write error if response was already started
			if ctx.Status() == 0 {
				huma.WriteErr(api, ctx, http.StatusInternalServerError, "Handler returned wrong number of values", fmt.Errorf("expected 2, got %d", len(results)))
			}
			return
		}

		// Check for error (second return value)
		if !results[1].IsNil() {
			err := results[1].Interface().(error)
			// Don't write error if response was already started
			if ctx.Status() == 0 {
				// Handle different error types appropriately
				var se huma.StatusError
				if errors.As(err, &se) {
					huma.WriteErr(api, ctx, se.GetStatus(), se.Error())
				} else {
					huma.WriteErr(api, ctx, http.StatusInternalServerError, "Handler error", err)
				}
			}
			return
		}

		// Handle successful response (first return value)
		output := results[0].Interface()
		if output != nil {
			// Don't write response if error was already written
			if ctx.Status() == 0 {
				// Use Huma's exact response pipeline: Transform -> Marshal
				writeHumaOutput(api, ctx, output, outputType, operation)
			}
		} else {
			if ctx.Status() == 0 {
				ctx.SetStatus(operation.DefaultStatus)
			}
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
		if err := parseInputFieldWithDefaults(ctx, field, fieldValue); err != nil {
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

// parseInputFieldWithDefaults parses a single input field and applies defaults from struct tags
func parseInputFieldWithDefaults(ctx huma.Context, field reflect.StructField, fieldValue reflect.Value) error {
	var value string

	// Handle path parameters
	if pathParam := field.Tag.Get("path"); pathParam != "" {
		value = ctx.Param(pathParam)
	} else if queryParam := field.Tag.Get("query"); queryParam != "" {
		// Handle query parameters
		paramName := strings.Split(queryParam, ",")[0] // Handle "name,explode" format
		value = ctx.Query(paramName)
	} else if headerParam := field.Tag.Get("header"); headerParam != "" {
		// Handle header parameters
		value = ctx.Header(headerParam)
	} else if cookieParam := field.Tag.Get("cookie"); cookieParam != "" {
		// Handle cookie parameters
		cookieHeader := ctx.Header("Cookie")
		if cookieHeader != "" {
			// Parse cookies manually since huma.Context doesn't provide Cookie method
			cookies := parseCookies(cookieHeader)
			if cookieValue, exists := cookies[cookieParam]; exists {
				value = cookieValue
			}
		}
	} else {
		return nil // Not a parameter field
	}

	// If value is empty, check for default value in struct tag
	if value == "" {
		if defaultValue := field.Tag.Get("default"); defaultValue != "" {
			value = defaultValue
		}
	}

	// Set the field value using the resolved value (either from request or default)
	return setFieldValue(fieldValue, value, field.Type)
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
	// Don't write anything if response has already been written
	if ctx.Status() != 0 {
		return
	}

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
				// Only write error if no response started yet
				if ctx.Status() == 0 {
					huma.WriteErr(api, ctx, http.StatusNotAcceptable, "unable to marshal response", err)
				}
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
			// Only write error if no response started yet
			if ctx.Status() == 0 {
				huma.WriteErr(api, ctx, http.StatusInternalServerError, "error transforming response", terr)
			}
			return
		}

		ctx.SetStatus(status)

		// Marshal and write the response (like Huma does)
		if status != http.StatusNoContent && status != http.StatusNotModified {
			if merr := api.Marshal(ctx.BodyWriter(), ct, tval); merr != nil {
				// Don't try to write error if response has already started
				// Just log it or handle it silently since headers are already sent
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
func applyMiddlewaresAndSecurity(operation *huma.Operation, procedure *Procedure, api huma.API) {
	// Create API injection middleware that runs FIRST
	apiInjectionMiddleware := func(ctx huma.Context, next func(huma.Context)) {
		// Inject API into context before any other middleware runs
		ctx = huma.WithValue(ctx, gofluxAPIKey, api)
		next(ctx)
	}

	// Add API injection middleware first
	operation.Middlewares = append(operation.Middlewares, apiInjectionMiddleware)

	// Then add user middlewares - they can access API from context
	for _, middleware := range procedure.middlewares {
		operation.Middlewares = append(operation.Middlewares, middleware)
	}

	// Apply security
	if len(procedure.security) > 0 && len(operation.Security) == 0 {
		operation.Security = procedure.security
	}
}

// Register is the simple version without DI - just a clean wrapper around huma.Register
func Register[I, O any](api huma.API, operation huma.Operation, handler func(context.Context, *I) (*O, error)) {
	huma.Register(api, operation, handler)
}

// RegisterWithDI registers an operation with dependency injection using a procedure
// This is the recommended way to register operations with dependencies
func RegisterWithDI(
	api huma.API,
	operation huma.Operation,
	procedure *Procedure,
	handler interface{},
) {
	procedure.Register(api, operation, handler)
}

// PublicProcedure creates a procedure for public endpoints (no auth required)
// Takes any number of dependencies and returns a procedure ready for registration
func PublicProcedure(deps ...Dependency) *Procedure {
	return NewProcedure().Inject(deps...)
}

// AuthenticatedProcedure creates a procedure pre-configured with auth middleware
// Takes a base procedure (typically PublicProcedure), auth middleware, and security requirements
func AuthenticatedProcedure(baseProcedure *Procedure, authMiddleware Middleware, security map[string][]string) *Procedure {
	return baseProcedure.Use(authMiddleware).WithSecurity(security)
}

// AdminProcedure creates a procedure pre-configured with auth + admin role check
// Takes an authenticated procedure and additional admin middleware
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
			for i := range dep.InputFields.NumField() {
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

// convenience is the shared function for all HTTP method convenience functions
func (p *Procedure) convenience(api huma.API, method, path string, handler interface{}, operationHandlers ...func(o *huma.Operation)) {
	// Use reflection to get the handler's output type for ID generation
	handlerType := reflect.TypeOf(handler)
	if handlerType.Kind() != reflect.Func || handlerType.NumOut() < 1 {
		panic("handler must be a function that returns an output type")
	}

	outputType := handlerType.Out(0) // First return value (*OutputType)

	// Auto-generate operation ID and summary like Huma does
	opID := huma.GenerateOperationID(method, path, reflect.Zero(outputType).Interface())
	opSummary := huma.GenerateSummary(method, path, reflect.Zero(outputType).Interface())

	operation := huma.Operation{
		OperationID: opID,
		Summary:     opSummary,
		Method:      method,
		Path:        path,
		Metadata:    map[string]any{},
	}

	// Apply any custom operation handlers
	for _, oh := range operationHandlers {
		oh(&operation)
	}

	// Mark auto-generated fields (like Huma does)
	if operation.OperationID == opID {
		operation.Metadata["_goflux_convenience_id"] = opID
	}
	if operation.Summary == opSummary {
		operation.Metadata["_goflux_convenience_summary"] = opSummary
	}

	p.Register(api, operation, handler)
}

// HTTP method convenience functions for procedures (Huma-style)

// Get registers a GET endpoint with automatic operation ID/summary generation
func (p *Procedure) Get(api huma.API, path string, handler interface{}, operationHandlers ...func(o *huma.Operation)) {
	p.convenience(api, http.MethodGet, path, handler, operationHandlers...)
}

// Post registers a POST endpoint with automatic operation ID/summary generation
func (p *Procedure) Post(api huma.API, path string, handler interface{}, operationHandlers ...func(o *huma.Operation)) {
	p.convenience(api, http.MethodPost, path, handler, operationHandlers...)
}

// Put registers a PUT endpoint with automatic operation ID/summary generation
func (p *Procedure) Put(api huma.API, path string, handler interface{}, operationHandlers ...func(o *huma.Operation)) {
	p.convenience(api, http.MethodPut, path, handler, operationHandlers...)
}

// Patch registers a PATCH endpoint with automatic operation ID/summary generation
func (p *Procedure) Patch(api huma.API, path string, handler interface{}, operationHandlers ...func(o *huma.Operation)) {
	p.convenience(api, http.MethodPatch, path, handler, operationHandlers...)
}

// Delete registers a DELETE endpoint with automatic operation ID/summary generation
func (p *Procedure) Delete(api huma.API, path string, handler interface{}, operationHandlers ...func(o *huma.Operation)) {
	p.convenience(api, http.MethodDelete, path, handler, operationHandlers...)
}

// Head registers a HEAD endpoint with automatic operation ID/summary generation
func (p *Procedure) Head(api huma.API, path string, handler interface{}, operationHandlers ...func(o *huma.Operation)) {
	p.convenience(api, http.MethodHead, path, handler, operationHandlers...)
}

// Options registers an OPTIONS endpoint with automatic operation ID/summary generation
func (p *Procedure) Options(api huma.API, path string, handler interface{}, operationHandlers ...func(o *huma.Operation)) {
	p.convenience(api, http.MethodOptions, path, handler, operationHandlers...)
}

// Top-level convenience functions (Huma-compatible API)

// Get registers a GET endpoint using a public procedure (no dependencies)
// This matches Huma's API exactly for backward compatibility
func Get(api huma.API, path string, handler interface{}, operationHandlers ...func(o *huma.Operation)) {
	PublicProcedure().Get(api, path, handler, operationHandlers...)
}

// Post registers a POST endpoint using a public procedure (no dependencies)
// This matches Huma's API exactly for backward compatibility
func Post(api huma.API, path string, handler interface{}, operationHandlers ...func(o *huma.Operation)) {
	PublicProcedure().Post(api, path, handler, operationHandlers...)
}

// Put registers a PUT endpoint using a public procedure (no dependencies)
// This matches Huma's API exactly for backward compatibility
func Put(api huma.API, path string, handler interface{}, operationHandlers ...func(o *huma.Operation)) {
	PublicProcedure().Put(api, path, handler, operationHandlers...)
}

// Patch registers a PATCH endpoint using a public procedure (no dependencies)
// This matches Huma's API exactly for backward compatibility
func Patch(api huma.API, path string, handler interface{}, operationHandlers ...func(o *huma.Operation)) {
	PublicProcedure().Patch(api, path, handler, operationHandlers...)
}

// Delete registers a DELETE endpoint using a public procedure (no dependencies)
// This matches Huma's API exactly for backward compatibility
func Delete(api huma.API, path string, handler interface{}, operationHandlers ...func(o *huma.Operation)) {
	PublicProcedure().Delete(api, path, handler, operationHandlers...)
}

// Head registers a HEAD endpoint using a public procedure (no dependencies)
// This matches Huma's API exactly for backward compatibility
func Head(api huma.API, path string, handler interface{}, operationHandlers ...func(o *huma.Operation)) {
	PublicProcedure().Head(api, path, handler, operationHandlers...)
}

// Options registers an OPTIONS endpoint using a public procedure (no dependencies)
// This matches Huma's API exactly for backward compatibility
func Options(api huma.API, path string, handler interface{}, operationHandlers ...func(o *huma.Operation)) {
	PublicProcedure().Options(api, path, handler, operationHandlers...)
}

// GoFluxAPI is a special context key for storing the API instance
const gofluxAPIKey = "goflux-api-do-not-use-this-key"

// GetAPI retrieves the API instance from the context for use in middleware and dependencies
func GetAPI(ctx huma.Context) huma.API {
	if api, ok := ctx.Context().Value(gofluxAPIKey).(huma.API); ok {
		return api
	}
	panic("GoFlux API not found in context - this should not happen in properly configured handlers")
}

// WriteErr writes an error response with the given status and message
func WriteErr(ctx huma.Context, status int, message string, errors ...error) {
	api := GetAPI(ctx)
	huma.WriteErr(api, ctx, status, message, errors...)
}

// ============================================================================
// TRPC-STYLE CONTEXT EXTENSIONS
// ============================================================================

// FluxContext extends huma.Context with convenience methods
type FluxContext struct {
	huma.Context
}

// Wrap wraps a huma.Context to add GoFlux convenience methods
func Wrap(ctx huma.Context) *FluxContext {
	return &FluxContext{Context: ctx}
}

// WriteErr writes an error response with the given status and message
func (ctx *FluxContext) WriteErr(status int, message string, errors ...error) {
	WriteErr(ctx.Context, status, message, errors...)
}

// WriteResponse writes a successful response with optional content type
func (ctx *FluxContext) WriteResponse(status int, body interface{}, contentType ...string) {
	api := GetAPI(ctx.Context)

	ctx.SetStatus(status)

	var ct string
	if len(contentType) > 0 && contentType[0] != "" {
		ct = contentType[0]
		ctx.SetHeader("Content-Type", ct)
	} else {
		// Content negotiation
		var err error
		ct, err = api.Negotiate(ctx.Header("Accept"))
		if err != nil {
			ctx.WriteErr(http.StatusNotAcceptable, "unable to marshal response", err)
			return
		}
		ctx.SetHeader("Content-Type", ct)
	}

	// Handle byte slice special case
	if b, ok := body.([]byte); ok {
		ctx.BodyWriter().Write(b)
		return
	}

	// Transform and marshal using Huma's pipeline
	tval, terr := api.Transform(ctx.Context, strconv.Itoa(status), body)
	if terr != nil {
		ctx.WriteErr(http.StatusInternalServerError, "error transforming response", terr)
		return
	}

	if err := api.Marshal(ctx.BodyWriter(), ct, tval); err != nil {
		ctx.WriteErr(http.StatusInternalServerError, "error marshaling response", err)
		return
	}
}

// ============================================================================
// RESPONSE WRITERS
// ============================================================================

// 2xx

// OK writes a 200 OK response
func (ctx *FluxContext) OK(body interface{}, contentType ...string) {
	ctx.WriteResponse(http.StatusOK, body, contentType...)
}

// Created writes a 201 Created response
func (ctx *FluxContext) Created(body interface{}, contentType ...string) {
	ctx.WriteResponse(http.StatusCreated, body, contentType...)
}

// NoContent writes a 204 No Content response
func (ctx *FluxContext) NoContent() {
	ctx.SetStatus(http.StatusNoContent)
}

// 3xx

// NotModified writes a 304 Not Modified response
func (ctx *FluxContext) NotModified() {
	ctx.SetStatus(http.StatusNotModified)
}

// For 4xx and 5xx, we can use error structs, that users can then pregenerate for common responses
type StatusError struct {
	Status  int
	Message string
}

// NewStatusError creates a new StatusError with the given status, message, and errors
func NewStatusError(status int, message string, errors ...error) *StatusError {
	return &StatusError{
		Status:  status,
		Message: message,
	}
}

func (ctx *FluxContext) WriteStatusError(statusError *StatusError, errors ...error) {
	ctx.WriteErr(statusError.Status, statusError.Message, errors...)
}

// 4xx

// NewBadRequestError writes a 400 Bad Request response
func (ctx *FluxContext) NewBadRequestError(message string, errors ...error) {
	ctx.WriteErr(http.StatusBadRequest, message, errors...)
}

// NewUnauthorizedError writes a 401 Unauthorized response
func (ctx *FluxContext) NewUnauthorizedError(message string, errors ...error) {
	ctx.WriteErr(http.StatusUnauthorized, message, errors...)
}

// NewForbiddenError writes a 403 Forbidden response
func (ctx *FluxContext) NewForbiddenError(message string, errors ...error) {
	ctx.WriteErr(http.StatusForbidden, message, errors...)
}

// NewNotFoundError writes a 404 Not Found response
func (ctx *FluxContext) NewNotFoundError(message string, errors ...error) {
	ctx.WriteErr(http.StatusNotFound, message, errors...)
}

// NewConflictError writes a 409 Conflict response
func (ctx *FluxContext) NewConflictError(message string, errors ...error) {
	ctx.WriteErr(http.StatusConflict, message, errors...)
}

// 5xx

// NewInternalServerError writes a 500 Internal Server Error response
func (ctx *FluxContext) NewInternalServerError(message string, errors ...error) {
	ctx.WriteErr(http.StatusInternalServerError, message, errors...)
}
