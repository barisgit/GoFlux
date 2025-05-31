package frontend

import (
	_ "embed"
	"fmt"

	"gopkg.in/yaml.v3"
)

//go:embed script_registry.yaml
var scriptRegistryData []byte

// ScriptRegistryManager manages script-based frontend templates
type ScriptRegistryManager struct {
	registry *ScriptRegistry
}

// NewScriptRegistryManager creates a new script registry manager
func NewScriptRegistryManager() (*ScriptRegistryManager, error) {
	var registry ScriptRegistry
	if err := yaml.Unmarshal(scriptRegistryData, &registry); err != nil {
		return nil, fmt.Errorf("failed to parse script registry: %w", err)
	}

	return &ScriptRegistryManager{
		registry: &registry,
	}, nil
}

// GetAllFrameworks returns all available script frameworks
func (m *ScriptRegistryManager) GetAllFrameworks() []ScriptFramework {
	var frameworks []ScriptFramework
	for _, category := range m.registry.Categories {
		frameworks = append(frameworks, category.Frameworks...)
	}
	return frameworks
}

// GetFrameworkByName returns a framework by its name
func (m *ScriptRegistryManager) GetFrameworkByName(name string) (*ScriptFramework, bool) {
	for _, category := range m.registry.Categories {
		for _, framework := range category.Frameworks {
			if framework.Name == name || framework.Framework == name {
				return &framework, true
			}
		}
	}
	return nil, false
}

// GetCategories returns all categories
func (m *ScriptRegistryManager) GetCategories() []ScriptCategory {
	return m.registry.Categories
}

// GetFrameworksByCategory returns frameworks in a specific category
func (m *ScriptRegistryManager) GetFrameworksByCategory(categoryName string) ([]ScriptFramework, bool) {
	for _, category := range m.registry.Categories {
		if category.Name == categoryName {
			return category.Frameworks, true
		}
	}
	return nil, false
}

// GetFrameworkNames returns all framework names with display names
func (m *ScriptRegistryManager) GetFrameworkNames() map[string]string {
	result := make(map[string]string)
	for _, category := range m.registry.Categories {
		for _, framework := range category.Frameworks {
			result[framework.Name] = framework.DisplayName + " - " + framework.Description
		}
	}
	return result
}

// IsValidFramework checks if a framework name is valid
func (m *ScriptRegistryManager) IsValidFramework(name string) bool {
	_, exists := m.GetFrameworkByName(name)
	return exists
}
