package storkutil

import (
	"bufio"
	"context"
	"os"
	"os/exec"

	"github.com/pkg/errors"
)

var (
	_ CommandExecutor        = (*systemCommandExecutor)(nil)
	_ CommandExecutorCommand = (*systemCommandExecutorCommand)(nil)
)

// A command executor command represents a command issued with the
// CommandExecutor.Start() function. It returns a scanner to be used
// to capture the command output. It also implements the Wait() function
// to wait for the command to complete.
type CommandExecutorCommand interface {
	GetScanner() *bufio.Scanner
	Wait() error
}

// Implements the CommandExecutorCommand interface for the system command executor.
type systemCommandExecutorCommand struct {
	cmd     *exec.Cmd
	scanner *bufio.Scanner
}

// Waits for the command to complete.
func (c *systemCommandExecutorCommand) Wait() error {
	return c.cmd.Wait()
}

// Returns the scanner to be used to capture the command output.
func (c *systemCommandExecutorCommand) GetScanner() *bufio.Scanner {
	return c.scanner
}

// The command executor is an abstraction layer on top of the exec package to
// improve testability and allow mock the operating system operations.
type CommandExecutor interface {
	Output(string, ...string) ([]byte, error)
	Start(context.Context, string, ...string) (CommandExecutorCommand, error)
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

// Executes a given command in the system shell without waiting for it to complete.
func (e *systemCommandExecutor) Start(ctx context.Context, command string, args ...string) (CommandExecutorCommand, error) {
	cmd := exec.CommandContext(ctx, command, args...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	scanner := bufio.NewScanner(stdout)
	return &systemCommandExecutorCommand{
		cmd:     cmd,
		scanner: scanner,
	}, nil
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
