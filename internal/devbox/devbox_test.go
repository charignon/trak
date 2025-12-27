package devbox

import (
	"fmt"
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

func TestCreate(t *testing.T) {
	mock := &MockRunner{}
	SetRunner(mock)
	defer ResetRunner()

	err := Create("my-devbox", "https://github.com/user/repo", "feature-branch")
	if err != nil {
		t.Errorf("Create() error = %v", err)
	}

	if len(mock.Calls) != 1 {
		t.Fatalf("Expected 1 call, got %d", len(mock.Calls))
	}
	call := mock.Calls[0]
	if call.Name != "devbox" {
		t.Errorf("Expected devbox command, got %s", call.Name)
	}
	expectedArgs := []string{"create", "--name", "my-devbox", "--repo", "https://github.com/user/repo", "--branch", "feature-branch"}
	if !slicesEqual(call.Args, expectedArgs) {
		t.Errorf("Expected args %v, got %v", expectedArgs, call.Args)
	}
}

func TestCreateError(t *testing.T) {
	mock := &MockRunner{
		RunFunc: func(name string, args ...string) (string, error) {
			return "", fmt.Errorf("failed to create devbox: quota exceeded")
		},
	}
	SetRunner(mock)
	defer ResetRunner()

	err := Create("my-devbox", "https://github.com/user/repo", "main")
	if err == nil {
		t.Error("Expected error, got nil")
	}
	if !strings.Contains(err.Error(), "quota exceeded") {
		t.Errorf("Expected error to contain 'quota exceeded', got: %v", err)
	}
}

func TestDelete(t *testing.T) {
	mock := &MockRunner{}
	SetRunner(mock)
	defer ResetRunner()

	err := Delete("my-devbox")
	if err != nil {
		t.Errorf("Delete() error = %v", err)
	}

	if len(mock.Calls) != 1 {
		t.Fatalf("Expected 1 call, got %d", len(mock.Calls))
	}
	call := mock.Calls[0]
	expectedArgs := []string{"delete", "my-devbox"}
	if !slicesEqual(call.Args, expectedArgs) {
		t.Errorf("Expected args %v, got %v", expectedArgs, call.Args)
	}
}

func TestDeleteError(t *testing.T) {
	mock := &MockRunner{
		RunFunc: func(name string, args ...string) (string, error) {
			return "", fmt.Errorf("devbox not found: my-devbox")
		},
	}
	SetRunner(mock)
	defer ResetRunner()

	err := Delete("my-devbox")
	if err == nil {
		t.Error("Expected error, got nil")
	}
}

func TestExists(t *testing.T) {
	tests := []struct {
		name       string
		devboxName string
		runFunc    func(name string, args ...string) (string, error)
		wantExists bool
		wantErr    bool
	}{
		{
			name:       "devbox exists",
			devboxName: "my-devbox",
			runFunc: func(name string, args ...string) (string, error) {
				return `{"devboxes": [{"name": "my-devbox", "status": "running"}, {"name": "other", "status": "stopped"}]}`, nil
			},
			wantExists: true,
			wantErr:    false,
		},
		{
			name:       "devbox does not exist",
			devboxName: "missing-devbox",
			runFunc: func(name string, args ...string) (string, error) {
				return `{"devboxes": [{"name": "my-devbox", "status": "running"}]}`, nil
			},
			wantExists: false,
			wantErr:    false,
		},
		{
			name:       "empty list",
			devboxName: "my-devbox",
			runFunc: func(name string, args ...string) (string, error) {
				return `{"devboxes": []}`, nil
			},
			wantExists: false,
			wantErr:    false,
		},
		{
			name:       "no devboxes error",
			devboxName: "my-devbox",
			runFunc: func(name string, args ...string) (string, error) {
				return "", fmt.Errorf("no devboxes found")
			},
			wantExists: false,
			wantErr:    false,
		},
		{
			name:       "list error",
			devboxName: "my-devbox",
			runFunc: func(name string, args ...string) (string, error) {
				return "", fmt.Errorf("connection refused")
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

			got, err := Exists(tt.devboxName)
			if (err != nil) != tt.wantErr {
				t.Errorf("Exists() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.wantExists {
				t.Errorf("Exists() = %v, want %v", got, tt.wantExists)
			}
		})
	}
}

func TestList(t *testing.T) {
	tests := []struct {
		name    string
		runFunc func(name string, args ...string) (string, error)
		want    []Devbox
		wantErr bool
	}{
		{
			name: "multiple devboxes",
			runFunc: func(name string, args ...string) (string, error) {
				return `{"devboxes": [{"name": "dev1", "status": "running"}, {"name": "dev2", "status": "stopped"}]}`, nil
			},
			want: []Devbox{
				{Name: "dev1", Status: "running"},
				{Name: "dev2", Status: "stopped"},
			},
			wantErr: false,
		},
		{
			name: "empty list",
			runFunc: func(name string, args ...string) (string, error) {
				return `{"devboxes": []}`, nil
			},
			want:    []Devbox{},
			wantErr: false,
		},
		{
			name: "empty output",
			runFunc: func(name string, args ...string) (string, error) {
				return "", nil
			},
			want:    []Devbox{},
			wantErr: false,
		},
		{
			name: "no devboxes message",
			runFunc: func(name string, args ...string) (string, error) {
				return "", fmt.Errorf("no devboxes found")
			},
			want:    []Devbox{},
			wantErr: false,
		},
		{
			name: "invalid json",
			runFunc: func(name string, args ...string) (string, error) {
				return "not json", nil
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "connection error",
			runFunc: func(name string, args ...string) (string, error) {
				return "", fmt.Errorf("connection refused")
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &MockRunner{RunFunc: tt.runFunc}
			SetRunner(mock)
			defer ResetRunner()

			got, err := List()
			if (err != nil) != tt.wantErr {
				t.Errorf("List() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if len(got) != len(tt.want) {
				t.Errorf("List() returned %d items, want %d", len(got), len(tt.want))
				return
			}

			for i, d := range got {
				if d.Name != tt.want[i].Name || d.Status != tt.want[i].Status {
					t.Errorf("List()[%d] = %+v, want %+v", i, d, tt.want[i])
				}
			}

			// Verify command args
			if len(mock.Calls) > 0 {
				call := mock.Calls[0]
				expectedArgs := []string{"list", "--json"}
				if !slicesEqual(call.Args, expectedArgs) {
					t.Errorf("Expected args %v, got %v", expectedArgs, call.Args)
				}
			}
		})
	}
}

func TestSSH(t *testing.T) {
	mock := &MockRunner{}
	SetRunner(mock)
	defer ResetRunner()

	err := SSH("my-devbox")
	if err != nil {
		t.Errorf("SSH() error = %v", err)
	}

	if len(mock.Calls) != 1 {
		t.Fatalf("Expected 1 call, got %d", len(mock.Calls))
	}
	call := mock.Calls[0]
	if call.Method != "Exec" {
		t.Errorf("Expected Exec method, got %s", call.Method)
	}
	if call.Name != "devbox" {
		t.Errorf("Expected devbox command, got %s", call.Name)
	}
	expectedArgs := []string{"ssh", "my-devbox"}
	if !slicesEqual(call.Args, expectedArgs) {
		t.Errorf("Expected args %v, got %v", expectedArgs, call.Args)
	}
}

func TestGetSSHCommand(t *testing.T) {
	tests := []struct {
		name       string
		devboxName string
		runFunc    func(name string, args ...string) (string, error)
		want       string
		wantErr    bool
	}{
		{
			name:       "success",
			devboxName: "my-devbox",
			runFunc: func(name string, args ...string) (string, error) {
				return "ssh -i /path/to/key user@10.0.0.1", nil
			},
			want:    "ssh -i /path/to/key user@10.0.0.1",
			wantErr: false,
		},
		{
			name:       "devbox not found",
			devboxName: "missing",
			runFunc: func(name string, args ...string) (string, error) {
				return "", fmt.Errorf("devbox not found: missing")
			},
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &MockRunner{RunFunc: tt.runFunc}
			SetRunner(mock)
			defer ResetRunner()

			got, err := GetSSHCommand(tt.devboxName)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetSSHCommand() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GetSSHCommand() = %v, want %v", got, tt.want)
			}

			// Verify command args
			if len(mock.Calls) > 0 {
				call := mock.Calls[0]
				expectedArgs := []string{"ssh-command", tt.devboxName}
				if !slicesEqual(call.Args, expectedArgs) {
					t.Errorf("Expected args %v, got %v", expectedArgs, call.Args)
				}
			}
		})
	}
}

func TestGetStatus(t *testing.T) {
	tests := []struct {
		name       string
		devboxName string
		runFunc    func(name string, args ...string) (string, error)
		want       string
		wantErr    bool
	}{
		{
			name:       "running devbox",
			devboxName: "my-devbox",
			runFunc: func(name string, args ...string) (string, error) {
				return `{"devboxes": [{"name": "my-devbox", "status": "running"}]}`, nil
			},
			want:    "running",
			wantErr: false,
		},
		{
			name:       "stopped devbox",
			devboxName: "my-devbox",
			runFunc: func(name string, args ...string) (string, error) {
				return `{"devboxes": [{"name": "my-devbox", "status": "stopped"}]}`, nil
			},
			want:    "stopped",
			wantErr: false,
		},
		{
			name:       "devbox not found",
			devboxName: "missing",
			runFunc: func(name string, args ...string) (string, error) {
				return `{"devboxes": [{"name": "other", "status": "running"}]}`, nil
			},
			want:    "",
			wantErr: false,
		},
		{
			name:       "list error",
			devboxName: "my-devbox",
			runFunc: func(name string, args ...string) (string, error) {
				return "", fmt.Errorf("connection refused")
			},
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &MockRunner{RunFunc: tt.runFunc}
			SetRunner(mock)
			defer ResetRunner()

			got, err := GetStatus(tt.devboxName)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetStatus() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GetStatus() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSetDevboxBinary(t *testing.T) {
	// Save original
	original := devboxBinary
	defer func() { devboxBinary = original }()

	mock := &MockRunner{}
	SetRunner(mock)
	defer ResetRunner()

	// Test with custom binary name
	SetDevboxBinary("custom-devbox-cli")

	Delete("test")

	if len(mock.Calls) != 1 {
		t.Fatalf("Expected 1 call, got %d", len(mock.Calls))
	}
	if mock.Calls[0].Name != "custom-devbox-cli" {
		t.Errorf("Expected custom-devbox-cli, got %s", mock.Calls[0].Name)
	}

	// Test reset
	ResetDevboxBinary()
	mock.Calls = nil

	Delete("test")
	if mock.Calls[0].Name != "devbox" {
		t.Errorf("Expected devbox after reset, got %s", mock.Calls[0].Name)
	}
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
			name:         "create",
			callFunc:     func() { Create("dev", "https://github.com/u/r", "main") },
			expectedCmd:  "devbox",
			expectedArgs: []string{"create", "--name", "dev", "--repo", "https://github.com/u/r", "--branch", "main"},
		},
		{
			name:         "delete",
			callFunc:     func() { Delete("dev") },
			expectedCmd:  "devbox",
			expectedArgs: []string{"delete", "dev"},
		},
		{
			name:         "list",
			callFunc:     func() { List() },
			expectedCmd:  "devbox",
			expectedArgs: []string{"list", "--json"},
		},
		{
			name:         "ssh-command",
			callFunc:     func() { GetSSHCommand("dev") },
			expectedCmd:  "devbox",
			expectedArgs: []string{"ssh-command", "dev"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &MockRunner{
				RunFunc: func(name string, args ...string) (string, error) {
					// Return valid JSON for List()
					if len(args) > 0 && args[0] == "list" {
						return `{"devboxes": []}`, nil
					}
					return "", nil
				},
			}
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

	// Names with special chars that might appear in branch names
	specialNames := []string{
		"feature-my-branch",
		"fix_bug_123",
		"release-v1.0.0",
		"user/feature",
	}

	for _, name := range specialNames {
		mock.Calls = nil // Reset calls

		err := Delete(name)
		if err != nil {
			t.Errorf("Delete(%q) failed: %v", name, err)
		}

		call := mock.Calls[0]
		// Verify the name is passed through unchanged
		if call.Args[1] != name {
			t.Errorf("Devbox name not preserved: got %q, want %q", call.Args[1], name)
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

	err := Create("test", "https://github.com/u/r", "main")
	if err == nil {
		t.Error("Expected error to be propagated")
	}
	if !strings.Contains(err.Error(), "connection refused") {
		t.Errorf("Expected error message to contain 'connection refused', got: %v", err)
	}
}
