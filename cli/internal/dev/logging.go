package dev

import (
	"fmt"
	"time"
)

// formatLog outputs a formatted log message
func (o *DevOrchestrator) formatLog(processName, line, color string) {
	prefix := "[?]"
	switch processName {
	case "Frontend":
		prefix = "[F]"
	case "Backend":
		prefix = "[B]"
	case "Orchestrator":
		prefix = "[O]"
	}

	fmt.Printf("%s%s\x1b[0m %s\n", color, prefix, line)
}

// startCapturingLogs enables log capture for replay
func (o *DevOrchestrator) startCapturingLogs() {
	o.logMutex.Lock()
	defer o.logMutex.Unlock()

	o.captureStartup = true
	o.startupLogs = o.startupLogs[:0] // Clear existing logs

	if o.debug {
		o.log("📝 Started capturing startup logs", "\x1b[36m")
	}
}

// stopCapturingLogs disables log capture
func (o *DevOrchestrator) stopCapturingLogs() {
	o.logMutex.Lock()
	defer o.logMutex.Unlock()

	o.captureStartup = false

	if o.debug {
		o.log(fmt.Sprintf("📝 Stopped capturing logs (%d entries)", len(o.startupLogs)), "\x1b[36m")
	}
}

// replayStartupLogs replays captured startup logs
func (o *DevOrchestrator) replayStartupLogs() {
	o.logMutex.RLock()
	logs := make([]LogEntry, len(o.startupLogs))
	copy(logs, o.startupLogs)
	o.logMutex.RUnlock()

	if len(logs) == 0 {
		return
	}

	o.log("📼 Replaying startup logs...", "\x1b[36m")

	for _, entry := range logs {
		o.formatLog(entry.Process, entry.Message, entry.Color)
	}

	o.log(fmt.Sprintf("📼 Replayed %d startup messages", len(logs)), "\x1b[36m")
}

// clearStartupLogs clears the captured startup logs
func (o *DevOrchestrator) clearStartupLogs() {
	o.logMutex.Lock()
	defer o.logMutex.Unlock()

	o.startupLogs = o.startupLogs[:0]
}

// addStartupLog adds a log entry to the startup capture (thread-safe)
func (o *DevOrchestrator) addStartupLog(process, message, color string) {
	o.logMutex.Lock()
	defer o.logMutex.Unlock()

	if !o.captureStartup || len(o.startupLogs) >= 100 {
		return
	}

	o.startupLogs = append(o.startupLogs, LogEntry{
		Timestamp: time.Now(),
		Process:   process,
		Message:   message,
		Color:     color,
	})
}
