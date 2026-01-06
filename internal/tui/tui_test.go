package tui

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/laurent/trak/internal/db"
	"github.com/laurent/trak/internal/ops"
	"github.com/laurent/trak/internal/track"
)

func TestNew(t *testing.T) {
	m := New(nil, "test-repo")

	if m.repoName != "test-repo" {
		t.Errorf("expected repoName to be 'test-repo', got '%s'", m.repoName)
	}

	if m.view != ViewMain {
		t.Errorf("expected initial view to be ViewMain, got %v", m.view)
	}

	if !m.loading {
		t.Error("expected loading to be true initially")
	}

	if m.width != 80 || m.height != 24 {
		t.Errorf("expected default dimensions 80x24, got %dx%d", m.width, m.height)
	}
}

func TestDefaultKeyMap(t *testing.T) {
	km := DefaultKeyMap()

	// Verify all keybindings are defined (non-empty)
	bindings := []struct {
		name    string
		binding key.Binding
	}{
		{"Up", km.Up},
		{"Down", km.Down},
		{"Enter", km.Enter},
		{"Refresh", km.Refresh},
		{"Browse", km.Browse},
		{"New", km.New},
		{"Delete", km.Delete},
		{"Sync", km.Sync},
		{"AI", km.AI},
		{"Back", km.Back},
		{"Quit", km.Quit},
		{"Help", km.Help},
	}

	for _, b := range bindings {
		t.Run(b.name, func(t *testing.T) {
			// Verify the binding has a help text defined
			help := b.binding.Help()
			if help.Key == "" {
				t.Errorf("binding %s has no key help", b.name)
			}
		})
	}
}

func TestKeyMapShortHelp(t *testing.T) {
	km := DefaultKeyMap()
	help := km.ShortHelp()

	if len(help) != 5 {
		t.Errorf("expected 5 short help bindings, got %d", len(help))
	}
}

func TestKeyMapFullHelp(t *testing.T) {
	km := DefaultKeyMap()
	help := km.FullHelp()

	if len(help) != 4 {
		t.Errorf("expected 4 full help rows, got %d", len(help))
	}

	// Row sizes: {Up, Down, Enter}, {Refresh, Browse, New}, {Sync, AI, Delete, ForceDelete}, {Back, Quit, Help}
	expectedSizes := []int{3, 3, 4, 3}
	for i, row := range help {
		if len(row) != expectedSizes[i] {
			t.Errorf("expected %d bindings in row %d, got %d", expectedSizes[i], i, len(row))
		}
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"short", 10, "short"},
		{"exactly10!", 10, "exactly10!"},
		{"this is a long string", 10, "this is..."},
		{"abc", 3, "abc"},
		{"abcd", 3, "..."},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := truncate(tt.input, tt.maxLen)
			if result != tt.expected {
				t.Errorf("truncate(%q, %d) = %q, want %q", tt.input, tt.maxLen, result, tt.expected)
			}
		})
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		duration time.Duration
		expected string
	}{
		{30 * time.Second, "now"},
		{5 * time.Minute, "5m"},
		{90 * time.Minute, "1h"},
		{3 * time.Hour, "3h"},
		{25 * time.Hour, "1d"},
		{5 * 24 * time.Hour, "5d"},
		{10 * 24 * time.Hour, "1w"},
		{21 * 24 * time.Hour, "3w"},
		{45 * 24 * time.Hour, "1mo"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := formatDuration(tt.duration)
			if result != tt.expected {
				t.Errorf("formatDuration(%v) = %q, want %q", tt.duration, result, tt.expected)
			}
		})
	}
}

func TestMinInt(t *testing.T) {
	tests := []struct {
		a, b     int
		expected int
	}{
		{1, 2, 1},
		{2, 1, 1},
		{5, 5, 5},
		{-1, 0, -1},
		{0, -1, -1},
	}

	for _, tt := range tests {
		result := minInt(tt.a, tt.b)
		if result != tt.expected {
			t.Errorf("minInt(%d, %d) = %d, want %d", tt.a, tt.b, result, tt.expected)
		}
	}
}

func TestModelInit(t *testing.T) {
	m := New(nil, "test")
	cmd := m.Init()

	if cmd == nil {
		t.Error("expected Init to return a command")
	}
}

