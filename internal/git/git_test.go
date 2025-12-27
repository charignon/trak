package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// setupTestRepo creates a temporary git repository for testing.
// It returns the repo path and a cleanup function.
func setupTestRepo(t *testing.T) (string, func()) {
	t.Helper()

	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "trak-git-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	cleanup := func() {
		os.RemoveAll(tmpDir)
	}

	// Initialize git repo
	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		cleanup()
		t.Fatalf("failed to init git repo: %v", err)
	}

	// Configure git user for commits
	cmd = exec.Command("git", "config", "user.email", "test@test.com")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		cleanup()
		t.Fatalf("failed to config user.email: %v", err)
	}

	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		cleanup()
		t.Fatalf("failed to config user.name: %v", err)
	}

	// Create initial commit
	testFile := filepath.Join(tmpDir, "README.md")
	if err := os.WriteFile(testFile, []byte("# Test Repo\n"), 0644); err != nil {
		cleanup()
		t.Fatalf("failed to create test file: %v", err)
	}

	cmd = exec.Command("git", "add", ".")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		cleanup()
		t.Fatalf("failed to git add: %v", err)
	}

	cmd = exec.Command("git", "commit", "-m", "Initial commit")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		cleanup()
		t.Fatalf("failed to git commit: %v", err)
	}

	// Rename default branch to main for consistency
	cmd = exec.Command("git", "branch", "-M", "main")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		cleanup()
		t.Fatalf("failed to rename branch to main: %v", err)
	}

	return tmpDir, cleanup
}

// setupTestRepoWithRemote creates a test repo with a "remote" (another local bare repo).
func setupTestRepoWithRemote(t *testing.T) (repoPath, remotePath string, cleanup func()) {
	t.Helper()

	// Create temp directory for both repos
	tmpDir, err := os.MkdirTemp("", "trak-git-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	cleanup = func() {
		os.RemoveAll(tmpDir)
	}

	// Create bare "remote" repo
	remotePath = filepath.Join(tmpDir, "remote.git")
	cmd := exec.Command("git", "init", "--bare", remotePath)
	if err := cmd.Run(); err != nil {
		cleanup()
		t.Fatalf("failed to create bare repo: %v", err)
	}

	// Create local repo
	repoPath = filepath.Join(tmpDir, "local")
	if err := os.MkdirAll(repoPath, 0755); err != nil {
		cleanup()
		t.Fatalf("failed to create local dir: %v", err)
	}

	cmd = exec.Command("git", "init")
	cmd.Dir = repoPath
	if err := cmd.Run(); err != nil {
		cleanup()
		t.Fatalf("failed to init local repo: %v", err)
	}

	// Configure git user
	cmd = exec.Command("git", "config", "user.email", "test@test.com")
	cmd.Dir = repoPath
	cmd.Run()

	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = repoPath
	cmd.Run()

	// Create initial commit
	testFile := filepath.Join(repoPath, "README.md")
	if err := os.WriteFile(testFile, []byte("# Test Repo\n"), 0644); err != nil {
		cleanup()
		t.Fatalf("failed to create test file: %v", err)
	}

	cmd = exec.Command("git", "add", ".")
	cmd.Dir = repoPath
	cmd.Run()

	cmd = exec.Command("git", "commit", "-m", "Initial commit")
	cmd.Dir = repoPath
	cmd.Run()

	// Rename to main
	cmd = exec.Command("git", "branch", "-M", "main")
	cmd.Dir = repoPath
	cmd.Run()

	// Add remote
	cmd = exec.Command("git", "remote", "add", "origin", remotePath)
	cmd.Dir = repoPath
	cmd.Run()

	// Push to remote
	cmd = exec.Command("git", "push", "-u", "origin", "main")
	cmd.Dir = repoPath
	cmd.Run()

	// Set up origin/HEAD
	cmd = exec.Command("git", "remote", "set-head", "origin", "main")
	cmd.Dir = repoPath
	cmd.Run()

	return repoPath, remotePath, cleanup
}

func TestGetHeadSHA(t *testing.T) {
	repoPath, cleanup := setupTestRepo(t)
	defer cleanup()

	sha, err := GetHeadSHA(repoPath)
	if err != nil {
		t.Fatalf("GetHeadSHA failed: %v", err)
	}

	if len(sha) != 40 {
		t.Errorf("expected 40 char SHA, got %d chars: %s", len(sha), sha)
	}
}

