package frontend

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/barisgit/goflux/internal/config"
)

// Manager handles frontend generation and setup
type Manager struct {
	config    *config.ProjectConfig
	templates *TemplateRegistry
	resolver  *TemplateResolver
	debug     bool
}

// NewManager creates a new frontend manager
func NewManager(cfg *config.ProjectConfig, debug bool) *Manager {
	registry := NewTemplateRegistry()
	resolver := NewTemplateResolver(registry, debug)

	return &Manager{
		config:    cfg,
		templates: registry,
		resolver:  resolver,
		debug:     debug,
	}
}

// Setup sets up the frontend based on the configuration
func (m *Manager) Setup(projectPath string) error {
	frontendPath := filepath.Join(projectPath, "frontend")

	// Check if frontend already exists
	if m.frontendExists(frontendPath) {
		if m.debug {
			fmt.Printf("📁 Frontend directory already exists at %s\n", frontendPath)
		}
		return nil
	}

	// Determine how to generate the frontend
	generator, err := m.resolver.ResolveTemplate(&m.config.Frontend)
	if err != nil {
		return fmt.Errorf("failed to resolve frontend template: %w", err)
	}

	// Generate the frontend
	if err := generator.Generate(frontendPath, m.config); err != nil {
		return fmt.Errorf("failed to generate frontend: %w", err)
	}

	return nil
}

// frontendExists checks if a frontend directory with package.json exists
func (m *Manager) frontendExists(frontendPath string) bool {
	packageJsonPath := filepath.Join(frontendPath, "package.json")
	_, err := os.Stat(packageJsonPath)
	return err == nil
}

// GetInstallCommand returns the command to install frontend dependencies
func (m *Manager) GetInstallCommand() string {
	// Check if we have a template-based setup
	if m.config.Frontend.Template.Type != "" {
		generator, err := m.resolver.ResolveTemplate(&m.config.Frontend)
		if err == nil {
			if installCmd := generator.GetInstallCommand(); installCmd != "" {
				return installCmd
			}
		}
	}

	// Fall back to legacy install_cmd
	if m.config.Frontend.InstallCmd != "" {
		return m.config.Frontend.InstallCmd
	}

	// Default to pnpm install
	return "pnpm install"
}

// GetDevCommand returns the development command for the frontend
func (m *Manager) GetDevCommand(port int) string {
	devCmd := m.config.Frontend.DevCmd

	// Replace port placeholders
	devCmd = strings.ReplaceAll(devCmd, "3001", fmt.Sprintf("%d", port))
	devCmd = strings.ReplaceAll(devCmd, "{{port}}", fmt.Sprintf("%d", port))

	return devCmd
}

// GetBuildCommand returns the build command for the frontend
func (m *Manager) GetBuildCommand() string {
	return m.config.Frontend.BuildCmd
}

// ListAvailableTemplates returns all available hardcoded templates
func (m *Manager) ListAvailableTemplates() []string {
	return m.templates.ListHardcodedTemplates()
}

// GetTemplateRegistry returns the template registry for accessing all templates
func (m *Manager) GetTemplateRegistry() *TemplateRegistry {
	return m.templates
}
