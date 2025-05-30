# Testing GoFlux Framework Locally

This guide shows how to test the GoFlux CLI + Framework functionality locally without publishing to GitHub.

## Quick Test (Automated)

Run the automated test that creates a project and tests framework integration:

```bash
make test-framework
```

This will:
1. Install GoFlux CLI in dev mode
2. Create a test project 
3. Add local framework dependencies
4. Test OpenAPI generation without server

## Manual Testing

### 1. Install CLI in Development Mode

```bash
make dev-install
```

This creates `~/bin/flux` that runs from your local source.

### 2. Create a Test Project

```bash
flux new my-test-app
cd my-test-app
```

### 3. Add Local Framework Dependency

Add this line to your project's `go.mod` file:

```go
replace github.com/yourusername/goflux => /Users/blaz/Programming_local/Projects/gofusion
```

Then run:
```bash
go mod tidy
```

### 4. Test OpenAPI Generation

```bash
# Generate OpenAPI spec without starting server
go run ./cmd/server openapi -o build/openapi.json

# Check the output
cat build/openapi.json | jq .info.title
```

### 5. Test Development Mode

```bash
# Start development mode (uses new OpenAPI generation)
flux dev
```

This will now use the new direct OpenAPI generation instead of fetching from server.

## What's Different

### Before (Server-based)
- Starts server on port
- Waits for server to be ready
- Fetches `/api/openapi.json` via HTTP
- Server must be running

### After (Direct Generation)
- Creates Huma API instance
- Generates OpenAPI spec directly
- No server startup required
- Much faster and more reliable

## Development Workflow

1. Make changes to framework code in `gofusion/pkg/`
2. Test in your project with `go run ./cmd/server openapi`
3. No need to reinstall CLI - changes are immediate
4. When ready, commit and publish

## Package Structure

```
gofusion/
├── pkg/                    # Framework packages
│   ├── openapi/           # OpenAPI generation utilities  
│   │   └── generator.go
│   ├── dev/               # Development CLI helpers
│   │   └── cli.go
│   └── README.md          # Package documentation
├── internal/              # CLI-only code
│   ├── dev/              # Development orchestrator
│   └── ...
└── templates/             # Project templates
    └── cmd/server/main.go.tmpl  # Uses framework packages
```

This hybrid approach gives you:
- ✅ CLI tool for scaffolding and dev orchestration
- ✅ Framework packages for enhanced functionality  
- ✅ Local testing without publishing
- ✅ Easy migration path to publish later 