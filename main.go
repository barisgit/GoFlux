package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/barisgit/goflux/cmd"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
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
	rootCmd.AddCommand(cmd.NewCmd())
	rootCmd.AddCommand(cmd.DevCmd())
	rootCmd.AddCommand(cmd.BuildCmd())
	rootCmd.AddCommand(cmd.GenerateTypesCmd())
	rootCmd.AddCommand(cmd.ConfigCmd())
	rootCmd.AddCommand(cmd.ListCmd())

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
