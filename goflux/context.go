package goflux

import (
	"net/http"
	"strconv"

	"github.com/danielgtaylor/huma/v2"
)

// ============================================================================
// TRPC-STYLE CONTEXT EXTENSIONS
// ============================================================================

// FluxContext extends huma.Context with convenience methods
type FluxContext struct {
	huma.Context
}

// Wrap wraps a huma.Context to add GoFlux convenience methods
func Wrap(ctx huma.Context) *FluxContext {
	return &FluxContext{Context: ctx}
}

// WriteErr writes an error response with the given status and message
func (ctx *FluxContext) WriteErr(status int, message string, errors ...error) {
	WriteErr(ctx.Context, status, message, errors...)
}

// WriteResponse writes a successful response with optional content type
func (ctx *FluxContext) WriteResponse(status int, body interface{}, contentType ...string) {
	api := GetAPI(ctx.Context)

	ctx.SetStatus(status)

	var ct string
	if len(contentType) > 0 && contentType[0] != "" {
		ct = contentType[0]
		ctx.SetHeader("Content-Type", ct)
	} else {
		// Content negotiation
		var err error
		ct, err = api.Negotiate(ctx.Header("Accept"))
		if err != nil {
			ctx.WriteErr(http.StatusNotAcceptable, "unable to marshal response", err)
			return
		}
		ctx.SetHeader("Content-Type", ct)
	}

	// Handle byte slice special case
	if b, ok := body.([]byte); ok {
		ctx.BodyWriter().Write(b)
		return
	}

	// Transform and marshal using Huma's pipeline
	tval, terr := api.Transform(ctx.Context, strconv.Itoa(status), body)
	if terr != nil {
		ctx.WriteErr(http.StatusInternalServerError, "error transforming response", terr)
		return
	}

	if err := api.Marshal(ctx.BodyWriter(), ct, tval); err != nil {
		ctx.WriteErr(http.StatusInternalServerError, "error marshaling response", err)
		return
	}
}

// ============================================================================
// RESPONSE WRITERS
// ============================================================================

// 1xx

// Continue writes a 100 Continue response
func (ctx *FluxContext) Continue() {
	ctx.SetStatus(http.StatusContinue)
}

// SwitchingProtocols writes a 101 Switching Protocols response
func (ctx *FluxContext) SwitchingProtocols() {
	ctx.SetStatus(http.StatusSwitchingProtocols)
}

// 2xx

// OK writes a 200 OK response
func (ctx *FluxContext) OK(body interface{}, contentType ...string) {
	ctx.WriteResponse(http.StatusOK, body, contentType...)
}

// Created writes a 201 Created response
func (ctx *FluxContext) Created(body interface{}, contentType ...string) {
	ctx.WriteResponse(http.StatusCreated, body, contentType...)
}

// Accepted writes a 202 Accepted response
func (ctx *FluxContext) Accepted(body interface{}, contentType ...string) {
	ctx.WriteResponse(http.StatusAccepted, body, contentType...)
}

// NoContent writes a 204 No Content response
func (ctx *FluxContext) NoContent() {
	ctx.SetStatus(http.StatusNoContent)
}

// 3xx

// MovedPermanently writes a 301 Moved Permanently response
func (ctx *FluxContext) MovedPermanently(location string) {
	ctx.SetStatus(http.StatusMovedPermanently)
	ctx.SetHeader("Location", location)
}

// Found writes a 302 Found response
func (ctx *FluxContext) Found(location string) {
	ctx.SetStatus(http.StatusFound)
	ctx.SetHeader("Location", location)
}

// NotModified writes a 304 Not Modified response
func (ctx *FluxContext) NotModified() {
	ctx.SetStatus(http.StatusNotModified)
}

// For 4xx and 5xx, we can use error structs, that users can then pregenerate for common responses
type StatusError struct {
	Status  int
	Message string
}

// NewStatusError creates a new StatusError with the given status, message, and errors
func NewStatusError(status int, message string, errors ...error) *StatusError {
	return &StatusError{
		Status:  status,
		Message: message,
	}
}

func (ctx *FluxContext) WriteStatusError(statusError *StatusError, errors ...error) {
	ctx.WriteErr(statusError.Status, statusError.Message, errors...)
}

// 4xx

// NewBadRequestError writes a 400 Bad Request response
func (ctx *FluxContext) NewBadRequestError(message string, errors ...error) {
	ctx.WriteErr(http.StatusBadRequest, message, errors...)
}

// NewUnauthorizedError writes a 401 Unauthorized response
func (ctx *FluxContext) NewUnauthorizedError(message string, errors ...error) {
	ctx.WriteErr(http.StatusUnauthorized, message, errors...)
}

// NewPaymentRequiredError writes a 402 Payment Required response
func (ctx *FluxContext) NewPaymentRequiredError(message string, errors ...error) {
	ctx.WriteErr(http.StatusPaymentRequired, message, errors...)
}

// NewForbiddenError writes a 403 Forbidden response
func (ctx *FluxContext) NewForbiddenError(message string, errors ...error) {
	ctx.WriteErr(http.StatusForbidden, message, errors...)
}

// NewNotFoundError writes a 404 Not Found response
func (ctx *FluxContext) NewNotFoundError(message string, errors ...error) {
	ctx.WriteErr(http.StatusNotFound, message, errors...)
}

// NewMethodNotAllowedError writes a 405 Method Not Allowed response
func (ctx *FluxContext) NewMethodNotAllowedError(message string, errors ...error) {
	ctx.WriteErr(http.StatusMethodNotAllowed, message, errors...)
}

// NewConflictError writes a 409 Conflict response
func (ctx *FluxContext) NewConflictError(message string, errors ...error) {
	ctx.WriteErr(http.StatusConflict, message, errors...)
}

// NewTooManyRequestsError writes a 429 Too Many Requests response
func (ctx *FluxContext) NewTooManyRequestsError(message string, errors ...error) {
	ctx.WriteErr(http.StatusTooManyRequests, message, errors...)
}

// 5xx

// NewInternalServerError writes a 500 Internal Server Error response
func (ctx *FluxContext) NewInternalServerError(message string, errors ...error) {
	ctx.WriteErr(http.StatusInternalServerError, message, errors...)
}

// NewNotImplementedError writes a 501 Not Implemented response
func (ctx *FluxContext) NewNotImplementedError(message string, errors ...error) {
	ctx.WriteErr(http.StatusNotImplemented, message, errors...)
}

// NewBadGatewayError writes a 502 Bad Gateway response
func (ctx *FluxContext) NewBadGatewayError(message string, errors ...error) {
	ctx.WriteErr(http.StatusBadGateway, message, errors...)
}

// NewServiceUnavailableError writes a 503 Service Unavailable response
func (ctx *FluxContext) NewServiceUnavailableError(message string, errors ...error) {
	ctx.WriteErr(http.StatusServiceUnavailable, message, errors...)
}
