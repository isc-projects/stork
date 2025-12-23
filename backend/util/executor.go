package storkutil

import (
	"context"
	"os"
	"os/exec"

	"github.com/pkg/errors"
)

// The command executor is an abstraction layer on top of the exec package to
// improve testability and allow mock the operating system operations.
type CommandExecutor interface {
	Output(string, ...string) ([]byte, error)
	LookPath(string) (string, error)
	IsFileExist(string) bool
	GetFileInfo(string) (os.FileInfo, error)
}

// Executes the given command in the operating system.
type systemCommandExecutor struct{}

// Constructs the command executor that invokes the requests within the system
// shell.
func NewSystemCommandExecutor() CommandExecutor {
	return &systemCommandExecutor{}
}

// Executes a given command in the system shell and returns an output.
func (e *systemCommandExecutor) Output(command string, args ...string) ([]byte, error) {
	return exec.CommandContext(context.Background(), command, args...).Output()
}

// Looks for a given command in the system PATH and returns absolute path if found.
func (e *systemCommandExecutor) LookPath(command string) (string, error) {
	return exec.LookPath(command)
}

// Looks for a given file. Returns true is the path exist, is accessible, and
// points to a file.
func (e *systemCommandExecutor) IsFileExist(path string) bool {
	if stat, err := os.Stat(path); err == nil {
		return stat.Mode().IsRegular()
	}
	return false
}

// Gets the file info for a given file.
func (e *systemCommandExecutor) GetFileInfo(path string) (os.FileInfo, error) {
	info, err := os.Stat(path)
	return info, errors.Wrapf(err, "cannot get file info for %s", path)
}