func TestModelUpdateQuit(t *testing.T) {
	m := New(nil, "test")

	// Send quit key
	newModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})

	model := newModel.(Model)
	if !model.quitting {
		t.Error("expected model to be quitting after 'q' press")
	}

	if cmd == nil {
		t.Error("expected quit command to be returned")
	}
}

func TestModelUpdateHelp(t *testing.T) {
	m := New(nil, "test")
	m.loading = false

	// Send help key
	initialShowAll := m.help.ShowAll
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})

	model := newModel.(Model)
	if model.help.ShowAll == initialShowAll {
		t.Error("expected help.ShowAll to toggle")
	}
}

func TestModelUpdateWindowSize(t *testing.T) {
	m := New(nil, "test")

	newModel, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})

	model := newModel.(Model)
	if model.width != 120 || model.height != 40 {
		t.Errorf("expected dimensions 120x40, got %dx%d", model.width, model.height)
	}
}

func TestModelUpdateTracksLoaded(t *testing.T) {
	m := New(nil, "test")

	now := time.Now()
	tracks := []ops.TrackWithStatus{
		{
			Track: db.Track{
				Branch:    "feature-1",
				RemoteURL: "owner/repo",
				HeadSHA:   "abc123",
				Type:      db.TrackTypeWorktree,
				CreatedAt: now,
			},
			Status: track.TrackStatus{},
		},
	}

	newModel, _ := m.Update(tracksLoadedMsg{tracks: tracks})

	model := newModel.(Model)
	if model.loading {
		t.Error("expected loading to be false after tracks loaded")
	}

	if len(model.tracks) != 1 {
		t.Errorf("expected 1 track, got %d", len(model.tracks))
	}
}

func TestModelUpdateRemoteBranchesLoaded(t *testing.T) {
	m := New(nil, "test")
	m.view = ViewRemoteBrowser

	branches := []ops.RemoteBranch{
		{Name: "feature-1", HasPR: true, PRNumber: 42},
		{Name: "feature-2", HasPR: false},
	}

	newModel, _ := m.Update(remoteBranchesLoadedMsg{branches: branches})

	model := newModel.(Model)
	if model.loading {
		t.Error("expected loading to be false after branches loaded")
	}

	if len(model.remoteBranches) != 2 {
		t.Errorf("expected 2 remote branches, got %d", len(model.remoteBranches))
	}
}

func TestModelUpdateError(t *testing.T) {
	m := New(nil, "test")

	newModel, _ := m.Update(errMsg{err: fmt.Errorf("test error")})

	model := newModel.(Model)
	if model.err == nil {
		t.Error("expected error to be set")
	}

	if model.notification != "test error" {
		t.Errorf("expected notification 'test error', got '%s'", model.notification)
	}
}

func TestModelUpdateOperationComplete(t *testing.T) {
	m := New(nil, "test")

	// Test successful operation
	newModel, _ := m.Update(operationCompleteMsg{message: "Success", isError: false})

	model := newModel.(Model)
	if model.notification != "Success" {
		t.Errorf("expected notification 'Success', got '%s'", model.notification)
	}
	if model.err != nil {
		t.Error("expected no error for successful operation")
	}

	// Test error operation
	m2 := New(nil, "test")
	newModel2, _ := m2.Update(operationCompleteMsg{message: "Failed", isError: true})

	model2 := newModel2.(Model)
	if model2.err == nil {
		t.Error("expected error to be set for failed operation")
	}
}

func TestModelUpdateBrowse(t *testing.T) {
	m := New(nil, "test")
	m.loading = false

	newModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}})

	model := newModel.(Model)
	if model.view != ViewRemoteBrowser {
		t.Error("expected view to switch to ViewRemoteBrowser")
	}

	if !model.loading {
		t.Error("expected loading to be true when switching to remote browser")
	}

	if cmd == nil {
		t.Error("expected command to load remote branches")
	}
}

func TestModelUpdateBack(t *testing.T) {
	m := New(nil, "test")
	m.view = ViewRemoteBrowser
	m.loading = false

	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyEscape})

	model := newModel.(Model)
	if model.view != ViewMain {
		t.Error("expected view to switch back to ViewMain")
	}
}

func TestModelUpdateRefresh(t *testing.T) {
	m := New(nil, "test")
	m.loading = false

	newModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})

	model := newModel.(Model)
	if !model.loading {
		t.Error("expected loading to be true on refresh")
	}

	if cmd == nil {
		t.Error("expected command to be returned for refresh")
	}
}

