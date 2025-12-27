// Package db provides SQLite database operations for trak.
package db

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

// TrackType represents the type of track (worktree or devbox).
type TrackType string

const (
	TrackTypeWorktree TrackType = "worktree"
	TrackTypeDevbox   TrackType = "devbox"
)

// Track represents a working environment associated with a branch.
type Track struct {
	ID           int64
	Branch       string
	RemoteURL    string
	HeadSHA      string
	Type         TrackType
	Path         *string // worktree path, nil for devbox
	DevboxName   *string // devbox name, nil for worktree
	CreatedAt    time.Time
	LastAccessed *time.Time
}

// DB wraps a SQLite database connection.
type DB struct {
	conn *sql.DB
}

// Open opens a SQLite database at the given path.
// Use ":memory:" for an in-memory database.
func Open(path string) (*DB, error) {
	conn, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Enable foreign keys
	if _, err := conn.Exec("PRAGMA foreign_keys = ON"); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	return &DB{conn: conn}, nil
}

// Close closes the database connection.
func (db *DB) Close() error {
	if db.conn != nil {
		return db.conn.Close()
	}
	return nil
}

// Migrate creates the database tables if they don't exist.
func (db *DB) Migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS tracks (
		id INTEGER PRIMARY KEY,
		branch TEXT NOT NULL,
		remote_url TEXT NOT NULL,
		head_sha TEXT NOT NULL,
		type TEXT NOT NULL CHECK (type IN ('worktree', 'devbox')),
		path TEXT,
		devbox_name TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		last_accessed TIMESTAMP,
		UNIQUE(remote_url, branch)
	);
	`
	if _, err := db.conn.Exec(schema); err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}
	return nil
}

// InsertTrack inserts a new track into the database.
func (db *DB) InsertTrack(track Track) error {
	query := `
	INSERT INTO tracks (branch, remote_url, head_sha, type, path, devbox_name, created_at, last_accessed)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	createdAt := track.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now()
	}

	_, err := db.conn.Exec(query,
		track.Branch,
		track.RemoteURL,
		track.HeadSHA,
		string(track.Type),
		track.Path,
		track.DevboxName,
		createdAt,
		track.LastAccessed,
	)
	if err != nil {
		return fmt.Errorf("failed to insert track: %w", err)
	}
	return nil
}

