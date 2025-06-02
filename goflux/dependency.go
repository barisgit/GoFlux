package goflux

import (
	"context"
	"reflect"

	"github.com/barisgit/goflux/internal/core"
	"github.com/danielgtaylor/huma/v2"
)

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
