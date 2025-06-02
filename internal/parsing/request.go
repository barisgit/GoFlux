package parsing

import (
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	_ "unsafe" // Required for go:linkname

	"github.com/danielgtaylor/huma/v2"
)

// WARNING: These linkname directives access unexported Huma functions
// They provide significant benefits by reusing Huma's battle-tested logic,
// but may break in future Huma versions if function signatures change.
// If any of these fail, we fall back to our own implementations.

//go:linkname humaFindParams github.com/danielgtaylor/huma/v2.findParams
func humaFindParams(registry huma.Registry, op *huma.Operation, t reflect.Type) *humaFindResult

//go:linkname humaParseInto github.com/danielgtaylor/huma/v2.parseInto
func humaParseInto(ctx huma.Context, f reflect.Value, value string, preSplit []string, p humaParamFieldInfo) (any, error)

//go:linkname humaReadCookies github.com/danielgtaylor/huma/v2.ReadCookies
func humaReadCookies(ctx huma.Context) []*http.Cookie

// Type definitions that mirror Huma's internal types
// These must match exactly or linkname will fail
type humaFindResult struct {
	Paths []humaFindResultPath
}

type humaFindResultPath struct {
	Path  []int
	Value *humaParamFieldInfo
}

type humaParamFieldInfo struct {
	Type       reflect.Type
	Name       string
	Loc        string
	Required   bool
	Default    string
	TimeFormat string
	Explode    bool
	Style      string
	Schema     *huma.Schema
}

// Every method for humaFindResult (simplified version)
func (r *humaFindResult) Every(v reflect.Value, f func(reflect.Value, *humaParamFieldInfo)) {
	current := v
	for _, path := range r.Paths {
		for _, index := range path.Path {
			if current.Kind() == reflect.Pointer {
				if current.IsNil() {
					return
				}
				current = current.Elem()
			}
			if current.Kind() == reflect.Struct && index < current.NumField() {
				current = current.Field(index)
			}
		}
		if path.Value != nil {
			f(current, path.Value)
		}
		current = v // Reset for next path
	}
}

// RequestParser handles parsing incoming HTTP requests into input structs
// Uses go:linkname to access Huma's internal functions with fallbacks
type RequestParser struct {
	useHumaInternals bool // Flag to enable/disable linkname usage
}

// NewRequestParser creates a new request parser
func NewRequestParser() *RequestParser {
	return &RequestParser{
		useHumaInternals: true, // Enable by default, can be disabled if needed
	}
}

// ParseInput parses the incoming request using Huma's internal logic when possible
func (p *RequestParser) ParseInput(api huma.API, ctx huma.Context, inputPtr reflect.Value, inputType reflect.Type) error {
	input := inputPtr.Elem()

	// Try to use Huma's internal parameter discovery first
	if p.useHumaInternals {
		if err := p.parseWithHumaInternals(api, ctx, input, inputType); err != nil {
			// If Huma internals fail (e.g., signature changed), fall back to our implementation
			p.useHumaInternals = false
			return p.parseWithFallback(api, ctx, input, inputType)
		}
	} else {
		return p.parseWithFallback(api, ctx, input, inputType)
	}

	// Handle special fields that our Huma integration doesn't cover
	return p.parseSpecialFields(api, ctx, input, inputType)
}

// parseWithHumaInternals uses Huma's internal functions via go:linkname
func (p *RequestParser) parseWithHumaInternals(api huma.API, ctx huma.Context, input reflect.Value, inputType reflect.Type) error {
	defer func() {
		if r := recover(); r != nil {
			// If linkname functions panic (signature mismatch), disable and fall back
			p.useHumaInternals = false
		}
	}()

	// Use Huma's parameter discovery
	dummyOp := &huma.Operation{Parameters: []*huma.Param{}}
	paramResults := humaFindParams(api.OpenAPI().Components.Schemas, dummyOp, inputType)

	// Use Huma's cookie parsing
	var cookies map[string]*http.Cookie

	// Apply Huma's parameter parsing logic
	paramResults.Every(input, func(fieldValue reflect.Value, paramInfo *humaParamFieldInfo) {
		if !fieldValue.CanSet() {
			return
		}

		// Use Huma's parameter value extraction
		var value string
		switch paramInfo.Loc {
		case "path":
			value = ctx.Param(paramInfo.Name)
		case "query":
			value = ctx.Query(paramInfo.Name)
		case "header":
			value = ctx.Header(paramInfo.Name)
		case "cookie":
			if cookies == nil {
				// Use Huma's ReadCookies
				cookies = make(map[string]*http.Cookie)
				for _, c := range humaReadCookies(ctx) {
					cookies[c.Name] = c
				}
			}
			if c, ok := cookies[paramInfo.Name]; ok {
				value = c.Value
			}
		}

		// Apply defaults
		if value == "" && paramInfo.Default != "" {
			value = paramInfo.Default
		}

		// Use Huma's parseInto function for type conversion
		if value != "" {
			if _, err := humaParseInto(ctx, fieldValue, value, nil, *paramInfo); err != nil {
				// Log error but continue - validation will catch issues later
				return
			}
		}
	})

	return nil
}

