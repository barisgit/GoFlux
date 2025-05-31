package commands

import (
	"fmt"

	"github.com/barisgit/goflux/cli/internal/templates"
	"github.com/spf13/cobra"
)

func ListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List available templates and frameworks",
		Long:  "Display all available backend templates and frontend frameworks",
		RunE:  runList,
	}

	return cmd
}

func runList(cmd *cobra.Command, args []string) error {
	// Get template manager
	templateManager, err := templates.GetTemplateManager()
	if err != nil {
		return fmt.Errorf("failed to load templates: %w", err)
	}

	// List backend templates
	templates := templateManager.GetTemplateNames()
	fmt.Println("📦 Available Backend Templates:")
	for name, description := range templates {
		fmt.Printf("  • %s - %s\n", name, description)
	}

	return nil
}
