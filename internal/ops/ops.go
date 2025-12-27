// Package ops provides high-level orchestration operations for trak.
// It coordinates between git, github, db, tmux, devbox, and config modules.
package ops

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/laurent/trak/internal/config"
	"github.com/laurent/trak/internal/db"
	"github.com/laurent/trak/internal/devbox"
	"github.com/laurent/trak/internal/git"
	"github.com/laurent/trak/internal/github"
	"github.com/laurent/trak/internal/tmux"
	"github.com/laurent/trak/internal/track"
)

// Ops provides high-level operations for managing tracks.
type Ops struct {
	db     *db.DB
	config *config.Config
}

// TrackWithStatus combines a track from the database with its live status.
type TrackWithStatus struct {
	Track  db.Track
	Status track.TrackStatus
}

// RemoteBranch represents a remote branch with metadata.
type RemoteBranch struct {
	Name       string
	LastCommit string
	Age        time.Duration
	HasPR      bool
	PRNumber   int
}

// New creates a new Ops instance with the given database and config.
func New(database *db.DB, cfg *config.Config) *Ops {
	return &Ops{
		db:     database,
		config: cfg,
	}
}

// NewTrackWorktree creates a new worktree-based track for the given branch.
// It creates the branch if it doesn't exist, adds a git worktree, and records it in the database.
func (o *Ops) NewTrackWorktree(branch string) error {
	repoPath := o.config.Repo.Path
	remote := o.config.Repo.Remote

	// Fetch latest from remote
	if err := git.Fetch(repoPath); err != nil {
		return fmt.Errorf("failed to fetch from remote: %w", err)
	}

	// Get the default branch to base new branch on
	defaultBranch, err := git.GetDefaultBranch(repoPath)
	if err != nil {
		return fmt.Errorf("failed to get default branch: %w", err)
	}

	// Check if branch already exists locally
	_, err = git.GetBranchSHA(repoPath, branch)
	branchExists := err == nil

	// Check if branch exists on remote
	_, remoteErr := git.GetBranchSHA(repoPath, "origin/"+branch)
	remoteBranchExists := remoteErr == nil

	if !branchExists {
		if remoteBranchExists {
			// Create local branch tracking remote
			if err := git.CreateBranch(repoPath, branch, "origin/"+branch); err != nil {
				return fmt.Errorf("failed to create branch from remote: %w", err)
			}
		} else {
			// Create new branch from default branch
			if err := git.CreateBranch(repoPath, branch, "origin/"+defaultBranch); err != nil {
				return fmt.Errorf("failed to create branch: %w", err)
			}
		}
	}

	// Get the SHA for the worktree path generation
	sha, err := git.GetBranchSHA(repoPath, branch)
	if err != nil {
		return fmt.Errorf("failed to get branch SHA: %w", err)
	}

	// Generate worktree path
	slug := track.GenerateSlug(branch, sha)
	worktreeBase := config.GetWorktreeBaseDir()
	worktreePath := filepath.Join(worktreeBase, slug)

	// Ensure worktree base directory exists
	if err := os.MkdirAll(worktreeBase, 0755); err != nil {
		return fmt.Errorf("failed to create worktree base directory: %w", err)
	}

	// Add the worktree
	if err := git.WorktreeAdd(repoPath, worktreePath, branch); err != nil {
		return fmt.Errorf("failed to add worktree: %w", err)
	}

	// Record in database
	now := time.Now()
	trackRecord := db.Track{
		Branch:       branch,
		RemoteURL:    remote,
		HeadSHA:      sha,
		Type:         db.TrackTypeWorktree,
		Path:         &worktreePath,
		CreatedAt:    now,
		LastAccessed: &now,
	}

	if err := o.db.InsertTrack(trackRecord); err != nil {
		// If DB insert fails, try to clean up the worktree
		_ = git.WorktreeRemove(repoPath, worktreePath)
		return fmt.Errorf("failed to record track in database: %w", err)
	}

	return nil
}

