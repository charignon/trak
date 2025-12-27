package github

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

// MockRunner is a mock command runner for testing.
type MockRunner struct {
	// Responses maps command strings to their outputs
	Responses map[string]string
	// Errors maps command strings to their errors
	Errors map[string]error
	// Calls records all commands that were executed
	Calls []string
}

func NewMockRunner() *MockRunner {
	return &MockRunner{
		Responses: make(map[string]string),
		Errors:    make(map[string]error),
		Calls:     make([]string, 0),
	}
}

func (m *MockRunner) Run(name string, args ...string) (string, error) {
	cmd := name + " " + strings.Join(args, " ")
	m.Calls = append(m.Calls, cmd)

	if err, ok := m.Errors[cmd]; ok {
		return "", err
	}

	if resp, ok := m.Responses[cmd]; ok {
		return resp, nil
	}

	return "", fmt.Errorf("unexpected command: %s", cmd)
}

func TestGetDefaultBranch(t *testing.T) {
	mock := NewMockRunner()
	SetRunner(mock)
	defer ResetRunner()

	mock.Responses["gh repo view owner/repo --json defaultBranchRef --jq .defaultBranchRef.name"] = "main"

	branch, err := GetDefaultBranch("owner/repo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if branch != "main" {
		t.Errorf("expected 'main', got %q", branch)
	}
}

func TestGetDefaultBranch_Master(t *testing.T) {
	mock := NewMockRunner()
	SetRunner(mock)
	defer ResetRunner()

	mock.Responses["gh repo view owner/repo --json defaultBranchRef --jq .defaultBranchRef.name"] = "master"

	branch, err := GetDefaultBranch("owner/repo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if branch != "master" {
		t.Errorf("expected 'master', got %q", branch)
	}
}

