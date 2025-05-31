package frontend

import (
	"fmt"

	"github.com/barisgit/goflux/config"
	"github.com/barisgit/goflux/cli/internal/templates"
)

// UnifiedManager handles both template-based and script-based frontend generation
type UnifiedManager struct {
	projectConfig   *config.ProjectConfig
	templateManager *templates.Manager
	scriptManager   *ScriptRegistryManager
	debug           bool
}

// NewUnifiedManager creates a new unified frontend manager
func NewUnifiedManager(projectConfig *config.ProjectConfig, debug bool) (*UnifiedManager, error) {
	templateManager, err := templates.GetTemplateManager()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize template manager: %w", err)
	}

	scriptManager, err := NewScriptRegistryManager()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize script registry: %w", err)
	}

	return &UnifiedManager{
		projectConfig:   projectConfig,
		templateManager: templateManager,
		scriptManager:   scriptManager,
		debug:           debug,
	}, nil
}

// GetSupportedFrontends returns all supported frontend options for the current template
func (um *UnifiedManager) GetSupportedFrontends() ([]templates.FrontendTemplateInfo, bool, error) {
	// Get template name from project config, fallback to "default"
	templateName := um.projectConfig.Backend.Template
	if templateName == "" {
		templateName = "default"
	}

	template, exists := um.templateManager.GetTemplate(templateName)
	if !exists {
		return nil, false, fmt.Errorf("template %s not found", templateName)
	}

	return template.Frontend.Options, template.Frontend.EnableScriptFrontends, nil
}

// GetScriptFrontends returns all available script-based frontends
func (um *UnifiedManager) GetScriptFrontends() []ScriptFramework {
	return um.scriptManager.GetAllFrameworks()
}

// GetScriptCategories returns script frontends organized by category
func (um *UnifiedManager) GetScriptCategories() []ScriptCategory {
	return um.scriptManager.GetCategories()
}

// IsScriptFramework checks if a framework name is a valid script framework
func (um *UnifiedManager) IsScriptFramework(name string) bool {
	return um.scriptManager.IsValidFramework(name)
}

// GenerateFrontend generates a frontend based on the configuration
func (um *UnifiedManager) GenerateFrontend(frontendPath string) error {
	frontendConfig := um.projectConfig.Frontend

	// Determine generation method
	switch frontendConfig.Template.Type {
	case "template", "built-in", "": // Default to template-based
		return um.generateFromTemplate(frontendPath, frontendConfig)
	case "script":
		return um.generateFromScript(frontendPath, frontendConfig)
	case "custom":
		return um.generateFromCustomCommand(frontendPath, frontendConfig)
	default:
		return fmt.Errorf("unsupported template type: %s", frontendConfig.Template.Type)
	}
}

// generateFromTemplate generates frontend from built-in template
func (um *UnifiedManager) generateFromTemplate(frontendPath string, frontendConfig config.FrontendConfig) error {
	// Get template name from project config, fallback to "default"
	templateName := um.projectConfig.Backend.Template
	if templateName == "" {
		templateName = "default"
	}

	frontendName := frontendConfig.Template.Source

	if frontendName == "" {
		frontendName = frontendConfig.Framework
	}

	// Verify frontend exists in template
	supportedFrontends, _, err := um.GetSupportedFrontends()
	if err != nil {
		return err
	}

	var frontendOption *templates.FrontendTemplateInfo
	for _, option := range supportedFrontends {
		if option.Name == frontendName || option.Framework == frontendName {
			frontendOption = &option
			break
		}
	}

	if frontendOption == nil {
		return fmt.Errorf("frontend %s not found in template", frontendName)
	}

	if um.debug {
		fmt.Printf("🎨 Generating frontend '%s' from template '%s'...\n", frontendName, templateName)
	}

	// Create template data
	data := templates.TemplateData{
		ProjectName:        um.projectConfig.Name,
		ModuleName:         um.projectConfig.Name,
		GoVersion:          "1.24.2",
		Port:               fmt.Sprintf("%d", um.projectConfig.Port),
		Router:             um.projectConfig.Backend.Router,
		SPARouting:         true,
		ProjectDescription: "Generated with GoFlux",
		CustomVars:         make(map[string]interface{}),
	}

	// Generate frontend using template manager
	return um.templateManager.GenerateFrontend(templateName, frontendName, frontendPath, data)
}

