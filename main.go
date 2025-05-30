package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/barisgit/goflux/cmd"
)

var version = "0.1.0"

func main() {
	rootCmd := &cobra.Command{
		Use:     "flux",
		Short:   "GoFlux - Full-stack Go development framework",
		Long:    `GoFlux is a modern full-stack framework combining Go backend with TypeScript frontend.`,
		Version: version,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("🚀 GoFlux CLI v" + version)
			fmt.Println("Run 'flux --help' for available commands")
		},
	}

	// Add commands
	rootCmd.AddCommand(cmd.NewCmd())
	rootCmd.AddCommand(cmd.DevCmd())
	rootCmd.AddCommand(cmd.BuildCmd())
	rootCmd.AddCommand(cmd.GenerateTypesCmd())
	rootCmd.AddCommand(cmd.ListCmd())

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
