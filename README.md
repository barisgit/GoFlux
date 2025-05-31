# GoFlux Framework Packages

This directory contains minimal, modular utilities that can be imported by GoFlux projects to reduce boilerplate and add common functionality.

## Available Packages

### `goflux/openapi`

OpenAPI specification generation utilities for Huma APIs.

**Functions:**

- `GenerateSpecToFile(api huma.API, outputPath string) error` - Generate and save OpenAPI spec to file
- `GenerateSpec(api huma.API) ([]byte, error)` - Generate OpenAPI spec as JSON bytes
- `GenerateSpecYAML(api huma.API) ([]byte, error)` - Generate OpenAPI spec as YAML bytes
- `GetRouteCount(api huma.API) int` - Count the number of routes in the API

### `goflux/dev`

Development utilities and CLI command helpers.

**Functions:**

- `AddOpenAPICommand(rootCmd *cobra.Command, apiProvider func() huma.API)` - Add OpenAPI generation command to CLI

### `goflux/base` (Base Package)
Core utilities for GoFlux applications.

**Static File Serving:**

- `StaticHandler(assets embed.FS, config StaticConfig) http.Handler` - Configurable static file serving
- `StaticConfig` - Configuration for static file behavior (SPA mode, dev mode, asset directory, etc.)

**Health Checks:**

- `AddHealthCheck(api huma.API, path, serviceName, version string)` - Add standard health endpoint
- `CustomHealthCheck(api huma.API, path string, healthFunc func(ctx context.Context) (*HealthResponse, error))` - Add custom health logic
- `HealthResponse` - Standard health check response structure

**OpenAPI Utilities:**

- `AddOpenAPICommand(rootCmd *cobra.Command, apiProvider func() huma.API)` - Add OpenAPI CLI command
- All OpenAPI generation functions re-exported from openapi package

## Usage Example

### Basic Health Check

```go
import goflux "github.com/barisgit/goflux"

// Simple health check
goflux.AddHealthCheck(api, "/api/health", "My Service", "1.0.0")

// Custom health check
goflux.CustomHealthCheck(api, "/api/health", func(ctx context.Context) (*goflux.HealthResponse, error) {
    // Your custom health logic here
    resp := &goflux.HealthResponse{}
    resp.Body.Status = "ok"
    resp.Body.Message = "Custom health check passed"
    return resp, nil
})
```

### Static File Serving

```go
import (
    "embed"
    goflux "github.com/barisgit/goflux"
)

//go:embed dist/*
var assets embed.FS

// Configure static serving
staticHandler := goflux.StaticHandler(assets, goflux.StaticConfig{
    AssetsDir: "dist",
    SPAMode:   true,  // Enable SPA routing
    DevMode:   false, // Production mode
    APIPrefix: "/api/",
})

// Use with any router
router.Handle("/*", staticHandler)
```

### OpenAPI CLI Command

```go
import goflux "github.com/barisgit/goflux"

// Add OpenAPI generation command
goflux.AddOpenAPICommand(cli.Root(), func() huma.API {
    return setupAPI() // Your API setup function
})

// Now supports: ./server openapi -o api.json
```

## Philosophy

GoFlux utilities follow these principles:

1. **Minimal & Modular** - Small, focused utilities that users can pick and choose
2. **User Control** - Users provide their own router setup and configuration  
3. **No Magic** - Clear, explicit behavior without hidden abstractions
4. **Framework Agnostic** - Works with any Huma-compatible router (Chi, Gin, Echo, etc.)
5. **Embedded-First** - Designed for single-binary deployment with embedded assets

## Local Development

To use these packages locally in your projects before publishing:

1. Add this to your project's `go.mod`:

```go
replace github.com/barisgit/goflux => /path/to/your/goflux
```

2. Import the packages:

```go
import goflux "github.com/barisgit/goflux"
```
