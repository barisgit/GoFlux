package goflux

import (
	"context"
	"fmt"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"

	"github.com/danielgtaylor/huma/v2"
)

// Dependency represents something that can be injected
type Dependency struct {
	Name   string
	LoadFn func(ctx context.Context, input interface{}) (interface{}, error)
	TypeFn func() reflect.Type
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

// RegisterWithDI provides registration with dependency injection
// Handler signature: func(ctx context.Context, input *I, ...deps) (*O, error)
// Dependencies are matched by type - if multiple deps have same type, registration fails
func RegisterWithDI[I, O any](
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

	// Validate handler signature at registration time
	handlerType := reflect.TypeOf(handler)
	if handlerType.Kind() != reflect.Func {
		panic(fmt.Sprintf("handler must be a function, got %T", handler))
	}

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
			FormatDuplicateDependenciesError(operation.OperationID, relativeFile, callerLine, DuplicateDependencies{
				ConflictingType: depType,
				ExistingDep:     existing,
				NewDep:          dep,
			})
			panic(fmt.Sprintf("duplicate dependency types for operation '%s' - see error details above", operation.OperationID))
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
		FormatUnusedDependenciesWarning(operation.OperationID, relativeFile, callerLine, unusedDeps)
	}

	// Validate that all handler parameters (except ctx and input) have corresponding dependencies
	missingDeps := make([]reflect.Type, 0)
	for i := 2; i < handlerType.NumIn(); i++ {
		paramType := handlerType.In(i)
		if _, exists := depsByType[paramType]; !exists {
			missingDeps = append(missingDeps, paramType)
		}
	}

	// If there are missing dependencies, log a nice error before panicking
	if len(missingDeps) > 0 {
		FormatMissingDependenciesError(operation.OperationID, relativeFile, callerLine, MissingDependencies{
			MissingTypes:  missingDeps,
			AvailableDeps: depsByType,
		})
		panic(fmt.Sprintf("missing dependencies for operation '%s' - see error details above", operation.OperationID))
	}

	// Create wrapper that handles middleware + DI
	wrappedHandler := func(ctx context.Context, input *I) (*O, error) {
		// Resolve only the dependencies that are actually used
		resolvedDeps := make(map[reflect.Type]interface{})
		for depType, dep := range depsByType {
			resolved, err := dep.Load(ctx, input)
			if err != nil {
				return nil, err
			}
			resolvedDeps[depType] = resolved
		}

		// Call handler with individual dependency parameters
		handlerValue := reflect.ValueOf(handler)
		handlerType := handlerValue.Type()

		// Build args with ctx and input first
		args := []reflect.Value{
			reflect.ValueOf(ctx),
			reflect.ValueOf(input),
		}

		// Match dependencies by exact type
		for i := 2; i < handlerType.NumIn(); i++ {
			paramType := handlerType.In(i)

			if resolved, exists := resolvedDeps[paramType]; exists {
				args = append(args, reflect.ValueOf(resolved))
			} else {
				return nil, fmt.Errorf("no dependency found for parameter %d of type %v", i-2, paramType)
			}
		}

		results := handlerValue.Call(args)

		// Extract result and error
		if len(results) >= 2 {
			var err error
			if !results[1].IsNil() {
				err = results[1].Interface().(error)
			}
			return results[0].Interface().(*O), err
		}

		return results[0].Interface().(*O), nil
	}

	// Add Huma middlewares to operation
	for _, middleware := range procedure.middlewares {
		operation.Middlewares = append(operation.Middlewares, middleware)
	}

	// Apply security requirements from procedure to operation (only if operation doesn't already have security)
	if len(procedure.security) > 0 && len(operation.Security) == 0 {
		operation.Security = procedure.security
	}

	// Register with Huma
	huma.Register(api, operation, wrappedHandler)
}

// Helper function to get available types for error messages
func getAvailableTypes(depsByType map[reflect.Type]*Dependency) []reflect.Type {
	types := make([]reflect.Type, 0, len(depsByType))
	for t := range depsByType {
		types = append(types, t)
	}
	return types
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
