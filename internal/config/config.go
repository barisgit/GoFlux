package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type ProjectConfig struct {
	Name     string         `yaml:"name"`
	Port     int            `yaml:"port"`
	Frontend FrontendConfig `yaml:"frontend"`
	Backend  BackendConfig  `yaml:"backend"`
	Build    BuildConfig    `yaml:"build"`
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
	Router string `yaml:"router"`
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

func ReadConfig(path string) (*ProjectConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config ProjectConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}