// NewTrackDevbox creates a new devbox-based track for the given branch.
// It creates a remote k8s dev environment and records it in the database.
func (o *Ops) NewTrackDevbox(branch string) error {
	remote := o.config.Repo.Remote
	repoPath := o.config.Repo.Path

	// Generate a unique devbox name from the branch
	devboxName := track.Slugify(branch)

	// Check if devbox already exists
	exists, err := devbox.Exists(devboxName)
	if err != nil {
		return fmt.Errorf("failed to check if devbox exists: %w", err)
	}
	if exists {
		return fmt.Errorf("devbox %s already exists", devboxName)
	}

	// Create the devbox
	// Note: remote format for devbox might need to be full URL
	repoURL := fmt.Sprintf("https://github.com/%s.git", remote)
	if err := devbox.Create(devboxName, repoURL, branch); err != nil {
		return fmt.Errorf("failed to create devbox: %w", err)
	}

	// Get current SHA for tracking
	// First try to get from remote
	if err := git.Fetch(repoPath); err != nil {
		// Non-fatal, continue with empty SHA
	}

	sha, _ := git.GetBranchSHA(repoPath, "origin/"+branch)
	if sha == "" {
		sha = "unknown"
	}

	// Record in database
	now := time.Now()
	trackRecord := db.Track{
		Branch:       branch,
		RemoteURL:    remote,
		HeadSHA:      sha,
		Type:         db.TrackTypeDevbox,
		DevboxName:   &devboxName,
		CreatedAt:    now,
		LastAccessed: &now,
	}

	if err := o.db.InsertTrack(trackRecord); err != nil {
		// If DB insert fails, try to clean up the devbox
		_ = devbox.Delete(devboxName)
		return fmt.Errorf("failed to record track in database: %w", err)
	}

	return nil
}

// DeleteTrack deletes a track (worktree or devbox) and optionally the remote branch.
func (o *Ops) DeleteTrack(branch string, deleteRemote bool) error {
	remote := o.config.Repo.Remote
	repoPath := o.config.Repo.Path

	// Get the track from database
	trk, err := o.db.GetTrack(remote, branch)
	if err != nil {
		return fmt.Errorf("failed to get track: %w", err)
	}
	if trk == nil {
		return fmt.Errorf("track not found for branch: %s", branch)
	}

	// Delete the local environment based on type
	switch trk.Type {
	case db.TrackTypeWorktree:
		if trk.Path != nil {
			if err := git.WorktreeRemove(repoPath, *trk.Path); err != nil {
				return fmt.Errorf("failed to remove worktree: %w", err)
			}
			// Prune stale worktree references
			_ = git.WorktreePrune(repoPath)
		}

	case db.TrackTypeDevbox:
		if trk.DevboxName != nil {
			if err := devbox.Delete(*trk.DevboxName); err != nil {
				return fmt.Errorf("failed to delete devbox: %w", err)
			}
		}
	}

	// Optionally delete remote branch
	if deleteRemote {
		if err := git.PushDelete(repoPath, branch); err != nil {
			// Log but don't fail if remote delete fails
			// The branch might not exist on remote
			fmt.Fprintf(os.Stderr, "warning: failed to delete remote branch: %v\n", err)
		}
	}

	// Remove from database
	if err := o.db.DeleteTrack(remote, branch); err != nil {
		return fmt.Errorf("failed to remove track from database: %w", err)
	}

	return nil
}

// SyncResult contains the result of a sync operation.
type SyncResult struct {
	Rebased       bool
	Pushed        bool
	PRCreated     bool
	PRNumber      int
	HasConflicts  bool
	ConflictsPath string // Path to worktree with conflicts
}

// SyncTrack syncs a track: pulls default branch, rebases, pushes, creates PR if needed.
// Returns a SyncResult and an error. If there are rebase conflicts, HasConflicts will be true.
func (o *Ops) SyncTrack(branch string) (*SyncResult, error) {
	remote := o.config.Repo.Remote

	// Get the track from database
	trk, err := o.db.GetTrack(remote, branch)
	if err != nil {
		return nil, fmt.Errorf("failed to get track: %w", err)
	}
	if trk == nil {
		return nil, fmt.Errorf("track not found for branch: %s", branch)
	}

	// Determine the working directory
	var workDir string
	switch trk.Type {
	case db.TrackTypeWorktree:
		if trk.Path == nil {
			return nil, fmt.Errorf("worktree track has no path")
		}
		workDir = *trk.Path
	case db.TrackTypeDevbox:
		return nil, fmt.Errorf("sync is not supported for devbox tracks (use devbox CLI directly)")
	}

	result := &SyncResult{}

	// Fetch latest from remote
	if err := git.Fetch(workDir); err != nil {
		return nil, fmt.Errorf("failed to fetch: %w", err)
	}

	// Get the default branch
	defaultBranch, err := git.GetDefaultBranch(workDir)
	if err != nil {
		return nil, fmt.Errorf("failed to get default branch: %w", err)
	}

	// Rebase onto default branch
	conflicts, err := git.Rebase(workDir, "origin/"+defaultBranch)
	if err != nil {
		return nil, fmt.Errorf("rebase failed: %w", err)
	}

	if conflicts {
		result.HasConflicts = true
		result.ConflictsPath = workDir
		return result, nil
	}

	result.Rebased = true

	// Push to remote (force after rebase)
	if err := git.PushForce(workDir, branch); err != nil {
		return nil, fmt.Errorf("failed to push: %w", err)
	}
	result.Pushed = true

	// Update HEAD SHA in database
	newSHA, err := git.GetHeadSHA(workDir)
	if err == nil {
		_ = o.db.UpdateHeadSHA(remote, branch, newSHA)
	}

	// Check if PR exists, create if not
	pr, err := github.GetPRForBranch(remote, branch)
	if err != nil {
		// Non-fatal, just skip PR creation
		return result, nil
	}

	if pr == nil {
		// Create PR
		title := branch // Use branch name as default title
		prNum, err := github.CreatePR(remote, branch, defaultBranch, title)
		if err != nil {
			// Non-fatal, just skip PR creation
			return result, nil
		}
		result.PRCreated = true
		result.PRNumber = prNum
	}

	return result, nil
}

