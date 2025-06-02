package core

import (
	"fmt"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
)

// DependencyRegistry manages dependency mapping and validation
type DependencyRegistry struct {
	deps map[reflect.Type]*DependencyCore
}

// NewDependencyRegistry creates a new dependency registry
func NewDependencyRegistry() *DependencyRegistry {
	return &DependencyRegistry{
		deps: make(map[reflect.Type]*DependencyCore),
	}
}

// Add adds a dependency to the registry
func (r *DependencyRegistry) Add(dep *DependencyCore) error {
	depType := dep.Type()
	if existing, exists := r.deps[depType]; exists {
		return fmt.Errorf("duplicate dependency type %v: existing '%s', new '%s'",
			depType, existing.Name, dep.Name)
	}
	r.deps[depType] = dep
	return nil
}

// Get retrieves a dependency by type
func (r *DependencyRegistry) Get(t reflect.Type) (*DependencyCore, bool) {
	dep, exists := r.deps[t]
	return dep, exists
}

// GetAll returns all dependencies
func (r *DependencyRegistry) GetAll() map[reflect.Type]*DependencyCore {
	result := make(map[reflect.Type]*DependencyCore)
	for k, v := range r.deps {
		result[k] = v
	}
	return result
}

// ValidationResult contains dependency validation results
type ValidationResult struct {
	MissingTypes []reflect.Type
	UnusedDeps   []*DependencyCore
	DepsByType   map[reflect.Type]*DependencyCore
}

// ValidateHandlerDependencies validates handler dependencies against registry
func (r *DependencyRegistry) ValidateHandlerDependencies(handlerType reflect.Type) (*ValidationResult, error) {
	if handlerType.Kind() != reflect.Func {
		return nil, fmt.Errorf("handler must be a function, got %T", handlerType)
	}

	if handlerType.NumIn() < 2 {
		return nil, fmt.Errorf("handler must have at least 2 parameters: (context.Context, *InputType)")
	}

	// Build set of types that the handler actually needs
	requiredTypes := make(map[reflect.Type]bool)
	for i := 2; i < handlerType.NumIn(); i++ {
		paramType := handlerType.In(i)
		requiredTypes[paramType] = true
	}

	// Create type-to-dependency mapping and check for usage
	depsByType := make(map[reflect.Type]*DependencyCore)
	unusedDeps := make([]*DependencyCore, 0)

	for depType, dep := range r.deps {
		if requiredTypes[depType] {
			depsByType[depType] = dep
		} else {
			unusedDeps = append(unusedDeps, dep)
		}
	}

	// Check for missing dependencies
	missingDeps := make([]reflect.Type, 0)
	for i := 2; i < handlerType.NumIn(); i++ {
		paramType := handlerType.In(i)
		if _, exists := depsByType[paramType]; !exists {
			missingDeps = append(missingDeps, paramType)
		}
	}

	return &ValidationResult{
		MissingTypes: missingDeps,
		UnusedDeps:   unusedDeps,
		DepsByType:   depsByType,
	}, nil
}

// CodeLocation represents a code location for error reporting
type CodeLocation struct {
	File string
	Line int
}

// FindUserCodeLocation walks up the call stack to find the first non-framework code location
func FindUserCodeLocation() CodeLocation {
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
			return CodeLocation{File: relativeFile, Line: callerLine}
		}
	}

	// Fallback if we can't find user code
	_, callerFile, callerLine, _ := runtime.Caller(1)
	return CodeLocation{File: filepath.Base(callerFile), Line: callerLine}
}

// MiddlewareUtils provides middleware management utilities
type MiddlewareUtils struct{}

// GetMiddlewarePointer returns the function pointer value for middleware deduplication
func (MiddlewareUtils) GetMiddlewarePointer(middleware MiddlewareFunc) uintptr {
	return reflect.ValueOf(middleware).Pointer()
}

// DeduplicateMiddleware removes duplicate middleware while preserving order
func (m MiddlewareUtils) DeduplicateMiddleware(middlewares []MiddlewareFunc) []MiddlewareFunc {
	seen := make(map[uintptr]bool)
	var result []MiddlewareFunc

	for _, middleware := range middlewares {
		ptr := m.GetMiddlewarePointer(middleware)
		if !seen[ptr] {
			seen[ptr] = true
			result = append(result, middleware)
		}
	}

	return result
}

// RemoveMiddleware removes specific middleware from a slice
func (m MiddlewareUtils) RemoveMiddleware(middlewares []MiddlewareFunc, toRemove ...MiddlewareFunc) []MiddlewareFunc {
	removeSet := make(map[uintptr]bool)
	for _, middleware := range toRemove {
		removeSet[m.GetMiddlewarePointer(middleware)] = true
	}

	var result []MiddlewareFunc
	for _, middleware := range middlewares {
		if !removeSet[m.GetMiddlewarePointer(middleware)] {
			result = append(result, middleware)
		}
	}

	return result
}
