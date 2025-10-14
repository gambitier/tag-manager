package interactive

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/gambitier/tag-manager/pkg/discovery"
	"github.com/gambitier/tag-manager/pkg/display"
)

// SelectPackageWithDisplay shows a package list and allows user to select one
func SelectPackageWithDisplay(packages []discovery.Package, title string) (*discovery.Package, error) {
	// Display available packages
	color.Cyan("%s:", title)
	display.ShowPackageList(packages, display.Compact)
	color.White("")

	// Let user select a package
	color.Cyan("Select option (enter number):")
	return SelectPackage(packages)
}

// SelectPackageWithDisplayAndPrompt shows a package list with custom prompt
func SelectPackageWithDisplayAndPrompt(packages []discovery.Package, title, prompt string) (*discovery.Package, error) {
	// Display available packages
	color.Cyan("%s:", title)
	display.ShowPackageList(packages, display.Compact)
	color.White("")

	// Let user select a package
	color.Cyan("%s:", prompt)
	return SelectPackage(packages)
}

// SelectOption allows user to select from a list of string options
func SelectOption(options []string) (int, error) {
	if len(options) == 0 {
		return 0, fmt.Errorf("no options provided")
	}

	// Display options
	for i, option := range options {
		color.White("%d. %s", i+1, option)
	}
	color.White("")

	// Get user input
	var choice int
	color.Cyan("Select option (enter number):")
	_, err := fmt.Scanln(&choice)
	if err != nil {
		return 0, fmt.Errorf("failed to read input: %w", err)
	}

	// Validate choice
	if choice < 1 || choice > len(options) {
		return 0, fmt.Errorf("invalid choice: %d. Must be between 1 and %d", choice, len(options))
	}

	return choice, nil
}
