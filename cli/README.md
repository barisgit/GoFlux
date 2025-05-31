# GoFlux CLI

This directory contains the CLI application for GoFlux.

## Structure

- `main.go` - CLI entry point
- The importable GoFlux framework is in the root directory

## Usage

The CLI is built and distributed as a single binary called `flux`.

### Development

```bash
# From project root
make dev-install   # Install in development mode
flux new myapp     # Use the CLI
```

### Building

```bash
# From project root
make build         # Creates ./flux binary
```

## Package vs CLI

- **CLI**: This directory (`./cli/`) - builds to `flux` binary
- **Framework Package**: Root directory (`./`) - import as `github.com/barisgit/goflux`
