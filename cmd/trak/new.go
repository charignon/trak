package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var (
	newWorktree bool
	newDevbox   bool
)

var newCmd = &cobra.Command{
	Use:   "new <branch>",
	Short: "Create a new track",
	Long: `Create a new track for the specified branch.

By default, prompts for track type (worktree or devbox).
Use --worktree or --devbox to skip the prompt.`,
	Args: cobra.ExactArgs(1),
	RunE: runNew,
}

func init() {
	newCmd.Flags().BoolVarP(&newWorktree, "worktree", "w", false, "Create a worktree track")
	newCmd.Flags().BoolVarP(&newDevbox, "devbox", "d", false, "Create a devbox track")
}

func runNew(cmd *cobra.Command, args []string) error {
	branch := args[0]

	// Validate flags
	if newWorktree && newDevbox {
		return fmt.Errorf("cannot specify both --worktree and --devbox")
	}

	opsLayer, cleanup, err := initOps()
	if err != nil {
		return err
	}
	defer cleanup()

	// Determine track type
	var trackType string
	if newWorktree {
		trackType = "worktree"
	} else if newDevbox {
		trackType = "devbox"
	} else {
		// Prompt user
		trackType, err = promptTrackType()
		if err != nil {
			return err
		}
	}

	// Create the track
	switch trackType {
	case "worktree":
		fmt.Printf("Creating worktree track for branch '%s'...\n", branch)
		if err := opsLayer.NewTrackWorktree(branch); err != nil {
			return fmt.Errorf("failed to create worktree: %w", err)
		}
		fmt.Println("Worktree track created successfully.")

	case "devbox":
		fmt.Printf("Creating devbox track for branch '%s'...\n", branch)
		if err := opsLayer.NewTrackDevbox(branch); err != nil {
			return fmt.Errorf("failed to create devbox: %w", err)
		}
		fmt.Println("Devbox track created successfully.")

	default:
		return fmt.Errorf("unknown track type: %s", trackType)
	}

	fmt.Printf("Use 'trak jump %s' to switch to the track.\n", branch)
	return nil
}

func promptTrackType() (string, error) {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("Select track type:")
	fmt.Println("  [w] worktree - Local git worktree")
	fmt.Println("  [d] devbox   - Remote k8s dev environment")
	fmt.Print("Choice (w/d): ")

	input, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("failed to read input: %w", err)
	}

	input = strings.TrimSpace(strings.ToLower(input))

	switch input {
	case "w", "worktree":
		return "worktree", nil
	case "d", "devbox":
		return "devbox", nil
	default:
		return "", fmt.Errorf("invalid choice: %s (expected 'w' or 'd')", input)
	}
}