func TestGetDefaultBranch_Error(t *testing.T) {
	mock := NewMockRunner()
	SetRunner(mock)
	defer ResetRunner()

	mock.Errors["gh repo view owner/repo --json defaultBranchRef --jq .defaultBranchRef.name"] = fmt.Errorf("repository not found")

	_, err := GetDefaultBranch("owner/repo")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGetCurrentUser(t *testing.T) {
	mock := NewMockRunner()
	SetRunner(mock)
	defer ResetRunner()

	mock.Responses["gh api user --jq .login"] = "testuser"

	user, err := GetCurrentUser()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user != "testuser" {
		t.Errorf("expected 'testuser', got %q", user)
	}
}

func TestGetCurrentUser_NotLoggedIn(t *testing.T) {
	mock := NewMockRunner()
	SetRunner(mock)
	defer ResetRunner()

	mock.Responses["gh api user --jq .login"] = ""

	_, err := GetCurrentUser()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestListMyPRs(t *testing.T) {
	mock := NewMockRunner()
	SetRunner(mock)
	defer ResetRunner()

	mock.Responses["gh api user --jq .login"] = "testuser"
	mock.Responses["gh pr list --repo owner/repo --author testuser --state open --json number,headRefName,state,statusCheckRollup,reviewDecision"] = `[
		{"number": 42, "headRefName": "feature-branch", "state": "OPEN", "statusCheckRollup": "SUCCESS", "reviewDecision": "APPROVED"},
		{"number": 38, "headRefName": "fix-bug", "state": "OPEN", "statusCheckRollup": "FAILURE", "reviewDecision": "CHANGES_REQUESTED"}
	]`

	prs, err := ListMyPRs("owner/repo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(prs) != 2 {
		t.Fatalf("expected 2 PRs, got %d", len(prs))
	}

	// Check first PR
	if prs[0].Number != 42 {
		t.Errorf("expected PR number 42, got %d", prs[0].Number)
	}
	if prs[0].Branch != "feature-branch" {
		t.Errorf("expected branch 'feature-branch', got %q", prs[0].Branch)
	}
	if prs[0].State != "open" {
		t.Errorf("expected state 'open', got %q", prs[0].State)
	}
	if prs[0].CIStatus != "success" {
		t.Errorf("expected CI status 'success', got %q", prs[0].CIStatus)
	}
	if prs[0].ReviewStatus != "approved" {
		t.Errorf("expected review status 'approved', got %q", prs[0].ReviewStatus)
	}

	// Check second PR
	if prs[1].Number != 38 {
		t.Errorf("expected PR number 38, got %d", prs[1].Number)
	}
	if prs[1].CIStatus != "failure" {
		t.Errorf("expected CI status 'failure', got %q", prs[1].CIStatus)
	}
	if prs[1].ReviewStatus != "changes_requested" {
		t.Errorf("expected review status 'changes_requested', got %q", prs[1].ReviewStatus)
	}
}

func TestListMyPRs_Empty(t *testing.T) {
	mock := NewMockRunner()
	SetRunner(mock)
	defer ResetRunner()

	mock.Responses["gh api user --jq .login"] = "testuser"
	mock.Responses["gh pr list --repo owner/repo --author testuser --state open --json number,headRefName,state,statusCheckRollup,reviewDecision"] = "[]"

	prs, err := ListMyPRs("owner/repo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(prs) != 0 {
		t.Errorf("expected 0 PRs, got %d", len(prs))
	}
}

func TestCreatePR(t *testing.T) {
	mock := NewMockRunner()
	SetRunner(mock)
	defer ResetRunner()

	mock.Responses["gh pr create --repo owner/repo --head feature-branch --base main --title My PR --body "] = "https://github.com/owner/repo/pull/123"

	prNum, err := CreatePR("owner/repo", "feature-branch", "main", "My PR")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if prNum != 123 {
		t.Errorf("expected PR number 123, got %d", prNum)
	}
}

func TestCreatePR_Error(t *testing.T) {
	mock := NewMockRunner()
	SetRunner(mock)
	defer ResetRunner()

	mock.Errors["gh pr create --repo owner/repo --head feature-branch --base main --title My PR --body "] = fmt.Errorf("branch not pushed")

	_, err := CreatePR("owner/repo", "feature-branch", "main", "My PR")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGetPRForBranch(t *testing.T) {
	mock := NewMockRunner()
	SetRunner(mock)
	defer ResetRunner()

	mock.Responses["gh pr view feature-branch --repo owner/repo --json number,headRefName,state,statusCheckRollup,reviewDecision"] = `{"number": 42, "headRefName": "feature-branch", "state": "OPEN", "statusCheckRollup": "PENDING", "reviewDecision": "REVIEW_REQUIRED"}`

	pr, err := GetPRForBranch("owner/repo", "feature-branch")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pr == nil {
		t.Fatal("expected PR, got nil")
	}
	if pr.Number != 42 {
		t.Errorf("expected PR number 42, got %d", pr.Number)
	}
	if pr.CIStatus != "pending" {
		t.Errorf("expected CI status 'pending', got %q", pr.CIStatus)
	}
	if pr.ReviewStatus != "pending" {
		t.Errorf("expected review status 'pending', got %q", pr.ReviewStatus)
	}
}

func TestGetPRForBranch_NotFound(t *testing.T) {
	mock := NewMockRunner()
	SetRunner(mock)
	defer ResetRunner()

	mock.Errors["gh pr view feature-branch --repo owner/repo --json number,headRefName,state,statusCheckRollup,reviewDecision"] = fmt.Errorf("no pull requests found for branch feature-branch")

	pr, err := GetPRForBranch("owner/repo", "feature-branch")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pr != nil {
		t.Errorf("expected nil PR, got %+v", pr)
	}
}

func TestListMyBranches(t *testing.T) {
	mock := NewMockRunner()
	SetRunner(mock)
	defer ResetRunner()

	mock.Responses["gh api user --jq .login"] = "testuser"
	mock.Responses["gh pr list --repo owner/repo --author testuser --state open --json number,headRefName,state,statusCheckRollup,reviewDecision"] = `[
		{"number": 42, "headRefName": "feature-branch", "state": "OPEN", "statusCheckRollup": "SUCCESS", "reviewDecision": "APPROVED"}
	]`

	// Mock the branch API call
	now := time.Now().Add(-2 * time.Hour).Format(time.RFC3339)
	mock.Responses["gh api repos/owner/repo/branches/feature-branch --jq .name, .commit.sha, .commit.commit.committer.date"] = fmt.Sprintf("feature-branch\nabc123\n%s", now)

	branches, err := ListMyBranches("owner/repo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(branches) != 1 {
		t.Fatalf("expected 1 branch, got %d", len(branches))
	}

	if branches[0].Name != "feature-branch" {
		t.Errorf("expected branch name 'feature-branch', got %q", branches[0].Name)
	}
	if branches[0].LastCommit != "abc123" {
		t.Errorf("expected last commit 'abc123', got %q", branches[0].LastCommit)
	}
	// Age should be approximately 2 hours (give some tolerance)
	if branches[0].Age < time.Hour || branches[0].Age > 3*time.Hour {
		t.Errorf("expected age around 2 hours, got %v", branches[0].Age)
	}
}

func TestListMyBranches_Empty(t *testing.T) {
	mock := NewMockRunner()
	SetRunner(mock)
	defer ResetRunner()

	mock.Responses["gh api user --jq .login"] = "testuser"
	mock.Responses["gh pr list --repo owner/repo --author testuser --state open --json number,headRefName,state,statusCheckRollup,reviewDecision"] = "[]"

	branches, err := ListMyBranches("owner/repo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(branches) != 0 {
		t.Errorf("expected 0 branches, got %d", len(branches))
	}
}

func TestNormalizeCIStatus(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"SUCCESS", "success"},
		{"FAILURE", "failure"},
		{"ERROR", "failure"},
		{"PENDING", "pending"},
		{"EXPECTED", "pending"},
		{"QUEUED", "pending"},
		{"IN_PROGRESS", "pending"},
		{"", "unknown"},
		{"SOMETHING_ELSE", "unknown"},
	}

	for _, tc := range tests {
		result := normalizeCIStatus(tc.input)
		if result != tc.expected {
			t.Errorf("normalizeCIStatus(%q) = %q, expected %q", tc.input, result, tc.expected)
		}
	}
}

func TestNormalizeReviewStatus(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"APPROVED", "approved"},
		{"CHANGES_REQUESTED", "changes_requested"},
		{"REVIEW_REQUIRED", "pending"},
		{"", "unknown"},
		{"SOMETHING_ELSE", "unknown"},
	}

	for _, tc := range tests {
		result := normalizeReviewStatus(tc.input)
		if result != tc.expected {
			t.Errorf("normalizeReviewStatus(%q) = %q, expected %q", tc.input, result, tc.expected)
		}
	}
}
