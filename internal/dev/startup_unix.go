//go:build unix

package dev

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"
)

func (o *DevOrchestrator) setupProcessGroup(cmd *exec.Cmd) {
	// Set process group for processes so we can kill the entire process tree
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}
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

func (o *DevOrchestrator) shutdownProcesses() {
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
}
