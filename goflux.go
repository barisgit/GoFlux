// Package goflux provides the GoFlux framework for building full-stack Go applications.
// This allows users to import: github.com/barisgit/goflux
package goflux

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strconv"

	"github.com/barisgit/goflux/internal/core"
	"github.com/barisgit/goflux/internal/features"
	"github.com/barisgit/goflux/internal/openapi"
	"github.com/barisgit/goflux/internal/parsing"
	"github.com/barisgit/goflux/internal/static"
	"github.com/barisgit/goflux/internal/upload"
	openapiutils "github.com/barisgit/goflux/openapi"

	"github.com/danielgtaylor/huma/v2"
	"github.com/spf13/cobra"
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

// MissingDependencies contains details about missing dependencies
type MissingDependencies struct {
	MissingTypes  []reflect.Type
	AvailableDeps map[reflect.Type]*Dependency
}

// FormatMissingDependenciesError formats and logs a missing dependencies error
func FormatMissingDependenciesError(operation, file string, line int, details MissingDependencies) {
	fmt.Printf("\x1b[31mERROR: \x1b[0m\x1b[1m%d missing dependencies in operation '\x1b[38;5;39m%s\x1b[0m\x1b[1m':\x1b[0m\n",
		len(details.MissingTypes), operation)
	fmt.Printf("\x1b[38;5;45m   Location: \x1b[38;5;255m%s:%d\x1b[0m\n", file, line)

	fmt.Printf("\x1b[31m   Missing dependencies:\x1b[0m\n")
	for i, missingType := range details.MissingTypes {
		fmt.Printf("\x1b[38;5;203m   - Parameter %d: \x1b[38;5;201m%v\x1b[0m\n", i, missingType)
	}

	if len(details.AvailableDeps) > 0 {
		fmt.Printf("\x1b[33m   Available dependencies:\x1b[0m\n")
		for depType, dep := range details.AvailableDeps {
			fmt.Printf("\x1b[38;5;118m   - '\x1b[38;5;226m%s\x1b[38;5;118m' (type: \x1b[38;5;201m%v\x1b[38;5;118m)\x1b[0m\n", dep.Name(), depType)
		}
	} else {
		fmt.Printf("\x1b[33m   No dependencies are currently registered for this procedure.\x1b[0m\n")
	}

	fmt.Printf("\x1b[38;5;118m   Solutions:\x1b[0m\n")
	fmt.Printf("\x1b[38;5;118m   • Add the missing dependencies to your procedure using .Inject()\x1b[0m\n")
	fmt.Printf("\x1b[38;5;118m   • Remove the unused parameters from your handler function\x1b[0m\n")
	fmt.Printf("\x1b[38;5;118m   • Create wrapper types if you have type conflicts\x1b[0m\n")
	fmt.Println() // Add spacing before panic
}

// FormatUnusedDependenciesWarning formats and logs unused dependencies warning
func FormatUnusedDependenciesWarning(operation, file string, line int, unusedDeps []*Dependency) {
	fmt.Printf("\x1b[38;5;208mWARNING: \x1b[0m\x1b[1m%d unused dependencies in operation '\x1b[38;5;39m%s\x1b[0m\x1b[1m':\x1b[0m\n",
		len(unusedDeps), operation)
	fmt.Printf("\x1b[38;5;45m   Location: \x1b[38;5;255m%s:%d\x1b[0m\n", file, line)

	for _, dep := range unusedDeps {
		fmt.Printf("\x1b[38;5;203m   - '\x1b[38;5;226m%s\x1b[38;5;203m' (type: \x1b[38;5;201m%v\x1b[38;5;203m) - consider removing from procedure or use it as a dependency\x1b[0m\n",
			dep.Name(), dep.Type())
	}
	fmt.Printf("\x1b[38;5;118m   Tip: Remove unused dependencies to improve performance or use them as dependencies\x1b[0m\n")
	fmt.Println() // Add spacing after warnings
}

// Dependency represents something that can be injected
type Dependency struct {
	core *core.DependencyCore
}

// Load executes the dependency's load function
func (d *Dependency) Load(ctx context.Context, input interface{}) (interface{}, error) {
	return d.core.Load(ctx, input)
}

