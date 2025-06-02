package dev

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"sync"

	"github.com/barisgit/goflux/cli/internal/typegen/analyzer"
	"github.com/barisgit/goflux/cli/internal/typegen/generator"
	"github.com/barisgit/goflux/config"
	"github.com/creack/pty"
)

// ProcessManager handles reliable process lifecycle management
type ProcessManager struct {
	orchestrator *DevOrchestrator
}

func (o *DevOrchestrator) newProcessManager() *ProcessManager {
	return &ProcessManager{orchestrator: o}
}

// startBackend starts the backend process with proper lifecycle management
func (pm *ProcessManager) startBackend() error {
	pm.orchestrator.processMutex.Lock()
	defer pm.orchestrator.processMutex.Unlock()

	o := pm.orchestrator

	// Stop existing backend if running
	if o.backendProcess != nil {
		o.log("🔄 Stopping existing backend...", "\x1b[33m")
		if err := pm.stopBackend(5 * time.Second); err != nil {
			o.log("⚠️  Failed to stop existing backend cleanly", "\x1b[33m")
		}
	}

	// Wait for port to be free
	if !pm.waitForPortFree(o.backendPort, 5*time.Second) {
		o.log("⚠️  Port still busy, finding alternative...", "\x1b[33m")
		o.backendPort = pm.findFreePort(o.backendPort + 1)
		o.log(fmt.Sprintf("🔄 Using port %d instead", o.backendPort), "\x1b[36m")
	}

	// Determine main.go path
	mainPath := pm.findMainPath()
	if mainPath == "" {
		return fmt.Errorf("no main.go found in project root or cmd/server/")
	}

	// Create backend process
	backend := &ProcessInfo{
		Name:    "Backend",
		Command: "go",
		Args:    []string{"run", mainPath, "--dev"},
		Dir:     ".",
		Color:   "\x1b[34m",
		State:   ProcessStarting,
	}

	// Setup environment
	env := pm.createBackendEnv()

	// Create command
	cmd := exec.CommandContext(o.ctx, backend.Command, backend.Args...)
	cmd.Env = env
	cmd.Dir = backend.Dir

	// Start with PTY for better output handling
	ptmx, err := pty.Start(cmd)
	if err != nil {
		return fmt.Errorf("failed to start backend: %w", err)
	}

	backend.Process = cmd
	o.backendProcess = backend

	o.log(fmt.Sprintf("✅ Backend started (PID: %d) on port %d", cmd.Process.Pid, o.backendPort), "\x1b[34m")

	// Start output monitoring
	go pm.monitorBackendOutput(ptmx, backend)

	// Wait for startup with timeout
	return pm.waitForBackendStartup(10 * time.Second)
}

// stopBackend stops the backend process gracefully
func (pm *ProcessManager) stopBackend(timeout time.Duration) error {
	o := pm.orchestrator

	if o.backendProcess == nil || o.backendProcess.Process == nil {
		return nil
	}

	backend := o.backendProcess
	backend.State = ProcessStopping

	pid := backend.Process.Process.Pid
	o.log(fmt.Sprintf("🛑 Stopping backend (PID: %d)...", pid), "\x1b[34m")

	// Try graceful shutdown first
	if err := backend.Process.Process.Signal(syscall.SIGTERM); err != nil {
		o.log("⚠️  Failed to send SIGTERM, trying SIGKILL", "\x1b[33m")
		backend.Process.Process.Kill()
	}

	// Wait for process to exit
	done := make(chan error, 1)
	go func() {
		done <- backend.Process.Wait()
	}()

	select {
	case err := <-done:
		if err != nil {
			o.log("⚠️  Backend exited with error", "\x1b[33m")
		} else {
			o.log("✅ Backend stopped gracefully", "\x1b[32m")
		}
	case <-time.After(timeout):
		o.log("💀 Force killing backend (timeout)...", "\x1b[31m")
		backend.Process.Process.Kill()

		// Wait a bit more for force kill
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			o.log("⚠️  Process may still be running", "\x1b[33m")
		}
	}

	backend.State = ProcessStopped
	o.backendProcess = nil
	return nil
}

