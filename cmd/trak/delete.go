package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var (
	deleteRemote bool
	deleteForce  bool
)

var deleteCmd = &cobra.Command{
	Use:   "delete <branch>",
	Short: "Delete a track",
	Long: `Delete a track's local worktree or devbox.

By default, only deletes the local environment and database record.
Use --remote to also delete the remote branch on GitHub.`,
	Args: cobra.ExactArgs(1),
	RunE: runDelete,
}

func init() {
	deleteCmd.Flags().BoolVarP(&deleteRemote, "remote", "r", false, "Also delete the remote branch")
	deleteCmd.Flags().BoolVarP(&deleteForce, "force", "f", false, "Skip confirmation prompt")
}

func runDelete(cmd *cobra.Command, args []string) error {
	branch := args[0]

	opsLayer, cleanup, err := initOps()
	if err != nil {
		return err
	}
	defer cleanup()

	// Confirm deletion unless --force is specified
	if !deleteForce {
		action := "local track"
		if deleteRemote {
			action = "local track AND remote branch"
		}

		fmt.Printf("Delete %s for branch '%s'? [y/N]: ", action, branch)

		reader := bufio.NewReader(os.Stdin)
		input, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read input: %w", err)
		}

		input = strings.TrimSpace(strings.ToLower(input))
		if input != "y" && input != "yes" {
			fmt.Println("Cancelled.")
			return nil
		}
	}

	fmt.Printf("Deleting track '%s'...\n", branch)

	if err := opsLayer.DeleteTrack(branch, deleteRemote); err != nil {
		return fmt.Errorf("failed to delete track: %w", err)
	}

	if deleteRemote {
		fmt.Println("Track and remote branch deleted.")
	} else {
		fmt.Println("Track deleted. Remote branch preserved.")
	}

	return nil
}
