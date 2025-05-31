.PHONY: help build dev-install dev-uninstall clean install test release-dry-run release tag

# Default target
help:
	@echo "🚀 GoFlux CLI Development"
	@echo ""
	@echo "Available targets:"
	@echo "  dev-install   - Install in development mode (editable)"
	@echo "  dev-uninstall - Remove development installation"
	@echo "  build         - Build the flux binary"
	@echo "  clean         - Clean build artifacts"
	@echo "  install       - Install to /usr/local/bin (production)"
	@echo "  test          - Run tests"
	@echo "  release-dry-run - Test release process without publishing"
	@echo "  release       - Create and publish a new release"
	@echo "  tag           - Create a new git tag (usage: make tag VERSION=v1.0.0)"
	@echo ""
	@echo "Development workflow:"
	@echo "  1. make dev-install    # Install in dev mode"
	@echo "  2. flux new myapp    # Use from anywhere"
	@echo "  3. make dev-uninstall  # Clean up when done"
	@echo ""
	@echo "Release workflow:"
	@echo "  1. make release-dry-run  # Test the release"
	@echo "  2. make tag VERSION=v1.0.0  # Create tag"
	@echo "  3. git push origin v1.0.0   # Trigger release"

# Development installation - creates a script that runs go run
dev-install:
	@echo "🔧 Installing GoFlux in development mode..."
	@mkdir -p ~/bin
	@echo '#!/bin/bash' > ~/bin/flux
	@echo '# GoFlux Development Mode' >> ~/bin/flux
	@echo 'GOFLUX_SOURCE="$(PWD)"' >> ~/bin/flux
	@echo 'flux_WORK_DIR="$$(pwd)"' >> ~/bin/flux
	@echo 'cd "$$GOFLUX_SOURCE/cli" && flux_WORK_DIR="$$flux_WORK_DIR" go run . "$$@"' >> ~/bin/flux
	@chmod +x ~/bin/flux
	@echo ""
	@echo "✅ GoFlux installed in development mode!"
	@echo "📍 Location: ~/bin/flux"
	@echo "📁 Source: $(PWD)/cli"
	@echo ""
	@echo "Add ~/bin to your PATH if not already there:"
	@echo "  echo 'export PATH=\"\$$HOME/bin:\$$PATH\"' >> ~/.bashrc"
	@echo "  echo 'export PATH=\"\$$HOME/bin:\$$PATH\"' >> ~/.zshrc"
	@echo "  source ~/.bashrc  # or ~/.zshrc"
	@echo ""
	@echo "Now you can run 'flux' from anywhere!"

# Remove development installation
dev-uninstall:
	@echo "🗑️  Removing development installation..."
	@rm -f ~/bin/flux
	@echo "✅ Development installation removed"

# Build the binary
build:
	@echo "🔨 Building GoFlux CLI..."
	@cd cli && go build -o ../flux .
	@echo "✅ Built: ./flux"

# Development mode - run with go run (local only)
dev:
	@cd cli && go run . $(ARGS)

# Clean build artifacts
clean:
	@echo "🧹 Cleaning build artifacts..."
	@rm -f flux
	@rm -rf test-*
	@echo "✅ Cleaned"

# Install to system (production)
install: build
	@echo "📦 Installing GoFlux CLI to /usr/local/bin..."
	@sudo mv flux /usr/local/bin/
	@echo "✅ Installed! Run 'flux --help' from anywhere"

# Run tests
test:
	@echo "🧪 Running framework tests..."
	@go test ./...
	@echo "🧪 Running CLI tests..."
	@cd cli && go test ./...

# Quick test - create and test a project
test-project: 
	@echo "🧪 Testing project creation..."
	@rm -rf test-project || true
	@cd cli && go run . new test-project
	@echo "✅ Project created successfully"

# Test the framework integration locally
test-framework: dev-install
	@echo "🧪 Testing GoFlux framework integration..."
	@rm -rf test-framework-project || true
	@flux new test-framework-project
	@cd test-framework-project && \
		echo "" >> go.mod && \
		echo "replace github.com/barisgit/goflux => $(PWD)" >> go.mod && \
		go mod tidy

# Development workflow
dev-workflow: clean test-project
	@echo "🎉 Full development workflow complete!"

# Show current installation status
status:
	@echo "📊 GoFlux Installation Status:"
	@echo ""
	@if [ -f ~/bin/flux ]; then \
		echo "✅ Development mode: ~/bin/flux"; \
		echo "   Source: $$(head -2 ~/bin/flux | tail -1 | cut -d' ' -f2)"; \
	else \
		echo "❌ Development mode: Not installed"; \
	fi
	@echo ""
	@if command -v flux >/dev/null 2>&1; then \
		echo "✅ System installation: $$(which flux)"; \
	else \
		echo "❌ System installation: Not found"; \
	fi

# Release management
release-dry-run:
	@echo "🧪 Testing release process..."
	@if ! command -v goreleaser >/dev/null 2>&1; then \
		echo "❌ goreleaser not found. Install with:"; \
		echo "  go install github.com/goreleaser/goreleaser@latest"; \
		exit 1; \
	fi
	@goreleaser release --snapshot --clean
	@echo "✅ Release dry run completed successfully"

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
	@echo "Creating release for version: $(VERSION)"
	@git tag $(VERSION)
	@git push origin $(VERSION)
	@echo "✅ Release $(VERSION) triggered"

tag:
	@if [ -z "$(VERSION)" ]; then \
		echo "❌ VERSION required. Usage: make tag VERSION=v1.0.0"; \
		exit 1; \
	fi
	@echo "🏷️  Creating tag $(VERSION)..."
	@git tag $(VERSION)
	@echo "✅ Tag $(VERSION) created. Push with: git push origin $(VERSION)"

# Install goreleaser for releases
install-goreleaser:
	@echo "📦 Installing goreleaser..."
	@go install github.com/goreleaser/goreleaser@latest
	@echo "✅ goreleaser installed"

# Development helpers for CLI module
cli-deps:
	@echo "📦 Installing CLI dependencies..."
	@cd cli && go mod tidy
	@echo "✅ CLI dependencies updated"

cli-test:
	@echo "🧪 Running CLI tests only..."
	@cd cli && go test ./...
	@echo "✅ CLI tests complete"

cli-build:
	@echo "🔨 Building CLI binary only..."
	@cd cli && go build -o flux .
	@echo "✅ CLI binary built: cli/flux" 