package git

import (
	"fmt"
	"os/exec"
	"strings"
)

// FileChange represents a git file change
type FileChange struct {
	Status  string
	OldPath string
	NewPath string
}

// Repository represents a git repository interface
type Repository interface {
	GetChanges(migrationDir, compareBranch string) ([]FileChange, error)
}

// LocalRepository implements Repository for local git operations
type LocalRepository struct {
	debug bool
}

// NewLocalRepository creates a new local repository instance
func NewLocalRepository() *LocalRepository {
	return &LocalRepository{}
}

// SetDebug enables or disables debug mode
func (r *LocalRepository) SetDebug(debug bool) {
	r.debug = debug
}

// GetChanges retrieves file changes from git diff
func (r *LocalRepository) GetChanges(migrationDir, compareBranch string) ([]FileChange, error) {
	cmdArgs := []string{"git", "diff", "--name-status"}

	if compareBranch != "" {
		cmdArgs = append(cmdArgs, compareBranch)
	}

	cmdArgs = append(cmdArgs, migrationDir)

	if r.debug {
		fmt.Printf("Executing git command: %s\n", strings.Join(cmdArgs, " "))
	}

	cmd := exec.Command(cmdArgs[0], cmdArgs[1:]...)
	output, err := cmd.Output()
	if err != nil {
		if r.debug {
			fmt.Printf("Git command failed: %v\n", err)
		}
		return nil, fmt.Errorf("failed to execute git diff: %w", err)
	}

	if r.debug {
		fmt.Printf("Git command output (%d bytes):\n%s\n", len(output), string(output))
		if string(output) == "" {
			fmt.Println("Git diff returned empty output (no changes detected)")
		}
	}

	changes, err := parseGitDiffOutput(string(output))
	if err != nil && r.debug {
		fmt.Printf("Error parsing git diff output: %v\n", err)
	}

	return changes, err
}

// parseGitDiffOutput parses the output of git diff --name-status
func parseGitDiffOutput(output string) ([]FileChange, error) {
	var changes []FileChange

	lines := strings.Split(strings.TrimSpace(output), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		change := FileChange{
			Status:  fields[0],
			OldPath: fields[1],
			NewPath: fields[1], // Default to same as old path
		}

		// Handle renamed files (status starts with R)
		if len(fields) == 3 {
			change.NewPath = fields[2]
		}

		changes = append(changes, change)
	}

	return changes, nil
}
