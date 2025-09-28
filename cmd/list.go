package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/gambitier/tag-manager/pkg/discovery"
	"github.com/olekukonko/tablewriter"
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

	color.Cyan("Discovered %d Go packages:", len(packages))
	color.White("Search paths: %s", strings.Join(searchPaths, ", "))
	color.White("")

	// Create table with modern API
	table := tablewriter.NewWriter(os.Stdout)

	if verbose {
		table.Header("#", "Module", "Package", "Go Version", "GitHub", "Latest Tag")
	} else {
		table.Header("#", "Package", "Latest Tag")
	}

	// Add rows
	for i, pkg := range packages {
		// Handle empty values
		goVersion := pkg.GoVersion
		if goVersion == "" {
			goVersion = "-"
		}

		github := pkg.GitHubRepo
		if github == "" {
			github = "-"
		}

		latestTag := pkg.LatestTag
		if latestTag == "" {
			latestTag = "(no tags)"
		}

		if verbose {
			table.Append(fmt.Sprintf("%d", i+1), pkg.ModulePath, pkg.PackageName, goVersion, github, latestTag)
		} else {
			table.Append(fmt.Sprintf("%d", i+1), pkg.PackageName, latestTag)
		}
	}

	table.Render()
	return nil
}
