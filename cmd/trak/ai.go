package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var aiCmd = &cobra.Command{
	Use:   "ai <branch>",
	Short: "Run AI assistant in a track",
	Long: `Run the AI assistant (toad) in the context of the specified track.

This opens a tmux window for the track and runs 'toad -a codex <path>'.
Only supported for worktree tracks (not devbox).`,
	Args: cobra.ExactArgs(1),
	RunE: runAI,
}

func runAI(cmd *cobra.Command, args []string) error {
	branch := args[0]

	opsLayer, cleanup, err := initOps()
	if err != nil {
		return err
	}
	defer cleanup()

	fmt.Printf("Starting AI assistant for track '%s'...\n", branch)

	if err := opsLayer.RunAI(branch); err != nil {
		return fmt.Errorf("failed to run AI: %w", err)
	}

	return nil
}