// Type returns the type this dependency provides
func (d *Dependency) Type() reflect.Type {
	return d.core.Type()
}

// Name returns the name of this dependency
func (d *Dependency) Name() string {
	return d.core.Name
}

// RequiresMiddleware adds middleware requirements to this dependency
// Dependencies can declare what middleware they need to function properly
// Example: CurrentUserDep.RequiresMiddleware(AuthMiddleware)
func (d Dependency) RequiresMiddleware(middleware ...Middleware) Dependency {
	middlewareFuncs := make([]core.MiddlewareFunc, len(middleware))
	for i, m := range middleware {
		middlewareFuncs[i] = m
	}

	return Dependency{
		core: d.core.RequiresMiddleware(middlewareFuncs...),
	}
}

// WithInputFields adds input field requirements to a dependency
// The inputExample should be a zero value of the input struct type
// Example: dep.WithInputFields(PaginationParams{})
func (d Dependency) WithInputFields(inputExample interface{}) Dependency {
	return Dependency{
		core: d.core.WithInputFields(inputExample),
	}
}

// NewDependency creates a new dependency with automatic type inference
// The loadFn must have signature: func(context.Context, interface{}) (T, error)
// where T is the type this dependency provides
func NewDependency(name string, loadFn interface{}) Dependency {
	return Dependency{
		core: core.NewDependencyCore(name, loadFn),
	}
}

// NewDependencyWithInput creates a dependency that requires additional input fields
// This is a convenience function that combines NewDependency and WithInputFields
// The inputExample should be a zero value of the input struct type
func NewDependencyWithInput(name string, inputExample interface{}, loadFn interface{}) Dependency {
	return NewDependency(name, loadFn).WithInputFields(inputExample)
}

// Middleware can modify context or halt execution (standard Huma signature with API available in context)
type Middleware func(ctx huma.Context, next func(huma.Context))

// Internal access for the core dependency (for internal packages only)
func (d *Dependency) getCore() *core.DependencyCore {
	return d.core
}

// Procedure represents a fluent builder for dependency injection
type Procedure struct {
	registry    *core.DependencyRegistry
	middlewares []Middleware
	security    []map[string][]string
	utils       core.MiddlewareUtils
}

// NewProcedure creates a new procedure builder
func NewProcedure() *Procedure {
	return &Procedure{
		registry:    core.NewDependencyRegistry(),
		middlewares: make([]Middleware, 0),
		security:    make([]map[string][]string, 0),
		utils:       core.MiddlewareUtils{},
	}
}

// InjectDeps adds dependencies to the procedure
func InjectDeps(deps ...Dependency) *Procedure {
	procedure := NewProcedure()
	return procedure.Inject(deps...)
}

// Use adds middleware to the procedure with automatic deduplication
// Duplicate middleware (identified by function pointer) are automatically filtered out
func (p *Procedure) Use(middleware ...Middleware) *Procedure {
	// Combine existing middleware with new ones, then deduplicate
	combined := append(p.middlewares, middleware...)

	// Convert to core middleware functions for deduplication
	coreMW := make([]core.MiddlewareFunc, len(combined))
	for i, mw := range combined {
		coreMW[i] = mw
	}

	// Deduplicate and convert back
	deduplicated := p.utils.DeduplicateMiddleware(coreMW)
	newMiddlewares := make([]Middleware, len(deduplicated))
	for i, mw := range deduplicated {
		newMiddlewares[i] = mw.(Middleware)
	}

	return &Procedure{
		registry:    p.registry,
		middlewares: newMiddlewares,
		security:    p.security,
		utils:       p.utils,
	}
}