func TestModelViewQuitting(t *testing.T) {
	m := New(nil, "test")
	m.quitting = true

	view := m.View()
	if view != "" {
		t.Errorf("expected empty view when quitting, got '%s'", view)
	}
}

func TestModelViewMainEmpty(t *testing.T) {
	m := New(nil, "test")
	m.loading = false

	view := m.View()

	if view == "" {
		t.Error("expected non-empty view")
	}

	// Should contain the empty state message
	if !strings.Contains(view, "No tracks yet") {
		t.Error("expected 'No tracks yet' message in empty view")
	}
}

func TestModelViewWithTracks(t *testing.T) {
	m := New(nil, "test")
	m.loading = false

	now := time.Now()
	m.tracks = []ops.TrackWithStatus{
		{
			Track: db.Track{
				Branch:    "feature-test",
				RemoteURL: "owner/repo",
				Type:      db.TrackTypeWorktree,
				CreatedAt: now,
			},
			Status: track.TrackStatus{
				GitStatus: track.GitStatus{Clean: true},
			},
		},
	}
	m.table = m.buildMainTable()

	view := m.View()

	if !strings.Contains(view, "trak") {
		t.Error("expected 'trak' in header")
	}
}

func TestModelViewWithNotification(t *testing.T) {
	m := New(nil, "test")
	m.loading = false
	m.notification = "Test notification"
	m.notifyTime = time.Now()

	view := m.View()

	if !strings.Contains(view, "Test notification") {
		t.Error("expected notification to appear in view")
	}
}

func TestBuildMainTable(t *testing.T) {
	m := New(nil, "test")
	m.height = 30

	now := time.Now()
	path := "/path/to/worktree"
	m.tracks = []ops.TrackWithStatus{
		{
			Track: db.Track{
				Branch:    "feature-1",
				RemoteURL: "owner/repo",
				HeadSHA:   "abc123",
				Type:      db.TrackTypeWorktree,
				Path:      &path,
				CreatedAt: now,
			},
			Status: track.TrackStatus{
				GitStatus: track.GitStatus{
					Clean:      true,
					Ahead:      true,
					AheadCount: 2,
				},
				PR: &track.PRStatus{Number: 42},
				CI: &track.CIStatus{Passing: true},
			},
		},
	}

	tbl := m.buildMainTable()

	// Table should be built without errors
	if tbl.Cursor() != 0 {
		t.Error("expected cursor at position 0")
	}
}

func TestBuildRemoteTable(t *testing.T) {
	m := New(nil, "test")
	m.height = 30

	m.remoteBranches = []ops.RemoteBranch{
		{Name: "feature-1", HasPR: true, PRNumber: 42, Age: 2 * time.Hour},
		{Name: "feature-2", HasPR: false, Age: 5 * 24 * time.Hour},
	}

	tbl := m.buildRemoteTable()

	// Table should be built without errors
	if tbl.Cursor() != 0 {
		t.Error("expected cursor at position 0")
	}
}

func TestRenderMainViewEmpty(t *testing.T) {
	m := New(nil, "test")
	m.tracks = nil

	view := m.renderMainView()

	if !strings.Contains(view, "No tracks yet") {
		t.Error("expected empty state message")
	}
}

func TestRenderRemoteBrowserViewEmpty(t *testing.T) {
	m := New(nil, "test")
	m.remoteBranches = nil

	view := m.renderRemoteBrowserView()

	if !strings.Contains(view, "No remote branches") {
		t.Error("expected empty state message")
	}
}

func TestFormatAge(t *testing.T) {
	// formatAge is a wrapper around formatDuration, just verify it works
	past := time.Now().Add(-2 * time.Hour)
	result := formatAge(past)

	if result != "2h" {
		t.Errorf("expected '2h', got '%s'", result)
	}
}

func TestViewConstants(t *testing.T) {
	if ViewMain != 0 {
		t.Errorf("expected ViewMain to be 0, got %d", ViewMain)
	}
	if ViewRemoteBrowser != 1 {
		t.Errorf("expected ViewRemoteBrowser to be 1, got %d", ViewRemoteBrowser)
	}
}

func TestMessagesTypes(t *testing.T) {
	// Test that message types can be created
	_ = tracksLoadedMsg{tracks: nil}
	_ = remoteBranchesLoadedMsg{branches: nil}
	_ = operationCompleteMsg{message: "test", isError: false}
	_ = errMsg{err: fmt.Errorf("test")}
}
