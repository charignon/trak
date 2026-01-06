# Contributing to Trak

This guide explains how to extend trak with new ways to track units of work and how the existing architecture works.

## Architecture Overview

Trak manages **tracks** - isolated development environments associated with branches. Each track can be either a local git worktree or a remote devbox.

```
┌─────────────────────────────────────────────────────────────────┐
│                         CLI Layer                               │
│  cmd/trak/*.go - Cobra commands, user interaction, flags        │
└──────────────────────────┬──────────────────────────────────────┘
                           │
┌──────────────────────────▼──────────────────────────────────────┐
│                      Operations Layer                           │
│  internal/ops/ops.go - Business logic, orchestration            │
└──────────────────────────┬──────────────────────────────────────┘
                           │
       ┌───────────────────┼───────────────────┐
       │                   │                   │
┌──────▼──────┐    ┌───────▼───────┐   ┌──────▼──────┐
│  Data Layer │    │  Integration  │   │   Utilities │
│ internal/db │    │   Modules     │   │internal/track│
│   SQLite    │    │ git, github,  │   │ slug, types │
│             │    │ tmux, devbox  │   │             │
└─────────────┘    └───────────────┘   └─────────────┘
```

## Project Structure

```
trak/
├── cmd/trak/              # CLI commands
│   ├── main.go            # Entry point
│   ├── root.go            # Root command, TUI launch
│   ├── new.go             # Create tracks
│   ├── list.go            # List tracks
│   ├── jump.go            # Switch to track
│   ├── delete.go          # Delete tracks
│   ├── sync.go            # Sync with remote
│   ├── ai.go              # AI assistant
│   └── remote.go          # Browse remote branches
├── internal/
│   ├── config/            # YAML configuration
│   │   └── config.go
│   ├── db/                # SQLite persistence
│   │   └── db.go
│   ├── ops/               # Business logic
│   │   └── ops.go
│   ├── track/             # Track utilities
│   │   ├── types.go       # Status structs
│   │   └── slug.go        # Name normalization
│   ├── git/               # Git CLI wrapper
│   │   └── git.go
│   ├── github/            # GitHub CLI wrapper
│   │   └── github.go
│   ├── tmux/              # Tmux CLI wrapper
│   │   └── tmux.go
│   ├── devbox/            # Devbox CLI wrapper
│   │   └── devbox.go
│   └── tui/               # Bubble Tea TUI
│       └── tui.go
├── Makefile
├── go.mod
└── go.sum
```

## Core Concepts

### Track

A track is the fundamental unit of work - an isolated dev environment for a branch:

```go
// internal/db/db.go
type Track struct {
    ID           int64
    Branch       string    // e.g., "feature/user-auth"
    RemoteURL    string    // e.g., "origin"
    HeadSHA      string    // Current commit SHA
    Type         string    // "worktree" or "devbox"
    Path         string    // Worktree path (worktree only)
    DevboxName   string    // Devbox name (devbox only)
    CreatedAt    time.Time
    LastAccessed time.Time
}
```

### Track Types

Currently supported:
- **worktree**: Local git worktree at `~/worktrees/<slug>/`
- **devbox**: Remote Kubernetes dev environment

### Status

Live status is fetched on-demand, not persisted:

```go
// internal/track/types.go
type TrackStatus struct {
    Git    GitStatus    // clean/dirty, ahead/behind
    PR     PRStatus     // PR number, state, draft
    CI     CIStatus     // passing/pending/failing
    Review ReviewStatus // approved/changes_requested/pending
}
```

## Adding a New Track Type

To add a new way to track units of work (e.g., Docker containers, cloud VMs, Codespaces):

### Step 1: Create the Integration Module

Create a new package under `internal/` for your track type:

```go
// internal/mytype/mytype.go
package mytype

type Client struct {
    // Configuration fields
}

func New() *Client {
    return &Client{}
}

// Create creates a new environment
func (c *Client) Create(name string) error {
    // Implementation
}

// Delete removes an environment
func (c *Client) Delete(name string) error {
    // Implementation
}

// Exists checks if environment exists
func (c *Client) Exists(name string) (bool, error) {
    // Implementation
}

// GetConnectionCommand returns the command to connect
func (c *Client) GetConnectionCommand(name string) (string, error) {
    // Implementation - e.g., SSH command, docker exec, etc.
}
```

### Step 2: Update the Database Schema

