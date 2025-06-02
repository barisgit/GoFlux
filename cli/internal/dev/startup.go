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
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/barisgit/goflux/cli/internal/frontend"
)

func (o *DevOrchestrator) Start() error {
	o.shutdownChan = make(chan bool, 1)
	o.setupGracefulShutdown()

	// Ensure cleanup happens no matter what
	defer func() {
		o.shutdownMutex.Lock()
		if !o.isShuttingDown {
			o.log("🚨 Emergency cleanup triggered", "\x1b[31m")
			o.isShuttingDown = true
			o.shutdown()
		}
		o.shutdownMutex.Unlock()
	}()

	o.log(fmt.Sprintf("🚀 Starting GoFlux development environment for '%s'", o.config.Name), "\x1b[32m")
	o.log("Logger legend: \x1b[35m[F]\x1b[0m Frontend, \x1b[34m[B]\x1b[0m Backend, \x1b[32m[O]\x1b[0m Orchestrator", "")

	// Create process manager
	pm := o.newProcessManager()

	// Assign dynamic ports
	o.frontendPort = pm.findFreePort(o.config.Port + 1)
	o.backendPort = pm.findFreePort(o.frontendPort + 1)

	o.log(fmt.Sprintf("🔧 Assigned ports - Frontend: %d, Backend: %d, Proxy: %d", o.frontendPort, o.backendPort, o.config.Port), "\x1b[36m")

	// Setup frontend if needed
	if err := o.setupFrontendIfNeeded(); err != nil {
		return err
	}

	// Check and install frontend dependencies
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

	// Start frontend process
	frontendDevCmd := strings.ReplaceAll(o.config.Frontend.DevCmd, "{{port}}", fmt.Sprintf("%d", o.frontendPort))
	o.processes = []ProcessInfo{
		{
			Name:    "Frontend",
			Command: "sh",
			Args:    []string{"-c", frontendDevCmd},
			Dir:     "",
			Color:   "\x1b[35m",
			State:   ProcessStarting,
		},
	}

	if err := o.startProcess(&o.processes[0]); err != nil {
		return err
	}

	// Wait for frontend to be ready
	o.log("⏳ Waiting for frontend dev server...", "\x1b[33m")
	start := time.Now()
	for time.Since(start) < 15*time.Second {
		if pm.checkPort(o.frontendPort) {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	if !pm.checkPort(o.frontendPort) {
		return fmt.Errorf("frontend dev server failed to start on port %d", o.frontendPort)
	}
	o.log(fmt.Sprintf("✅ Frontend dev server ready on port %d", o.frontendPort), "\x1b[32m")

	// Setup watchers for file and config changes
	if err := o.setupWatchers(); err != nil {
		o.log("⚠️  Warning: Could not setup watchers", "\x1b[33m")
		if o.debug {
			o.log(fmt.Sprintf("Watcher setup error: %v", err), "\x1b[31m")
		}
	}

	// Generate initial API types before starting backend
	o.log("🔧 Generating initial API types...", "\x1b[36m")
	if err := o.generateTypes(); err != nil {
		o.log("⚠️  Warning: Could not generate initial types", "\x1b[33m")
		if o.debug {
			o.log(fmt.Sprintf("Type generation error: %v", err), "\x1b[31m")
		}
	}

	// Start backend using new process manager
	o.log("🚀 Starting backend server...", "\x1b[34m")
	if err := pm.startBackend(); err != nil {
		o.log("❌ Failed to start backend", "\x1b[31m")
		o.log(fmt.Sprintf("Error: %v", err), "\x1b[31m")
		o.log("💡 Check for compilation errors or missing dependencies", "\x1b[33m")
		return err
	}

	// Start proxy server
	o.log("🔗 Starting development proxy...", "\x1b[36m")
	go o.startProxy()

	// Wait for proxy to be ready
	o.log("⏳ Waiting for proxy server...", "\x1b[33m")
	if !pm.checkPort(o.config.Port) {
		// Wait a bit for proxy to be ready
		time.Sleep(500 * time.Millisecond)
		if !pm.checkPort(o.config.Port) {
			return fmt.Errorf("proxy server failed to start on port %d", o.config.Port)
		}
	}

	o.log("✅ Development proxy started on port "+fmt.Sprintf("%d", o.config.Port), "\x1b[32m")
	o.log("🎉 Development environment ready!", "\x1b[32m")
	o.log(fmt.Sprintf("🌐 Development: http://localhost:%d (proxy)", o.config.Port), "\x1b[36m")
	o.log(fmt.Sprintf("🌐 Frontend: http://localhost:%d (direct)", o.frontendPort), "\x1b[36m")
	o.log(fmt.Sprintf("🔌 Backend: http://localhost:%d (direct)", o.backendPort), "\x1b[36m")
	o.log(fmt.Sprintf("📡 API: http://localhost:%d/api/health", o.config.Port), "\x1b[36m")
	o.log("⚡ Hot reload enabled with intelligent file watching", "\x1b[36m")
	o.log("🔄 Automatic backend restart and type generation", "\x1b[36m")
	o.log("⚙️  Configuration hot reload (flux.yaml)", "\x1b[36m")
	o.log("", "")
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

	// Cancel context to stop all goroutines
	o.cancel()

	// Close watchers
	o.closeWatchers()

	// Stop backend process using process manager
	o.processMutex.Lock()
	if o.backendProcess != nil {
		pm := o.newProcessManager()
		if err := pm.stopBackend(5 * time.Second); err != nil {
			o.log("⚠️  Warning: Backend shutdown had issues", "\x1b[33m")
		}
	}
	o.processMutex.Unlock()

	// Shutdown proxy server
	if o.proxyServer != nil {
		o.log("🛑 Stopping proxy server...", "\x1b[36m")
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		if err := o.proxyServer.Shutdown(ctx); err != nil {
			o.log("⚠️  Force closing proxy server", "\x1b[33m")
		}
	}

	// Shutdown frontend and other processes
	o.shutdownProcesses()

	o.log("✅ Development environment stopped", "\x1b[32m")

	// Signal shutdown completion
	select {
	case o.shutdownChan <- true:
	default:
	}
}

// setupGracefulShutdown sets up signal handling for graceful shutdown
func (o *DevOrchestrator) setupGracefulShutdown() {
	c := make(chan os.Signal, 1)

	// Setup signal handling based on platform
	if syscall.SIGTERM != 0 { // Unix-like systems
		signal.Notify(c, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGHUP)
	} else { // Windows
		signal.Notify(c, os.Interrupt)
	}

	go func() {
		sig := <-c
		o.log(fmt.Sprintf("📡 Received signal: %v", sig), "\x1b[33m")

		o.shutdownMutex.Lock()
		if !o.isShuttingDown {
			o.isShuttingDown = true
			o.shutdownMutex.Unlock()
			o.shutdown()
			os.Exit(0)
		} else {
			o.shutdownMutex.Unlock()
		}
	}()
}
