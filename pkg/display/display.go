package display

import (
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/gambitier/tag-manager/pkg/discovery"
	"github.com/olekukonko/tablewriter"
)

// DisplayMode represents the display mode for package lists
type DisplayMode int

const (
	// Compact shows only package name and latest tag
	Compact DisplayMode = iota
	// Verbose shows all details
	Verbose
)

// ShowPackageList displays a list of packages in a table format
func ShowPackageList(packages []discovery.Package, mode DisplayMode) {
	if len(packages) == 0 {
		color.Red("No Go packages found.")
		return
	}

	// Create table
	table := tablewriter.NewWriter(os.Stdout)

	if mode == Verbose {
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

		if mode == Verbose {
			table.Append(fmt.Sprintf("%d", i+1), pkg.ModulePath, pkg.PackageName, goVersion, github, latestTag)
		} else {
			table.Append(fmt.Sprintf("%d", i+1), pkg.PackageName, latestTag)
		}
	}

	table.Render()
}

// ShowPackageListWithHeader displays a list of packages with a header
func ShowPackageListWithHeader(packages []discovery.Package, mode DisplayMode, searchPaths []string) {
	color.Cyan("Discovered %d Go packages:", len(packages))
	color.White("Search paths: %s", strings.Join(searchPaths, ", "))
	color.White("")

	ShowPackageList(packages, mode)
}
