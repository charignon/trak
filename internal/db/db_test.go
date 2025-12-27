package db

import (
	"testing"
	"time"
)

func setupTestDB(t *testing.T) *DB {
	t.Helper()
	db, err := Open(":memory:")
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	if err := db.Migrate(); err != nil {
		t.Fatalf("failed to migrate test db: %v", err)
	}
	return db
}

func TestOpenClose(t *testing.T) {
	db, err := Open(":memory:")
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("failed to close db: %v", err)
	}
}

func TestMigrate(t *testing.T) {
	db, err := Open(":memory:")
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()

	if err := db.Migrate(); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	// Running migrate again should be idempotent
	if err := db.Migrate(); err != nil {
		t.Fatalf("second migrate failed: %v", err)
	}
}

func TestInsertAndGetTrack(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	path := "/home/user/worktrees/feature-test"
	track := Track{
		Branch:    "feature-test",
		RemoteURL: "https://github.com/user/repo",
		HeadSHA:   "abc123def456",
		Type:      TrackTypeWorktree,
		Path:      &path,
	}

	if err := db.InsertTrack(track); err != nil {
		t.Fatalf("failed to insert track: %v", err)
	}

	got, err := db.GetTrack(track.RemoteURL, track.Branch)
	if err != nil {
		t.Fatalf("failed to get track: %v", err)
	}
	if got == nil {
		t.Fatal("expected track, got nil")
	}

	if got.Branch != track.Branch {
		t.Errorf("branch = %q, want %q", got.Branch, track.Branch)
	}
	if got.RemoteURL != track.RemoteURL {
		t.Errorf("remote_url = %q, want %q", got.RemoteURL, track.RemoteURL)
	}
	if got.HeadSHA != track.HeadSHA {
		t.Errorf("head_sha = %q, want %q", got.HeadSHA, track.HeadSHA)
	}
	if got.Type != track.Type {
		t.Errorf("type = %q, want %q", got.Type, track.Type)
	}
	if got.Path == nil || *got.Path != *track.Path {
		t.Errorf("path = %v, want %v", got.Path, track.Path)
	}
	if got.ID == 0 {
		t.Error("expected ID to be set")
	}
	if got.CreatedAt.IsZero() {
		t.Error("expected created_at to be set")
	}
}

func TestInsertDevboxTrack(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	devboxName := "mydevbox-123"
	track := Track{
		Branch:     "feature-devbox",
		RemoteURL:  "https://github.com/user/repo",
		HeadSHA:    "def789abc012",
		Type:       TrackTypeDevbox,
		DevboxName: &devboxName,
	}

	if err := db.InsertTrack(track); err != nil {
		t.Fatalf("failed to insert devbox track: %v", err)
	}

	got, err := db.GetTrack(track.RemoteURL, track.Branch)
	if err != nil {
		t.Fatalf("failed to get track: %v", err)
	}
	if got == nil {
		t.Fatal("expected track, got nil")
	}

	if got.Type != TrackTypeDevbox {
		t.Errorf("type = %q, want %q", got.Type, TrackTypeDevbox)
	}
	if got.DevboxName == nil || *got.DevboxName != devboxName {
		t.Errorf("devbox_name = %v, want %v", got.DevboxName, devboxName)
	}
	if got.Path != nil {
		t.Errorf("path = %v, want nil", got.Path)
	}
}

func TestGetTrackNotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	got, err := db.GetTrack("https://github.com/user/nonexistent", "nonexistent-branch")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil, got %v", got)
	}
}

func TestUpdateTrack(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	path := "/home/user/worktrees/feature-update"
	track := Track{
		Branch:    "feature-update",
		RemoteURL: "https://github.com/user/repo",
		HeadSHA:   "initial123",
		Type:      TrackTypeWorktree,
		Path:      &path,
	}

	if err := db.InsertTrack(track); err != nil {
		t.Fatalf("failed to insert track: %v", err)
	}

	// Update the track
	now := time.Now()
	newPath := "/home/user/worktrees/feature-update-new"
	track.HeadSHA = "updated456"
	track.Path = &newPath
	track.LastAccessed = &now

	if err := db.UpdateTrack(track); err != nil {
		t.Fatalf("failed to update track: %v", err)
	}

	got, err := db.GetTrack(track.RemoteURL, track.Branch)
	if err != nil {
		t.Fatalf("failed to get track: %v", err)
	}

	if got.HeadSHA != "updated456" {
		t.Errorf("head_sha = %q, want %q", got.HeadSHA, "updated456")
	}
	if got.Path == nil || *got.Path != newPath {
		t.Errorf("path = %v, want %v", got.Path, newPath)
	}
	if got.LastAccessed == nil {
		t.Error("expected last_accessed to be set")
	}
}

func TestUpdateTrackNotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	track := Track{
		Branch:    "nonexistent",
		RemoteURL: "https://github.com/user/repo",
		HeadSHA:   "abc123",
		Type:      TrackTypeWorktree,
	}

	err := db.UpdateTrack(track)
	if err == nil {
		t.Error("expected error for non-existent track")
	}
}

