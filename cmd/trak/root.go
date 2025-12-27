// Package main provides the CLI commands for trak.
package main

import (
	"fmt"
	"os"

	"github.com/laurent/trak/internal/config"
	"github.com/laurent/trak/internal/db"
	"github.com/laurent/trak/internal/ops"
	"github.com/laurent/trak/internal/tui"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "trak",
	Short: "Track management for git worktrees and devboxes",
	Long: `trak provides a unified view of git worktrees and devboxes for a single repo.

Running trak without arguments opens the TUI dashboard.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		opsLayer, cleanup, err := initOps()
		if err != nil {
			return err
		}
		defer cleanup()

		cfg, _ := config.Load()
		return tui.Run(opsLayer, cfg.Repo.Remote)
	},
}

// initOps initializes the ops layer with config and database.
// Returns the ops instance and a cleanup function.
func initOps() (*ops.Ops, func(), error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load config: %w", err)
	}

	dbPath := config.GetDBPath()
	database, err := db.Open(dbPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := database.Migrate(); err != nil {
		database.Close()
		return nil, nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	opsLayer := ops.New(database, cfg)
	cleanup := func() {
		database.Close()
	}

	return opsLayer, cleanup, nil
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	// Add subcommands
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(jumpCmd)
	rootCmd.AddCommand(newCmd)
	rootCmd.AddCommand(syncCmd)
	rootCmd.AddCommand(deleteCmd)
	rootCmd.AddCommand(aiCmd)
	rootCmd.AddCommand(remoteCmd)
}
