package generator

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/barisgit/goflux/templates"
)

//go:embed templates/basic-client.js.tmpl
var basicClientTemplate string

//go:embed templates/basic-ts-client.ts.tmpl
var basicTSClientTemplate string

//go:embed templates/axios-client.ts.tmpl
var axiosClientTemplate string

//go:embed templates/trpc-like-client.ts.tmpl
var trpcLikeClientTemplate string

//go:embed templates/basic-method.js.tmpl
var basicMethodTemplate string

//go:embed templates/basic-ts-method.ts.tmpl
var basicTSMethodTemplate string

//go:embed templates/axios-method.ts.tmpl
var axiosMethodTemplate string

//go:embed templates/trpc-get-method.ts.tmpl
var trpcGetMethodTemplate string

//go:embed templates/trpc-mutation-method.ts.tmpl
var trpcMutationMethodTemplate string

// generateFromTemplate creates a file from an embedded template
func generateFromTemplate(templateStr string, data ClientTemplateData, outputPath string) error {
	// Create custom function map for templates
	funcMap := template.FuncMap{
		"join": strings.Join,
	}

	tmpl, err := template.New("client").Funcs(funcMap).Parse(templateStr)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	// Create output directory
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return fmt.Errorf("failed to create directory for %s: %w", outputPath, err)
	}

	// Create output file
	outputFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file %s: %w", outputPath, err)
	}
	defer outputFile.Close()

	// Execute template
	if err := tmpl.Execute(outputFile, data); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	return nil
}

// executeMethodTemplate executes a method template and returns the generated code
func executeMethodTemplate(templateStr string, data MethodTemplateData) (string, error) {
	// Create custom function map for templates
	funcMap := template.FuncMap{
		"eq":  func(a, b string) bool { return a == b },
		"and": func(a, b bool) bool { return a && b },
	}

	tmpl, err := template.New("method").Funcs(funcMap).Parse(templateStr)
	if err != nil {
		return "", fmt.Errorf("failed to parse method template: %w", err)
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute method template: %w", err)
	}

	return buf.String(), nil
}

// generateFileFromEmbeddedTemplate generates a file from an embedded template
func generateFileFromEmbeddedTemplate(templatePath, outputPath string, data interface{}) error {
	// Read template content from embedded filesystem
	content, err := templates.TemplatesFS.ReadFile(templatePath)
	if err != nil {
		return fmt.Errorf("failed to read embedded template %s: %w", templatePath, err)
	}

	// Parse template
	tmpl, err := template.New(filepath.Base(templatePath)).Parse(string(content))
	if err != nil {
		return fmt.Errorf("failed to parse template %s: %w", templatePath, err)
	}

	// Create output directory
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return fmt.Errorf("failed to create directory for %s: %w", outputPath, err)
	}

	// Create output file
	outputFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file %s: %w", outputPath, err)
	}
	defer outputFile.Close()

	// Execute template
	if err := tmpl.Execute(outputFile, data); err != nil {
		return fmt.Errorf("failed to execute template %s: %w", templatePath, err)
	}

	return nil
}
