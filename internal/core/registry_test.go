package core

import (
	"context"
	"reflect"
	"strings"
	"testing"
)

func TestNewDependencyRegistry(t *testing.T) {
	registry := NewDependencyRegistry()

	if registry == nil {
		t.Fatal("Expected registry to be created")
	}

	if registry.deps == nil {
		t.Error("Expected deps map to be initialized")
	}

	if len(registry.deps) != 0 {
		t.Error("Expected empty registry initially")
	}
}

func TestDependencyRegistryAdd(t *testing.T) {
	registry := NewDependencyRegistry()

	dep := NewDependencyCore("testService", func(ctx context.Context, input interface{}) (*TestService, error) {
		return &TestService{}, nil
	})

	err := registry.Add(dep)
	if err != nil {
		t.Errorf("Expected no error when adding dependency, got %v", err)
	}

	if len(registry.deps) != 1 {
		t.Errorf("Expected 1 dependency in registry, got %d", len(registry.deps))
	}

	depType := reflect.TypeOf(&TestService{})
	storedDep, exists := registry.deps[depType]
	if !exists {
		t.Error("Expected dependency to be stored in registry")
	}

	if storedDep != dep {
		t.Error("Expected stored dependency to be the same instance")
	}
}

func TestDependencyRegistryAddDuplicate(t *testing.T) {
	registry := NewDependencyRegistry()

	dep1 := NewDependencyCore("testService1", func(ctx context.Context, input interface{}) (*TestService, error) {
		return &TestService{Name: "first"}, nil
	})

	dep2 := NewDependencyCore("testService2", func(ctx context.Context, input interface{}) (*TestService, error) {
		return &TestService{Name: "second"}, nil
	})

	// Add first dependency
	err1 := registry.Add(dep1)
	if err1 != nil {
		t.Errorf("Expected no error for first dependency, got %v", err1)
	}

	// Try to add second dependency of same type
	err2 := registry.Add(dep2)
	if err2 == nil {
		t.Error("Expected error when adding duplicate dependency type")
	}

	expectedErrMsg := "duplicate dependency type"
	if !strings.Contains(err2.Error(), expectedErrMsg) {
		t.Errorf("Expected error message to contain '%s', got '%s'", expectedErrMsg, err2.Error())
	}

	// Should still only have one dependency
	if len(registry.deps) != 1 {
		t.Errorf("Expected 1 dependency in registry, got %d", len(registry.deps))
	}

	// And it should be the first one
	depType := reflect.TypeOf(&TestService{})
	storedDep, _ := registry.deps[depType]
	if storedDep.Name != "testService1" {
		t.Errorf("Expected first dependency to remain, got '%s'", storedDep.Name)
	}
}

func TestDependencyRegistryGet(t *testing.T) {
	registry := NewDependencyRegistry()

	dep := NewDependencyCore("testService", func(ctx context.Context, input interface{}) (*TestService, error) {
		return &TestService{}, nil
	})

	registry.Add(dep)

	// Test getting existing dependency
	depType := reflect.TypeOf(&TestService{})
	retrieved, exists := registry.Get(depType)

	if !exists {
		t.Error("Expected dependency to exist in registry")
	}

	if retrieved != dep {
		t.Error("Expected retrieved dependency to be the same instance")
	}

	// Test getting non-existing dependency
	nonExistentType := reflect.TypeOf(&TestDatabase{})
	_, exists2 := registry.Get(nonExistentType)

	if exists2 {
		t.Error("Expected non-existent dependency to return false")
	}
}

func TestDependencyRegistryGetAll(t *testing.T) {
	registry := NewDependencyRegistry()

	dep1 := NewDependencyCore("testService", func(ctx context.Context, input interface{}) (*TestService, error) {
		return &TestService{}, nil
	})

	dep2 := NewDependencyCore("testDatabase", func(ctx context.Context, input interface{}) (*TestDatabase, error) {
		return &TestDatabase{}, nil
	})

	registry.Add(dep1)
	registry.Add(dep2)

	all := registry.GetAll()

	if len(all) != 2 {
		t.Errorf("Expected 2 dependencies, got %d", len(all))
	}

	// Check that we got copies, not references to internal map
	serviceType := reflect.TypeOf(&TestService{})
	dbType := reflect.TypeOf(&TestDatabase{})

	if all[serviceType] != dep1 {
		t.Error("Expected service dependency to match")
	}

	if all[dbType] != dep2 {
		t.Error("Expected database dependency to match")
	}

	// Modify the returned map to ensure it's a copy
	delete(all, serviceType)
	if len(registry.deps) != 2 {
		t.Error("GetAll should return a copy, not a reference")
	}
}

