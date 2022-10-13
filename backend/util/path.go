package storkutil

import (
	"io/ioutil"
	"path/filepath"
	"sort"

	"github.com/pkg/errors"
)

// List all file paths in a given directory. It looks only at the top level.
// Returned paths may be sorted lexicographically.
func ListFilePaths(directory string, sortByPath bool) ([]string, error) {
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
	if sortByPath {
		sort.Slice(files, func(i, j int) bool {
			return files[i] < files[j]
		})
	}

	return files, nil
}
