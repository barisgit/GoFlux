package frontend

import (
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"goflux/internal/config"

	"gopkg.in/yaml.v3"
)

// RemoteTemplateManifest defines the structure of flux-template.yaml
type RemoteTemplateManifest struct {
	Name        string            `yaml:"name"`
	Description string            `yaml:"description"`
	Version     string            `yaml:"version"`
	Author      string            `yaml:"author"`
	Framework   string            `yaml:"framework"`
	Commands    CommandsConfig    `yaml:"commands"`
	Variables   map[string]string `yaml:"variables,omitempty"`
	Files       []FileConfig      `yaml:"files,omitempty"`
}

// CommandsConfig defines commands for the template
type CommandsConfig struct {
	Install string `yaml:"install"`
	Dev     string `yaml:"dev"`
	Build   string `yaml:"build"`
}

// FileConfig defines file processing rules
type FileConfig struct {
	Source      string `yaml:"source"`
	Destination string `yaml:"destination"`
	Template    bool   `yaml:"template,omitempty"`
	Executable  bool   `yaml:"executable,omitempty"`
}

// RemoteTemplateManager handles remote template operations
type RemoteTemplateManager struct {
	cacheDir string
	debug    bool
}

// NewRemoteTemplateManager creates a new remote template manager
func NewRemoteTemplateManager(debug bool) *RemoteTemplateManager {
	// Default cache directory in user's home
	homeDir, _ := os.UserHomeDir()
	cacheDir := filepath.Join(homeDir, ".flux", "templates")

	return &RemoteTemplateManager{
		cacheDir: cacheDir,
		debug:    debug,
	}
}

// Download downloads a remote template and returns the local path
func (m *RemoteTemplateManager) Download(url, version string, useCache bool) (string, error) {
	if m.debug {
		fmt.Printf("📥 Downloading remote template from: %s\n", url)
	}

	// Determine if it's a local path or remote URL
	if m.isLocalPath(url) {
		return m.handleLocalTemplate(url)
	}

	return m.handleRemoteTemplate(url, version, useCache)
}

// isLocalPath checks if the URL is a local file path
func (m *RemoteTemplateManager) isLocalPath(url string) bool {
	return !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://")
}

// handleLocalTemplate handles local template paths
func (m *RemoteTemplateManager) handleLocalTemplate(path string) (string, error) {
	// Check if path exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return "", fmt.Errorf("local template path does not exist: %s", path)
	}

	// Check if it's a directory with flux-template.yaml
	manifestPath := filepath.Join(path, "flux-template.yaml")
	if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
		return "", fmt.Errorf("flux-template.yaml not found in local template: %s", path)
	}

	return path, nil
}

// handleRemoteTemplate handles remote GitHub templates
func (m *RemoteTemplateManager) handleRemoteTemplate(url, version string, useCache bool) (string, error) {
	// Parse GitHub URL and convert to download URL
	downloadURL, err := m.convertToDownloadURL(url, version)
	if err != nil {
		return "", err
	}

	// Generate cache key
	cacheKey := m.generateCacheKey(url, version)
	cachePath := filepath.Join(m.cacheDir, cacheKey)

	// Check cache if enabled
	if useCache && m.isCached(cachePath) {
		if m.debug {
			fmt.Printf("📦 Using cached template: %s\n", cachePath)
		}
		return cachePath, nil
	}

	// Download and extract
	if err := m.downloadAndExtract(downloadURL, cachePath); err != nil {
		return "", err
	}

	return cachePath, nil
}

// convertToDownloadURL converts GitHub repository URL to download URL
func (m *RemoteTemplateManager) convertToDownloadURL(url, version string) (string, error) {
	// Handle different GitHub URL formats
	if strings.Contains(url, "github.com") {
		// Extract owner and repo from URL
		parts := strings.Split(strings.TrimPrefix(url, "https://github.com/"), "/")
		if len(parts) < 2 {
			return "", fmt.Errorf("invalid GitHub URL format: %s", url)
		}

		owner := parts[0]
		repo := parts[1]

		// Use specific version or default to main
		if version == "" || version == "latest" {
			version = "main"
		}

		return fmt.Sprintf("https://github.com/%s/%s/archive/refs/heads/%s.zip", owner, repo, version), nil
	}

	// For other URLs, assume they're direct download links
	return url, nil
}

