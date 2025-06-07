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

// Test input types
type TestParamsInput struct {
	ID       string `path:"id"`
	Name     string `query:"name"`
	AuthKey  string `header:"X-Auth-Key"`
	Session  string `cookie:"session"`
	Optional string `query:"optional" default:"default-value"`
}

type TestBodyInput struct {
	Body TestUserData `json:"user"`
}

type TestUserData struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type TestRawBodyInput struct {
	RawBody []byte
}

type TestComplexInput struct {
	ID     string       `path:"id"`
	Query  string       `query:"q"`
	Body   TestUserData `json:"user"`
	Header string       `header:"X-Custom"`
	Cookie string       `cookie:"token"`
}

type TestSliceInput struct {
	Tags     []string `query:"tags"`
	Numbers  []int    `query:"numbers"`
	Booleans []bool   `query:"booleans"`
}

func createTestAPI() huma.API {
	mux := http.NewServeMux()
	config := huma.DefaultConfig("Test API", "1.0.0")
	return humago.New(mux, config)
}

func TestNewRequestParser(t *testing.T) {
	parser := NewRequestParser()
	if parser == nil {
		t.Fatal("Expected parser to be created")
	}

	if !parser.useHumaInternals {
		t.Error("Expected Huma internals to be enabled by default")
	}
}

func TestParseInput_Parameters(t *testing.T) {
	api := createTestAPI()
	parser := NewRequestParser()

	ctx := testutil.NewMockContext().
		WithParam("id", "123").
		WithQuery("name", "test-user").
		WithHeader("X-Auth-Key", "secret-key").
		WithHeader("Cookie", "session=session-123").
		WithCookie(&http.Cookie{Name: "session", Value: "session-123"})

	inputType := reflect.TypeOf(TestParamsInput{})
	inputPtr := reflect.New(inputType)

	err := parser.ParseInput(api, ctx, inputPtr, inputType)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	input := inputPtr.Elem().Interface().(TestParamsInput)

	if input.ID != "123" {
		t.Errorf("Expected ID '123', got '%s'", input.ID)
	}

	if input.Name != "test-user" {
		t.Errorf("Expected Name 'test-user', got '%s'", input.Name)
	}

	if input.AuthKey != "secret-key" {
		t.Errorf("Expected AuthKey 'secret-key', got '%s'", input.AuthKey)
	}

	if input.Session != "session-123" {
		t.Errorf("Expected Session 'session-123', got '%s'", input.Session)
	}
}

func TestParseInput_DefaultValues(t *testing.T) {
	api := createTestAPI()
	parser := NewRequestParser()
	parser.useHumaInternals = false // Test fallback implementation

	ctx := testutil.NewMockContext()

	inputType := reflect.TypeOf(TestParamsInput{})
	inputPtr := reflect.New(inputType)

	err := parser.ParseInput(api, ctx, inputPtr, inputType)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	input := inputPtr.Elem().Interface().(TestParamsInput)

	if input.Optional != "default-value" {
		t.Errorf("Expected Optional 'default-value', got '%s'", input.Optional)
	}
}

func TestParseInput_Body(t *testing.T) {
	api := createTestAPI()
	parser := NewRequestParser()

	bodyJSON := `{"name": "John Doe", "email": "john@example.com"}`
	ctx := testutil.NewMockContext().
		WithJSONBody(bodyJSON)

	inputType := reflect.TypeOf(TestBodyInput{})
	inputPtr := reflect.New(inputType)

	err := parser.ParseInput(api, ctx, inputPtr, inputType)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	input := inputPtr.Elem().Interface().(TestBodyInput)

	if input.Body.Name != "John Doe" {
		t.Errorf("Expected Name 'John Doe', got '%s'", input.Body.Name)
	}

	if input.Body.Email != "john@example.com" {
		t.Errorf("Expected Email 'john@example.com', got '%s'", input.Body.Email)
	}
}

func TestParseInput_RawBody(t *testing.T) {
	api := createTestAPI()
	parser := NewRequestParser()

	bodyData := []byte("raw body content")
	ctx := testutil.NewMockContext().
		WithBody(bytes.NewReader(bodyData))

	inputType := reflect.TypeOf(TestRawBodyInput{})
	inputPtr := reflect.New(inputType)

	err := parser.ParseInput(api, ctx, inputPtr, inputType)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	input := inputPtr.Elem().Interface().(TestRawBodyInput)

	if !bytes.Equal(input.RawBody, bodyData) {
		t.Errorf("Expected RawBody %v, got %v", bodyData, input.RawBody)
	}
}

