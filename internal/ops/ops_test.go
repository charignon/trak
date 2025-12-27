package ops

import (
	"testing"
	"time"

	"github.com/laurent/trak/internal/config"
	"github.com/laurent/trak/internal/db"
)

// testDB creates an in-memory database for testing.
func testDB(t *testing.T) *db.DB {
	t.Helper()
	database, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}
	if err := database.Migrate(); err != nil {
		t.Fatalf("failed to migrate test database: %v", err)
	}
	return database
}

// testConfig creates a test configuration.
func testConfig() *config.Config {
	return &config.Config{
		Repo: config.RepoConfig{
			Path:   "/tmp/test-repo",
			Remote: "testowner/testrepo",
		},
	}
}

func TestNew(t *testing.T) {
	database := testDB(t)
	defer database.Close()

	cfg := testConfig()
	ops := New(database, cfg)

	if ops == nil {
		t.Fatal("New returned nil")
	}
	if ops.db != database {
		t.Error("Ops.db not set correctly")
	}
	if ops.config != cfg {
		t.Error("Ops.config not set correctly")
	}
}

func TestTrackWithStatus(t *testing.T) {
	// Test that TrackWithStatus struct works correctly
	now := time.Now()
	path := "/tmp/worktree"

	tws := TrackWithStatus{
		Track: db.Track{
			ID:           1,
			Branch:       "feature/test",
			RemoteURL:    "owner/repo",
			HeadSHA:      "abc123",
			Type:         db.TrackTypeWorktree,
			Path:         &path,
			CreatedAt:    now,
			LastAccessed: &now,
		},
	}

	if tws.Track.Branch != "feature/test" {
		t.Errorf("expected branch 'feature/test', got '%s'", tws.Track.Branch)
	}
}

func TestRemoteBranch(t *testing.T) {
	// Test that RemoteBranch struct works correctly
	rb := RemoteBranch{
		Name:       "feature/test",
		LastCommit: "abc123",
		Age:        24 * time.Hour,
		HasPR:      true,
		PRNumber:   42,
	}

	if rb.Name != "feature/test" {
		t.Errorf("expected name 'feature/test', got '%s'", rb.Name)
	}
	if !rb.HasPR {
		t.Error("expected HasPR to be true")
	}
	if rb.PRNumber != 42 {
		t.Errorf("expected PRNumber 42, got %d", rb.PRNumber)
	}
}

func TestSyncResult(t *testing.T) {
	// Test SyncResult struct
	result := SyncResult{
		Rebased:       true,
		Pushed:        true,
		PRCreated:     true,
		PRNumber:      123,
		HasConflicts:  false,
		ConflictsPath: "",
	}

	if !result.Rebased {
		t.Error("expected Rebased to be true")
	}
	if !result.Pushed {
		t.Error("expected Pushed to be true")
	}
	if !result.PRCreated {
		t.Error("expected PRCreated to be true")
	}
	if result.PRNumber != 123 {
		t.Errorf("expected PRNumber 123, got %d", result.PRNumber)
	}
}

func TestSyncResultWithConflicts(t *testing.T) {
	result := SyncResult{
		HasConflicts:  true,
		ConflictsPath: "/tmp/worktree",
	}

	if !result.HasConflicts {
		t.Error("expected HasConflicts to be true")
	}
	if result.ConflictsPath != "/tmp/worktree" {
		t.Errorf("expected ConflictsPath '/tmp/worktree', got '%s'", result.ConflictsPath)
	}
}

func TestDeleteTrackNotFound(t *testing.T) {
	database := testDB(t)
	defer database.Close()

	cfg := testConfig()
	ops := New(database, cfg)

	err := ops.DeleteTrack("nonexistent-branch", false)
	if err == nil {
		t.Error("expected error for non-existent track")
	}
}

