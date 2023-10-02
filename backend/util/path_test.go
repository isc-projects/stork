package storkutil

import (
	"path"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"isc.org/stork/testutil"
)

// Test that the listing function finds the expected paths.
func TestListFilePaths(t *testing.T) {
	// Arrange
	directory := "testdata/files"

	// Act
	paths, err := ListFilePaths(directory, false)

	// Assert
	require.NoError(t, err)
	require.Len(t, paths, 5)
}

// Test that the listing function returns an error if the directory doesn't
// exist.
func TestListFilePathsForInvalidDirectory(t *testing.T) {
	// Arrange
	sb := testutil.NewSandbox()
	defer sb.Close()
	directory := path.Join(sb.BasePath, "non-exist-directory")

	// Act
	paths, err := ListFilePaths(directory, false)

	// Assert
	require.Nil(t, paths)
	require.Error(t, err)
}

// Test that the listing function returns the paths relative to a given directory
// prefixed by this directory.
func TestListFilePathsStartWithDirectory(t *testing.T) {
	// Arrange
	directory := "testdata/files"

	// Act
	paths, _ := ListFilePaths(directory, false)

	// Assert
	for _, path := range paths {
		require.True(t, strings.HasPrefix(path, directory+"/"))
	}
}

// Test that the listing function search only in the top directory.
func TestListFilePathsOnlyTopLevel(t *testing.T) {
	// Arrange
	directory := "testdata/files"

	// Act
	paths, _ := ListFilePaths(directory, false)

	// Assert
	for _, path := range paths {
		require.NotContains(t, path, "/dir/")
	}
}

// Test that the returned paths are sorted if requested.
func TestListFilePathsSort(t *testing.T) {
	// Arrange
	directory := "testdata/files"

	// Act
	paths, _ := ListFilePaths(directory, true)

	// Assert
	for i := 1; i < len(paths); i++ {
		require.LessOrEqual(t, paths[i-1], paths[i])
	}
}
