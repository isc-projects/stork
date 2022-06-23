package hooks

import (
	"io/ioutil"

	"github.com/sirupsen/logrus"
	"isc.org/stork"
)

func LoadAllHooks(program string, directory string) []interface{} {
	files, err := ioutil.ReadDir(directory)
	if err != nil {
		logrus.
			WithError(err).
			WithField("path", directory).
			Error("cannot list hook directory")
		return []interface{}{}
	}

	allCallouts := []interface{}{}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		library, err := NewLibraryManager(file.Name())
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