// Inject adds additional dependencies with automatic middleware collection and deduplication
// Also collects any middleware required by the new dependencies
func (p *Procedure) Inject(deps ...Dependency) *Procedure {
	newRegistry := core.NewDependencyRegistry()

	// Copy existing dependencies
	for _, dep := range p.registry.GetAll() {
		newRegistry.Add(dep)
	}

	// Add new dependencies and collect middleware
	var allMiddleware []Middleware
	allMiddleware = append(allMiddleware, p.middlewares...)

	for _, dep := range deps {
		// Add dependency to registry
		if err := newRegistry.Add(dep.getCore()); err != nil {
			// Log warning but continue (duplicate dependencies)
			// In production, might want proper logging
		}

		// Collect middleware from this dependency
		for _, mw := range dep.getCore().RequiredMiddleware {
			if middleware, ok := mw.(Middleware); ok {
				allMiddleware = append(allMiddleware, middleware)
			}
		}
	}

	// Deduplicate middleware
	coreMW := make([]core.MiddlewareFunc, len(allMiddleware))
	for i, mw := range allMiddleware {
		coreMW[i] = mw
	}
	deduplicated := p.utils.DeduplicateMiddleware(coreMW)
	newMiddlewares := make([]Middleware, len(deduplicated))
	for i, mw := range deduplicated {
		newMiddlewares[i] = mw.(Middleware)
	}

	return &Procedure{
		registry:    newRegistry,
		middlewares: newMiddlewares,
		security:    p.security,
		utils:       p.utils,
	}
}

// WithSecurity adds security requirements to the procedure
func (p *Procedure) WithSecurity(security ...map[string][]string) *Procedure {
	return &Procedure{
		registry:    p.registry,
		middlewares: p.middlewares,
		security:    append(p.security, security...),
		utils:       p.utils,
	}
}

// Internal access methods for the register.go file
func (p *Procedure) getRegistry() *core.DependencyRegistry {
	return p.registry
}

func (p *Procedure) getMiddlewares() []Middleware {
	return p.middlewares
}

func (p *Procedure) getSecurity() []map[string][]string {
	return p.security
}

