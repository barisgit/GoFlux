package dev

import (
	"bufio"
	"encoding/json"
	"fmt"
	"goflux/internal/typegen/analyzer"
	"goflux/internal/typegen/generator"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

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

func (o *DevOrchestrator) stopBackendProcessUnsafe() {
	if o.backendProcess == nil || o.backendProcess.Process == nil {
		return
	}

	pid := o.backendProcess.Process.Pid
	o.log(fmt.Sprintf("🛑 Stopping backend (PID: %d)...", pid), "\x1b[34m")

	// Try graceful shutdown first
	o.backendProcess.Process.Signal(syscall.SIGTERM)

	// Wait up to 3 seconds for graceful shutdown
	done := make(chan error, 1)
	go func() {
		done <- o.backendProcess.Wait()
	}()

	select {
	case err := <-done:
		if err != nil {
			o.log("⚠️  Backend exited with error", "\x1b[33m")
		} else {
			o.log("✅ Backend stopped gracefully", "\x1b[32m")
		}
	case <-time.After(3 * time.Second):
		o.log("💀 Force killing backend (timeout)...", "\x1b[31m")
		o.backendProcess.Process.Kill()
		syscall.Kill(pid, syscall.SIGKILL)
	}

	o.backendProcess = nil
}

func (o *DevOrchestrator) restartBackend() error {
	o.log("🔄 Restarting backend...", "\x1b[36m")
	return o.startBackendProcess()
}

func (o *DevOrchestrator) fetchAndSaveOpenAPISpec() error {
	o.log("📋 Fetching OpenAPI specification...", "\x1b[36m")

	// Create build directory if it doesn't exist
	buildDir := "build"
	if err := os.MkdirAll(buildDir, 0755); err != nil {
		return fmt.Errorf("failed to create build directory: %w", err)
	}

	// Fetch OpenAPI spec from the server
	openAPIURL := fmt.Sprintf("http://localhost:%d/api/openapi.json", o.backendPort)

	// Wait a bit more to ensure the server is fully ready
	time.Sleep(1 * time.Second)

	resp, err := http.Get(openAPIURL)
	if err != nil {
		return fmt.Errorf("failed to fetch OpenAPI spec from %s: %w", openAPIURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned status %d when fetching OpenAPI spec", resp.StatusCode)
	}

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read OpenAPI response: %w", err)
	}

	// Validate it's valid JSON
	var openAPISpec map[string]interface{}
	if err := json.Unmarshal(body, &openAPISpec); err != nil {
		return fmt.Errorf("invalid OpenAPI JSON received: %w", err)
	}

	// Save to build/openapi.json
	openAPIPath := filepath.Join(buildDir, "openapi.json")
	if err := os.WriteFile(openAPIPath, body, 0644); err != nil {
		return fmt.Errorf("failed to save OpenAPI spec to %s: %w", openAPIPath, err)
	}

	o.log(fmt.Sprintf("✅ OpenAPI spec saved to %s", openAPIPath), "\x1b[32m")

	// Log some info about what we found
	if info, ok := openAPISpec["info"].(map[string]interface{}); ok {
		if title, ok := info["title"].(string); ok {
			o.log(fmt.Sprintf("📖 API Title: %s", title), "\x1b[36m")
		}
		if version, ok := info["version"].(string); ok {
			o.log(fmt.Sprintf("🏷️  API Version: %s", version), "\x1b[36m")
		}
	}

	if paths, ok := openAPISpec["paths"].(map[string]interface{}); ok {
		routeCount := 0
		for path := range paths {
			if pathItem, ok := paths[path].(map[string]interface{}); ok {
				// Count HTTP methods in this path
				for method := range pathItem {
					if method == "get" || method == "post" || method == "put" || method == "delete" || method == "patch" {
						routeCount++
					}
				}
			}
		}
		o.log(fmt.Sprintf("🛣️  Found %d API routes", routeCount), "\x1b[36m")
	}

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

func (o *DevOrchestrator) forceKillPortProcesses() {
	ports := []int{o.config.Port, o.frontendPort, o.backendPort}

	for _, port := range ports {
		if port == 0 {
			continue
		}

		// Use lsof to find processes listening on the port
		cmd := exec.Command("lsof", "-ti", fmt.Sprintf(":%d", port))
		output, err := cmd.Output()
		if err != nil {
			continue // No process found on this port
		}

		pids := strings.Fields(strings.TrimSpace(string(output)))
		for _, pidStr := range pids {
			if pidStr == "" {
				continue
			}

			// Kill each PID
			killCmd := exec.Command("kill", "-9", pidStr)
			if err := killCmd.Run(); err == nil {
				o.log(fmt.Sprintf("💀 Force killed process %s on port %d", pidStr, port), "\x1b[31m")
			}
		}
	}
}
