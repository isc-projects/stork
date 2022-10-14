package hooksutil

import (
	"github.com/pkg/errors"
	"isc.org/stork"
	"isc.org/stork/hooks"
	storkutil "isc.org/stork/util"
)

// Loads all hook files from a given directory for a specific program (server
// or agent). Returns a list of extracted callout objects.
// The hook must be compiled with a matching version and application name.
// Otherwise, it's skipped.
// The hooks are loaded in the lexicographic order of hook file names.
func LoadAllHooks(program string, directory string) ([]hooks.Callout, error) {
	// Search for files.
	paths, err := storkutil.ListFilePaths(directory, true)
	if err != nil {
		err = errors.WithMessagef(err, "cannot find plugin paths in: %s", directory)
		return nil, err
	}

	allCallouts := []hooks.Callout{}

	for _, path := range paths {
		// Extract the Go plugins.
		library, err := NewLibraryManager(path)
		if err != nil {
			err = errors.WithMessagef(err, "cannot open hook library: %s", path)
			return nil, err
		}

		// Load the hook callouts.
		callouts, err := extractCallouts(library, program)
		if err != nil {
			err = errors.WithMessagef(err, "cannot extract callouts from library: %s", path)
			return nil, err
		}

		allCallouts = append(allCallouts, callouts)
	}

	return allCallouts, nil
}

// Extracts the object with callout points implementations from a given library
// (Go plugin). The library is validated. The version and program name must match
// the caller application.
func extractCallouts(library *LibraryManager, expectedProgram string) (hooks.Callout, error) {
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

	callouts, err := library.Load()
	if err != nil {
		err = errors.WithMessage(err, "cannot load hook library")
		return nil, err
	}

	return callouts, nil
}
