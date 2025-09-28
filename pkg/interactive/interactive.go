package interactive

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/fatih/color"
	"github.com/gambitier/tag-manager/pkg/config"
	"github.com/gambitier/tag-manager/pkg/discovery"
	"github.com/gambitier/tag-manager/pkg/tagutils"
)

// SetupPackageConfig interactively sets up configuration for a package
func SetupPackageConfig(cfg *config.Config, pkg discovery.Package) (*config.PackageConfig, error) {
	color.Cyan("\n=== Package Configuration Setup ===")
	color.White("Package: %s", pkg.ModulePath)
	color.White("Path: %s", pkg.Path)
	color.White("Package Name: %s", pkg.PackageName)

	// Check if package already has configuration
	if existingConfig, exists := cfg.Packages[pkg.ModulePath]; exists {
		color.Green("Package already configured:")
		color.White("  Tag Format: %s", existingConfig.TagFormat)
		color.White("  Use Default: %t", existingConfig.UseDefault)

		if AskForConfirmation("Do you want to reconfigure this package?") {
			return configurePackage(cfg, pkg)
		}
		return &existingConfig, nil
	}

	return configurePackage(cfg, pkg)
}

// configurePackage handles the interactive configuration process
func configurePackage(cfg *config.Config, pkg discovery.Package) (*config.PackageConfig, error) {
	pkgConfig := config.PackageConfig{
		ModulePath: pkg.ModulePath,
	}

	// Ask if user wants to use default format
	color.Cyan("\nTag Format Options:")
	color.White("1. Use default format: %s", cfg.Defaults.TagFormat)
	color.White("2. Define custom format")

	choice, err := selectOption(1, 2)
	if err != nil {
		return nil, err
	}

	if choice == 1 {
		// Use default format
		pkgConfig.TagFormat = cfg.Defaults.TagFormat
		pkgConfig.UseDefault = true
		color.Green("Using default tag format: %s", pkgConfig.TagFormat)
	} else {
		// Custom format
		customFormat, err := getCustomTagFormat()
		if err != nil {
			return nil, err
		}
		pkgConfig.TagFormat = customFormat
		pkgConfig.UseDefault = false
		color.Green("Using custom tag format: %s", pkgConfig.TagFormat)
	}

	// Show example of how the tag will look
	exampleTag := showTagExample(pkgConfig.TagFormat, pkg)
	color.Cyan("Example tag: %s", exampleTag)

	// Confirm configuration
	if !AskForConfirmation("Save this configuration?") {
		return nil, fmt.Errorf("configuration cancelled")
	}

	// Save to config
	cfg.SetPackageConfig(pkg.ModulePath, pkgConfig)

	return &pkgConfig, nil
}

// getCustomTagFormat gets a custom tag format from user input
func getCustomTagFormat() (string, error) {
	color.Cyan("\nCustom Tag Format Configuration:")
	color.White("Available placeholders:")
	color.White("  {package-name} - Package name")
	color.White("  {major} - Major version number")
	color.White("  {minor} - Minor version number")
	color.White("  {patch} - Patch version number")
	color.White("  {version} - Full version (e.g., v1.2.3)")

	color.Cyan("\nExamples:")
	color.White("  {package-name}/v{major}.{minor}.{patch}")
	color.White("  {package-name}-{major}.{minor}.{patch}")
	color.White("  v{major}.{minor}.{patch}")

	for {
		color.Cyan("Enter your custom tag format: ")
		reader := bufio.NewReader(os.Stdin)
		input, err := reader.ReadString('\n')
		if err != nil {
			return "", fmt.Errorf("failed to read input: %w", err)
		}

		format := strings.TrimSpace(input)
		if format == "" {
			color.Red("Tag format cannot be empty. Please try again.")
			continue
		}

		// Validate format
		if err := tagutils.ValidateTagFormat(format); err != nil {
			color.Red("Invalid format: %v", err)
			color.Yellow("Please try again.")
			continue
		}

		return format, nil
	}
}

// showTagExample shows an example of how a tag will look
func showTagExample(format string, pkg discovery.Package) string {
	packageName := tagutils.ExtractPackageNameFromModule(pkg.ModulePath)
	exampleInfo := tagutils.TagInfo{
		PackageName: packageName,
		Major:       1,
		Minor:       2,
		Patch:       3,
		Version:     "v1.2.3",
	}

	return tagutils.FormatTag(format, exampleInfo)
}

// SelectPackage allows user to select a package from a list
func SelectPackage(packages []discovery.Package) (*discovery.Package, error) {
	if len(packages) == 0 {
		return nil, fmt.Errorf("no packages found")
	}

	color.Cyan("\nAvailable packages:")
	for i, pkg := range packages {
		color.White("%d. %s (%s)", i+1, pkg.ModulePath, pkg.PackageName)
	}

	selection, err := selectOption(1, len(packages))
	if err != nil {
		return nil, err
	}

	return &packages[selection-1], nil
}

// SelectVersionType allows user to select a version type
func SelectVersionType() (string, error) {
	color.Cyan("\nVersion types:")
	color.White("1. major - Breaking changes (e.g., v1.2.3 → v2.0.0)")
	color.White("2. minor - New features (e.g., v1.2.3 → v1.3.0)")
	color.White("3. patch - Bug fixes (e.g., v1.2.3 → v1.2.4)")

	selection, err := selectOption(1, 3)
	if err != nil {
		return "", err
	}

	versionTypes := []string{"major", "minor", "patch"}
	return versionTypes[selection-1], nil
}

// selectOption handles generic option selection
func selectOption(min, max int) (int, error) {
	color.Cyan("Select option (enter number): ")
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return 0, fmt.Errorf("failed to read input: %w", err)
	}

	input = strings.TrimSpace(input)
	selection, err := strconv.Atoi(input)
	if err != nil {
		return 0, fmt.Errorf("invalid input, please enter a number: %w", err)
	}

	if selection < min || selection > max {
		return 0, fmt.Errorf("invalid selection, please choose a number between %d and %d", min, max)
	}

	return selection, nil
}

// AskForConfirmation asks for yes/no confirmation
func AskForConfirmation(prompt string) bool {
	color.Cyan("%s (y/N): ", prompt)
	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false
	}

	response = strings.ToLower(strings.TrimSpace(response))
	return response == "y" || response == "yes"
}
