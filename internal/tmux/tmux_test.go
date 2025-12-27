package tmux

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

// MockRunner is a mock implementation of CommandRunner for testing.
type MockRunner struct {
	// RunFunc is called for Run() calls
	RunFunc func(name string, args ...string) (string, error)
	// ExecFunc is called for Exec() calls
	ExecFunc func(name string, args ...string) error
	// Calls records all calls made
	Calls []MockCall
}

// MockCall records a single call to the runner.
type MockCall struct {
	Method string
	Name   string
	Args   []string
}

func (m *MockRunner) Run(name string, args ...string) (string, error) {
	m.Calls = append(m.Calls, MockCall{Method: "Run", Name: name, Args: args})
	if m.RunFunc != nil {
		return m.RunFunc(name, args...)
	}
	return "", nil
}

func (m *MockRunner) Exec(name string, args ...string) error {
	m.Calls = append(m.Calls, MockCall{Method: "Exec", Name: name, Args: args})
	if m.ExecFunc != nil {
		return m.ExecFunc(name, args...)
	}
	return nil
}

func TestSessionExists(t *testing.T) {
	tests := []struct {
		name       string
		session    string
		runFunc    func(name string, args ...string) (string, error)
		wantExists bool
		wantErr    bool
	}{
		{
			name:    "session exists",
			session: "mysession",
			runFunc: func(name string, args ...string) (string, error) {
				return "", nil
			},
			wantExists: true,
			wantErr:    false,
		},
		{
			name:    "session does not exist",
			session: "nosession",
			runFunc: func(name string, args ...string) (string, error) {
				return "", fmt.Errorf("tmux has-session failed: exit status 1\nstderr: can't find session: nosession")
			},
			wantExists: false,
			wantErr:    false,
		},
		{
			name:    "tmux error",
			session: "mysession",
			runFunc: func(name string, args ...string) (string, error) {
				return "", fmt.Errorf("tmux: connection refused")
			},
			wantExists: false,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &MockRunner{RunFunc: tt.runFunc}
			SetRunner(mock)
			defer ResetRunner()

			got, err := SessionExists(tt.session)
			if (err != nil) != tt.wantErr {
				t.Errorf("SessionExists() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.wantExists {
				t.Errorf("SessionExists() = %v, want %v", got, tt.wantExists)
			}

			// Verify the command was called correctly
			if len(mock.Calls) != 1 {
				t.Errorf("Expected 1 call, got %d", len(mock.Calls))
				return
			}
			call := mock.Calls[0]
			if call.Name != "tmux" {
				t.Errorf("Expected tmux command, got %s", call.Name)
			}
			expectedArgs := []string{"has-session", "-t", tt.session}
			if !slicesEqual(call.Args, expectedArgs) {
				t.Errorf("Expected args %v, got %v", expectedArgs, call.Args)
			}
		})
	}
}

func TestCreateSession(t *testing.T) {
	mock := &MockRunner{}
	SetRunner(mock)
	defer ResetRunner()

	err := CreateSession("newsession")
	if err != nil {
		t.Errorf("CreateSession() error = %v", err)
	}

	if len(mock.Calls) != 1 {
		t.Fatalf("Expected 1 call, got %d", len(mock.Calls))
	}
	call := mock.Calls[0]
	expectedArgs := []string{"new-session", "-d", "-s", "newsession"}
	if !slicesEqual(call.Args, expectedArgs) {
		t.Errorf("Expected args %v, got %v", expectedArgs, call.Args)
	}
}