func TestGetBranchSHA(t *testing.T) {
	repoPath, cleanup := setupTestRepo(t)
	defer cleanup()

	headSHA, _ := GetHeadSHA(repoPath)
	branchSHA, err := GetBranchSHA(repoPath, "main")
	if err != nil {
		t.Fatalf("GetBranchSHA failed: %v", err)
	}

	if branchSHA != headSHA {
		t.Errorf("expected branch SHA %s to match HEAD SHA %s", branchSHA, headSHA)
	}
}

func TestIsDirty(t *testing.T) {
	repoPath, cleanup := setupTestRepo(t)
	defer cleanup()

	// Should be clean initially
	dirty, err := IsDirty(repoPath)
	if err != nil {
		t.Fatalf("IsDirty failed: %v", err)
	}
	if dirty {
		t.Error("expected clean repo, got dirty")
	}

	// Make it dirty
	testFile := filepath.Join(repoPath, "new-file.txt")
	if err := os.WriteFile(testFile, []byte("new content"), 0644); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}

	dirty, err = IsDirty(repoPath)
	if err != nil {
		t.Fatalf("IsDirty failed: %v", err)
	}
	if !dirty {
		t.Error("expected dirty repo, got clean")
	}
}

func TestCreateBranch(t *testing.T) {
	repoPath, cleanup := setupTestRepo(t)
	defer cleanup()

	err := CreateBranch(repoPath, "feature-test", "main")
	if err != nil {
		t.Fatalf("CreateBranch failed: %v", err)
	}

	// Verify branch exists
	sha, err := GetBranchSHA(repoPath, "feature-test")
	if err != nil {
		t.Fatalf("feature-test branch not created: %v", err)
	}

	mainSHA, _ := GetBranchSHA(repoPath, "main")
	if sha != mainSHA {
		t.Errorf("new branch SHA should match main SHA")
	}
}

func TestCheckout(t *testing.T) {
	repoPath, cleanup := setupTestRepo(t)
	defer cleanup()

	// Create and checkout a new branch
	CreateBranch(repoPath, "feature-test", "main")

	err := Checkout(repoPath, "feature-test")
	if err != nil {
		t.Fatalf("Checkout failed: %v", err)
	}

	currentBranch, err := GetCurrentBranch(repoPath)
	if err != nil {
		t.Fatalf("GetCurrentBranch failed: %v", err)
	}

	if currentBranch != "feature-test" {
		t.Errorf("expected current branch 'feature-test', got '%s'", currentBranch)
	}
}

func TestGetCurrentBranch(t *testing.T) {
	repoPath, cleanup := setupTestRepo(t)
	defer cleanup()

	branch, err := GetCurrentBranch(repoPath)
	if err != nil {
		t.Fatalf("GetCurrentBranch failed: %v", err)
	}

	if branch != "main" {
		t.Errorf("expected 'main', got '%s'", branch)
	}
}

func TestWorktreeOperations(t *testing.T) {
	repoPath, cleanup := setupTestRepo(t)
	defer cleanup()

	// Create a branch for the worktree
	CreateBranch(repoPath, "worktree-test", "main")

	// Create worktree in a unique subdirectory
	worktreePath := filepath.Join(filepath.Dir(repoPath), "wt-"+filepath.Base(repoPath))
	err := WorktreeAdd(repoPath, worktreePath, "worktree-test")
	if err != nil {
		t.Fatalf("WorktreeAdd failed: %v", err)
	}

	// Verify worktree exists
	if _, err := os.Stat(worktreePath); os.IsNotExist(err) {
		t.Error("worktree directory was not created")
	}

	// Verify we can get HEAD in worktree
	_, err = GetHeadSHA(worktreePath)
	if err != nil {
		t.Errorf("could not get HEAD in worktree: %v", err)
	}

	// Remove worktree
	err = WorktreeRemove(repoPath, worktreePath)
	if err != nil {
		t.Fatalf("WorktreeRemove failed: %v", err)
	}

	// Prune
	err = WorktreePrune(repoPath)
	if err != nil {
		t.Fatalf("WorktreePrune failed: %v", err)
	}
}

func TestFetch(t *testing.T) {
	repoPath, _, cleanup := setupTestRepoWithRemote(t)
	defer cleanup()

	err := Fetch(repoPath)
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}
}

func TestGetDefaultBranch(t *testing.T) {
	repoPath, _, cleanup := setupTestRepoWithRemote(t)
	defer cleanup()

	branch, err := GetDefaultBranch(repoPath)
	if err != nil {
		t.Fatalf("GetDefaultBranch failed: %v", err)
	}

	if branch != "main" {
		t.Errorf("expected 'main', got '%s'", branch)
	}
}

