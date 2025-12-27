// Package tmux provides functions to interact with tmux via shell commands.
package tmux

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// CommandRunner is the interface for running tmux commands.
// This allows for mocking in tests.
type CommandRunner interface {
	Run(name string, args ...string) (string, error)
	Exec(name string, args ...string) error
}

// DefaultRunner executes real tmux commands.
type DefaultRunner struct{}

// Run executes a command and returns its output.
func (r *DefaultRunner) Run(name string, args ...string) (string, error) {
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

// Exec executes a command that replaces the current process (for attach).
func (r *DefaultRunner) Exec(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// runner is the current command runner (can be swapped for testing).
var runner CommandRunner = &DefaultRunner{}

// SetRunner sets the command runner (for testing).
func SetRunner(r CommandRunner) {
	runner = r
}

// ResetRunner resets the command runner to the default.
func ResetRunner() {
	runner = &DefaultRunner{}
}

// SessionExists checks if a tmux session with the given name exists.
func SessionExists(name string) (bool, error) {
	_, err := runner.Run("tmux", "has-session", "-t", name)
	if err != nil {
		// tmux has-session returns exit code 1 if session doesn't exist
		if strings.Contains(err.Error(), "exit status 1") || strings.Contains(err.Error(), "can't find session") {
			return false, nil
		}
		// Some other error occurred
		return false, err
	}
	return true, nil
}

// CreateSession creates a new tmux session with the given name.
// The session is created detached.
func CreateSession(name string) error {
	_, err := runner.Run("tmux", "new-session", "-d", "-s", name)
	return err
}

// WindowExists checks if a window exists in the given session.
func WindowExists(session, windowName string) (bool, error) {
	// List windows in the session and check if our window name is there
	output, err := runner.Run("tmux", "list-windows", "-t", session, "-F", "#{window_name}")
	if err != nil {
		// Session might not exist
		if strings.Contains(err.Error(), "can't find session") {
			return false, nil
		}
		return false, err
	}

	windows := strings.Split(output, "\n")
	for _, w := range windows {
		if w == windowName {
			return true, nil
		}
	}
	return false, nil
}

// CreateWindow creates a new window in a session with optional start directory.
func CreateWindow(session, windowName, startDir string) error {
	args := []string{"new-window", "-t", session, "-n", windowName}
	if startDir != "" {
		args = append(args, "-c", startDir)
	}
	_, err := runner.Run("tmux", args...)
	return err
}

// SwitchToWindow switches to a window in the given session.
// This only works when already inside tmux.
func SwitchToWindow(session, windowName string) error {
	target := fmt.Sprintf("%s:%s", session, windowName)
	_, err := runner.Run("tmux", "switch-client", "-t", target)
	return err
}

// AttachSession attaches to a session.
// This should be used when not inside tmux.
func AttachSession(session string) error {
	return runner.Exec("tmux", "attach-session", "-t", session)
}

// IsInsideTmux returns true if currently running inside tmux.
func IsInsideTmux() bool {
	return os.Getenv("TMUX") != ""
}

// RunInWindow sends a command to run in a specific window.
// Uses send-keys to type the command and execute it.
func RunInWindow(session, windowName, command string) error {
	target := fmt.Sprintf("%s:%s", session, windowName)
	// send-keys with Enter to execute the command
	_, err := runner.Run("tmux", "send-keys", "-t", target, command, "Enter")
	return err
}

// SelectWindow selects a window (makes it the current window in the session).
func SelectWindow(session, windowName string) error {
	target := fmt.Sprintf("%s:%s", session, windowName)
	_, err := runner.Run("tmux", "select-window", "-t", target)
	return err
}

// KillWindow kills a window in a session.
func KillWindow(session, windowName string) error {
	target := fmt.Sprintf("%s:%s", session, windowName)
	_, err := runner.Run("tmux", "kill-window", "-t", target)
	return err
}

// KillSession kills an entire session.
func KillSession(session string) error {
	_, err := runner.Run("tmux", "kill-session", "-t", session)
	return err
}

// ListSessions returns a list of all tmux session names.
func ListSessions() ([]string, error) {
	output, err := runner.Run("tmux", "list-sessions", "-F", "#{session_name}")
	if err != nil {
		// No server running is not an error for our purposes
		if strings.Contains(err.Error(), "no server running") {
			return []string{}, nil
		}
		return nil, err
	}

	if output == "" {
		return []string{}, nil
	}

	return strings.Split(output, "\n"), nil
}

// ListWindows returns a list of all window names in a session.
func ListWindows(session string) ([]string, error) {
	output, err := runner.Run("tmux", "list-windows", "-t", session, "-F", "#{window_name}")
	if err != nil {
		return nil, err
	}

	if output == "" {
		return []string{}, nil
	}

	return strings.Split(output, "\n"), nil
}
