package dev

import (
	"fmt"
)

func (o *DevOrchestrator) formatLog(processName, line, color string) {
	// Simple prefix formatting
	prefix := "[?]"
	switch processName {
	case "Frontend":
		prefix = "[F]"
	case "Backend":
		prefix = "[B]"
	}

	fmt.Printf("%s%s\x1b[0m %s\n", color, prefix, line)
}
