package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/barisgit/goflux/internal/typegen/analyzer"

	"github.com/barisgit/goflux/internal/typegen/generator"

	"github.com/barisgit/goflux/internal/config"

	"github.com/spf13/cobra"
)

type BuildOrchestrator struct {
	config *config.ProjectConfig
	debug  bool
}

func BuildCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "build",
		Short: "Build production binary",
		Long:  "Build a complete production binary with embedded frontend assets",
		RunE:  runBuild,
	}

	cmd.Flags().Bool("debug", false, "Enable debug logging")
	cmd.Flags().Bool("linux", false, "Build for Linux (cross-compilation)")
	cmd.Flags().Bool("clean", true, "Clean build artifacts before building")

	return cmd
}

func runBuild(cmd *cobra.Command, args []string) error {
	// Get flags
	debug, _ := cmd.Flags().GetBool("debug")
	forLinux, _ := cmd.Flags().GetBool("linux")
	cleanFirst, _ := cmd.Flags().GetBool("clean")

	// Check if we're running in development mode with a different working directory
	workDir := os.Getenv("flux_WORK_DIR")
	if workDir != "" {
		if err := os.Chdir(workDir); err != nil {
			return fmt.Errorf("failed to change to work directory %s: %w", workDir, err)
		}
	}

	// Load configuration using enhanced system
	cm := config.NewConfigManager(config.ConfigLoadOptions{
		Path:              "flux.yaml",
		AllowMissing:      false,
		ValidateStructure: true,
		ApplyDefaults:     true,
		WarnOnDeprecated:  false, // Don't show warnings during build
		Quiet:             false,
	})

	cfg, err := cm.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	orchestrator := &BuildOrchestrator{
		config: cfg,
		debug:  debug,
	}

	return orchestrator.Build(forLinux, cleanFirst)
}

func (b *BuildOrchestrator) log(message, color string) {
	timestamp := time.Now().Format("15:04:05")
	if color == "" {
		color = "\x1b[0m"
	}
	fmt.Printf("%s[%s] %s\x1b[0m\n", color, timestamp, message)
}

func (b *BuildOrchestrator) Build(forLinux, cleanFirst bool) error {
	b.log("🚀 Building complete fullstack application...", "\x1b[32m")
	fmt.Println()

	// Clean build artifacts if requested
	if cleanFirst {
		if err := b.clean(); err != nil {
			return err
		}
	}

	// Step 1: Install dependencies
	if err := b.installDependencies(); err != nil {
		return err
	}

	// Step 2: Generate TypeScript types from Go/OpenAPI
	if err := b.generateTypes(); err != nil {
		return err
	}

	// Step 3: Build frontend
	if err := b.buildFrontend(); err != nil {
		return err
	}

	// Step 4: Generate static HTML files (if supported)
	if err := b.generateStaticSiteFiles(); err != nil {
		return err
	}

	// Step 5: Build Go binary with embedded assets
	if err := b.buildGoBinary(forLinux); err != nil {
		return err
	}

	b.logBuildSuccess()
	return nil
}

func (b *BuildOrchestrator) clean() error {
	b.log("🧹 Cleaning build artifacts...", "\x1b[33m")

	// Get frontend static directory from config
	frontendStaticDir := b.config.Build.StaticDir
	if frontendStaticDir == "" {
		frontendStaticDir = "frontend/dist"
	}

	// Directories to clean
	dirsToClean := []string{
		b.config.Build.OutputDir,
		frontendStaticDir,
		"frontend/node_modules/.vite",
		"tmp",
		"build",
	}

	for _, dir := range dirsToClean {
		if err := os.RemoveAll(dir); err != nil && !os.IsNotExist(err) {
			b.log(fmt.Sprintf("⚠️  Warning: Could not clean %s: %v", dir, err), "\x1b[33m")
		}
	}

	b.log("✅ Build artifacts cleaned", "\x1b[32m")
	fmt.Println()
	return nil
}

