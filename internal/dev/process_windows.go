//go:build windows

package dev

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

func (o *DevOrchestrator) stopBackendProcessUnsafe() {
	if o.backendProcess == nil || o.backendProcess.Process == nil {
		return
	}

	pid := o.backendProcess.Process.Pid
	o.log(fmt.Sprintf("🛑 Stopping backend (PID: %d)...", pid), "\x1b[34m")

	// Wait up to 3 seconds for process to exit
	done := make(chan error, 1)
	go func() {
		done <- o.backendProcess.Wait()
	}()

	select {
	case err := <-done:
		if err != nil {
			o.log("⚠️  Backend exited with error", "\x1b[33m")
		} else {
			o.log("✅ Backend stopped gracefully", "\x1b[32m")
		}
	case <-time.After(3 * time.Second):
		o.log("💀 Force killing backend (timeout)...", "\x1b[31m")
		o.backendProcess.Process.Kill()
	}

	o.backendProcess = nil
}

func (o *DevOrchestrator) killProcessGroup(pid int) error {
	// On Windows, use taskkill to terminate process
	cmd := exec.Command("taskkill", "/F", "/T", "/PID", strconv.Itoa(pid))
	return cmd.Run()
}

func (o *DevOrchestrator) forceKillProcessGroup(pid int) error {
	// On Windows, force kill is the same as regular kill
	return o.killProcessGroup(pid)
}

func (o *DevOrchestrator) forceKillPortProcesses() {
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
