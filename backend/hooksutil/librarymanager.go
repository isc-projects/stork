package hooksutil

import (
	"io"
	"plugin"

	"github.com/pkg/errors"
	"isc.org/stork/hooks"
)

// Wrapper for a raw Go plugin to easier extraction of expected symbols
// (functions).
type LibraryManager struct {
	path string
	p    *plugin.Plugin
}

// Opens a hook file and constructs the library manager object. Returns an
// error if the provided file isn't a valid Go plugin. It doesn't validate if
// the file is a valid Stork hook; the hook library will be created for any
// proper Go plugin.
func NewLibraryManager(path string) (*LibraryManager, error) {
	p, err := plugin.Open(path)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot open a plugin: %s", path)
	}

	return &LibraryManager{path, p}, nil
}

// Extracts and calls the load function of the Stork hook. Returns an error if
// the function is missing or fails. On success, returns an object with the
// callout point implementations. On success, it returns an object with the
// callout point implementations. The object also implements the Closer interface
// that must be called to unload the hook.
func (lm *LibraryManager) Load() (io.Closer, error) {
	symbolName := hooks.HookLoadFunctionName
	symbol, err := lm.p.Lookup(symbolName)
	if err != nil {
		return nil, errors.Wrapf(err, "lookup for symbol: %s failed", symbolName)
	}

	load, ok := symbol.(hooks.HookLoadFunction)
	if !ok {
		return nil, errors.Errorf("symbol %s has unexpected signature", symbolName)
	}

	callouts, err := load()
	err = errors.Wrap(err, "cannot load the hook")

	return callouts, err
}

// Extracts and calls the version function of the Stork hook. Returns an error if
// the function is missing or fails. The output contains the compatible
// application name (agent or server) and the expected Stork version.
func (lm *LibraryManager) Version() (program string, version string, err error) {
	symbolName := hooks.HookVersionFunctionName
	symbol, err := lm.p.Lookup(symbolName)
	if err != nil {
		err = errors.Wrapf(err, "lookup for symbol: %s failed", symbolName)
		return
	}

	versionFunction, ok := symbol.(hooks.HookVersionFunction)
	if !ok {
		err = errors.Errorf("symbol %s has unexpected signature", symbolName)
		return
	}

	program, version = versionFunction()
	return
}

// Returns a path to the hook file.
func (lm *LibraryManager) Path() string {
	return lm.path
}