func TestValidateHandlerDependencies(t *testing.T) {
	registry := NewDependencyRegistry()

	// Add some dependencies
	serviceDep := NewDependencyCore("testService", func(ctx context.Context, input interface{}) (*TestService, error) {
		return &TestService{}, nil
	})
	dbDep := NewDependencyCore("testDatabase", func(ctx context.Context, input interface{}) (*TestDatabase, error) {
		return &TestDatabase{}, nil
	})

	registry.Add(serviceDep)
	registry.Add(dbDep)

	tests := []struct {
		name            string
		handler         interface{}
		expectedMissing int
		expectedUnused  int
		expectedError   bool
		errorMessage    string
	}{
		{
			name: "valid handler with dependencies",
			handler: func(ctx context.Context, input *TestInputType, service *TestService, db *TestDatabase) (*TestService, error) {
				return service, nil
			},
			expectedMissing: 0,
			expectedUnused:  0,
			expectedError:   false,
		},
		{
			name: "handler with some dependencies",
			handler: func(ctx context.Context, input *TestInputType, service *TestService) (*TestService, error) {
				return service, nil
			},
			expectedMissing: 0,
			expectedUnused:  1, // database is unused
			expectedError:   false,
		},
		{
			name: "handler with missing dependency",
			handler: func(ctx context.Context, input *TestInputType, service *TestService, unknownType *string) (*TestService, error) {
				return service, nil
			},
			expectedMissing: 1, // *string is not registered
			expectedUnused:  1, // database is unused
			expectedError:   false,
		},
		{
			name: "handler with no dependencies",
			handler: func(ctx context.Context, input *TestInputType) (*TestService, error) {
				return &TestService{}, nil
			},
			expectedMissing: 0,
			expectedUnused:  2, // both service and database are unused
			expectedError:   false,
		},
		{
			name:          "not a function",
			handler:       "not a function",
			expectedError: true,
			errorMessage:  "handler must be a function",
		},
		{
			name: "function with too few parameters",
			handler: func(ctx context.Context) (*TestService, error) {
				return &TestService{}, nil
			},
			expectedError: true,
			errorMessage:  "handler must have at least 2 parameters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlerType := reflect.TypeOf(tt.handler)
			result, err := registry.ValidateHandlerDependencies(handlerType)

			if tt.expectedError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				if !strings.Contains(err.Error(), tt.errorMessage) {
					t.Errorf("Expected error message to contain '%s', got '%s'", tt.errorMessage, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("Expected no error, got %v", err)
				return
			}

			if result == nil {
				t.Fatal("Expected validation result")
			}

			if len(result.MissingTypes) != tt.expectedMissing {
				t.Errorf("Expected %d missing types, got %d", tt.expectedMissing, len(result.MissingTypes))
			}

			if len(result.UnusedDeps) != tt.expectedUnused {
				t.Errorf("Expected %d unused dependencies, got %d", tt.expectedUnused, len(result.UnusedDeps))
			}

			if len(result.DepsByType) != (2 - tt.expectedUnused) {
				t.Errorf("Expected %d dependencies by type, got %d", 2-tt.expectedUnused, len(result.DepsByType))
			}
		})
	}
}

func TestFindUserCodeLocation(t *testing.T) {
	location := FindUserCodeLocation()

	if location.File == "" {
		t.Error("Expected file to be set")
	}

	if location.Line == 0 {
		t.Error("Expected line to be set")
	}

	// Should contain the test file name
	if !strings.Contains(location.File, "registry_test.go") {
		t.Errorf("Expected file to contain 'registry_test.go', got '%s'", location.File)
	}
}

func TestMiddlewareUtils(t *testing.T) {
	utils := MiddlewareUtils{}

	middleware1 := func() {}
	middleware2 := func() {}

	// Test GetMiddlewarePointer
	ptr1 := utils.GetMiddlewarePointer(middleware1)
	ptr2 := utils.GetMiddlewarePointer(middleware2)
	ptr1_again := utils.GetMiddlewarePointer(middleware1)

	if ptr1 == 0 {
		t.Error("Expected non-zero pointer")
	}

	if ptr1 == ptr2 {
		t.Error("Different functions should have different pointers")
	}

	if ptr1 != ptr1_again {
		t.Error("Same function should have same pointer")
	}
}

