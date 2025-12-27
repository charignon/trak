// Package github provides functions to interact with GitHub via the gh CLI.
// All operations shell out to gh CLI for authentication and API access.
package github

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// PR represents a GitHub pull request.
type PR struct {
	Number       int
	Branch       string
	State        string // "open", "closed", "merged"
	CIStatus     string // "success", "failure", "pending", "unknown"
	ReviewStatus string // "approved", "changes_requested", "pending", "unknown"
}

// RemoteBranch represents a remote branch with metadata.
type RemoteBranch struct {
	Name       string
	LastCommit string
	Age        time.Duration
}

// CommandRunner is an interface for running commands, allowing for mocking in tests.
type CommandRunner interface {
	Run(name string, args ...string) (string, error)
}

// DefaultRunner executes commands using os/exec.
type DefaultRunner struct{}

// Run executes a command and returns its output.
func (r DefaultRunner) Run(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("%s %s failed: %w\nstderr: %s", name, strings.Join(args, " "), err, stderr.String())
	}

	return strings.TrimSpace(stdout.String()), nil
}

// runner is the command runner used by this package.
// It can be replaced with a mock for testing.
var runner CommandRunner = DefaultRunner{}

// SetRunner sets the command runner (used for testing).
func SetRunner(r CommandRunner) {
	runner = r
}

// ResetRunner resets the command runner to the default.
func ResetRunner() {
	runner = DefaultRunner{}
}

// runGH executes a gh command and returns output.
func runGH(args ...string) (string, error) {
	return runner.Run("gh", args...)
}

// GetDefaultBranch returns the default branch for a repository.
func GetDefaultBranch(remote string) (string, error) {
	output, err := runGH("repo", "view", remote, "--json", "defaultBranchRef", "--jq", ".defaultBranchRef.name")
	if err != nil {
		return "", fmt.Errorf("failed to get default branch: %w", err)
	}
	if output == "" {
		return "", fmt.Errorf("no default branch found for %s", remote)
	}
	return output, nil
}

// GetCurrentUser returns the authenticated GitHub username.
func GetCurrentUser() (string, error) {
	output, err := runGH("api", "user", "--jq", ".login")
	if err != nil {
		return "", fmt.Errorf("failed to get current user: %w", err)
	}
	if output == "" {
		return "", fmt.Errorf("no user logged in")
	}
	return output, nil
}

// ghPR is the JSON structure returned by gh pr list.
type ghPR struct {
	Number      int    `json:"number"`
	HeadRefName string `json:"headRefName"`
	State       string `json:"state"`
	StatusRollup string `json:"statusCheckRollup"`
	ReviewDecision string `json:"reviewDecision"`
}

// ListMyPRs returns all open PRs authored by the current user for a repository.
func ListMyPRs(remote string) ([]PR, error) {
	user, err := GetCurrentUser()
	if err != nil {
		return nil, err
	}

	output, err := runGH("pr", "list",
		"--repo", remote,
		"--author", user,
		"--state", "open",
		"--json", "number,headRefName,state,statusCheckRollup,reviewDecision",
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list PRs: %w", err)
	}

	if output == "" || output == "[]" {
		return []PR{}, nil
	}

	var ghPRs []ghPR
	if err := json.Unmarshal([]byte(output), &ghPRs); err != nil {
		return nil, fmt.Errorf("failed to parse PR list: %w", err)
	}

	prs := make([]PR, len(ghPRs))
	for i, p := range ghPRs {
		prs[i] = PR{
			Number:       p.Number,
			Branch:       p.HeadRefName,
			State:        strings.ToLower(p.State),
			CIStatus:     normalizeCIStatus(p.StatusRollup),
			ReviewStatus: normalizeReviewStatus(p.ReviewDecision),
		}
	}

	return prs, nil
}

// normalizeCIStatus converts gh CLI status to our standard values.
func normalizeCIStatus(status string) string {
	switch strings.ToUpper(status) {
	case "SUCCESS":
		return "success"
	case "FAILURE", "ERROR":
		return "failure"
	case "PENDING", "EXPECTED", "QUEUED", "IN_PROGRESS":
		return "pending"
	default:
		return "unknown"
	}
}