// JumpToTrack switches to the tmux window for the given track, creating it if needed.
func (o *Ops) JumpToTrack(branch string) error {
	remote := o.config.Repo.Remote

	// Get the track from database
	trk, err := o.db.GetTrack(remote, branch)
	if err != nil {
		return fmt.Errorf("failed to get track: %w", err)
	}
	if trk == nil {
		return fmt.Errorf("track not found for branch: %s", branch)
	}

	// Update last accessed time
	_ = o.db.UpdateLastAccessed(remote, branch)

	// Determine tmux session name (use repo name or a default)
	sessionName := "trak"

	// Sanitize branch name for tmux
	windowName := track.SanitizeForTmux(branch)

	// Ensure session exists
	sessionExists, err := tmux.SessionExists(sessionName)
	if err != nil {
		return fmt.Errorf("failed to check session: %w", err)
	}

	if !sessionExists {
		if err := tmux.CreateSession(sessionName); err != nil {
			return fmt.Errorf("failed to create session: %w", err)
		}
	}

	// Check if window exists
	windowExists, err := tmux.WindowExists(sessionName, windowName)
	if err != nil {
		return fmt.Errorf("failed to check window: %w", err)
	}

	if !windowExists {
		// Create window with appropriate start directory
		var startDir string
		switch trk.Type {
		case db.TrackTypeWorktree:
			if trk.Path != nil {
				startDir = *trk.Path
			}
		case db.TrackTypeDevbox:
			// For devbox, we'll create window and then run SSH command
			startDir = ""
		}

		if err := tmux.CreateWindow(sessionName, windowName, startDir); err != nil {
			return fmt.Errorf("failed to create window: %w", err)
		}

		// For devbox, send SSH command to the window
		if trk.Type == db.TrackTypeDevbox && trk.DevboxName != nil {
			sshCmd, err := devbox.GetSSHCommand(*trk.DevboxName)
			if err == nil && sshCmd != "" {
				_ = tmux.RunInWindow(sessionName, windowName, sshCmd)
			}
		}
	}

	// Switch to the window
	if tmux.IsInsideTmux() {
		return tmux.SwitchToWindow(sessionName, windowName)
	} else {
		// Select the window first, then attach
		_ = tmux.SelectWindow(sessionName, windowName)
		return tmux.AttachSession(sessionName)
	}
}

// RunAI runs the AI assistant (toad) in the context of the given track.
func (o *Ops) RunAI(branch string) error {
	remote := o.config.Repo.Remote

	// Get the track from database
	trk, err := o.db.GetTrack(remote, branch)
	if err != nil {
		return fmt.Errorf("failed to get track: %w", err)
	}
	if trk == nil {
		return fmt.Errorf("track not found for branch: %s", branch)
	}

	// Determine the path for toad
	var toadPath string
	switch trk.Type {
	case db.TrackTypeWorktree:
		if trk.Path == nil {
			return fmt.Errorf("worktree track has no path")
		}
		toadPath = *trk.Path
	case db.TrackTypeDevbox:
		return fmt.Errorf("AI assistant is not supported for devbox tracks")
	}

	// Update last accessed time
	_ = o.db.UpdateLastAccessed(remote, branch)

	// Run tc (toad claude) command
	cmd := fmt.Sprintf("tc %s", toadPath)

	// Get tmux context
	sessionName := "trak"
	windowName := track.SanitizeForTmux(branch)

	// Check if we need to create window
	sessionExists, _ := tmux.SessionExists(sessionName)
	if !sessionExists {
		if err := tmux.CreateSession(sessionName); err != nil {
			return fmt.Errorf("failed to create session: %w", err)
		}
	}

	windowExists, _ := tmux.WindowExists(sessionName, windowName)
	if !windowExists {
		if err := tmux.CreateWindow(sessionName, windowName, toadPath); err != nil {
			return fmt.Errorf("failed to create window: %w", err)
		}
	}

	// Send the toad command to the window
	if err := tmux.RunInWindow(sessionName, windowName, cmd); err != nil {
		return fmt.Errorf("failed to run AI command: %w", err)
	}

	// Switch to the window
	if tmux.IsInsideTmux() {
		return tmux.SwitchToWindow(sessionName, windowName)
	} else {
		_ = tmux.SelectWindow(sessionName, windowName)
		return tmux.AttachSession(sessionName)
	}
}

