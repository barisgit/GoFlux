package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/barisgit/goflux/internal/config"
	"github.com/barisgit/goflux/internal/frontend"
	"github.com/barisgit/goflux/internal/templates"

	"github.com/AlecAivazis/survey/v2"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "new [project-name]",
		Short: "Create a new GoFlux project",
		Long:  "Create a new full-stack project with Go backend and TypeScript frontend",
		Args:  cobra.MaximumNArgs(1),
		RunE:  runNew,
	}

	cmd.Flags().Bool("debug", false, "Enable debug logging")
	cmd.Flags().String("template", "", "Specify frontend template (hardcoded, script, custom, or remote)")
	cmd.Flags().String("template-source", "", "Template source (name, command, or URL)")
	cmd.Flags().String("framework", "", "Frontend framework name")

	return cmd
}

func runNew(cmd *cobra.Command, args []string) error {
	debug, _ := cmd.Flags().GetBool("debug")
	templateType, _ := cmd.Flags().GetString("template")
	templateSource, _ := cmd.Flags().GetString("template-source")
	framework, _ := cmd.Flags().GetString("framework")

	var projectName string

	if len(args) > 0 {
		projectName = args[0]
	} else {
		prompt := &survey.Input{
			Message: "Project name:",
			Default: "my-flux-app",
		}
		if err := survey.AskOne(prompt, &projectName); err != nil {
			return err
		}
	}

	// Create frontend manager to get available options
	tempConfig := &config.ProjectConfig{Name: projectName}
	frontendManager := frontend.NewManager(tempConfig, debug)

	// Frontend selection
	var frontendConfig config.FrontendConfig
	if templateType != "" && templateSource != "" {
		// Use command line flags
		frontendConfig = createFrontendConfigFromFlags(templateType, templateSource, framework)
	} else {
		// Interactive selection
		var err error
		frontendConfig, err = selectFrontendInteractive(frontendManager)
		if err != nil {
			return err
		}
	}

	// Backend router selection
	var backendRouter string
	routerPrompt := &survey.Select{
		Message: "Choose backend router:",
		Options: []string{
			"Chi (Recommended)",
			"Fiber",
			"Gin",
			"Echo",
			"Go 1.22+ ServeMux",
			"Gorilla Mux",
		},
		Default: "Chi (Recommended)",
	}
	if err := survey.AskOne(routerPrompt, &backendRouter); err != nil {
		return err
	}

	// Create project directory
	if err := os.MkdirAll(projectName, 0755); err != nil {
		return fmt.Errorf("failed to create project directory: %w", err)
	}

	// Initialize git repository
	if err := exec.Command("git", "init", projectName).Run(); err != nil {
		return fmt.Errorf("failed to initialize git repository: %w", err)
	}

	// Generate project config
	cfg := generateProjectConfig(projectName, frontendConfig, backendRouter)
	if err := writeConfig(filepath.Join(projectName, "flux.yaml"), cfg); err != nil {
		return err
	}

	// Create project structure
	if err := createProjectStructure(projectName, backendRouter, debug); err != nil {
		return err
	}

	// Setup frontend if it's a hardcoded template without install command
	if shouldSetupFrontendDuringCreation(frontendConfig, &cfg) {
		fmt.Printf("📦 Setting up frontend from template...\n")

		// Create frontend manager and setup frontend
		frontendManager := frontend.NewManager(&cfg, debug)

		if err := frontendManager.Setup(projectName); err != nil {
			return fmt.Errorf("failed to setup frontend: %w", err)
		}

		fmt.Printf("✅ Frontend setup complete!\n")
	}

	fmt.Printf("\n🎉 Created %s successfully!\n\n", projectName)
	fmt.Printf("Next steps:\n")
	fmt.Printf("  cd %s\n", projectName)
	fmt.Printf("  flux dev\n\n")

	return nil
}

func createFrontendConfigFromFlags(templateType, templateSource, framework string) config.FrontendConfig {
	// Default config with fallbacks
	frontendConfig := config.FrontendConfig{
		Framework: framework,
		DevCmd:    "cd frontend && pnpm dev --port {{port}} --host", // Default fallback
		BuildCmd:  "cd frontend && pnpm build",
		TypesDir:  "src/types",
		LibDir:    "src/lib",
		Template: config.TemplateConfig{
			Type:   templateType,
			Source: templateSource,
		},
		StaticGen: config.StaticGenConfig{
			Enabled:    false,
			SPARouting: true,
		},
	}

	// Set framework-specific defaults if not specified
	if framework == "" {
		switch templateType {
		case "hardcoded":
			frontendConfig.Framework = templateSource
		default:
			frontendConfig.Framework = "custom"
		}
	}

	// If it's a hardcoded template, try to get better defaults from registry
	if templateType == "hardcoded" {
		tempConfig := &config.ProjectConfig{Name: "temp"}
		frontendManager := frontend.NewManager(tempConfig, false)
		allTemplates := frontendManager.GetTemplateRegistry().GetAllTemplates()

		if template, exists := allTemplates[templateSource]; exists {
			// Use template fields if available
			if template.DevCmd != "" {
				frontendConfig.DevCmd = template.DevCmd
			}
			if template.BuildCmd != "" {
				frontendConfig.BuildCmd = template.BuildCmd
			}
			if template.TypesDir != "" {
				frontendConfig.TypesDir = template.TypesDir
			}
			if template.LibDir != "" {
				frontendConfig.LibDir = template.LibDir
			}
			// Use template's StaticGen config
			frontendConfig.StaticGen = template.StaticGen
		}
	}

	return frontendConfig
}

