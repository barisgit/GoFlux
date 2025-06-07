package core

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

// Test types for dependency testing
type TestService struct {
	Name string
}

type TestDatabase struct {
	ConnectionString string
}

type TestInputType struct {
	Page     int `query:"page" minimum:"1" default:"1"`
	PageSize int `query:"page_size" minimum:"1" maximum:"100" default:"20"`
}

func TestNewDependencyCore(t *testing.T) {
	tests := []struct {
		name         string
		depName      string
		loadFn       interface{}
		expectPanic  bool
		expectedType reflect.Type
		panicMessage string
	}{
		{
			name:    "valid dependency function",
			depName: "testService",
			loadFn: func(ctx context.Context, input interface{}) (*TestService, error) {
				return &TestService{Name: "test"}, nil
			},
			expectPanic:  false,
			expectedType: reflect.TypeOf(&TestService{}),
		},
		{
			name:         "invalid - not a function",
			depName:      "invalid",
			loadFn:       "not a function",
			expectPanic:  true,
			panicMessage: "loadFn must be a function",
		},
		{
			name:    "invalid - wrong number of inputs",
			depName: "invalid",
			loadFn: func(ctx context.Context) (*TestService, error) {
				return nil, nil
			},
			expectPanic:  true,
			panicMessage: "loadFn must have signature func(context.Context, interface{}) (T, error)",
		},
		{
			name:    "invalid - wrong number of outputs",
			depName: "invalid",
			loadFn: func(ctx context.Context, input interface{}) *TestService {
				return nil
			},
			expectPanic:  true,
			panicMessage: "loadFn must have signature func(context.Context, interface{}) (T, error)",
		},
		{
			name:    "invalid - first parameter not context",
			depName: "invalid",
			loadFn: func(s string, input interface{}) (*TestService, error) {
				return nil, nil
			},
			expectPanic:  true,
			panicMessage: "first parameter must be context.Context",
		},
		{
			name:    "invalid - second parameter not interface{}",
			depName: "invalid",
			loadFn: func(ctx context.Context, input string) (*TestService, error) {
				return nil, nil
			},
			expectPanic:  true,
			panicMessage: "second parameter must be interface{}",
		},
		{
			name:    "invalid - last return not error",
			depName: "invalid",
			loadFn: func(ctx context.Context, input interface{}) (*TestService, string) {
				return nil, ""
			},
			expectPanic:  true,
			panicMessage: "last return value must be error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.expectPanic {
				defer func() {
					if r := recover(); r != nil {
						if !strings.Contains(fmt.Sprintf("%v", r), tt.panicMessage) {
							t.Errorf("Expected panic message containing '%s', got '%v'", tt.panicMessage, r)
						}
					} else {
						t.Error("Expected panic, but function didn't panic")
					}
				}()
			}

			dep := NewDependencyCore(tt.depName, tt.loadFn)

			if !tt.expectPanic {
				if dep.Name != tt.depName {
					t.Errorf("Expected name %s, got %s", tt.depName, dep.Name)
				}

				if dep.Type() != tt.expectedType {
					t.Errorf("Expected type %v, got %v", tt.expectedType, dep.Type())
				}

				if dep.InputFields != nil {
					t.Error("Expected InputFields to be nil by default")
				}

				if len(dep.RequiredMiddleware) != 0 {
					t.Error("Expected RequiredMiddleware to be empty by default")
				}
			}
		})
	}
}

func TestDependencyCoreLoad(t *testing.T) {
	// Test successful load
	dep := NewDependencyCore("testService", func(ctx context.Context, input interface{}) (*TestService, error) {
		return &TestService{Name: "loaded"}, nil
	})

	ctx := context.Background()
	result, err := dep.Load(ctx, nil)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	service, ok := result.(*TestService)
	if !ok {
		t.Fatalf("Expected *TestService, got %T", result)
	}

	if service.Name != "loaded" {
		t.Errorf("Expected name 'loaded', got '%s'", service.Name)
	}
}

func TestDependencyCoreLoadWithError(t *testing.T) {
	expectedErr := fmt.Errorf("load failed")
	dep := NewDependencyCore("testService", func(ctx context.Context, input interface{}) (*TestService, error) {
		return nil, expectedErr
	})

	ctx := context.Background()
	result, err := dep.Load(ctx, nil)

	if err != expectedErr {
		t.Errorf("Expected error %v, got %v", expectedErr, err)
	}

	if result != nil {
		t.Errorf("Expected nil result on error, got %v", result)
	}
}

func TestDependencyCoreLoadWithInput(t *testing.T) {
	dep := NewDependencyCore("testService", func(ctx context.Context, input interface{}) (*TestService, error) {
		// Check if input is passed correctly
		if input == nil {
			return &TestService{Name: "no-input"}, nil
		}
		return &TestService{Name: "with-input"}, nil
	})

	ctx := context.Background()

	// Test with nil input
	result1, err := dep.Load(ctx, nil)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	service1 := result1.(*TestService)
	if service1.Name != "no-input" {
		t.Errorf("Expected 'no-input', got '%s'", service1.Name)
	}

	// Test with actual input
	result2, err := dep.Load(ctx, "some input")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	service2 := result2.(*TestService)
	if service2.Name != "with-input" {
		t.Errorf("Expected 'with-input', got '%s'", service2.Name)
	}
}

