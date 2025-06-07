package parsing

import (
	"bytes"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/barisgit/goflux/internal/testutil"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humago"
)

// Test output types
type TestOutput struct {
	Body TestData `json:"data"`
}

type TestData struct {
	ID      int    `json:"id"`
	Message string `json:"message"`
}

type TestOutputWithStatus struct {
	Status int      `json:"-"`
	Body   TestData `json:"data"`
}

type TestOutputWithHeaders struct {
	ContentType  string   `header:"Content-Type"`
	CustomHeader string   `header:"X-Custom-Header"`
	Body         TestData `json:"data"`
}

type TestByteOutput struct {
	Body []byte
}

func createTestResponseAPI() huma.API {
	mux := http.NewServeMux()
	config := huma.DefaultConfig("Test API", "1.0.0")
	return humago.New(mux, config)
}

func TestNewResponseWriter(t *testing.T) {
	writer := NewResponseWriter()
	if writer == nil {
		t.Fatal("Expected response writer to be created")
	}
}

func TestWriteOutput_BasicResponse(t *testing.T) {
	api := createTestResponseAPI()
	writer := NewResponseWriter()

	ctx := testutil.NewMockContext()

	output := &TestOutput{
		Body: TestData{
			ID:      1,
			Message: "test message",
		},
	}

	operation := huma.Operation{
		DefaultStatus: http.StatusOK,
	}

	err := writer.WriteOutput(api, ctx, output, reflect.TypeOf(*output), operation)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if ctx.Status() != http.StatusOK {
		t.Errorf("Expected status 200, got %d", ctx.Status())
	}

	bodyContent := ctx.GetResponseBody()
	if !strings.Contains(bodyContent, "test message") {
		t.Errorf("Expected body to contain 'test message', got '%s'", bodyContent)
	}
}

func TestWriteOutput_WithCustomStatus(t *testing.T) {
	api := createTestResponseAPI()
	writer := NewResponseWriter()

	ctx := testutil.NewMockContext()

	output := &TestOutputWithStatus{
		Status: http.StatusCreated,
		Body: TestData{
			ID:      2,
			Message: "created message",
		},
	}

	operation := huma.Operation{
		DefaultStatus: http.StatusOK,
	}

	err := writer.WriteOutput(api, ctx, output, reflect.TypeOf(*output), operation)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if ctx.Status() != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", ctx.Status())
	}
}

func TestWriteOutput_WithHeaders(t *testing.T) {
	api := createTestResponseAPI()
	writer := NewResponseWriter()

	ctx := testutil.NewMockContext()

	output := &TestOutputWithHeaders{
		ContentType:  "application/custom+json",
		CustomHeader: "custom-value",
		Body: TestData{
			ID:      3,
			Message: "header message",
		},
	}

	operation := huma.Operation{
		DefaultStatus: http.StatusOK,
	}

	err := writer.WriteOutput(api, ctx, output, reflect.TypeOf(*output), operation)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if ctx.GetResponseHeader("Content-Type") != "application/custom+json" {
		t.Errorf("Expected Content-Type 'application/custom+json', got '%s'", ctx.GetResponseHeader("Content-Type"))
	}

	if ctx.GetResponseHeader("X-Custom-Header") != "custom-value" {
		t.Errorf("Expected X-Custom-Header 'custom-value', got '%s'", ctx.GetResponseHeader("X-Custom-Header"))
	}
}

func TestWriteOutput_ByteResponse(t *testing.T) {
	api := createTestResponseAPI()
	writer := NewResponseWriter()

	ctx := testutil.NewMockContext()

	testBytes := []byte("raw byte content")
	output := &TestByteOutput{
		Body: testBytes,
	}

	operation := huma.Operation{
		DefaultStatus: http.StatusOK,
	}

	err := writer.WriteOutput(api, ctx, output, reflect.TypeOf(*output), operation)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if ctx.Status() != http.StatusOK {
		t.Errorf("Expected status 200, got %d", ctx.Status())
	}

	bodyContent := []byte(ctx.GetResponseBody())
	if !bytes.Equal(bodyContent, testBytes) {
		t.Errorf("Expected body %v, got %v", testBytes, bodyContent)
	}
}

func TestWriteOutput_NoContentStatus(t *testing.T) {
	api := createTestResponseAPI()
	writer := NewResponseWriter()

	ctx := testutil.NewMockContext()

	output := &TestOutputWithStatus{
		Status: http.StatusNoContent,
		Body: TestData{
			ID:      4,
			Message: "no content",
		},
	}

	operation := huma.Operation{
		DefaultStatus: http.StatusOK,
	}

	err := writer.WriteOutput(api, ctx, output, reflect.TypeOf(*output), operation)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if ctx.Status() != http.StatusNoContent {
		t.Errorf("Expected status 204, got %d", ctx.Status())
	}

	// Should not write body for 204 status
	if len(ctx.GetResponseBody()) != 0 {
		t.Errorf("Expected empty body for 204 status, got %d bytes", len(ctx.GetResponseBody()))
	}
}