func TestParseInput_Complex(t *testing.T) {
	api := createTestAPI()
	parser := NewRequestParser()

	bodyJSON := `{"name": "Jane Smith", "email": "jane@example.com"}`
	ctx := testutil.NewMockContext().
		WithParam("id", "456").
		WithQuery("q", "search-term").
		WithHeader("X-Custom", "custom-value").
		WithHeader("Content-Type", "application/json").
		WithHeader("Cookie", "token=token-789").
		WithCookie(&http.Cookie{Name: "token", Value: "token-789"}).
		WithBody(strings.NewReader(bodyJSON))

	inputType := reflect.TypeOf(TestComplexInput{})
	inputPtr := reflect.New(inputType)

	err := parser.ParseInput(api, ctx, inputPtr, inputType)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	input := inputPtr.Elem().Interface().(TestComplexInput)

	if input.ID != "456" {
		t.Errorf("Expected ID '456', got '%s'", input.ID)
	}

	if input.Query != "search-term" {
		t.Errorf("Expected Query 'search-term', got '%s'", input.Query)
	}

	if input.Header != "custom-value" {
		t.Errorf("Expected Header 'custom-value', got '%s'", input.Header)
	}

	if input.Cookie != "token-789" {
		t.Errorf("Expected Cookie 'token-789', got '%s'", input.Cookie)
	}

	if input.Body.Name != "Jane Smith" {
		t.Errorf("Expected Body.Name 'Jane Smith', got '%s'", input.Body.Name)
	}
}

func TestParseInput_SliceValues(t *testing.T) {
	api := createTestAPI()
	parser := NewRequestParser()
	parser.useHumaInternals = false // Test fallback implementation

	ctx := testutil.NewMockContext().
		WithQuery("tags", "tag1,tag2,tag3").
		WithQuery("numbers", "1,2,3").
		WithQuery("booleans", "true,false,true")

	inputType := reflect.TypeOf(TestSliceInput{})
	inputPtr := reflect.New(inputType)

	err := parser.ParseInput(api, ctx, inputPtr, inputType)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	input := inputPtr.Elem().Interface().(TestSliceInput)

	expectedTags := []string{"tag1", "tag2", "tag3"}
	if !reflect.DeepEqual(input.Tags, expectedTags) {
		t.Errorf("Expected Tags %v, got %v", expectedTags, input.Tags)
	}

	expectedNumbers := []int{1, 2, 3}
	if !reflect.DeepEqual(input.Numbers, expectedNumbers) {
		t.Errorf("Expected Numbers %v, got %v", expectedNumbers, input.Numbers)
	}

	expectedBooleans := []bool{true, false, true}
	if !reflect.DeepEqual(input.Booleans, expectedBooleans) {
		t.Errorf("Expected Booleans %v, got %v", expectedBooleans, input.Booleans)
	}
}

func TestParseInput_FallbackMode(t *testing.T) {
	api := createTestAPI()
	parser := NewRequestParser()
	parser.useHumaInternals = false // Force fallback mode

	ctx := testutil.NewMockContext().
		WithParam("id", "fallback-test").
		WithQuery("name", "fallback-user")

	inputType := reflect.TypeOf(TestParamsInput{})
	inputPtr := reflect.New(inputType)

	err := parser.ParseInput(api, ctx, inputPtr, inputType)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	input := inputPtr.Elem().Interface().(TestParamsInput)

	if input.ID != "fallback-test" {
		t.Errorf("Expected ID 'fallback-test', got '%s'", input.ID)
	}

	if input.Name != "fallback-user" {
		t.Errorf("Expected Name 'fallback-user', got '%s'", input.Name)
	}
}

func TestParseFieldParameter_PathParam(t *testing.T) {
	parser := NewRequestParser()

	ctx := testutil.NewMockContext().
		WithParam("test", "path-value")

	field := reflect.StructField{
		Name: "Test",
		Tag:  `path:"test"`,
		Type: reflect.TypeOf(""),
	}

	fieldValue := reflect.New(reflect.TypeOf("")).Elem()

	err := parser.parseFieldParameter(ctx, field, fieldValue)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if fieldValue.String() != "path-value" {
		t.Errorf("Expected 'path-value', got '%s'", fieldValue.String())
	}
}

func TestParseFieldParameter_QueryParam(t *testing.T) {
	parser := NewRequestParser()

	ctx := testutil.NewMockContext().
		WithQuery("test", "query-value")

	field := reflect.StructField{
		Name: "Test",
		Tag:  `query:"test"`,
		Type: reflect.TypeOf(""),
	}

	fieldValue := reflect.New(reflect.TypeOf("")).Elem()

	err := parser.parseFieldParameter(ctx, field, fieldValue)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if fieldValue.String() != "query-value" {
		t.Errorf("Expected 'query-value', got '%s'", fieldValue.String())
	}
}

func TestParseFieldParameter_HeaderParam(t *testing.T) {
	parser := NewRequestParser()

	ctx := testutil.NewMockContext().
		WithHeader("X-Test", "header-value")

	field := reflect.StructField{
		Name: "Test",
		Tag:  `header:"X-Test"`,
		Type: reflect.TypeOf(""),
	}

	fieldValue := reflect.New(reflect.TypeOf("")).Elem()

	err := parser.parseFieldParameter(ctx, field, fieldValue)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if fieldValue.String() != "header-value" {
		t.Errorf("Expected 'header-value', got '%s'", fieldValue.String())
	}
}
