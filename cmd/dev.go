package cmd

import (
	"fmt"
	"os"

	"github.com/barisgit/goflux/internal/config"
	"github.com/barisgit/goflux/internal/dev"

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

	// Load configuration using enhanced system
	cm := config.NewConfigManager(config.ConfigLoadOptions{
		Path:              "flux.yaml",
		AllowMissing:      false,
		ValidateStructure: true,
		ApplyDefaults:     true,
		WarnOnDeprecated:  !debug, // Only show warnings if not in debug mode
		Quiet:             false,
	})

	cfg, err := cm.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Create and start orchestrator
	orchestrator := dev.NewDevOrchestrator(cfg, debug)
	return orchestrator.Start()
}
