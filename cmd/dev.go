package cmd

import (
	"bufio"
	"context"
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
	"github.com/fsnotify/fsnotify"
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
	proxyServer    *http.Server
	shutdownChan   chan bool
	fileWatcher    *fsnotify.Watcher
	lastTypeGen    time.Time
	typeGenMutex   sync.Mutex
	backendProcess *exec.Cmd
	backendMutex   sync.Mutex
	// Dynamic port assignments
	frontendPort int
	backendPort  int
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
		// Set process group for non-PTY processes so we can kill the entire process tree
		cmd.SysProcAttr = &syscall.SysProcAttr{
			Setpgid: true,
		}

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
	// Capture more signal types to ensure cleanup
	signal.Notify(c, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGHUP)

	go func() {
		sig := <-c
		o.log(fmt.Sprintf("📡 Received signal: %v", sig), "\x1b[33m")
		if !o.isShuttingDown {
			o.isShuttingDown = true
			o.shutdown()
		}
	}()
}

func (o *DevOrchestrator) shutdown() {
	o.log("🔄 Shutting down development environment...", "\x1b[33m")

	// Close file watcher first
	if o.fileWatcher != nil {
		o.log("🛑 Stopping file watcher...", "\x1b[36m")
		o.fileWatcher.Close()
	}

	// Stop our backend process
	o.backendMutex.Lock()
	if o.backendProcess != nil {
		o.stopBackendProcessUnsafe()
	}
	o.backendMutex.Unlock()

	// Shutdown proxy server
	if o.proxyServer != nil {
		o.log("🛑 Stopping proxy server...", "\x1b[36m")
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		if err := o.proxyServer.Shutdown(ctx); err != nil {
			o.log("⚠️  Force closing proxy server", "\x1b[33m")
		}
	}

	// Channel to track process shutdown completion
	done := make(chan bool, len(o.processes))

	// Step 1: Try graceful shutdown of remaining processes (frontend)
	for _, processInfo := range o.processes {
		if processInfo.Process != nil && processInfo.Process.Process != nil {
			o.log(fmt.Sprintf("🛑 Stopping %s (PID: %d)...", processInfo.Name, processInfo.Process.Process.Pid), processInfo.Color)

			// Regular process with process group
			pgid, err := syscall.Getpgid(processInfo.Process.Process.Pid)
			if err != nil {
				pgid = processInfo.Process.Process.Pid // fallback to PID if can't get PGID
			}

			// Try graceful shutdown first - send SIGTERM to the entire process group
			syscall.Kill(-pgid, syscall.SIGTERM)

			go func(proc *exec.Cmd, name string, processGroupID int) {
				defer func() { done <- true }()

				// Wait up to 5 seconds for graceful shutdown
				gracefulDone := make(chan error, 1)
				go func() {
					gracefulDone <- proc.Wait()
				}()

				select {
				case err := <-gracefulDone:
					if err != nil {
						o.log(fmt.Sprintf("⚠️  %s exited with error: %v", name, err), "\x1b[33m")
					} else {
						o.log(fmt.Sprintf("✅ %s stopped gracefully", name), "\x1b[32m")
					}
				case <-time.After(5 * time.Second):
					// Force kill the entire process group
					o.log(fmt.Sprintf("💀 Force killing %s and children (timeout)...", name), "\x1b[31m")
					syscall.Kill(-processGroupID, syscall.SIGKILL)
					// Also try individual process kill as fallback
					if proc.Process != nil {
						proc.Process.Kill()
					}
				}
			}(processInfo.Process, processInfo.Name, pgid)
		} else {
			done <- true // No process to stop
		}
	}

	// Wait for all processes to finish or timeout after 10 seconds
	timeout := time.After(10 * time.Second)
	processesLeft := len(o.processes)

	for processesLeft > 0 {
		select {
		case <-done:
			processesLeft--
		case <-timeout:
			o.log("⚠️  Timeout waiting for processes to stop", "\x1b[33m")
			processesLeft = 0
		}
	}

	// Step 2: Nuclear option - kill anything listening on our ports
	o.log("🧹 Cleaning up any remaining processes on our ports...", "\x1b[33m")
	o.forceKillPortProcesses()

	o.log("✅ Development environment stopped", "\x1b[32m")

	// Signal the main thread that shutdown is complete
	select {
	case o.shutdownChan <- true:
	default:
	}
}

