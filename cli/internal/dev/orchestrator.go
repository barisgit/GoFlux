package dev

import (
	"context"
	"fmt"
	"net/http"
	"os/exec"
	"sync"
	"time"

	"github.com/barisgit/goflux/config"

	"github.com/fsnotify/fsnotify"
)

type ProcessState int

const (
	ProcessStopped ProcessState = iota
	ProcessStarting
	ProcessRunning
	ProcessStopping
)

type ProcessInfo struct {
	Name    string
	Process *exec.Cmd
	Command string
	Args    []string
	Dir     string
	Color   string
	State   ProcessState
}

type LogEntry struct {
	Timestamp time.Time
	Process   string
	Message   string
	Color     string
}

type DevOrchestrator struct {
	// Configuration
	config *config.ProjectConfig
	debug  bool

	// Process management
	backendProcess *ProcessInfo
	processes      []ProcessInfo
	processMutex   sync.RWMutex

	// Networking
	proxyServer  *http.Server
	frontendPort int
	backendPort  int

	// File watching
	fileWatcher   *fsnotify.Watcher
	configWatcher *fsnotify.Watcher

	// Restart management
	restartChan       chan string
	restartDebounceMS int
	isRestarting      bool
	lastRestartTime   time.Time
	restartMutex      sync.Mutex

	// Logging
	startupLogs    []LogEntry
	captureStartup bool
	logMutex       sync.RWMutex

	// Lifecycle management
	ctx            context.Context
	cancel         context.CancelFunc
	shutdownChan   chan bool
	isShuttingDown bool
	shutdownMutex  sync.Mutex
}

func NewDevOrchestrator(cfg *config.ProjectConfig, debug bool) *DevOrchestrator {
	ctx, cancel := context.WithCancel(context.Background())

	return &DevOrchestrator{
		config:            cfg,
		debug:             debug,
		ctx:               ctx,
		cancel:            cancel,
		restartDebounceMS: 500,
		restartChan:       make(chan string, 10),
		shutdownChan:      make(chan bool, 1),
		startupLogs:       make([]LogEntry, 0, 100),
	}
}

func (o *DevOrchestrator) log(message, color string) {
	if color == "" {
		color = "\x1b[0m"
	}
	fmt.Printf("\x1b[32m[O]\x1b[0m %s%s\x1b[0m\n", color, message)
}
