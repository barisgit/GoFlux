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

func (o *DevOrchestrator) setupFileWatcher() error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create file watcher: %w", err)
	}
	o.fileWatcher = watcher

	// Watch directories that contain backend code for hot reload
	watchPaths := []string{
		"internal", // API code, DB models, etc.
		"pkg",      // Shared packages
		"cmd",      // Command code (including cmd/server for backward compatibility)
		"main.go",  // Root main.go file
		"*.go",     // Any other Go files in project root
	}

	// Add directories recursively since fsnotify doesn't watch subdirectories automatically
	for _, dir := range watchPaths {
		if err := o.addDirectoryRecursively(dir); err != nil {
			o.log(fmt.Sprintf("⚠️  Warning: Could not watch %s: %v", dir, err), "\x1b[33m")
		}
	}

	// Start watching in a goroutine
	go o.handleFileEvents()

	return nil
}

// addDirectoryRecursively adds a directory and all its subdirectories to the file watcher
func (o *DevOrchestrator) addDirectoryRecursively(root string) error {
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			// If the directory doesn't exist, skip it silently
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}

		// Only add directories to the watcher
		if info.IsDir() {
			if err := o.fileWatcher.Add(path); err != nil {
				o.log(fmt.Sprintf("⚠️  Warning: Could not watch %s: %v", path, err), "\x1b[33m")
			} else if o.debug {
				o.log(fmt.Sprintf("👁️  Watching directory: %s", path), "\x1b[36m")
			}
		}

		return nil
	})
}

func (o *DevOrchestrator) setupConfigWatcher() error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create config watcher: %w", err)
	}
	o.configWatcher = watcher

	// Watch the current directory for flux.yaml changes
	if err := o.configWatcher.Add("."); err != nil {
		return fmt.Errorf("failed to watch current directory: %w", err)
	}

	// Start watching in a goroutine
	go o.handleConfigEvents()

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
							o.log("💡 Check the error output above for compilation or runtime issues", "\x1b[33m")
							o.log("🔧 Fix the errors and save the file again to retry", "\x1b[36m")
							return
						}

						// Wait for backend to be ready
						o.log("⏳ Waiting for backend to be ready...", "\x1b[33m")
						if o.waitForPort(fmt.Sprintf("%d", o.backendPort), 15*time.Second) {
							// Small delay to ensure server is fully initialized
							time.Sleep(500 * time.Millisecond)

							// Stop capturing backend logs and replay startup messages (including register warnings)
							o.StopCapturingStartupLogs()
							o.ReplayBackendStartupLogs()

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
							o.log("⚠️  Backend not ready after restart - check for build errors or panics above", "\x1b[33m")
							o.log("🔧 The backend process may have exited due to compilation errors", "\x1b[33m")
							o.log("💡 Fix any errors shown and save the file again to retry", "\x1b[36m")
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

func (o *DevOrchestrator) handleConfigEvents() {
	for {
		select {
		case event, ok := <-o.configWatcher.Events:
			if !ok {
				return
			}

			// Only trigger on flux.yaml changes
			if strings.HasSuffix(event.Name, "flux.yaml") &&
				(event.Op&fsnotify.Write == fsnotify.Write) {

				// Debounce config changes
				o.typeGenMutex.Lock()
				if time.Since(o.lastTypeGen) > 1*time.Second {
					o.lastTypeGen = time.Now()
					o.typeGenMutex.Unlock()

					go func() {
						o.log("⚙️  flux.yaml changed, reloading configuration...", "\x1b[33m")

						// Use enhanced config loading with validation
						cm := config.NewConfigManager(config.ConfigLoadOptions{
							Path:              "flux.yaml",
							AllowMissing:      false,
							ValidateStructure: true,
							ApplyDefaults:     true,
							WarnOnDeprecated:  false, // Don't show warnings during hot reload
							Quiet:             false,
						})

						newConfig, err := cm.LoadConfig()
						if err != nil {
							o.log(fmt.Sprintf("❌ Failed to reload config: %v", err), "\x1b[31m")
							o.log("💡 Please check your flux.yaml syntax and fix any errors", "\x1b[36m")
							return
						}

						// Update config with write lock
						o.configMutex.Lock()
						oldConfig := o.config
						o.config = newConfig
						o.configMutex.Unlock()

						// Check if we need to restart services
						configChanged := o.checkConfigChanges(oldConfig, newConfig)
						if configChanged {
							o.log("🔄 Configuration changes detected, restarting services...", "\x1b[36m")
							o.restartServicesForConfig()
						} else {
							o.log("✅ Configuration reloaded (no restart needed)", "\x1b[32m")
						}
					}()
				} else {
					o.typeGenMutex.Unlock()
				}
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
func (o *DevOrchestrator) checkConfigChanges(old, new *config.ProjectConfig) bool {
	// Check if changes require service restart
	if old.Port != new.Port ||
		old.Frontend.DevCmd != new.Frontend.DevCmd ||
		old.Backend.Router != new.Backend.Router {
		return true
	}

	// TODO: Do something here

	return false
}

func (o *DevOrchestrator) restartServicesForConfig() {
	// This would restart services based on config changes
	// For now, we'll just log that this functionality exists
	o.log("🔄 Service restart for config changes not yet implemented", "\x1b[33m")
	o.log("💡 Please restart the dev environment manually for now", "\x1b[36m")
}