func TestJumpToTrackNotFound(t *testing.T) {
	database := testDB(t)
	defer database.Close()

	cfg := testConfig()
	ops := New(database, cfg)

	err := ops.JumpToTrack("nonexistent-branch")
	if err == nil {
		t.Error("expected error for non-existent track")
	}
}

func TestRunAINotFound(t *testing.T) {
	database := testDB(t)
	defer database.Close()

	cfg := testConfig()
	ops := New(database, cfg)

	err := ops.RunAI("nonexistent-branch")
	if err == nil {
		t.Error("expected error for non-existent track")
	}
}

func TestSyncTrackNotFound(t *testing.T) {
	database := testDB(t)
	defer database.Close()

	cfg := testConfig()
	ops := New(database, cfg)

	_, err := ops.SyncTrack("nonexistent-branch")
	if err == nil {
		t.Error("expected error for non-existent track")
	}
}

func TestSyncTrackDevboxUnsupported(t *testing.T) {
	database := testDB(t)
	defer database.Close()

	cfg := testConfig()
	ops := New(database, cfg)

	// Insert a devbox track
	devboxName := "test-devbox"
	now := time.Now()
	err := database.InsertTrack(db.Track{
		Branch:       "feature/devbox",
		RemoteURL:    cfg.Repo.Remote,
		HeadSHA:      "abc123",
		Type:         db.TrackTypeDevbox,
		DevboxName:   &devboxName,
		CreatedAt:    now,
		LastAccessed: &now,
	})
	if err != nil {
		t.Fatalf("failed to insert test track: %v", err)
	}

	_, err = ops.SyncTrack("feature/devbox")
	if err == nil {
		t.Error("expected error for devbox track sync")
	}
	if err.Error() != "sync is not supported for devbox tracks (use devbox CLI directly)" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestRunAIDevboxUnsupported(t *testing.T) {
	database := testDB(t)
	defer database.Close()

	cfg := testConfig()
	ops := New(database, cfg)

	// Insert a devbox track
	devboxName := "test-devbox"
	now := time.Now()
	err := database.InsertTrack(db.Track{
		Branch:       "feature/devbox",
		RemoteURL:    cfg.Repo.Remote,
		HeadSHA:      "abc123",
		Type:         db.TrackTypeDevbox,
		DevboxName:   &devboxName,
		CreatedAt:    now,
		LastAccessed: &now,
	})
	if err != nil {
		t.Fatalf("failed to insert test track: %v", err)
	}

	err = ops.RunAI("feature/devbox")
	if err == nil {
		t.Error("expected error for devbox AI")
	}
	if err.Error() != "AI assistant is not supported for devbox tracks" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestListTracksWithStatusEmpty(t *testing.T) {
	database := testDB(t)
	defer database.Close()

	cfg := testConfig()
	ops := New(database, cfg)

	tracks, err := ops.ListTracksWithStatus()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tracks) != 0 {
		t.Errorf("expected 0 tracks, got %d", len(tracks))
	}
}

func TestListTracksWithStatusWithTracks(t *testing.T) {
	database := testDB(t)
	defer database.Close()

	cfg := testConfig()
	ops := New(database, cfg)

	// Insert a track (status refresh will fail but we shouldn't error)
	path := "/tmp/nonexistent"
	now := time.Now()
	err := database.InsertTrack(db.Track{
		Branch:       "feature/test",
		RemoteURL:    cfg.Repo.Remote,
		HeadSHA:      "abc123",
		Type:         db.TrackTypeWorktree,
		Path:         &path,
		CreatedAt:    now,
		LastAccessed: &now,
	})
	if err != nil {
		t.Fatalf("failed to insert test track: %v", err)
	}

	tracks, err := ops.ListTracksWithStatus()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tracks) != 1 {
		t.Errorf("expected 1 track, got %d", len(tracks))
	}
	if tracks[0].Track.Branch != "feature/test" {
		t.Errorf("expected branch 'feature/test', got '%s'", tracks[0].Track.Branch)
	}
}