// monitorBackendOutput handles backend output and startup detection
func (pm *ProcessManager) monitorBackendOutput(ptmx *os.File, backend *ProcessInfo) {
	defer ptmx.Close()

	o := pm.orchestrator
	scanner := bufio.NewScanner(ptmx)

	for scanner.Scan() {
		select {
		case <-o.ctx.Done():
			return
		default:
		}

		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}

		// Capture startup logs if needed
		o.logMutex.Lock()
		if o.captureStartup && len(o.startupLogs) < 100 {
			o.startupLogs = append(o.startupLogs, LogEntry{
				Timestamp: time.Now(),
				Process:   "Backend",
				Message:   line,
				Color:     backend.Color,
			})
		}
		o.logMutex.Unlock()

		// Log to console (unless capturing startup)
		if !o.captureStartup || o.debug {
			o.formatLog("Backend", line, backend.Color)
		}

		// Check for startup completion
		if backend.State == ProcessStarting {
			if pm.isStartupComplete(line) {
				backend.State = ProcessRunning
				o.log("✅ Backend startup completed", "\x1b[32m")
			}
		}
	}
}

// isStartupComplete checks if a log line indicates startup completion
func (pm *ProcessManager) isStartupComplete(line string) bool {
	lineLower := strings.ToLower(line)

	// Look for server ready indicators
	return (strings.Contains(lineLower, "server") &&
		(strings.Contains(lineLower, "running") ||
			strings.Contains(lineLower, "listening") ||
			strings.Contains(lineLower, "started") ||
			strings.Contains(line, "://"))) ||
		(strings.Contains(line, "═══") && strings.Contains(line, "GoFlux"))
}

// waitForBackendStartup waits for backend to be ready
func (pm *ProcessManager) waitForBackendStartup(timeout time.Duration) error {
	o := pm.orchestrator

	start := time.Now()
	for time.Since(start) < timeout {
		select {
		case <-o.ctx.Done():
			return fmt.Errorf("context cancelled during startup")
		default:
		}

		// Check if process is still running
		if o.backendProcess == nil || o.backendProcess.Process == nil {
			return fmt.Errorf("backend process exited during startup")
		}

		// Check if process has indicated it's ready
		if o.backendProcess.State == ProcessRunning {
			return nil
		}

		// Check if port is responding
		if pm.checkPort(o.backendPort) {
			o.backendProcess.State = ProcessRunning
			o.log("✅ Backend port is responding", "\x1b[32m")
			return nil
		}

		time.Sleep(100 * time.Millisecond)
	}

	// Timeout reached - assume ready if process is still running
	if o.backendProcess != nil && o.backendProcess.Process != nil {
		if err := o.backendProcess.Process.Process.Signal(syscall.Signal(0)); err == nil {
			o.backendProcess.State = ProcessRunning
			o.log("✅ Backend startup timeout - assuming ready", "\x1b[32m")
			return nil
		}
	}

	return fmt.Errorf("backend failed to start within %v", timeout)
}

// findMainPath locates the main.go file
func (pm *ProcessManager) findMainPath() string {
	if _, err := os.Stat("main.go"); err == nil {
		return "."
	}
	if _, err := os.Stat("cmd/server/main.go"); err == nil {
		return "./cmd/server"
	}
	return ""
}

// createBackendEnv creates environment variables for backend
func (pm *ProcessManager) createBackendEnv() []string {
	o := pm.orchestrator

	// Filter out existing port variables
	env := make([]string, 0, len(os.Environ())+6)
	for _, e := range os.Environ() {
		if !strings.HasPrefix(e, "PORT=") &&
			!strings.HasPrefix(e, "BACKEND_PORT=") &&
			!strings.HasPrefix(e, "PROXY_PORT=") {
			env = append(env, e)
		}
	}

	// Add our configuration
	env = append(env,
		fmt.Sprintf("PORT=%d", o.backendPort),
		fmt.Sprintf("BACKEND_PORT=%d", o.backendPort),
		fmt.Sprintf("PROXY_PORT=%d", o.config.Port),
		fmt.Sprintf("FRONTEND_PORT=%d", o.frontendPort),
		"FLUX_DEV_MODE=true",
		"GO_ENV=development")

	return env
}

// checkPort checks if a port is in use
func (pm *ProcessManager) checkPort(port int) bool {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("localhost:%d", port), 100*time.Millisecond)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// waitForPortFree waits for a port to become available
func (pm *ProcessManager) waitForPortFree(port int, timeout time.Duration) bool {
	start := time.Now()
	for time.Since(start) < timeout {
		if !pm.checkPort(port) {
			return true
		}
		time.Sleep(50 * time.Millisecond)
	}
	return false
}