func TestWindowExists(t *testing.T) {
	tests := []struct {
		name       string
		session    string
		windowName string
		runFunc    func(name string, args ...string) (string, error)
		wantExists bool
		wantErr    bool
	}{
		{
			name:       "window exists",
			session:    "mysession",
			windowName: "mywindow",
			runFunc: func(name string, args ...string) (string, error) {
				return "bash\nmywindow\nzsh", nil
			},
			wantExists: true,
			wantErr:    false,
		},
		{
			name:       "window does not exist",
			session:    "mysession",
			windowName: "nowindow",
			runFunc: func(name string, args ...string) (string, error) {
				return "bash\nmywindow\nzsh", nil
			},
			wantExists: false,
			wantErr:    false,
		},
		{
			name:       "session does not exist",
			session:    "nosession",
			windowName: "mywindow",
			runFunc: func(name string, args ...string) (string, error) {
				return "", fmt.Errorf("can't find session: nosession")
			},
			wantExists: false,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &MockRunner{RunFunc: tt.runFunc}
			SetRunner(mock)
			defer ResetRunner()

			got, err := WindowExists(tt.session, tt.windowName)
			if (err != nil) != tt.wantErr {
				t.Errorf("WindowExists() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.wantExists {
				t.Errorf("WindowExists() = %v, want %v", got, tt.wantExists)
			}
		})
	}
}

func TestCreateWindow(t *testing.T) {
	tests := []struct {
		name         string
		session      string
		windowName   string
		startDir     string
		expectedArgs []string
	}{
		{
			name:         "without start dir",
			session:      "mysession",
			windowName:   "mywindow",
			startDir:     "",
			expectedArgs: []string{"new-window", "-t", "mysession", "-n", "mywindow"},
		},
		{
			name:         "with start dir",
			session:      "mysession",
			windowName:   "mywindow",
			startDir:     "/home/user/project",
			expectedArgs: []string{"new-window", "-t", "mysession", "-n", "mywindow", "-c", "/home/user/project"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &MockRunner{}
			SetRunner(mock)
			defer ResetRunner()

			err := CreateWindow(tt.session, tt.windowName, tt.startDir)
			if err != nil {
				t.Errorf("CreateWindow() error = %v", err)
			}

			if len(mock.Calls) != 1 {
				t.Fatalf("Expected 1 call, got %d", len(mock.Calls))
			}
			call := mock.Calls[0]
			if !slicesEqual(call.Args, tt.expectedArgs) {
				t.Errorf("Expected args %v, got %v", tt.expectedArgs, call.Args)
			}
		})
	}
}

func TestSwitchToWindow(t *testing.T) {
	mock := &MockRunner{}
	SetRunner(mock)
	defer ResetRunner()

	err := SwitchToWindow("mysession", "mywindow")
	if err != nil {
		t.Errorf("SwitchToWindow() error = %v", err)
	}

	if len(mock.Calls) != 1 {
		t.Fatalf("Expected 1 call, got %d", len(mock.Calls))
	}
	call := mock.Calls[0]
	expectedArgs := []string{"switch-client", "-t", "mysession:mywindow"}
	if !slicesEqual(call.Args, expectedArgs) {
		t.Errorf("Expected args %v, got %v", expectedArgs, call.Args)
	}
}

func TestAttachSession(t *testing.T) {
	mock := &MockRunner{}
	SetRunner(mock)
	defer ResetRunner()

	err := AttachSession("mysession")
	if err != nil {
		t.Errorf("AttachSession() error = %v", err)
	}

	if len(mock.Calls) != 1 {
		t.Fatalf("Expected 1 call, got %d", len(mock.Calls))
	}
	call := mock.Calls[0]
	if call.Method != "Exec" {
		t.Errorf("Expected Exec method, got %s", call.Method)
	}
	expectedArgs := []string{"attach-session", "-t", "mysession"}
	if !slicesEqual(call.Args, expectedArgs) {
		t.Errorf("Expected args %v, got %v", expectedArgs, call.Args)
	}
}

func TestIsInsideTmux(t *testing.T) {
	// Save original value
	originalTmux := os.Getenv("TMUX")
	defer os.Setenv("TMUX", originalTmux)

	// Test when inside tmux
	os.Setenv("TMUX", "/tmp/tmux-1000/default,12345,0")
	if !IsInsideTmux() {
		t.Error("Expected IsInsideTmux() to return true when TMUX is set")
	}

	// Test when not inside tmux
	os.Unsetenv("TMUX")
	if IsInsideTmux() {
		t.Error("Expected IsInsideTmux() to return false when TMUX is not set")
	}
}

