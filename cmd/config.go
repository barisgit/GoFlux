package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/barisgit/goflux/internal/config"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func ConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage project configuration",
		Long:  "Validate, view, and manage your GoFlux project configuration",
	}

	cmd.AddCommand(configValidateCmd())
	cmd.AddCommand(configShowCmd())
	cmd.AddCommand(configInitCmd())
	cmd.AddCommand(configUpgradeCmd())

	return cmd
}

func configValidateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validate [config-file]",
		Short: "Validate configuration file",
		Long:  "Validate the syntax and structure of a GoFlux configuration file",
		Args:  cobra.MaximumNArgs(1),
		RunE:  runConfigValidate,
	}

	cmd.Flags().Bool("strict", false, "Enable strict validation (fail on warnings)")

	return cmd
}

func configShowCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show [config-file]",
		Short: "Show configuration information",
		Long:  "Display detailed information about the current configuration",
		Args:  cobra.MaximumNArgs(1),
		RunE:  runConfigShow,
	}

	cmd.Flags().Bool("verbose", false, "Show detailed configuration breakdown")

	return cmd
}

func configInitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize a new configuration file",
		Long:  "Create a new flux.yaml configuration file with default values",
		RunE:  runConfigInit,
	}

	cmd.Flags().Bool("force", false, "Overwrite existing configuration file")
	cmd.Flags().String("name", "", "Project name (default: current directory name)")
	cmd.Flags().String("backend", "chi", "Backend router")
	cmd.Flags().String("frontend", "react", "Frontend framework")

	return cmd
}

func configUpgradeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "upgrade [config-file]",
		Short: "Upgrade configuration to latest format",
		Long:  "Upgrade an existing configuration file to the latest format and add missing defaults",
		Args:  cobra.MaximumNArgs(1),
		RunE:  runConfigUpgrade,
	}

	cmd.Flags().Bool("backup", true, "Create backup of original file")

	return cmd
}

func runConfigValidate(cmd *cobra.Command, args []string) error {
	// Handle flux_WORK_DIR for development mode
	if err := handleWorkDir(); err != nil {
		return err
	}

	configPath := getConfigPath(args)
	strict, _ := cmd.Flags().GetBool("strict")

	fmt.Printf("🔍 Validating configuration file: %s\n", configPath)

	// Use enhanced validation
	cm := config.NewConfigManager(config.ConfigLoadOptions{
		Path:              configPath,
		AllowMissing:      false,
		ValidateStructure: true,
		ApplyDefaults:     false,
		WarnOnDeprecated:  true,
		Quiet:             false,
	})

	cfg, err := cm.LoadConfigFromPath(configPath)
	if err != nil {
		fmt.Printf("❌ Configuration validation failed:\n%v\n", err)
		return err
	}

	fmt.Printf("✅ Configuration is valid!\n")

	// Show summary
	info, err := config.GetConfigInfo(configPath)
	if err == nil {
		fmt.Printf("\n%s\n", info.String())
	}

	// Check for potential issues in strict mode
	if strict {
		issues := checkConfigIssues(cfg)
		if len(issues) > 0 {
			fmt.Printf("\n⚠️  Potential issues found:\n")
			for i, issue := range issues {
				fmt.Printf("  %d. %s\n", i+1, issue)
			}
			if strict {
				return fmt.Errorf("strict validation failed due to %d issue(s)", len(issues))
			}
		}
	}

	return nil
}

func runConfigShow(cmd *cobra.Command, args []string) error {
	// Handle flux_WORK_DIR for development mode
	if err := handleWorkDir(); err != nil {
		return err
	}

	configPath := getConfigPath(args)
	verbose, _ := cmd.Flags().GetBool("verbose")

	info, err := config.GetConfigInfo(configPath)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	fmt.Printf("%s\n", info.String())

	if verbose {
		fmt.Printf("\n📝 Detailed Configuration:\n")

		// Load and display full config using the same path logic
		cm := config.NewConfigManager(config.DefaultLoadOptions())
		cfg, err := cm.LoadConfigFromPath(configPath)
		if err != nil {
			return fmt.Errorf("failed to load full configuration: %w", err)
		}

		// Pretty print the config as YAML
		data, err := yaml.Marshal(cfg)
		if err != nil {
			return fmt.Errorf("failed to marshal configuration: %w", err)
		}

		fmt.Printf("```yaml\n%s```\n", string(data))
	}

	return nil
}

