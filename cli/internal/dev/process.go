package dev

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/barisgit/goflux/cli/internal/typegen/analyzer"
	"github.com/barisgit/goflux/cli/internal/typegen/generator"
	"github.com/barisgit/goflux/config"

	"github.com/creack/pty"
)

func (o *DevOrchestrator) checkPort(port string) bool {
	conn, err := net.DialTimeout("tcp", "localhost:"+port, time.Second)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

func (o *DevOrchestrator) waitForPort(port string, maxWait time.Duration) bool {
	start := time.Now()
	for time.Since(start) < maxWait {
		if o.checkPort(port) {
			return true
		}
		time.Sleep(500 * time.Millisecond)
	}
	return false
}

func (o *DevOrchestrator) startBackendProcess() error {
	o.backendMutex.Lock()
	defer o.backendMutex.Unlock()

	// Stop existing process if running
	if o.backendProcess != nil && o.backendProcess.Process != nil {
		o.log("🔄 Stopping existing backend process...", "\x1b[33m")
		o.stopBackendProcessUnsafe()
	}

	o.log("🚀 Starting backend server...", "\x1b[34m")

	// Determine the main.go path (root first, then cmd/server for backward compatibility)
	var mainPath string
	if _, err := os.Stat("main.go"); err == nil {
		mainPath = "."
	} else if _, err := os.Stat("cmd/server/main.go"); err == nil {
		mainPath = "./cmd/server"
	} else {
		return fmt.Errorf("no main.go found in project root or cmd/server/")
	}

	// Build command arguments - always add --dev flag in development mode
	args := []string{"run", mainPath, "--dev"}

	cmd := exec.Command("go", args...)
	cmd.Env = append(os.Environ(), fmt.Sprintf("PORT=%d", o.backendPort))

	// Use PTY for colored output
	ptmx, err := pty.Start(cmd)
	if err != nil {
		return fmt.Errorf("failed to start backend with PTY: %w", err)
	}

	o.backendProcess = cmd
	o.log(fmt.Sprintf("✅ Backend started (PID: %d)", cmd.Process.Pid), "\x1b[34m")

	// Enable capture mode
	o.captureBackendLogs = true
	startupLogs := make([]string, 0)

	// Monitor process exit in a separate goroutine
	go func() {
		err := cmd.Wait()
		o.backendMutex.Lock()
		defer o.backendMutex.Unlock()

		if err != nil {
			o.log("❌ Backend process exited with error", "\x1b[31m")

			// If we have captured logs, display them immediately
			if len(startupLogs) > 0 {
				o.log("📋 Backend error output:", "\x1b[31m")
				for _, line := range startupLogs {
					o.formatLog("Backend", line, "\x1b[31m")
				}
			} else {
				o.log("💭 No error output captured (process may have exited too quickly)", "\x1b[33m")
			}

			// Additional error details
			if exitError, ok := err.(*exec.ExitError); ok {
				o.log(fmt.Sprintf("💥 Exit code: %d", exitError.ExitCode()), "\x1b[31m")
			}
		}
		o.backendProcess = nil
	}()

	// Handle PTY output with immediate error display
	go func() {
		defer ptmx.Close()
		scanner := bufio.NewScanner(ptmx)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.TrimSpace(line) != "" {
				o.backendMutex.Lock()
				if o.captureBackendLogs {
					// Capture logs during startup
					startupLogs = append(startupLogs, line)
					o.backendStartupLogs = startupLogs

					// If this looks like an error, display it immediately
					lineLC := strings.ToLower(line)
					if strings.Contains(lineLC, "error") ||
						strings.Contains(lineLC, "panic") ||
						strings.Contains(lineLC, "fatal") ||
						strings.Contains(lineLC, "build failed") ||
						strings.Contains(lineLC, "cannot") ||
						strings.Contains(lineLC, "undefined") {
						o.formatLog("Backend", line, "\x1b[31m")
					} else if o.debug {
						// In debug mode, show all output immediately
						o.formatLog("Backend", line, "\x1b[34m")
					}
				} else {
					// Normal logging after startup
					o.formatLog("Backend", line, "\x1b[34m")
				}
				o.backendMutex.Unlock()
			}
		}
	}()

	// Give the process a moment to potentially fail fast
	time.Sleep(100 * time.Millisecond)

	// Check if process is still running after initial startup
	if o.backendProcess == nil || o.backendProcess.Process == nil {
		return fmt.Errorf("backend process exited immediately after startup")
	}

	// Check if process is still alive
	if err := o.backendProcess.Process.Signal(syscall.Signal(0)); err != nil {
		return fmt.Errorf("backend process is not running: %w", err)
	}

	return nil
}

func (o *DevOrchestrator) restartBackend() error {
	o.log("🔄 Restarting backend...", "\x1b[36m")
	return o.startBackendProcess()
}

func (o *DevOrchestrator) fetchAndSaveOpenAPISpec() error {
	o.log("📋 Generating OpenAPI specification directly...", "\x1b[36m")

	// Create build directory if it doesn't exist
	buildDir := "build"
	if err := os.MkdirAll(buildDir, 0755); err != nil {
		return fmt.Errorf("failed to create build directory: %w", err)
	}

	// Determine the main.go path (root first, then cmd/server for backward compatibility)
	var mainPath string
	if _, err := os.Stat("main.go"); err == nil {
		mainPath = "."
	} else if _, err := os.Stat("cmd/server/main.go"); err == nil {
		mainPath = "./cmd/server"
	} else {
		return fmt.Errorf("no main.go found in project root or cmd/server/")
	}

	// Generate OpenAPI spec using the built-in command
	outputPath := filepath.Join(buildDir, "openapi.json")
	cmd := exec.Command("go", "run", mainPath, "openapi", "-o", outputPath)

	// Suppress output to avoid duplicate warnings and noise
	cmd.Stdout = nil
	cmd.Stderr = nil

	if err := cmd.Run(); err != nil {
		o.log("⚠️  Warning: Could not generate OpenAPI spec directly", "\x1b[33m")
		if o.debug {
			o.log(fmt.Sprintf("OpenAPI generation error: %v", err), "\x1b[31m")
		}
		return fmt.Errorf("failed to generate OpenAPI spec: %w", err)
	}

	o.log(fmt.Sprintf("✅ OpenAPI spec saved to %s", outputPath), "\x1b[32m")
	return nil
}

func (o *DevOrchestrator) generateTypes() error {
	o.log("🔧 Generating API types...", "\x1b[36m")

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

func (o *DevOrchestrator) findFreePort(startPort int) int {
	for port := startPort; port < startPort+100; port++ {
		if !o.checkPort(fmt.Sprintf("%d", port)) {
			return port
		}
	}
	// Fallback to a random high port if we can't find one in range
	return startPort + 1000
}

// StopCapturingStartupLogs stops capturing and enables normal logging
func (o *DevOrchestrator) StopCapturingStartupLogs() {
	o.backendMutex.Lock()
	defer o.backendMutex.Unlock()
	o.captureBackendLogs = false
}

// ReplayBackendStartupLogs replays the captured startup logs
func (o *DevOrchestrator) ReplayBackendStartupLogs() {
	if len(o.backendStartupLogs) > 0 {
		for _, line := range o.backendStartupLogs {
			o.formatLog("Backend", line, "\x1b[34m")
		}
		// Clear the logs after replaying
		o.backendStartupLogs = nil
	}
}
