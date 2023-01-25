package testutil

import (
	"os"
	"path"

	log "github.com/sirupsen/logrus"
)

// Sandbox is an object that creates a sandbox for files or directories
// that needs to be created to do some tests. Sandbox provides
// utility functions for creating files and dirs, getting paths to them
// and at the end removing whole sandbox with its content.
// Each created sandbox has its own, unique directory so two sandboxes
// never interfere.

// Struct that holds information about sandbox.
type Sandbox struct {
	BasePath string
}

// Create a new sandbox. The sandbox is located in a temporary
// directory.
func NewSandbox() *Sandbox {
	dir, err := os.MkdirTemp("", "stork_ut_*")
	if err != nil {
		log.Fatal(err)
	}
	sb := &Sandbox{
		BasePath: dir,
	}

	return sb
}

// Close sandbox and remove all its contents.
func (sb *Sandbox) Close() {
	os.RemoveAll(sb.BasePath)
}

// Create parent directory in sandbox (and all missing directories
// above it if needed, similar to -p option in mkdir), create
// indicated file in this parent directory, and return a full path to
// this file.
func (sb *Sandbox) Join(name string) (string, error) {
	// build full path
	filePath := path.Join(sb.BasePath, name)

	// ensure directory
	dir := path.Dir(filePath)
	err := os.MkdirAll(dir, 0o777)
	if err != nil {
		return "", err
	}

	// create file in the filesystem
	file, err := os.Create(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	return filePath, nil
}

// Create indicated directory in sandbox and all parent directories
// and return a full path.
func (sb *Sandbox) JoinDir(name string) (string, error) {
	// build full path
	filePath := path.Join(sb.BasePath, name)

	// ensure directory
	err := os.MkdirAll(filePath, 0o777)
	if err != nil {
		return "", err
	}

	return filePath, nil
}

// Create a file and write provided content to it.
func (sb *Sandbox) Write(name string, content string) (string, error) {
	filePath, err := sb.Join(name)
	if err != nil {
		return "", err
	}

	err = os.WriteFile(filePath, []byte(content), 0o600)
	if err != nil {
		log.Fatal(err)
		return "", err
	}

	return filePath, err
}
