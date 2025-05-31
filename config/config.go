package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// ValidationError represents a configuration validation error
type ValidationError struct {
	Field   string
	Value   interface{}
	Message string
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("config validation error in field '%s': %s (value: %v)", e.Field, e.Message, e.Value)
}

// ValidationErrors represents multiple validation errors
type ValidationErrors []ValidationError

func (errs ValidationErrors) Error() string {
	if len(errs) == 0 {
		return "no validation errors"
	}

	var messages []string
	for _, err := range errs {
		messages = append(messages, err.Error())
	}
	return strings.Join(messages, "; ")
}

func (errs ValidationErrors) HasErrors() bool {
	return len(errs) > 0
}

// ConfigLoadOptions provides options for loading configuration
type ConfigLoadOptions struct {
	Path              string
	AllowMissing      bool
	ValidateStructure bool
	ApplyDefaults     bool
	WarnOnDeprecated  bool
	Quiet             bool
}

// DefaultLoadOptions returns sensible defaults for config loading
func DefaultLoadOptions() ConfigLoadOptions {
	return ConfigLoadOptions{
		Path:              "flux.yaml",
		AllowMissing:      false,
		ValidateStructure: true,
		ApplyDefaults:     true,
		WarnOnDeprecated:  true,
		Quiet:             false,
	}
}

// ConfigManager handles configuration loading, validation, and management
type ConfigManager struct {
	options ConfigLoadOptions
}

// NewConfigManager creates a new configuration manager
func NewConfigManager(options ConfigLoadOptions) *ConfigManager {
	return &ConfigManager{
		options: options,
	}
}

// LoadConfig loads and validates the configuration with comprehensive error handling
func (cm *ConfigManager) LoadConfig() (*ProjectConfig, error) {
	return cm.LoadConfigFromPath(cm.options.Path)
}

// LoadConfigFromPath loads configuration from a specific path
func (cm *ConfigManager) LoadConfigFromPath(path string) (*ProjectConfig, error) {
	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if cm.options.AllowMissing {
			if !cm.options.Quiet {
				fmt.Printf("⚠️  Configuration file not found at %s, using defaults\n", path)
			}
			return cm.createDefaultConfig(), nil
		}
		return nil, fmt.Errorf("configuration file not found: %s\n\nAre you in a GoFlux project directory?\nRun 'flux new <project-name>' to create a new project", path)
	}

	// Read file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read configuration file %s: %w", path, err)
	}

	// Parse YAML
	var config ProjectConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse configuration file %s: %w\n\nPlease check your YAML syntax", path, err)
	}

	// Apply defaults if requested
	if cm.options.ApplyDefaults {
		cm.applyDefaults(&config)
	}

	// Validate structure if requested
	if cm.options.ValidateStructure {
		if errs := cm.validateConfig(&config); errs.HasErrors() {
			return nil, fmt.Errorf("configuration validation failed:\n%s", cm.formatValidationErrors(errs))
		}
	}

	// Check for deprecated fields and warn
	if cm.options.WarnOnDeprecated && !cm.options.Quiet {
		cm.checkDeprecatedFields(&config)
	}

	return &config, nil
}

