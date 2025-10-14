package interactive

import (
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
