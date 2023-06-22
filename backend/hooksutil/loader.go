package hooksutil

import (
	"github.com/pkg/errors"
	"isc.org/stork"
	"isc.org/stork/hooks"
	storkutil "isc.org/stork/util"
)

type walkCallback = func(path string, library *LibraryManager, err error) bool

// Iterates over the plugins in a given directory. Every entry in the directory
// (file or folder) calls the callback function. It accepts a path, library
// wrapper (if no error), and error. The callback should return true to
// continue. The function returns true on the general failure to access the
// directory. It isn't guaranteed that the open library is a valid Stork hook.
// The libraries are loaded in the lexicographic order of hook file names.
func WalkPluginLibraries(directory string, callback walkCallback) error {
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

// Check if the library is compatible with the current application.
func checkLibraryCompatibility(library *LibraryManager, expectedProgram string) error {
	hookProgram, hookVersion, err := library.Version()
	if err != nil {
		return errors.WithMessage(err, "cannot call version of hook library")
	}

	if expectedProgram != hookProgram {
		return errors.Errorf("hook library dedicated for another program: %s", hookProgram)
	}

	if hookVersion != stork.Version {
		return errors.Errorf("incompatible hook version: %s", hookVersion)
	}

	return nil
}

// Iterates over the plugins in a given directory but skips the libraries that
// are not compatible with the current application (are dedicated for different
// program or have a wrong version).
func WalkCompatiblePluginLibraries(directory, program string, walkCallback walkCallback) error {
	var libraryErr error
	err := WalkPluginLibraries(directory, func(path string, library *LibraryManager, err error) bool {
		if err != nil {
			libraryErr = errors.WithMessagef(err, "cannot open hook library: %s", path)
			return false
		}

		if err := checkLibraryCompatibility(library, program); err != nil {
			libraryErr = err
			return false
		}

		return walkCallback(path, library, err)
	})
	if err != nil {
		return err
	}
	return libraryErr
}

// Loads all hook files from a given directory for a specific program (server
// or agent). Returns a list of extracted callout carriers.
// The hook must be compiled with a matching version and application name.
// Otherwise, the loading is stopped.
// The hooks are loaded in the lexicographic order of hook file names.
func LoadAllHooks(program string, directory string, allSettings map[string]hooks.HookSettings) ([]hooks.CalloutCarrier, error) {
	var (
		carriers   []hooks.CalloutCarrier
		libraryErr error
	)

	err := WalkCompatiblePluginLibraries(directory, program, func(path string, library *LibraryManager, err error) bool {
		if err != nil {
			libraryErr = errors.WithMessagef(err, "cannot open hook library: %s", path)
			return false
		}

		settings := allSettings[library.GetName()]
		carrier, err := extractCarrier(library, settings)
		if err != nil {
			libraryErr = err
			return false
		}

		carriers = append(carriers, carrier)
		return true
	})
	if err != nil {
		return nil, err
	}

	return carriers, libraryErr
}

// Extracts the object with callouts (callout specification implementations)
// from a given library (Go plugin). The library is validated. The version and
// program name must match the caller application.
func extractCarrier(library *LibraryManager, settings hooks.HookSettings) (hooks.CalloutCarrier, error) {
	// Load the hook callout carrier.
	carrier, err := library.Load(settings)
	if err != nil {
		return nil, errors.WithMessage(err, "cannot load hook library")
	}
	return carrier, nil
}

// Iterates over the compatible plugins in a given directory and extracts
// their settings.
func CollectProtoSettings(program, directory string) (map[string]hooks.HookSettings, error) {
	allSettings := map[string]hooks.HookSettings{}
	var libraryErr error

	err := WalkCompatiblePluginLibraries(directory, program, func(path string, library *LibraryManager, err error) bool {
		if err != nil {
			libraryErr = errors.WithMessagef(err, "cannot open hook library: %s", path)
			return false
		}

		proto, libraryErr := library.ProtoSettings()
		if libraryErr != nil {
			return false
		}

		allSettings[library.GetName()] = proto
		return true
	})
	if err != nil {
		return nil, err
	}

	return allSettings, libraryErr
}
