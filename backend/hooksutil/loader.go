package hooksutil

import (
	"io/ioutil"
	"path/filepath"
	"sort"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"isc.org/stork"
)

// Loads all hook files from a given directory for a specific program (server
// or agent). Returns a list of extracted callout objects.
// The hook must be compiled with a matching version and application name.
// Otherwise, it's skipped.
// The hooks are loaded in the lexicographic order of hook file names.
func LoadAllHooks(program string, directory string) []any {
	// Search for files.
	paths, err := findFilePaths(directory)
	if err != nil {
		logrus.
			WithError(err).
			Error("cannot find plugin paths")
		return []any{}
	}

	// Extract the Go plugins.
	libraries := []*LibraryManager{}
	for _, path := range paths {
		library, err := NewLibraryManager(path)
		if err != nil {
			logrus.
				WithError(err).
				WithField("library", path).
				Error("cannot open hook library")
			continue
		}
		libraries = append(libraries, library)
	}

	// Load the hook callouts.
	allCallouts := []any{}
	for _, library := range libraries {
		callouts, err := extractCallouts(library, program)
		if err != nil {
			logrus.
				WithError(err).
				WithField("library", library.GetPath()).
				Error("cannot extract callouts from library")
			continue
		}

		allCallouts = append(allCallouts, callouts)
	}

	return allCallouts
}

// Finds all file paths in a given directory. It looks only at the top level.
// Returned paths are sorted lexicographically.
func findFilePaths(directory string) ([]string, error) {
	entries, err := ioutil.ReadDir(directory)
	if err != nil {
		err = errors.Wrapf(err, "cannot list hook directory: %s", directory)
		return nil, err
	}

	files := []string{}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		files = append(files, filepath.Join(directory, entry.Name()))
	}

	// Sorts files by name
	sort.Slice(files, func(i, j int) bool {
		return files[i] < files[j]
	})

	return files, nil
}

// Extracts the object with callout points implementations from a given library
// (Go plugin). The library is validated. The version and program name must match
// the caller application.
func extractCallouts(library *LibraryManager, expectedProgram string) (any, error) {
	hookProgram, hookVersion, err := library.Version()
	if err != nil {
		err = errors.WithMessage(err, "cannot check version of hook library")
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
