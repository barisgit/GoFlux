package dev

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/barisgit/goflux/cli/internal/frontend"
)

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

	o.log("Logger legend: \x1b[35m[F]\x1b[0m Frontend, \x1b[34m[B]\x1b[0m Backend, \x1b[32m[O]\x1b[0m Orchestrator", "")

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
	cmd := exec.Command("go", "mod", "tidy")
	if err := cmd.Run(); err != nil {
		o.log("⚠️  Warning: Could not install Go dependencies", "\x1b[33m")
	}

	// Get frontend dev command from config
	frontendDevCmd := strings.ReplaceAll(o.config.Frontend.DevCmd, "{{port}}", fmt.Sprintf("%d", o.frontendPort))

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

	// Setup file watcher for automatic type generation (before backend starts)
	if err := o.setupFileWatcher(); err != nil {
		o.log("⚠️  Warning: Could not setup file watcher", "\x1b[33m")
		if o.debug {
			o.log(fmt.Sprintf("File watcher error: %v", err), "\x1b[31m")
		}
	}

	// Setup config watcher for hot reload
	if err := o.setupConfigWatcher(); err != nil {
		o.log("⚠️  Warning: Could not setup config watcher", "\x1b[33m")
		if o.debug {
			o.log(fmt.Sprintf("Config watcher error: %v", err), "\x1b[31m")
		}
	} else {
		o.log("⚙️  Config hot reload enabled", "\x1b[36m")
	}

	// Generate OpenAPI spec and types before starting backend
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

	// Start backend with our own process manager (after OpenAPI/type generation)
	if err := o.startBackendProcess(); err != nil {
		o.log("❌ Failed to start backend process", "\x1b[31m")
		o.log(fmt.Sprintf("Error details: %v", err), "\x1b[31m")
		o.log("💡 This usually indicates a compilation error or missing dependencies", "\x1b[33m")
		o.log("🔧 Please check your Go code for syntax errors and ensure all imports are available", "\x1b[36m")
		return err
	}

	// Wait for backend to be ready
	o.log("⏳ Waiting for backend server...", "\x1b[33m")
	if !o.waitForPort(fmt.Sprintf("%d", o.backendPort), 15*time.Second) {
		o.log("❌ Backend server failed to start - check error output above", "\x1b[31m")
		o.log("💡 Common issues:", "\x1b[33m")
		o.log("   • Compilation errors in Go code", "\x1b[33m")
		o.log("   • Missing imports or dependencies", "\x1b[33m")
		o.log("   • Runtime panics during server initialization", "\x1b[33m")
		o.log("   • Port conflicts (check if something else is using the port)", "\x1b[33m")
		return fmt.Errorf("backend server failed to start on port %d", o.backendPort)
	}

	// Stop capturing backend logs now that the server is ready
	o.StopCapturingStartupLogs()

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
	o.log("⚙️  Configuration hot reload (flux.yaml)", "\x1b[36m")
	o.log("", "")

	// Now replay the captured backend startup logs
	o.ReplayBackendStartupLogs()

	o.log("Press Ctrl+C to stop all servers", "\x1b[33m")

	// Wait for shutdown signal
	<-o.shutdownChan
	return nil
}

func (o *DevOrchestrator) setupFrontendIfNeeded() error {
	frontendPath := "frontend"
	packageJsonPath := filepath.Join(frontendPath, "package.json")

	if _, err := os.Stat(packageJsonPath); os.IsNotExist(err) {
		o.log("📦 Setting up frontend for the first time...", "\x1b[33m")

		// Use the unified frontend management system
		unifiedManager, err := frontend.NewUnifiedManager(o.config, o.debug)
		if err != nil {
			return fmt.Errorf("failed to create unified frontend manager: %w", err)
		}

		// Generate frontend using the unified system
		if err := unifiedManager.GenerateFrontend(frontendPath); err != nil {
			return fmt.Errorf("failed to setup frontend: %w", err)
		}

		// Install dependencies if package.json was created
		if _, err := os.Stat(packageJsonPath); err == nil {
			o.log("📦 Installing frontend dependencies...", "\x1b[33m")

			// Use pnpm install as the default
			cmd := exec.Command("pnpm", "install")
			cmd.Dir = frontendPath
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr

			if err := cmd.Run(); err != nil {
				return fmt.Errorf("failed to install frontend dependencies: %w", err)
			}
		}

		o.log("✅ Frontend setup complete!", "\x1b[32m")
	} else {
		o.log("✅ Frontend already configured", "\x1b[32m")
	}

	return nil
}

func (o *DevOrchestrator) startProcess(processInfo *ProcessInfo) error {
	o.log(fmt.Sprintf("🚀 Starting %s...", processInfo.Name), processInfo.Color)

	cmd := exec.Command(processInfo.Command, processInfo.Args...)
	if processInfo.Dir != "" {
		cmd.Dir = processInfo.Dir
	}

	// Set process group for processes so we can kill the entire process tree
	o.setupProcessGroup(cmd)

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

	return nil
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

func (o *DevOrchestrator) shutdown() {
	o.log("🔄 Shutting down development environment...", "\x1b[33m")

	// Close file watchers first
	if o.fileWatcher != nil {
		o.log("🛑 Stopping file watcher...", "\x1b[36m")
		o.fileWatcher.Close()
	}
	if o.configWatcher != nil {
		o.log("🛑 Stopping config watcher...", "\x1b[36m")
		o.configWatcher.Close()
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

	// Shutdown all processes using platform-specific method
	o.shutdownProcesses()

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