// parseWithFallback uses our own implementation when Huma internals aren't available
func (p *RequestParser) parseWithFallback(api huma.API, ctx huma.Context, input reflect.Value, inputType reflect.Type) error {
	// Parse each field in the input struct using our fallback implementation
	for i := 0; i < inputType.NumField(); i++ {
		field := inputType.Field(i)
		if !field.IsExported() {
			continue
		}

		fieldValue := input.Field(i)

		// Handle parameters with our simplified logic
		if err := p.parseFieldParameter(ctx, field, fieldValue); err != nil {
			return fmt.Errorf("failed to parse field %s: %w", field.Name, err)
		}
	}

	return nil
}

// parseSpecialFields handles Body and RawBody fields
func (p *RequestParser) parseSpecialFields(api huma.API, ctx huma.Context, input reflect.Value, inputType reflect.Type) error {
	for i := 0; i < inputType.NumField(); i++ {
		field := inputType.Field(i)
		if !field.IsExported() {
			continue
		}

		fieldValue := input.Field(i)

		// Handle body field
		if field.Name == "Body" {
			if err := p.parseBodyFieldWithHuma(api, ctx, fieldValue); err != nil {
				return fmt.Errorf("failed to parse body: %w", err)
			}
		}

		// Handle RawBody field (including MultipartFormFiles)
		if field.Name == "RawBody" {
			if err := p.parseRawBodyField(api, ctx, fieldValue, field.Type); err != nil {
				return fmt.Errorf("failed to parse raw body: %w", err)
			}
		}
	}

	return nil
}

// parseFieldParameter handles path/query/header/cookie parameters with simplified logic
func (p *RequestParser) parseFieldParameter(ctx huma.Context, field reflect.StructField, fieldValue reflect.Value) error {
	var value string
	var paramFound bool

	// Check parameter tags - this is the core logic we need to keep
	if pathParam := field.Tag.Get("path"); pathParam != "" {
		value = ctx.Param(pathParam)
		paramFound = true
	} else if queryParam := field.Tag.Get("query"); queryParam != "" {
		paramName := strings.Split(queryParam, ",")[0] // Handle "name,explode" format
		value = ctx.Query(paramName)
		paramFound = true
	} else if headerParam := field.Tag.Get("header"); headerParam != "" {
		value = ctx.Header(headerParam)
		paramFound = true
	} else if cookieParam := field.Tag.Get("cookie"); cookieParam != "" {
		// Parse cookies manually from Cookie header since Huma doesn't expose Request()
		cookieHeader := ctx.Header("Cookie")
		if cookieHeader != "" {
			value = p.parseCookieValue(cookieHeader, cookieParam)
		}
		paramFound = true
	}

	if !paramFound {
		return nil // Not a parameter field
	}

	// Apply defaults if value is empty
	if value == "" {
		if defaultValue := field.Tag.Get("default"); defaultValue != "" {
			value = defaultValue
		}
	}

	// Use simplified field value setting
	return p.setFieldValue(fieldValue, value, field.Type)
}

// parseBodyFieldWithHuma uses Huma's body reading and unmarshaling logic
func (p *RequestParser) parseBodyFieldWithHuma(api huma.API, ctx huma.Context, fieldValue reflect.Value) error {
	bodyReader := ctx.BodyReader()
	if bodyReader == nil {
		return nil
	}

	// Use simple io.ReadAll - Huma handles efficiency elsewhere
	bodyBytes, err := io.ReadAll(bodyReader)
	if err != nil {
		return fmt.Errorf("error reading body: %w", err)
	}

	if len(bodyBytes) > 0 {
		contentType := ctx.Header("Content-Type")
		// Use Huma's Unmarshal method which handles all content types
		if err := api.Unmarshal(contentType, bodyBytes, fieldValue.Addr().Interface()); err != nil {
			return fmt.Errorf("error unmarshaling body: %w", err)
		}
	}

	return nil
}

