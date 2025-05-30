package dev

import (
	"fmt"
	"net/http"
	"os/exec"
	"sync"
	"time"

	"github.com/barisgit/goflux/internal/config"

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
	processes      []ProcessInfo
	isShuttingDown bool
	config         *config.ProjectConfig
	debug          bool
	proxyServer    *http.Server
	shutdownChan   chan bool
	fileWatcher    *fsnotify.Watcher
	configWatcher  *fsnotify.Watcher
	lastTypeGen    time.Time
	typeGenMutex   sync.Mutex
	backendProcess *exec.Cmd
	backendMutex   sync.Mutex
	configMutex    sync.RWMutex
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
	timestamp := time.Now().Format("15:04:05")
	if color == "" {
		color = "\x1b[0m"
	}
	fmt.Printf("%s[%s] %s\x1b[0m\n", color, timestamp, message)
}
