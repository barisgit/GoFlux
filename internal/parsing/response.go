package parsing

import (
	"fmt"
	"net/http"
	"reflect"
	"strconv"

	"github.com/danielgtaylor/huma/v2"
)

// ResponseWriter handles writing HTTP responses
type ResponseWriter struct{}

// NewResponseWriter creates a new response writer
func NewResponseWriter() *ResponseWriter {
	return &ResponseWriter{}
}

// WriteOutput writes the output response using Huma's exact pipeline
func (w *ResponseWriter) WriteOutput(api huma.API, ctx huma.Context, output interface{}, outputType reflect.Type, operation huma.Operation) error {
	// Don't write anything if response has already been written
	if ctx.Status() != 0 {
		return nil
	}

	outputValue := reflect.ValueOf(output)
	if outputValue.Kind() == reflect.Pointer {
		outputValue = outputValue.Elem()
	}

	// Set the default status if not already set
	status := operation.DefaultStatus
	if status == 0 {
		status = http.StatusOK
	}

	// Check for Status field in output
	if statusField := outputValue.FieldByName("Status"); statusField.IsValid() && statusField.Kind() == reflect.Int {
		if statusField.Int() != 0 {
			status = int(statusField.Int())
		}
	}

	// Handle response headers (like Huma does)
	ct := ""
	for i := 0; i < outputValue.NumField(); i++ {
		field := outputType.Field(i)
		if !field.IsExported() {
			continue
		}

		if headerName := w.getHeaderName(field); headerName != "" {
			headerValue := outputValue.Field(i)
			if headerValue.IsValid() && !headerValue.IsZero() {
				if headerValue.Kind() == reflect.String && headerName == "Content-Type" {
					// Track custom content type (like Huma does)
					ct = headerValue.String()
				}
				ctx.SetHeader(headerName, fmt.Sprintf("%v", headerValue.Interface()))
			}
		}
	}

	// Handle response body (exactly like Huma's Register function)
	if bodyField := outputValue.FieldByName("Body"); bodyField.IsValid() {
		body := bodyField.Interface()

		// Handle byte slice special case (like Huma does)
		if b, ok := body.([]byte); ok {
			ctx.SetStatus(status)
			if _, err := ctx.BodyWriter().Write(b); err != nil {
				return fmt.Errorf("error writing byte response: %w", err)
			}
			return nil
		}

		// Use Huma's exact response pipeline: Transform -> Marshal
		if ct == "" {
			// Content negotiation (like Huma does)
			var err error
			ct, err = api.Negotiate(ctx.Header("Accept"))
			if err != nil {
				return fmt.Errorf("unable to negotiate content type: %w", err)
			}

			if ctf, ok := body.(huma.ContentTypeFilter); ok {
				ct = ctf.ContentType(ct)
			}

			ctx.SetHeader("Content-Type", ct)
		}

		// Transform the response body (like Huma does)
		tval, terr := api.Transform(ctx, strconv.Itoa(status), body)
		if terr != nil {
			return fmt.Errorf("error transforming response: %w", terr)
		}

		ctx.SetStatus(status)

		// Marshal and write the response (like Huma does)
		if status != http.StatusNoContent && status != http.StatusNotModified {
			if merr := api.Marshal(ctx.BodyWriter(), ct, tval); merr != nil {
				return fmt.Errorf("error marshaling response: %w", merr)
			}
		}
	} else {
		// No body field, just set status
		ctx.SetStatus(status)
	}

	return nil
}

// getHeaderName extracts header name from struct field
func (w *ResponseWriter) getHeaderName(field reflect.StructField) string {
	if header := field.Tag.Get("header"); header != "" {
		return header
	}
	// Default to field name for headers (excluding special fields)
	if field.Name != "Body" && field.Name != "Status" {
		return field.Name
	}
	return ""
}