func (b *BuildOrchestrator) installDependencies() error {
	b.log("📦 Installing dependencies...", "\x1b[36m")

	var wg sync.WaitGroup
	errorChan := make(chan error, 2)

	// Install Go dependencies
	wg.Add(1)
	go func() {
		defer wg.Done()
		cmd := exec.Command("go", "mod", "download")
		if err := cmd.Run(); err != nil {
			errorChan <- fmt.Errorf("failed to install Go dependencies: %w", err)
		}
	}()

	// Install frontend dependencies
	wg.Add(1)
	go func() {
		defer wg.Done()
		if _, err := os.Stat("frontend/package.json"); err == nil {
			cmd := exec.Command("pnpm", "install", "--silent")
			cmd.Dir = "frontend"
			if err := cmd.Run(); err != nil {
				// Fallback to npm if pnpm fails
				cmd = exec.Command("npm", "install", "--silent")
				cmd.Dir = "frontend"
				if err := cmd.Run(); err != nil {
					errorChan <- fmt.Errorf("failed to install frontend dependencies: %w", err)
				}
			}
		}
	}()

	// Wait for both installations
	go func() {
		wg.Wait()
		close(errorChan)
	}()

	for err := range errorChan {
		if err != nil {
			return err
		}
	}

	b.log("✅ Dependencies installed", "\x1b[32m")
	fmt.Println()
	return nil
}

func (b *BuildOrchestrator) generateTypes() error {
	b.log("🔧 Generating TypeScript types from Go...", "\x1b[36m")

	// First, try to generate OpenAPI spec if it doesn't exist
	if err := b.ensureOpenAPISpec(); err != nil {
		b.log("⚠️  Warning: Could not generate OpenAPI spec", "\x1b[33m")
		if b.debug {
			b.log(fmt.Sprintf("OpenAPI generation error: %v", err), "\x1b[31m")
		}
	}

	// Use the new modular type generation system
	analysis, err := analyzer.AnalyzeProject(".", b.debug)
	if err != nil {
		b.log("⚠️  Warning: Could not analyze project for type generation", "\x1b[33m")
		if b.debug {
			b.log(fmt.Sprintf("Analysis error: %v", err), "\x1b[31m")
		}
		b.log("Continuing build without type generation...", "\x1b[36m")
		fmt.Println()
		return nil
	}

	// Generate all outputs concurrently
	var wg sync.WaitGroup
	errorChan := make(chan error, 3)

	// Generate TypeScript types
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := generator.GenerateTypeScriptTypes(analysis.TypeDefs); err != nil {
			errorChan <- fmt.Errorf("generating TypeScript types: %w", err)
		}
	}()

	// Generate API client using project config
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := generator.GenerateAPIClient(analysis.Routes, analysis.TypeDefs, &b.config.APIClient); err != nil {
			errorChan <- fmt.Errorf("generating API client: %w", err)
		}
	}()

	// Generate route manifest
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := generator.GenerateRouteManifest(analysis.Routes); err != nil {
			errorChan <- fmt.Errorf("generating route manifest: %w", err)
		}
	}()

	// Wait for all generators to complete
	go func() {
		wg.Wait()
		close(errorChan)
	}()

	// Check for errors
	for err := range errorChan {
		if err != nil {
			b.log("⚠️  Warning: Error in type generation", "\x1b[33m")
			if b.debug {
				b.log(fmt.Sprintf("Type generation error: %v", err), "\x1b[31m")
			}
			b.log("Continuing build without complete type generation...", "\x1b[36m")
			fmt.Println()
			return nil
		}
	}

	b.log("✅ Types generated", "\x1b[32m")
	b.log(fmt.Sprintf("Generated %d TypeScript types", len(analysis.TypeDefs)), "\x1b[36m")
	b.log(fmt.Sprintf("Generated API client with %d routes", len(analysis.Routes)), "\x1b[36m")
	if b.config.APIClient.Generator != "basic" {
		b.log(fmt.Sprintf("Using %s generator", b.config.APIClient.Generator), "\x1b[36m")
	}
	fmt.Println()
	return nil
}

func (b *BuildOrchestrator) buildFrontend() error {
	b.log("🎨 Building frontend...", "\x1b[36m")

	// Check if frontend exists
	if _, err := os.Stat("frontend/package.json"); os.IsNotExist(err) {
		b.log("⚠️  No frontend found, skipping frontend build", "\x1b[33m")
		return nil
	}

	// Run the frontend build command
	cmd := exec.Command("sh", "-c", b.config.Frontend.BuildCmd)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("frontend build failed: %w", err)
	}

	b.log("✅ Frontend built", "\x1b[32m")
	fmt.Println()
	return nil
}

