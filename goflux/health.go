package goflux

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
)

// HealthResponse represents a standard health check response
type HealthResponse struct {
	Body struct {
		Status  string `json:"status" example:"ok" doc:"Service status"`
		Message string `json:"message,omitempty" example:"Service is running" doc:"Optional status message"`
		Version string `json:"version,omitempty" example:"1.0.0" doc:"Optional service version"`
	}
}

// AddHealthCheck adds a standard health check endpoint to a Huma API
func AddHealthCheck(api huma.API, path string, serviceName string, version string) {
	if path == "" {
		path = "/api/health"
	}

	huma.Register(api, huma.Operation{
		OperationID: "health-check",
		Method:      http.MethodGet,
		Path:        path,
		Summary:     "Health Check",
		Description: "Check if the service is running and healthy",
		Tags:        []string{"Health"},
	}, func(ctx context.Context, input *struct{}) (*HealthResponse, error) {
		resp := &HealthResponse{}
		resp.Body.Status = "ok"

		if serviceName != "" {
			resp.Body.Message = serviceName + " is running"
		}

		if version != "" {
			resp.Body.Version = version
		}

		return resp, nil
	})
}

// CustomHealthCheck allows users to provide their own health check logic
func CustomHealthCheck(api huma.API, path string, healthFunc func(ctx context.Context) (*HealthResponse, error)) {
	if path == "" {
		path = "/api/health"
	}

	huma.Register(api, huma.Operation{
		OperationID: "health-check",
		Method:      http.MethodGet,
		Path:        path,
		Summary:     "Health Check",
		Description: "Check if the service is running and healthy",
		Tags:        []string{"Health"},
	}, func(ctx context.Context, input *struct{}) (*HealthResponse, error) {
		return healthFunc(ctx)
	})
}
