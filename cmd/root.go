package cmd

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "tag-manager",
	Short: "A generic tool for managing tags across Go repositories",
	Long: `Tag Manager is a CLI tool that helps manage version tags for Go packages
across multiple repositories. It can discover packages automatically, support
custom tag naming conventions, and provides interactive confirmation before making changes.

The tool will:
- Scan for go.mod files to discover packages
- Support custom tag naming conventions via configuration
- Provide interactive setup for new packages
- Update major, minor, or patch versions with confirmation`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// Add subcommands
	rootCmd.AddCommand(updateCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(configCmd)
}
