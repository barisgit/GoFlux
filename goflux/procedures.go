package goflux

import (
	"context"
	"reflect"
)

// Dependency represents something that can be injected
type Dependency func(ctx context.Context) (interface{}, error)

// Middleware can modify context or halt execution
type Middleware func(ctx context.Context) (context.Context, error)

// Procedure represents a chainable procedure builder (like tRPC)
type Procedure struct {
	middlewares []Middleware
	deps        []Dependency
}

// NewProcedure creates a new procedure builder
func NewProcedure() *Procedure {
	return &Procedure{
		middlewares: make([]Middleware, 0),
		deps:        make([]Dependency, 0),
	}
}

// Use adds middleware to the procedure (returns new procedure)
func (p *Procedure) Use(middleware Middleware) *Procedure {
	newProc := &Procedure{
		middlewares: append(p.middlewares, middleware),
		deps:        p.deps,
	}
	return newProc
}

// Inject adds dependencies to the procedure (returns new procedure)
func (p *Procedure) Inject(deps ...Dependency) *Procedure {
	newProc := &Procedure{
		middlewares: p.middlewares,
		deps:        append(p.deps, deps...),
	}
	return newProc
}

// ExecuteWithDI is a helper function that users can call from their huma handlers
// to execute logic with dependency injection and middleware
func (p *Procedure) ExecuteWithDI(ctx context.Context, input interface{}, handler interface{}) (interface{}, error) {
	// Run middlewares
	for _, middleware := range p.middlewares {
		var err error
		ctx, err = middleware(ctx)
		if err != nil {
			return nil, err
		}
	}

	// Resolve dependencies
	resolvedDeps := make([]reflect.Value, len(p.deps))
	for i, dep := range p.deps {
		resolved, err := dep(ctx)
		if err != nil {
			return nil, err
		}
		resolvedDeps[i] = reflect.ValueOf(resolved)
	}

	// Build call arguments: ctx, input, ...dependencies
	handlerValue := reflect.ValueOf(handler)
	args := []reflect.Value{
		reflect.ValueOf(ctx),
		reflect.ValueOf(input),
	}
	args = append(args, resolvedDeps...)

	// Call handler with injected dependencies
	results := handlerValue.Call(args)

	// Return result and error
	if len(results) >= 2 {
		var err error
		if !results[1].IsNil() {
			err = results[1].Interface().(error)
		}
		return results[0].Interface(), err
	}

	return results[0].Interface(), nil
}

// Convenience functions

// Inject creates a procedure with dependencies (skips step 1)
func Inject(deps ...Dependency) *Procedure {
	return NewProcedure().Inject(deps...)
}

// WithMiddleware creates a procedure with middleware (skips step 1)
func WithMiddleware(middlewares ...Middleware) *Procedure {
	proc := NewProcedure()
	for _, mw := range middlewares {
		proc = proc.Use(mw)
	}
	return proc
}
