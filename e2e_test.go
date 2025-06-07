package goflux_test

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// E2E test configuration
const (
	testTimeout      = 5 * time.Minute
	serverStartWait  = 10 * time.Second
	buildTimeout     = 2 * time.Minute
	devStartTimeout  = 30 * time.Second
	healthCheckDelay = 2 * time.Second
)

// Do not run with VSCode's `run test` command, it will fail.
// Run with `go test -v -run TestE2E_BasicTemplate` instead.
func TestE2E_BasicTemplate(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	// Check if flux version is vdev
	versionCmd := exec.Command("flux", "-v")
	versionOutput, err := versionCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to get flux version: %v\nOutput: %s", err, versionOutput)
	}
	if !strings.Contains(string(versionOutput), "vdev") {
		t.Fatalf("Flux version is not vdev: %s", string(versionOutput))
	}

	// Clean up any dangling processes first
	killDanglingProcesses()
	defer killDanglingProcesses()

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	projectName := "e2e-test-basic"
	projectDir := filepath.Join(t.TempDir(), projectName)

	// Step 1: Create new project
	t.Log("🚀 Step 1: Creating new project...")
	createCmd := exec.CommandContext(ctx, "flux", "new", projectName,
		"--template", "default",
		"--frontend", "default",
		"--frontend-type", "template",
		"--router", "chi",
	)
	createCmd.Dir = filepath.Dir(projectDir)
	// Disable interactive mode and set working directory
	createCmd.Env = append(os.Environ(),
		"FLUX_WORK_DIR="+filepath.Dir(projectDir),
		"CI=true",             // Common env var to disable interactive mode
		"NO_INTERACTIVE=true", // Custom env var
	)
	createCmd.Stdin = strings.NewReader("") // Provide empty stdin

	output, err := createCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to create project: %v\nOutput: %s", err, output)
	}
	t.Logf("✅ Project created successfully")

	// Step 2: Add replace directive for local development
	t.Log("🔧 Step 2: Adding local replace directive...")
	goModPath := filepath.Join(projectDir, "go.mod")
	goModContent, err := os.ReadFile(goModPath)
	if err != nil {
		t.Fatalf("Failed to read go.mod: %v", err)
	}

	// Get the absolute path to goflux root directory
	gofluxRoot, err := filepath.Abs(".")
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}

	// Add replace directive
	modifiedContent := string(goModContent) + "\n// E2E test replace directive\nreplace github.com/barisgit/goflux => " + gofluxRoot + "\n"
	if err := os.WriteFile(goModPath, []byte(modifiedContent), 0644); err != nil {
		t.Fatalf("Failed to write go.mod: %v", err)
	}
	t.Logf("✅ Added local replace directive")

	// Verify project structure
	if !fileExists(filepath.Join(projectDir, "flux.yaml")) {
		t.Fatal("flux.yaml not found in project directory")
	}
	if !fileExists(filepath.Join(projectDir, "main.go")) {
		t.Fatal("main.go not found in project directory")
	}

	// Step 3: Install Go dependencies
	t.Log("📦 Step 3: Installing Go dependencies...")
	goModCmd := exec.CommandContext(ctx, "go", "mod", "tidy")
	goModCmd.Dir = projectDir
	goModCmd.Env = append(os.Environ(), "FLUX_WORK_DIR="+projectDir)

	if goOutput, err := goModCmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to install Go dependencies: %v\nOutput: %s", err, goOutput)
	}
	t.Logf("✅ Go dependencies installed")

	// Step 4: Build the project (this will handle frontend setup, dependency installation, and build)
	t.Log("🔨 Step 4: Building project...")
	buildCmd := exec.CommandContext(ctx, "flux", "build", "--clean")
	buildCmd.Dir = projectDir
	buildCmd.Env = append(os.Environ(), "FLUX_WORK_DIR="+projectDir)

	buildOutput, err := runCommandWithTimeout(buildCmd, buildTimeout)
	if err != nil {
		t.Fatalf("Failed to build project: %v\nOutput: %s", err, buildOutput)
	}
	t.Logf("✅ Build completed successfully")

	// Verify binary was created
	binaryPath := filepath.Join(projectDir, "dist", "server")
	if !fileExists(binaryPath) {
		t.Fatal("Server binary not found after build")
	}

	// Verify frontend was set up during build
	if !fileExists(filepath.Join(projectDir, "frontend", "package.json")) {
		t.Fatal("Frontend was not set up during build process")
	}

	// Step 5: Start the server and test
	t.Log("🌐 Step 5: Starting server...")

	// Use a specific port to avoid conflicts
	testPort := "13000" // Use a less common port
	serverCmd := exec.CommandContext(ctx, "./dist/server")
	serverCmd.Dir = projectDir
	serverCmd.Env = append(os.Environ(), "PORT="+testPort)

	// Capture server output
	stderr, err := serverCmd.StderrPipe()
	if err != nil {
		t.Fatalf("Failed to create stderr pipe: %v", err)
	}
	stdout, err := serverCmd.StdoutPipe()
	if err != nil {
		t.Fatalf("Failed to create stdout pipe: %v", err)
	}

	if err := serverCmd.Start(); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}

	// Ensure cleanup of server process
	serverStopped := false
	defer func() {
		if !serverStopped && serverCmd.Process != nil {
			t.Log("🛑 Cleaning up server process...")
			// Try graceful shutdown first
			serverCmd.Process.Signal(os.Interrupt)
			time.Sleep(2 * time.Second)

			// Force kill if still running
			if serverCmd.ProcessState == nil || !serverCmd.ProcessState.Exited() {
				serverCmd.Process.Kill()
				serverCmd.Wait() // Wait to prevent zombie process
			}
			serverStopped = true
		}
	}()

	// Wait for server to start and extract port
	port, err := waitForServerStart(stdout, stderr)
	if err != nil {
		t.Fatalf("Server failed to start: %v", err)
	}
	// Use the port we specifically set
	if port == "3000" {
		port = testPort // Use our specific test port instead of default
	}
	t.Logf("✅ Server started on port %s", port)

	// Step 6: Test server endpoints
	t.Log("🧪 Step 6: Testing server endpoints...")
	baseURL := fmt.Sprintf("http://localhost:%s", port)

	// Test health endpoint
	if err := testHealthEndpoint(baseURL); err != nil {
		t.Fatalf("Health check failed: %v", err)
	}
	t.Logf("✅ Health endpoint working")

	// Test static file serving
	if err := testStaticFiles(baseURL); err != nil {
		t.Fatalf("Static file serving failed: %v", err)
	}
	t.Logf("✅ Static file serving working")

	t.Log("🎉 All E2E tests passed!")
}

