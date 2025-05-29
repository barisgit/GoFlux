package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"goflux/internal/typegen/analyzer"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"goflux/internal/typegen/generator"

	"github.com/creack/pty"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

type ProcessInfo struct {
	Name    string
	Process *exec.Cmd
	Command string
	Args    []string
	Dir     string
	Color   string
}

type DevOrchestrator struct {
	processes      []ProcessInfo
	isShuttingDown bool
	config         *ProjectConfig
	debug          bool
}

func DevCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dev",
		Short: "Start development server",
		Long:  "Start the Go backend and frontend development servers with hot reload",
		RunE:  runDev,
	}

	cmd.Flags().Bool("debug", false, "Enable debug logging for type generation")

	return cmd
}

func runDev(cmd *cobra.Command, args []string) error {
	// Get debug flag
	debug, _ := cmd.Flags().GetBool("debug")

	// Check if we're running in development mode with a different working directory
	workDir := os.Getenv("flux_WORK_DIR")
	if workDir != "" {
		// Change to the original working directory
		if err := os.Chdir(workDir); err != nil {
			return fmt.Errorf("failed to change to work directory %s: %w", workDir, err)
		}
	}

	// Check if we're in a GoFlux project
	configPath := "flux.yaml"
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return fmt.Errorf("flux.yaml not found. Are you in a GoFlux project directory?\nRun 'flux new <project-name>' to create a new project")
	}

	// Read config
	config, err := readConfig(configPath)
	if err != nil {
		return fmt.Errorf("failed to read flux.yaml: %w", err)
	}

	orchestrator := &DevOrchestrator{
		config: config,
		debug:  debug,
	}

	return orchestrator.Start()
}

func (o *DevOrchestrator) log(message, color string) {
	timestamp := time.Now().Format("15:04:05")
	if color == "" {
		color = "\x1b[0m"
	}
	fmt.Printf("%s[%s] %s\x1b[0m\n", color, timestamp, message)
}

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

