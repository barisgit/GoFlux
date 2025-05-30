package frontend

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/barisgit/goflux/internal/config"
)

// Generator interface defines how frontend templates are generated
type Generator interface {
	Generate(frontendPath string, projectConfig *config.ProjectConfig) error
	GetInstallCommand() string
	GetDescription() string
}

// HardcodedGenerator generates frontend from hardcoded templates
type HardcodedGenerator struct {
	template *TemplateInfo
	debug    bool
}

// NewHardcodedGenerator creates a new hardcoded generator
func NewHardcodedGenerator(template *TemplateInfo, debug bool) *HardcodedGenerator {
	return &HardcodedGenerator{
		template: template,
		debug:    debug,
	}
}

// Generate generates the frontend using the hardcoded template
func (g *HardcodedGenerator) Generate(frontendPath string, projectConfig *config.ProjectConfig) error {
	if g.debug {
		fmt.Printf("🏗️  Generating frontend using hardcoded template: %s\n", g.template.Name)
	}

	// Create frontend directory
	if err := os.MkdirAll(frontendPath, 0755); err != nil {
		return fmt.Errorf("failed to create frontend directory: %w", err)
	}

	// Check if this is a filesystem template (no install command)
	if g.template.InstallCmd == "" {
		// Copy from filesystem template
		return g.generateFromFilesystem(frontendPath, projectConfig)
	}

	// If template has an install command, run it (legacy behavior)
	cmd := exec.Command("sh", "-c", g.template.InstallCmd)
	cmd.Dir = frontendPath
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run install command: %w", err)
	}

	return nil
}

// generateFromFilesystem copies template files from the frontend-templates directory
func (g *HardcodedGenerator) generateFromFilesystem(frontendPath string, projectConfig *config.ProjectConfig) error {
	// Find the GoFlux CLI executable directory to locate templates
	templatePath, err := g.findTemplatePath()
	if err != nil {
		return fmt.Errorf("failed to locate template directory: %w", err)
	}

	templateDir := filepath.Join(templatePath, "frontend-templates", g.template.Name)

	// Check if template directory exists
	if _, err := os.Stat(templateDir); os.IsNotExist(err) {
		return fmt.Errorf("template directory not found: %s", templateDir)
	}

	if g.debug {
		fmt.Printf("📁 Copying template from: %s\n", templateDir)
	}

	// Prepare template variables
	vars := g.prepareTemplateVars(projectConfig)

	// Copy all files from template directory
	return filepath.Walk(templateDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Calculate relative path
		relPath, err := filepath.Rel(templateDir, path)
		if err != nil {
			return err
		}

		// Skip the root directory
		if relPath == "." {
			return nil
		}

		destPath := filepath.Join(frontendPath, relPath)

		if info.IsDir() {
			return os.MkdirAll(destPath, info.Mode())
		}

		// Check if file is a template (ends with .tmpl)
		if strings.HasSuffix(path, ".tmpl") {
			// Remove .tmpl extension from destination
			destPath = strings.TrimSuffix(destPath, ".tmpl")
			return g.processTemplateFile(path, destPath, vars)
		} else {
			// Copy file as-is
			return g.copyFile(path, destPath)
		}
	})
}

// findTemplatePath finds the directory containing frontend templates
func (g *HardcodedGenerator) findTemplatePath() (string, error) {
	// Get the directory of the current executable or working directory
	// This assumes templates are in the same directory as the CLI or in the working directory

	// Try current working directory first (for development)
	if _, err := os.Stat("frontend-templates"); err == nil {
		return ".", nil
	}

	// Try relative to executable
	execPath, err := os.Executable()
	if err == nil {
		execDir := filepath.Dir(execPath)
		templatePath := filepath.Join(execDir, "frontend-templates")
		if _, err := os.Stat(templatePath); err == nil {
			return execDir, nil
		}
	}

	return "", fmt.Errorf("frontend-templates directory not found")
}

// prepareTemplateVars prepares variables for template processing
func (g *HardcodedGenerator) prepareTemplateVars(projectConfig *config.ProjectConfig) map[string]interface{} {
	vars := make(map[string]interface{})

	vars["ProjectName"] = projectConfig.Name
	vars["project_name"] = strings.ToLower(projectConfig.Name)
	vars["PROJECT_NAME"] = strings.ToUpper(projectConfig.Name)
	vars["port"] = "{{port}}" // This will be replaced by the dev command later

	return vars
}

