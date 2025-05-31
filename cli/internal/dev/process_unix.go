//go:build unix

package dev

import (
	"fmt"
	"os/exec"
	"strings"
	"syscall"
	"time"
)

func (o *DevOrchestrator) stopBackendProcessUnsafe() {
	if o.backendProcess == nil || o.backendProcess.Process == nil {
		return
	}

	pid := o.backendProcess.Process.Pid
	o.log(fmt.Sprintf("🛑 Stopping backend (PID: %d)...", pid), "\x1b[34m")

	// Try graceful shutdown first
	o.backendProcess.Process.Signal(syscall.SIGTERM)

	// Wait up to 3 seconds for graceful shutdown
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
		syscall.Kill(pid, syscall.SIGKILL)
	}

	o.backendProcess = nil
}

func (o *DevOrchestrator) killProcessGroup(pid int) error {
	pgid, err := syscall.Getpgid(pid)
	if err != nil {
		pgid = pid // fallback to PID if can't get PGID
	}
	return syscall.Kill(-pgid, syscall.SIGTERM)
}

func (o *DevOrchestrator) forceKillProcessGroup(pid int) error {
	pgid, err := syscall.Getpgid(pid)
	if err != nil {
		pgid = pid // fallback to PID if can't get PGID
	}
	return syscall.Kill(-pgid, syscall.SIGKILL)
}

func (o *DevOrchestrator) forceKillPortProcesses() {
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
