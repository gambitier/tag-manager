package cmd

import (
	"fmt"
	"os/exec"

	"github.com/fatih/color"
	"github.com/gambitier/tag-manager/pkg/config"
	"github.com/gambitier/tag-manager/pkg/deps"
	"github.com/gambitier/tag-manager/pkg/discovery"
	"github.com/gambitier/tag-manager/pkg/interactive"
	"github.com/gambitier/tag-manager/pkg/tagutils"
	"github.com/spf13/cobra"
)

var (
	depsUpdateAuto    bool
	depsUpdateVersion string
)

var depsUpdateCmd = &cobra.Command{
	Use:   "deps-update",
	Short: "Update package and all its dependents",
	Long: `Update a package and automatically update all packages that depend on it.
This command analyzes the dependency graph and updates packages in the correct order
to maintain dependency consistency across the repository.`,
	RunE: runDepsUpdate,
}

func init() {
	depsUpdateCmd.Flags().BoolVarP(&depsUpdateAuto, "auto", "a", false, "Automatically update all dependents without confirmation")
	depsUpdateCmd.Flags().StringVarP(&depsUpdateVersion, "version", "v", "", "Version type: major, minor, or patch (if not specified, will be asked interactively)")
}

func runDepsUpdate(cmd *cobra.Command, args []string) error {
	// Load configuration
	configPath := config.GetConfigPath()
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Discover packages and analyze dependencies
	searchPaths := discovery.GetDefaultSearchPaths()
	graph, err := deps.AnalyzeDependencies(searchPaths)
	if err != nil {
		return fmt.Errorf("failed to analyze dependencies: %w", err)
	}

	// Validate graph for general circular dependencies
	if err := graph.ValidateGraph(); err != nil {
		return fmt.Errorf("dependency validation failed: %w", err)
	}

	// Let user select a package to update
	packages := make([]discovery.Package, 0, len(graph.Packages))
	for _, pkg := range graph.Packages {
		packages = append(packages, discovery.Package{
			ModulePath:  pkg.ModulePath,
			PackageName: pkg.PackageName,
			Path:        pkg.Directory,
			GoVersion:   "", // We don't need this for dependency updates
			GitHubRepo:  "", // We don't need this for dependency updates
		})
	}

	color.Cyan("Select a package to update:")
	selectedPackage, err := interactive.SelectPackage(packages)
	if err != nil {
		return fmt.Errorf("failed to select package: %w", err)
	}

	// Get the package from the graph
	pkg, exists := graph.GetPackageByPath(selectedPackage.ModulePath)
	if !exists {
		return fmt.Errorf("package not found in dependency graph")
	}

	// Check for circular dependencies involving this specific package
	color.Yellow("\n🔍 Checking for circular dependencies...")
	if err := graph.CheckCircularDependencyWithPackage(selectedPackage.ModulePath); err != nil {
		color.Red("❌ CIRCULAR DEPENDENCY DETECTED!")
		color.Red("Cannot update package %s due to circular dependency", pkg.PackageName)
		color.Red("Error: %v", err)

		// Try to get the circular dependency path for more details
		if cyclePath, pathErr := graph.GetCircularDependencyPath(selectedPackage.ModulePath); pathErr == nil {
			color.Yellow("\n🔄 Circular dependency path:")
			for i, cyclePkg := range cyclePath {
				if i == len(cyclePath)-1 {
					color.Red("  %s", cyclePkg)
				} else {
					color.White("  %s →", cyclePkg)
				}
			}
		}

		color.Yellow("\n💡 To resolve this issue:")
		color.White("  1. Review the dependency structure")
		color.White("  2. Remove or refactor the circular dependency")
		color.White("  3. Use 'tag-manager deps' to visualize dependencies")

		return fmt.Errorf("update cancelled due to circular dependency")
	}

	color.Green("✅ No circular dependencies detected")

	// Show dependency impact
	impact := graph.GetUpdateImpact(selectedPackage.ModulePath)
	color.Yellow("\n📊 Update Impact Analysis")
	color.White("=========================")
	color.White("Package to update: %s", pkg.PackageName)
	color.White("Dependencies: %d", len(pkg.Dependencies))
	color.White("Dependents to update: %d", len(impact))

	if len(impact) > 0 {
		color.Cyan("\nPackages that will be updated:")
		for _, dependentPath := range impact {
			if dependentPkg, exists := graph.GetPackageByPath(dependentPath); exists {
				color.White("  • %s", dependentPkg.PackageName)
			}
		}
	}

	// Get version type
	var versionType string
	if depsUpdateVersion != "" {
		versionType = depsUpdateVersion
		if versionType != "major" && versionType != "minor" && versionType != "patch" {
			return fmt.Errorf("invalid version type: %s. Must be major, minor, or patch", versionType)
		}
	} else {
		versionType, err = interactive.SelectVersionType()
		if err != nil {
			return fmt.Errorf("failed to select version type: %w", err)
		}
	}

	// Setup package configuration
	pkgConfig, err := interactive.SetupPackageConfig(cfg, *selectedPackage)
	if err != nil {
		return fmt.Errorf("failed to setup package configuration: %w", err)
	}

	// Save configuration if updated
	if err := config.SaveConfig(cfg, configPath); err != nil {
		color.Yellow("Warning: failed to save configuration: %v", err)
	}

	// Create update plan
	updatePlan, err := createUpdatePlan(graph, pkg, versionType, pkgConfig.TagFormat)
	if err != nil {
		return fmt.Errorf("failed to create update plan: %w", err)
	}

	// Show update plan
	color.Green("\n🚀 Update Plan")
	color.White("=============")
	for i, step := range updatePlan {
		color.White("%d. %s → %s", i+1, step.PackageName, step.NewTag)
	}

	// Confirm update
	if !depsUpdateAuto {
		if !interactive.AskForConfirmation("\nDo you want to proceed with the updates?") {
			color.Yellow("Update cancelled.")
			return nil
		}
	}

	// Execute updates
	color.Green("\n🔄 Executing Updates")
	color.White("===================")

	successCount := 0
	for i, step := range updatePlan {
		color.Cyan("Updating %s (%d/%d)...", step.PackageName, i+1, len(updatePlan))

		if err := executeUpdate(step); err != nil {
			color.Red("❌ Failed to update %s: %v", step.PackageName, err)
			if !depsUpdateAuto {
				if !interactive.AskForConfirmation("Continue with remaining updates?") {
					break
				}
			}
		} else {
			color.Green("✅ Successfully updated %s to %s", step.PackageName, step.NewTag)
			successCount++
		}
	}

	// Summary
	color.Green("\n📋 Update Summary")
	color.White("=================")
	color.White("Total packages: %d", len(updatePlan))
	color.White("Successfully updated: %d", successCount)
	color.White("Failed: %d", len(updatePlan)-successCount)

	if successCount == len(updatePlan) {
		color.Green("🎉 All packages updated successfully!")
	} else if successCount > 0 {
		color.Yellow("⚠️  Some packages failed to update. Check the errors above.")
	} else {
		color.Red("❌ No packages were updated successfully.")
	}

	return nil
}

