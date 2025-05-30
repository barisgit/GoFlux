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
go run . openapi -o build/openapi.json

# Check the output
cat build/openapi.json | jq .info.title
```

### 5. Test Development Mode

```bash
# Start development mode (uses new OpenAPI generation)
flux dev
```