// generateCacheKey generates a cache key for the template
func (m *RemoteTemplateManager) generateCacheKey(url, version string) string {
	// Simple hash-like key generation
	key := strings.ReplaceAll(url, "/", "_")
	key = strings.ReplaceAll(key, ":", "_")
	key = strings.ReplaceAll(key, ".", "_")

	if version != "" && version != "latest" {
		key += "_" + version
	}

	return key
}

// isCached checks if template is already cached
func (m *RemoteTemplateManager) isCached(path string) bool {
	manifestPath := filepath.Join(path, "flux-template.yaml")
	_, err := os.Stat(manifestPath)
	return err == nil
}

// downloadAndExtract downloads and extracts a template archive
func (m *RemoteTemplateManager) downloadAndExtract(url, extractPath string) error {
	if m.debug {
		fmt.Printf("⬇️  Downloading from: %s\n", url)
	}

	// Create cache directory
	if err := os.MkdirAll(m.cacheDir, 0755); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	// Download the file
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to download template: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download template: HTTP %d", resp.StatusCode)
	}

	// Create temporary file for download
	tmpFile, err := os.CreateTemp("", "flux-template-*.zip")
	if err != nil {
		return fmt.Errorf("failed to create temporary file: %w", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	// Copy downloaded content to temp file
	if _, err := io.Copy(tmpFile, resp.Body); err != nil {
		return fmt.Errorf("failed to save download: %w", err)
	}

	// Extract ZIP file
	if err := m.extractZip(tmpFile.Name(), extractPath); err != nil {
		return fmt.Errorf("failed to extract template: %w", err)
	}

	if m.debug {
		fmt.Printf("✅ Template extracted to: %s\n", extractPath)
	}

	return nil
}

// extractZip extracts a ZIP file to the destination
func (m *RemoteTemplateManager) extractZip(zipPath, destPath string) error {
	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer reader.Close()

	// Create destination directory
	if err := os.MkdirAll(destPath, 0755); err != nil {
		return err
	}

	// Extract files
	for _, file := range reader.File {
		// Skip the root directory (GitHub creates one)
		path := file.Name
		if idx := strings.Index(path, "/"); idx != -1 {
			path = path[idx+1:]
		}

		if path == "" {
			continue
		}

		fullPath := filepath.Join(destPath, path)

		if file.FileInfo().IsDir() {
			os.MkdirAll(fullPath, file.FileInfo().Mode())
			continue
		}

		// Create file directories
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			return err
		}

		// Extract file
		rc, err := file.Open()
		if err != nil {
			return err
		}

		outFile, err := os.OpenFile(fullPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.FileInfo().Mode())
		if err != nil {
			rc.Close()
			return err
		}

		_, err = io.Copy(outFile, rc)
		rc.Close()
		outFile.Close()

		if err != nil {
			return err
		}
	}

	return nil
}

// Generate generates a frontend from a remote template
func (m *RemoteTemplateManager) Generate(templatePath, frontendPath string, projectConfig *config.ProjectConfig, vars map[string]string) error {
	// Load template manifest
	manifest, err := m.loadManifest(templatePath)
	if err != nil {
		return err
	}

	if m.debug {
		fmt.Printf("🎨 Generating frontend using template: %s\n", manifest.Name)
	}

	// Prepare template variables
	templateVars := m.prepareTemplateVars(projectConfig, vars, manifest.Variables)

	// Create frontend directory
	if err := os.MkdirAll(frontendPath, 0755); err != nil {
		return fmt.Errorf("failed to create frontend directory: %w", err)
	}

	// Process files based on manifest
	if len(manifest.Files) > 0 {
		// Use explicit file configuration
		for _, fileConfig := range manifest.Files {
			if err := m.processFile(templatePath, frontendPath, fileConfig, templateVars); err != nil {
				return err
			}
		}
	} else {
		// Copy all files except manifest
		if err := m.copyAllFiles(templatePath, frontendPath, templateVars); err != nil {
			return err
		}
	}

	// Run post-generation commands if specified
	if manifest.Commands.Install != "" {
		if m.debug {
			fmt.Printf("🔧 Running post-generation command: %s\n", manifest.Commands.Install)
		}
		// Commands could be run here, but we'll leave that to the dev orchestrator
	}

	return nil
}

