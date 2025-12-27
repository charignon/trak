// Package git provides functions to interact with git via shell commands.
// All operations shell out to git CLI rather than using go-git for complex operations.
package git

import (
	"bytes"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// runGit executes a git command in the specified directory and returns output.
func runGit(repoPath string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = repoPath

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("git %s failed: %w\nstderr: %s", strings.Join(args, " "), err, stderr.String())
	}

	return strings.TrimSpace(stdout.String()), nil
}

// Fetch fetches from origin.
func Fetch(repoPath string) error {
	_, err := runGit(repoPath, "fetch", "origin")
	return err
}

// GetDefaultBranch detects the default branch dynamically.
// It first tries to get it from the remote HEAD ref, falling back to checking
// common branch names if that fails.
func GetDefaultBranch(repoPath string) (string, error) {
	// Try to get from symbolic-ref of origin/HEAD
	output, err := runGit(repoPath, "symbolic-ref", "refs/remotes/origin/HEAD")
	if err == nil {
		// Output is like "refs/remotes/origin/main"
		parts := strings.Split(output, "/")
		if len(parts) > 0 {
			return parts[len(parts)-1], nil
		}
	}

	// Fallback: check if origin/main or origin/master exists
	_, err = runGit(repoPath, "rev-parse", "--verify", "origin/main")
	if err == nil {
		return "main", nil
	}

	_, err = runGit(repoPath, "rev-parse", "--verify", "origin/master")
	if err == nil {
		return "master", nil
	}

	return "", fmt.Errorf("could not determine default branch")
}

// CreateBranch creates a new branch from a base branch.
func CreateBranch(repoPath, branchName, baseBranch string) error {
	_, err := runGit(repoPath, "branch", branchName, baseBranch)
	return err
}

// WorktreeAdd adds a new worktree at the specified path for the given branch.
func WorktreeAdd(repoPath, worktreePath, branch string) error {
	_, err := runGit(repoPath, "worktree", "add", worktreePath, branch)
	return err
}

// WorktreeRemove removes a worktree at the specified path.
// repoPath is the main repository path (not the worktree itself).
func WorktreeRemove(repoPath, worktreePath string) error {
	_, err := runGit(repoPath, "worktree", "remove", worktreePath)
	return err
}

// WorktreePrune prunes stale worktree references.
func WorktreePrune(repoPath string) error {
	_, err := runGit(repoPath, "worktree", "prune")
	return err
}

// Rebase rebases current branch onto another branch.
// Returns true if there are conflicts, false otherwise.
func Rebase(repoPath, ontoBranch string) (conflicts bool, err error) {
	_, err = runGit(repoPath, "rebase", ontoBranch)
	if err != nil {
		// Check if it's a conflict situation
		if strings.Contains(err.Error(), "CONFLICT") || strings.Contains(err.Error(), "conflict") {
			return true, nil
		}
		// Check rebase status
		_, statusErr := runGit(repoPath, "rebase", "--show-current-patch")
		if statusErr == nil {
			// We're in a rebase with conflicts
			return true, nil
		}
		return false, err
	}
	return false, nil
}

// RebaseAbort aborts an in-progress rebase.
func RebaseAbort(repoPath string) error {
	_, err := runGit(repoPath, "rebase", "--abort")
	return err
}

// RebaseContinue continues a rebase after conflicts are resolved.
func RebaseContinue(repoPath string) error {
	_, err := runGit(repoPath, "rebase", "--continue")
	return err
}

// Push pushes a branch to origin.
func Push(repoPath, branch string) error {
	_, err := runGit(repoPath, "push", "origin", branch)
	return err
}

// PushForce force-pushes a branch to origin (needed after rebase).
func PushForce(repoPath, branch string) error {
	_, err := runGit(repoPath, "push", "--force-with-lease", "origin", branch)
	return err
}

// PushDelete deletes a remote branch.
func PushDelete(repoPath, branch string) error {
	_, err := runGit(repoPath, "push", "origin", "--delete", branch)
	return err
}

// AheadBehind returns how many commits ahead and behind a branch is from base.
// Uses git rev-list --left-right --count.
func AheadBehind(repoPath, branch, baseBranch string) (ahead, behind int, err error) {
	output, err := runGit(repoPath, "rev-list", "--left-right", "--count", fmt.Sprintf("%s...%s", branch, baseBranch))
	if err != nil {
		return 0, 0, err
	}

	parts := strings.Fields(output)
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("unexpected rev-list output: %s", output)
	}

	ahead, err = strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, fmt.Errorf("failed to parse ahead count: %w", err)
	}

	behind, err = strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, fmt.Errorf("failed to parse behind count: %w", err)
	}

	return ahead, behind, nil
}

// GetHeadSHA returns the SHA of HEAD.
func GetHeadSHA(repoPath string) (string, error) {
	return runGit(repoPath, "rev-parse", "HEAD")
}

// GetBranchSHA returns the SHA of a specific branch.
func GetBranchSHA(repoPath, branch string) (string, error) {
	return runGit(repoPath, "rev-parse", branch)
}

// IsDirty returns true if there are uncommitted changes.
func IsDirty(repoPath string) (bool, error) {
	output, err := runGit(repoPath, "status", "--porcelain")
	if err != nil {
		return false, err
	}
	return len(output) > 0, nil
}

// Checkout checks out a branch.
func Checkout(repoPath, branch string) error {
	_, err := runGit(repoPath, "checkout", branch)
	return err
}

// Pull pulls from origin for a specific branch with rebase.
func Pull(repoPath, branch string) error {
	_, err := runGit(repoPath, "pull", "--rebase", "origin", branch)
	return err
}

// GetCurrentBranch returns the name of the current branch.
func GetCurrentBranch(repoPath string) (string, error) {
	return runGit(repoPath, "rev-parse", "--abbrev-ref", "HEAD")
}
