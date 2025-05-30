# GoFlux Framework Packages

This directory contains framework packages that can be imported by GoFlux projects to extend functionality.

## Available Packages

### `goflux/pkg/openapi`
OpenAPI specification generation utilities for Huma APIs.

**Functions:**
- `GenerateSpecToFile(api huma.API, outputPath string) error` - Generate and save OpenAPI spec to file
- `GenerateSpec(api huma.API) ([]byte, error)` - Generate OpenAPI spec as JSON bytes
- `GenerateSpecYAML(api huma.API) ([]byte, error)` - Generate OpenAPI spec as YAML bytes
- `GetRouteCount(api huma.API) int` - Count the number of routes in the API

### `goflux/pkg/dev`
Development utilities and CLI command helpers.

**Functions:**
- `AddOpenAPICommand(rootCmd *cobra.Command, apiProvider func() huma.API)` - Add OpenAPI generation command to CLI

## Usage Example

In your GoFlux project's `cmd/server/main.go`:

```go
import (
    "github.com/barisgit/goflux/pkg/dev"
    "github.com/barisgit/goflux/pkg/openapi"
)

// In your main function after setting up Huma API:
dev.AddOpenAPICommand(hooks.CLI(), func() huma.API {
    return humaAPI
})
```

This enables:
```bash
# Generate OpenAPI spec without starting server
./server openapi -o build/openapi.json

# Or in YAML format
./server openapi -o docs/api.yaml -f yaml
```

## Local Development

To use these packages locally in your projects before publishing:

1. Add this to your project's `go.mod`:
```go
replace github.com/barisgit/goflux => /path/to/your/goflux
```

2. Import the packages:
```go
import "github.com/barisgit/goflux/pkg/openapi"
import "github.com/barisgit/goflux/pkg/dev"
``` 