func TestRunInWindow(t *testing.T) {
	mock := &MockRunner{}
	SetRunner(mock)
	defer ResetRunner()

	err := RunInWindow("mysession", "mywindow", "ls -la")
	if err != nil {
		t.Errorf("RunInWindow() error = %v", err)
	}

	if len(mock.Calls) != 1 {
		t.Fatalf("Expected 1 call, got %d", len(mock.Calls))
	}
	call := mock.Calls[0]
	expectedArgs := []string{"send-keys", "-t", "mysession:mywindow", "ls -la", "Enter"}
	if !slicesEqual(call.Args, expectedArgs) {
		t.Errorf("Expected args %v, got %v", expectedArgs, call.Args)
	}
}

func TestListSessions(t *testing.T) {
	tests := []struct {
		name     string
		runFunc  func(name string, args ...string) (string, error)
		want     []string
		wantErr  bool
	}{
		{
			name: "multiple sessions",
			runFunc: func(name string, args ...string) (string, error) {
				return "session1\nsession2\nsession3", nil
			},
			want:    []string{"session1", "session2", "session3"},
			wantErr: false,
		},
		{
			name: "no server running",
			runFunc: func(name string, args ...string) (string, error) {
				return "", fmt.Errorf("no server running on /tmp/tmux-1000/default")
			},
			want:    []string{},
			wantErr: false,
		},
		{
			name: "empty output",
			runFunc: func(name string, args ...string) (string, error) {
				return "", nil
			},
			want:    []string{},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &MockRunner{RunFunc: tt.runFunc}
			SetRunner(mock)
			defer ResetRunner()

			got, err := ListSessions()
			if (err != nil) != tt.wantErr {
				t.Errorf("ListSessions() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !slicesEqual(got, tt.want) {
				t.Errorf("ListSessions() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestListWindows(t *testing.T) {
	mock := &MockRunner{
		RunFunc: func(name string, args ...string) (string, error) {
			return "window1\nwindow2", nil
		},
	}
	SetRunner(mock)
	defer ResetRunner()

	got, err := ListWindows("mysession")
	if err != nil {
		t.Errorf("ListWindows() error = %v", err)
	}

	expected := []string{"window1", "window2"}
	if !slicesEqual(got, expected) {
		t.Errorf("ListWindows() = %v, want %v", got, expected)
	}
}

func TestKillWindow(t *testing.T) {
	mock := &MockRunner{}
	SetRunner(mock)
	defer ResetRunner()

	err := KillWindow("mysession", "mywindow")
	if err != nil {
		t.Errorf("KillWindow() error = %v", err)
	}

	if len(mock.Calls) != 1 {
		t.Fatalf("Expected 1 call, got %d", len(mock.Calls))
	}
	call := mock.Calls[0]
	expectedArgs := []string{"kill-window", "-t", "mysession:mywindow"}
	if !slicesEqual(call.Args, expectedArgs) {
		t.Errorf("Expected args %v, got %v", expectedArgs, call.Args)
	}
}

func TestKillSession(t *testing.T) {
	mock := &MockRunner{}
	SetRunner(mock)
	defer ResetRunner()

	err := KillSession("mysession")
	if err != nil {
		t.Errorf("KillSession() error = %v", err)
	}

	if len(mock.Calls) != 1 {
		t.Fatalf("Expected 1 call, got %d", len(mock.Calls))
	}
	call := mock.Calls[0]
	expectedArgs := []string{"kill-session", "-t", "mysession"}
	if !slicesEqual(call.Args, expectedArgs) {
		t.Errorf("Expected args %v, got %v", expectedArgs, call.Args)
	}
}

func TestSelectWindow(t *testing.T) {
	mock := &MockRunner{}
	SetRunner(mock)
	defer ResetRunner()

	err := SelectWindow("mysession", "mywindow")
	if err != nil {
		t.Errorf("SelectWindow() error = %v", err)
	}

	if len(mock.Calls) != 1 {
		t.Fatalf("Expected 1 call, got %d", len(mock.Calls))
	}
	call := mock.Calls[0]
	expectedArgs := []string{"select-window", "-t", "mysession:mywindow"}
	if !slicesEqual(call.Args, expectedArgs) {
		t.Errorf("Expected args %v, got %v", expectedArgs, call.Args)
	}
}

// slicesEqual compares two string slices.
func slicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// TestCommandConstruction verifies that commands are constructed correctly.
func TestCommandConstruction(t *testing.T) {
	tests := []struct {
		name         string
		callFunc     func()
		expectedCmd  string
		expectedArgs []string
	}{
		{
			name:         "has-session",
			callFunc:     func() { SessionExists("test") },
			expectedCmd:  "tmux",
			expectedArgs: []string{"has-session", "-t", "test"},
		},
		{
			name:         "new-session detached",
			callFunc:     func() { CreateSession("test") },
			expectedCmd:  "tmux",
			expectedArgs: []string{"new-session", "-d", "-s", "test"},
		},
		{
			name:         "list-windows format",
			callFunc:     func() { WindowExists("sess", "win") },
			expectedCmd:  "tmux",
			expectedArgs: []string{"list-windows", "-t", "sess", "-F", "#{window_name}"},
		},
		{
			name:         "new-window with dir",
			callFunc:     func() { CreateWindow("sess", "win", "/tmp") },
			expectedCmd:  "tmux",
			expectedArgs: []string{"new-window", "-t", "sess", "-n", "win", "-c", "/tmp"},
		},
		{
			name:         "switch-client target format",
			callFunc:     func() { SwitchToWindow("sess", "win") },
			expectedCmd:  "tmux",
			expectedArgs: []string{"switch-client", "-t", "sess:win"},
		},
		{
			name:         "send-keys with enter",
			callFunc:     func() { RunInWindow("sess", "win", "echo hello") },
			expectedCmd:  "tmux",
			expectedArgs: []string{"send-keys", "-t", "sess:win", "echo hello", "Enter"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &MockRunner{}
			SetRunner(mock)
			defer ResetRunner()

			tt.callFunc()

			if len(mock.Calls) == 0 {
				t.Fatal("Expected at least 1 call")
			}
			call := mock.Calls[0]
			if call.Name != tt.expectedCmd {
				t.Errorf("Expected command %s, got %s", tt.expectedCmd, call.Name)
			}
			if !slicesEqual(call.Args, tt.expectedArgs) {
				t.Errorf("Expected args %v, got %v", tt.expectedArgs, call.Args)
			}
		})
	}
}

// TestSpecialCharactersInNames tests handling of special characters.
func TestSpecialCharactersInNames(t *testing.T) {
	mock := &MockRunner{}
	SetRunner(mock)
	defer ResetRunner()

	// Branch names with special chars
	specialNames := []string{
		"feature/my-branch",
		"fix_bug_123",
		"release-v1.0.0",
	}

	for _, name := range specialNames {
		mock.Calls = nil // Reset calls

		err := CreateSession(name)
		if err != nil {
			t.Errorf("CreateSession(%q) failed: %v", name, err)
		}

		call := mock.Calls[0]
		// Verify the name is passed through unchanged
		if call.Args[3] != name {
			t.Errorf("Session name not preserved: got %q, want %q", call.Args[3], name)
		}
	}
}

// TestErrorPropagation tests that errors are properly propagated.
func TestErrorPropagation(t *testing.T) {
	expectedErr := fmt.Errorf("connection refused")
	mock := &MockRunner{
		RunFunc: func(name string, args ...string) (string, error) {
			return "", expectedErr
		},
	}
	SetRunner(mock)
	defer ResetRunner()

	_, err := ListSessions()
	if err == nil {
		t.Error("Expected error to be propagated")
	}
	if !strings.Contains(err.Error(), "connection refused") {
		t.Errorf("Expected error message to contain 'connection refused', got: %v", err)
	}
}
