package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidationError(t *testing.T) {
	err := ValidationError{
		Field:   "test_field",
		Value:   "test_value",
		Message: "test message",
	}

	expectedError := "config validation error in field 'test_field': test message (value: test_value)"
	if err.Error() != expectedError {
		t.Errorf("Expected error message '%s', got '%s'", expectedError, err.Error())
	}
}

func TestValidationErrors(t *testing.T) {
	// Test empty validation errors
	emptyErrs := ValidationErrors{}
	if emptyErrs.Error() != "no validation errors" {
		t.Errorf("Expected 'no validation errors', got '%s'", emptyErrs.Error())
	}
	if emptyErrs.HasErrors() {
		t.Error("Expected HasErrors() to be false for empty errors")
	}

	// Test validation errors with content
	errs := ValidationErrors{
		ValidationError{Field: "field1", Value: "value1", Message: "message1"},
		ValidationError{Field: "field2", Value: "value2", Message: "message2"},
	}

	if !errs.HasErrors() {
		t.Error("Expected HasErrors() to be true for non-empty errors")
	}

	errorMsg := errs.Error()
	if !strings.Contains(errorMsg, "field1") || !strings.Contains(errorMsg, "field2") {
		t.Errorf("Expected error message to contain both fields, got '%s'", errorMsg)
	}
}

func TestDefaultLoadOptions(t *testing.T) {
	options := DefaultLoadOptions()

	if options.Path != "flux.yaml" {
		t.Errorf("Expected default path 'flux.yaml', got '%s'", options.Path)
	}
	if options.AllowMissing {
		t.Error("Expected AllowMissing to be false by default")
	}
	if !options.ValidateStructure {
		t.Error("Expected ValidateStructure to be true by default")
	}
	if !options.ApplyDefaults {
		t.Error("Expected ApplyDefaults to be true by default")
	}
	if !options.WarnOnDeprecated {
		t.Error("Expected WarnOnDeprecated to be true by default")
	}
	if options.Quiet {
		t.Error("Expected Quiet to be false by default")
	}
}

func TestNewConfigManager(t *testing.T) {
	options := ConfigLoadOptions{
		Path:         "test.yaml",
		AllowMissing: true,
	}

	cm := NewConfigManager(options)
	if cm == nil {
		t.Fatal("Expected non-nil config manager")
	}

	if cm.options.Path != "test.yaml" {
		t.Errorf("Expected path 'test.yaml', got '%s'", cm.options.Path)
	}
	if !cm.options.AllowMissing {
		t.Error("Expected AllowMissing to be true")
	}
}

func TestConfigManagerLoadConfigMissingFile(t *testing.T) {
	// Test with AllowMissing = false
	cm := NewConfigManager(ConfigLoadOptions{
		Path:         "nonexistent.yaml",
		AllowMissing: false,
	})

	_, err := cm.LoadConfig()
	if err == nil {
		t.Error("Expected error for missing file when AllowMissing is false")
	}

	// Test with AllowMissing = true
	cm2 := NewConfigManager(ConfigLoadOptions{
		Path:         "nonexistent.yaml",
		AllowMissing: true,
		Quiet:        true, // Avoid output during test
	})

	config, err := cm2.LoadConfig()
	if err != nil {
		t.Errorf("Expected no error when AllowMissing is true, got %v", err)
	}
	if config == nil {
		t.Error("Expected default config when file is missing and AllowMissing is true")
	}
}

func TestConfigManagerLoadConfigValidYAML(t *testing.T) {
	// Create a temporary valid config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test-config.yaml")

	validConfig := `
name: test-project
port: 8080
frontend:
  framework: react
  dev_cmd: npm run dev
  build_cmd: npm run build
  types_dir: src/types
  lib_dir: src/lib
  static_gen:
    enabled: false
backend:
  router: chi
build:
  output_dir: dist
  binary_name: server
  embed_static: true
  static_dir: frontend/dist
api_client:
  generator: basic-ts
  react_query:
    enabled: false
    version: v5
`

	err := os.WriteFile(configPath, []byte(validConfig), 0644)
	if err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	cm := NewConfigManager(ConfigLoadOptions{
		Path:              configPath,
		ValidateStructure: true,
		ApplyDefaults:     true,
		Quiet:             true,
	})

	config, err := cm.LoadConfig()
	if err != nil {
		t.Fatalf("Expected no error for valid config, got %v", err)
	}

	if config.Name != "test-project" {
		t.Errorf("Expected name 'test-project', got '%s'", config.Name)
	}
	if config.Port != 8080 {
		t.Errorf("Expected port 8080, got %d", config.Port)
	}
	if config.Frontend.Framework != "react" {
		t.Errorf("Expected frontend framework 'react', got '%s'", config.Frontend.Framework)
	}
	if config.Backend.Router != "chi" {
		t.Errorf("Expected backend router 'chi', got '%s'", config.Backend.Router)
	}
}

