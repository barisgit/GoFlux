package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"goflux/internal/templates"
)

type ProjectConfig struct {
	Name     string         `yaml:"name"`
	Frontend FrontendConfig `yaml:"frontend"`
	Backend  BackendConfig  `yaml:"backend"`
}

type FrontendConfig struct {
	Framework  string `yaml:"framework"`
	InstallCmd string `yaml:"install_cmd"`
	DevCmd     string `yaml:"dev_cmd"`
	BuildCmd   string `yaml:"build_cmd"`
	TypesDir   string `yaml:"types_dir"`
	LibDir     string `yaml:"lib_dir"`
}

type BackendConfig struct {
	Port string `yaml:"port"`
}

func NewCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "new [project-name]",
		Short: "Create a new GoFlux project",
		Long:  "Create a new full-stack project with Go backend and TypeScript frontend",
		Args:  cobra.MaximumNArgs(1),
		RunE:  runNew,
	}
}

func runNew(cmd *cobra.Command, args []string) error {
	var projectName string

	if len(args) > 0 {
		projectName = args[0]
	} else {
		prompt := &survey.Input{
			Message: "Project name:",
			Default: "my-flux-app",
		}
		if err := survey.AskOne(prompt, &projectName); err != nil {
			return err
		}
	}

	// Frontend framework selection
	var frontendFramework string
	frontendPrompt := &survey.Select{
		Message: "Choose frontend framework:",
		Options: []string{
			"TanStack Router (Recommended)",
			"Next.js",
			"Vite + React",
		},
		Default: "TanStack Router (Recommended)",
	}
	if err := survey.AskOne(frontendPrompt, &frontendFramework); err != nil {
		return err
	}

	// Create project directory
	if err := os.MkdirAll(projectName, 0755); err != nil {
		return fmt.Errorf("failed to create project directory: %w", err)
	}

	// Initialize git repository
	if err := exec.Command("git", "init", projectName).Run(); err != nil {
		return fmt.Errorf("failed to initialize git repository: %w", err)
	}

	// Generate flux.yaml config
	config := generateConfig(projectName, frontendFramework)
	if err := writeConfig(filepath.Join(projectName, "flux.yaml"), config); err != nil {
		return err
	}

	// Create project structure
	if err := createProjectStructure(projectName, frontendFramework); err != nil {
		return err
	}

	fmt.Printf("\n🎉 Created %s successfully!\n\n", projectName)
	fmt.Printf("Next steps:\n")
	fmt.Printf("  cd %s\n", projectName)
	fmt.Printf("  flux dev\n\n")

	return nil
}

func generateConfig(name, frontend string) ProjectConfig {
	var frontendConfig FrontendConfig

	switch {
	case strings.Contains(frontend, "TanStack"):
		frontendConfig = FrontendConfig{
			Framework:  "tanstack-router",
			InstallCmd: "pnpx create-tsrouter-app@latest . --template file-router",
			DevCmd:     "cd frontend && pnpm dev --port 3001 --host",
			BuildCmd:   "cd frontend && pnpm build",
			TypesDir:   "src/types",
			LibDir:     "src/lib",
		}
	case strings.Contains(frontend, "Next.js"):
		frontendConfig = FrontendConfig{
			Framework:  "nextjs",
			InstallCmd: "pnpm create next-app@latest . --typescript --tailwind --eslint --app --src-dir --import-alias '@/*' --yes",
			DevCmd:     "cd frontend && pnpm dev --port 3001",
			BuildCmd:   "cd frontend && pnpm build",
			TypesDir:   "src/types",
			LibDir:     "src/lib",
		}
	default: // Vite + React
		frontendConfig = FrontendConfig{
			Framework:  "vite-react",
			InstallCmd: "pnpm create vite@latest . -- --template react-ts",
			DevCmd:     "cd frontend && pnpm dev --port 3001 --host",
			BuildCmd:   "cd frontend && pnpm build",
			TypesDir:   "src/types",
			LibDir:     "src/lib",
		}
	}

	return ProjectConfig{
		Name:     name,
		Frontend: frontendConfig,
		Backend: BackendConfig{
			Port: "3002",
		},
	}
}

func writeConfig(path string, config ProjectConfig) error {
	data, err := yaml.Marshal(config)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

func createProjectStructure(projectName, frontend string) error {
	fmt.Printf("📦 Generating project from templates...\n")

	// Use template system to generate the base project structure
	if err := templates.GenerateProject(projectName, projectName); err != nil {
		return fmt.Errorf("failed to generate project from templates: %w", err)
	}

	fmt.Printf("📦 Installing frontend dependencies...\n")

	// Install frontend based on selection
	return createFrontend(projectName, frontend)
}

func createFrontend(projectName, frontend string) error {
	// Create frontend directory structure
	frontendDir := filepath.Join(projectName, "frontend")
	if err := os.MkdirAll(frontendDir, 0755); err != nil {
		return err
	}

	// Create a minimal README that explains setup is deferred to dev command
	readme := fmt.Sprintf(`# %s Frontend

This frontend will be set up automatically when you run 'flux dev' for the first time.

**Framework**: %s

## Development

To start development:
1. Run 'flux dev' from the project root
2. The frontend will be automatically configured and started
3. Visit http://localhost:3000 to see your app

The frontend setup is deferred to the first dev run to ensure:
- Faster project creation
- Latest package versions
- No interactive prompts during automated setup
`, projectName, frontend)

	return os.WriteFile(filepath.Join(frontendDir, "README.md"), []byte(readme), 0644)
}
