package main

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/laurent/trak/internal/db"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all tracks",
	Long:  `List all tracks with their status in a table format.`,
	RunE:  runList,
}

func runList(cmd *cobra.Command, args []string) error {
	opsLayer, cleanup, err := initOps()
	if err != nil {
		return err
	}
	defer cleanup()

	tracks, err := opsLayer.ListTracksWithStatus()
	if err != nil {
		return fmt.Errorf("failed to list tracks: %w", err)
	}

	if len(tracks) == 0 {
		fmt.Println("No tracks found. Use 'trak new <branch>' to create one.")
		return nil
	}

	// Create a tabwriter for aligned columns
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	// Print header
	fmt.Fprintln(w, "BRANCH\tTYPE\tGIT\tPR\tCI\tREVIEW\tAGE")
	fmt.Fprintln(w, "──────\t────\t───\t──\t──\t──────\t───")

	for _, t := range tracks {
		branch := t.Track.Branch
		if len(branch) > 25 {
			branch = branch[:22] + "..."
		}

		trackType := string(t.Track.Type)

		// Git status
		gitStatus := t.Status.GitStatus.String()

		// PR status
		prStatus := "—"
		if t.Status.PR != nil {
			prStatus = fmt.Sprintf("#%d", t.Status.PR.Number)
		}

		// CI status
		ciStatus := "—"
		if t.Status.CI != nil {
			ciStatus = t.Status.CI.Symbol()
		}

		// Review status
		reviewStatus := "—"
		if t.Status.Review != nil {
			reviewStatus = t.Status.Review.Symbol()
		}

		// Age
		age := formatAge(t.Track.CreatedAt)

		// Mark stale tracks
		if t.Status.IsStale {
			branch = branch + " (stale)"
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			branch, trackType, gitStatus, prStatus, ciStatus, reviewStatus, age)
	}

	w.Flush()
	return nil
}

// formatAge returns a human-readable age string.
func formatAge(t time.Time) string {
	d := time.Since(t)

	if d < time.Minute {
		return "now"
	}
	if d < time.Hour {
		mins := int(d.Minutes())
		return fmt.Sprintf("%dm", mins)
	}
	if d < 24*time.Hour {
		hours := int(d.Hours())
		return fmt.Sprintf("%dh", hours)
	}
	days := int(d.Hours() / 24)
	if days < 7 {
		return fmt.Sprintf("%dd", days)
	}
	weeks := days / 7
	if weeks < 4 {
		return fmt.Sprintf("%dw", weeks)
	}
	months := days / 30
	return fmt.Sprintf("%dmo", months)
}

// truncate truncates a string to max length with ellipsis.
func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	if max <= 3 {
		return s[:max]
	}
	return s[:max-3] + "..."
}

// padRight pads a string to the right with spaces.
func padRight(s string, length int) string {
	if len(s) >= length {
		return s
	}
	return s + strings.Repeat(" ", length-len(s))
}

func init() {
	// Track type is determined from db.TrackType
	_ = db.TrackTypeWorktree
}