// validateConfig performs comprehensive validation on the configuration
func (cm *ConfigManager) validateConfig(config *ProjectConfig) ValidationErrors {
	var errors ValidationErrors

	// Validate basic fields
	if config.Name == "" {
		errors = append(errors, ValidationError{
			Field:   "name",
			Value:   config.Name,
			Message: "project name cannot be empty",
		})
	}

	if config.Port <= 0 || config.Port > 65535 {
		errors = append(errors, ValidationError{
			Field:   "port",
			Value:   config.Port,
			Message: "port must be between 1 and 65535",
		})
	}

	// Validate backend configuration
	if config.Backend.Router == "" {
		errors = append(errors, ValidationError{
			Field:   "backend.router",
			Value:   config.Backend.Router,
			Message: "backend router cannot be empty",
		})
	} else {
		validRouters := []string{"chi", "gin", "fiber", "echo", "gorilla", "mux", "fasthttp"}
		if !contains(validRouters, config.Backend.Router) {
			errors = append(errors, ValidationError{
				Field:   "backend.router",
				Value:   config.Backend.Router,
				Message: fmt.Sprintf("unsupported router '%s', valid options are: %s", config.Backend.Router, strings.Join(validRouters, ", ")),
			})
		}
	}

	// Validate frontend configuration
	if config.Frontend.Framework == "" {
		errors = append(errors, ValidationError{
			Field:   "frontend.framework",
			Value:   config.Frontend.Framework,
			Message: "frontend framework cannot be empty",
		})
	}

	if config.Frontend.DevCmd == "" {
		errors = append(errors, ValidationError{
			Field:   "frontend.dev_cmd",
			Value:   config.Frontend.DevCmd,
			Message: "frontend dev command cannot be empty",
		})
	}

	if config.Frontend.BuildCmd == "" {
		errors = append(errors, ValidationError{
			Field:   "frontend.build_cmd",
			Value:   config.Frontend.BuildCmd,
			Message: "frontend build command cannot be empty",
		})
	}

	// Validate API client configuration
	validGenerators := []string{"basic", "basic-ts", "axios", "trpc-like"}
	if !contains(validGenerators, config.APIClient.Generator) {
		errors = append(errors, ValidationError{
			Field:   "api_client.generator",
			Value:   config.APIClient.Generator,
			Message: fmt.Sprintf("unsupported generator '%s', valid options are: %s", config.APIClient.Generator, strings.Join(validGenerators, ", ")),
		})
	}

	// Validate React Query version if enabled
	if config.APIClient.ReactQuery.Enabled {
		validVersions := []string{"v4", "v5"}
		if !contains(validVersions, config.APIClient.ReactQuery.Version) {
			errors = append(errors, ValidationError{
				Field:   "api_client.react_query.version",
				Value:   config.APIClient.ReactQuery.Version,
				Message: fmt.Sprintf("unsupported React Query version '%s', valid options are: %s", config.APIClient.ReactQuery.Version, strings.Join(validVersions, ", ")),
			})
		}
	}

	// Validate build configuration
	if config.Build.OutputDir == "" {
		errors = append(errors, ValidationError{
			Field:   "build.output_dir",
			Value:   config.Build.OutputDir,
			Message: "build output directory cannot be empty",
		})
	}

	if config.Build.BinaryName == "" {
		errors = append(errors, ValidationError{
			Field:   "build.binary_name",
			Value:   config.Build.BinaryName,
			Message: "build binary name cannot be empty",
		})
	}

	// Validate static directory exists if embed_static is enabled
	if config.Build.EmbedStatic && config.Build.StaticDir != "" {
		// Only check during build, not during config load
		// This will be checked when actually building
	}

	return errors
}

// applyDefaults sets default values for missing configuration fields
func (cm *ConfigManager) applyDefaults(config *ProjectConfig) {
	// Set default port
	if config.Port == 0 {
		config.Port = 3000
	}

	// Set default API client config
	if config.APIClient.Generator == "" {
		defaultConfig := GetDefaultAPIClientConfig()
		config.APIClient = defaultConfig
	} else {
		// Apply defaults to existing config
		if config.APIClient.OutputFile == "" {
			config.APIClient.OutputFile = "api-client.ts"
		}
		if config.APIClient.TypesImport == "" {
			config.APIClient.TypesImport = "../types/generated"
		}
		if config.APIClient.ReactQuery.Version == "" {
			config.APIClient.ReactQuery.Version = "v5"
		}
		if config.APIClient.Options == nil {
			config.APIClient.Options = make(map[string]string)
		}
	}

	// Set default frontend directories
	if config.Frontend.TypesDir == "" {
		config.Frontend.TypesDir = "src/types"
	}
	if config.Frontend.LibDir == "" {
		config.Frontend.LibDir = "src/lib"
	}

	// Set default build configuration
	if config.Build.OutputDir == "" {
		config.Build.OutputDir = "dist"
	}
	if config.Build.BinaryName == "" {
		config.Build.BinaryName = "server"
	}
	if config.Build.StaticDir == "" {
		config.Build.StaticDir = "frontend/dist"
	}
	if config.Build.BuildTags == "" {
		config.Build.BuildTags = "embed_static"
	}
	if config.Build.LDFlags == "" {
		config.Build.LDFlags = "-s -w"
	}
}

// checkDeprecatedFields warns about deprecated configuration fields
func (cm *ConfigManager) checkDeprecatedFields(config *ProjectConfig) {
	if config.Frontend.InstallCmd != "" {
		fmt.Printf("⚠️  Deprecated field 'frontend.install_cmd' is no longer used\n")
		fmt.Printf("   Consider removing it from your flux.yaml\n")
	}
}

// createDefaultConfig creates a default configuration when no config file exists
func (cm *ConfigManager) createDefaultConfig() *ProjectConfig {
	config := &ProjectConfig{
		Name: "my-flux-app",
		Port: 3000,
		Frontend: FrontendConfig{
			Framework: "react",
			DevCmd:    "cd frontend && pnpm dev --port {{port}} --host",
			BuildCmd:  "cd frontend && pnpm build",
			TypesDir:  "src/types",
			LibDir:    "src/lib",
			StaticGen: StaticGenConfig{
				Enabled:    false,
				SPARouting: true,
			},
		},
		Backend: BackendConfig{
			Router: "chi",
		},
		Build: BuildConfig{
			OutputDir:   "dist",
			BinaryName:  "server",
			EmbedStatic: true,
			StaticDir:   "frontend/dist",
			BuildTags:   "embed_static",
			LDFlags:     "-s -w",
			CGOEnabled:  false,
		},
		APIClient: GetDefaultAPIClientConfig(),
	}

	return config
}