Add any new fields needed to the tracks table in `internal/db/db.go`:

```go
// Update the schema
const schema = `
CREATE TABLE IF NOT EXISTS tracks (
    id INTEGER PRIMARY KEY,
    branch TEXT NOT NULL,
    remote_url TEXT NOT NULL,
    head_sha TEXT NOT NULL,
    type TEXT NOT NULL CHECK (type IN ('worktree', 'devbox', 'mytype')),  -- Add new type
    path TEXT,
    devbox_name TEXT,
    mytype_id TEXT,  -- Add new field for your type
    created_at TIMESTAMP,
    last_accessed TIMESTAMP,
    UNIQUE(remote_url, branch)
)
`

// Update the Track struct
type Track struct {
    // ... existing fields ...
    MyTypeID string // New field
}
```

### Step 3: Add Operations

Add methods to `internal/ops/ops.go`:

```go
// NewTrackMyType creates a new mytype track
func (o *Ops) NewTrackMyType(branch string) (*db.Track, error) {
    // 1. Generate a name for the environment
    slug := track.Slugify(branch)

    // 2. Create the environment using your module
    myClient := mytype.New()
    if err := myClient.Create(slug); err != nil {
        return nil, fmt.Errorf("failed to create mytype: %w", err)
    }

    // 3. Record in database
    tr := &db.Track{
        Branch:    branch,
        RemoteURL: o.cfg.Repo.Remote,
        HeadSHA:   "",  // or fetch from git
        Type:      "mytype",
        MyTypeID:  slug,
        CreatedAt: time.Now(),
    }

    if err := o.db.InsertTrack(tr); err != nil {
        // Cleanup on failure
        myClient.Delete(slug)
        return nil, err
    }

    return tr, nil
}
```

Update existing operations to handle your type:

```go
// In DeleteTrack
func (o *Ops) DeleteTrack(branch string, deleteRemote bool) error {
    tr, err := o.db.GetTrack(o.cfg.Repo.Remote, branch)
    if err != nil {
        return err
    }

    switch tr.Type {
    case "worktree":
        // existing worktree cleanup
    case "devbox":
        // existing devbox cleanup
    case "mytype":
        myClient := mytype.New()
        if err := myClient.Delete(tr.MyTypeID); err != nil {
            return err
        }
    }

    return o.db.DeleteTrack(o.cfg.Repo.Remote, branch)
}

// In JumpToTrack
func (o *Ops) JumpToTrack(branch string) error {
    tr, err := o.db.GetTrack(o.cfg.Repo.Remote, branch)
    if err != nil {
        return err
    }

    var cmd string
    switch tr.Type {
    case "worktree":
        cmd = fmt.Sprintf("cd %s", tr.Path)
    case "devbox":
        cmd, _ = devbox.New().GetSSHCommand(tr.DevboxName)
    case "mytype":
        cmd, _ = mytype.New().GetConnectionCommand(tr.MyTypeID)
    }

    // Create/switch tmux window with command
    // ...
}
```

### Step 4: Update the CLI

Add your type to the `new` command in `cmd/trak/new.go`:

```go
func init() {
    newCmd.Flags().BoolP("worktree", "w", false, "Create a worktree track")
    newCmd.Flags().BoolP("devbox", "d", false, "Create a devbox track")
    newCmd.Flags().BoolP("mytype", "m", false, "Create a mytype track")  // Add flag
}

var newCmd = &cobra.Command{
    // ...
    RunE: func(cmd *cobra.Command, args []string) error {
        // ... flag parsing ...

        if mytype, _ := cmd.Flags().GetBool("mytype"); mytype {
            _, err := ops.NewTrackMyType(branch)
            return err
        }

        // ... existing logic ...
    },
}
```

### Step 5: Update the TUI

In `internal/tui/tui.go`, handle your type in the display and interaction:

```go
// In renderTrackRow
func (m Model) renderTrackRow(t ops.TrackWithStatus) string {
    typeIcon := ""
    switch t.Track.Type {
    case "worktree":
        typeIcon = "W"
    case "devbox":
        typeIcon = "D"
    case "mytype":
        typeIcon = "M"  // Add icon
    }
    // ...
}
```

### Step 6: Add Tests

Create tests for your module:

```go
// internal/mytype/mytype_test.go
package mytype

import "testing"

func TestCreate(t *testing.T) {
    // Test environment creation
}

func TestDelete(t *testing.T) {
    // Test environment deletion
}
```

