package templates

import (
	"fmt"
	"io/fs"
	"strings"

	"github.com/barisgit/goflux/templates"
)

// TemplateData contains the data passed to templates
type TemplateData struct {
	ProjectName        string
	ModuleName         string
	GoVersion          string
	Port               string
	Router             string
	SPARouting         bool
	ProjectDescription string
	CustomVars         map[string]interface{}
}

// GenerateProject creates a new project from templates using the new unified system
func GenerateProject(templateName, projectPath, projectName, router string) error {
	manager := NewManager()
	if err := manager.LoadTemplates(); err != nil {
		return fmt.Errorf("failed to load templates: %w", err)
	}

	customVars := make(map[string]interface{})
	return manager.GenerateProject(templateName, projectPath, projectName, router, customVars)
}

// ListTemplates returns all available template files (legacy function for compatibility)
func ListTemplates() ([]string, error) {
	var templateFiles []string

	err := fs.WalkDir(templates.TemplatesFS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !d.IsDir() && strings.HasSuffix(d.Name(), ".tmpl") {
			templateFiles = append(templateFiles, path)
		}

		return nil
	})

	return templateFiles, err
}

// GetTemplateManager creates and loads a template manager
func GetTemplateManager() (*Manager, error) {
	manager := NewManager()
	if err := manager.LoadTemplates(); err != nil {
		return nil, fmt.Errorf("failed to load templates: %w", err)
	}
	return manager, nil
}
