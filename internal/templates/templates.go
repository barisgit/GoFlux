package templates

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

// TemplateData contains the data passed to templates
type TemplateData struct {
	ProjectName string
	ModuleName  string
	GoVersion   string
	BackendPort string
	Router      string
}

// GenerateProject creates a new project from templates
func GenerateProject(projectPath, projectName, router string) error {
	data := TemplateData{
		ProjectName: projectName,
		ModuleName:  projectName,
		GoVersion:   "1.24.2",
		BackendPort: "3002",
		Router:      router,
	}

	// Get the path to templates directory relative to the CLI
	templatesDir := filepath.Join("templates")

	// Walk through all template files in the templates directory
	return filepath.Walk(templatesDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		// Only process .tmpl files
		if !strings.HasSuffix(path, ".tmpl") {
			return nil
		}

		// Read template file
		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read template %s: %w", path, err)
		}

		// Parse and execute template
		tmpl, err := template.New(filepath.Base(path)).Parse(string(content))
		if err != nil {
			return fmt.Errorf("failed to parse template %s: %w", path, err)
		}

		// Determine output path (remove templates/ prefix and .tmpl extension)
		relPath, err := filepath.Rel(templatesDir, path)
		if err != nil {
			return err
		}
		outputPath := strings.TrimSuffix(relPath, ".tmpl")
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
	var templates []string
	templatesDir := filepath.Join("templates")

	err := filepath.Walk(templatesDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && strings.HasSuffix(info.Name(), ".tmpl") {
			templates = append(templates, path)
		}

		return nil
	})

	return templates, err
}