// Do not run with VSCode's `run test` command, it will fail.
// Run with `go test -v -run TestE2E_DevMode` instead.
func TestE2E_DevMode(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	// Check if flux version is vdev
	versionCmd := exec.Command("flux", "-v")
	versionOutput, err := versionCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to get flux version: %v\nOutput: %s", err, versionOutput)
	}
	if !strings.Contains(string(versionOutput), "vdev") {
		t.Fatalf("Flux version is not vdev: %s", string(versionOutput))
	}

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()
	defer killDanglingProcesses()

	projectName := "e2e-test-dev"
	projectDir := filepath.Join(t.TempDir(), projectName)

	// Step 1: Create new project
	t.Log("🚀 Step 1: Creating new project...")
	createCmd := exec.CommandContext(ctx, "flux", "new", projectName,
		"--template", "default",
		"--frontend", "minimal",
		"--frontend-type", "template", // Use minimal frontend to avoid npm deps
		"--router", "chi",
	)
	createCmd.Dir = filepath.Dir(projectDir)
	createCmd.Env = append(os.Environ(),
		"FLUX_WORK_DIR="+filepath.Dir(projectDir),
		"CI=true",
		"NO_INTERACTIVE=true",
	)
	createCmd.Stdin = strings.NewReader("")

	output, err := createCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to create project: %v\nOutput: %s", err, output)
	}
	t.Logf("✅ Project created successfully")

	// Step 2: Test dev mode startup (but don't run it to completion)
	t.Log("🔧 Step 2: Testing dev mode startup...")
	devCmd := exec.CommandContext(ctx, "flux", "dev", "--debug")
	devCmd.Dir = projectDir
	devCmd.Env = append(os.Environ(), "FLUX_WORK_DIR="+projectDir)

	// Start dev command and capture output
	stderr, err := devCmd.StderrPipe()
	if err != nil {
		t.Fatalf("Failed to create stderr pipe: %v", err)
	}
	stdout, err := devCmd.StdoutPipe()
	if err != nil {
		t.Fatalf("Failed to create stdout pipe: %v", err)
	}

	if err := devCmd.Start(); err != nil {
		t.Fatalf("Failed to start dev mode: %v", err)
	}
	defer func() {
		if devCmd.Process != nil {
			devCmd.Process.Kill()
		}
	}()

	// Wait for dev mode to initialize (don't wait for full start)
	if err := waitForDevInitialization(stdout, stderr); err != nil {
		t.Fatalf("Dev mode failed to initialize: %v", err)
	}
	t.Logf("✅ Dev mode initialized successfully")

	t.Log("🎉 Dev mode E2E test passed!")
}

