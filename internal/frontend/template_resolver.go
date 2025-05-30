package frontend

import (
	"fmt"
	"strings"

	"github.com/barisgit/goflux/internal/config"
)

// TemplateResolver resolves the appropriate template generator based on configuration
type TemplateResolver struct {
	registry *TemplateRegistry
	debug    bool
}

// NewTemplateResolver creates a new template resolver
func NewTemplateResolver(registry *TemplateRegistry, debug bool) *TemplateResolver {
	return &TemplateResolver{
		registry: registry,
		debug:    debug,
	}
}

// ResolveTemplate determines which generator to use based on the frontend configuration
func (r *TemplateResolver) ResolveTemplate(frontend *config.FrontendConfig) (Generator, error) {
	// 1. Check if explicit template configuration exists
	if frontend.Template.Type != "" {
		return r.resolveByTemplateType(frontend)
	}

	// 2. Check if legacy install_cmd exists
	if frontend.InstallCmd != "" {
		return r.resolveLegacyInstallCmd(frontend)
	}

	// 3. Try to resolve by framework name
	if frontend.Framework != "" {
		return r.resolveByFramework(frontend)
	}

	// 4. Default fallback
	return r.getDefaultGenerator(frontend)
}

// resolveByTemplateType resolves based on explicit template type
func (r *TemplateResolver) resolveByTemplateType(frontend *config.FrontendConfig) (Generator, error) {
	switch frontend.Template.Type {
	case "hardcoded":
		return r.resolveHardcodedTemplate(frontend)
	case "script":
		return r.resolveScriptTemplate(frontend)
	case "custom":
		return r.resolveCustomTemplate(frontend)
	case "remote":
		return r.resolveRemoteTemplate(frontend)
	default:
		return nil, fmt.Errorf("unknown template type: %s", frontend.Template.Type)
	}
}

// resolveHardcodedTemplate resolves a hardcoded template
func (r *TemplateResolver) resolveHardcodedTemplate(frontend *config.FrontendConfig) (Generator, error) {
	templateName := frontend.Template.Source
	if templateName == "" {
		templateName = frontend.Framework
	}

	template, exists := r.registry.GetHardcodedTemplate(templateName)
	if !exists {
		return nil, fmt.Errorf("hardcoded template '%s' not found", templateName)
	}

	return NewHardcodedGenerator(template, r.debug), nil
}

// resolveScriptTemplate resolves a script-based template (pnpx, npm create, etc.)
func (r *TemplateResolver) resolveScriptTemplate(frontend *config.FrontendConfig) (Generator, error) {
	script := frontend.Template.Source
	if script == "" {
		script = frontend.InstallCmd
	}

	if script == "" {
		return nil, fmt.Errorf("no script specified for script template")
	}

	return NewScriptGenerator(script, frontend, r.debug), nil
}

// resolveCustomTemplate resolves a custom command template
func (r *TemplateResolver) resolveCustomTemplate(frontend *config.FrontendConfig) (Generator, error) {
	command := frontend.Template.Command
	if command == "" {
		command = frontend.Template.Source
	}

	if command == "" {
		return nil, fmt.Errorf("no command specified for custom template")
	}

	return NewCustomGenerator(command, frontend.Template.Dir, frontend, r.debug), nil
}

// resolveRemoteTemplate resolves a remote template (GitHub, local path)
func (r *TemplateResolver) resolveRemoteTemplate(frontend *config.FrontendConfig) (Generator, error) {
	url := frontend.Template.URL
	if url == "" {
		url = frontend.Template.Source
	}

	if url == "" {
		return nil, fmt.Errorf("no URL specified for remote template")
	}

	return NewRemoteGenerator(url, frontend.Template, r.debug), nil
}

// resolveLegacyInstallCmd resolves based on legacy install_cmd (backward compatibility)
func (r *TemplateResolver) resolveLegacyInstallCmd(frontend *config.FrontendConfig) (Generator, error) {
	installCmd := frontend.InstallCmd

	// Detect common patterns in install commands
	switch {
	case strings.Contains(installCmd, "pnpx") || strings.Contains(installCmd, "npx") || strings.Contains(installCmd, "npm create") || strings.Contains(installCmd, "pnpm create"):
		// This is a script-based template
		return NewScriptGenerator(installCmd, frontend, r.debug), nil
	default:
		// Treat as custom command
		return NewCustomGenerator(installCmd, "", frontend, r.debug), nil
	}
}

// resolveByFramework resolves based on framework name using hardcoded templates
func (r *TemplateResolver) resolveByFramework(frontend *config.FrontendConfig) (Generator, error) {
	template, exists := r.registry.GetTemplateByFramework(frontend.Framework)
	if !exists {
		return r.getDefaultGenerator(frontend)
	}

	return NewHardcodedGenerator(template, r.debug), nil
}

// getDefaultGenerator returns a default generator (default template)
func (r *TemplateResolver) getDefaultGenerator(frontend *config.FrontendConfig) (Generator, error) {
	template, exists := r.registry.GetHardcodedTemplate("default")
	if !exists {
		// Fallback to minimal if default is not available
		template, exists = r.registry.GetHardcodedTemplate("minimal")
		if !exists {
			return nil, fmt.Errorf("default templates not found")
		}
	}

	return NewHardcodedGenerator(template, r.debug), nil
}

// ValidateConfiguration validates a frontend template configuration
func (r *TemplateResolver) ValidateConfiguration(frontend *config.FrontendConfig) error {
	if frontend.Template.Type == "" && frontend.InstallCmd == "" && frontend.Framework == "" {
		return fmt.Errorf("no frontend configuration specified")
	}

	// Validate based on template type
	switch frontend.Template.Type {
	case "hardcoded":
		if frontend.Template.Source == "" && frontend.Framework == "" {
			return fmt.Errorf("hardcoded template requires source or framework name")
		}
	case "script":
		if frontend.Template.Source == "" && frontend.InstallCmd == "" {
			return fmt.Errorf("script template requires source or install_cmd")
		}
	case "custom":
		if frontend.Template.Command == "" && frontend.Template.Source == "" {
			return fmt.Errorf("custom template requires command or source")
		}
	case "remote":
		if frontend.Template.URL == "" && frontend.Template.Source == "" {
			return fmt.Errorf("remote template requires URL or source")
		}
	}

	return nil
}
