package dev

import (
	"fmt"

	"github.com/barisgit/goflux/openapi"

	"github.com/danielgtaylor/huma/v2"
	"github.com/spf13/cobra"
)

// AddOpenAPICommand adds an OpenAPI generation command to a cobra CLI
// This can be used by user projects to add "openapi" command to their CLI
func AddOpenAPICommand(cli *cobra.Command, apiProvider func() huma.API) {
	openAPICmd := &cobra.Command{
		Use:   "openapi",
		Short: "Generate OpenAPI specification",
		Long:  "Generate OpenAPI specification from your Huma API without starting the server",
		RunE: func(cmd *cobra.Command, args []string) error {
			outputPath, _ := cmd.Flags().GetString("output")
			format, _ := cmd.Flags().GetString("format")

			api := apiProvider()
			if api == nil {
				return fmt.Errorf("failed to get API instance")
			}

			var err error
			var spec []byte

			switch format {
			case "yaml":
				spec, err = openapi.GenerateSpecYAML(api)
			case "json":
				spec, err = openapi.GenerateSpec(api)
			default:
				return fmt.Errorf("unsupported format: %s (use 'json' or 'yaml')", format)
			}

			if err != nil {
				return fmt.Errorf("failed to generate OpenAPI spec: %w", err)
			}

			if outputPath != "" {
				err = openapi.GenerateSpecToFile(api, outputPath)
				if err != nil {
					return err
				}
				fmt.Printf("✅ OpenAPI spec saved to %s\n", outputPath)
			} else {
				fmt.Print(string(spec))
			}

			// Print some stats
			routeCount := openapi.GetRouteCount(api)
			if routeCount > 0 {
				fmt.Printf("🛣️  Found %d API routes\n", routeCount)
			}

			return nil
		},
	}

	openAPICmd.Flags().StringP("output", "o", "", "Output file path (prints to stdout if not specified)")
	openAPICmd.Flags().StringP("format", "f", "json", "Output format (json or yaml)")

	cli.AddCommand(openAPICmd)
}