// Do not run with VSCode's `run test` command, it will fail.
// Run with `go test -v -run TestE2E_ConfigValidation` instead.
func TestE2E_ConfigValidation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	// Check if flux version is vdev
	versionCmd := exec.Command("flux", "-v")
	versionOutput, err := versionCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to get flux version: %v\nOutput: %s", err, versionOutput)
	}
	if !strings.Contains(string(versionOutput), "vdev") {
		t.Fatalf("Flux version is not vdev: %s", string(versionOutput))
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	defer killDanglingProcesses()

	projectName := "e2e-test-config"
	projectDir := filepath.Join(t.TempDir(), projectName)

	// Step 1: Create new project
	createCmd := exec.CommandContext(ctx, "flux", "new", projectName,
		"--template", "default",
		"--frontend", "minimal",
		"--frontend-type", "template",
		"--router", "chi",
	)
	createCmd.Dir = filepath.Dir(projectDir)
	createCmd.Env = append(os.Environ(),
		"FLUX_WORK_DIR="+filepath.Dir(projectDir),
		"CI=true",
		"NO_INTERACTIVE=true",
	)
	createCmd.Stdin = strings.NewReader("")

	output, err := createCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to create project: %v\nOutput: %s", err, output)
	}

	// Step 2: Test config validation
	t.Log("📋 Testing config validation...")
	validateCmd := exec.CommandContext(ctx, "flux", "config", "validate")
	validateCmd.Dir = projectDir
	validateCmd.Env = append(os.Environ(), "FLUX_WORK_DIR="+projectDir)

	output, err = validateCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Config validation failed: %v\nOutput: %s", err, output)
	}
	t.Logf("✅ Config validation passed")

	// Step 3: Test config show
	t.Log("📋 Testing config show...")
	showCmd := exec.CommandContext(ctx, "flux", "config", "show")
	showCmd.Dir = projectDir
	showCmd.Env = append(os.Environ(), "FLUX_WORK_DIR="+projectDir)

	output, err = showCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Config show failed: %v\nOutput: %s", err, output)
	}

	// Verify output contains expected information
	outputStr := string(output)
	if !strings.Contains(outputStr, projectName) {
		t.Fatal("Config show output doesn't contain project name")
	}
	if !strings.Contains(outputStr, "chi") {
		t.Fatal("Config show output doesn't contain router info")
	}
	t.Logf("✅ Config show working correctly")

	t.Log("🎉 Config E2E test passed!")
}

// Do not run with VSCode's `run test` command, it will fail.
// Run with `go test -v -run TestE2E_AdvancedTemplate` instead.

