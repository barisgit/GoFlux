package cmd

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/barisgit/goflux/internal/config"
	"github.com/barisgit/goflux/internal/typegen/analyzer"
	"github.com/barisgit/goflux/internal/typegen/generator"

	"github.com/spf13/cobra"
)

func GenerateTypesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "generate-types",
		Short: "Generate TypeScript types from Go code",
		Long:  "Analyze Go code and generate TypeScript types, API client, and route manifest",
		RunE:  runGenerateTypes,
	}

	cmd.Flags().Bool("debug", false, "Enable debug logging")
	cmd.Flags().Bool("quiet", false, "Suppress output (for use in build scripts)")

	return cmd
}

func runGenerateTypes(cmd *cobra.Command, args []string) error {
	debug, _ := cmd.Flags().GetBool("debug")
	quiet, _ := cmd.Flags().GetBool("quiet")

	// Check if we're running in development mode with a different working directory
	workDir := os.Getenv("flux_WORK_DIR")
	if workDir != "" {
		if err := os.Chdir(workDir); err != nil {
			return fmt.Errorf("failed to change to work directory %s: %w", workDir, err)
		}
	}

	// Check if we're in a GoFlux project
	configPath := "flux.yaml"
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		if !quiet {
			fmt.Println("⚠️  flux.yaml not found. Attempting type generation anyway...")
		}
	}

	return generateTypes(debug, quiet)
}

func generateTypes(debug, quiet bool) error {
	if !quiet {
		log("🔧 Generating API types...", "\x1b[36m")
	}

	// Read project configuration
	projectConfig, err := config.ReadConfig("flux.yaml")
	if err != nil {
		if !quiet {
			log("⚠️  Warning: Could not read flux.yaml, using defaults", "\x1b[33m")
			if debug {
				log(fmt.Sprintf("Config read error: %v", err), "\x1b[33m")
			}
		}
		// Use defaults if config file doesn't exist
		defaultAPIConfig := config.GetDefaultAPIClientConfig()
		projectConfig = &config.ProjectConfig{
			APIClient: defaultAPIConfig,
		}
	}

	// Use the new modular type generation system
	analysis, err := analyzer.AnalyzeProject(".", debug)
	if err != nil {
		if !quiet {
			log("❌ Failed to analyze project", "\x1b[31m")
		}
		return fmt.Errorf("project analysis failed: %w", err)
	}

	// Generate all outputs concurrently
	var wg sync.WaitGroup
	errorChan := make(chan error, 3)

	// Generate TypeScript types
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := generator.GenerateTypeScriptTypes(analysis.TypeDefs); err != nil {
			errorChan <- fmt.Errorf("generating TypeScript types: %w", err)
		}
	}()

	// Generate API client
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := generator.GenerateAPIClient(analysis.Routes, analysis.TypeDefs, &projectConfig.APIClient); err != nil {
			errorChan <- fmt.Errorf("generating API client: %w", err)
		}
	}()

	// Generate route manifest
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := generator.GenerateRouteManifest(analysis.Routes); err != nil {
			errorChan <- fmt.Errorf("generating route manifest: %w", err)
		}
	}()

	// Wait for all generators to complete
	go func() {
		wg.Wait()
		close(errorChan)
	}()

	// Check for errors
	for err := range errorChan {
		if !quiet {
			log("❌ Error in type generation", "\x1b[31m")
		}
		return err
	}

	if !quiet {
		log("✅ API types generated successfully", "\x1b[32m")
		log(fmt.Sprintf("Generated %d TypeScript types", len(analysis.TypeDefs)), "\x1b[36m")
		log(fmt.Sprintf("Generated API client with %d routes", len(analysis.Routes)), "\x1b[36m")
		if projectConfig.APIClient.Generator != "basic" {
			log(fmt.Sprintf("Using %s generator", projectConfig.APIClient.Generator), "\x1b[36m")
		}
	}

	return nil
}

func log(message, color string) {
	timestamp := time.Now().Format("15:04:05")
	if color == "" {
		color = "\x1b[0m"
	}
	fmt.Printf("%s[%s] %s\x1b[0m\n", color, timestamp, message)
}
