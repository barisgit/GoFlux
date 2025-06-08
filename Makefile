.PHONY: help build dev-install dev-uninstall clean install test coverage cli-coverage e2e-test release-dry-run release tag status

# Default target
help:
	@echo "🚀 GoFlux CLI Development"
	@echo ""
	@echo "Build & Install:"
	@echo "  build         - Build the flux binary"
	@echo "  dev-install   - Install in development mode"
	@echo "  dev-uninstall - Remove development installation"
	@echo "  install       - Install to /usr/local/bin"
	@echo "  clean         - Clean build artifacts"
	@echo ""
	@echo "Testing:"
	@echo "  test          - Run all tests"
	@echo "  coverage      - Run framework tests with coverage report"
	@echo "  cli-coverage  - Run CLI tests with coverage report"
	@echo "  e2e-test      - Run end-to-end CLI tests"
	@echo ""
	@echo "Release:"
	@echo "  release-dry-run - Test release process"
	@echo "  tag VERSION=x   - Create git tag"
	@echo "  release         - Publish release"
	@echo ""
	@echo "Utilities:"
	@echo "  status        - Show installation status"

# Build the binary
build:
	@echo "🔨 Building GoFlux CLI..."
	@cd cli && go build -o ../flux .
	@echo "✅ Built: ./flux"

# Development installation
dev-install:
	@echo "🔧 Installing GoFlux in development mode..."
	@mkdir -p ~/bin
	@echo '#!/bin/bash' > ~/bin/flux
	@echo 'GOFLUX_SOURCE="$(PWD)"' >> ~/bin/flux
	@echo 'flux_WORK_DIR="$$(pwd)"' >> ~/bin/flux
	@echo 'cd "$$GOFLUX_SOURCE/cli" && flux_WORK_DIR="$$flux_WORK_DIR" go run . "$$@"' >> ~/bin/flux
	@chmod +x ~/bin/flux
	@echo "✅ GoFlux installed in development mode at ~/bin/flux"
	@echo "Add ~/bin to your PATH if needed:"
	@echo "  export PATH=\"$$HOME/bin:$$PATH\""

# Remove development installation
dev-uninstall:
	@echo "🗑️  Removing development installation..."
	@rm -f ~/bin/flux
	@echo "✅ Development installation removed"

# Install to system
install: build
	@echo "📦 Installing GoFlux CLI to /usr/local/bin..."
	@sudo mv flux /usr/local/bin/
	@echo "✅ Installed! Run 'flux --help' from anywhere"

# Clean build artifacts
clean:
	@echo "🧹 Cleaning build artifacts..."
	@rm -f flux *.out *.html
	@rm -f cli/coverage.out cli/coverage.html
	@rm -rf test-* 
	@echo "✅ Cleaned"

# Run all tests
test:
	@echo "🧪 Running framework tests..."
	@go test $(shell go list ./... | grep -v -E 'templates|examples') -short
	@echo "🧪 Running CLI tests..."
	@cd cli && go test ./... -short

# Run tests with coverage and open HTML report
coverage:
	@echo "📊 Generating framework coverage report..."
	@go test -coverprofile=coverage.out $(shell go list ./... | grep -v -E 'templates|examples') -short
	@echo ""
	@echo "📈 Coverage Summary:"
	@go tool cover -func=coverage.out | tail -1
	@echo ""
	@go tool cover -html=coverage.out -o coverage.html
	@echo "✅ Coverage report generated: coverage.html"
	@if command -v open >/dev/null 2>&1; then \
		open coverage.html; \
	elif command -v xdg-open >/dev/null 2>&1; then \
		xdg-open coverage.html; \
	else \
		echo "Open coverage.html in your browser"; \
	fi