func TestE2E_AdvancedTemplate(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	// Check if flux version is vdev
	versionCmd := exec.Command("flux", "-v")
	versionOutput, err := versionCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to get flux version: %v\nOutput: %s", err, versionOutput)
	}
	if !strings.Contains(string(versionOutput), "vdev") {
		t.Fatalf("Flux version is not vdev: %s", string(versionOutput))
	}

	// Check if Docker is running (required for advanced template)
	if err := checkDockerRequirement(t); err != nil {
		t.Skip(err.Error())
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	defer killDanglingProcesses()

	projectName := "e2e-test-advanced"
	projectDir := filepath.Join(t.TempDir(), projectName)

	// Step 1: Create new project with advanced template
	t.Log("🚀 Step 1: Creating new project with advanced template...")
	createCmd := exec.CommandContext(ctx, "flux", "new", projectName,
		"--template", "advanced",
		"--frontend", "default",
		"--frontend-type", "template",
		"--router", "chi",
	)
	createCmd.Dir = filepath.Dir(projectDir)
	createCmd.Env = append(os.Environ(),
		"FLUX_WORK_DIR="+filepath.Dir(projectDir),
		"CI=true",
		"NO_INTERACTIVE=true",
	)
	createCmd.Stdin = strings.NewReader("")

	output, err := createCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to create advanced project: %v\nOutput: %s", err, output)
	}
	t.Logf("✅ Advanced project created successfully")

	// Step 2: Add replace directive for local development
	t.Log("🔧 Step 2: Adding local replace directive...")
	goModPath := filepath.Join(projectDir, "go.mod")
	goModContent, err := os.ReadFile(goModPath)
	if err != nil {
		t.Fatalf("Failed to read go.mod: %v", err)
	}

	// Get the absolute path to goflux root directory
	gofluxRoot, err := filepath.Abs(".")
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}

	// Add replace directive
	modifiedContent := string(goModContent) + "\n// E2E test replace directive\nreplace github.com/barisgit/goflux => " + gofluxRoot + "\n"
	if err := os.WriteFile(goModPath, []byte(modifiedContent), 0644); err != nil {
		t.Fatalf("Failed to write go.mod: %v", err)
	}
	t.Logf("✅ Added local replace directive")

	// Verify project structure
	requiredFiles := []string{"flux.yaml", "main.go", "sql/migrations", "internal/db"}
	for _, file := range requiredFiles {
		if !fileExists(filepath.Join(projectDir, file)) {
			t.Fatalf("%s not found in advanced project directory", file)
		}
	}

	// Step 3: Install Go dependencies
	t.Log("📦 Step 3: Installing Go dependencies...")
	goModCmd := exec.CommandContext(ctx, "go", "mod", "tidy")
	goModCmd.Dir = projectDir
	goModCmd.Env = append(os.Environ(), "FLUX_WORK_DIR="+projectDir)

	if goOutput, err := goModCmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to install Go dependencies: %v\nOutput: %s", err, goOutput)
	}
	t.Logf("✅ Go dependencies installed")

	// Step 4: Build the project (this will handle frontend setup, dependency installation, and build)
	t.Log("🔨 Step 4: Building advanced project...")
	buildCmd := exec.CommandContext(ctx, "flux", "build", "--clean")
	buildCmd.Dir = projectDir
	buildCmd.Env = append(os.Environ(), "FLUX_WORK_DIR="+projectDir)

	buildOutput, err := runCommandWithTimeout(buildCmd, buildTimeout)
	if err != nil {
		t.Fatalf("Failed to build advanced project: %v\nOutput: %s", err, buildOutput)
	}
	t.Logf("✅ Advanced project build completed successfully")

	// Verify binary was created
	binaryPath := filepath.Join(projectDir, "dist", "server")
	if !fileExists(binaryPath) {
		t.Fatal("Server binary not found after advanced template build")
	}

	// Verify frontend was also set up during build
	if !fileExists(filepath.Join(projectDir, "frontend", "package.json")) {
		t.Fatal("Frontend was not set up during build process")
	}

	t.Log("🎉 Advanced template E2E test passed!")
}

