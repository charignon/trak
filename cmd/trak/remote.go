package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

var remoteCmd = &cobra.Command{
	Use:   "remote",
	Short: "Browse remote branches",
	Long: `Interactively browse your remote branches and optionally create tracks from them.

Shows branches from your open PRs on GitHub.`,
	RunE: runRemote,
}

func runRemote(cmd *cobra.Command, args []string) error {
	opsLayer, cleanup, err := initOps()
	if err != nil {
		return err
	}
	defer cleanup()

	fmt.Println("Fetching remote branches...")

	branches, err := opsLayer.ListRemoteBranches()
	if err != nil {
		return fmt.Errorf("failed to list remote branches: %w", err)
	}

	if len(branches) == 0 {
		fmt.Println("No remote branches found with open PRs.")
		return nil
	}

	// Display branches
	fmt.Println("\nRemote branches:")
	fmt.Println("  #  BRANCH                         PR      AGE")
	fmt.Println("  ── ──────                         ──      ───")

	for i, b := range branches {
		branchName := b.Name
		if len(branchName) > 30 {
			branchName = branchName[:27] + "..."
		}

		prInfo := "—"
		if b.HasPR {
			prInfo = fmt.Sprintf("#%d", b.PRNumber)
		}

		age := formatAgeDuration(b.Age)

		fmt.Printf("  %-2d %-30s %-7s %s\n", i+1, branchName, prInfo, age)
	}

	// Prompt for selection
	fmt.Print("\nEnter number to create track (or 'q' to quit): ")

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read input: %w", err)
	}

	input = strings.TrimSpace(input)
	if input == "q" || input == "" {
		return nil
	}

	// Parse selection
	num, err := strconv.Atoi(input)
	if err != nil || num < 1 || num > len(branches) {
		return fmt.Errorf("invalid selection: %s", input)
	}

	selectedBranch := branches[num-1]

	// Prompt for track type
	trackType, err := promptTrackType()
	if err != nil {
		return err
	}

	// Create the track
	switch trackType {
	case "worktree":
		fmt.Printf("Creating worktree track for '%s'...\n", selectedBranch.Name)
		if err := opsLayer.NewTrackWorktree(selectedBranch.Name); err != nil {
			return fmt.Errorf("failed to create worktree: %w", err)
		}
	case "devbox":
		fmt.Printf("Creating devbox track for '%s'...\n", selectedBranch.Name)
		if err := opsLayer.NewTrackDevbox(selectedBranch.Name); err != nil {
			return fmt.Errorf("failed to create devbox: %w", err)
		}
	}

	fmt.Printf("Track created. Use 'trak jump %s' to switch to it.\n", selectedBranch.Name)
	return nil
}

// formatAgeDuration formats a duration as a human-readable age string.
func formatAgeDuration(d interface{}) string {
	// Handle both time.Duration and the Age field from RemoteBranch
	switch v := d.(type) {
	case interface{ Hours() float64 }:
		hours := v.Hours()
		if hours < 1 {
			return "now"
		}
		if hours < 24 {
			return fmt.Sprintf("%dh", int(hours))
		}
		days := int(hours / 24)
		if days < 7 {
			return fmt.Sprintf("%dd", days)
		}
		weeks := days / 7
		if weeks < 4 {
			return fmt.Sprintf("%dw", weeks)
		}
		months := days / 30
		return fmt.Sprintf("%dmo", months)
	default:
		return "—"
	}
}
