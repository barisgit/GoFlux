//go:build windows

package dev

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"time"
)

func (o *DevOrchestrator) setupProcessGroup(cmd *exec.Cmd) {
	// Windows doesn't support process groups in the same way as Unix
	// We'll handle process termination differently
}

func (o *DevOrchestrator) setupGracefulShutdown() {
	c := make(chan os.Signal, 1)
	// Windows only supports os.Interrupt and os.Kill
	signal.Notify(c, os.Interrupt)

	go func() {
		sig := <-c
		o.log(fmt.Sprintf("📡 Received signal: %v", sig), "\x1b[33m")
		if !o.isShuttingDown {
			o.isShuttingDown = true
			o.shutdown()
		}
	}()
}

func (o *DevOrchestrator) shutdownProcesses() {
	// Channel to track process shutdown completion
	done := make(chan bool, len(o.processes))

	// Windows process shutdown - simpler than Unix
	for _, processInfo := range o.processes {
		if processInfo.Process != nil && processInfo.Process.Process != nil {
			o.log(fmt.Sprintf("🛑 Stopping %s (PID: %d)...", processInfo.Name, processInfo.Process.Process.Pid), processInfo.Color)

			go func(proc *exec.Cmd, name string, pid int) {
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
					// Force kill the process
					o.log(fmt.Sprintf("💀 Force killing %s (timeout)...", name), "\x1b[31m")
					if proc.Process != nil {
						proc.Process.Kill()
					}
					// Also try taskkill as backup
					o.killProcessGroup(pid)
				}
			}(processInfo.Process, processInfo.Name, processInfo.Process.Process.Pid)
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
}
