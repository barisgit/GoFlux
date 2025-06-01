package dev

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/barisgit/goflux/config"
	"github.com/fsnotify/fsnotify"
)

// RestartManager handles debounced restarts
type RestartManager struct {
	orchestrator *DevOrchestrator
	pm           *ProcessManager
}

func (o *DevOrchestrator) newRestartManager() *RestartManager {
	return &RestartManager{
		orchestrator: o,
		pm:           o.newProcessManager(),
	}
}

// setupWatchers sets up both file and config watchers
func (o *DevOrchestrator) setupWatchers() error {
	// Setup file watcher for backend restarts
	if err := o.setupFileWatcher(); err != nil {
		o.log("⚠️  Warning: Could not setup file watcher", "\x1b[33m")
		if o.debug {
			o.log(fmt.Sprintf("File watcher error: %v", err), "\x1b[31m")
		}
	}

	// Setup config watcher for configuration changes
	if err := o.setupConfigWatcher(); err != nil {
		o.log("⚠️  Warning: Could not setup config watcher", "\x1b[33m")
		if o.debug {
			o.log(fmt.Sprintf("Config watcher error: %v", err), "\x1b[31m")
		}
	} else {
		o.log("⚙️  Configuration hot reload enabled", "\x1b[36m")
	}

	// Start the restart worker
	rm := o.newRestartManager()
	go rm.restartWorker()

	return nil
}

// setupFileWatcher creates a file watcher for backend code changes
func (o *DevOrchestrator) setupFileWatcher() error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create file watcher: %w", err)
	}
	o.fileWatcher = watcher

	// Watch key directories for backend changes
	watchPaths := []string{
		"internal", // API code, DB models, etc.
		"pkg",      // Shared packages
		"cmd",      // Command code
		".",        // Root directory for main.go and other root files
	}

	for _, path := range watchPaths {
		if err := o.addDirectoryRecursively(path); err != nil {
			o.log(fmt.Sprintf("⚠️  Warning: Could not watch %s: %v", path, err), "\x1b[33m")
		}
	}

	// Start file event handling
	go o.handleFileEvents()

	return nil
}

// setupConfigWatcher creates a watcher for flux.yaml changes
func (o *DevOrchestrator) setupConfigWatcher() error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create config watcher: %w", err)
	}
	o.configWatcher = watcher

	// Watch current directory for flux.yaml
	if err := o.configWatcher.Add("."); err != nil {
		return fmt.Errorf("failed to watch current directory: %w", err)
	}

	// Start config event handling
	go o.handleConfigEvents()

	return nil
}

// addDirectoryRecursively adds a directory and subdirectories to the watcher
func (o *DevOrchestrator) addDirectoryRecursively(root string) error {
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return nil // Skip non-existent directories
			}
			return err
		}

		// Only watch directories and skip common ignore patterns
		if info.IsDir() {
			// Skip common directories we don't want to watch
			name := filepath.Base(path)
			if name == ".git" || name == "node_modules" || name == "build" || name == "dist" || name == ".next" {
				return filepath.SkipDir
			}

			if err := o.fileWatcher.Add(path); err != nil {
				if o.debug {
					o.log(fmt.Sprintf("⚠️  Could not watch %s: %v", path, err), "\x1b[33m")
				}
			} else if o.debug {
				o.log(fmt.Sprintf("👁️  Watching: %s", path), "\x1b[36m")
			}
		}

		return nil
	})
}