func TestConfigManagerLoadConfigInvalidYAML(t *testing.T) {
	// Create a temporary invalid config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "invalid-config.yaml")

	invalidConfig := `
name: test-project
port: invalid-port
frontend:
  framework: [invalid syntax
`

	err := os.WriteFile(configPath, []byte(invalidConfig), 0644)
	if err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	cm := NewConfigManager(ConfigLoadOptions{
		Path:  configPath,
		Quiet: true,
	})

	_, err = cm.LoadConfig()
	if err == nil {
		t.Error("Expected error for invalid YAML")
	}
}

func TestConfigValidation(t *testing.T) {
	cm := NewConfigManager(ConfigLoadOptions{})

	tests := []struct {
		name           string
		config         ProjectConfig
		expectErrors   bool
		expectedFields []string
	}{
		{
			name: "valid config",
			config: ProjectConfig{
				Name: "test-project",
				Port: 8080,
				Frontend: FrontendConfig{
					Framework: "react",
					DevCmd:    "npm run dev",
					BuildCmd:  "npm run build",
				},
				Backend: BackendConfig{
					Router: "chi",
				},
				Build: BuildConfig{
					OutputDir:  "dist",
					BinaryName: "server",
				},
				APIClient: APIClientConfig{
					Generator: "basic-ts",
				},
			},
			expectErrors: false,
		},
		{
			name: "empty name",
			config: ProjectConfig{
				Name: "",
				Port: 8080,
				Frontend: FrontendConfig{
					Framework: "react",
					DevCmd:    "npm run dev",
					BuildCmd:  "npm run build",
				},
				Backend: BackendConfig{
					Router: "chi",
				},
				Build: BuildConfig{
					OutputDir:  "dist",
					BinaryName: "server",
				},
				APIClient: APIClientConfig{
					Generator: "basic-ts",
				},
			},
			expectErrors:   true,
			expectedFields: []string{"name"},
		},
		{
			name: "invalid port",
			config: ProjectConfig{
				Name: "test-project",
				Port: -1,
				Frontend: FrontendConfig{
					Framework: "react",
					DevCmd:    "npm run dev",
					BuildCmd:  "npm run build",
				},
				Backend: BackendConfig{
					Router: "chi",
				},
				Build: BuildConfig{
					OutputDir:  "dist",
					BinaryName: "server",
				},
				APIClient: APIClientConfig{
					Generator: "basic-ts",
				},
			},
			expectErrors:   true,
			expectedFields: []string{"port"},
		},
		{
			name: "invalid router",
			config: ProjectConfig{
				Name: "test-project",
				Port: 8080,
				Frontend: FrontendConfig{
					Framework: "react",
					DevCmd:    "npm run dev",
					BuildCmd:  "npm run build",
				},
				Backend: BackendConfig{
					Router: "invalid-router",
				},
				Build: BuildConfig{
					OutputDir:  "dist",
					BinaryName: "server",
				},
				APIClient: APIClientConfig{
					Generator: "basic-ts",
				},
			},
			expectErrors:   true,
			expectedFields: []string{"backend.router"},
		},
		{
			name: "empty frontend fields",
			config: ProjectConfig{
				Name: "test-project",
				Port: 8080,
				Frontend: FrontendConfig{
					Framework: "",
					DevCmd:    "",
					BuildCmd:  "",
				},
				Backend: BackendConfig{
					Router: "chi",
				},
				Build: BuildConfig{
					OutputDir:  "dist",
					BinaryName: "server",
				},
				APIClient: APIClientConfig{
					Generator: "basic-ts",
				},
			},
			expectErrors:   true,
			expectedFields: []string{"frontend.framework", "frontend.dev_cmd", "frontend.build_cmd"},
		},
		{
			name: "invalid api client generator",
			config: ProjectConfig{
				Name: "test-project",
				Port: 8080,
				Frontend: FrontendConfig{
					Framework: "react",
					DevCmd:    "npm run dev",
					BuildCmd:  "npm run build",
				},
				Backend: BackendConfig{
					Router: "chi",
				},
				Build: BuildConfig{
					OutputDir:  "dist",
					BinaryName: "server",
				},
				APIClient: APIClientConfig{
					Generator: "invalid-generator",
				},
			},
			expectErrors:   true,
			expectedFields: []string{"api_client.generator"},
		},
		{
			name: "empty build fields",
			config: ProjectConfig{
				Name: "test-project",
				Port: 8080,
				Frontend: FrontendConfig{
					Framework: "react",
					DevCmd:    "npm run dev",
					BuildCmd:  "npm run build",
				},
				Backend: BackendConfig{
					Router: "chi",
				},
				Build: BuildConfig{
					OutputDir:  "",
					BinaryName: "",
				},
				APIClient: APIClientConfig{
					Generator: "basic-ts",
				},
			},
			expectErrors:   true,
			expectedFields: []string{"build.output_dir", "build.binary_name"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := cm.validateConfig(&tt.config)

			if tt.expectErrors && !errs.HasErrors() {
				t.Error("Expected validation errors but got none")
			}

			if !tt.expectErrors && errs.HasErrors() {
				t.Errorf("Expected no validation errors but got: %v", errs)
			}

			if tt.expectErrors {
				for _, expectedField := range tt.expectedFields {
					found := false
					for _, err := range errs {
						if err.Field == expectedField {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("Expected validation error for field '%s' but didn't find it", expectedField)
					}
				}
			}
		})
	}
}

func TestApplyDefaults(t *testing.T) {
	cm := NewConfigManager(ConfigLoadOptions{})
	config := &ProjectConfig{}

	cm.applyDefaults(config)

	// Check that defaults were applied
	if config.Port == 0 {
		t.Error("Expected port to have a default value")
	}

	// Check that API client defaults were applied
	if config.APIClient.Generator == "" {
		t.Error("Expected API client generator to have a default value")
	}

	// Check build defaults
	if config.Build.OutputDir == "" {
		t.Error("Expected build output dir to have a default value")
	}
	if config.Build.BinaryName == "" {
		t.Error("Expected build binary name to have a default value")
	}

	// Check frontend directory defaults
	if config.Frontend.TypesDir == "" {
		t.Error("Expected frontend types dir to have a default value")
	}
	if config.Frontend.LibDir == "" {
		t.Error("Expected frontend lib dir to have a default value")
	}

	// Note: Framework and Router are NOT set by applyDefaults - they need to be provided
	// This is intentional as they require explicit choice by the user
}

func TestCreateDefaultConfig(t *testing.T) {
	cm := NewConfigManager(ConfigLoadOptions{})
	config := cm.createDefaultConfig()

	if config == nil {
		t.Fatal("Expected non-nil default config")
	}

	// Basic checks for default config
	if config.Name == "" {
		t.Error("Expected default config to have a name")
	}
	if config.Port <= 0 {
		t.Error("Expected default config to have a valid port")
	}
	if config.Frontend.Framework == "" {
		t.Error("Expected default config to have a frontend framework")
	}
	if config.Backend.Router == "" {
		t.Error("Expected default config to have a backend router")
	}
}

func TestValidateConfigFile(t *testing.T) {
	// Test with non-existent file
	err := ValidateConfigFile("nonexistent.yaml")
	if err == nil {
		t.Error("Expected error for non-existent config file")
	}

	// Create a valid config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "valid-config.yaml")

	validConfig := `
name: test-project
port: 8080
frontend:
  framework: react
  dev_cmd: npm run dev
  build_cmd: npm run build
  types_dir: src/types
  lib_dir: src/lib
backend:
  router: chi
build:
  output_dir: dist
  binary_name: server
api_client:
  generator: basic-ts
`

	err = os.WriteFile(configPath, []byte(validConfig), 0644)
	if err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	// Test validation of valid file
	err = ValidateConfigFile(configPath)
	if err != nil {
		t.Errorf("Expected no error for valid config file, got %v", err)
	}

	// Create an invalid config file
	invalidConfigPath := filepath.Join(tmpDir, "invalid-config.yaml")
	invalidConfig := `
name: ""
port: -1
backend:
  router: invalid-router
`

	err = os.WriteFile(invalidConfigPath, []byte(invalidConfig), 0644)
	if err != nil {
		t.Fatalf("Failed to create invalid test config file: %v", err)
	}

	// Test validation of invalid file
	err = ValidateConfigFile(invalidConfigPath)
	if err == nil {
		t.Error("Expected error for invalid config file")
	}
}

func TestGetConfigInfo(t *testing.T) {
	// Test with non-existent file
	_, err := GetConfigInfo("nonexistent.yaml")
	if err == nil {
		t.Error("Expected error for non-existent config file")
	}

	// Create a test config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test-config.yaml")

	testConfig := `
name: test-project
port: 8080
frontend:
  framework: react
  dev_cmd: npm run dev
  build_cmd: npm run build
backend:
  router: chi
api_client:
  generator: basic-ts
  react_query:
    enabled: true
build:
  output_dir: dist
  binary_name: server
  static_dir: frontend/dist
  embed_static: true
`

	err = os.WriteFile(configPath, []byte(testConfig), 0644)
	if err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	// Test getting config info
	info, err := GetConfigInfo(configPath)
	if err != nil {
		t.Fatalf("Expected no error getting config info, got %v", err)
	}

	if info.ProjectName != "test-project" {
		t.Errorf("Expected project name 'test-project', got '%s'", info.ProjectName)
	}
	if info.Port != 8080 {
		t.Errorf("Expected port 8080, got %d", info.Port)
	}
	if info.BackendRouter != "chi" {
		t.Errorf("Expected backend router 'chi', got '%s'", info.BackendRouter)
	}
	if info.FrontendFramework != "react" {
		t.Errorf("Expected frontend framework 'react', got '%s'", info.FrontendFramework)
	}
	if info.APIClientGenerator != "basic-ts" {
		t.Errorf("Expected API client generator 'basic-ts', got '%s'", info.APIClientGenerator)
	}
	if !info.ReactQueryEnabled {
		t.Error("Expected React Query to be enabled")
	}
	if info.BuildOutputDir != "dist" {
		t.Errorf("Expected build output dir 'dist', got '%s'", info.BuildOutputDir)
	}
	if info.StaticDir != "frontend/dist" {
		t.Errorf("Expected static dir 'frontend/dist', got '%s'", info.StaticDir)
	}
	if !info.EmbedStatic {
		t.Error("Expected embed static to be true")
	}
}

func TestConfigInfoString(t *testing.T) {
	info := &ConfigInfo{
		Path:               "/path/to/config.yaml",
		ProjectName:        "test-project",
		Port:               8080,
		BackendRouter:      "chi",
		FrontendFramework:  "react",
		APIClientGenerator: "basic-ts",
		ReactQueryEnabled:  true,
		BuildOutputDir:     "dist",
		StaticDir:          "frontend/dist",
		EmbedStatic:        true,
	}

	infoStr := info.String()

	// Check that the string contains expected information
	expectedFields := []string{
		"test-project",
		"8080",
		"chi",
		"react",
		"basic-ts",
		"enabled",
		"dist",
		"frontend/dist",
	}

	for _, field := range expectedFields {
		if !strings.Contains(infoStr, field) {
			t.Errorf("Expected info string to contain '%s', got '%s'", field, infoStr)
		}
	}
}

func TestConvenienceFunctions(t *testing.T) {
	// Create a test config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test-config.yaml")

	testConfig := `
name: test-project
port: 8080
frontend:
  framework: react
  dev_cmd: npm run dev
  build_cmd: npm run build
  types_dir: src/types
  lib_dir: src/lib
backend:
  router: chi
build:
  output_dir: dist
  binary_name: server
api_client:
  generator: basic-ts
`

	err := os.WriteFile(configPath, []byte(testConfig), 0644)
	if err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	// Test ReadConfig
	config, err := ReadConfig(configPath)
	if err != nil {
		t.Fatalf("ReadConfig failed: %v", err)
	}
	if config.Name != "test-project" {
		t.Errorf("Expected name 'test-project', got '%s'", config.Name)
	}

	// Change working directory for other tests
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)

	err = os.Chdir(tmpDir)
	if err != nil {
		t.Fatalf("Failed to change working directory: %v", err)
	}

	// Copy config to default name for other tests
	err = os.Rename(configPath, "flux.yaml")
	if err != nil {
		t.Fatalf("Failed to rename config file: %v", err)
	}

	// Test LoadConfig
	config, err = LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}
	if config.Name != "test-project" {
		t.Errorf("Expected name 'test-project', got '%s'", config.Name)
	}

	// Test LoadConfigQuiet
	config, err = LoadConfigQuiet()
	if err != nil {
		t.Fatalf("LoadConfigQuiet failed: %v", err)
	}
	if config.Name != "test-project" {
		t.Errorf("Expected name 'test-project', got '%s'", config.Name)
	}

	// Test LoadConfigWithDefaults
	config, err = LoadConfigWithDefaults()
	if err != nil {
		t.Fatalf("LoadConfigWithDefaults failed: %v", err)
	}
	if config.Name != "test-project" {
		t.Errorf("Expected name 'test-project', got '%s'", config.Name)
	}
}

func TestContainsFunction(t *testing.T) {
	slice := []string{"apple", "banana", "cherry"}

	if !contains(slice, "banana") {
		t.Error("Expected contains to return true for 'banana'")
	}

	if contains(slice, "grape") {
		t.Error("Expected contains to return false for 'grape'")
	}

	if contains([]string{}, "anything") {
		t.Error("Expected contains to return false for empty slice")
	}
}

func TestGetDefaultAPIClientConfig(t *testing.T) {
	config := GetDefaultAPIClientConfig()

	if config.Generator == "" {
		t.Error("Expected default generator to be set")
	}

	// Check that react query has some defaults
	if config.ReactQuery.Version == "" {
		t.Error("Expected default React Query version to be set")
	}
}