func runConfigInit(cmd *cobra.Command, args []string) error {
	// Handle flux_WORK_DIR for development mode
	if err := handleWorkDir(); err != nil {
		return err
	}

	force, _ := cmd.Flags().GetBool("force")
	projectName, _ := cmd.Flags().GetString("name")
	backend, _ := cmd.Flags().GetString("backend")
	frontend, _ := cmd.Flags().GetString("frontend")

	configPath := "flux.yaml"

	// Check if file exists
	if !force {
		if _, err := os.Stat(configPath); err == nil {
			return fmt.Errorf("configuration file already exists: %s\nUse --force to overwrite", configPath)
		}
	}

	// Get project name from directory if not provided
	if projectName == "" {
		wd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get working directory: %w", err)
		}
		projectName = filepath.Base(wd)
	}

	// Create default config with provided values
	cfg := &config.ProjectConfig{
		Name: projectName,
		Port: 3000,
		Frontend: config.FrontendConfig{
			Framework: frontend,
			DevCmd:    "cd frontend && pnpm dev --port {{port}} --host",
			BuildCmd:  "cd frontend && pnpm build",
			TypesDir:  "src/types",
			LibDir:    "src/lib",
			StaticGen: config.StaticGenConfig{
				Enabled:    false,
				SPARouting: true,
			},
		},
		Backend: config.BackendConfig{
			Router: backend,
		},
		Build: config.BuildConfig{
			OutputDir:   "dist",
			BinaryName:  "server",
			EmbedStatic: true,
			StaticDir:   "frontend/dist",
			BuildTags:   "embed_static",
			LDFlags:     "-s -w",
			CGOEnabled:  false,
		},
		APIClient: config.GetDefaultAPIClientConfig(),
	}

	// Write configuration file
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal configuration: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write configuration file: %w", err)
	}

	fmt.Printf("✅ Created configuration file: %s\n", configPath)
	fmt.Printf("   Project: %s\n", projectName)
	fmt.Printf("   Backend: %s\n", backend)
	fmt.Printf("   Frontend: %s\n", frontend)

	return nil
}

func runConfigUpgrade(cmd *cobra.Command, args []string) error {
	// Handle flux_WORK_DIR for development mode
	if err := handleWorkDir(); err != nil {
		return err
	}

	configPath := getConfigPath(args)
	backup, _ := cmd.Flags().GetBool("backup")

	fmt.Printf("🔄 Upgrading configuration file: %s\n", configPath)

	// Create backup if requested
	if backup {
		backupPath := configPath + ".backup"
		if err := copyFile(configPath, backupPath); err != nil {
			return fmt.Errorf("failed to create backup: %w", err)
		}
		fmt.Printf("📋 Created backup: %s\n", backupPath)
	}

	// Load current config with defaults applied
	cm := config.NewConfigManager(config.ConfigLoadOptions{
		Path:              configPath,
		AllowMissing:      false,
		ValidateStructure: false, // Don't fail on old format
		ApplyDefaults:     true,
		WarnOnDeprecated:  true,
		Quiet:             false,
	})

	cfg, err := cm.LoadConfigFromPath(configPath)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Write upgraded configuration
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal configuration: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write upgraded configuration: %w", err)
	}

	fmt.Printf("✅ Configuration upgraded successfully\n")

	// Validate upgraded config
	fmt.Printf("🔍 Validating upgraded configuration...\n")
	if err := config.ValidateConfigFile(configPath); err != nil {
		fmt.Printf("⚠️  Warning: Upgraded configuration has validation issues:\n%v\n", err)
	} else {
		fmt.Printf("✅ Upgraded configuration is valid\n")
	}

	return nil
}

// handleWorkDir changes to the flux_WORK_DIR if set (for development mode)
func handleWorkDir() error {
	workDir := os.Getenv("flux_WORK_DIR")
	if workDir != "" {
		if err := os.Chdir(workDir); err != nil {
			return fmt.Errorf("failed to change to work directory %s: %w", workDir, err)
		}
	}
	return nil
}

func getConfigPath(args []string) string {
	if len(args) > 0 {
		return args[0]
	}
	return "flux.yaml"
}

func checkConfigIssues(cfg *config.ProjectConfig) []string {
	var issues []string

	// Check for common issues
	if cfg.Port == 3000 {
		issues = append(issues, "Using default port 3000 - consider changing if running multiple projects")
	}

	if cfg.Frontend.Framework == "react" && cfg.APIClient.Generator == "basic" {
		issues = append(issues, "Consider using 'trpc-like' generator with React Query for better DX")
	}

	if !cfg.Build.EmbedStatic {
		issues = append(issues, "Static file embedding is disabled - deployment may require additional setup")
	}

	if cfg.Frontend.TypesDir == "" || cfg.Frontend.LibDir == "" {
		issues = append(issues, "Frontend directories not configured - type generation may fail")
	}

	return issues
}

func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0644)
}
