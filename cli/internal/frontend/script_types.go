package frontend

// ScriptRegistry represents the structure of the script registry YAML
type ScriptRegistry struct {
	Name        string           `yaml:"name"`
	Description string           `yaml:"description"`
	Version     string           `yaml:"version"`
	Categories  []ScriptCategory `yaml:"categories"`
}

// ScriptCategory represents a category of frontend frameworks
type ScriptCategory struct {
	Name       string            `yaml:"name"`
	Frameworks []ScriptFramework `yaml:"frameworks"`
}

// ScriptFramework represents a script-based frontend framework
type ScriptFramework struct {
	Name        string `yaml:"name"`
	DisplayName string `yaml:"display_name"`
	Description string `yaml:"description"`
	Script      string `yaml:"script"`
	Framework   string `yaml:"framework"`
	DevCmd      string `yaml:"dev_cmd"`
	BuildCmd    string `yaml:"build_cmd"`
	TypesDir    string `yaml:"types_dir"`
	LibDir      string `yaml:"lib_dir"`
}