// parseRawBodyField handles RawBody fields, including huma.MultipartFormFiles
func (p *RequestParser) parseRawBodyField(api huma.API, ctx huma.Context, fieldValue reflect.Value, fieldType reflect.Type) error {
	// Check if this is a huma.MultipartFormFiles type
	if strings.Contains(fieldType.String(), "MultipartFormFiles") {
		// Use Huma's native multipart parsing via GetMultipartForm()
		form, err := ctx.GetMultipartForm()
		if err != nil {
			return fmt.Errorf("failed to parse multipart form: %w", err)
		}

		// Create a new instance of the MultipartFormFiles type
		instance := reflect.New(fieldType).Elem()

		// Set the Form field on the MultipartFormFiles instance
		formField := instance.FieldByName("Form")
		if formField.IsValid() && formField.CanSet() {
			formField.Set(reflect.ValueOf(form))
		}

		// Set the parsed value
		fieldValue.Set(instance)
		return nil
	}

	// Handle other RawBody types using simple io.ReadAll
	bodyReader := ctx.BodyReader()
	if bodyReader == nil {
		return nil
	}

	bodyBytes, err := io.ReadAll(bodyReader)
	if err != nil {
		return fmt.Errorf("error reading body: %w", err)
	}

	// For binary RawBody, just set the bytes
	if fieldType.Kind() == reflect.Slice && fieldType.Elem().Kind() == reflect.Uint8 {
		fieldValue.SetBytes(bodyBytes)
	}

	return nil
}

// setFieldValue sets a field value from a string with basic type conversion
// Simplified version that covers the most common cases
func (p *RequestParser) setFieldValue(fieldValue reflect.Value, value string, fieldType reflect.Type) error {
	if value == "" {
		return nil
	}

	// Handle pointer types
	if fieldType.Kind() == reflect.Pointer {
		if fieldValue.IsNil() {
			fieldValue.Set(reflect.New(fieldType.Elem()))
		}
		return p.setFieldValue(fieldValue.Elem(), value, fieldType.Elem())
	}

	// Handle basic types - the essential ones for web APIs
	switch fieldType.Kind() {
	case reflect.String:
		fieldValue.SetString(value)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if intVal, err := strconv.ParseInt(value, 10, 64); err == nil {
			fieldValue.SetInt(intVal)
		} else {
			return fmt.Errorf("invalid integer value: %s", value)
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if uintVal, err := strconv.ParseUint(value, 10, 64); err == nil {
			fieldValue.SetUint(uintVal)
		} else {
			return fmt.Errorf("invalid unsigned integer value: %s", value)
		}
	case reflect.Float32, reflect.Float64:
		if floatVal, err := strconv.ParseFloat(value, 64); err == nil {
			fieldValue.SetFloat(floatVal)
		} else {
			return fmt.Errorf("invalid float value: %s", value)
		}
	case reflect.Bool:
		if boolVal, err := strconv.ParseBool(value); err == nil {
			fieldValue.SetBool(boolVal)
		} else {
			return fmt.Errorf("invalid boolean value: %s", value)
		}
	case reflect.Slice:
		// Handle slice types for multiple query parameters
		return p.setSliceValue(fieldValue, value, fieldType)
	default:
		return fmt.Errorf("unsupported field type: %v", fieldType)
	}

	return nil
}

// setSliceValue handles slice values - simplified but functional
func (p *RequestParser) setSliceValue(fieldValue reflect.Value, value string, fieldType reflect.Type) error {
	elemType := fieldType.Elem()
	values := strings.Split(value, ",")

	slice := reflect.MakeSlice(fieldType, len(values), len(values))

	for i, v := range values {
		elemValue := slice.Index(i)
		if err := p.setFieldValue(elemValue, strings.TrimSpace(v), elemType); err != nil {
			return err
		}
	}

	fieldValue.Set(slice)
	return nil
}

// parseCookieValue extracts a specific cookie value from the Cookie header
func (p *RequestParser) parseCookieValue(cookieHeader, cookieName string) string {
	for _, cookie := range strings.Split(cookieHeader, ";") {
		cookie = strings.TrimSpace(cookie)
		if cookie == "" {
			continue
		}

		parts := strings.SplitN(cookie, "=", 2)
		if len(parts) == 2 {
			name := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			if name == cookieName {
				return value
			}
		}
	}
	return ""
}