func TestPush(t *testing.T) {
	repoPath, _, cleanup := setupTestRepoWithRemote(t)
	defer cleanup()

	// Create a new branch and push it
	CreateBranch(repoPath, "push-test", "main")
	Checkout(repoPath, "push-test")

	// Make a commit
	testFile := filepath.Join(repoPath, "push-test.txt")
	os.WriteFile(testFile, []byte("push test"), 0644)

	cmd := exec.Command("git", "add", ".")
	cmd.Dir = repoPath
	cmd.Run()

	cmd = exec.Command("git", "commit", "-m", "Push test commit")
	cmd.Dir = repoPath
	cmd.Run()

	// Push
	err := Push(repoPath, "push-test")
	if err != nil {
		t.Fatalf("Push failed: %v", err)
	}
}

func TestPushDelete(t *testing.T) {
	repoPath, _, cleanup := setupTestRepoWithRemote(t)
	defer cleanup()

	// Create and push a branch
	CreateBranch(repoPath, "delete-test", "main")
	Push(repoPath, "delete-test")

	// Delete remote branch
	err := PushDelete(repoPath, "delete-test")
	if err != nil {
		t.Fatalf("PushDelete failed: %v", err)
	}
}

func TestAheadBehind(t *testing.T) {
	repoPath, _, cleanup := setupTestRepoWithRemote(t)
	defer cleanup()

	// Create a branch with additional commits
	CreateBranch(repoPath, "ahead-test", "main")
	Checkout(repoPath, "ahead-test")

	// Make commits on ahead-test
	for i := 0; i < 3; i++ {
		testFile := filepath.Join(repoPath, "ahead-test.txt")
		os.WriteFile(testFile, []byte("ahead "+string(rune('0'+i))), 0644)

		cmd := exec.Command("git", "add", ".")
		cmd.Dir = repoPath
		cmd.Run()

		cmd = exec.Command("git", "commit", "-m", "Ahead commit")
		cmd.Dir = repoPath
		cmd.Run()
	}

	ahead, behind, err := AheadBehind(repoPath, "ahead-test", "main")
	if err != nil {
		t.Fatalf("AheadBehind failed: %v", err)
	}

	if ahead != 3 {
		t.Errorf("expected 3 ahead, got %d", ahead)
	}

	if behind != 0 {
		t.Errorf("expected 0 behind, got %d", behind)
	}
}

func TestRebase(t *testing.T) {
	repoPath, cleanup := setupTestRepo(t)
	defer cleanup()

	// Create feature branch from main
	CreateBranch(repoPath, "feature", "main")

	// Add commit to main
	Checkout(repoPath, "main")
	testFile := filepath.Join(repoPath, "main-change.txt")
	os.WriteFile(testFile, []byte("main change"), 0644)

	cmd := exec.Command("git", "add", ".")
	cmd.Dir = repoPath
	cmd.Run()

	cmd = exec.Command("git", "commit", "-m", "Main commit")
	cmd.Dir = repoPath
	cmd.Run()

	// Add commit to feature (different file, no conflict)
	Checkout(repoPath, "feature")
	testFile = filepath.Join(repoPath, "feature-change.txt")
	os.WriteFile(testFile, []byte("feature change"), 0644)

	cmd = exec.Command("git", "add", ".")
	cmd.Dir = repoPath
	cmd.Run()

	cmd = exec.Command("git", "commit", "-m", "Feature commit")
	cmd.Dir = repoPath
	cmd.Run()

	// Rebase feature onto main
	conflicts, err := Rebase(repoPath, "main")
	if err != nil {
		t.Fatalf("Rebase failed: %v", err)
	}

	if conflicts {
		t.Error("expected no conflicts")
	}

	// Verify feature has both changes
	if _, err := os.Stat(filepath.Join(repoPath, "main-change.txt")); os.IsNotExist(err) {
		t.Error("main-change.txt should exist after rebase")
	}
	if _, err := os.Stat(filepath.Join(repoPath, "feature-change.txt")); os.IsNotExist(err) {
		t.Error("feature-change.txt should exist after rebase")
	}
}

