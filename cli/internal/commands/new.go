package commands

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/barisgit/goflux/cli/internal/frontend"
	"github.com/barisgit/goflux/cli/internal/templates"
	"github.com/barisgit/goflux/config"

	"github.com/AlecAivazis/survey/v2"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// FrontendSelection contains both frontend config and API client config from template
type FrontendSelection struct {
	FrontendConfig  config.FrontendConfig
	APIClientConfig *config.APIClientConfig
}

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "new [project-name]",
		Short: "Create a new GoFlux project",
		Long:  "Create a new full-stack project with Go backend and TypeScript frontend",
		Args:  cobra.MaximumNArgs(1),
		RunE:  runNew,
	}

	cmd.Flags().Bool("debug", false, "Enable debug logging")
	cmd.Flags().String("template", "default", "Backend template name")
	cmd.Flags().String("frontend", "", "Frontend template name")
	cmd.Flags().String("frontend-type", "", "Frontend type (template, script, custom)")
	cmd.Flags().String("router", "", "Backend router (chi, gin, fiber, etc.)")

	return cmd
}

func runNew(cmd *cobra.Command, args []string) error {
	debug, _ := cmd.Flags().GetBool("debug")
	templateName, _ := cmd.Flags().GetString("template")
	frontendName, _ := cmd.Flags().GetString("frontend")
	frontendType, _ := cmd.Flags().GetString("frontend-type")
	router, _ := cmd.Flags().GetString("router")

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

	// Create temporary config for unified manager
	tempConfig := &config.ProjectConfig{Name: projectName}
	unifiedManager, err := frontend.NewUnifiedManager(tempConfig, debug)
	if err != nil {
		return fmt.Errorf("failed to initialize frontend manager: %w", err)
	}

	// Backend template selection
	if templateName == "default" {
		// Interactive template selection
		availableTemplates := unifiedManager.GetAvailableTemplates()
		if len(availableTemplates) > 0 {
			var templateOptions []string
			for name, desc := range availableTemplates {
				templateOptions = append(templateOptions, fmt.Sprintf("%s - %s", name, desc))
			}

			// Add external template option
			templateOptions = append(templateOptions, "external - Use external template (GitHub repo or local path)")

			var selectedTemplate string
			templatePrompt := &survey.Select{
				Message: "Choose backend template:",
				Options: templateOptions,
				Default: "default - " + availableTemplates["default"],
			}
			if err := survey.AskOne(templatePrompt, &selectedTemplate); err != nil {
				return err
			}

			if strings.HasPrefix(selectedTemplate, "external") {
				// Handle external template
				var templateSource string
				sourcePrompt := &survey.Input{
					Message: "Enter template source (GitHub URL or local path):",
					Help:    "Examples: https://github.com/user/repo or /path/to/template",
				}
				if err := survey.AskOne(sourcePrompt, &templateSource); err != nil {
					return err
				}

				// Load external template
				externalTemplate, err := loadExternalTemplate(templateSource)
				if err != nil {
					return fmt.Errorf("failed to load external template: %w", err)
				}

				templateName = "external"
				// Store the external template info for later use
				tempConfig.ExternalTemplate = &config.ExternalTemplateConfig{
					Source:      templateSource,
					Name:        externalTemplate.Name,
					Description: externalTemplate.Description,
				}
			} else {
				templateName = strings.Split(selectedTemplate, " - ")[0]
			}
		}
	}

	// Update temp config with selected template for frontend selection
	tempConfig.Backend.Template = templateName

	// Frontend selection
	var frontendConfig config.FrontendConfig
	if frontendName != "" && frontendType != "" {
		// Use command line flags
		frontendConfig = createFrontendConfigFromFlags(frontendType, frontendName, unifiedManager)
	} else {
		// Interactive selection
		frontendConfig, err = selectFrontendInteractive(unifiedManager)
		if err != nil {
			return err
		}
	}

	// Backend router selection
	if router == "" {
		var template *templates.TemplateManifest
		var err error

		if templateName == "external" && tempConfig.ExternalTemplate != nil {
			// Load external template for router selection
			template, err = loadExternalTemplate(tempConfig.ExternalTemplate.Source)
			if err != nil {
				return fmt.Errorf("failed to load external template for router selection: %w", err)
			}
		} else {
			// Get supported routers from regular template
			templateManager, err := templates.GetTemplateManager()
			if err != nil {
				return fmt.Errorf("failed to get template manager: %w", err)
			}

			var exists bool
			template, exists = templateManager.GetTemplate(templateName)
			if !exists {
				return fmt.Errorf("template %s not found", templateName)
			}
		}

		// Define router options with clean values and descriptions
		type RouterOption struct {
			Value       string
			Description string
		}

		var routerMap = map[string]RouterOption{
			"chi":      {Value: "chi", Description: "Chi (Recommended)"},
			"fiber":    {Value: "fiber", Description: "Fiber"},
			"gin":      {Value: "gin", Description: "Gin"},
			"echo":     {Value: "echo", Description: "Echo"},
			"gorilla":  {Value: "gorilla", Description: "Gorilla Mux"},
			"mux":      {Value: "mux", Description: "Go 1.22+ ServeMux"},
			"fasthttp": {Value: "fasthttp", Description: "FastHTTP"},
		}

		// Create options list with only supported routers
		var routerOptions []string
		var routerLookup = make(map[string]string) // description -> value

		for _, supportedRouter := range template.Backend.SupportedRouters {
			if option, exists := routerMap[supportedRouter]; exists {
				routerOptions = append(routerOptions, option.Description)
				routerLookup[option.Description] = option.Value
			} else {
				// Fallback for unknown routers
				routerOptions = append(routerOptions, supportedRouter)
				routerLookup[supportedRouter] = supportedRouter
			}
		}

		var selectedDescription string
		routerPrompt := &survey.Select{
			Message: "Choose backend router:",
			Options: routerOptions,
			Default: routerOptions[0], // First supported router
		}
		if err := survey.AskOne(routerPrompt, &selectedDescription); err != nil {
			return err
		}

		// Get the clean router value
		router = routerLookup[selectedDescription]
		if router == "" {
			router = selectedDescription // Fallback
		}
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
	cfg := generateProjectConfig(projectName, frontendConfig, router, templateName, tempConfig, unifiedManager)
	if err := writeConfig(filepath.Join(projectName, "flux.yaml"), cfg); err != nil {
		return err
	}

	// Create project structure using unified template system
	if err := createProjectStructure(projectName, templateName, router, debug); err != nil {
		return err
	}

	// Setup frontend if it's a template-based frontend
	if shouldSetupFrontendDuringCreation(frontendConfig) {
		fmt.Printf("📦 Setting up frontend from template...\n")

		// Create unified manager with the full config
		unifiedManager, err := frontend.NewUnifiedManager(&cfg, debug)
		if err != nil {
			return fmt.Errorf("failed to create unified manager: %w", err)
		}

		frontendPath := filepath.Join(projectName, "frontend")
		if err := unifiedManager.GenerateFrontend(frontendPath); err != nil {
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

func createFrontendConfigFromFlags(frontendType, frontendName string, unifiedManager *frontend.UnifiedManager) config.FrontendConfig {
	// Default config
	frontendConfig := config.FrontendConfig{
		Framework: frontendName,
		DevCmd:    "cd frontend && pnpm dev --port {{port}} --host",
		BuildCmd:  "cd frontend && pnpm build",
		TypesDir:  "src/types",
		LibDir:    "src/lib",
		Template: config.TemplateConfig{
			Type:   frontendType,
			Source: frontendName,
		},
		StaticGen: config.StaticGenConfig{
			Enabled:    false,
			SPARouting: true,
		},
	}

	// Get better defaults from template or script registry
	switch frontendType {
	case "template":
		// Get from template-based frontends
		supportedFrontends, _, err := unifiedManager.GetSupportedFrontends()
		if err == nil {
			for _, option := range supportedFrontends {
				if option.Name == frontendName || option.Framework == frontendName {
					frontendConfig.Framework = option.Framework
					frontendConfig.DevCmd = option.DevCmd
					frontendConfig.BuildCmd = option.BuildCmd
					frontendConfig.TypesDir = option.TypesDir
					frontendConfig.LibDir = option.LibDir
					frontendConfig.StaticGen = option.StaticGen
					break
				}
			}
		}
	case "script":
		// Get from script registry
		if unifiedManager.IsScriptFramework(frontendName) {
			// The unified manager will handle getting the right script
			scriptFrameworks := unifiedManager.GetScriptFrontends()
			for _, framework := range scriptFrameworks {
				if framework.Name == frontendName {
					frontendConfig.Framework = framework.Framework
					frontendConfig.DevCmd = framework.DevCmd
					frontendConfig.BuildCmd = framework.BuildCmd
					frontendConfig.TypesDir = framework.TypesDir
					frontendConfig.LibDir = framework.LibDir
					frontendConfig.Template.Source = framework.Script
					break
				}
			}
		}
	}

	return frontendConfig
}

func selectFrontendInteractive(unifiedManager *frontend.UnifiedManager) (config.FrontendConfig, error) {
	// First, ask for frontend type
	var frontendType string
	typePrompt := &survey.Select{
		Message: "Choose frontend generation method:",
		Options: []string{
			"Template-based (Built into backend template)",
			"Script-based (Popular frameworks via package managers)",
			"Custom Command",
		},
		Default: "Template-based (Built into backend template)",
	}
	if err := survey.AskOne(typePrompt, &frontendType); err != nil {
		return config.FrontendConfig{}, err
	}

	switch {
	case strings.Contains(frontendType, "Template-based"):
		return selectTemplateFrontend(unifiedManager)
	case strings.Contains(frontendType, "Script-based"):
		return selectScriptFrontend(unifiedManager)
	case strings.Contains(frontendType, "Custom"):
		return selectCustomFrontend()
	default:
		return selectTemplateFrontend(unifiedManager)
	}
}

func selectTemplateFrontend(unifiedManager *frontend.UnifiedManager) (config.FrontendConfig, error) {
	supportedFrontends, _, err := unifiedManager.GetSupportedFrontends()
	if err != nil {
		return config.FrontendConfig{}, err
	}

	if len(supportedFrontends) == 0 {
		return config.FrontendConfig{}, fmt.Errorf("no template-based frontends available")
	}

	var frontendOptions []string
	for _, frontend := range supportedFrontends {
		frontendOptions = append(frontendOptions, fmt.Sprintf("%s - %s", frontend.Name, frontend.Description))
	}

	var selectedFrontend string
	prompt := &survey.Select{
		Message: "Choose template frontend:",
		Options: frontendOptions,
		Default: frontendOptions[0],
	}
	if err := survey.AskOne(prompt, &selectedFrontend); err != nil {
		return config.FrontendConfig{}, err
	}

	// Extract frontend name
	frontendName := strings.Split(selectedFrontend, " - ")[0]

	// Find the selected frontend
	for _, frontend := range supportedFrontends {
		if frontend.Name == frontendName {
			frontendConfig := config.FrontendConfig{
				Framework: frontend.Framework,
				DevCmd:    frontend.DevCmd,
				BuildCmd:  frontend.BuildCmd,
				TypesDir:  frontend.TypesDir,
				LibDir:    frontend.LibDir,
				Template: config.TemplateConfig{
					Type:   "template",
					Source: frontend.Name,
				},
				StaticGen: frontend.StaticGen,
			}

			// Store the API client config from the template in a global variable or pass it through
			// For now, we'll return it and handle it in the calling function
			return frontendConfig, nil
		}
	}

	return config.FrontendConfig{}, fmt.Errorf("frontend %s not found", frontendName)
}

func selectScriptFrontend(unifiedManager *frontend.UnifiedManager) (config.FrontendConfig, error) {
	// Check if script frontends are supported
	_, scriptSupported, err := unifiedManager.GetSupportedFrontends()
	if err != nil {
		return config.FrontendConfig{}, err
	}

	if !scriptSupported {
		return config.FrontendConfig{}, fmt.Errorf("script-based frontends are not supported by the current template")
	}

	// Get script categories
	categories := unifiedManager.GetScriptCategories()
	if len(categories) == 0 {
		return config.FrontendConfig{}, fmt.Errorf("no script frontends available")
	}

	// First select category
	var categoryNames []string
	for _, category := range categories {
		categoryNames = append(categoryNames, category.Name)
	}

	var selectedCategory string
	categoryPrompt := &survey.Select{
		Message: "Choose frontend category:",
		Options: categoryNames,
		Default: "React",
	}
	if err := survey.AskOne(categoryPrompt, &selectedCategory); err != nil {
		return config.FrontendConfig{}, err
	}

	// Get frameworks for selected category
	var selectedCategoryFrameworks []frontend.ScriptFramework
	for _, category := range categories {
		if category.Name == selectedCategory {
			selectedCategoryFrameworks = category.Frameworks
			break
		}
	}

	if len(selectedCategoryFrameworks) == 0 {
		return config.FrontendConfig{}, fmt.Errorf("no frameworks found in category %s", selectedCategory)
	}

	// Select framework
	var frameworkOptions []string
	for _, framework := range selectedCategoryFrameworks {
		frameworkOptions = append(frameworkOptions, fmt.Sprintf("%s - %s", framework.DisplayName, framework.Description))
	}

	var selectedFramework string
	frameworkPrompt := &survey.Select{
		Message: "Choose framework:",
		Options: frameworkOptions,
		Default: frameworkOptions[0],
	}
	if err := survey.AskOne(frameworkPrompt, &selectedFramework); err != nil {
		return config.FrontendConfig{}, err
	}

	// Extract framework name and find the framework
	displayName := strings.Split(selectedFramework, " - ")[0]
	for _, framework := range selectedCategoryFrameworks {
		if framework.DisplayName == displayName {
			return config.FrontendConfig{
				Framework: framework.Framework,
				DevCmd:    framework.DevCmd,
				BuildCmd:  framework.BuildCmd,
				TypesDir:  framework.TypesDir,
				LibDir:    framework.LibDir,
				Template: config.TemplateConfig{
					Type:   "script",
					Source: framework.Script,
				},
				StaticGen: config.StaticGenConfig{
					Enabled:    false,
					SPARouting: true,
				},
			}, nil
		}
	}

	return config.FrontendConfig{}, fmt.Errorf("framework not found")
}

func selectCustomFrontend() (config.FrontendConfig, error) {
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

func generateProjectConfig(name string, frontendConfig config.FrontendConfig, backend, templateName string, tempConfig *config.ProjectConfig, unifiedManager *frontend.UnifiedManager) config.ProjectConfig {
	// Determine the appropriate static directory based on frontend framework
	staticDir := "frontend/dist" // Default for most frameworks

	// Next.js with static export outputs to 'out' by default
	if frontendConfig.Framework == "nextjs" {
		staticDir = "frontend/out"
	}

	// Use API client config from template frontend if available, otherwise use default
	var apiClientConfig config.APIClientConfig
	foundApiConfig := false

	if frontendConfig.Template.Type == "template" {
		supportedFrontends, _, err := unifiedManager.GetSupportedFrontends()
		if err == nil {
			for _, frontend := range supportedFrontends {
				// Try multiple matching criteria to find the right frontend
				nameMatch := frontend.Name == frontendConfig.Template.Source
				frameworkMatch := frontend.Framework == frontendConfig.Framework
				sourceMatch := frontend.Name == frontendConfig.Framework

				if nameMatch || frameworkMatch || sourceMatch {
					if frontend.APIClient.Generator != "" || frontend.APIClient.ReactQuery.Enabled {
						apiClientConfig = frontend.APIClient
						foundApiConfig = true
						break
					}
				}
			}
		}
	}
	if !foundApiConfig {
		apiClientConfig = config.GetDefaultAPIClientConfig()
	}

	return config.ProjectConfig{
		Name:     name,
		Port:     3000,
		Frontend: frontendConfig,
		Backend: config.BackendConfig{
			Router:   backend,
			Template: templateName,
		},
		Build: config.BuildConfig{
			OutputDir:   "dist",
			BinaryName:  "server",
			EmbedStatic: true,
			StaticDir:   staticDir,
			BuildTags:   "embed_static",
			LDFlags:     "-s -w",
			CGOEnabled:  false,
		},
		APIClient:        apiClientConfig,
		ExternalTemplate: tempConfig.ExternalTemplate,
	}
}

func writeConfig(path string, cfg config.ProjectConfig) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	return os.WriteFile(path, data, 0644)
}

func createProjectStructure(projectName, templateName, backend string, debug bool) error {
	fmt.Printf("📦 Generating project from template '%s'...\n", templateName)

	// Get template manager and generate project
	templateManager, err := templates.GetTemplateManager()
	if err != nil {
		return fmt.Errorf("failed to get template manager: %w", err)
	}

	customVars := make(map[string]interface{})
	if err := templateManager.GenerateProject(templateName, projectName, projectName, backend, customVars); err != nil {
		return fmt.Errorf("failed to generate project from template: %w", err)
	}

	fmt.Printf("📦 Backend structure created. Frontend will be set up when you run 'flux dev'...\n")
	return nil
}

func shouldSetupFrontendDuringCreation(frontendConfig config.FrontendConfig) bool {
	// Only setup during creation for template-based frontends
	// Script and custom frontends should be generated during dev command
	return frontendConfig.Template.Type == "template" || frontendConfig.Template.Type == ""
}

func loadExternalTemplate(source string) (*templates.TemplateManifest, error) {
	// Check if it's a local path
	if strings.HasPrefix(source, "/") || strings.HasPrefix(source, "./") || strings.HasPrefix(source, "../") {
		// Local path - look for template.yaml
		templatePath := filepath.Join(source, "template.yaml")
		data, err := os.ReadFile(templatePath)
		if err != nil {
			return nil, fmt.Errorf("failed to read template.yaml from %s: %w", templatePath, err)
		}

		return templates.LoadTemplateFromData(data)
	}

	// Check if it's a GitHub URL
	if strings.Contains(source, "github.com") {
		// Extract owner/repo from URL
		parts := strings.Split(source, "/")
		if len(parts) < 5 {
			return nil, fmt.Errorf("invalid GitHub URL format. Expected: https://github.com/owner/repo")
		}

		owner := parts[3]
		repo := parts[4]

		// Try to fetch template.yaml from main branch first, then master
		branches := []string{"main", "master"}

		for _, branch := range branches {
			templateURL := fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s/template.yaml", owner, repo, branch)

			resp, err := http.Get(templateURL)
			if err != nil {
				continue
			}
			defer resp.Body.Close()

			if resp.StatusCode == http.StatusOK {
				data, err := io.ReadAll(resp.Body)
				if err != nil {
					return nil, fmt.Errorf("failed to read template.yaml: %w", err)
				}

				return templates.LoadTemplateFromData(data)
			}
		}

		return nil, fmt.Errorf("could not find template.yaml in repository %s/%s (tried main and master branches)", owner, repo)
	}

	return nil, fmt.Errorf("unsupported template source format. Use GitHub URL (https://github.com/owner/repo) or local path")
}