// RefreshTrackStatus fetches the current status of a track from git and GitHub.
func (o *Ops) RefreshTrackStatus(trk db.Track) (track.TrackStatus, error) {
	status := track.TrackStatus{}
	repoPath := o.config.Repo.Path
	remote := o.config.Repo.Remote

	// Determine working directory
	var workDir string
	switch trk.Type {
	case db.TrackTypeWorktree:
		if trk.Path != nil {
			workDir = *trk.Path
		} else {
			workDir = repoPath
		}
	case db.TrackTypeDevbox:
		// For devbox, we can only check GitHub status, not local git status
		workDir = ""
	}

	// Get git status if we have a working directory
	if workDir != "" {
		// Check if dirty
		dirty, err := git.IsDirty(workDir)
		if err == nil {
			status.GitStatus.Clean = !dirty
		}

		// Get default branch for ahead/behind
		defaultBranch, err := git.GetDefaultBranch(workDir)
		if err == nil {
			ahead, behind, err := git.AheadBehind(workDir, trk.Branch, "origin/"+defaultBranch)
			if err == nil {
				status.GitStatus.Ahead = ahead > 0
				status.GitStatus.Behind = behind > 0
				status.GitStatus.AheadCount = ahead
				status.GitStatus.BehindCount = behind
			}
		}

		// Check for SHA mismatch (force-push detection)
		currentSHA, err := git.GetHeadSHA(workDir)
		if err == nil && trk.HeadSHA != "" && trk.HeadSHA != "unknown" {
			if currentSHA != trk.HeadSHA {
				status.SHAMismatch = true
				// Update the SHA in database
				_ = o.db.UpdateHeadSHA(remote, trk.Branch, currentSHA)
			}
		}
	}

	// Get PR status from GitHub
	pr, err := github.GetPRForBranch(remote, trk.Branch)
	if err == nil && pr != nil {
		status.PR = &track.PRStatus{
			Number: pr.Number,
			URL:    fmt.Sprintf("https://github.com/%s/pull/%d", remote, pr.Number),
			State:  pr.State,
			Draft:  false, // gh CLI doesn't expose draft status in our current implementation
		}

		// Map CI status
		status.CI = &track.CIStatus{
			Passing: pr.CIStatus == "success",
			Pending: pr.CIStatus == "pending",
			Failing: pr.CIStatus == "failure",
		}

		// Map review status
		status.Review = &track.ReviewStatus{
			Approved:         pr.ReviewStatus == "approved",
			ChangesRequested: pr.ReviewStatus == "changes_requested",
			Pending:          pr.ReviewStatus == "pending",
		}
	}

	// Check if stale (not accessed in over 7 days)
	if trk.LastAccessed != nil {
		staleDuration := 7 * 24 * time.Hour
		if time.Since(*trk.LastAccessed) > staleDuration {
			status.IsStale = true
		}
	}

	return status, nil
}

// ListTracksWithStatus returns all tracks with their current status.
func (o *Ops) ListTracksWithStatus() ([]TrackWithStatus, error) {
	tracks, err := o.db.ListTracks()
	if err != nil {
		return nil, fmt.Errorf("failed to list tracks: %w", err)
	}

	result := make([]TrackWithStatus, 0, len(tracks))
	for _, trk := range tracks {
		status, err := o.RefreshTrackStatus(trk)
		if err != nil {
			// Use empty status on error, don't fail the whole list
			status = track.TrackStatus{}
		}
		result = append(result, TrackWithStatus{
			Track:  trk,
			Status: status,
		})
	}

	return result, nil
}

// ListRemoteBranches returns remote branches from GitHub that the user has PRs for.
func (o *Ops) ListRemoteBranches() ([]RemoteBranch, error) {
	remote := o.config.Repo.Remote

	// Get branches with open PRs
	ghBranches, err := github.ListMyBranches(remote)
	if err != nil {
		return nil, fmt.Errorf("failed to list remote branches: %w", err)
	}

	// Get all PRs for enrichment
	prs, err := github.ListMyPRs(remote)
	if err != nil {
		prs = []github.PR{} // Non-fatal, continue without PR info
	}

	// Build a map of branch -> PR
	prMap := make(map[string]github.PR)
	for _, pr := range prs {
		prMap[pr.Branch] = pr
	}

	result := make([]RemoteBranch, 0, len(ghBranches))
	for _, b := range ghBranches {
		rb := RemoteBranch{
			Name:       b.Name,
			LastCommit: b.LastCommit,
			Age:        b.Age,
		}

		if pr, ok := prMap[b.Name]; ok {
			rb.HasPR = true
			rb.PRNumber = pr.Number
		}

		result = append(result, rb)
	}

	return result, nil
}