func (o *DevOrchestrator) setupFrontendIfNeeded() error {
	frontendPath := "frontend"
	packageJsonPath := filepath.Join(frontendPath, "package.json")

	if _, err := os.Stat(packageJsonPath); os.IsNotExist(err) {
		o.log(fmt.Sprintf("📦 Setting up %s frontend for the first time...", o.config.Frontend.Framework), "\x1b[33m")

		// Create frontend directory first
		if err := os.MkdirAll(frontendPath, 0755); err != nil {
			return fmt.Errorf("failed to create frontend directory: %w", err)
		}

		// Use the install command from config
		installCmd := exec.Command("sh", "-c", o.config.Frontend.InstallCmd)
		installCmd.Dir = frontendPath
		installCmd.Stdout = os.Stdout
		installCmd.Stderr = os.Stderr

		if err := installCmd.Run(); err != nil {
			return fmt.Errorf("failed to setup frontend: %w", err)
		}

		// Install dependencies
		if _, err := os.Stat(filepath.Join(frontendPath, "package.json")); err == nil {
			o.log("📦 Installing frontend dependencies...", "\x1b[33m")
			pnpmCmd := exec.Command("pnpm", "install")
			pnpmCmd.Dir = frontendPath
			pnpmCmd.Stdout = os.Stdout
			pnpmCmd.Stderr = os.Stderr

			if err := pnpmCmd.Run(); err != nil {
				return fmt.Errorf("failed to install frontend dependencies: %w", err)
			}
		}

		o.log("✅ Frontend setup complete!", "\x1b[32m")
	} else {
		o.log("✅ Frontend already configured", "\x1b[32m")
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

func (o *DevOrchestrator) startProcess(processInfo *ProcessInfo) error {
	o.log(fmt.Sprintf("🚀 Starting %s...", processInfo.Name), processInfo.Color)

	cmd := exec.Command(processInfo.Command, processInfo.Args...)
	if processInfo.Dir != "" {
		cmd.Dir = processInfo.Dir
	}

	// Use PTY for backend to preserve colors, regular pipes for frontend
	if processInfo.Name == "Backend" {
		// Start the process with a PTY to preserve colors
		ptmx, err := pty.Start(cmd)
		if err != nil {
			return fmt.Errorf("failed to start %s with PTY: %w", processInfo.Name, err)
		}

		processInfo.Process = cmd

		o.log(fmt.Sprintf("✅ %s started (PID: %d)", processInfo.Name, cmd.Process.Pid), processInfo.Color)

		// Handle PTY output
		go func() {
			defer ptmx.Close()
			scanner := bufio.NewScanner(ptmx)
			for scanner.Scan() {
				line := scanner.Text()
				if strings.TrimSpace(line) != "" {
					o.formatLog(processInfo.Name, line, processInfo.Color)
				}
			}
		}()
	} else {
		// Use regular pipes for frontend and other processes
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			return err
		}
		stderr, err := cmd.StderrPipe()
		if err != nil {
			return err
		}

		processInfo.Process = cmd

		// Start the process
		if err := cmd.Start(); err != nil {
			return fmt.Errorf("failed to start %s: %w", processInfo.Name, err)
		}

		o.log(fmt.Sprintf("✅ %s started (PID: %d)", processInfo.Name, cmd.Process.Pid), processInfo.Color)

		// Handle stdout
		go func() {
			scanner := bufio.NewScanner(stdout)
			for scanner.Scan() {
				line := scanner.Text()
				if strings.TrimSpace(line) != "" {
					o.formatLog(processInfo.Name, line, processInfo.Color)
				}
			}
		}()

		// Handle stderr
		go func() {
			scanner := bufio.NewScanner(stderr)
			for scanner.Scan() {
				line := scanner.Text()
				if strings.TrimSpace(line) != "" {
					o.formatLog(processInfo.Name, line, processInfo.Color)
				}
			}
		}()
	}

	return nil
}

func (o *DevOrchestrator) formatLog(processName, line, color string) {
	// Skip noisy logs
	if strings.Contains(line, "watching") ||
		strings.Contains(line, "!exclude") ||
		strings.Contains(line, "building...") ||
		strings.Contains(line, "running...") {
		return
	}

	// Handle specific log formats
	if processName == "Frontend" {
		if strings.Contains(line, "VITE v") {
			o.log("⚡ Frontend: Vite ready", color)
		} else if strings.Contains(line, "Local:") {
			o.log("🌐 Frontend: Dev server ready", color)
		} else if strings.Contains(line, "Network:") {
			o.log("🌍 Frontend: Network access ready", color)
		} else if strings.TrimSpace(line) != "" {
			o.log(fmt.Sprintf("⚡ Frontend: %s", line), color)
		}
	} else if processName == "Backend" {
		// Check if this is a Huma-style log (contains HTTP request info)
		if o.isHumaLog(line) {
			// Pass Huma logs through directly to preserve their native formatting and colors
			fmt.Println(line)
			return
		}

		// Show HTTP request logs (contain | separators for Fiber logs) - keep native format
		if strings.Count(line, "|") >= 4 {
			o.formatHttpLog(line)
		} else if strings.Contains(line, "Server starting") ||
			strings.Contains(line, "Fiber") ||
			strings.Contains(line, "http://") ||
			strings.Contains(line, "Handlers") ||
			strings.Contains(line, "Processes") ||
			strings.Contains(line, "PID") ||
			strings.Contains(line, "├") ||
			strings.Contains(line, "│") ||
			strings.Contains(line, "└") ||
			strings.Contains(line, "┌") ||
			strings.Contains(line, "┐") ||
			strings.Contains(line, "─") {
			o.log(fmt.Sprintf("🔧 Backend: %s", line), color)
		} else if strings.TrimSpace(line) != "" && !strings.Contains(line, "bound on host") {
			// Show other backend logs but not the "bound on host" message
			o.log(fmt.Sprintf("🔧 Backend: %s", line), color)
		}
	} else {
		if strings.TrimSpace(line) != "" {
			o.log(fmt.Sprintf("%s: %s", processName, line), color)
		}
	}
}

// isHumaLog detects if a log line is from Huma based on its characteristic format
func (o *DevOrchestrator) isHumaLog(line string) bool {
	// Huma logs typically contain:
	// - A timestamp in format "2006/01/02 15:04:05"
	// - HTTP method and URL in quotes
	// - "from" keyword
	// - Status code, size, and duration

	// Check for characteristic Huma log patterns
	if strings.Contains(line, "\"GET ") ||
		strings.Contains(line, "\"POST ") ||
		strings.Contains(line, "\"PUT ") ||
		strings.Contains(line, "\"DELETE ") ||
		strings.Contains(line, "\"PATCH ") ||
		strings.Contains(line, "\"HEAD ") ||
		strings.Contains(line, "\"OPTIONS ") {

		// Additional validation: check for "from" and typical status/timing pattern
		if strings.Contains(line, " from ") &&
			(strings.Contains(line, " - ") || strings.Contains(line, " in ")) {
			return true
		}
	}

	// Also check for Huma startup messages
	if strings.Contains(line, "server starting on") ||
		strings.Contains(line, "API documentation available") ||
		strings.Contains(line, "OpenAPI spec available") {
		return true
	}

	return false
}

func (o *DevOrchestrator) formatHttpLog(line string) {
	// Parse HTTP log like dev.ts: "14:30:36 | 200 | 76.959µs | 127.0.0.1 | GET | /api/health"
	parts := strings.Split(line, "|")
	if len(parts) < 6 {
		// If it doesn't match expected format, just print as-is
		fmt.Println(line)
		return
	}

	// Trim whitespace from parts
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}

	timeStr := parts[0]
	status := parts[1]
	duration := parts[2]
	_ = parts[3] // ip (unused)
	method := parts[4]
	path := parts[5]

	// Color code by status
	statusColor := "\x1b[32m" // Green for 2xx
	if strings.HasPrefix(status, "3") {
		statusColor = "\x1b[33m" // Yellow for 3xx
	} else if strings.HasPrefix(status, "4") {
		statusColor = "\x1b[31m" // Red for 4xx
	} else if strings.HasPrefix(status, "5") {
		statusColor = "\x1b[35m" // Magenta for 5xx
	}

	// Color code by method
	methodColor := "\x1b[36m" // Cyan for GET
	if method == "POST" {
		methodColor = "\x1b[32m" // Green
	} else if method == "PUT" {
		methodColor = "\x1b[33m" // Yellow
	} else if method == "DELETE" {
		methodColor = "\x1b[31m" // Red
	}

	// Skip some noisy requests
	if strings.Contains(path, "/@vite/") || strings.Contains(path, "/node_modules/") || path == "/@react-refresh" {
		return
	}

	// Format the log nicely with colors
	fmt.Printf("\x1b[90m[%s]\x1b[0m %s%s\x1b[0m %s%-6s\x1b[0m \x1b[90m%10s\x1b[0m %s\n",
		timeStr, statusColor, status, methodColor, method, duration, path)
}