func TestMiddlewareUtilsDeduplicateMiddleware(t *testing.T) {
	utils := MiddlewareUtils{}

	middleware1 := func() {}
	middleware2 := func() {}

	// Test deduplication
	middlewares := []MiddlewareFunc{middleware1, middleware2, middleware1, middleware2, middleware1}
	deduplicated := utils.DeduplicateMiddleware(middlewares)

	if len(deduplicated) != 2 {
		t.Errorf("Expected 2 unique middleware, got %d", len(deduplicated))
	}

	// Check that order is preserved (first occurrence)
	ptr1 := utils.GetMiddlewarePointer(deduplicated[0])
	ptr2 := utils.GetMiddlewarePointer(deduplicated[1])
	expectedPtr1 := utils.GetMiddlewarePointer(middleware1)
	expectedPtr2 := utils.GetMiddlewarePointer(middleware2)

	if ptr1 != expectedPtr1 || ptr2 != expectedPtr2 {
		t.Error("Expected order to be preserved")
	}
}

func TestMiddlewareUtilsRemoveMiddleware(t *testing.T) {
	utils := MiddlewareUtils{}

	middleware1 := func() {}
	middleware2 := func() {}
	middleware3 := func() {}

	middlewares := []MiddlewareFunc{middleware1, middleware2, middleware3}

	// Remove middleware2
	result := utils.RemoveMiddleware(middlewares, middleware2)

	if len(result) != 2 {
		t.Errorf("Expected 2 middleware after removal, got %d", len(result))
	}

	// Check that middleware1 and middleware3 remain
	ptr1 := utils.GetMiddlewarePointer(result[0])
	ptr3 := utils.GetMiddlewarePointer(result[1])
	expectedPtr1 := utils.GetMiddlewarePointer(middleware1)
	expectedPtr3 := utils.GetMiddlewarePointer(middleware3)

	if ptr1 != expectedPtr1 || ptr3 != expectedPtr3 {
		t.Error("Expected middleware1 and middleware3 to remain")
	}

	// Remove multiple middleware
	result2 := utils.RemoveMiddleware(middlewares, middleware1, middleware3)

	if len(result2) != 1 {
		t.Errorf("Expected 1 middleware after removing multiple, got %d", len(result2))
	}

	ptr2 := utils.GetMiddlewarePointer(result2[0])
	expectedPtr2 := utils.GetMiddlewarePointer(middleware2)

	if ptr2 != expectedPtr2 {
		t.Error("Expected middleware2 to remain")
	}
}

func TestMiddlewareUtilsRemoveNonExistentMiddleware(t *testing.T) {
	utils := MiddlewareUtils{}

	middleware1 := func() {}
	middleware2 := func() {}
	nonExistentMiddleware := func() {}

	middlewares := []MiddlewareFunc{middleware1, middleware2}

	// Try to remove non-existent middleware
	result := utils.RemoveMiddleware(middlewares, nonExistentMiddleware)

	if len(result) != 2 {
		t.Errorf("Expected 2 middleware when removing non-existent, got %d", len(result))
	}

	// Original middleware should remain
	ptr1 := utils.GetMiddlewarePointer(result[0])
	ptr2 := utils.GetMiddlewarePointer(result[1])
	expectedPtr1 := utils.GetMiddlewarePointer(middleware1)
	expectedPtr2 := utils.GetMiddlewarePointer(middleware2)

	if ptr1 != expectedPtr1 || ptr2 != expectedPtr2 {
		t.Error("Expected original middleware to remain unchanged")
	}
}

func TestValidationResultStructure(t *testing.T) {
	registry := NewDependencyRegistry()

	// Add a dependency
	serviceDep := NewDependencyCore("testService", func(ctx context.Context, input interface{}) (*TestService, error) {
		return &TestService{}, nil
	})
	registry.Add(serviceDep)

	// Handler that uses the dependency plus has a missing one
	handler := func(ctx context.Context, input *TestInputType, service *TestService, missing *TestDatabase) (*TestService, error) {
		return service, nil
	}

	handlerType := reflect.TypeOf(handler)
	result, err := registry.ValidateHandlerDependencies(handlerType)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Check MissingTypes
	if len(result.MissingTypes) != 1 {
		t.Errorf("Expected 1 missing type, got %d", len(result.MissingTypes))
	}

	expectedMissingType := reflect.TypeOf(&TestDatabase{})
	if result.MissingTypes[0] != expectedMissingType {
		t.Errorf("Expected missing type %v, got %v", expectedMissingType, result.MissingTypes[0])
	}

	// Check DepsByType contains used dependency
	if len(result.DepsByType) != 1 {
		t.Errorf("Expected 1 dependency by type, got %d", len(result.DepsByType))
	}

	serviceType := reflect.TypeOf(&TestService{})
	if result.DepsByType[serviceType] != serviceDep {
		t.Error("Expected service dependency to be in DepsByType")
	}

	// Check UnusedDeps (none in this case since we only have one dependency and it's used)
	if len(result.UnusedDeps) != 0 {
		t.Errorf("Expected 0 unused dependencies, got %d", len(result.UnusedDeps))
	}
}
