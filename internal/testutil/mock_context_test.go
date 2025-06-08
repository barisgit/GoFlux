package testutil

import (
	"net/http"
	"testing"
)

func TestNewMockContext(t *testing.T) {
	ctx := NewMockContext()

	if ctx == nil {
		t.Fatal("Expected mock context to be created")
	}

	if ctx.Method() != "GET" {
		t.Errorf("Expected default method GET, got %s", ctx.Method())
	}

	if ctx.Host() != "localhost:8080" {
		t.Errorf("Expected default host localhost:8080, got %s", ctx.Host())
	}
}

func TestMockContextBuilder(t *testing.T) {
	ctx := NewMockContext().
		WithMethod("POST").
		WithPath("/users").
		WithParam("id", "123").
		WithQuery("page", "1").
		WithHeader("Authorization", "Bearer token").
		WithJSONBody(`{"name": "test"}`)

	if ctx.Method() != "POST" {
		t.Errorf("Expected method POST, got %s", ctx.Method())
	}

	if ctx.Param("id") != "123" {
		t.Errorf("Expected param id=123, got %s", ctx.Param("id"))
	}

	if ctx.Query("page") != "1" {
		t.Errorf("Expected query page=1, got %s", ctx.Query("page"))
	}

	if ctx.Header("Authorization") != "Bearer token" {
		t.Errorf("Expected Authorization header, got %s", ctx.Header("Authorization"))
	}

	if ctx.Header("Content-Type") != "application/json" {
		t.Errorf("Expected Content-Type to be set to application/json, got %s", ctx.Header("Content-Type"))
	}

	bodyReader := ctx.BodyReader()
	if bodyReader == nil {
		t.Fatal("Expected body reader to be available")
	}
}

func TestMockContextCookie(t *testing.T) {
	cookie := &http.Cookie{
		Name:  "session",
		Value: "abc123",
	}

	ctx := NewMockContext().WithCookie(cookie)

	// Note: The current implementation doesn't expose cookies directly
	// through the huma.Context interface, but they're stored internally
	if len(ctx.cookies) != 1 {
		t.Errorf("Expected 1 cookie, got %d", len(ctx.cookies))
	}

	if ctx.cookies[0].Name != "session" {
		t.Errorf("Expected cookie name 'session', got '%s'", ctx.cookies[0].Name)
	}
}

func TestMockContextResponse(t *testing.T) {
	ctx := NewMockContext()

	// Test response writing
	ctx.SetStatus(201)
	ctx.SetHeader("X-Custom", "value")

	writer := ctx.BodyWriter()
	writer.Write([]byte("response body"))

	if ctx.Status() != 201 {
		t.Errorf("Expected status 201, got %d", ctx.Status())
	}

	if ctx.GetResponseHeader("X-Custom") != "value" {
		t.Errorf("Expected X-Custom header 'value', got '%s'", ctx.GetResponseHeader("X-Custom"))
	}

	if ctx.GetResponseBody() != "response body" {
		t.Errorf("Expected response body 'response body', got '%s'", ctx.GetResponseBody())
	}
}

func TestMockContextReset(t *testing.T) {
	ctx := NewMockContext()

	// Set some response state
	ctx.SetStatus(404)
	ctx.SetHeader("X-Test", "value")
	ctx.BodyWriter().Write([]byte("some content"))

	// Reset
	ctx.Reset()

	if ctx.Status() != 0 {
		t.Errorf("Expected status to be reset to 0, got %d", ctx.Status())
	}

	if ctx.GetResponseHeader("X-Test") != "" {
		t.Errorf("Expected X-Test header to be cleared, got '%s'", ctx.GetResponseHeader("X-Test"))
	}

	if ctx.GetResponseBody() != "" {
		t.Errorf("Expected response body to be cleared, got '%s'", ctx.GetResponseBody())
	}
}

func TestMockContextEachHeader(t *testing.T) {
	ctx := NewMockContext().
		WithHeader("Authorization", "Bearer token").
		WithHeader("Content-Type", "application/json")

	headers := make(map[string]string)
	ctx.EachHeader(func(name, value string) {
		headers[name] = value
	})

	if headers["Authorization"] != "Bearer token" {
		t.Errorf("Expected Authorization header in EachHeader iteration")
	}

	if headers["Content-Type"] != "application/json" {
		t.Errorf("Expected Content-Type header in EachHeader iteration")
	}
}

func TestMockContextBodyReader(t *testing.T) {
	jsonData := `{"name": "John", "age": 30}`
	ctx := NewMockContext().WithJSONBody(jsonData)

	reader := ctx.BodyReader()
	buf := make([]byte, len(jsonData))
	n, err := reader.Read(buf)

	if err != nil {
		t.Fatalf("Expected no error reading body, got %v", err)
	}

	if n != len(jsonData) {
		t.Errorf("Expected to read %d bytes, got %d", len(jsonData), n)
	}

	if string(buf) != jsonData {
		t.Errorf("Expected body content '%s', got '%s'", jsonData, string(buf))
	}
}

func TestMockContextDefaultBodyReader(t *testing.T) {
	ctx := NewMockContext()

	reader := ctx.BodyReader()
	if reader == nil {
		t.Fatal("Expected default body reader to be available")
	}

	// Should return empty string reader by default
	buf := make([]byte, 10)
	n, _ := reader.Read(buf)

	if n != 0 {
		t.Errorf("Expected empty body by default, got %d bytes", n)
	}
}
