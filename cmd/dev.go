package cmd

import (
	"fmt"
	"goflux/internal/config"
	"goflux/internal/dev"
	"os"

	"github.com/spf13/cobra"
)

func DevCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dev",
		Short: "Start development server",
		Long:  "Start the Go backend and frontend development servers with hot reload and config hot reload",
		RunE:  runDev,
	}

	cmd.Flags().Bool("debug", false, "Enable debug logging for type generation")

	return cmd
}

func runDev(cmd *cobra.Command, args []string) error {
	// Get debug flag
	debug, _ := cmd.Flags().GetBool("debug")

	// Check if we're running in development mode with a different working directory
	workDir := os.Getenv("flux_WORK_DIR")
	if workDir != "" {
		// Change to the original working directory
		if err := os.Chdir(workDir); err != nil {
			return fmt.Errorf("failed to change to work directory %s: %w", workDir, err)
		}
	}

	// Check if we're in a GoFlux project
	configPath := "flux.yaml"
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return fmt.Errorf("flux.yaml not found. Are you in a GoFlux project directory?\nRun 'flux new <project-name>' to create a new project")
	}

	// Read config
	cfg, err := config.ReadConfig(configPath)
	if err != nil {
		return fmt.Errorf("failed to read flux.yaml: %w", err)
	}

	// Create and start orchestrator
	orchestrator := dev.NewDevOrchestrator(cfg, debug)
	return orchestrator.Start()
}