func TestWriteOutput_NotModifiedStatus(t *testing.T) {
	api := createTestResponseAPI()
	writer := NewResponseWriter()

	ctx := testutil.NewMockContext()

	output := &TestOutputWithStatus{
		Status: http.StatusNotModified,
		Body: TestData{
			ID:      5,
			Message: "not modified",
		},
	}

	operation := huma.Operation{
		DefaultStatus: http.StatusOK,
	}

	err := writer.WriteOutput(api, ctx, output, reflect.TypeOf(*output), operation)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if ctx.Status() != http.StatusNotModified {
		t.Errorf("Expected status 304, got %d", ctx.Status())
	}

	// Should not write body for 304 status
	if len(ctx.GetResponseBody()) != 0 {
		t.Errorf("Expected empty body for 304 status, got %d bytes", len(ctx.GetResponseBody()))
	}
}

func TestWriteOutput_AlreadyWritten(t *testing.T) {
	api := createTestResponseAPI()
	writer := NewResponseWriter()

	ctx := testutil.NewMockContext()
	// Simulate already written by setting status
	ctx.SetStatus(http.StatusInternalServerError)

	output := &TestOutput{
		Body: TestData{
			ID:      6,
			Message: "should not write",
		},
	}

	operation := huma.Operation{
		DefaultStatus: http.StatusOK,
	}

	err := writer.WriteOutput(api, ctx, output, reflect.TypeOf(*output), operation)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Should not change status when already written
	if ctx.Status() != http.StatusInternalServerError {
		t.Errorf("Expected status to remain 500, got %d", ctx.Status())
	}
}

func TestWriteOutput_NilOutput(t *testing.T) {
	api := createTestResponseAPI()
	writer := NewResponseWriter()

	ctx := testutil.NewMockContext()

	operation := huma.Operation{
		DefaultStatus: http.StatusOK,
	}

	err := writer.WriteOutput(api, ctx, nil, reflect.TypeOf(TestOutput{}), operation)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if ctx.Status() != http.StatusOK {
		t.Errorf("Expected status 200, got %d", ctx.Status())
	}
}

func TestWriteOutput_DefaultStatus(t *testing.T) {
	api := createTestResponseAPI()
	writer := NewResponseWriter()

	ctx := testutil.NewMockContext()

	output := &TestOutput{
		Body: TestData{
			ID:      7,
			Message: "default status",
		},
	}

	operation := huma.Operation{
		DefaultStatus: 0, // No default status set
	}

	err := writer.WriteOutput(api, ctx, output, reflect.TypeOf(*output), operation)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if ctx.Status() != http.StatusOK {
		t.Errorf("Expected default status 200, got %d", ctx.Status())
	}
}

func TestGetHeaderName(t *testing.T) {
	writer := NewResponseWriter()

	tests := []struct {
		name     string
		field    reflect.StructField
		expected string
	}{
		{
			name: "explicit header tag",
			field: reflect.StructField{
				Name: "Authorization",
				Tag:  `header:"X-Auth-Token"`,
			},
			expected: "X-Auth-Token",
		},
		{
			name: "body field - no header",
			field: reflect.StructField{
				Name: "Body",
			},
			expected: "",
		},
		{
			name: "status field - no header",
			field: reflect.StructField{
				Name: "Status",
			},
			expected: "",
		},
		{
			name: "regular field - use field name",
			field: reflect.StructField{
				Name: "ContentType",
			},
			expected: "ContentType",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := writer.getHeaderName(tt.field)
			if result != tt.expected {
				t.Errorf("Expected header name '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestWriteOutput_WithPointer(t *testing.T) {
	api := createTestResponseAPI()
	writer := NewResponseWriter()

	ctx := testutil.NewMockContext()

	output := &TestOutput{
		Body: TestData{
			ID:      8,
			Message: "pointer test",
		},
	}

	operation := huma.Operation{
		DefaultStatus: http.StatusOK,
	}

	// Test with pointer to output
	err := writer.WriteOutput(api, ctx, output, reflect.TypeOf(*output), operation)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if ctx.Status() != http.StatusOK {
		t.Errorf("Expected status 200, got %d", ctx.Status())
	}

	bodyContent := ctx.GetResponseBody()
	if !strings.Contains(bodyContent, "pointer test") {
		t.Errorf("Expected body to contain 'pointer test', got '%s'", bodyContent)
	}
}