## Key Integration Points

### Git Integration (`internal/git/git.go`)

All git operations shell out to the git CLI:

```go
// Key functions you might use:
git.Fetch(repoPath)                      // Fetch from remote
git.GetDefaultBranch(repoPath)           // Get main/master
git.CreateBranch(repoPath, branch, base) // Create branch
git.GetHeadSHA(repoPath)                 // Get current SHA
git.IsDirty(repoPath)                    // Check for changes
git.AheadBehind(repoPath, branch, base)  // Compare branches
```

### GitHub Integration (`internal/github/github.go`)

Uses the `gh` CLI for GitHub operations:

```go
// Key functions:
github.GetPRForBranch(remote, branch)  // Get PR info
github.CreatePR(repoPath, branch, ...)  // Create PR
github.ListMyBranches(remote)           // List user's branches
```

### Tmux Integration (`internal/tmux/tmux.go`)

Manages tmux sessions/windows:

```go
// Key functions:
tmux.SessionExists(name)                        // Check session
tmux.CreateSession(name)                        // New session
tmux.WindowExists(session, window)              // Check window
tmux.CreateWindow(session, window, dir, cmd)    // New window
tmux.SwitchToWindow(session, window)            // Switch focus
```

### Config (`internal/config/config.go`)

Loaded from `~/.config/trak/config.yaml`:

```go
type Config struct {
    Repo struct {
        Path   string // Main repo path
        Remote string // GitHub remote (owner/repo)
    }
    Cache struct {
        DefaultBranch        string
        DefaultBranchUpdated time.Time
    }
}
```

## Development Workflow

```bash
# Initial setup
make deps

# Run tests
make test

# Build and install
make dev

# Run the TUI
trak

# Test a specific package
make test-pkg PKG=./internal/db

# Format and lint
make check
```

## Adding a New Command

1. Create `cmd/trak/mycommand.go`:

```go
package main

import (
    "github.com/spf13/cobra"
)

var myCmd = &cobra.Command{
    Use:   "mycommand <branch>",
    Short: "Do something with a track",
    Args:  cobra.ExactArgs(1),
    RunE: func(cmd *cobra.Command, args []string) error {
        branch := args[0]
        // Use global `ops` instance from root.go
        return ops.MyOperation(branch)
    },
}

func init() {
    rootCmd.AddCommand(myCmd)
}
```

2. Add the operation to `internal/ops/ops.go`
3. Add tests

## Database Migrations

Currently, trak uses a simple schema creation on startup. For schema changes:

1. Update the `schema` constant in `internal/db/db.go`
2. For existing databases, either:
   - Delete `~/.config/trak/trak.db` (loses data)
   - Add migration logic in `New()` function

## Testing Guidelines

- Each package has a `*_test.go` file
- Use table-driven tests
- Mock external commands (git, gh, tmux) when needed
- Tests timeout after 30s (enforced by Makefile)

```go
func TestSomething(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    string
        wantErr bool
    }{
        {"basic", "input", "expected", false},
        {"error case", "bad", "", true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := DoSomething(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if got != tt.want {
                t.Errorf("got %v, want %v", got, tt.want)
            }
        })
    }
}
```

## Status Display Reference

The list view shows these status indicators:

| Column | Values | Meaning |
|--------|--------|---------|
| TYPE | W, D | Worktree, Devbox |
| GIT | clean, dirty, ↑N, ↓N | Clean, uncommitted changes, ahead, behind |
| PR | #123, — | PR number or none |
| CI | ✓, ○, ✗, — | Passing, pending, failing, none |
| REVIEW | ✓, ○, ✗, — | Approved, pending, changes requested, none |

## Configuration Reference

`~/.config/trak/config.yaml`:

```yaml
repo:
  path: /path/to/main/repo
  remote: owner/repo
cache:
  default_branch: main
  default_branch_updated: 2024-01-05T00:00:00Z
```

## Troubleshooting

### Database Issues

```bash
# Reset database
rm ~/.config/trak/trak.db
trak  # Will recreate
```

### Stale Worktrees

```bash
# Git's worktree list can get out of sync
cd /path/to/main/repo
git worktree prune
```

### Tmux Session Issues

```bash
# List all trak windows
tmux list-windows -t trak

# Kill stuck session
tmux kill-session -t trak
```
