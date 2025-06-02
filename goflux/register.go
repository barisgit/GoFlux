package goflux

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"reflect"

	"github.com/barisgit/goflux/internal/core"
	"github.com/barisgit/goflux/internal/openapi"
	"github.com/barisgit/goflux/internal/parsing"
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