// normalizeReviewStatus converts gh CLI review decision to our standard values.
func normalizeReviewStatus(decision string) string {
	switch strings.ToUpper(decision) {
	case "APPROVED":
		return "approved"
	case "CHANGES_REQUESTED":
		return "changes_requested"
	case "REVIEW_REQUIRED":
		return "pending"
	default:
		return "unknown"
	}
}

// CreatePR creates a new pull request and returns the PR number.
func CreatePR(remote, branch, baseBranch, title string) (int, error) {
	output, err := runGH("pr", "create",
		"--repo", remote,
		"--head", branch,
		"--base", baseBranch,
		"--title", title,
		"--body", "",
	)
	if err != nil {
		return 0, fmt.Errorf("failed to create PR: %w", err)
	}

	// gh pr create outputs the PR URL, we need to extract the number
	// URL format: https://github.com/owner/repo/pull/123
	parts := strings.Split(output, "/")
	if len(parts) == 0 {
		return 0, fmt.Errorf("unexpected PR create output: %s", output)
	}

	numStr := parts[len(parts)-1]
	num, err := strconv.Atoi(numStr)
	if err != nil {
		return 0, fmt.Errorf("failed to parse PR number from %s: %w", output, err)
	}

	return num, nil
}

// GetPRForBranch returns the PR for a specific branch, or nil if none exists.
func GetPRForBranch(remote, branch string) (*PR, error) {
	output, err := runGH("pr", "view", branch,
		"--repo", remote,
		"--json", "number,headRefName,state,statusCheckRollup,reviewDecision",
	)
	if err != nil {
		// Check if the error is "no pull requests found"
		if strings.Contains(err.Error(), "no pull requests found") ||
			strings.Contains(err.Error(), "Could not resolve") {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get PR for branch %s: %w", branch, err)
	}

	if output == "" {
		return nil, nil
	}

	var p ghPR
	if err := json.Unmarshal([]byte(output), &p); err != nil {
		return nil, fmt.Errorf("failed to parse PR: %w", err)
	}

	return &PR{
		Number:       p.Number,
		Branch:       p.HeadRefName,
		State:        strings.ToLower(p.State),
		CIStatus:     normalizeCIStatus(p.StatusRollup),
		ReviewStatus: normalizeReviewStatus(p.ReviewDecision),
	}, nil
}

// ghBranch is the JSON structure for branch info from gh API.
type ghBranch struct {
	Name   string `json:"name"`
	Commit struct {
		SHA       string `json:"sha"`
		Committer struct {
			Date string `json:"date"`
		} `json:"committer"`
	} `json:"commit"`
}

// ListMyBranches returns remote branches where the current user has open PRs.
func ListMyBranches(remote string) ([]RemoteBranch, error) {
	// Get all PRs by the current user to find their branches
	prs, err := ListMyPRs(remote)
	if err != nil {
		return nil, err
	}

	if len(prs) == 0 {
		return []RemoteBranch{}, nil
	}

	// Build a set of branch names from PRs
	branchSet := make(map[string]bool)
	for _, pr := range prs {
		branchSet[pr.Branch] = true
	}

	// Get branch details from the API
	branches := make([]RemoteBranch, 0, len(branchSet))
	for branchName := range branchSet {
		output, err := runGH("api",
			fmt.Sprintf("repos/%s/branches/%s", remote, branchName),
			"--jq", ".name, .commit.sha, .commit.commit.committer.date",
		)
		if err != nil {
			// Branch might have been deleted, skip it
			continue
		}

		lines := strings.Split(output, "\n")
		if len(lines) < 3 {
			continue
		}

		var age time.Duration
		if dateStr := lines[2]; dateStr != "" {
			if t, err := time.Parse(time.RFC3339, dateStr); err == nil {
				age = time.Since(t)
			}
		}

		branches = append(branches, RemoteBranch{
			Name:       lines[0],
			LastCommit: lines[1],
			Age:        age,
		})
	}

	return branches, nil
}
