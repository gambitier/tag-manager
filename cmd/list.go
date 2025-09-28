package cmd

import (
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/gambitier/tag-manager/pkg/discovery"
	"github.com/gambitier/tag-manager/pkg/display"
	"github.com/spf13/cobra"
)

var (
	verbose bool
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List discovered Go packages",
	Long:  `List all discovered Go packages across multiple repositories with their configuration status.`,
	RunE:  runList,
}

func init() {
	listCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Show detailed information (module path, go version, github repo)")
}

func runList(cmd *cobra.Command, args []string) error {
	// Discover packages
	searchPaths := discovery.GetDefaultSearchPaths()
	packages, err := discovery.DiscoverPackages(searchPaths)
	if err != nil {
		return fmt.Errorf("failed to discover packages: %w", err)
	}

	if len(packages) == 0 {
		color.Red("No Go packages found in the search paths.")
		color.Yellow("Searched in: %s", strings.Join(searchPaths, ", "))
		return nil
	}

	// Determine display mode
	mode := display.Compact
	if verbose {
		mode = display.Verbose
	}

	// Show package list with header
	display.ShowPackageListWithHeader(packages, mode, searchPaths)
	return nil
}
