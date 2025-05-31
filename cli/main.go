package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/barisgit/goflux/cli/internal/commands"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	// Handle working directory for development mode
	if workDir := os.Getenv("flux_WORK_DIR"); workDir != "" {
		if err := os.Chdir(workDir); err != nil {
			fmt.Fprintf(os.Stderr, "Error: Failed to change to work directory %s: %v\n", workDir, err)
			os.Exit(1)
		}
	}

	rootCmd := &cobra.Command{
		Use:     "flux",
		Short:   "GoFlux - Full-stack Go development framework",
		Long:    `GoFlux is a modern full-stack framework combining Go backend with TypeScript frontend.`,
		Version: buildVersion(),
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("🚀 GoFlux CLI " + buildVersion())
			fmt.Println("Run 'flux --help' for available commands")
		},
	}

	// Add commands
	rootCmd.AddCommand(commands.NewCmd())
	rootCmd.AddCommand(commands.DevCmd())
	rootCmd.AddCommand(commands.BuildCmd())
	rootCmd.AddCommand(commands.GenerateTypesCmd())
	rootCmd.AddCommand(commands.ConfigCmd())
	rootCmd.AddCommand(commands.ListCmd())

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func buildVersion() string {
	if version == "dev" {
		return "v" + version
	}
	return version
}
