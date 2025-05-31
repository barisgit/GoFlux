package goflux

import (
	"context"
	"reflect"

	"github.com/danielgtaylor/huma/v2"
)

// Injectable represents a fluent builder for dependency injection
type Injectable struct {
	deps        []Dependency
	middlewares []Middleware
}

// NewInjectable creates a new injectable builder
func NewInjectable() *Injectable {
	return &Injectable{
		deps:        make([]Dependency, 0),
		middlewares: make([]Middleware, 0),
	}
}

// InjectDeps adds dependencies to the injectable (fluent interface)
func InjectDeps(deps ...Dependency) *Injectable {
	return &Injectable{
		deps:        deps,
		middlewares: make([]Middleware, 0),
	}
}

// Use adds middleware to the injectable (fluent interface)
func (inj *Injectable) Use(middleware Middleware) *Injectable {
	return &Injectable{
		deps:        inj.deps,
		middlewares: append(inj.middlewares, middleware),
	}
}

// InjectMore adds additional dependencies (fluent interface)
func (inj *Injectable) InjectMore(deps ...Dependency) *Injectable {
	return &Injectable{
		deps:        append(inj.deps, deps...),
		middlewares: inj.middlewares,
	}
}

// RegisterWithDI provides registration with dependency injection
// Usage: goflux.RegisterWithDI[Input, Output](api, operation, injectable, handler)
func RegisterWithDI[I, O any](
	api huma.API,
	operation huma.Operation,
	injectable *Injectable,
	handler interface{},
) {
	// Create wrapper that handles middleware + DI
	wrappedHandler := func(ctx context.Context, input *I) (*O, error) {
		// Run middlewares first
		processedCtx := ctx
		for _, middleware := range injectable.middlewares {
			var err error
			processedCtx, err = middleware(processedCtx)
			if err != nil {
				return nil, err
			}
		}

		// Resolve dependencies
		resolvedDeps := make([]reflect.Value, len(injectable.deps))
		for i, dep := range injectable.deps {
			resolved, err := dep(processedCtx)
			if err != nil {
				return nil, err
			}
			resolvedDeps[i] = reflect.ValueOf(resolved)
		}

		// Call handler with injected dependencies
		handlerValue := reflect.ValueOf(handler)
		args := []reflect.Value{
			reflect.ValueOf(processedCtx),
			reflect.ValueOf(input),
		}
		args = append(args, resolvedDeps...)

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

	// Register with Huma
	huma.Register(api, operation, wrappedHandler)
}

// Register is the simple version without DI (maintains backward compatibility)
func Register[I, O any](api huma.API, operation huma.Operation, handler func(context.Context, *I) (*O, error)) {
	huma.Register(api, operation, handler)
}

// Pre-configured procedure builders (like tRPC's protectedProcedure)

// PublicProcedure is for public endpoints (no auth required)
var PublicProcedure = NewInjectable()

// AuthenticatedProcedure is pre-configured with auth middleware
// Note: AuthMiddleware would need to be imported from procedures package
func AuthenticatedProcedure(authMiddleware Middleware) *Injectable {
	return NewInjectable().Use(authMiddleware)
}

// AdminProcedure is pre-configured with auth + admin role check
func AdminProcedure(authMiddleware, adminMiddleware Middleware) *Injectable {
	return NewInjectable().Use(authMiddleware).Use(adminMiddleware)
}
