package testutil

import (
	"io/ioutil"
	"log"
	"os"
	"path"
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
	dir, err := ioutil.TempDir("", "stork_ut_*")
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
func (sb *Sandbox) Join(name string) string {
	// build full path
	fpath := path.Join(sb.BasePath, name)

	// ensure directory
	dir := path.Dir(fpath)
	err := os.MkdirAll(dir, 0777)
	if err != nil {
		log.Fatal(err)
	}

	// create file in the filesystem
	file, err := os.Create(fpath)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	return fpath
}

// Create indicated directory in sandbox and all parent directories
// and return a full path.
func (sb *Sandbox) JoinDir(name string) string {
	// build full path
	fpath := path.Join(sb.BasePath, name)

	// ensure directory
	err := os.MkdirAll(fpath, 0777)
	if err != nil {
		log.Fatal(err)
	}

	return fpath
}

// Create a file and write provided contect to it.
func (sb *Sandbox) Write(name string, content string) string {
	fpath := sb.Join(name)

	err := ioutil.WriteFile(fpath, []byte(content), 0600)
	if err != nil {
		log.Fatal(err)
	}

	return fpath
}
