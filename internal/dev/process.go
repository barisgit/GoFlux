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
	"time"

	"github.com/barisgit/goflux/internal/typegen/analyzer"
	"github.com/barisgit/goflux/internal/typegen/generator"

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

	// Create new process
	cmd := exec.Command("go", "run", "./cmd/server")
	cmd.Env = append(os.Environ(), fmt.Sprintf("PORT=%d", o.backendPort))

	// Use PTY for colored output
	ptmx, err := pty.Start(cmd)
	if err != nil {
		return fmt.Errorf("failed to start backend with PTY: %w", err)
	}

	o.backendProcess = cmd
	o.log(fmt.Sprintf("✅ Backend started (PID: %d)", cmd.Process.Pid), "\x1b[34m")

	// Handle PTY output
	go func() {
		defer ptmx.Close()
		scanner := bufio.NewScanner(ptmx)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.TrimSpace(line) != "" {
				o.formatLog("Backend", line, "\x1b[34m")
			}
		}
	}()

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

	// Generate OpenAPI spec using the built-in command
	outputPath := filepath.Join(buildDir, "openapi.json")
	cmd := exec.Command("go", "run", "./cmd/server", "openapi", "-o", outputPath)

	// Capture output for debugging
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		o.log("⚠️  Warning: Could not generate OpenAPI spec directly", "\x1b[33m")
		if o.debug {
			o.log(fmt.Sprintf("OpenAPI generation error: %v", err), "\x1b[31m")
			if stderr.String() != "" {
				o.log(fmt.Sprintf("Stderr: %s", stderr.String()), "\x1b[31m")
			}
		}
		return fmt.Errorf("failed to generate OpenAPI spec: %w", err)
	}

	// Log success and any output
	if stdout.String() != "" {
		lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
		for _, line := range lines {
			if strings.TrimSpace(line) != "" {
				o.log(line, "\x1b[36m")
			}
		}
	}

	o.log(fmt.Sprintf("✅ OpenAPI spec saved to %s", outputPath), "\x1b[32m")
	return nil
}

func (o *DevOrchestrator) generateTypes() error {
	o.log("🔧 Generating API types...", "\x1b[36m")

	// Use the new modular type generation system
	analysis, err := analyzer.AnalyzeProject(".", o.debug)
	if err != nil {
		o.log("❌ Failed to analyze project", "\x1b[31m")
		return fmt.Errorf("project analysis failed: %w", err)
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

	// Generate API client
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := generator.GenerateAPIClient(analysis.Routes, analysis.TypeDefs); err != nil {
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