func (b *BuildOrchestrator) generateStaticSiteFiles() error {
	if !b.config.Frontend.StaticGen.Enabled {
		b.log("📄 Static site generation disabled, skipping...", "\x1b[36m")
		fmt.Println()
		return nil
	}

	b.log("📄 Generating static HTML files...", "\x1b[36m")

	// Run SSR build if specified
	if b.config.Frontend.StaticGen.BuildSSRCmd != "" {
		cmd := exec.Command("sh", "-c", b.config.Frontend.StaticGen.BuildSSRCmd)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			b.log("⚠️  SSR build failed, continuing without static generation", "\x1b[33m")
			if b.debug {
				b.log(fmt.Sprintf("SSR build error: %v", err), "\x1b[31m")
			}
			fmt.Println()
			return nil
		}
	}

	// Run static generation command if specified
	if b.config.Frontend.StaticGen.GenerateCmd != "" {
		cmd := exec.Command("sh", "-c", b.config.Frontend.StaticGen.GenerateCmd)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			b.log("⚠️  Static generation failed, continuing with SPA mode", "\x1b[33m")
			if b.debug {
				b.log(fmt.Sprintf("Static generation error: %v", err), "\x1b[31m")
			}
			fmt.Println()
			return nil
		}
	}

	b.log("✅ Static files generated", "\x1b[32m")
	fmt.Println()
	return nil
}

func (b *BuildOrchestrator) generateStaticHandler() error {
	// This method is now redundant since generateStaticFiles handles everything
	// Keeping for backward compatibility but it does nothing
	return nil
}

func (b *BuildOrchestrator) buildGoBinary(forLinux bool) error {
	b.log("🔨 Building Go binary with embedded assets...", "\x1b[36m")

	// Create output directory
	if err := os.MkdirAll(b.config.Build.OutputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Copy frontend files to cmd/server/static for embedding
	if err := b.copyFrontendForEmbedding(); err != nil {
		return err
	}

	// Determine binary name
	binaryName := b.config.Build.BinaryName
	if forLinux {
		binaryName += "-linux"
	}
	binaryPath := filepath.Join(b.config.Build.OutputDir, binaryName)

	// Build command
	args := []string{"build"}

	// Add build tags
	if b.config.Build.BuildTags != "" {
		args = append(args, "-tags", b.config.Build.BuildTags)
	}

	// Add ldflags
	ldflags := b.config.Build.LDFlags
	if ldflags != "" {
		if forLinux {
			ldflags += " -extldflags '-static'"
		}
		args = append(args, "-ldflags", ldflags)
	}

	// Add output path
	args = append(args, "-o", binaryPath)

	// Add package path (look for main.go in root first, then cmd/server/ for backward compatibility)
	packagePath := "."
	if _, err := os.Stat("main.go"); os.IsNotExist(err) {
		// Fall back to cmd/server for backward compatibility
		if _, err := os.Stat("cmd/server/main.go"); os.IsNotExist(err) {
			return fmt.Errorf("no main.go found in project root or cmd/server/")
		}
		packagePath = "./cmd/server"
	}
	args = append(args, packagePath)

	// Prepare environment variables
	env := os.Environ()

	// Set CGO_ENABLED
	cgoEnabled := "0"
	if b.config.Build.CGOEnabled {
		cgoEnabled = "1"
	}
	env = append(env, "CGO_ENABLED="+cgoEnabled)

	// Set cross-compilation environment for Linux
	if forLinux {
		env = append(env, "GOOS=linux", "GOARCH=amd64")
	}

	// Execute build command
	cmd := exec.Command("go", args...)
	cmd.Env = env
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("go build failed: %w", err)
	}

	// Clean up copied frontend files
	b.cleanupFrontendCopy()

	b.log("✅ Binary built with embedded assets", "\x1b[32m")
	fmt.Println()
	return nil
}

func (b *BuildOrchestrator) copyFrontendForEmbedding() error {
	// Use StaticDir from config instead of hardcoded path
	frontendDistPath := b.config.Build.StaticDir
	if frontendDistPath == "" {
		// Fallback to default if not specified in config
		frontendDistPath = "frontend/dist"
	}
	staticPath := "internal/static/assets"

	// Remove existing static directory if it exists
	if err := os.RemoveAll(staticPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove existing static directory: %w", err)
	}

	if _, err := os.Stat(frontendDistPath); os.IsNotExist(err) {
		b.log(fmt.Sprintf("⚠️  No %s directory found, creating placeholder for embedding", frontendDistPath), "\x1b[33m")

		// Create static directory with placeholder file so embed doesn't fail
		if err := os.MkdirAll(staticPath, 0755); err != nil {
			return fmt.Errorf("failed to create static directory: %w", err)
		}

		// Create a placeholder file
		placeholderPath := filepath.Join(staticPath, ".placeholder")
		placeholderContent := "# This is a placeholder file for Go embed directive\n# Frontend assets will be placed here during build\n"
		if err := os.WriteFile(placeholderPath, []byte(placeholderContent), 0644); err != nil {
			return fmt.Errorf("failed to create placeholder file: %w", err)
		}

		return nil
	}

	b.log("📁 Copying frontend files for embedding...", "\x1b[36m")

	// Copy configured static directory to internal/static/assets
	return b.copyDir(frontendDistPath, staticPath)
}