func (o *DevOrchestrator) setupGracefulShutdown() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c
		if !o.isShuttingDown {
			o.isShuttingDown = true
			o.shutdown()
		}
	}()
}

func (o *DevOrchestrator) shutdown() {
	o.log("🔄 Shutting down development environment...", "\x1b[33m")

	for _, processInfo := range o.processes {
		if processInfo.Process != nil && processInfo.Process.Process != nil {
			o.log(fmt.Sprintf("🛑 Stopping %s...", processInfo.Name), processInfo.Color)

			// Try graceful shutdown first
			processInfo.Process.Process.Signal(syscall.SIGTERM)

			// Force kill after 5 seconds if still running
			go func(proc *exec.Cmd, name string) {
				time.Sleep(5 * time.Second)
				if proc.Process != nil {
					o.log(fmt.Sprintf("💀 Force killing %s...", name), "\x1b[31m")
					proc.Process.Kill()
				}
			}(processInfo.Process, processInfo.Name)
		}
	}

	time.Sleep(1 * time.Second)
	o.log("✅ Development environment stopped", "\x1b[32m")
	os.Exit(0)
}

func (o *DevOrchestrator) startProxy() {
	// Create proxy to backend
	backendURL, _ := url.Parse(fmt.Sprintf("http://localhost:%s", o.config.Backend.Port))
	backendProxy := httputil.NewSingleHostReverseProxy(backendURL)

	// Create proxy to frontend
	frontendURL, _ := url.Parse("http://localhost:3001")
	frontendProxy := httputil.NewSingleHostReverseProxy(frontendURL)

	// Create HTTP handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		// Route API requests to backend
		if path == "/api" || strings.HasPrefix(path, "/api/") {
			backendProxy.ServeHTTP(w, r)
			return
		}

		// Route everything else to frontend
		frontendProxy.ServeHTTP(w, r)
	})

	// Start proxy server
	server := &http.Server{
		Addr:    ":3000",
		Handler: handler,
	}

	o.log("✅ Development proxy started on port 3000", "\x1b[36m")
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		o.log(fmt.Sprintf("❌ Proxy server error: %v", err), "\x1b[31m")
	}
}