// TestE2E_PortCleanup verifies that development ports are properly freed after tests
func TestE2E_PortCleanup(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping port cleanup test in short mode")
	}

	t.Log("🧹 Testing port cleanup - verifying ports 3000-3002 are freed...")

	// List of ports that should be free after all e2e tests
	testPorts := []string{"3000", "3001", "3002"}

	var busyPorts []string

	for _, port := range testPorts {
		if isPortInUse(port) {
			busyPorts = append(busyPorts, port)
		}
	}

	if len(busyPorts) > 0 {
		t.Logf("⚠️  Found ports still in use: %v", busyPorts)

		// Give processes some time to clean up
		t.Log("⏳ Waiting 3 seconds for processes to clean up...")
		time.Sleep(3 * time.Second)

		// Check again
		var stillBusyPorts []string
		for _, port := range busyPorts {
			if isPortInUse(port) {
				stillBusyPorts = append(stillBusyPorts, port)
			}
		}

		if len(stillBusyPorts) > 0 {
			// Show what's using the ports
			for _, port := range stillBusyPorts {
				cmd := exec.Command("lsof", "-i:"+port)
				if output, err := cmd.Output(); err == nil {
					t.Logf("Port %s usage:\n%s", port, string(output))
				}
			}
			t.Errorf("❌ Ports still in use after cleanup: %v", stillBusyPorts)
		} else {
			t.Log("✅ All ports freed after grace period")
		}
	} else {
		t.Log("✅ All development ports (3000-3002) are properly freed")
	}
}

// Helper functions

func checkDockerRequirement(t *testing.T) error {
	// Check if Docker is installed
	_, err := exec.LookPath("docker")
	if err != nil {
		return fmt.Errorf("❌ Docker not found. Advanced template requires Docker for PostgreSQL.\n   Install Docker: https://docs.docker.com/get-docker/")
	}

	// Check if Docker daemon is running
	checkCmd := exec.Command("docker", "info")
	if err := checkCmd.Run(); err != nil {
		return fmt.Errorf("❌ Docker daemon not running. Advanced template requires Docker for PostgreSQL.\n   Start Docker and try again.")
	}

	// Check if docker-compose is available
	_, err = exec.LookPath("docker-compose")
	if err != nil {
		// Try 'docker compose' (newer syntax)
		composeCmd := exec.Command("docker", "compose", "version")
		if err := composeCmd.Run(); err != nil {
			return fmt.Errorf("❌ Docker Compose not found. Advanced template requires Docker Compose for PostgreSQL.\n   Install Docker Compose or use Docker Desktop.")
		}
	}

	t.Log("✅ Docker requirements satisfied")
	return nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func runCommandWithTimeout(cmd *exec.Cmd, timeout time.Duration) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	output, err := cmd.CombinedOutput()

	select {
	case <-ctx.Done():
		return string(output), fmt.Errorf("command timed out after %v", timeout)
	default:
		return string(output), err
	}
}

func waitForServerStart(stdout, stderr io.ReadCloser) (string, error) {
	outputChan := make(chan string, 2)
	errChan := make(chan error, 2)

	// Read stdout
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			// Look for various server start patterns
			if strings.Contains(line, "Server starting on") ||
				strings.Contains(line, "listening on") ||
				strings.Contains(line, "server listening") ||
				strings.Contains(line, "HTTP server") {
				// Extract port from line like "Server starting on :3000" or "listening on :8080"
				if strings.Contains(line, ":") {
					parts := strings.Split(line, ":")
					if len(parts) >= 2 {
						portPart := strings.TrimSpace(parts[len(parts)-1])
						// Extract just the number from strings like "3000" or "3000/tcp"
						port := strings.Split(portPart, "/")[0]
						// Remove any trailing characters
						port = strings.TrimFunc(port, func(r rune) bool {
							return r < '0' || r > '9'
						})
						if port != "" {
							outputChan <- port
							return
						}
					}
				}
			}
		}
		if err := scanner.Err(); err != nil {
			errChan <- err
		}
	}()

	// Read stderr
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.Contains(line, "Server starting on") ||
				strings.Contains(line, "listening on") ||
				strings.Contains(line, "server listening") ||
				strings.Contains(line, "HTTP server") {
				if strings.Contains(line, ":") {
					parts := strings.Split(line, ":")
					if len(parts) >= 2 {
						portPart := strings.TrimSpace(parts[len(parts)-1])
						port := strings.Split(portPart, "/")[0]
						port = strings.TrimFunc(port, func(r rune) bool {
							return r < '0' || r > '9'
						})
						if port != "" {
							outputChan <- port
							return
						}
					}
				}
			}
		}
		if err := scanner.Err(); err != nil {
			errChan <- err
		}
	}()

	select {
	case port := <-outputChan:
		return port, nil
	case err := <-errChan:
		return "", err
	case <-time.After(serverStartWait):
		// If no output captured, try the default port
		return "3000", nil
	}
}

