package hooksutil

import (
	"github.com/pkg/errors"
	"isc.org/stork"
	"isc.org/stork/hooks"
	storkutil "isc.org/stork/util"
)

// Iterates over the plugins in a given directory. Every entry in the directory
// (file or folder) calls the callback function. It accepts a path, library
// wrapper (if no error), and error. The callback should return true to
// continue. The function returns true on the general failure to access the
// directory. It isn't guaranteed that the open library is a valid Stork hook.
// The libraries are loaded in the lexicographic order of hook file names.
func WalkPluginLibraries(directory string, callback func(path string, library *LibraryManager, err error) bool) error {
	paths, err := storkutil.ListFilePaths(directory, true)
	if err != nil {
		err = errors.WithMessagef(err, "cannot find plugin paths in: %s", directory)
		return err
	}

	for _, path := range paths {
		// Extract the Go plugins.
		library, err := NewLibraryManager(path)
		err = errors.WithMessagef(err, "cannot open hook library: %s", path)
		if !callback(path, library, err) {
			break
		}
	}

	return nil
}

// Loads all hook files from a given directory for a specific program (server
// or agent). Returns a list of extracted callout objects.
// The hook must be compiled with a matching version and application name.
// Otherwise, the loading is stopped.
// The hooks are loaded in the lexicographic order of hook file names.
func LoadAllHookCallouts(program string, directory string) ([]hooks.Callout, error) {
	var (
		callouts   []hooks.Callout
		callout    hooks.Callout
		libraryErr error
	)

	err := WalkPluginLibraries(directory, func(path string, library *LibraryManager, err error) bool {
		if err != nil {
			libraryErr = errors.WithMessagef(err, "cannot open hook library: %s", path)
			return false
		}

		// Load the hook callouts.
		callout, libraryErr = extractCallout(library, program)
		if libraryErr != nil {
			libraryErr = errors.WithMessagef(libraryErr, "cannot extract callouts from library: %s", path)
			return false
		}

		callouts = append(callouts, callout)
		return true
	})
	if err != nil {
		return nil, err
	}

	return callouts, libraryErr
}

// Extracts the object with callout points implementations from a given library
// (Go plugin). The library is validated. The version and program name must match
// the caller application.
func extractCallout(library *LibraryManager, expectedProgram string) (hooks.Callout, error) {
	hookProgram, hookVersion, err := library.Version()
	if err != nil {
		err = errors.WithMessage(err, "cannot call version of hook library")
		return nil, err
	}

	if expectedProgram != hookProgram {
		return nil, errors.Errorf("hook library dedicated for another program: %s", hookProgram)
	}

	if hookVersion != stork.Version {
		return nil, errors.Errorf("incompatible hook version: %s", hookVersion)
	}

	callout, err := library.Load()
	if err != nil {
		err = errors.WithMessage(err, "cannot load hook library")
		return nil, err
	}

	return callout, nil
}