func TestRefreshTrackStatusStale(t *testing.T) {
	database := testDB(t)
	defer database.Close()

	cfg := testConfig()
	ops := New(database, cfg)

	// Create a track that was accessed 10 days ago
	path := "/tmp/nonexistent"
	oldTime := time.Now().Add(-10 * 24 * time.Hour)
	trk := db.Track{
		Branch:       "feature/old",
		RemoteURL:    cfg.Repo.Remote,
		HeadSHA:      "abc123",
		Type:         db.TrackTypeWorktree,
		Path:         &path,
		CreatedAt:    oldTime,
		LastAccessed: &oldTime,
	}

	status, _ := ops.RefreshTrackStatus(trk)
	if !status.IsStale {
		t.Error("expected track to be marked as stale")
	}
}

func TestRefreshTrackStatusNotStale(t *testing.T) {
	database := testDB(t)
	defer database.Close()

	cfg := testConfig()
	ops := New(database, cfg)

	// Create a track that was accessed recently
	path := "/tmp/nonexistent"
	now := time.Now()
	trk := db.Track{
		Branch:       "feature/recent",
		RemoteURL:    cfg.Repo.Remote,
		HeadSHA:      "abc123",
		Type:         db.TrackTypeWorktree,
		Path:         &path,
		CreatedAt:    now,
		LastAccessed: &now,
	}

	status, _ := ops.RefreshTrackStatus(trk)
	if status.IsStale {
		t.Error("expected track to not be marked as stale")
	}
}

func TestRefreshTrackStatusDevbox(t *testing.T) {
	database := testDB(t)
	defer database.Close()

	cfg := testConfig()
	ops := New(database, cfg)

	// Create a devbox track
	devboxName := "test-devbox"
	now := time.Now()
	trk := db.Track{
		Branch:       "feature/devbox",
		RemoteURL:    cfg.Repo.Remote,
		HeadSHA:      "abc123",
		Type:         db.TrackTypeDevbox,
		DevboxName:   &devboxName,
		CreatedAt:    now,
		LastAccessed: &now,
	}

	// Should not error, but git status will be empty
	status, err := ops.RefreshTrackStatus(trk)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Devbox tracks have no local git status
	if status.GitStatus.Clean {
		// Default value is false, so this is expected
	}
}

func TestSyncTrackWorktreeNoPath(t *testing.T) {
	database := testDB(t)
	defer database.Close()

	cfg := testConfig()
	ops := New(database, cfg)

	// Insert a worktree track with nil path (edge case)
	now := time.Now()
	err := database.InsertTrack(db.Track{
		Branch:       "feature/nopath",
		RemoteURL:    cfg.Repo.Remote,
		HeadSHA:      "abc123",
		Type:         db.TrackTypeWorktree,
		Path:         nil, // No path
		CreatedAt:    now,
		LastAccessed: &now,
	})
	if err != nil {
		t.Fatalf("failed to insert test track: %v", err)
	}

	_, err = ops.SyncTrack("feature/nopath")
	if err == nil {
		t.Error("expected error for worktree with no path")
	}
	if err.Error() != "worktree track has no path" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestRunAIWorktreeNoPath(t *testing.T) {
	database := testDB(t)
	defer database.Close()

	cfg := testConfig()
	ops := New(database, cfg)

	// Insert a worktree track with nil path (edge case)
	now := time.Now()
	err := database.InsertTrack(db.Track{
		Branch:       "feature/nopath",
		RemoteURL:    cfg.Repo.Remote,
		HeadSHA:      "abc123",
		Type:         db.TrackTypeWorktree,
		Path:         nil, // No path
		CreatedAt:    now,
		LastAccessed: &now,
	})
	if err != nil {
		t.Fatalf("failed to insert test track: %v", err)
	}

	err = ops.RunAI("feature/nopath")
	if err == nil {
		t.Error("expected error for worktree with no path")
	}
	if err.Error() != "worktree track has no path" {
		t.Errorf("unexpected error message: %v", err)
	}
}