func (b *BuildOrchestrator) cleanupFrontendCopy() {
	staticPath := "internal/static/assets"
	if err := os.RemoveAll(staticPath); err != nil {
		b.log("⚠️  Warning: Could not clean up copied frontend files", "\x1b[33m")
	}
}

func (b *BuildOrchestrator) copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Calculate destination path
		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		destPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(destPath, info.Mode())
		}

		// Copy file
		srcFile, err := os.Open(path)
		if err != nil {
			return err
		}
		defer srcFile.Close()

		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			return err
		}

		destFile, err := os.Create(destPath)
		if err != nil {
			return err
		}
		defer destFile.Close()

		_, err = destFile.ReadFrom(srcFile)
		if err != nil {
			return err
		}

		return os.Chmod(destPath, info.Mode())
	})
}

func (b *BuildOrchestrator) logBuildSuccess() {
	binaryPath := filepath.Join(b.config.Build.OutputDir, b.config.Build.BinaryName)

	b.log("🎉 BUILD COMPLETE!", "\x1b[32m")
	b.log(fmt.Sprintf("📦 Single binary: %s", binaryPath), "\x1b[36m")

	// Get binary size if possible
	if info, err := os.Stat(binaryPath); err == nil {
		sizeInMB := float64(info.Size()) / (1024 * 1024)
		b.log(fmt.Sprintf("📊 Binary size: %.1f MB", sizeInMB), "\x1b[36m")
	}

	fmt.Println()
	b.log(fmt.Sprintf("🚀 Run with: ./%s", binaryPath), "\x1b[33m")
	b.log(fmt.Sprintf("🌐 Then visit: http://localhost:%d", b.config.Port), "\x1b[33m")
}

func (b *BuildOrchestrator) ensureOpenAPISpec() error {
	b.log("📋 Checking for OpenAPI specification...", "\x1b[36m")

	// Check if OpenAPI spec already exists in common locations
	possiblePaths := []string{
		"openapi.json",
		"build/openapi.json",
		"docs/openapi.json",
		"api/openapi.json",
	}

	for _, path := range possiblePaths {
		if _, err := os.Stat(path); err == nil {
			b.log("✅ Found existing OpenAPI spec at "+path, "\x1b[32m")
			fmt.Println()
			return nil
		}
	}

	// No existing spec found, try to generate one directly
	b.log("📋 No OpenAPI spec found, generating directly...", "\x1b[36m")

	return b.generateOpenAPIDirectly()
}

func (b *BuildOrchestrator) generateOpenAPIDirectly() error {
	// Check if we have main.go in root or cmd/server directory
	var mainPath string
	if _, err := os.Stat("main.go"); err == nil {
		mainPath = "."
	} else if _, err := os.Stat("cmd/server/main.go"); err == nil {
		mainPath = "./cmd/server"
	} else {
		return fmt.Errorf("no main.go found in project root or cmd/server/, cannot generate OpenAPI spec")
	}

	// Create build directory if it doesn't exist
	if err := os.MkdirAll("build", 0755); err != nil {
		return fmt.Errorf("failed to create build directory: %w", err)
	}

	b.log("🔧 Generating OpenAPI spec using built-in command...", "\x1b[36m")

	// Generate OpenAPI spec using the built-in command
	outputPath := "build/openapi.json"
	cmd := exec.Command("go", "run", mainPath, "openapi", "-o", outputPath)

	// Capture output for debugging
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		b.log("⚠️  Warning: Could not generate OpenAPI spec directly", "\x1b[33m")
		if b.debug {
			b.log(fmt.Sprintf("OpenAPI generation error: %v", err), "\x1b[31m")
			if stderr.String() != "" {
				b.log(fmt.Sprintf("Stderr: %s", stderr.String()), "\x1b[31m")
			}
		}
		return fmt.Errorf("failed to generate OpenAPI spec: %w", err)
	}

	// Log success and any output
	if stdout.String() != "" {
		lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
		for _, line := range lines {
			if strings.TrimSpace(line) != "" {
				b.log(line, "\x1b[36m")
			}
		}
	}

	b.log("✅ OpenAPI spec generated successfully", "\x1b[32m")
	fmt.Println()
	return nil
}
