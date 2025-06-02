//go:build windows

package dev

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
)

// setupProcessGroup sets up process group for Windows systems
func (o *DevOrchestrator) setupProcessGroup(cmd *exec.Cmd) {
	// Windows doesn't have process groups like Unix, but we can set CREATE_NEW_PROCESS_GROUP
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP,
	}
}

// shutdownProcesses handles graceful shutdown of frontend and other processes on Windows
func (o *DevOrchestrator) shutdownProcesses() {
	done := make(chan bool, len(o.processes))

	// Try graceful shutdown of all processes
	for _, processInfo := range o.processes {
		if processInfo.Process != nil && processInfo.Process.Process != nil {
			o.log(fmt.Sprintf("🛑 Stopping %s (PID: %d)...", processInfo.Name, processInfo.Process.Process.Pid), processInfo.Color)

			go func(proc *exec.Cmd, name string, pid int) {
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
					// Context cancelled - force kill using taskkill
					o.log(fmt.Sprintf("💀 Force killing %s...", name), "\x1b[31m")
					killCmd := exec.Command("taskkill", "/F", "/T", "/PID", strconv.Itoa(pid))
					killCmd.Run()
				}
			}(processInfo.Process, processInfo.Name, processInfo.Process.Process.Pid)
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

// forceKillPortProcesses kills any processes using the configured ports on Windows
func (pm *ProcessManager) forceKillPortProcesses() {
	o := pm.orchestrator
	ports := []int{o.config.Port, o.frontendPort, o.backendPort}

	for _, port := range ports {
		if port == 0 {
			continue
		}

		// Use netstat to find processes listening on the port
		cmd := exec.Command("netstat", "-ano")
		output, err := cmd.Output()
		if err != nil {
			continue
		}

		lines := strings.Split(string(output), "\n")
		for _, line := range lines {
			if strings.Contains(line, fmt.Sprintf(":%d", port)) && strings.Contains(line, "LISTENING") {
				fields := strings.Fields(line)
				if len(fields) >= 5 {
					pidStr := fields[len(fields)-1]
					if pidStr != "" && pidStr != "0" {
						// Kill the PID using taskkill
						killCmd := exec.Command("taskkill", "/F", "/PID", pidStr)
						if err := killCmd.Run(); err == nil {
							o.log(fmt.Sprintf("💀 Force killed process %s on port %d", pidStr, port), "\x1b[31m")
						}
					}
				}
			}
		}
	}
}