func selectFrontendInteractive(frontendManager *frontend.Manager) (config.FrontendConfig, error) {
	// First, ask for template type
	var templateType string
	templatePrompt := &survey.Select{
		Message: "Choose frontend generation method:",
		Options: []string{
			"Hardcoded Template (File-based)",
			"Script Template (Install command)",
			// "Custom Command",
			// "Remote Template (GitHub/Local)",
		},
		Default: "Hardcoded Template (File-based)",
	}
	if err := survey.AskOne(templatePrompt, &templateType); err != nil {
		return config.FrontendConfig{}, err
	}

	switch {
	case strings.Contains(templateType, "Hardcoded"):
		return selectHardcodedTemplate(frontendManager)
	case strings.Contains(templateType, "Script"):
		return selectScriptTemplate(frontendManager)
	// case strings.Contains(templateType, "Custom"):
	// 	return selectCustomTemplate()
	// case strings.Contains(templateType, "Remote"):
	// 	return selectRemoteTemplate()
	default:
		return selectHardcodedTemplate(frontendManager)
	}
}

func selectHardcodedTemplate(frontendManager *frontend.Manager) (config.FrontendConfig, error) {
	// Get all templates and filter for hardcoded ones (no install command)
	allTemplates := frontendManager.GetTemplateRegistry().GetAllTemplates()
	var hardcodedTemplates []string
	var templateDescriptions []string

	for name, template := range allTemplates {
		if template.InstallCmd == "" { // Hardcoded = no install command
			hardcodedTemplates = append(hardcodedTemplates, name)
			templateDescriptions = append(templateDescriptions, fmt.Sprintf("%s - %s", name, template.Description))
		}
	}

	if len(hardcodedTemplates) == 0 {
		return config.FrontendConfig{}, fmt.Errorf("no hardcoded templates available")
	}

	var selectedTemplate string
	prompt := &survey.Select{
		Message: "Choose hardcoded template:",
		Options: templateDescriptions,
		Default: "default - " + allTemplates["default"].Description,
	}
	if err := survey.AskOne(prompt, &selectedTemplate); err != nil {
		return config.FrontendConfig{}, err
	}

	// Extract template name from "name - description" format
	templateName := strings.Split(selectedTemplate, " - ")[0]

	// Get the template to use all its fields
	template := allTemplates[templateName]

	// Use template fields with fallbacks
	devCmd := template.DevCmd
	if devCmd == "" {
		devCmd = "cd frontend && pnpm dev --port {{port}} --host"
	}

	buildCmd := template.BuildCmd
	if buildCmd == "" {
		buildCmd = "cd frontend && pnpm build"
	}

	typesDir := template.TypesDir
	if typesDir == "" {
		typesDir = "src/types"
	}

	libDir := template.LibDir
	if libDir == "" {
		libDir = "src/lib"
	}

	return config.FrontendConfig{
		Framework: templateName,
		DevCmd:    devCmd,
		BuildCmd:  buildCmd,
		TypesDir:  typesDir,
		LibDir:    libDir,
		Template: config.TemplateConfig{
			Type:   "hardcoded",
			Source: templateName,
		},
		StaticGen: template.StaticGen, // Use entire StaticGen config from template
	}, nil
}

func selectScriptTemplate(frontendManager *frontend.Manager) (config.FrontendConfig, error) {
	// Get all templates and filter for script ones (have install command)
	allTemplates := frontendManager.GetTemplateRegistry().GetAllTemplates()
	var scriptTemplates []string
	var templateDescriptions []string

	for name, template := range allTemplates {
		if template.InstallCmd != "" { // Script = has install command
			scriptTemplates = append(scriptTemplates, name)
			templateDescriptions = append(templateDescriptions, fmt.Sprintf("%s - %s", name, template.Description))
		}
	}

	if len(scriptTemplates) == 0 {
		return config.FrontendConfig{}, fmt.Errorf("no script templates available")
	}

	var selectedTemplate string
	prompt := &survey.Select{
		Message: "Choose script template:",
		Options: templateDescriptions,
	}
	if err := survey.AskOne(prompt, &selectedTemplate); err != nil {
		return config.FrontendConfig{}, err
	}

	// Extract template name from "name - description" format
	templateName := strings.Split(selectedTemplate, " - ")[0]

	// Get the template to use all its fields
	template := allTemplates[templateName]

	// Use template fields with fallbacks
	devCmd := template.DevCmd
	if devCmd == "" {
		devCmd = "cd frontend && pnpm dev --port {{port}} --host"
	}

	buildCmd := template.BuildCmd
	if buildCmd == "" {
		buildCmd = "cd frontend && pnpm build"
	}

	typesDir := template.TypesDir
	if typesDir == "" {
		typesDir = "src/types"
	}

	libDir := template.LibDir
	if libDir == "" {
		libDir = "src/lib"
	}

	return config.FrontendConfig{
		Framework: templateName,
		DevCmd:    devCmd,
		BuildCmd:  buildCmd,
		TypesDir:  typesDir,
		LibDir:    libDir,
		Template: config.TemplateConfig{
			Type:   "hardcoded", // Still hardcoded type, but with install command
			Source: templateName,
		},
		StaticGen: template.StaticGen, // Use entire StaticGen config from template
	}, nil
}