// generateFromScript generates frontend using a script command
func (um *UnifiedManager) generateFromScript(frontendPath string, frontendConfig config.FrontendConfig) error {
	// Verify script frontends are supported
	_, scriptSupported, err := um.GetSupportedFrontends()
	if err != nil {
		return err
	}

	if !scriptSupported {
		return fmt.Errorf("script-based frontends are not supported by the current template")
	}

	scriptCommand := frontendConfig.Template.Source

	// Check if it's a registered script framework
	if framework, exists := um.scriptManager.GetFrameworkByName(frontendConfig.Framework); exists {
		scriptCommand = framework.Script
		if um.debug {
			fmt.Printf("🎨 Using registered script framework '%s': %s\n", framework.DisplayName, framework.Script)
		}
	} else if um.debug {
		fmt.Printf("🎨 Using custom script command: %s\n", scriptCommand)
	}

	// Create script generator using the existing constructor
	scriptGen := NewScriptGenerator(scriptCommand, &frontendConfig, um.debug)
	return scriptGen.Generate(frontendPath, um.projectConfig)
}

// generateFromCustomCommand generates frontend using a custom command
func (um *UnifiedManager) generateFromCustomCommand(frontendPath string, frontendConfig config.FrontendConfig) error {
	if um.debug {
		fmt.Printf("🎨 Generating frontend using custom command: %s\n", frontendConfig.Template.Command)
	}

	// Create custom generator using the existing constructor
	customGen := NewCustomGenerator(frontendConfig.Template.Command, frontendConfig.Template.Dir, &frontendConfig, um.debug)
	return customGen.Generate(frontendPath, um.projectConfig)
}

// UpdateProjectConfigWithScript updates the project configuration for a script framework
func (um *UnifiedManager) UpdateProjectConfigWithScript(frameworkName string) error {
	framework, exists := um.scriptManager.GetFrameworkByName(frameworkName)
	if !exists {
		return fmt.Errorf("script framework %s not found", frameworkName)
	}

	// Update frontend config with script framework details
	um.projectConfig.Frontend.Framework = framework.Framework
	um.projectConfig.Frontend.DevCmd = framework.DevCmd
	um.projectConfig.Frontend.BuildCmd = framework.BuildCmd
	um.projectConfig.Frontend.TypesDir = framework.TypesDir
	um.projectConfig.Frontend.LibDir = framework.LibDir

	// Set template config for script
	um.projectConfig.Frontend.Template.Type = "script"
	um.projectConfig.Frontend.Template.Source = framework.Script

	return nil
}

// UpdateProjectConfig updates the project configuration with frontend template info
func (um *UnifiedManager) UpdateProjectConfig(frontendName string) error {
	// First check if it's a script framework
	if um.IsScriptFramework(frontendName) {
		return um.UpdateProjectConfigWithScript(frontendName)
	}

	// Otherwise, check template frontends
	supportedFrontends, _, err := um.GetSupportedFrontends()
	if err != nil {
		return err
	}

	// Find the frontend option and update config
	for _, option := range supportedFrontends {
		if option.Name == frontendName || option.Framework == frontendName {
			um.projectConfig.Frontend.Framework = option.Framework
			um.projectConfig.Frontend.DevCmd = option.DevCmd
			um.projectConfig.Frontend.BuildCmd = option.BuildCmd
			um.projectConfig.Frontend.TypesDir = option.TypesDir
			um.projectConfig.Frontend.LibDir = option.LibDir
			um.projectConfig.Frontend.StaticGen = option.StaticGen

			// Set template config
			if um.projectConfig.Frontend.Template.Type == "" {
				um.projectConfig.Frontend.Template.Type = "template"
			}
			if um.projectConfig.Frontend.Template.Source == "" {
				um.projectConfig.Frontend.Template.Source = frontendName
			}

			return nil
		}
	}

	return fmt.Errorf("frontend %s not found", frontendName)
}

// GetAvailableTemplates returns all available backend templates
func (um *UnifiedManager) GetAvailableTemplates() map[string]string {
	return um.templateManager.GetTemplateNames()
}

// GetAvailableScriptFrameworks returns all available script frameworks
func (um *UnifiedManager) GetAvailableScriptFrameworks() map[string]string {
	return um.scriptManager.GetFrameworkNames()
}

// ShouldRunInstallCommand determines if we need to run an install command
func (um *UnifiedManager) ShouldRunInstallCommand() bool {
	// Template-based frontends don't need install commands (they include all files)
	// Script and custom frontends handle their own installation
	return false
}