// UpdateStep represents a single update operation
type UpdateStep struct {
	PackageName string
	ModulePath  string
	Directory   string
	CurrentTag  string
	NewTag      string
	VersionType string
}

// createUpdatePlan creates a plan for updating packages in the correct order
func createUpdatePlan(graph *deps.DependencyGraph, rootPkg *deps.Package, versionType, tagFormat string) ([]UpdateStep, error) {
	var plan []UpdateStep

	// Get all packages that need to be updated (root + dependents)
	impact := graph.GetUpdateImpact(rootPkg.ModulePath)
	allPackages := append([]string{rootPkg.ModulePath}, impact...)

	// Create update steps for each package
	for _, modulePath := range allPackages {
		pkg, exists := graph.GetPackageByPath(modulePath)
		if !exists {
			continue
		}

		// Get current tag
		currentTag, err := getCurrentTag(pkg.Directory, tagFormat)
		if err != nil {
			return nil, fmt.Errorf("failed to get current tag for %s: %w", pkg.PackageName, err)
		}

		// Parse current tag
		currentTagInfo, err := tagutils.ParseTag(currentTag)
		if err != nil {
			// If we can't parse the current tag, start from v0.0.0
			currentTagInfo = &tagutils.TagInfo{
				PackageName: pkg.PackageName,
				Major:       0,
				Minor:       0,
				Patch:       0,
				Version:     "v0.0.0",
			}
		}

		// Calculate new version
		newVersion, err := tagutils.CalculateNewVersion(currentTagInfo, versionType)
		if err != nil {
			return nil, fmt.Errorf("failed to calculate new version for %s: %w", pkg.PackageName, err)
		}

		// Format new tag
		newTag := tagutils.FormatTag(tagFormat, *newVersion)

		plan = append(plan, UpdateStep{
			PackageName: pkg.PackageName,
			ModulePath:  pkg.ModulePath,
			Directory:   pkg.Directory,
			CurrentTag:  currentTag,
			NewTag:      newTag,
			VersionType: versionType,
		})
	}

	return plan, nil
}

// executeUpdate executes a single update step
func executeUpdate(step UpdateStep) error {
	// Create git tag
	cmd := exec.Command("git", "tag", "-a", step.NewTag, "-m", fmt.Sprintf("Release %s for %s", step.NewTag, step.ModulePath))
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create git tag: %w", err)
	}

	// Push the tag
	pushCmd := exec.Command("git", "push", "origin", step.NewTag)
	if err := pushCmd.Run(); err != nil {
		return fmt.Errorf("failed to push git tag: %w", err)
	}

	return nil
}