// handleFileEvents processes file system events
func (o *DevOrchestrator) handleFileEvents() {
	for {
		select {
		case <-o.ctx.Done():
			return
		case event, ok := <-o.fileWatcher.Events:
			if !ok {
				return
			}

			// Only react to Go file changes
			if strings.HasSuffix(event.Name, ".go") &&
				(event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Create == fsnotify.Create) {

				// Send restart request (non-blocking)
				select {
				case o.restartChan <- event.Name:
					// Request queued
				default:
					// Channel full, skip this event
					if o.debug {
						o.log("⚠️  Restart queue full, skipping", "\x1b[33m")
					}
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

// handleConfigEvents processes configuration file changes
func (o *DevOrchestrator) handleConfigEvents() {
	lastConfigChange := time.Time{}

	for {
		select {
		case <-o.ctx.Done():
			return
		case event, ok := <-o.configWatcher.Events:
			if !ok {
				return
			}

			// Only react to flux.yaml changes
			if strings.HasSuffix(event.Name, "flux.yaml") && event.Op&fsnotify.Write == fsnotify.Write {
				// Debounce config changes (1 second)
				if time.Since(lastConfigChange) < time.Second {
					continue
				}
				lastConfigChange = time.Now()

				go o.handleConfigChange()
			}

		case err, ok := <-o.configWatcher.Errors:
			if !ok {
				return
			}
			if o.debug {
				o.log(fmt.Sprintf("Config watcher error: %v", err), "\x1b[31m")
			}
		}
	}
}

// handleConfigChange processes configuration file changes
func (o *DevOrchestrator) handleConfigChange() {
	o.log("⚙️  flux.yaml changed, reloading configuration...", "\x1b[33m")

	// Load new configuration
	cm := config.NewConfigManager(config.ConfigLoadOptions{
		Path:              "flux.yaml",
		AllowMissing:      false,
		ValidateStructure: true,
		ApplyDefaults:     true,
		WarnOnDeprecated:  false,
		Quiet:             false,
	})

	newConfig, err := cm.LoadConfig()
	if err != nil {
		o.log(fmt.Sprintf("❌ Failed to reload config: %v", err), "\x1b[31m")
		o.log("💡 Please check your flux.yaml syntax", "\x1b[36m")
		return
	}

	// Check if restart is needed
	needsRestart := o.configNeedsRestart(o.config, newConfig)

	// Update configuration
	o.config = newConfig

	if needsRestart {
		o.log("🔄 Configuration changes require restart...", "\x1b[36m")
		// Send restart request
		select {
		case o.restartChan <- "flux.yaml":
		default:
			o.log("⚠️  Could not queue restart for config change", "\x1b[33m")
		}
	} else {
		o.log("✅ Configuration reloaded (no restart needed)", "\x1b[32m")
	}
}

// configNeedsRestart checks if configuration changes require a restart
func (o *DevOrchestrator) configNeedsRestart(old, new *config.ProjectConfig) bool {
	// Check critical configuration that requires restart
	return old.Port != new.Port ||
		old.Frontend.DevCmd != new.Frontend.DevCmd ||
		old.Backend.Router != new.Backend.Router
}

// restartWorker processes restart requests with debouncing
func (rm *RestartManager) restartWorker() {
	debounceTimer := time.NewTimer(0)
	debounceTimer.Stop()

	var pendingFile string

	for {
		select {
		case <-rm.orchestrator.ctx.Done():
			return
		case fileName := <-rm.orchestrator.restartChan:
			// New restart request - reset debounce timer
			pendingFile = fileName
			debounceTimer.Reset(time.Duration(rm.orchestrator.restartDebounceMS) * time.Millisecond)

		case <-debounceTimer.C:
			// Timer expired - execute restart
			if pendingFile != "" {
				rm.executeRestart(pendingFile)
				pendingFile = ""
			}
		}
	}
}

// executeRestart performs the actual backend restart
func (rm *RestartManager) executeRestart(fileName string) {
	o := rm.orchestrator

	// Check if already restarting
	o.restartMutex.Lock()
	if o.isRestarting {
		o.restartMutex.Unlock()
		o.log("⏳ Restart already in progress, skipping...", "\x1b[33m")
		return
	}
	o.isRestarting = true
	o.lastRestartTime = time.Now()
	o.restartMutex.Unlock()

	defer func() {
		o.restartMutex.Lock()
		o.isRestarting = false
		o.restartMutex.Unlock()
	}()

	o.log(fmt.Sprintf("📝 %s changed, restarting backend...", filepath.Base(fileName)), "\x1b[36m")

	// Start capturing logs for replay
	o.startCapturingLogs()
	defer o.stopCapturingLogs()

	// Restart backend using the process manager
	if err := rm.pm.startBackend(); err != nil {
		o.log(fmt.Sprintf("❌ Failed to restart backend: %v", err), "\x1b[31m")
		o.log("💡 Check error output above for issues", "\x1b[33m")
		return
	}

	// Wait for backend to be ready
	o.log("⏳ Waiting for backend to be ready...", "\x1b[33m")
	if rm.pm.checkPort(o.backendPort) || rm.pm.waitForPortFree(o.backendPort, 15*time.Second) {
		// Small delay for initialization
		time.Sleep(500 * time.Millisecond)

		// Replay captured startup logs
		o.replayStartupLogs()

		// Regenerate types
		if err := o.generateTypes(); err != nil {
			o.log("⚠️  Warning: Could not regenerate types", "\x1b[33m")
			if o.debug {
				o.log(fmt.Sprintf("Type generation error: %v", err), "\x1b[31m")
			}
		} else {
			o.log("✅ Backend restarted and types regenerated!", "\x1b[32m")
		}
	} else {
		o.log("⚠️  Backend not ready after restart", "\x1b[33m")
		o.log("💡 Check for compilation errors above", "\x1b[36m")
	}
}

// closeWatchers closes all file watchers
func (o *DevOrchestrator) closeWatchers() {
	if o.fileWatcher != nil {
		o.log("🛑 Stopping file watcher...", "\x1b[36m")
		o.fileWatcher.Close()
	}
	if o.configWatcher != nil {
		o.log("🛑 Stopping config watcher...", "\x1b[36m")
		o.configWatcher.Close()
	}
}