// forceKillPortProcesses kills any processes still listening on our development ports
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

func (o *DevOrchestrator) startProxy() {
	// Create proxy to backend
	backendURL, _ := url.Parse(fmt.Sprintf("http://localhost:%d", o.backendPort))
	backendProxy := httputil.NewSingleHostReverseProxy(backendURL)

	// Create proxy to frontend
	frontendURL, _ := url.Parse(fmt.Sprintf("http://localhost:%d", o.frontendPort))
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
	o.proxyServer = &http.Server{
		Addr:    fmt.Sprintf(":%d", o.config.Port),
		Handler: handler,
	}

	o.log(fmt.Sprintf("✅ Development proxy started on port %d", o.config.Port), "\x1b[36m")
	if err := o.proxyServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
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

func (o *DevOrchestrator) Start() error {
	o.shutdownChan = make(chan bool, 1)
	o.setupGracefulShutdown()

	// Ensure cleanup happens no matter what
	defer func() {
		if !o.isShuttingDown {
			o.log("🚨 Emergency cleanup triggered", "\x1b[31m")
			o.isShuttingDown = true
			o.shutdown()
		}
	}()

	o.log(fmt.Sprintf("🚀 Starting GoFlux development environment for '%s'", o.config.Name), "\x1b[32m")

	// Assign dynamic ports
	o.frontendPort = o.findFreePort(o.config.Port + 1)
	o.backendPort = o.findFreePort(o.frontendPort + 1)

	o.log(fmt.Sprintf("🔧 Assigned ports - Frontend: %d, Backend: %d, Proxy: %d", o.frontendPort, o.backendPort, o.config.Port), "\x1b[36m")

	// Clean up any existing processes on our ports before starting
	o.log("🧹 Cleaning up any existing processes on target ports...", "\x1b[33m")
	o.forceKillPortProcesses()

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

	// Define processes with dynamic ports
	frontendDevCmd := strings.Replace(o.config.Frontend.DevCmd, "3001", fmt.Sprintf("%d", o.frontendPort), -1)

	o.processes = []ProcessInfo{
		{
			Name:    "Frontend",
			Command: "sh",
			Args:    []string{"-c", frontendDevCmd},
			Dir:     "",
			Color:   "\x1b[35m", // Magenta
		},
	}

	// Start frontend
	if err := o.startProcess(&o.processes[0]); err != nil {
		return err
	}

	// Wait for frontend to be ready
	o.log("⏳ Waiting for frontend dev server...", "\x1b[33m")
	if !o.waitForPort(fmt.Sprintf("%d", o.frontendPort), 15*time.Second) {
		return fmt.Errorf("frontend dev server failed to start on port %d", o.frontendPort)
	}
	o.log(fmt.Sprintf("✅ Frontend dev server ready on port %d", o.frontendPort), "\x1b[32m")

	// Start backend with our own process manager
	if err := o.startBackendProcess(); err != nil {
		return err
	}

	// Wait for backend to be ready
	o.log("⏳ Waiting for backend server...", "\x1b[33m")
	if !o.waitForPort(fmt.Sprintf("%d", o.backendPort), 15*time.Second) {
		return fmt.Errorf("backend server failed to start on port %d", o.backendPort)
	}

	// Setup file watcher for automatic type generation
	if err := o.setupFileWatcher(); err != nil {
		o.log("⚠️  Warning: Could not setup file watcher", "\x1b[33m")
		if o.debug {
			o.log(fmt.Sprintf("File watcher error: %v", err), "\x1b[31m")
		}
	}

	// Fetch and save OpenAPI spec from running server
	if err := o.fetchAndSaveOpenAPISpec(); err != nil {
		o.log("⚠️  Warning: Could not fetch OpenAPI spec", "\x1b[33m")
		if o.debug {
			o.log(fmt.Sprintf("OpenAPI fetch error: %v", err), "\x1b[31m")
		}
	} else {
		// Generate initial types
		o.log("🔧 Generating initial API types...", "\x1b[36m")
		if err := o.generateTypes(); err != nil {
			o.log("⚠️  Warning: Could not generate initial types", "\x1b[33m")
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
	if !o.waitForPort(fmt.Sprintf("%d", o.config.Port), 10*time.Second) {
		return fmt.Errorf("proxy server failed to start on port %d", o.config.Port)
	}

	o.log("🎉 Development environment ready!", "\x1b[32m")
	o.log(fmt.Sprintf("🌐 Development: http://localhost:%d (proxy)", o.config.Port), "\x1b[36m")
	o.log(fmt.Sprintf("🌐 Frontend: http://localhost:%d (direct)", o.frontendPort), "\x1b[36m")
	o.log(fmt.Sprintf("🔌 Backend: http://localhost:%d (direct)", o.backendPort), "\x1b[36m")
	o.log(fmt.Sprintf("📡 API: http://localhost:%d/api/health", o.config.Port), "\x1b[36m")
	o.log("⚡ Hot reload enabled with intelligent file watching", "\x1b[36m")
	o.log("🔄 Automatic backend restart and type generation", "\x1b[36m")
	o.log("", "")
	o.log("Press Ctrl+C to stop all servers", "\x1b[33m")

	// Wait for shutdown signal
	<-o.shutdownChan
	return nil
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

func (o *DevOrchestrator) findFreePort(startPort int) int {
	for port := startPort; port < startPort+100; port++ {
		if !o.checkPort(fmt.Sprintf("%d", port)) {
			return port
		}
	}
	// Fallback to a random high port if we can't find one in range
	return startPort + 1000
}

func (o *DevOrchestrator) setupFileWatcher() error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create file watcher: %w", err)
	}
	o.fileWatcher = watcher

	// Watch directories that contain API-related files
	watchDirs := []string{
		"internal/api",
		"internal/types",
		"cmd/server",
	}

	for _, dir := range watchDirs {
		if _, err := os.Stat(dir); !os.IsNotExist(err) {
			if err := o.fileWatcher.Add(dir); err != nil {
				o.log(fmt.Sprintf("⚠️  Warning: Could not watch %s: %v", dir, err), "\x1b[33m")
			}
		}
	}

	// Start watching in a goroutine
	go o.handleFileEvents()

	return nil
}

func (o *DevOrchestrator) handleFileEvents() {
	for {
		select {
		case event, ok := <-o.fileWatcher.Events:
			if !ok {
				return
			}

			// Only trigger on .go files in API-related directories
			if strings.HasSuffix(event.Name, ".go") &&
				(event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Create == fsnotify.Create) {

				// Debounce - only restart if it's been at least 2 seconds since last restart
				o.typeGenMutex.Lock()
				if time.Since(o.lastTypeGen) > 2*time.Second {
					o.lastTypeGen = time.Now()
					o.typeGenMutex.Unlock()

					// Restart backend and regenerate types
					go func(fileName string) {
						o.log(fmt.Sprintf("📝 %s changed, restarting backend...", filepath.Base(fileName)), "\x1b[36m")

						// Restart backend directly
						if err := o.restartBackend(); err != nil {
							o.log(fmt.Sprintf("❌ Failed to restart backend: %v", err), "\x1b[31m")
							return
						}

						// Wait for backend to be ready
						o.log("⏳ Waiting for backend to be ready...", "\x1b[33m")
						if o.waitForPort(fmt.Sprintf("%d", o.backendPort), 15*time.Second) {
							// Small delay to ensure server is fully initialized
							time.Sleep(1 * time.Second)

							// Fetch fresh OpenAPI spec
							if err := o.fetchAndSaveOpenAPISpec(); err != nil {
								o.log("⚠️  Warning: Could not fetch fresh OpenAPI spec", "\x1b[33m")
								if o.debug {
									o.log(fmt.Sprintf("OpenAPI fetch error: %v", err), "\x1b[31m")
								}
								return
							}

							// Generate types with fresh spec
							o.log("🔧 Regenerating types from fresh OpenAPI spec...", "\x1b[36m")
							if err := o.generateTypes(); err != nil {
								o.log("⚠️  Warning: Could not regenerate types", "\x1b[33m")
								if o.debug {
									o.log(fmt.Sprintf("Type generation error: %v", err), "\x1b[31m")
								}
							} else {
								o.log("✅ Backend restarted and types regenerated!", "\x1b[32m")
							}
						} else {
							o.log("⚠️  Backend not ready after restart, skipping type generation", "\x1b[33m")
						}
					}(event.Name)
				} else {
					o.typeGenMutex.Unlock()
				}
			}

		case err, ok := <-o.fileWatcher.Errors:
			if !ok {
				return
			}
			if o.debug {
				o.log(fmt.Sprintf("File watcher error: %v", err), "\x1b[31m")
			}
		}
	}
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
