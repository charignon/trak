// Package devbox provides functions to interact with remote k8s dev environments.
// All operations shell out to the devbox CLI rather than using k8s APIs directly.
// NOTE: This uses placeholder commands - update when actual devbox CLI is available.
package devbox

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Devbox represents a remote k8s development environment.
type Devbox struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

// CommandRunner is the interface for running devbox commands.
// This allows for mocking in tests.
type CommandRunner interface {
	Run(name string, args ...string) (string, error)
	Exec(name string, args ...string) error
}

// DefaultRunner executes real devbox commands.
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

// Exec executes a command that replaces the current process (for SSH).
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

// devboxBinary is the name of the devbox CLI binary.
// This can be overridden for testing or if the CLI has a different name.
var devboxBinary = "devbox"

// SetDevboxBinary sets the devbox CLI binary name (for testing).
func SetDevboxBinary(binary string) {
	devboxBinary = binary
}

// ResetDevboxBinary resets the devbox CLI binary to the default.
func ResetDevboxBinary() {
	devboxBinary = "devbox"
}

// Create creates a new devbox environment.
// Expected CLI: devbox create --name <name> --repo <repoURL> --branch <branch>
func Create(name, repoURL, branch string) error {
	_, err := runner.Run(devboxBinary, "create",
		"--name", name,
		"--repo", repoURL,
		"--branch", branch,
	)
	return err
}

// Delete deletes a devbox environment.
// Expected CLI: devbox delete <name>
func Delete(name string) error {
	_, err := runner.Run(devboxBinary, "delete", name)
	return err
}

// Exists checks if a devbox with the given name exists.
// Uses List() and searches for the name.
func Exists(name string) (bool, error) {
	devboxes, err := List()
	if err != nil {
		return false, err
	}

	for _, d := range devboxes {
		if d.Name == name {
			return true, nil
		}
	}
	return false, nil
}

// listResponse is the expected JSON structure from devbox list --json.
type listResponse struct {
	Devboxes []Devbox `json:"devboxes"`
}

// List returns all devbox environments.
// Expected CLI: devbox list --json
// Expected output: {"devboxes": [{"name": "...", "status": "..."}]}
func List() ([]Devbox, error) {
	output, err := runner.Run(devboxBinary, "list", "--json")
	if err != nil {
		// If no devboxes exist, CLI might return empty or error
		if strings.Contains(err.Error(), "no devboxes") {
			return []Devbox{}, nil
		}
		return nil, err
	}

	if output == "" {
		return []Devbox{}, nil
	}

	var resp listResponse
	if err := json.Unmarshal([]byte(output), &resp); err != nil {
		return nil, fmt.Errorf("failed to parse devbox list output: %w", err)
	}

	return resp.Devboxes, nil
}

// SSH connects to a devbox environment interactively.
// This replaces the current process with an SSH session.
// Expected CLI: devbox ssh <name>
func SSH(name string) error {
	return runner.Exec(devboxBinary, "ssh", name)
}

// GetSSHCommand returns the SSH command string for a devbox.
// This is useful for tmux integration where we need the command string
// rather than executing it directly.
// Expected CLI: devbox ssh-command <name>
// Expected output: ssh -i /path/to/key user@host
func GetSSHCommand(name string) (string, error) {
	output, err := runner.Run(devboxBinary, "ssh-command", name)
	if err != nil {
		return "", err
	}
	return output, nil
}

// GetStatus returns the status of a specific devbox.
// Returns empty string if devbox doesn't exist.
func GetStatus(name string) (string, error) {
	devboxes, err := List()
	if err != nil {
		return "", err
	}

	for _, d := range devboxes {
		if d.Name == name {
			return d.Status, nil
		}
	}
	return "", nil
}