// findFreePort finds an available port starting from the given port
func (pm *ProcessManager) findFreePort(startPort int) int {
	for port := startPort; port < startPort+100; port++ {
		if !pm.checkPort(port) {
			return port
		}
	}
	return startPort + 1000 // Fallback to high port
}

// generateTypes generates API types using the project analyzer
func (o *DevOrchestrator) generateTypes() error {
	o.log("🔧 Generating API types...", "\x1b[36m")

	// Always generate a new OpenAPI spec
	if err := o.generateOpenAPIDirectly(); err != nil {
		o.log("⚠️  Warning: Could not generate OpenAPI spec", "\x1b[33m")
		if o.debug {
			o.log(fmt.Sprintf("OpenAPI generation error: %v", err), "\x1b[31m")
		}
	}

	// Load configuration using enhanced system with fallback to defaults
	cm := config.NewConfigManager(config.ConfigLoadOptions{
		Path:              "flux.yaml",
		AllowMissing:      true,  // Allow missing for type generation
		ValidateStructure: false, // Don't validate during type generation
		ApplyDefaults:     true,
		WarnOnDeprecated:  false,
		Quiet:             true, // Quiet during dev generation
	})

	projectConfig, err := cm.LoadConfig()
	if err != nil {
		o.log("⚠️  Warning: Could not read flux.yaml, using defaults", "\x1b[33m")
		if o.debug {
			o.log(fmt.Sprintf("Config read error: %v", err), "\x1b[33m")
		}
		// Use defaults if config file doesn't exist or is invalid
		projectConfig = &config.ProjectConfig{
			APIClient: config.GetDefaultAPIClientConfig(),
		}
	}

	o.log(fmt.Sprintf("Using API client generator: %s", projectConfig.APIClient.Generator), "\x1b[36m")

	// Use the new modular type generation system
	analysis, err := analyzer.AnalyzeProject(".", o.debug)
	if err != nil {
		o.log("❌ Failed to analyze project", "\x1b[31m")
		return fmt.Errorf("project analysis failed: %w", err)
	}

	// Generate all outputs concurrently
	var wg sync.WaitGroup
	errorChan := make(chan error, 3)

	// Generate TypeScript types only if needed
	if generator.ShouldGenerateTypeScriptTypes(projectConfig.APIClient.Generator) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := generator.GenerateTypeScriptTypes(analysis.TypeDefs); err != nil {
				errorChan <- fmt.Errorf("generating TypeScript types: %w", err)
			}
		}()
	}

	// Generate API client
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := generator.GenerateAPIClient(analysis.Routes, analysis.TypeDefs, &projectConfig.APIClient); err != nil {
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
		o.log("❌ Error in type generation", "\x1b[31m")
		return err
	}

	o.log("✅ API types generated successfully", "\x1b[32m")

	// Log summary
	o.log(fmt.Sprintf("Generated %d TypeScript types", len(analysis.TypeDefs)), "\x1b[36m")
	o.log(fmt.Sprintf("Generated API client with %d routes", len(analysis.Routes)), "\x1b[36m")
	if projectConfig.APIClient.Generator != "basic" {
		o.log(fmt.Sprintf("Using %s generator", projectConfig.APIClient.Generator), "\x1b[36m")
	}

	return nil
}

// generateOpenAPIDirectly generates the OpenAPI spec using the built-in command
func (o *DevOrchestrator) generateOpenAPIDirectly() error {
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

	if o.debug {
		o.log("🔧 Generating OpenAPI spec using built-in command...", "\x1b[36m")
	}

	// Generate OpenAPI spec using the built-in command
	outputPath := "build/openapi.json"
	cmd := exec.Command("go", "run", mainPath, "openapi", "-o", outputPath)

	// Capture output for debugging
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if o.debug {
			o.log(fmt.Sprintf("OpenAPI generation error: %v", err), "\x1b[31m")
			if stderr.String() != "" {
				o.log(fmt.Sprintf("Stderr: %s", stderr.String()), "\x1b[31m")
			}
		}
		return fmt.Errorf("failed to generate OpenAPI spec: %w", err)
	}

	// Log success and any output
	if o.debug && stdout.String() != "" {
		lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
		for _, line := range lines {
			if strings.TrimSpace(line) != "" {
				o.log(line, "\x1b[36m")
			}
		}
	}

	if o.debug {
		o.log("✅ OpenAPI spec generated successfully", "\x1b[32m")
	}
	return nil
}