# Run CLI tests with coverage and open HTML report  
cli-coverage:
	@echo "📊 Generating CLI coverage report..."
	@cd cli && go test -coverprofile=coverage.out ./... -short
	@echo ""
	@echo "📈 CLI Coverage Summary:"
	@cd cli && go tool cover -func=coverage.out | tail -1
	@echo ""
	@cd cli && go tool cover -html=coverage.out -o coverage.html
	@echo "✅ CLI coverage report generated: cli/coverage.html"
	@if command -v open >/dev/null 2>&1; then \
		open cli/coverage.html; \
	elif command -v xdg-open >/dev/null 2>&1; then \
		xdg-open cli/coverage.html; \
	else \
		echo "Open cli/coverage.html in your browser"; \
	fi

# End-to-end tests
e2e-test:
	@echo "🧪 Running end-to-end CLI tests..."
	@which flux > /dev/null || (echo "❌ flux binary not found. Run 'make dev-install' first." && exit 1)
	@echo "ℹ️  Note: Advanced template tests require Docker for PostgreSQL"
	@echo "ℹ️  If Docker tests fail, they will be skipped automatically"
	@go test -v -run "TestE2E" ./e2e_test.go -timeout=10m

# Show installation status
status:
	@echo "📊 GoFlux Installation Status:"
	@if [ -f ~/bin/flux ]; then \
		echo "✅ Development mode: ~/bin/flux"; \
	else \
		echo "❌ Development mode: Not installed"; \
	fi
	@if command -v flux >/dev/null 2>&1; then \
		echo "✅ System installation: $$(which flux)"; \
	else \
		echo "❌ System installation: Not found"; \
	fi

# Create git tag
tag:
	@if [ -z "$(VERSION)" ]; then \
		echo "❌ VERSION required. Usage: make tag VERSION=v1.0.0"; \
		exit 1; \
	fi
	@if [ "$$(git rev-parse --abbrev-ref HEAD)" != "main" ]; then \
		echo "❌ Must be on main branch to tag"; \
		exit 1; \
	fi
	@if [ -n "$$(git status --porcelain)" ]; then \
		echo "❌ Working directory not clean. Commit or stash changes first."; \
		exit 1; \
	fi
	@if [ "$$(git rev-list --count origin/main..main)" -gt 0 ]; then \
		echo "❌ Local main branch has unpushed commits"; \
		exit 1; \
	fi
	@echo "🏷️  Creating tag $(VERSION)..."
	@git tag $(VERSION)
	@echo "✅ Tag $(VERSION) created. Push with: git push origin $(VERSION)"

# Test release process
release-dry-run:
	@echo "🧪 Testing release process..."
	@if ! command -v goreleaser >/dev/null 2>&1; then \
		echo "❌ goreleaser not found. Install with:"; \
		echo "  go install github.com/goreleaser/goreleaser@latest"; \
		exit 1; \
	fi
	@goreleaser release --snapshot --clean
	@echo "✅ Release dry run completed"

# Create and publish release
release:
	@echo "🚀 Creating release..."
	@if ! command -v goreleaser >/dev/null 2>&1; then \
		echo "❌ goreleaser not found. Install with:"; \
		echo "  go install github.com/goreleaser/goreleaser@latest"; \
		exit 1; \
	fi
	@if [ -z "$(VERSION)" ]; then \
		echo "❌ VERSION required. Usage: make release VERSION=v1.0.0"; \
		exit 1; \
	fi
	@if [ "$$(git rev-parse --abbrev-ref HEAD)" != "main" ]; then \
		echo "❌ Must be on main branch to release"; \
		exit 1; \
	fi
	@if [ -n "$$(git status --porcelain)" ]; then \
		echo "❌ Working directory not clean. Commit or stash changes first."; \
		exit 1; \
	fi
	@if [ "$$(git rev-list --count origin/main..main)" -gt 0 ]; then \
		echo "❌ Local main branch has unpushed commits"; \
		exit 1; \
	fi
	@echo "Creating release for version: $(VERSION)"
	@git tag $(VERSION)
	@git push origin $(VERSION)
	@echo "✅ Release $(VERSION) triggered"