// formatValidationErrors formats validation errors in a user-friendly way
func (cm *ConfigManager) formatValidationErrors(errors ValidationErrors) string {
	var lines []string
	for i, err := range errors {
		lines = append(lines, fmt.Sprintf("  %d. %s", i+1, err.Error()))
	}
	return strings.Join(lines, "\n")
}

// ValidateConfigFile validates a configuration file without loading it fully
func ValidateConfigFile(path string) error {
	cm := NewConfigManager(ConfigLoadOptions{
		Path:              path,
		AllowMissing:      false,
		ValidateStructure: true,
		ApplyDefaults:     false,
		WarnOnDeprecated:  true,
		Quiet:             false,
	})

	_, err := cm.LoadConfigFromPath(path)
	return err
}

// GetConfigInfo returns information about the current configuration
func GetConfigInfo(path string) (*ConfigInfo, error) {
	cm := NewConfigManager(DefaultLoadOptions())
	config, err := cm.LoadConfigFromPath(path)
	if err != nil {
		return nil, err
	}

	absPath, _ := filepath.Abs(path)

	return &ConfigInfo{
		Path:               absPath,
		ProjectName:        config.Name,
		Port:               config.Port,
		BackendRouter:      config.Backend.Router,
		FrontendFramework:  config.Frontend.Framework,
		APIClientGenerator: config.APIClient.Generator,
		ReactQueryEnabled:  config.APIClient.ReactQuery.Enabled,
		BuildOutputDir:     config.Build.OutputDir,
		StaticDir:          config.Build.StaticDir,
		EmbedStatic:        config.Build.EmbedStatic,
	}, nil
}

// ConfigInfo contains summary information about a configuration
type ConfigInfo struct {
	Path               string
	ProjectName        string
	Port               int
	BackendRouter      string
	FrontendFramework  string
	APIClientGenerator string
	ReactQueryEnabled  bool
	BuildOutputDir     string
	StaticDir          string
	EmbedStatic        bool
}

// String returns a formatted string representation of config info
func (info *ConfigInfo) String() string {
	var lines []string
	lines = append(lines, fmt.Sprintf("📋 Configuration Summary"))
	lines = append(lines, fmt.Sprintf("   Path: %s", info.Path))
	lines = append(lines, fmt.Sprintf("   Project: %s", info.ProjectName))
	lines = append(lines, fmt.Sprintf("   Port: %d", info.Port))
	lines = append(lines, fmt.Sprintf("   Backend: %s", info.BackendRouter))
	lines = append(lines, fmt.Sprintf("   Frontend: %s", info.FrontendFramework))
	lines = append(lines, fmt.Sprintf("   API Client: %s", info.APIClientGenerator))
	if info.ReactQueryEnabled {
		lines = append(lines, fmt.Sprintf("   React Query: enabled"))
	}
	lines = append(lines, fmt.Sprintf("   Build Output: %s", info.BuildOutputDir))
	lines = append(lines, fmt.Sprintf("   Static Assets: %s (embed: %t)", info.StaticDir, info.EmbedStatic))

	return strings.Join(lines, "\n")
}

// Convenience functions for backward compatibility and common use cases

// LoadConfig loads configuration using default options
func LoadConfig() (*ProjectConfig, error) {
	cm := NewConfigManager(DefaultLoadOptions())
	return cm.LoadConfig()
}

// LoadConfigQuiet loads configuration without warnings or messages
func LoadConfigQuiet() (*ProjectConfig, error) {
	options := DefaultLoadOptions()
	options.Quiet = true
	options.WarnOnDeprecated = false

	cm := NewConfigManager(options)
	return cm.LoadConfig()
}

// LoadConfigWithDefaults loads configuration, creating defaults if missing
func LoadConfigWithDefaults() (*ProjectConfig, error) {
	options := DefaultLoadOptions()
	options.AllowMissing = true

	cm := NewConfigManager(options)
	return cm.LoadConfig()
}

// ReadConfig is kept for backward compatibility but now uses the enhanced system
func ReadConfig(path string) (*ProjectConfig, error) {
	cm := NewConfigManager(ConfigLoadOptions{
		Path:              path,
		AllowMissing:      false,
		ValidateStructure: false, // Keep old behavior for backward compatibility
		ApplyDefaults:     true,
		WarnOnDeprecated:  false,
		Quiet:             true,
	})
	return cm.LoadConfigFromPath(path)
}

// Helper functions

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// Rest of the original structs and functions remain unchanged...

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
		Generator:   "basic-ts",
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
