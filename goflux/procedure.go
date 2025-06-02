package goflux

import (
	"github.com/barisgit/goflux/internal/core"
)

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