// Register method for Procedure - procedures can now register themselves!
func (p *Procedure) Register(
	api huma.API,
	operation huma.Operation,
	handler interface{},
) {
	// Find the actual user code location (skip framework code)
	location := core.FindUserCodeLocation()

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

	// Validate dependencies and build mapping
	validationResult, err := p.getRegistry().ValidateHandlerDependencies(handlerType)
	if err != nil {
		panic(fmt.Sprintf("Handler validation failed: %v", err))
	}

	// Report errors if any
	if len(validationResult.MissingTypes) > 0 {
		FormatMissingDependenciesError(operation.OperationID, location.File, location.Line, MissingDependencies{
			MissingTypes:  validationResult.MissingTypes,
			AvailableDeps: convertCoreDepsToPublic(validationResult.DepsByType),
		})
		panic(fmt.Sprintf("missing dependencies for operation '%s' - see error details above", operation.OperationID))
	}

	// Report warnings for unused dependencies
	if len(validationResult.UnusedDeps) > 0 {
		FormatUnusedDependenciesWarning(operation.OperationID, location.File, location.Line, convertCoreDepsListToPublic(validationResult.UnusedDeps))
	}

	// Apply middlewares and security to operation first
	applyMiddlewaresAndSecurity(&operation, p, api)

	// Process the operation using the schema processor
	schemaProcessor := openapi.NewSchemaProcessor()
	deps := make([]*core.DependencyCore, 0, len(validationResult.DepsByType))
	for _, dep := range validationResult.DepsByType {
		deps = append(deps, dep)
	}

	if err := schemaProcessor.ProcessOperation(&operation, api, inputType, outputType, deps); err != nil {
		panic(fmt.Sprintf("Failed to process operation schema: %v", err))
	}

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

		// Parse the input from the request using the request parser
		requestParser := parsing.NewRequestParser()
		if err := requestParser.ParseInput(api, ctx, inputPtr, inputType); err != nil {
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

			if dep, exists := validationResult.DepsByType[paramType]; exists {
				// Parse dependency-specific input if the dependency has InputFields
				var depInput interface{}
				if dep.InputFields != nil {
					// Create an instance of the dependency's input type
					depInputPtr := reflect.New(dep.InputFields)

					// Parse dependency input fields from the request using the request parser
					if err := requestParser.ParseInput(api, ctx, depInputPtr, dep.InputFields); err != nil {
						huma.WriteErr(api, ctx, http.StatusBadRequest, "Failed to parse dependency input", err)
						return
					}

					depInput = depInputPtr.Interface()
				} else {
					// No specific input fields, pass the main input
					depInput = inputPtr.Interface()
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
				// Use the response writer
				responseWriter := parsing.NewResponseWriter()
				if err := responseWriter.WriteOutput(api, ctx, output, outputType, operation); err != nil {
					// Log the error but don't write response since headers might be sent
					// In production, would use proper logging
				}
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
	for _, middleware := range procedure.getMiddlewares() {
		operation.Middlewares = append(operation.Middlewares, middleware)
	}

	// Apply security
	if len(procedure.getSecurity()) > 0 && len(operation.Security) == 0 {
		operation.Security = procedure.getSecurity()
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

// Helper functions for converting between public and internal types
func convertCoreDepsToPublic(coreDeps map[reflect.Type]*core.DependencyCore) map[reflect.Type]*Dependency {
	result := make(map[reflect.Type]*Dependency)
	for k, v := range coreDeps {
		result[k] = &Dependency{core: v}
	}
	return result
}

func convertCoreDepsListToPublic(coreDeps []*core.DependencyCore) []*Dependency {
	result := make([]*Dependency, len(coreDeps))
	for i, v := range coreDeps {
		result[i] = &Dependency{core: v}
	}
	return result
}

// ============================================================================
// TRPC-STYLE CONTEXT EXTENSIONS
// ============================================================================

// FluxContext extends huma.Context with convenience methods
type FluxContext struct {
	huma.Context
	api huma.API
}

// Wrap wraps a huma.Context to add GoFlux convenience methods
// Automatically retrieves the API from the context using GetAPI
func Wrap(ctx huma.Context) *FluxContext {
	return &FluxContext{Context: ctx, api: GetAPI(ctx)}
}

// WriteErr writes an error response with the given status and message
func (ctx *FluxContext) WriteErr(status int, message string, errors ...error) {
	huma.WriteErr(ctx.api, ctx.Context, status, message, errors...)
}

// WriteResponse writes a successful response with optional content type
func (ctx *FluxContext) WriteResponse(status int, body interface{}, contentType ...string) {
	ctx.SetStatus(status)

	var ct string
	if len(contentType) > 0 && contentType[0] != "" {
		ct = contentType[0]
		ctx.SetHeader("Content-Type", ct)
	} else {
		// Content negotiation
		var err error
		ct, err = ctx.api.Negotiate(ctx.Header("Accept"))
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
	tval, terr := ctx.api.Transform(ctx.Context, strconv.Itoa(status), body)
	if terr != nil {
		ctx.WriteErr(http.StatusInternalServerError, "error transforming response", terr)
		return
	}

	if err := ctx.api.Marshal(ctx.BodyWriter(), ct, tval); err != nil {
		ctx.WriteErr(http.StatusInternalServerError, "error marshaling response", err)
		return
	}
}

// ============================================================================
// RESPONSE WRITERS
// ============================================================================

// 1xx

// Continue writes a 100 Continue response
func (ctx *FluxContext) Continue() {
	ctx.SetStatus(http.StatusContinue)
}

// SwitchingProtocols writes a 101 Switching Protocols response
func (ctx *FluxContext) SwitchingProtocols() {
	ctx.SetStatus(http.StatusSwitchingProtocols)
}

// 2xx

// OK writes a 200 OK response
func (ctx *FluxContext) OK(body interface{}, contentType ...string) {
	ctx.WriteResponse(http.StatusOK, body, contentType...)
}

// Created writes a 201 Created response
func (ctx *FluxContext) Created(body interface{}, contentType ...string) {
	ctx.WriteResponse(http.StatusCreated, body, contentType...)
}

// Accepted writes a 202 Accepted response
func (ctx *FluxContext) Accepted(body interface{}, contentType ...string) {
	ctx.WriteResponse(http.StatusAccepted, body, contentType...)
}

// NoContent writes a 204 No Content response
func (ctx *FluxContext) NoContent() {
	ctx.SetStatus(http.StatusNoContent)
}

// 3xx

// MovedPermanently writes a 301 Moved Permanently response
func (ctx *FluxContext) MovedPermanently(location string) {
	ctx.SetStatus(http.StatusMovedPermanently)
	ctx.SetHeader("Location", location)
}

// Found writes a 302 Found response
func (ctx *FluxContext) Found(location string) {
	ctx.SetStatus(http.StatusFound)
	ctx.SetHeader("Location", location)
}

// NotModified writes a 304 Not Modified response
func (ctx *FluxContext) NotModified() {
	ctx.SetStatus(http.StatusNotModified)
}

// For 4xx and 5xx, we can use error structs, that users can then pregenerate for common responses
type StatusError struct {
	Status  int
	Message string
}

// Error implements the error interface, returning the message
func (e *StatusError) Error() string {
	return e.Message
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
func (ctx *FluxContext) WriteBadRequestError(message string, errors ...error) {
	ctx.WriteErr(http.StatusBadRequest, message, errors...)
}

// NewUnauthorizedError writes a 401 Unauthorized response
func (ctx *FluxContext) WriteUnauthorizedError(message string, errors ...error) {
	ctx.WriteErr(http.StatusUnauthorized, message, errors...)
}

// NewPaymentRequiredError writes a 402 Payment Required response
func (ctx *FluxContext) WritePaymentRequiredError(message string, errors ...error) {
	ctx.WriteErr(http.StatusPaymentRequired, message, errors...)
}

// NewForbiddenError writes a 403 Forbidden response
func (ctx *FluxContext) WriteForbiddenError(message string, errors ...error) {
	ctx.WriteErr(http.StatusForbidden, message, errors...)
}

// NewNotFoundError writes a 404 Not Found response
func (ctx *FluxContext) WriteNotFoundError(message string, errors ...error) {
	ctx.WriteErr(http.StatusNotFound, message, errors...)
}

// NewMethodNotAllowedError writes a 405 Method Not Allowed response
func (ctx *FluxContext) WriteMethodNotAllowedError(message string, errors ...error) {
	ctx.WriteErr(http.StatusMethodNotAllowed, message, errors...)
}

// NewConflictError writes a 409 Conflict response
func (ctx *FluxContext) WriteConflictError(message string, errors ...error) {
	ctx.WriteErr(http.StatusConflict, message, errors...)
}

// NewTooManyRequestsError writes a 429 Too Many Requests response
func (ctx *FluxContext) WriteTooManyRequestsError(message string, errors ...error) {
	ctx.WriteErr(http.StatusTooManyRequests, message, errors...)
}

// 5xx

// NewInternalServerError writes a 500 Internal Server Error response
func (ctx *FluxContext) WriteInternalServerError(message string, errors ...error) {
	ctx.WriteErr(http.StatusInternalServerError, message, errors...)
}

// NewNotImplementedError writes a 501 Not Implemented response
func (ctx *FluxContext) WriteNotImplementedError(message string, errors ...error) {
	ctx.WriteErr(http.StatusNotImplemented, message, errors...)
}

// NewBadGatewayError writes a 502 Bad Gateway response
func (ctx *FluxContext) WriteBadGatewayError(message string, errors ...error) {
	ctx.WriteErr(http.StatusBadGateway, message, errors...)
}

// NewServiceUnavailableError writes a 503 Service Unavailable response
func (ctx *FluxContext) WriteServiceUnavailableError(message string, errors ...error) {
	ctx.WriteErr(http.StatusServiceUnavailable, message, errors...)
}

// AddOpenAPICommand adds an OpenAPI generation command to any cobra CLI
// This is a convenience function that wraps the dev package
func AddOpenAPICommand(rootCmd *cobra.Command, apiProvider func() huma.API) {
	openAPICmd := &cobra.Command{
		Use:   "openapi",
		Short: "Generate OpenAPI specification",
		Long:  "Generate OpenAPI specification from your Huma API without starting the server",
		RunE: func(cmd *cobra.Command, args []string) error {
			outputPath, _ := cmd.Flags().GetString("output")
			format, _ := cmd.Flags().GetString("format")

			api := apiProvider()
			if api == nil {
				return fmt.Errorf("failed to get API instance")
			}

			var err error
			var spec []byte

			switch format {
			case "yaml":
				spec, err = openapiutils.GenerateSpecYAML(api)
			case "json":
				spec, err = openapiutils.GenerateSpec(api)
			default:
				return fmt.Errorf("unsupported format: %s (use 'json' or 'yaml')", format)
			}

			if err != nil {
				return fmt.Errorf("failed to generate OpenAPI spec: %w", err)
			}

			if outputPath != "" {
				err = openapiutils.GenerateSpecToFile(api, outputPath)
				if err != nil {
					return err
				}
				fmt.Printf("✅ OpenAPI spec saved to %s\n", outputPath)
			} else {
				fmt.Print(string(spec))
			}

			// Print some stats
			routeCount := openapiutils.GetRouteCount(api)
			if routeCount > 0 {
				fmt.Printf("🛣️  Found %d API routes\n", routeCount)
			}

			return nil
		},
	}

	openAPICmd.Flags().StringP("output", "o", "", "Output file path (prints to stdout if not specified)")
	openAPICmd.Flags().StringP("format", "f", "json", "Output format (json or yaml)")

	rootCmd.AddCommand(openAPICmd)
}

// OpenAPI generation utilities - re-export from openapi package
var (
	GenerateSpecToFile = openapiutils.GenerateSpecToFile
	GenerateSpec       = openapiutils.GenerateSpec
	GenerateSpecYAML   = openapiutils.GenerateSpecYAML
	GetRouteCount      = openapiutils.GetRouteCount
)

// Re-export feature functions from internal/features
var (
	Greet             = features.Greet
	QuickGreet        = features.QuickGreet
	AddHealthCheck    = features.AddHealthCheck
	CustomHealthCheck = features.CustomHealthCheck
)

// Re-export feature types from internal/features
type (
	GreetOptions   = features.GreetOptions
	HealthResponse = features.HealthResponse
)

// Re-export upload functionality from internal/upload
var (
	NewFile               = upload.NewFile
	NewFileList           = upload.NewFileList
	NewFileUploadResponse = upload.NewFileUploadResponse
	NewFileUploadError    = upload.NewFileUploadError
	GetFileFromForm       = upload.GetFileFromForm
	GetFormValue          = upload.GetFormValue
)

// Re-export upload types from internal/upload
type (
	File                   = upload.File
	FileList               = upload.FileList
	FileUploadResponseBody = upload.FileUploadResponseBody
	FileInfo               = upload.FileInfo
	FileUploadError        = upload.FileUploadError
)

// Re-export upload errors from internal/upload
var (
	ErrNoFileUploaded     = upload.ErrNoFileUploaded
	ErrFileTooLarge       = upload.ErrFileTooLarge
	ErrInvalidFileType    = upload.ErrInvalidFileType
	ErrTooManyFiles       = upload.ErrTooManyFiles
	ErrInvalidFileContent = upload.ErrInvalidFileContent
)

// Re-export static functionality from internal/static
var (
	ServeStaticFile = static.ServeStaticFile
)

// Re-export static types from internal/static
type (
	StaticConfig   = static.StaticConfig
	StaticResponse = static.StaticResponse
)

// RegisterMultipartUpload creates a simple multipart file upload endpoint with minimal boilerplate
func RegisterMultipartUpload(api huma.API, path string, handler interface{}, options ...func(*huma.Operation)) {
	NewProcedure().RegisterMultipartUpload(api, path, handler, options...)
}

// RegisterMultipartUpload method for Procedure
func (p *Procedure) RegisterMultipartUpload(api huma.API, path string, handler interface{}, options ...func(*huma.Operation)) {
	// Now that our parsing system supports MultipartFormFiles, just use regular Post
	p.Post(api, path, handler, func(o *huma.Operation) {
		// Force multipart content type
		if o.RequestBody == nil {
			o.RequestBody = &huma.RequestBody{}
		}
		if o.RequestBody.Content == nil {
			o.RequestBody.Content = map[string]*huma.MediaType{}
		}
		if _, exists := o.RequestBody.Content["multipart/form-data"]; !exists {
			o.RequestBody.Content["multipart/form-data"] = &huma.MediaType{}
		}

		// Apply user options
		for _, opt := range options {
			opt(o)
		}
	})
}
