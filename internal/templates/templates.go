package templates

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/barisgit/goflux/templates"
)

// TemplateData contains the data passed to templates
type TemplateData struct {
	ProjectName string
	ModuleName  string
	GoVersion   string
	BackendPort string
	Router      string
	SPARouting  bool
}

// GenerateProject creates a new project from templates
func GenerateProject(projectPath, projectName, router string) error {
	data := TemplateData{
		ProjectName: projectName,
		ModuleName:  projectName,
		GoVersion:   "1.24.2",
		BackendPort: "3000",
		Router:      router,
		SPARouting:  true, // Default to SPA routing enabled
	}

	// Use embedded templates
	return generateFromEmbedded(projectPath, data)
}

// generateFromEmbedded generates project from embedded templates
func generateFromEmbedded(projectPath string, data TemplateData) error {
	// Walk through all template files in the embedded filesystem
	return fs.WalkDir(templates.TemplatesFS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
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

		// Determine output path (remove .tmpl extension)
		outputPath := strings.TrimSuffix(path, ".tmpl")
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

// ListTemplates returns all available template files
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