func TestDependencyCoreType(t *testing.T) {
	dep := NewDependencyCore("testService", func(ctx context.Context, input interface{}) (*TestService, error) {
		return nil, nil
	})

	expectedType := reflect.TypeOf(&TestService{})
	if dep.Type() != expectedType {
		t.Errorf("Expected type %v, got %v", expectedType, dep.Type())
	}
}

func TestDependencyCoreWithInputFields(t *testing.T) {
	dep := NewDependencyCore("testService", func(ctx context.Context, input interface{}) (*TestService, error) {
		return &TestService{}, nil
	})

	// Add input fields
	depWithInput := dep.WithInputFields(TestInputType{})

	if depWithInput.InputFields == nil {
		t.Error("Expected InputFields to be set")
	}

	expectedType := reflect.TypeOf(TestInputType{})
	if depWithInput.InputFields != expectedType {
		t.Errorf("Expected input fields type %v, got %v", expectedType, depWithInput.InputFields)
	}

	// Original dependency should be unchanged
	if dep.InputFields != nil {
		t.Error("Expected original dependency to be unchanged")
	}
}

func TestDependencyCoreWithInputFieldsPointer(t *testing.T) {
	dep := NewDependencyCore("testService", func(ctx context.Context, input interface{}) (*TestService, error) {
		return &TestService{}, nil
	})

	// Test with pointer type
	depWithInput := dep.WithInputFields(&TestInputType{})

	expectedType := reflect.TypeOf(TestInputType{}) // Should be dereferenced
	if depWithInput.InputFields != expectedType {
		t.Errorf("Expected input fields type %v, got %v", expectedType, depWithInput.InputFields)
	}
}

func TestDependencyCoreRequiresMiddleware(t *testing.T) {
	dep := NewDependencyCore("testService", func(ctx context.Context, input interface{}) (*TestService, error) {
		return &TestService{}, nil
	})

	middleware1 := func() {}
	middleware2 := func() {}

	// Add middleware
	depWithMiddleware := dep.RequiresMiddleware(middleware1, middleware2)

	if len(depWithMiddleware.RequiredMiddleware) != 2 {
		t.Errorf("Expected 2 middleware, got %d", len(depWithMiddleware.RequiredMiddleware))
	}

	// Original dependency should be unchanged
	if len(dep.RequiredMiddleware) != 0 {
		t.Error("Expected original dependency to be unchanged")
	}
}

func TestDependencyCoreCopySemantics(t *testing.T) {
	// Test that WithInputFields and RequiresMiddleware create copies
	dep := NewDependencyCore("testService", func(ctx context.Context, input interface{}) (*TestService, error) {
		return &TestService{}, nil
	})

	depWithInput := dep.WithInputFields(TestInputType{})
	depWithMiddleware := dep.RequiresMiddleware(func() {})

	// All should have different addresses (copies)
	if dep == depWithInput {
		t.Error("WithInputFields should create a copy")
	}

	if dep == depWithMiddleware {
		t.Error("RequiresMiddleware should create a copy")
	}

	if depWithInput == depWithMiddleware {
		t.Error("Different operations should create different copies")
	}

	// But they should have the same name and load function
	if dep.Name != depWithInput.Name || dep.Name != depWithMiddleware.Name {
		t.Error("Copies should have the same name")
	}

	if dep.Type() != depWithInput.Type() || dep.Type() != depWithMiddleware.Type() {
		t.Error("Copies should have the same type")
	}
}

func TestDependencyCoreComplexTypes(t *testing.T) {
	// Test with interface type
	dep1 := NewDependencyCore("interface", func(ctx context.Context, input interface{}) (interface{}, error) {
		return "string", nil
	})

	expectedType1 := reflect.TypeOf((*interface{})(nil)).Elem()
	if dep1.Type() != expectedType1 {
		t.Errorf("Expected interface{} type %v, got %v", expectedType1, dep1.Type())
	}

	// Test with slice type
	dep2 := NewDependencyCore("slice", func(ctx context.Context, input interface{}) ([]string, error) {
		return []string{"test"}, nil
	})

	expectedType2 := reflect.TypeOf([]string{})
	if dep2.Type() != expectedType2 {
		t.Errorf("Expected slice type %v, got %v", expectedType2, dep2.Type())
	}

	// Test with map type
	dep3 := NewDependencyCore("map", func(ctx context.Context, input interface{}) (map[string]int, error) {
		return map[string]int{"test": 1}, nil
	})

	expectedType3 := reflect.TypeOf(map[string]int{})
	if dep3.Type() != expectedType3 {
		t.Errorf("Expected map type %v, got %v", expectedType3, dep3.Type())
	}
}
