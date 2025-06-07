package testutil

import (
	"bytes"
	"context"
	"crypto/tls"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
)

// MockContext provides a comprehensive mock implementation of huma.Context
// for testing purposes. It can be configured to simulate various request scenarios.
type MockContext struct {
	// Request data
	method     string
	path       string
	params     map[string]string
	query      map[string]string
	headers    map[string]string
	cookies    []*http.Cookie
	body       io.Reader
	host       string
	remoteAddr string
	operation  *huma.Operation

	// Response data
	status      int
	respHeaders map[string]string
	respBody    *bytes.Buffer

	// Connection info
	tls     *tls.ConnectionState
	url     url.URL
	version huma.ProtoVersion
}

// NewMockContext creates a new mock context with sensible defaults
func NewMockContext() *MockContext {
	return &MockContext{
		method:      "GET",
		path:        "/",
		params:      make(map[string]string),
		query:       make(map[string]string),
		headers:     make(map[string]string),
		cookies:     make([]*http.Cookie, 0),
		respHeaders: make(map[string]string),
		respBody:    &bytes.Buffer{},
		host:        "localhost:8080",
		remoteAddr:  "127.0.0.1:12345",
		operation:   &huma.Operation{},
		url:         url.URL{Path: "/"},
		version:     huma.ProtoVersion{ProtoMajor: 1, ProtoMinor: 1},
	}
}

// Builder methods for configuring the mock context

func (m *MockContext) WithMethod(method string) *MockContext {
	m.method = method
	return m
}

func (m *MockContext) WithPath(path string) *MockContext {
	m.path = path
	m.url.Path = path
	return m
}

func (m *MockContext) WithParam(name, value string) *MockContext {
	m.params[name] = value
	return m
}

func (m *MockContext) WithQuery(name, value string) *MockContext {
	m.query[name] = value
	return m
}

func (m *MockContext) WithHeader(name, value string) *MockContext {
	m.headers[name] = value
	return m
}

func (m *MockContext) WithCookie(cookie *http.Cookie) *MockContext {
	m.cookies = append(m.cookies, cookie)
	return m
}

func (m *MockContext) WithBody(body io.Reader) *MockContext {
	m.body = body
	return m
}

func (m *MockContext) WithJSONBody(json string) *MockContext {
	m.body = strings.NewReader(json)
	m.headers["Content-Type"] = "application/json"
	return m
}

func (m *MockContext) WithOperation(op *huma.Operation) *MockContext {
	m.operation = op
	return m
}

func (m *MockContext) WithHost(host string) *MockContext {
	m.host = host
	return m
}

func (m *MockContext) WithTLS(tls *tls.ConnectionState) *MockContext {
	m.tls = tls
	return m
}

// huma.Context interface implementation

func (m *MockContext) Context() context.Context {
	return context.Background()
}

func (m *MockContext) Operation() *huma.Operation {
	return m.operation
}

func (m *MockContext) Method() string {
	return m.method
}

func (m *MockContext) Param(name string) string {
	return m.params[name]
}

func (m *MockContext) Query(name string) string {
	return m.query[name]
}

func (m *MockContext) Header(name string) string {
	return m.headers[name]
}

func (m *MockContext) Host() string {
	return m.host
}

func (m *MockContext) RemoteAddr() string {
	return m.remoteAddr
}

func (m *MockContext) EachHeader(fn func(name, value string)) {
	for k, v := range m.headers {
		fn(k, v)
	}
}

func (m *MockContext) BodyReader() io.Reader {
	if m.body != nil {
		return m.body
	}
	return strings.NewReader("")
}

func (m *MockContext) GetMultipartForm() (*multipart.Form, error) {
	return nil, nil
}

func (m *MockContext) SetReadDeadline(deadline time.Time) error {
	return nil
}

func (m *MockContext) SetStatus(code int) {
	m.status = code
}

func (m *MockContext) Status() int {
	return m.status
}

func (m *MockContext) AppendHeader(name string, value string) {
	if m.respHeaders == nil {
		m.respHeaders = make(map[string]string)
	}
	// For testing, just set the header (real implementation would append)
	m.respHeaders[name] = value
}

func (m *MockContext) SetHeader(name string, value string) {
	if m.respHeaders == nil {
		m.respHeaders = make(map[string]string)
	}
	m.respHeaders[name] = value
}

func (m *MockContext) BodyWriter() io.Writer {
	return m.respBody
}

func (m *MockContext) TLS() *tls.ConnectionState {
	return m.tls
}

func (m *MockContext) URL() url.URL {
	return m.url
}

func (m *MockContext) Version() huma.ProtoVersion {
	return m.version
}

// Helper methods for testing

// GetResponseBody returns the response body as a string
func (m *MockContext) GetResponseBody() string {
	return m.respBody.String()
}

// GetResponseHeader returns a response header value
func (m *MockContext) GetResponseHeader(name string) string {
	return m.respHeaders[name]
}

// Reset clears the response state for reuse
func (m *MockContext) Reset() {
	m.status = 0
	m.respHeaders = make(map[string]string)
	m.respBody.Reset()
}
