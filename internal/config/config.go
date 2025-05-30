package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type ProjectConfig struct {
	Name             string                  `yaml:"name"`
	Port             int                     `yaml:"port"`
	Frontend         FrontendConfig          `yaml:"frontend"`
	Backend          BackendConfig           `yaml:"backend"`
	Build            BuildConfig             `yaml:"build"`
	APIClient        APIClientConfig         `yaml:"api_client"`
	ExternalTemplate *ExternalTemplateConfig `yaml:"external_template,omitempty"`
}

type APIClientConfig struct {
	Generator   string            `yaml:"generator"`    // "basic", "axios", "trpc-like"
	ReactQuery  ReactQueryConfig  `yaml:"react_query"`  // React Query specific options
	Options     map[string]string `yaml:"options"`      // Additional generator options
	OutputFile  string            `yaml:"output_file"`  // Custom output file name
	TypesImport string            `yaml:"types_import"` // Custom types import path
}

type ReactQueryConfig struct {
	Enabled       bool   `yaml:"enabled"`        // Enable React Query integration
	Version       string `yaml:"version"`        // React Query version (v4, v5)
	QueryOptions  bool   `yaml:"query_options"`  // Generate queryOptions functions
	QueryKeys     bool   `yaml:"query_keys"`     // Generate query key factories
	DevTools      bool   `yaml:"devtools"`       // Include devtools setup
	ErrorBoundary bool   `yaml:"error_boundary"` // Generate error boundary components
}

type FrontendConfig struct {
	Framework  string          `yaml:"framework"`
	InstallCmd string          `yaml:"install_cmd,omitempty"` // Legacy support
	DevCmd     string          `yaml:"dev_cmd"`
	BuildCmd   string          `yaml:"build_cmd"`
	TypesDir   string          `yaml:"types_dir"`
	LibDir     string          `yaml:"lib_dir"`
	StaticGen  StaticGenConfig `yaml:"static_gen"`
	Template   TemplateConfig  `yaml:"template,omitempty"` // New template configuration
}

// TemplateConfig defines how the frontend should be generated
type TemplateConfig struct {
	Type   string `yaml:"type"`   // "hardcoded", "script", "custom", "remote"
	Source string `yaml:"source"` // template name, script command, or URL/path

	// For remote templates
	URL     string            `yaml:"url,omitempty"`     // GitHub URL or local path
	Version string            `yaml:"version,omitempty"` // Git tag, branch, or "latest"
	Cache   bool              `yaml:"cache,omitempty"`   // Whether to cache the template
	Vars    map[string]string `yaml:"vars,omitempty"`    // Template variables

	// For custom commands
	Command string `yaml:"command,omitempty"` // Custom installation command
	Dir     string `yaml:"dir,omitempty"`     // Working directory for command
}

type StaticGenConfig struct {
	Enabled     bool     `yaml:"enabled"`
	BuildSSRCmd string   `yaml:"build_ssr_cmd"`
	GenerateCmd string   `yaml:"generate_cmd"`
	Routes      []string `yaml:"routes"`
	SPARouting  bool     `yaml:"spa_routing"`
}

type BackendConfig struct {
	Router   string `yaml:"router"`
	Template string `yaml:"template,omitempty"` // Backend template name
}

type BuildConfig struct {
	OutputDir   string `yaml:"output_dir"`
	BinaryName  string `yaml:"binary_name"`
	EmbedStatic bool   `yaml:"embed_static"`
	StaticDir   string `yaml:"static_dir"`
	BuildTags   string `yaml:"build_tags"`
	LDFlags     string `yaml:"ldflags"`
	CGOEnabled  bool   `yaml:"cgo_enabled"`
}

// ExternalTemplateConfig holds information about external templates
type ExternalTemplateConfig struct {
	Source      string `yaml:"source"`                // GitHub URL or local path
	Name        string `yaml:"name"`                  // Template name from template.yaml
	Description string `yaml:"description"`           // Template description
	CachedPath  string `yaml:"cached_path,omitempty"` // Local cache path
}

// GetDefaultAPIClientConfig returns the default API client configuration
func GetDefaultAPIClientConfig() APIClientConfig {
	return APIClientConfig{
		Generator:   "basic",
		OutputFile:  "api-client.ts",
		TypesImport: "../types/generated",
		ReactQuery: ReactQueryConfig{
			Enabled:       false,
			Version:       "v5",
			QueryOptions:  true,
			QueryKeys:     true,
			DevTools:      true,
			ErrorBoundary: false,
		},
		Options: make(map[string]string),
	}
}

func ReadConfig(path string) (*ProjectConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config ProjectConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	// Set defaults for API client config if not specified
	if config.APIClient.Generator == "" {
		defaultConfig := GetDefaultAPIClientConfig()
		config.APIClient = defaultConfig
	}

	return &config, nil
}
