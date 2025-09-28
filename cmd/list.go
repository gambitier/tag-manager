package cmd

import (
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/gambitier/tag-manager/pkg/discovery"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List discovered Go packages",
	Long:  `List all discovered Go packages across multiple repositories with their configuration status.`,
	RunE:  runList,
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

	for i, pkg := range packages {
		color.White("%d. %s", i+1, pkg.ModulePath)
		fmt.Printf("   Path: %s\n", pkg.Path)
		fmt.Printf("   Package: %s\n", pkg.PackageName)
		if pkg.GoVersion != "" {
			fmt.Printf("   Go Version: %s\n", pkg.GoVersion)
		}
		if pkg.GitHubRepo != "" {
			fmt.Printf("   GitHub: %s\n", pkg.GitHubRepo)
		}
		if pkg.LatestTag != "" {
			fmt.Printf("   Latest Tag: %s\n", pkg.LatestTag)
		} else {
			fmt.Printf("   Latest Tag: (no tags found)\n")
		}
		color.White("")
	}

	return nil
}