func selectCustomTemplate() (config.FrontendConfig, error) {
	var command string
	commandPrompt := &survey.Input{
		Message: "Enter custom command:",
		Help:    "Use {{frontend_path}} and {{project_name}} as placeholders",
	}
	if err := survey.AskOne(commandPrompt, &command); err != nil {
		return config.FrontendConfig{}, err
	}

	var workDir string
	dirPrompt := &survey.Input{
		Message: "Working directory (optional):",
		Default: "",
	}
	if err := survey.AskOne(dirPrompt, &workDir); err != nil {
		return config.FrontendConfig{}, err
	}

	return config.FrontendConfig{
		Framework: "custom",
		DevCmd:    "cd frontend && pnpm dev --port {{port}} --host",
		BuildCmd:  "cd frontend && pnpm build",
		TypesDir:  "src/types",
		LibDir:    "src/lib",
		Template: config.TemplateConfig{
			Type:    "custom",
			Command: command,
			Dir:     workDir,
		},
		StaticGen: config.StaticGenConfig{
			Enabled:    false,
			SPARouting: true,
		},
	}, nil
}

func selectRemoteTemplate() (config.FrontendConfig, error) {
	var url string
	urlPrompt := &survey.Input{
		Message: "Enter template URL or local path:",
		Help:    "GitHub URL (e.g., https://github.com/user/template) or local path",
	}
	if err := survey.AskOne(urlPrompt, &url); err != nil {
		return config.FrontendConfig{}, err
	}

	var version string
	versionPrompt := &survey.Input{
		Message: "Version/branch (optional):",
		Default: "main",
	}
	if err := survey.AskOne(versionPrompt, &version); err != nil {
		return config.FrontendConfig{}, err
	}

	var useCache bool
	cachePrompt := &survey.Confirm{
		Message: "Cache template locally?",
		Default: true,
	}
	if err := survey.AskOne(cachePrompt, &useCache); err != nil {
		return config.FrontendConfig{}, err
	}

	return config.FrontendConfig{
		Framework: "remote",
		DevCmd:    "cd frontend && pnpm dev --port {{port}} --host",
		BuildCmd:  "cd frontend && pnpm build",
		TypesDir:  "src/types",
		LibDir:    "src/lib",
		Template: config.TemplateConfig{
			Type:    "remote",
			URL:     url,
			Version: version,
			Cache:   useCache,
		},
		StaticGen: config.StaticGenConfig{
			Enabled:    false,
			SPARouting: true,
		},
	}, nil
}

func generateProjectConfig(name string, frontendConfig config.FrontendConfig, backend string) config.ProjectConfig {
	return config.ProjectConfig{
		Name:     name,
		Port:     3000,
		Frontend: frontendConfig,
		Backend: config.BackendConfig{
			Router: backend,
		},
		Build: config.BuildConfig{
			OutputDir:   "dist",
			BinaryName:  "server",
			EmbedStatic: true,
			StaticDir:   "frontend/dist",
			BuildTags:   "embed_static",
			LDFlags:     "-s -w",
			CGOEnabled:  false,
		},
	}
}

func writeConfig(path string, cfg config.ProjectConfig) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	return os.WriteFile(path, data, 0644)
}

func createProjectStructure(projectName, backend string, debug bool) error {
	fmt.Printf("📦 Generating project from templates...\n")

	// Use template system to generate the base project structure
	if err := templates.GenerateProject(projectName, projectName, backend); err != nil {
		return fmt.Errorf("failed to generate project from templates: %w", err)
	}

	// Frontend will be generated during dev command using the new system
	fmt.Printf("📦 Frontend will be set up when you run 'flux dev'...\n")

	return nil
}

func shouldSetupFrontendDuringCreation(frontendConfig config.FrontendConfig, projectConfig *config.ProjectConfig) bool {
	// Only setup during creation for hardcoded templates that use filesystem copying
	if frontendConfig.Template.Type != "hardcoded" {
		return false
	}

	// Create a temporary frontend manager to check template properties
	frontendManager := frontend.NewManager(projectConfig, false)
	allTemplates := frontendManager.GetTemplateRegistry().GetAllTemplates()

	if template, exists := allTemplates[frontendConfig.Template.Source]; exists {
		// Setup during creation only if template has no install command (filesystem template)
		return template.InstallCmd == ""
	}

	return false
}
