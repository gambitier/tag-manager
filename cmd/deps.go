package cmd

import (
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/gambitier/tag-manager/pkg/deps"
	"github.com/gambitier/tag-manager/pkg/discovery"
	"github.com/spf13/cobra"
)

var (
	depsFormat string
	depsFilter string
)

var depsCmd = &cobra.Command{
	Use:   "deps",
	Short: "Show dependency graph for discovered packages",
	Long: `Show the dependency graph for all discovered Go packages.
This command analyzes go.mod files to understand how packages depend on each other
and displays the dependency relationships in a clear, hierarchical format.`,
	RunE: runDeps,
}

func init() {
	depsCmd.Flags().StringVarP(&depsFormat, "format", "f", "tree", "Output format: tree, matrix, levels")
	depsCmd.Flags().StringVarP(&depsFilter, "filter", "", "", "Filter packages by name (partial match)")
}

func runDeps(cmd *cobra.Command, args []string) error {
	// Discover packages
	searchPaths := discovery.GetDefaultSearchPaths()
	graph, err := deps.AnalyzeDependencies(searchPaths)
	if err != nil {
		return fmt.Errorf("failed to analyze dependencies: %w", err)
	}

	// Validate graph for circular dependencies
	if err := graph.ValidateGraph(); err != nil {
		color.Red("❌ %v", err)
		return err
	}

	color.Green("✅ No circular dependencies found")

	// Filter packages if requested
	if depsFilter != "" {
		filteredPackages := make(map[string]*deps.Package)
		for path, pkg := range graph.Packages {
			if strings.Contains(strings.ToLower(pkg.PackageName), strings.ToLower(depsFilter)) ||
				strings.Contains(strings.ToLower(pkg.ModulePath), strings.ToLower(depsFilter)) {
				filteredPackages[path] = pkg
			}
		}
		graph.Packages = filteredPackages
	}

	// Display based on format
	switch depsFormat {
	case "tree":
		displayDependencyTree(graph)
	case "matrix":
		displayDependencyMatrix(graph)
	case "levels":
		displayDependencyLevels(graph)
	default:
		return fmt.Errorf("unknown format: %s. Available formats: tree, matrix, levels", depsFormat)
	}

	return nil
}

func displayDependencyTree(graph *deps.DependencyGraph) {
	color.Cyan("\n📊 Dependency Tree")
	color.White("================\n")

	// Show foundation packages first
	foundation := graph.GetFoundationPackages()
	if len(foundation) > 0 {
		color.Yellow("🏗️  Foundation Packages (no internal dependencies):")
		for _, pkg := range foundation {
			color.White("  • %s", pkg.PackageName)
			if len(pkg.Dependents) > 0 {
				color.White("    └─ Used by: %d package(s)", len(pkg.Dependents))
			}
		}
		color.White("")
	}

	// Show dependency levels
	maxLevel := 0
	for level := range graph.Levels {
		if level > maxLevel {
			maxLevel = level
		}
	}

	for level := 1; level <= maxLevel; level++ {
		if packages, exists := graph.Levels[level]; exists && len(packages) > 0 {
			color.Cyan("📦 Level %d Packages:", level)
			for _, pkg := range packages {
				color.White("  • %s", pkg.PackageName)
				if len(pkg.Dependencies) > 0 {
					color.White("    └─ Depends on: %s", strings.Join(getPackageNames(pkg.Dependencies, graph), ", "))
				}
				if len(pkg.Dependents) > 0 {
					color.White("    └─ Used by: %d package(s)", len(pkg.Dependents))
				}
			}
			color.White("")
		}
	}

	// Show top-level packages
	topLevel := graph.GetTopLevelPackages()
	if len(topLevel) > 0 {
		color.Green("🎯 Top-Level Packages (no dependents):")
		for _, pkg := range topLevel {
			color.White("  • %s", pkg.PackageName)
		}
	}
}

func displayDependencyMatrix(graph *deps.DependencyGraph) {
	color.Cyan("\n📊 Dependency Matrix")
	color.White("==================\n")

	// Create a sorted list of all packages
	var allPackages []*deps.Package
	for _, pkg := range graph.Packages {
		allPackages = append(allPackages, pkg)
	}

	// Sort by package name
	for i := 0; i < len(allPackages)-1; i++ {
		for j := i + 1; j < len(allPackages); j++ {
			if allPackages[i].PackageName > allPackages[j].PackageName {
				allPackages[i], allPackages[j] = allPackages[j], allPackages[i]
			}
		}
	}

	// Print header
	fmt.Printf("%-20s", "Package")
	for _, pkg := range allPackages {
		fmt.Printf(" %-8s", pkg.PackageName[:min(8, len(pkg.PackageName))])
	}
	fmt.Println()

	// Print matrix
	for _, pkg := range allPackages {
		fmt.Printf("%-20s", pkg.PackageName)
		for _, otherPkg := range allPackages {
			if contains(pkg.Dependencies, otherPkg.ModulePath) {
				fmt.Printf(" %-8s", "✓")
			} else {
				fmt.Printf(" %-8s", "-")
			}
		}
		fmt.Println()
	}
}

func displayDependencyLevels(graph *deps.DependencyGraph) {
	color.Cyan("\n📊 Dependency Levels")
	color.White("===================\n")

	// Show packages grouped by level
	maxLevel := 0
	for level := range graph.Levels {
		if level > maxLevel {
			maxLevel = level
		}
	}

	for level := 0; level <= maxLevel; level++ {
		if packages, exists := graph.Levels[level]; exists && len(packages) > 0 {
			if level == 0 {
				color.Yellow("🏗️  Level %d (Foundation):", level)
			} else {
				color.Cyan("📦 Level %d:", level)
			}

			for _, pkg := range packages {
				color.White("  • %s", pkg.PackageName)
				if len(pkg.Dependencies) > 0 {
					color.White("    └─ Dependencies: %s", strings.Join(getPackageNames(pkg.Dependencies, graph), ", "))
				}
				if len(pkg.Dependents) > 0 {
					color.White("    └─ Dependents: %d", len(pkg.Dependents))
				}
			}
			color.White("")
		}
	}

	// Show update impact summary
	color.Green("🔄 Update Impact Summary:")
	color.White("When you update a package, you need to update all packages that depend on it.")
	color.White("")

	// Show most impactful packages
	var impactSummary []struct {
		pkg    *deps.Package
		impact int
	}

	for _, pkg := range graph.Packages {
		impact := len(graph.GetUpdateImpact(pkg.ModulePath))
		impactSummary = append(impactSummary, struct {
			pkg    *deps.Package
			impact int
		}{pkg, impact})
	}

	// Sort by impact
	for i := 0; i < len(impactSummary)-1; i++ {
		for j := i + 1; j < len(impactSummary); j++ {
			if impactSummary[i].impact < impactSummary[j].impact {
				impactSummary[i], impactSummary[j] = impactSummary[j], impactSummary[i]
			}
		}
	}

	color.Yellow("Most impactful packages (update affects most dependents):")
	for i, item := range impactSummary {
		if i >= 5 { // Show top 5
			break
		}
		if item.impact > 0 {
			color.White("  • %s → affects %d package(s)", item.pkg.PackageName, item.impact)
		}
	}
}

func getPackageNames(modulePaths []string, graph *deps.DependencyGraph) []string {
	var names []string
	for _, path := range modulePaths {
		if pkg, exists := graph.GetPackageByPath(path); exists {
			names = append(names, pkg.PackageName)
		}
	}
	return names
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