func TestDeleteTrack(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	path := "/home/user/worktrees/feature-delete"
	track := Track{
		Branch:    "feature-delete",
		RemoteURL: "https://github.com/user/repo",
		HeadSHA:   "abc123",
		Type:      TrackTypeWorktree,
		Path:      &path,
	}

	if err := db.InsertTrack(track); err != nil {
		t.Fatalf("failed to insert track: %v", err)
	}

	if err := db.DeleteTrack(track.RemoteURL, track.Branch); err != nil {
		t.Fatalf("failed to delete track: %v", err)
	}

	got, err := db.GetTrack(track.RemoteURL, track.Branch)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Error("expected track to be deleted")
	}
}

func TestDeleteTrackNotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	err := db.DeleteTrack("https://github.com/user/repo", "nonexistent")
	if err == nil {
		t.Error("expected error for non-existent track")
	}
}

func TestListTracks(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Insert multiple tracks
	tracks := []Track{
		{
			Branch:    "feature-a",
			RemoteURL: "https://github.com/user/repo",
			HeadSHA:   "sha-a",
			Type:      TrackTypeWorktree,
		},
		{
			Branch:    "feature-b",
			RemoteURL: "https://github.com/user/repo",
			HeadSHA:   "sha-b",
			Type:      TrackTypeDevbox,
		},
		{
			Branch:    "feature-c",
			RemoteURL: "https://github.com/user/other-repo",
			HeadSHA:   "sha-c",
			Type:      TrackTypeWorktree,
		},
	}

	for _, track := range tracks {
		if err := db.InsertTrack(track); err != nil {
			t.Fatalf("failed to insert track: %v", err)
		}
	}

	got, err := db.ListTracks()
	if err != nil {
		t.Fatalf("failed to list tracks: %v", err)
	}

	if len(got) != len(tracks) {
		t.Errorf("got %d tracks, want %d", len(got), len(tracks))
	}
}

func TestListTracksEmpty(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	got, err := db.ListTracks()
	if err != nil {
		t.Fatalf("failed to list tracks: %v", err)
	}

	if got == nil {
		t.Error("expected empty slice, got nil")
	}
	if len(got) != 0 {
		t.Errorf("expected 0 tracks, got %d", len(got))
	}
}

func TestUpdateLastAccessed(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	track := Track{
		Branch:    "feature-access",
		RemoteURL: "https://github.com/user/repo",
		HeadSHA:   "abc123",
		Type:      TrackTypeWorktree,
	}

	if err := db.InsertTrack(track); err != nil {
		t.Fatalf("failed to insert track: %v", err)
	}

	// Verify last_accessed is initially nil
	got, _ := db.GetTrack(track.RemoteURL, track.Branch)
	if got.LastAccessed != nil {
		t.Error("expected last_accessed to be nil initially")
	}

	// Update last accessed
	if err := db.UpdateLastAccessed(track.RemoteURL, track.Branch); err != nil {
		t.Fatalf("failed to update last accessed: %v", err)
	}

	got, _ = db.GetTrack(track.RemoteURL, track.Branch)
	if got.LastAccessed == nil {
		t.Error("expected last_accessed to be set")
	}
}

func TestUpdateLastAccessedNotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	err := db.UpdateLastAccessed("https://github.com/user/repo", "nonexistent")
	if err == nil {
		t.Error("expected error for non-existent track")
	}
}

func TestUpdateHeadSHA(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	track := Track{
		Branch:    "feature-sha",
		RemoteURL: "https://github.com/user/repo",
		HeadSHA:   "initial123",
		Type:      TrackTypeWorktree,
	}

	if err := db.InsertTrack(track); err != nil {
		t.Fatalf("failed to insert track: %v", err)
	}

	newSHA := "updated456"
	if err := db.UpdateHeadSHA(track.RemoteURL, track.Branch, newSHA); err != nil {
		t.Fatalf("failed to update head SHA: %v", err)
	}

	got, _ := db.GetTrack(track.RemoteURL, track.Branch)
	if got.HeadSHA != newSHA {
		t.Errorf("head_sha = %q, want %q", got.HeadSHA, newSHA)
	}
}

func TestUpdateHeadSHANotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	err := db.UpdateHeadSHA("https://github.com/user/repo", "nonexistent", "sha123")
	if err == nil {
		t.Error("expected error for non-existent track")
	}
}

func TestUniqueConstraint(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	track := Track{
		Branch:    "feature-unique",
		RemoteURL: "https://github.com/user/repo",
		HeadSHA:   "abc123",
		Type:      TrackTypeWorktree,
	}

	if err := db.InsertTrack(track); err != nil {
		t.Fatalf("failed to insert first track: %v", err)
	}

	// Try to insert duplicate
	err := db.InsertTrack(track)
	if err == nil {
		t.Error("expected error for duplicate track")
	}
}

func TestTypeConstraint(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	track := Track{
		Branch:    "feature-badtype",
		RemoteURL: "https://github.com/user/repo",
		HeadSHA:   "abc123",
		Type:      "invalid",
	}

	err := db.InsertTrack(track)
	if err == nil {
		t.Error("expected error for invalid type")
	}
}
