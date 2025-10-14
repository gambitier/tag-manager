package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

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

	// Get actual package information with tags
	actualPackages, err := discovery.DiscoverPackages(searchPaths)
	if err != nil {
		return fmt.Errorf("failed to discover packages: %w", err)
	}

	// Filter packages to only include those in the dependency graph
	var packages []discovery.Package
	for _, actualPkg := range actualPackages {
		if _, exists := graph.Packages[actualPkg.ModulePath]; exists {
			packages = append(packages, actualPkg)
		}
	}

	if len(packages) == 0 {
		color.Red("No packages found in dependency graph.")
		return nil
	}

	// Use the new wrapper function
	selectedPackage, err := interactive.SelectPackageWithDisplayAndPrompt(packages, "Available packages", "Select a package to update")
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

	// Show configuration information and options
	color.Cyan("\n⚙️  Configuration Management")
	color.White("============================")
	color.White("Config file location: %s", configPath)

	// Show configuration status for packages that will be updated
	allPackages := append([]string{selectedPackage.ModulePath}, impact...)
	var packagesNeedingConfig []string
	var packagesAlreadyConfigured []string

	for _, modulePath := range allPackages {
		pkgName := filepath.Base(modulePath)

		// Check if package actually exists in config (not just has default format)
		if _, exists := cfg.Packages[modulePath]; exists {
			packagesAlreadyConfigured = append(packagesAlreadyConfigured, pkgName)
		} else {
			packagesNeedingConfig = append(packagesNeedingConfig, pkgName)
		}
	}

	color.Yellow("\nPackage Configuration Status:")
	if len(packagesNeedingConfig) > 0 {
		color.Red("  ❌ NEEDS CONFIGURATION: %s", strings.Join(packagesNeedingConfig, ", "))
		color.White("     These packages don't have tag formats set up yet")
	}
	if len(packagesAlreadyConfigured) > 0 {
		color.Green("  ✅ ALREADY CONFIGURED: %s", strings.Join(packagesAlreadyConfigured, ", "))
		color.White("     These packages have tag formats set up")
	}

	// Show summary
	totalPackages := len(allPackages)
	configuredCount := len(packagesAlreadyConfigured)
	needingConfigCount := len(packagesNeedingConfig)

	color.Cyan("\nSummary: %d/%d packages configured", configuredCount, totalPackages)
	if needingConfigCount > 0 {
		color.Yellow("⚠️  %d packages need configuration before update", needingConfigCount)
	} else {
		color.Green("🎉 All packages are configured and ready for update!")
	}

	// Ask user about configuration management
	color.Yellow("\nConfiguration Options:")

	var options []string
	if needingConfigCount > 0 {
		options = []string{
			"Bulk configure all packages with same format (recommended)",
			"Configure packages one by one",
			"Manually edit config file and resume",
		}
	} else {
		options = []string{
			"Continue with dependency update (all packages configured)",
			"Manually edit config file and resume",
		}
	}

	configChoice, err := interactive.SelectOption(options)
	if err != nil {
		return fmt.Errorf("failed to select configuration option: %w", err)
	}

	if needingConfigCount > 0 {
		// Packages need configuration
		switch configChoice {
		case 1: // Bulk configure packages
			return handleBulkConfigSetup(cfg, configPath, allPackages)
		case 2: // Configure packages one by one
			color.Green("Continuing with individual package configuration setup...")
		case 3: // Manually edit config file
			return handleManualConfigEdit(configPath, selectedPackage.ModulePath)
		}
	} else {
		// All packages already configured
		switch configChoice {
		case 1: // Continue with dependency update
			color.Green("All packages configured! Continuing with dependency update...")
		case 2: // Manually edit config file
			return handleManualConfigEdit(configPath, selectedPackage.ModulePath)
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

	// Setup package configuration for the root package
	_, err = interactive.SetupPackageConfig(cfg, *selectedPackage)
	if err != nil {
		return fmt.Errorf("failed to setup package configuration: %w", err)
	}

	// Save configuration if updated
	if err := config.SaveConfig(cfg, configPath); err != nil {
		color.Yellow("Warning: failed to save configuration: %v", err)
	}

	// Create update plan
	updatePlan, err := createUpdatePlan(graph, pkg, versionType, cfg)
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
func createUpdatePlan(graph *deps.DependencyGraph, rootPkg *deps.Package, versionType string, cfg *config.Config) ([]UpdateStep, error) {
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

		// Get package-specific configuration
		pkgConfig := cfg.GetPackageConfig(pkg.ModulePath)

		// Get current tag using package-specific format
		currentTag, err := getCurrentTag(pkg.Directory, pkgConfig.TagFormat)
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

		// Format new tag using package-specific format
		newTag := tagutils.FormatTag(pkgConfig.TagFormat, *newVersion)

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

// handleManualConfigEdit handles the manual configuration editing flow
func handleManualConfigEdit(configPath, packageModulePath string) error {
	color.Cyan("\n📝 Manual Configuration Edit")
	color.White("============================")

	// Check if config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		color.Yellow("Config file does not exist. Creating default configuration...")
		// Create default config
		defaultConfig := config.GetDefaultConfig()
		if err := config.SaveConfig(defaultConfig, configPath); err != nil {
			return fmt.Errorf("failed to create default config: %w", err)
		}
	}

	// Show config file path and instructions
	color.Yellow("\n📁 Configuration File:")
	color.White("File: %s", configPath)

	// Get file size
	if fileInfo, err := os.Stat(configPath); err == nil {
		color.White("Size: %d bytes", fileInfo.Size())
	}

	// Instructions for editing
	color.Cyan("\n📋 Instructions:")
	color.White("1. Edit the config file above to customize package tag formats")
	color.White("2. Save the file when done")
	color.White("3. Run the command again to continue")

	color.Green("\n✅ Ready for manual editing!")
	color.White("Run the command again after editing the config file.")

	// Exit gracefully without error
	os.Exit(0)
	return nil // This will never be reached, but satisfies the compiler
}

// handleBulkConfigSetup handles bulk configuration setup for multiple packages
func handleBulkConfigSetup(cfg *config.Config, configPath string, allPackages []string) error {
	color.Cyan("\n🔧 Bulk Configuration Setup")
	color.White("============================")

	// Show packages that need configuration
	var packagesNeedingConfig []string
	for _, modulePath := range allPackages {
		pkgConfig := cfg.GetPackageConfig(modulePath)
		if pkgConfig.TagFormat == "" {
			packagesNeedingConfig = append(packagesNeedingConfig, filepath.Base(modulePath))
		}
	}

	if len(packagesNeedingConfig) == 0 {
		color.Green("All packages already have configurations!")
		color.White("Continuing with existing configurations...")
		return nil
	}

	color.Yellow("Packages needing configuration: %s", strings.Join(packagesNeedingConfig, ", "))

	// Ask for tag format
	color.Cyan("\nTag Format Options:")
	tagFormatChoice, err := interactive.SelectOption([]string{
		"Default format: {package-name}/v{major}.{minor}.{patch}",
		"Simple format: v{major}.{minor}.{patch}",
		"Custom format (you'll be prompted)",
	})
	if err != nil {
		return fmt.Errorf("failed to select tag format: %w", err)
	}

	var tagFormat string
	switch tagFormatChoice {
	case 1: // Default format
		tagFormat = "{package-name}/v{major}.{minor}.{patch}"
	case 2: // Simple format
		tagFormat = "v{major}.{minor}.{patch}"
	case 3: // Custom format
		color.Cyan("\nEnter custom tag format:")
		color.White("Available placeholders: {package-name}, {major}, {minor}, {patch}")
		color.White("Example: {package-name}-v{major}.{minor}.{patch}")
		fmt.Print("Format: ")
		fmt.Scanln(&tagFormat)
		if tagFormat == "" {
			return fmt.Errorf("tag format cannot be empty")
		}
	}

	// Apply configuration to all packages
	color.Yellow("\nApplying configuration to packages...")
	updatedCount := 0

	for _, modulePath := range allPackages {
		pkgConfig := cfg.GetPackageConfig(modulePath)
		if pkgConfig.TagFormat == "" {
			// Set the configuration
			newConfig := config.PackageConfig{
				TagFormat:  tagFormat,
				UseDefault: tagFormatChoice == 1,
			}
			cfg.SetPackageConfig(modulePath, newConfig)
			updatedCount++
		}
	}

	// Save configuration
	if err := config.SaveConfig(cfg, configPath); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	color.Green("✅ Bulk configuration completed!")
	color.White("Updated %d packages with format: %s", updatedCount, tagFormat)
	color.White("Continuing with dependency update...")

	return nil
}
