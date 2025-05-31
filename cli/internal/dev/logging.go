package dev

import (
	"fmt"
	"strings"
)

func (o *DevOrchestrator) formatLog(processName, line, color string) {
	// Skip noisy logs
	if strings.Contains(line, "watching") ||
		strings.Contains(line, "!exclude") ||
		strings.Contains(line, "building...") ||
		strings.Contains(line, "running...") {
		return
	}

	// Handle specific log formats
	if processName == "Frontend" {
		if strings.Contains(line, "VITE v") {
			o.log("⚡ Frontend: Vite ready", color)
		} else if strings.Contains(line, "Local:") {
			o.log("🌐 Frontend: Dev server ready", color)
		} else if strings.Contains(line, "Network:") {
			o.log("🌍 Frontend: Network access ready", color)
		} else if strings.TrimSpace(line) != "" {
			o.log(fmt.Sprintf("⚡ Frontend: %s", line), color)
		}
	} else if processName == "Backend" {
		// Check if this is a Huma-style log (contains HTTP request info)
		if o.isHumaLog(line) {
			// Pass Huma logs through directly to preserve their native formatting and colors
			fmt.Println(line)
			return
		}

		// Show HTTP request logs (contain | separators for Fiber logs) - keep native format
		if strings.Count(line, "|") >= 4 {
			o.formatHttpLog(line)
		} else if strings.Contains(line, "Server starting") ||
			strings.Contains(line, "Fiber") ||
			strings.Contains(line, "http://") ||
			strings.Contains(line, "Handlers") ||
			strings.Contains(line, "Processes") ||
			strings.Contains(line, "PID") ||
			strings.Contains(line, "├") ||
			strings.Contains(line, "│") ||
			strings.Contains(line, "└") ||
			strings.Contains(line, "┌") ||
			strings.Contains(line, "┐") ||
			strings.Contains(line, "─") {
			o.log(fmt.Sprintf("🔧 Backend: %s", line), color)
		} else if strings.TrimSpace(line) != "" && !strings.Contains(line, "bound on host") {
			// Show other backend logs but not the "bound on host" message
			o.log(fmt.Sprintf("🔧 Backend: %s", line), color)
		}
	} else {
		if strings.TrimSpace(line) != "" {
			o.log(fmt.Sprintf("%s: %s", processName, line), color)
		}
	}
}

// isHumaLog detects if a log line is from Huma based on its characteristic format
func (o *DevOrchestrator) isHumaLog(line string) bool {
	// Huma logs typically contain:
	// - A timestamp in format "2006/01/02 15:04:05"
	// - HTTP method and URL in quotes
	// - "from" keyword
	// - Status code, size, and duration

	// Check for characteristic Huma log patterns
	if strings.Contains(line, "\"GET ") ||
		strings.Contains(line, "\"POST ") ||
		strings.Contains(line, "\"PUT ") ||
		strings.Contains(line, "\"DELETE ") ||
		strings.Contains(line, "\"PATCH ") ||
		strings.Contains(line, "\"HEAD ") ||
		strings.Contains(line, "\"OPTIONS ") {

		// Additional validation: check for "from" and typical status/timing pattern
		if strings.Contains(line, " from ") &&
			(strings.Contains(line, " - ") || strings.Contains(line, " in ")) {
			return true
		}
	}

	// Also check for Huma startup messages
	if strings.Contains(line, "server starting on") ||
		strings.Contains(line, "API documentation available") ||
		strings.Contains(line, "OpenAPI spec available") {
		return true
	}

	return false
}

func (o *DevOrchestrator) formatHttpLog(line string) {
	// Parse HTTP log like dev.ts: "14:30:36 | 200 | 76.959µs | 127.0.0.1 | GET | /api/health"
	parts := strings.Split(line, "|")
	if len(parts) < 6 {
		// If it doesn't match expected format, just print as-is
		fmt.Println(line)
		return
	}

	// Trim whitespace from parts
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}

	timeStr := parts[0]
	status := parts[1]
	duration := parts[2]
	_ = parts[3] // ip (unused)
	method := parts[4]
	path := parts[5]

	// Color code by status
	statusColor := "\x1b[32m" // Green for 2xx
	if strings.HasPrefix(status, "3") {
		statusColor = "\x1b[33m" // Yellow for 3xx
	} else if strings.HasPrefix(status, "4") {
		statusColor = "\x1b[31m" // Red for 4xx
	} else if strings.HasPrefix(status, "5") {
		statusColor = "\x1b[35m" // Magenta for 5xx
	}

	// Color code by method
	methodColor := "\x1b[36m" // Cyan for GET
	if method == "POST" {
		methodColor = "\x1b[32m" // Green
	} else if method == "PUT" {
		methodColor = "\x1b[33m" // Yellow
	} else if method == "DELETE" {
		methodColor = "\x1b[31m" // Red
	}

	// Skip some noisy requests
	if strings.Contains(path, "/@vite/") || strings.Contains(path, "/node_modules/") || path == "/@react-refresh" {
		return
	}

	// Format the log nicely with colors
	fmt.Printf("\x1b[90m[%s]\x1b[0m %s%s\x1b[0m %s%-6s\x1b[0m \x1b[90m%10s\x1b[0m %s\n",
		timeStr, statusColor, status, methodColor, method, duration, path)
}
