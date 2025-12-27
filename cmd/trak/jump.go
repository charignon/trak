package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var jumpCmd = &cobra.Command{
	Use:   "jump <branch>",
	Short: "Jump to a track's tmux window",
	Long: `Open or switch to the tmux window for the specified track.

If the window doesn't exist, it will be created. For worktree tracks,
the window opens in the worktree directory. For devbox tracks, an SSH
session is started.`,
	Args: cobra.ExactArgs(1),
	RunE: runJump,
}

func runJump(cmd *cobra.Command, args []string) error {
	branch := args[0]

	opsLayer, cleanup, err := initOps()
	if err != nil {
		return err
	}
	defer cleanup()

	if err := opsLayer.JumpToTrack(branch); err != nil {
		return fmt.Errorf("failed to jump to track: %w", err)
	}

	return nil
}
