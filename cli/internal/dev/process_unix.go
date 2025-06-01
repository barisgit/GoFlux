//go:build unix

package dev

import (
	"fmt"
	"os/exec"
	"strings"
	"syscall"
)

// setupProcessGroup sets up process group for Unix systems
func (o *DevOrchestrator) setupProcessGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
		Pgid:    0,
	}
}

// shutdownProcesses handles graceful shutdown of frontend and other processes on Unix
func (o *DevOrchestrator) shutdownProcesses() {
	done := make(chan bool, len(o.processes))

	// Try graceful shutdown of all processes
	for _, processInfo := range o.processes {
		if processInfo.Process != nil && processInfo.Process.Process != nil {
			o.log(fmt.Sprintf("🛑 Stopping %s (PID: %d)...", processInfo.Name, processInfo.Process.Process.Pid), processInfo.Color)

			// Get process group ID
			pgid, err := syscall.Getpgid(processInfo.Process.Process.Pid)
			if err != nil {
				pgid = processInfo.Process.Process.Pid // fallback to PID if can't get PGID
			}

			// Send SIGTERM to process group
			syscall.Kill(-pgid, syscall.SIGTERM)

			go func(proc *exec.Cmd, name string, processGroupID int) {
				defer func() { done <- true }()

				// Wait for graceful shutdown
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
				case <-o.ctx.Done():
					// Context cancelled - force kill
					o.log(fmt.Sprintf("💀 Force killing %s and children...", name), "\x1b[31m")
					syscall.Kill(-processGroupID, syscall.SIGKILL)
				}
			}(processInfo.Process, processInfo.Name, pgid)
		} else {
			done <- true // No process to stop
		}
	}

	// Wait for all processes to finish
	for i := 0; i < len(o.processes); i++ {
		select {
		case <-done:
			// Process finished
		case <-o.ctx.Done():
			// Context cancelled, stop waiting
			return
		}
	}
}

// forceKillPortProcesses kills any processes using the configured ports on Unix
func (pm *ProcessManager) forceKillPortProcesses() {
	o := pm.orchestrator
	ports := []int{o.config.Port, o.frontendPort, o.backendPort}

	for _, port := range ports {
		if port == 0 {
			continue
		}

		// Use lsof to find processes listening on the port
		cmd := exec.Command("lsof", "-ti", fmt.Sprintf(":%d", port))
		output, err := cmd.Output()
		if err != nil {
			continue // No process found on this port
		}

		pids := strings.Fields(strings.TrimSpace(string(output)))
		for _, pidStr := range pids {
			if pidStr == "" {
				continue
			}

			// Kill each PID
			killCmd := exec.Command("kill", "-9", pidStr)
			if err := killCmd.Run(); err == nil {
				o.log(fmt.Sprintf("💀 Force killed process %s on port %d", pidStr, port), "\x1b[31m")
			}
		}
	}
}