func (o *DevOrchestrator) fetchAndSaveOpenAPISpec() error {
	o.log("📋 Fetching OpenAPI specification...", "\x1b[36m")

	// Create build directory if it doesn't exist
	buildDir := "build"
	if err := os.MkdirAll(buildDir, 0755); err != nil {
		return fmt.Errorf("failed to create build directory: %w", err)
	}

	// Fetch OpenAPI spec from running server
	openAPIURL := fmt.Sprintf("http://localhost:%s/openapi.json", o.config.Backend.Port)

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

func (o *DevOrchestrator) Start() error {
	o.setupGracefulShutdown()

	o.log(fmt.Sprintf("🚀 Starting GoFlux development environment for '%s'", o.config.Name), "\x1b[32m")

	// Setup frontend if needed
	if err := o.setupFrontendIfNeeded(); err != nil {
		return err
	}

	// Check if frontend dependencies are installed
	frontendDir := "frontend"
	if _, err := os.Stat(filepath.Join(frontendDir, "node_modules")); os.IsNotExist(err) {
		o.log("📦 Installing frontend dependencies...", "\x1b[33m")
		installCmd := exec.Command("pnpm", "install")
		installCmd.Dir = frontendDir
		installCmd.Stdout = os.Stdout
		installCmd.Stderr = os.Stderr
		if err := installCmd.Run(); err != nil {
			return fmt.Errorf("failed to install frontend dependencies: %w", err)
		}
	}

	// Install Go dependencies
	o.log("📦 Installing Go dependencies...", "\x1b[33m")
	goModCmd := exec.Command("go", "mod", "tidy")
	goModCmd.Stdout = os.Stdout
	goModCmd.Stderr = os.Stderr
	if err := goModCmd.Run(); err != nil {
		o.log("⚠️  Warning: Could not install Go dependencies", "\x1b[33m")
	}

	// Define processes
	o.processes = []ProcessInfo{
		{
			Name:    "Frontend",
			Command: "sh",
			Args:    []string{"-c", o.config.Frontend.DevCmd},
			Dir:     "",
			Color:   "\x1b[35m", // Magenta
		},
		{
			Name:    "Backend",
			Command: "air",
			Args:    []string{},
			Dir:     "",
			Color:   "\x1b[34m", // Blue
		},
	}

	// Start frontend
	if err := o.startProcess(&o.processes[0]); err != nil {
		return err
	}

	// Wait for frontend to be ready
	o.log("⏳ Waiting for frontend dev server...", "\x1b[33m")
	if !o.waitForPort("3001", 15*time.Second) {
		return fmt.Errorf("frontend dev server failed to start on port 3001")
	}
	o.log("✅ Frontend dev server ready on port 3001", "\x1b[32m")

	// Start backend
	if err := o.startProcess(&o.processes[1]); err != nil {
		return err
	}

	// Wait for backend to be ready
	o.log("⏳ Waiting for backend server...", "\x1b[33m")
	if !o.waitForPort(o.config.Backend.Port, 15*time.Second) {
		return fmt.Errorf("backend server failed to start on port %s", o.config.Backend.Port)
	}

	// Fetch and save OpenAPI spec from running server
	if err := o.fetchAndSaveOpenAPISpec(); err != nil {
		o.log("⚠️  Warning: Could not fetch OpenAPI spec", "\x1b[33m")
		if o.debug {
			o.log(fmt.Sprintf("OpenAPI fetch error: %v", err), "\x1b[31m")
		}
	} else {
		// Now generate types with fresh OpenAPI spec
		o.log("🔧 Generating API types from OpenAPI spec...", "\x1b[36m")
		if err := o.generateTypes(); err != nil {
			o.log("⚠️  Warning: Could not generate types", "\x1b[33m")
			if o.debug {
				o.log(fmt.Sprintf("Type generation error: %v", err), "\x1b[31m")
			}
		}
	}

	// Start proxy server
	o.log("🔗 Starting development proxy...", "\x1b[36m")
	go o.startProxy()

	// Wait for proxy to be ready
	o.log("⏳ Waiting for proxy server...", "\x1b[33m")
	if !o.waitForPort("3000", 10*time.Second) {
		return fmt.Errorf("proxy server failed to start on port 3000")
	}

	o.log("🎉 Development environment ready!", "\x1b[32m")
	o.log("🌐 Development: http://localhost:3000 (proxy)", "\x1b[36m")
	o.log("🌐 Frontend: http://localhost:3001 (direct)", "\x1b[36m")
	o.log(fmt.Sprintf("🔌 Backend: http://localhost:%s (direct)", o.config.Backend.Port), "\x1b[36m")
	o.log("📡 API: http://localhost:3000/api/health", "\x1b[36m")
	o.log("⚡ Hot reload enabled for both frontend and backend", "\x1b[36m")
	o.log("", "")
	o.log("Press Ctrl+C to stop all servers", "\x1b[33m")

	// Keep the process alive
	select {} // Block forever until interrupted
}

func readConfig(path string) (*ProjectConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config ProjectConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}
