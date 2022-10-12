package hooksutil

import (
	"io/ioutil"
	"path/filepath"
	"sort"

	"github.com/sirupsen/logrus"
	"isc.org/stork"
)

// Loads all hook files from a given directory for a specific program (server
// or agent). Returns a list of extracted callout objects.
// The hook must be compiled with a matching version and application name.
// Otherwise, it's skipped.
// The hooks are loaded in the lexicographic order of hook file names.
func LoadAllHooks(program string, directory string) []any {
	files, err := ioutil.ReadDir(directory)
	if err != nil {
		logrus.
			WithError(err).
			WithField("path", directory).
			Error("cannot list hook directory")
		return []any{}
	}

	// Sorts files by name
	sort.Slice(files, func(i, j int) bool {
		return files[i].Name() < files[j].Name()
	})

	allCallouts := []any{}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		library, err := NewLibraryManager(filepath.Join(directory, file.Name()))
		if err != nil {
			logrus.
				WithError(err).
				WithField("library", file.Name()).
				Error("cannot open hook library")
			continue
		}

		hookProgram, hookVersion, err := library.Version()
		if err != nil {
			logrus.
				WithError(err).
				WithField("library", file.Name()).
				Error("cannot check version of hook library")
			continue
		}

		if program != hookProgram {
			logrus.
				WithField("library", file.Name()).
				WithField("program", hookProgram).
				Error("hook library dedicated for another program")
			continue
		}

		if hookVersion != stork.Version {
			logrus.
				WithField("library", file.Name()).
				WithField("version", hookVersion).
				Error("incompatible hook version")
			continue
		}

		callouts, err := library.Load()
		if err != nil {
			logrus.
				WithError(err).
				WithField("library", file.Name()).
				Error("cannot load hook library")
			continue
		}

		allCallouts = append(allCallouts, callouts)
	}

	return allCallouts
}
