package templates

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/barisgit/goflux/templates"
	"gopkg.in/yaml.v3"
)

// Manager handles unified template operations
type Manager struct {
	templates map[string]*TemplateManifest
}

// NewManager creates a new template manager
func NewManager() *Manager {
	return &Manager{
		templates: make(map[string]*TemplateManifest),
	}
}

// LoadTemplates loads all available templates from embedded filesystem
func (m *Manager) LoadTemplates() error {
	// Walk through embedded templates directory
	return fs.WalkDir(templates.TemplatesFS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Look for template.yaml files
		if !d.IsDir() && d.Name() == "template.yaml" {
			templateDir := filepath.Dir(path)
			templateName := filepath.Base(templateDir)

			// Skip if at root level
			if templateDir == "." {
				return nil
			}

			manifest, err := m.loadTemplateManifest(path)
			if err != nil {
				return fmt.Errorf("failed to load template %s: %w", templateName, err)
			}

			m.templates[templateName] = manifest
		}

		return nil
	})
}

// loadTemplateManifest loads a template manifest from embedded filesystem
func (m *Manager) loadTemplateManifest(path string) (*TemplateManifest, error) {
	data, err := templates.TemplatesFS.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read template.yaml: %w", err)
	}

	var manifest TemplateManifest
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("failed to parse template.yaml: %w", err)
	}

	return &manifest, nil
}

// GetTemplate returns a template by name
func (m *Manager) GetTemplate(name string) (*TemplateManifest, bool) {
	template, exists := m.templates[name]
	return template, exists
}

// ListTemplates returns all available template names
func (m *Manager) ListTemplates() []string {
	var names []string
	for name := range m.templates {
		names = append(names, name)
	}
	return names
}

// GetTemplateNames returns all available template names with descriptions
func (m *Manager) GetTemplateNames() map[string]string {
	result := make(map[string]string)
	for name, manifest := range m.templates {
		result[name] = manifest.Description
	}
	return result
}

// GenerateProject creates a new project from a template
func (m *Manager) GenerateProject(templateName, projectPath, projectName, router string, customVars map[string]interface{}) error {
	manifest, exists := m.GetTemplate(templateName)
	if !exists {
		return fmt.Errorf("template %s not found", templateName)
	}

	// Validate router is supported
	if !m.isRouterSupported(manifest, router) {
		return fmt.Errorf("router %s is not supported by template %s. Supported routers: %v",
			router, templateName, manifest.Backend.SupportedRouters)
	}

	data := TemplateData{
		ProjectName:        projectName,
		ModuleName:         projectName,
		GoVersion:          "1.24.2",
		Port:               "3000",
		Router:             router,
		SPARouting:         true,
		ProjectDescription: manifest.Description,
		CustomVars:         customVars,
	}

	// Generate backend from template
	return m.generateFromTemplate(templateName, projectPath, data)
}

// isRouterSupported checks if a router is supported by the template
func (m *Manager) isRouterSupported(manifest *TemplateManifest, router string) bool {
	for _, supportedRouter := range manifest.Backend.SupportedRouters {
		if supportedRouter == router {
			return true
		}
	}
	return false
}

// generateFromTemplate generates project from a specific template
func (m *Manager) generateFromTemplate(templateName, projectPath string, data TemplateData) error {
	templatePrefix := templateName + "/"

	// Walk through all template files for this specific template
	return fs.WalkDir(templates.TemplatesFS, templateName, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		// Skip template.yaml and frontend directories
		if d.Name() == "template.yaml" || strings.Contains(path, "/frontends/") {
			return nil
		}

		// Only process .tmpl files
		if !strings.HasSuffix(path, ".tmpl") {
			return nil
		}

		// Read template file from embedded filesystem
		content, err := templates.TemplatesFS.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read template %s: %w", path, err)
		}

		// Parse and execute template
		tmpl, err := template.New(filepath.Base(path)).Parse(string(content))
		if err != nil {
			return fmt.Errorf("failed to parse template %s: %w", path, err)
		}

		// Determine output path (remove template prefix and .tmpl extension)
		outputPath := strings.TrimPrefix(path, templatePrefix)
		outputPath = strings.TrimSuffix(outputPath, ".tmpl")
		fullOutputPath := filepath.Join(projectPath, outputPath)

		// Create directory if it doesn't exist
		outputDir := filepath.Dir(fullOutputPath)
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", outputDir, err)
		}

		// Create output file
		outputFile, err := os.Create(fullOutputPath)
		if err != nil {
			return fmt.Errorf("failed to create file %s: %w", fullOutputPath, err)
		}
		defer outputFile.Close()

		// Execute template and write to file
		if err := tmpl.Execute(outputFile, data); err != nil {
			return fmt.Errorf("failed to execute template %s: %w", path, err)
		}

		return nil
	})
}

// GenerateFrontend creates a frontend from template's frontend options
func (m *Manager) GenerateFrontend(templateName, frontendName, frontendPath string, data TemplateData) error {
	manifest, exists := m.GetTemplate(templateName)
	if !exists {
		return fmt.Errorf("template %s not found", templateName)
	}

	// Find the frontend option
	var frontendOption *FrontendTemplateInfo
	for _, option := range manifest.Frontend.Options {
		if option.Name == frontendName {
			frontendOption = &option
			break
		}
	}

	if frontendOption == nil {
		return fmt.Errorf("frontend %s not found in template %s", frontendName, templateName)
	}

	// Generate frontend from embedded template
	frontendTemplatePrefix := templateName + "/frontends/" + frontendName + "/"

	return fs.WalkDir(templates.TemplatesFS, templateName+"/frontends/"+frontendName, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		// Read file from embedded filesystem
		content, err := templates.TemplatesFS.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read frontend file %s: %w", path, err)
		}

		// Determine output path
		outputPath := strings.TrimPrefix(path, frontendTemplatePrefix)
		fullOutputPath := filepath.Join(frontendPath, outputPath)

		// Create directory if it doesn't exist
		outputDir := filepath.Dir(fullOutputPath)
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", outputDir, err)
		}

		// Process .tmpl files as templates, copy others as-is
		if strings.HasSuffix(path, ".tmpl") {
			// Remove .tmpl extension from output path
			fullOutputPath = strings.TrimSuffix(fullOutputPath, ".tmpl")

			// Parse and execute template
			tmpl, err := template.New(filepath.Base(path)).Parse(string(content))
			if err != nil {
				return fmt.Errorf("failed to parse frontend template %s: %w", path, err)
			}

			// Create output file
			outputFile, err := os.Create(fullOutputPath)
			if err != nil {
				return fmt.Errorf("failed to create frontend file %s: %w", fullOutputPath, err)
			}
			defer outputFile.Close()

			// Execute template and write to file
			if err := tmpl.Execute(outputFile, data); err != nil {
				return fmt.Errorf("failed to execute frontend template %s: %w", path, err)
			}
		} else {
			// Copy file as-is
			if err := os.WriteFile(fullOutputPath, content, 0644); err != nil {
				return fmt.Errorf("failed to copy frontend file %s: %w", fullOutputPath, err)
			}
		}

		return nil
	})
}

// LoadTemplateFromData loads a template from raw data
func LoadTemplateFromData(data []byte) (*TemplateManifest, error) {
	var manifest TemplateManifest
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("failed to parse template.yaml: %w", err)
	}
	return &manifest, nil
}
