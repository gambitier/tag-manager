package cmd

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/fatih/color"
	"github.com/gambitier/tag-manager/pkg/config"
	"github.com/gambitier/tag-manager/pkg/discovery"
	"github.com/gambitier/tag-manager/pkg/interactive"
	"github.com/gambitier/tag-manager/pkg/tagutils"
	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update version tags for a package",
	Long:  `Update version tags for a package across multiple repositories. You will be guided through selecting a package and version type interactively.`,
	RunE:  runUpdate,
}

var (
// No flags needed - everything will be interactive
)

func runUpdate(cmd *cobra.Command, args []string) error {
	// Load configuration
	configPath := config.GetConfigPath()
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

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

	// Let user select a package
	selectedPackage, err := interactive.SelectPackage(packages)
	if err != nil {
		return fmt.Errorf("failed to select package: %w", err)
	}

	// Setup package configuration if needed
	pkgConfig, err := interactive.SetupPackageConfig(cfg, *selectedPackage)
	if err != nil {
		return fmt.Errorf("failed to setup package configuration: %w", err)
	}

	// Save configuration if it was updated
	if err := config.SaveConfig(cfg, configPath); err != nil {
		color.Yellow("Warning: failed to save configuration: %v", err)
	}

	// Let user select version type
	versionType, err := interactive.SelectVersionType()
	if err != nil {
		return fmt.Errorf("failed to select version type: %w", err)
	}

	// Get current tag
	currentTag, err := getCurrentTag(selectedPackage.ModulePath, pkgConfig.TagFormat)
	if err != nil {
		return fmt.Errorf("failed to get current tag: %w", err)
	}

	// Parse current tag
	currentTagInfo, err := tagutils.ParseTag(currentTag)
	if err != nil {
		// If we can't parse the current tag, start from v0.0.0
		packageName := tagutils.ExtractPackageNameFromModule(selectedPackage.ModulePath)
		currentTagInfo = &tagutils.TagInfo{
			PackageName: packageName,
			Major:       0,
			Minor:       0,
			Patch:       0,
			Version:     "v0.0.0",
		}
	}

	// Calculate new version
	newVersion, err := tagutils.CalculateNewVersion(currentTagInfo, versionType)
	if err != nil {
		return fmt.Errorf("failed to calculate new version: %w", err)
	}

	// Format new tag
	newTag := tagutils.FormatTag(pkgConfig.TagFormat, *newVersion)

	// Display information
	color.Green("\n=== Tag Update Summary ===")
	color.White("Package: %s", selectedPackage.ModulePath)
	color.White("Package Name: %s", selectedPackage.PackageName)
	color.White("Tag Format: %s", pkgConfig.TagFormat)
	color.Yellow("Current tag: %s", currentTag)
	color.Cyan("New tag: %s", newTag)
	color.Cyan("Version type: %s", versionType)

	// Ask for confirmation
	if !interactive.AskForConfirmation("Do you want to update the tag?") {
		color.Yellow("Tag update cancelled.")
		return nil
	}

	// Update the tag
	if err := updateTag(selectedPackage.ModulePath, newTag); err != nil {
		return fmt.Errorf("failed to update tag: %w", err)
	}

	color.Green("Successfully updated tag to %s for package %s", newTag, selectedPackage.ModulePath)
	return nil
}

func getCurrentTag(modulePath, tagFormat string) (string, error) {
	// Try to find existing tags that match the expected format
	// We'll search for tags that could match our format
	packageName := tagutils.ExtractPackageNameFromModule(modulePath)

	// Try different tag patterns
	patterns := []string{
		fmt.Sprintf("%s/*", packageName),
		fmt.Sprintf("%s-*", packageName),
		"v*",
		"*",
	}

	for _, pattern := range patterns {
		cmd := exec.Command("git", "tag", "--list", pattern, "--sort=-version:refname")
		output, err := cmd.Output()
		if err != nil {
			continue
		}

		tags := strings.Split(strings.TrimSpace(string(output)), "\n")
		if len(tags) > 0 && tags[0] != "" {
			// Return the most recent tag
			return tags[0], nil
		}
	}

	// If no tags found, return empty string (will be handled as v0.0.0)
	return "", nil
}

func updateTag(modulePath, newTag string) error {
	// Run git tag command with annotated tag and message
	cmd := exec.Command("git", "tag", "-a", newTag, "-m", fmt.Sprintf("Release %s for %s", newTag, modulePath))
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create git tag: %w", err)
	}

	// Push the tag
	pushCmd := exec.Command("git", "push", "origin", newTag)
	if err := pushCmd.Run(); err != nil {
		return fmt.Errorf("failed to push git tag: %w", err)
	}

	return nil
}
