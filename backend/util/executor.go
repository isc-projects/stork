package storkutil

import (
	"bufio"
	"context"
	"os"
	"os/exec"

	"github.com/pkg/errors"
)

var (
	_ CommandExecutor       = (*systemCommandExecutor)(nil)
	_ CommandExecutorOutput = (*systemCommandExecutorOutput)(nil)
)

// A CommandExecutorOutput represents output from the CommandExecutor.Start()
// function. It returns a scanner to be used to capture the command output.
// It also implements the Wait() function to wait for the command to complete.
type CommandExecutorOutput interface {
	// Returns the scanner to be used to capture the command output.
	GetScanner() *bufio.Scanner
	// Waits for the command to exit. The concrete implementation calls the
	// exec.Cmd.Wait function. It waits for the command to exit and waits
	// for any copying to stdin or copying from stdout or stderr to complete.
	// It is called when the caller has received expected output from the command
	// and is waiting for the command to exit. For long running processes, such as
	// log tail with following, the caller should cancel the context passed to the
	// CommandExecutor.Start(). Cancelling the context will cause the command to exit.
	// This function should be called after cancelling the context to cleanly handle
	// process termination.
	Wait() error
}

// Implements the CommandExecutorOutput interface for the system command executor.
type systemCommandExecutorOutput struct {
	cmd     *exec.Cmd
	scanner *bufio.Scanner
}

// Waits for the executed command to exit. It calls the exec.Cmd.Wait function
// which waits for the command to exit and for any copying to stdin or copying
// from stdout or stderr to complete.
func (c *systemCommandExecutorOutput) Wait() error {
	return c.cmd.Wait()
}

// Returns the scanner to be used to capture the command output.
func (c *systemCommandExecutorOutput) GetScanner() *bufio.Scanner {
	return c.scanner
}

// The command executor is an abstraction layer on top of the exec package to
// improve testability and allow mock the operating system operations.
type CommandExecutor interface {
	Output(string, ...string) ([]byte, error)
	Start(context.Context, string, ...string) (CommandExecutorOutput, error)
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
// The context passed to the function can be used to cancel the command execution.
// Suppose the function is used to tail and follow the log file, cancelling the context
// will kill the process. Call Wait() after cancelling the context to cleanly handle
// process termination.
func (e *systemCommandExecutor) Start(ctx context.Context, command string, args ...string) (CommandExecutorOutput, error) {
	cmd := exec.CommandContext(ctx, command, args...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	scanner := bufio.NewScanner(stdout)
	return &systemCommandExecutorOutput{
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
