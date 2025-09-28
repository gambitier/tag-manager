package cmd

import (
	"os"

	"github.com/fatih/color"
	"github.com/gambitier/tag-manager/pkg/config"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Show current configuration",
	Long:  `Display the current tag-manager configuration including file location and package settings.`,
	RunE:  runConfig,
}

func runConfig(cmd *cobra.Command, args []string) error {
	// Get config path
	configPath := config.GetConfigPath()

	// Check if config file exists
	exists := true
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		exists = false
	}

	// Display config file location
	color.Cyan("=== Tag Manager Configuration ===")
	color.White("Config file: %s", configPath)
	if exists {
		color.Green("Status: ✓ Found")
	} else {
		color.Yellow("Status: ✗ Not found (will be created on first use)")
	}
	color.White("")

	// Load and display configuration
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		if exists {
			color.Red("Error loading configuration: %v", err)
			return nil
		}
		// If file doesn't exist, show default config
		color.Yellow("No configuration file found. Here are the defaults:")
		color.White("")

		// Show default configuration
		color.Cyan("Default Tag Format: %s", config.GetDefaultConfig().Defaults.TagFormat)
		color.White("")
		color.White("Configuration will be created automatically when you:")
		color.White("  • Run 'tag-manager update' for the first time")
		color.White("  • Configure a package's tag format")
		return nil
	}

	// Display current configuration
	color.Cyan("Current Configuration:")
	color.White("")

	// Show default tag format
	color.Cyan("Default Tag Format: %s", cfg.Defaults.TagFormat)
	color.White("")

	// Show configured packages
	if len(cfg.Packages) == 0 {
		color.Yellow("No packages configured yet.")
		color.White("Packages will be configured automatically when you run 'tag-manager update'.")
	} else {
		color.Cyan("Configured Packages (%d):", len(cfg.Packages))
		for modulePath, pkgConfig := range cfg.Packages {
			color.White("")
			color.White("  Package: %s", modulePath)
			color.White("    Tag Format: %s", pkgConfig.TagFormat)
			color.White("    Use Default: %t", pkgConfig.UseDefault)
			if pkgConfig.LastUpdated != "" {
				color.White("    Last Updated: %s", pkgConfig.LastUpdated)
			}
		}
	}

	return nil
}
