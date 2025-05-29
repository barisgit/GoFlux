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
	InstallCmd string          `yaml:"install_cmd"`
	DevCmd     string          `yaml:"dev_cmd"`
	BuildCmd   string          `yaml:"build_cmd"`
	TypesDir   string          `yaml:"types_dir"`
	LibDir     string          `yaml:"lib_dir"`
	StaticGen  StaticGenConfig `yaml:"static_gen"`
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