// UpdateTrack updates an existing track in the database.
func (db *DB) UpdateTrack(track Track) error {
	query := `
	UPDATE tracks
	SET head_sha = ?, type = ?, path = ?, devbox_name = ?, last_accessed = ?
	WHERE remote_url = ? AND branch = ?
	`

	result, err := db.conn.Exec(query,
		track.HeadSHA,
		string(track.Type),
		track.Path,
		track.DevboxName,
		track.LastAccessed,
		track.RemoteURL,
		track.Branch,
	)
	if err != nil {
		return fmt.Errorf("failed to update track: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return errors.New("track not found")
	}
	return nil
}

// DeleteTrack deletes a track by remote URL and branch.
func (db *DB) DeleteTrack(remoteURL, branch string) error {
	query := `DELETE FROM tracks WHERE remote_url = ? AND branch = ?`

	result, err := db.conn.Exec(query, remoteURL, branch)
	if err != nil {
		return fmt.Errorf("failed to delete track: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return errors.New("track not found")
	}
	return nil
}

// GetTrack retrieves a track by remote URL and branch.
func (db *DB) GetTrack(remoteURL, branch string) (*Track, error) {
	query := `
	SELECT id, branch, remote_url, head_sha, type, path, devbox_name, created_at, last_accessed
	FROM tracks
	WHERE remote_url = ? AND branch = ?
	`

	row := db.conn.QueryRow(query, remoteURL, branch)
	return scanTrack(row)
}

// ListTracks retrieves all tracks from the database.
func (db *DB) ListTracks() ([]Track, error) {
	query := `
	SELECT id, branch, remote_url, head_sha, type, path, devbox_name, created_at, last_accessed
	FROM tracks
	ORDER BY last_accessed DESC NULLS LAST, created_at DESC
	`

	rows, err := db.conn.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to list tracks: %w", err)
	}
	defer rows.Close()

	tracks := make([]Track, 0)
	for rows.Next() {
		track, err := scanTrackRow(rows)
		if err != nil {
			return nil, err
		}
		tracks = append(tracks, *track)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating tracks: %w", err)
	}

	return tracks, nil
}

// UpdateLastAccessed updates the last_accessed timestamp for a track.
func (db *DB) UpdateLastAccessed(remoteURL, branch string) error {
	query := `UPDATE tracks SET last_accessed = ? WHERE remote_url = ? AND branch = ?`

	result, err := db.conn.Exec(query, time.Now(), remoteURL, branch)
	if err != nil {
		return fmt.Errorf("failed to update last accessed: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return errors.New("track not found")
	}
	return nil
}

// UpdateHeadSHA updates the head SHA for a track.
func (db *DB) UpdateHeadSHA(remoteURL, branch, sha string) error {
	query := `UPDATE tracks SET head_sha = ? WHERE remote_url = ? AND branch = ?`

	result, err := db.conn.Exec(query, sha, remoteURL, branch)
	if err != nil {
		return fmt.Errorf("failed to update head SHA: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return errors.New("track not found")
	}
	return nil
}

// rowScanner is an interface satisfied by both *sql.Row and *sql.Rows.
type rowScanner interface {
	Scan(dest ...any) error
}

func scanTrack(row *sql.Row) (*Track, error) {
	var track Track
	var trackType string
	var createdAt string
	var lastAccessed sql.NullString

	err := row.Scan(
		&track.ID,
		&track.Branch,
		&track.RemoteURL,
		&track.HeadSHA,
		&trackType,
		&track.Path,
		&track.DevboxName,
		&createdAt,
		&lastAccessed,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to scan track: %w", err)
	}

	track.Type = TrackType(trackType)

	// Parse created_at
	t, err := time.Parse("2006-01-02 15:04:05", createdAt)
	if err != nil {
		// Try alternative format
		t, err = time.Parse(time.RFC3339, createdAt)
		if err != nil {
			return nil, fmt.Errorf("failed to parse created_at: %w", err)
		}
	}
	track.CreatedAt = t

	// Parse last_accessed if present
	if lastAccessed.Valid {
		t, err := time.Parse("2006-01-02 15:04:05", lastAccessed.String)
		if err != nil {
			t, err = time.Parse(time.RFC3339, lastAccessed.String)
			if err != nil {
				return nil, fmt.Errorf("failed to parse last_accessed: %w", err)
			}
		}
		track.LastAccessed = &t
	}

	return &track, nil
}

func scanTrackRow(rows *sql.Rows) (*Track, error) {
	var track Track
	var trackType string
	var createdAt string
	var lastAccessed sql.NullString

	err := rows.Scan(
		&track.ID,
		&track.Branch,
		&track.RemoteURL,
		&track.HeadSHA,
		&trackType,
		&track.Path,
		&track.DevboxName,
		&createdAt,
		&lastAccessed,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to scan track: %w", err)
	}

	track.Type = TrackType(trackType)

	// Parse created_at
	t, err := time.Parse("2006-01-02 15:04:05", createdAt)
	if err != nil {
		t, err = time.Parse(time.RFC3339, createdAt)
		if err != nil {
			return nil, fmt.Errorf("failed to parse created_at: %w", err)
		}
	}
	track.CreatedAt = t

	// Parse last_accessed if present
	if lastAccessed.Valid {
		t, err := time.Parse("2006-01-02 15:04:05", lastAccessed.String)
		if err != nil {
			t, err = time.Parse(time.RFC3339, lastAccessed.String)
			if err != nil {
				return nil, fmt.Errorf("failed to parse last_accessed: %w", err)
			}
		}
		track.LastAccessed = &t
	}

	return &track, nil
}
