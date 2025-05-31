package frontend

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/barisgit/goflux/config"
)

// ScriptGenerator generates frontend using script commands
type ScriptGenerator struct {
	script   string
	frontend *config.FrontendConfig
	debug    bool
}

// NewScriptGenerator creates a new script generator
func NewScriptGenerator(script string, frontend *config.FrontendConfig, debug bool) *ScriptGenerator {
	return &ScriptGenerator{
		script:   script,
		frontend: frontend,
		debug:    debug,
	}
}

// Generate generates the frontend using the script
func (g *ScriptGenerator) Generate(frontendPath string, projectConfig *config.ProjectConfig) error {
	if g.debug {
		fmt.Printf("🏗️  Generating frontend using script: %s\n", g.script)
	}

	// Create frontend directory
	if err := os.MkdirAll(frontendPath, 0755); err != nil {
		return fmt.Errorf("failed to create frontend directory: %w", err)
	}

	// Run the script
	cmd := exec.Command("sh", "-c", g.script)
	cmd.Dir = frontendPath
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run script: %w", err)
	}

	return nil
}

// CustomGenerator generates frontend using custom commands
type CustomGenerator struct {
	command  string
	workDir  string
	frontend *config.FrontendConfig
	debug    bool
}

// NewCustomGenerator creates a new custom generator
func NewCustomGenerator(command, workDir string, frontend *config.FrontendConfig, debug bool) *CustomGenerator {
	return &CustomGenerator{
		command:  command,
		workDir:  workDir,
		frontend: frontend,
		debug:    debug,
	}
}

// Generate generates the frontend using the custom command
func (g *CustomGenerator) Generate(frontendPath string, projectConfig *config.ProjectConfig) error {
	if g.debug {
		fmt.Printf("🏗️  Generating frontend using custom command: %s\n", g.command)
	}

	// Create frontend directory
	if err := os.MkdirAll(frontendPath, 0755); err != nil {
		return fmt.Errorf("failed to create frontend directory: %w", err)
	}

	// Determine working directory
	workDir := frontendPath
	if g.workDir != "" {
		workDir = g.workDir
	}

	// Replace template variables in command
	command := strings.ReplaceAll(g.command, "{{frontend_path}}", frontendPath)
	command = strings.ReplaceAll(command, "{{project_name}}", projectConfig.Name)

	// Run the command
	cmd := exec.Command("sh", "-c", command)
	cmd.Dir = workDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run custom command: %w", err)
	}

	return nil
}