func TestRebaseWithConflicts(t *testing.T) {
	repoPath, cleanup := setupTestRepo(t)
	defer cleanup()

	// Create feature branch from main
	CreateBranch(repoPath, "feature", "main")

	// Add commit to main (modify same file)
	Checkout(repoPath, "main")
	testFile := filepath.Join(repoPath, "README.md")
	os.WriteFile(testFile, []byte("# Main change\n"), 0644)

	cmd := exec.Command("git", "add", ".")
	cmd.Dir = repoPath
	cmd.Run()

	cmd = exec.Command("git", "commit", "-m", "Main commit")
	cmd.Dir = repoPath
	cmd.Run()

	// Add commit to feature (modify same file - conflict)
	Checkout(repoPath, "feature")
	os.WriteFile(testFile, []byte("# Feature change\n"), 0644)

	cmd = exec.Command("git", "add", ".")
	cmd.Dir = repoPath
	cmd.Run()

	cmd = exec.Command("git", "commit", "-m", "Feature commit")
	cmd.Dir = repoPath
	cmd.Run()

	// Rebase should detect conflict
	conflicts, err := Rebase(repoPath, "main")

	// It's okay if err is not nil here, as rebase returns non-zero on conflicts
	if !conflicts && err == nil {
		// Read the file to check for conflict markers
		content, _ := os.ReadFile(testFile)
		if !strings.Contains(string(content), "<<<<<<<") {
			t.Error("expected conflicts or conflict markers")
		}
	}

	// Cleanup: abort the rebase if in progress
	RebaseAbort(repoPath)
}

func TestPull(t *testing.T) {
	// Create a simpler setup: bare remote, then two clones from it
	tmpDir, err := os.MkdirTemp("", "trak-git-pull-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create bare remote
	remotePath := filepath.Join(tmpDir, "remote.git")
	cmd := exec.Command("git", "init", "--bare", remotePath)
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to create bare repo: %v", err)
	}

	// Create first clone (this will be our "repoPath")
	repoPath := filepath.Join(tmpDir, "repo1")
	cmd = exec.Command("git", "clone", remotePath, repoPath)
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to clone repo1: %v", err)
	}

	// Configure repo1
	cmd = exec.Command("git", "config", "user.email", "test@test.com")
	cmd.Dir = repoPath
	cmd.Run()
	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = repoPath
	cmd.Run()

	// Create initial commit in repo1 and push
	testFile := filepath.Join(repoPath, "README.md")
	os.WriteFile(testFile, []byte("# Test\n"), 0644)
	cmd = exec.Command("git", "add", ".")
	cmd.Dir = repoPath
	cmd.Run()
	cmd = exec.Command("git", "commit", "-m", "Initial commit")
	cmd.Dir = repoPath
	cmd.Run()
	cmd = exec.Command("git", "push", "-u", "origin", "HEAD:main")
	cmd.Dir = repoPath
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to push initial commit: %v, output: %s", err, out)
	}

	// Checkout main explicitly
	cmd = exec.Command("git", "checkout", "-B", "main")
	cmd.Dir = repoPath
	cmd.Run()

	// Create second clone
	repo2Path := filepath.Join(tmpDir, "repo2")
	cmd = exec.Command("git", "clone", "-b", "main", remotePath, repo2Path)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to clone repo2: %v, output: %s", err, out)
	}

	// Configure repo2
	cmd = exec.Command("git", "config", "user.email", "test@test.com")
	cmd.Dir = repo2Path
	cmd.Run()
	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = repo2Path
	cmd.Run()

	// Make a commit in repo2
	newFile := filepath.Join(repo2Path, "from-repo2.txt")
	os.WriteFile(newFile, []byte("from repo2"), 0644)
	cmd = exec.Command("git", "add", ".")
	cmd.Dir = repo2Path
	cmd.Run()
	cmd = exec.Command("git", "commit", "-m", "Commit from repo2")
	cmd.Dir = repo2Path
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to commit in repo2: %v, output: %s", err, out)
	}

	// Push from repo2 - use HEAD to avoid branch name issues
	cmd = exec.Command("git", "push", "origin", "HEAD:main")
	cmd.Dir = repo2Path
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to push from repo2: %v, output: %s", err, out)
	}

	// Now pull in repo1 (repoPath)
	Fetch(repoPath)
	err = Pull(repoPath, "main")
	if err != nil {
		t.Fatalf("Pull failed: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(filepath.Join(repoPath, "from-repo2.txt")); os.IsNotExist(err) {
		files, _ := os.ReadDir(repoPath)
		var names []string
		for _, f := range files {
			names = append(names, f.Name())
		}
		t.Errorf("from-repo2.txt should exist after pull. Files in repo: %v", names)
	}
}
