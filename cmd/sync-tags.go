package cmd

import (
	"fmt"
	"os/exec"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var syncTagsCmd = &cobra.Command{
	Use:   "sync-tags",
	Short: "Sync local tags with upstream repository",
	Long: `Sync local tags with upstream repository by:
1. Deleting all local tags
2. Fetching all tags from upstream

This ensures your local tags match exactly what's on the remote repository.`,
	RunE: runSyncTags,
}

func init() {
	// Add any flags here if needed
}

func runSyncTags(cmd *cobra.Command, args []string) error {
	color.Cyan("🔄 Syncing local tags with upstream...")
	color.White("================================")

	// Step 1: Delete all local tags
	color.Yellow("Step 1: Deleting all local tags...")
	deleteCmd := exec.Command("sh", "-c", "git tag -l | xargs git tag -d")
	if err := deleteCmd.Run(); err != nil {
		color.Red("❌ Failed to delete local tags: %v", err)
		return fmt.Errorf("failed to delete local tags: %w", err)
	}
	color.Green("✅ Local tags deleted")

	// Step 2: Fetch all tags from upstream
	color.Yellow("Step 2: Fetching all tags from upstream...")
	fetchCmd := exec.Command("git", "fetch", "origin", "--prune", "--tags")
	if err := fetchCmd.Run(); err != nil {
		color.Red("❌ Failed to fetch tags from upstream: %v", err)
		return fmt.Errorf("failed to fetch tags from upstream: %w", err)
	}
	color.Green("✅ Tags fetched from upstream")

	// Show summary
	color.Green("\n🎉 Tag sync completed successfully!")
	color.White("Your local tags now match the upstream repository.")

	return nil
}