// loadManifest loads the flux-template.yaml manifest
func (m *RemoteTemplateManager) loadManifest(templatePath string) (*RemoteTemplateManifest, error) {
	manifestPath := filepath.Join(templatePath, "flux-template.yaml")

	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read flux-template.yaml: %w", err)
	}

	var manifest RemoteTemplateManifest
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("failed to parse flux-template.yaml: %w", err)
	}

	return &manifest, nil
}

// prepareTemplateVars prepares variables for template processing
func (m *RemoteTemplateManager) prepareTemplateVars(projectConfig *config.ProjectConfig, userVars, manifestVars map[string]string) map[string]interface{} {
	vars := make(map[string]interface{})

	// Add built-in variables
	vars["ProjectName"] = projectConfig.Name
	vars["project_name"] = projectConfig.Name
	vars["PROJECT_NAME"] = strings.ToUpper(projectConfig.Name)

	// Add manifest default variables
	for k, v := range manifestVars {
		vars[k] = v
	}

	// Add user-provided variables (override defaults)
	for k, v := range userVars {
		vars[k] = v
	}

	return vars
}

// processFile processes a single file according to its configuration
func (m *RemoteTemplateManager) processFile(templatePath, frontendPath string, fileConfig FileConfig, vars map[string]interface{}) error {
	sourcePath := filepath.Join(templatePath, fileConfig.Source)
	destPath := filepath.Join(frontendPath, fileConfig.Destination)

	// Create destination directory
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return fmt.Errorf("failed to create directory for %s: %w", destPath, err)
	}

	if fileConfig.Template {
		// Process as template
		return m.processTemplateFile(sourcePath, destPath, vars)
	} else {
		// Copy as-is
		return m.copyFile(sourcePath, destPath, fileConfig.Executable)
	}
}

// processTemplateFile processes a file as a Go template
func (m *RemoteTemplateManager) processTemplateFile(sourcePath, destPath string, vars map[string]interface{}) error {
	// Read template content
	content, err := os.ReadFile(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to read template file %s: %w", sourcePath, err)
	}

	// Parse and execute template
	tmpl, err := template.New(filepath.Base(sourcePath)).Parse(string(content))
	if err != nil {
		return fmt.Errorf("failed to parse template %s: %w", sourcePath, err)
	}

	// Create output file
	outFile, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create output file %s: %w", destPath, err)
	}
	defer outFile.Close()

	// Execute template
	if err := tmpl.Execute(outFile, vars); err != nil {
		return fmt.Errorf("failed to execute template %s: %w", sourcePath, err)
	}

	return nil
}

// copyFile copies a file from source to destination
func (m *RemoteTemplateManager) copyFile(sourcePath, destPath string, executable bool) error {
	sourceFile, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to open source file %s: %w", sourcePath, err)
	}
	defer sourceFile.Close()

	// Determine file mode
	mode := os.FileMode(0644)
	if executable {
		mode = 0755
	}

	destFile, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode)
	if err != nil {
		return fmt.Errorf("failed to create destination file %s: %w", destPath, err)
	}
	defer destFile.Close()

	if _, err := io.Copy(destFile, sourceFile); err != nil {
		return fmt.Errorf("failed to copy file %s to %s: %w", sourcePath, destPath, err)
	}

	return nil
}

// copyAllFiles copies all files from template to frontend (except manifest)
func (m *RemoteTemplateManager) copyAllFiles(templatePath, frontendPath string, vars map[string]interface{}) error {
	return filepath.Walk(templatePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip the template manifest and other flux files
		if strings.HasSuffix(info.Name(), "flux-template.yaml") || strings.HasPrefix(info.Name(), ".flux") {
			return nil
		}

		// Calculate relative path
		relPath, err := filepath.Rel(templatePath, path)
		if err != nil {
			return err
		}

		destPath := filepath.Join(frontendPath, relPath)

		if info.IsDir() {
			return os.MkdirAll(destPath, info.Mode())
		}

		// Determine if file should be processed as template
		isTemplate := strings.HasSuffix(path, ".tmpl") || strings.Contains(path, "{{")

		if isTemplate {
			// Remove .tmpl extension from destination
			if strings.HasSuffix(destPath, ".tmpl") {
				destPath = strings.TrimSuffix(destPath, ".tmpl")
			}
			return m.processTemplateFile(path, destPath, vars)
		} else {
			return m.copyFile(path, destPath, false)
		}
	})
}
