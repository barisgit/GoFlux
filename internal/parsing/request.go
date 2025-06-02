package parsing

import (
	"fmt"
	"io"
	"reflect"
	"strconv"
	"strings"

	"github.com/danielgtaylor/huma/v2"
)

// RequestParser handles parsing incoming HTTP requests into input structs
type RequestParser struct{}

// NewRequestParser creates a new request parser
func NewRequestParser() *RequestParser {
	return &RequestParser{}
}

// ParseInput parses the incoming request into the input struct
func (p *RequestParser) ParseInput(api huma.API, ctx huma.Context, inputPtr reflect.Value, inputType reflect.Type) error {
	input := inputPtr.Elem()

	// Parse each field in the input struct
	for i := 0; i < inputType.NumField(); i++ {
		field := inputType.Field(i)
		if !field.IsExported() {
			continue
		}

		fieldValue := input.Field(i)

		// Handle different parameter types
		if err := p.parseInputFieldWithDefaults(ctx, field, fieldValue); err != nil {
			return fmt.Errorf("failed to parse field %s: %w", field.Name, err)
		}

		// Handle body field
		if field.Name == "Body" {
			if err := p.parseBodyField(api, ctx, fieldValue); err != nil {
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

// parseInputFieldWithDefaults parses a single input field and applies defaults from struct tags
func (p *RequestParser) parseInputFieldWithDefaults(ctx huma.Context, field reflect.StructField, fieldValue reflect.Value) error {
	var value string

	// Handle path parameters
	if pathParam := field.Tag.Get("path"); pathParam != "" {
		value = ctx.Param(pathParam)
	} else if queryParam := field.Tag.Get("query"); queryParam != "" {
		// Handle query parameters
		paramName := strings.Split(queryParam, ",")[0] // Handle "name,explode" format
		value = ctx.Query(paramName)
	} else if headerParam := field.Tag.Get("header"); headerParam != "" {
		// Handle header parameters
		value = ctx.Header(headerParam)
	} else if cookieParam := field.Tag.Get("cookie"); cookieParam != "" {
		// Handle cookie parameters
		cookieHeader := ctx.Header("Cookie")
		if cookieHeader != "" {
			// Parse cookies manually since huma.Context doesn't provide Cookie method
			cookies := p.parseCookies(cookieHeader)
			if cookieValue, exists := cookies[cookieParam]; exists {
				value = cookieValue
			}
		}
	} else {
		return nil // Not a parameter field
	}

	// If value is empty, check for default value in struct tag
	if value == "" {
		if defaultValue := field.Tag.Get("default"); defaultValue != "" {
			value = defaultValue
		}
	}

	// Set the field value using the resolved value (either from request or default)
	return p.setFieldValue(fieldValue, value, field.Type)
}

// parseCookies parses the Cookie header string and returns a map of cookie name to value
func (p *RequestParser) parseCookies(cookieHeader string) map[string]string {
	cookies := make(map[string]string)

	// Split by semicolon and parse each cookie
	for _, cookie := range strings.Split(cookieHeader, ";") {
		cookie = strings.TrimSpace(cookie)
		if cookie == "" {
			continue
		}

		// Split by = to get name and value
		parts := strings.SplitN(cookie, "=", 2)
		if len(parts) == 2 {
			name := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			cookies[name] = value
		}
	}

	return cookies
}

// parseBodyField parses the request body into the field
func (p *RequestParser) parseBodyField(api huma.API, ctx huma.Context, fieldValue reflect.Value) error {
	bodyReader := ctx.BodyReader()
	if bodyReader == nil {
		return nil
	}

	var bodyBytes []byte
	buf := make([]byte, 1024)
	for {
		n, err := bodyReader.Read(buf)
		if n > 0 {
			bodyBytes = append(bodyBytes, buf[:n]...)
		}
		if err != nil {
			if err != io.EOF {
				return fmt.Errorf("error reading body: %w", err)
			}
			break
		}
	}

	if len(bodyBytes) > 0 {
		contentType := ctx.Header("Content-Type")
		if err := api.Unmarshal(contentType, bodyBytes, fieldValue.Addr().Interface()); err != nil {
			return fmt.Errorf("error unmarshaling body: %w", err)
		}
	}

	return nil
}

// parseRawBodyField parses RawBody fields, including huma.MultipartFormFiles
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

	// Handle other RawBody types (binary data, etc.)
	bodyReader := ctx.BodyReader()
	if bodyReader == nil {
		return nil
	}

	var bodyBytes []byte
	buf := make([]byte, 1024)
	for {
		n, err := bodyReader.Read(buf)
		if n > 0 {
			bodyBytes = append(bodyBytes, buf[:n]...)
		}
		if err != nil {
			if err != io.EOF {
				return fmt.Errorf("error reading body: %w", err)
			}
			break
		}
	}

	// For binary RawBody, just set the bytes
	if fieldType.Kind() == reflect.Slice && fieldType.Elem().Kind() == reflect.Uint8 {
		fieldValue.SetBytes(bodyBytes)
	}

	return nil
}

// setFieldValue sets a field value from a string, handling type conversion
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

	// Handle different types
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
		// Handle slice types (e.g., multiple query parameters)
		return p.setSliceValue(fieldValue, value, fieldType)
	default:
		return fmt.Errorf("unsupported field type: %v", fieldType)
	}

	return nil
}

// setSliceValue handles setting slice values from comma-separated strings
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
