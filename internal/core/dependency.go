package core

import (
	"context"
	"fmt"
	"reflect"
)

// DependencyCore contains the core dependency functionality
type DependencyCore struct {
	Name               string
	LoadFn             func(ctx context.Context, input interface{}) (interface{}, error)
	TypeFn             func() reflect.Type
	InputFields        reflect.Type
	RequiredMiddleware []MiddlewareFunc
}

// MiddlewareFunc represents middleware signature
type MiddlewareFunc interface{}

// Load executes the dependency's load function
func (d *DependencyCore) Load(ctx context.Context, input interface{}) (interface{}, error) {
	return d.LoadFn(ctx, input)
}

// Type returns the type this dependency provides
func (d *DependencyCore) Type() reflect.Type {
	return d.TypeFn()
}

// NewDependencyCore creates a new dependency with automatic type inference
func NewDependencyCore(name string, loadFn interface{}) *DependencyCore {
	fnType := reflect.TypeOf(loadFn)

	// Validate it's a function
	if fnType.Kind() != reflect.Func {
		panic(fmt.Sprintf("loadFn must be a function, got %T", loadFn))
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

	return &DependencyCore{
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
func (d *DependencyCore) WithInputFields(inputExample interface{}) *DependencyCore {
	inputType := reflect.TypeOf(inputExample)
	if inputType.Kind() == reflect.Ptr {
		inputType = inputType.Elem()
	}

	newDep := *d // Copy the dependency
	newDep.InputFields = inputType
	return &newDep
}

// RequiresMiddleware adds middleware requirements to this dependency
func (d *DependencyCore) RequiresMiddleware(middleware ...MiddlewareFunc) *DependencyCore {
	newDep := *d // Copy the dependency
	newDep.RequiredMiddleware = append(newDep.RequiredMiddleware, middleware...)
	return &newDep
}
