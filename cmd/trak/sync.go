package main

import (
	"fmt"

	"github.com/laurent/trak/internal/config"
	"github.com/laurent/trak/internal/db"
	"github.com/spf13/cobra"
)

var syncCmd = &cobra.Command{
	Use:   "sync [branch]",
	Short: "Sync a track with remote",
	Long: `Sync a track: fetch, rebase onto default branch, push, and create PR if needed.

If no branch is specified, syncs the current track (if in a worktree).
Returns information about conflicts if the rebase fails.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runSync,
}

func runSync(cmd *cobra.Command, args []string) error {
	var branch string

	if len(args) > 0 {
		branch = args[0]
	} else {
		// Try to detect current branch from working directory
		// For now, require explicit branch
		return fmt.Errorf("branch argument required (auto-detect not yet implemented)")
	}

	opsLayer, cleanup, err := initOps()
	if err != nil {
		return err
	}
	defer cleanup()

	fmt.Printf("Syncing track '%s'...\n", branch)

	result, err := opsLayer.SyncTrack(branch)
	if err != nil {
		return fmt.Errorf("sync failed: %w", err)
	}

	// Report results
	if result.HasConflicts {
		fmt.Println("\nRebase conflicts detected!")
		fmt.Printf("Please resolve conflicts in: %s\n", result.ConflictsPath)
		fmt.Println("Then run 'git rebase --continue' and 'trak sync' again.")
		return nil
	}

	if result.Rebased {
		fmt.Println("Rebased onto default branch.")
	}

	if result.Pushed {
		fmt.Println("Pushed to remote.")
	}

	if result.PRCreated {
		fmt.Printf("Created PR #%d\n", result.PRNumber)
	}

	fmt.Println("Sync complete.")
	return nil
}

// getCurrentBranch attempts to detect the current branch from the working directory.
// This is a placeholder for future implementation.
func getCurrentBranch() (string, error) {
	cfg, err := config.Load()
	if err != nil {
		return "", err
	}

	dbPath := config.GetDBPath()
	database, err := db.Open(dbPath)
	if err != nil {
		return "", err
	}
	defer database.Close()

	// Get current working directory and check if it's a track
	// TODO: Implement this by checking if cwd matches a track path
	_ = cfg
	return "", fmt.Errorf("not in a track directory")
}