// processTemplateFile processes a file as a Go template
func (g *HardcodedGenerator) processTemplateFile(sourcePath, destPath string, vars map[string]interface{}) error {
	// Read template content
	content, err := os.ReadFile(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to read template file %s: %w", sourcePath, err)
	}

	// Parse and execute template
	tmpl, err := template.New(filepath.Base(sourcePath)).Parse(string(content))
	if err != nil {
		return fmt.Errorf("failed to parse template %s: %w", sourcePath, err)
	}

	// Create output file
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return fmt.Errorf("failed to create directory for %s: %w", destPath, err)
	}

	outFile, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create output file %s: %w", destPath, err)
	}
	defer outFile.Close()

	// Execute template
	if err := tmpl.Execute(outFile, vars); err != nil {
		return fmt.Errorf("failed to execute template %s: %w", sourcePath, err)
	}

	if g.debug {
		fmt.Printf("📝 Processed template: %s -> %s\n", sourcePath, destPath)
	}

	return nil
}

// copyFile copies a regular file
func (g *HardcodedGenerator) copyFile(sourcePath, destPath string) error {
	sourceFile, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to open source file %s: %w", sourcePath, err)
	}
	defer sourceFile.Close()

	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return fmt.Errorf("failed to create directory for %s: %w", destPath, err)
	}

	destFile, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create destination file %s: %w", destPath, err)
	}
	defer destFile.Close()

	if _, err := io.Copy(destFile, sourceFile); err != nil {
		return fmt.Errorf("failed to copy file %s to %s: %w", sourcePath, destPath, err)
	}

	if g.debug {
		fmt.Printf("📄 Copied file: %s -> %s\n", sourcePath, destPath)
	}

	return nil
}

// GetInstallCommand returns the install command for dependencies
func (g *HardcodedGenerator) GetInstallCommand() string {
	return "pnpm install"
}

// GetDescription returns a description of this generator
func (g *HardcodedGenerator) GetDescription() string {
	return fmt.Sprintf("Hardcoded template: %s", g.template.Description)
}

// ScriptGenerator generates frontend using pnpx/npm create scripts
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

// GetInstallCommand returns the install command for dependencies
func (g *ScriptGenerator) GetInstallCommand() string {
	return "pnpm install"
}

// GetDescription returns a description of this generator
func (g *ScriptGenerator) GetDescription() string {
	return fmt.Sprintf("Script-based template: %s", g.script)
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

// GetInstallCommand returns the install command for dependencies
func (g *CustomGenerator) GetInstallCommand() string {
	return "pnpm install"
}

// GetDescription returns a description of this generator
func (g *CustomGenerator) GetDescription() string {
	return fmt.Sprintf("Custom command: %s", g.command)
}

// RemoteGenerator generates frontend from remote templates
type RemoteGenerator struct {
	url      string
	template config.TemplateConfig
	debug    bool
}

// NewRemoteGenerator creates a new remote generator
func NewRemoteGenerator(url string, template config.TemplateConfig, debug bool) *RemoteGenerator {
	return &RemoteGenerator{
		url:      url,
		template: template,
		debug:    debug,
	}
}

// Generate generates the frontend from a remote template
func (g *RemoteGenerator) Generate(frontendPath string, projectConfig *config.ProjectConfig) error {
	if g.debug {
		fmt.Printf("🏗️  Generating frontend using remote template: %s\n", g.url)
	}

	// Download and extract remote template
	remoteManager := NewRemoteTemplateManager(g.debug)
	templatePath, err := remoteManager.Download(g.url, g.template.Version, g.template.Cache)
	if err != nil {
		return fmt.Errorf("failed to download remote template: %w", err)
	}

	// Generate frontend from template
	if err := remoteManager.Generate(templatePath, frontendPath, projectConfig, g.template.Vars); err != nil {
		return fmt.Errorf("failed to generate from remote template: %w", err)
	}

	return nil
}

// GetInstallCommand returns the install command for dependencies
func (g *RemoteGenerator) GetInstallCommand() string {
	// This might be determined from the remote template manifest
	return "pnpm install"
}

// GetDescription returns a description of this generator
func (g *RemoteGenerator) GetDescription() string {
	return fmt.Sprintf("Remote template: %s", g.url)
}
