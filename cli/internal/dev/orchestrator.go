package dev

import (
	"fmt"
	"net/http"
	"os/exec"
	"sync"
	"time"

	"github.com/barisgit/goflux/config"

	"github.com/fsnotify/fsnotify"
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
	processes          []ProcessInfo
	isShuttingDown     bool
	config             *config.ProjectConfig
	debug              bool
	proxyServer        *http.Server
	shutdownChan       chan bool
	fileWatcher        *fsnotify.Watcher
	configWatcher      *fsnotify.Watcher
	lastTypeGen        time.Time
	typeGenMutex       sync.Mutex
	backendProcess     *exec.Cmd
	backendMutex       sync.Mutex
	configMutex        sync.RWMutex
	backendStartupLogs []string
	captureBackendLogs bool
	// Dynamic port assignments
	frontendPort int
	backendPort  int
}

func NewDevOrchestrator(cfg *config.ProjectConfig, debug bool) *DevOrchestrator {
	return &DevOrchestrator{
		config: cfg,
		debug:  debug,
	}
}

func (o *DevOrchestrator) log(message, color string) {
	if color == "" {
		color = "\x1b[0m"
	}
	fmt.Printf("\x1b[32m[O]\x1b[0m %s%s\x1b[0m\n", color, message)
}