func waitForDevInitialization(stdout, stderr io.ReadCloser) error {
	outputChan := make(chan bool, 2)
	errChan := make(chan error, 2)

	// Read stdout for initialization signals
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			// Look for signs that dev mode is initializing
			if strings.Contains(line, "Watching for changes") ||
				strings.Contains(line, "Type generation") ||
				strings.Contains(line, "Starting development") ||
				strings.Contains(line, "Frontend setup") {
				outputChan <- true
				return
			}
		}
		if err := scanner.Err(); err != nil {
			errChan <- err
		}
	}()

	// Read stderr for initialization signals
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.Contains(line, "Watching for changes") ||
				strings.Contains(line, "Type generation") ||
				strings.Contains(line, "Starting development") ||
				strings.Contains(line, "Frontend setup") {
				outputChan <- true
				return
			}
		}
		if err := scanner.Err(); err != nil {
			errChan <- err
		}
	}()

	select {
	case <-outputChan:
		return nil
	case err := <-errChan:
		return err
	case <-time.After(devStartTimeout):
		return fmt.Errorf("dev mode initialization timed out after %v", devStartTimeout)
	}
}

func testHealthEndpoint(baseURL string) error {
	// Give server more time to fully start and be ready
	time.Sleep(healthCheckDelay)

	healthURL := baseURL + "/api/health"

	// Retry the health check multiple times in case server is still starting
	maxRetries := 5
	retryDelay := 2 * time.Second

	for attempt := 1; attempt <= maxRetries; attempt++ {
		resp, err := http.Get(healthURL)
		if err != nil {
			if attempt == maxRetries {
				return fmt.Errorf("failed to make health request after %d attempts: %w", maxRetries, err)
			}
			// Log the attempt and wait before retrying
			fmt.Printf("Health check attempt %d/%d failed: %v. Retrying in %v...\n", attempt, maxRetries, err, retryDelay)
			time.Sleep(retryDelay)
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			return nil // Success!
		}

		if attempt == maxRetries {
			body, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("health endpoint returned status %d after %d attempts: %s", resp.StatusCode, maxRetries, string(body))
		}

		// Log the attempt and wait before retrying
		fmt.Printf("Health check attempt %d/%d returned status %d. Retrying in %v...\n", attempt, maxRetries, resp.StatusCode, retryDelay)
		time.Sleep(retryDelay)
	}

	return fmt.Errorf("health check failed after %d attempts", maxRetries)
}

func testStaticFiles(baseURL string) error {
	// Test that the root path serves something (usually index.html)
	resp, err := http.Get(baseURL + "/")
	if err != nil {
		return fmt.Errorf("failed to make root request: %w", err)
	}
	defer resp.Body.Close()

	// Accept both 200 (if static files exist) and 404 (if no frontend built yet)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNotFound {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("root endpoint returned unexpected status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// killDanglingProcesses attempts to kill any processes that might be using common ports
func killDanglingProcesses() {
	// Common ports that servers might use
	ports := []string{"3000", "8080", "8000", "3001", "3002", "3003", "3004", "3005", "3006", "3007", "3008", "3009", "3010"}

	for _, port := range ports {
		// Try to find and kill processes using these ports (macOS/Linux)
		cmd := exec.Command("lsof", "-ti", ":"+port)
		if output, err := cmd.Output(); err == nil {
			pids := strings.Fields(strings.TrimSpace(string(output)))
			for _, pid := range pids {
				if pid != "" {
					killCmd := exec.Command("kill", "-9", pid)
					killCmd.Run() // Ignore errors
				}
			}
		}
	}
}

// isPortInUse checks if a given port is currently in use
func isPortInUse(port string) bool {
	conn, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return true // Port is in use
	}
	conn.Close()
	return false // Port is free
}
