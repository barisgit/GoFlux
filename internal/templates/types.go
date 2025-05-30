package templates

import "github.com/barisgit/goflux/internal/config"

// TemplateManifest represents the main template.yaml configuration
type TemplateManifest struct {
	Name        string          `yaml:"name"`
	Description string          `yaml:"description"`
	URL         string          `yaml:"url,omitempty"`
	Version     string          `yaml:"version"`
	Backend     BackendConfig   `yaml:"backend"`
	Frontend    FrontendOptions `yaml:"frontend"`
	Variables   []TemplateVar   `yaml:"template_variables,omitempty"`
}

// BackendConfig defines the backend configuration for a template
type BackendConfig struct {
	SupportedRouters []string `yaml:"supported_routers"`
	Features         []string `yaml:"features"`
}

// FrontendOptions defines the frontend configuration options
type FrontendOptions struct {
	EnableScriptFrontends bool                   `yaml:"enable_script_frontends"`
	Options               []FrontendTemplateInfo `yaml:"options"`
}

// FrontendTemplateInfo contains metadata about a frontend template option
type FrontendTemplateInfo struct {
	Name        string                 `yaml:"name"`
	Description string                 `yaml:"description"`
	Framework   string                 `yaml:"framework"`
	DevCmd      string                 `yaml:"dev_cmd"`
	BuildCmd    string                 `yaml:"build_cmd"`
	TypesDir    string                 `yaml:"types_dir"`
	LibDir      string                 `yaml:"lib_dir"`
	StaticGen   config.StaticGenConfig `yaml:"static_gen"`
}

// TemplateVar represents a template variable definition
type TemplateVar struct {
	Name        string           `yaml:"name"`
	Type        string           `yaml:"type"`
	Description string           `yaml:"description"`
	Default     interface{}      `yaml:"default,omitempty"`
	Options     []TemplateOption `yaml:"options,omitempty"`
}

// TemplateOption represents an option for select-type template variables
type TemplateOption struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
}
