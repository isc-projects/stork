package storkutil

import (
	"archive/tar"
	"bytes"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// When the tarball is invalid the error
// should occur.
func TestWalkNonExistsTarball(t *testing.T) {
	// Act
	err := WalkFilesInTarball(bytes.NewBufferString(""),
		func(header *tar.Header, read func() ([]byte, error)) bool {
			return true
		})

	// Assert
	require.Error(t, err)
}

// When walk through the empty tarball then
// no files should occur.
func TestWalkEmptyTarball(t *testing.T) {
	// Arrange
	var buffer bytes.Buffer
	tarballWriter := NewTarballWriter(&buffer)
	tarballWriter.Close()
	callbackCallCount := 0

	// Act
	err := WalkFilesInTarball(&buffer,
		func(header *tar.Header, read func() ([]byte, error)) bool {
			callbackCallCount++
			return true
		})

	// Assert
	require.NoError(t, err)
	require.Zero(t, callbackCallCount)
}

// Test that all files are visited by the walk function.
func TestWalkFilledTarball(t *testing.T) {
	// Arrange
	var buffer bytes.Buffer
	tarballWriter := NewTarballWriter(&buffer)
	for _, filename := range []string{"aaa", "bbb", "ccc"} {
		tarballWriter.AddContent(filename, []byte(filename), time.Now())
	}
	tarballWriter.Close()
	callbackCallCount := 0

	// Act
	err := WalkFilesInTarball(&buffer,
		func(header *tar.Header, read func() ([]byte, error)) bool {
			callbackCallCount++
			return true
		},
	)

	// Assert
	require.NoError(t, err)
	require.EqualValues(t, 3, callbackCallCount)
}

// Test that during walk the files are read properly.
func TestWalkAndReadTarball(t *testing.T) {
	// Arrange
	var buffer bytes.Buffer
	tarballWriter := NewTarballWriter(&buffer)
	tarballWriter.AddContent("foo", []byte("foobar"), time.Now())
	tarballWriter.Close()
	var content []byte
	var readErr error

	// Act
	_ = WalkFilesInTarball(&buffer,
		func(header *tar.Header, read func() ([]byte, error)) bool {
			content, readErr = read()
			return true
		},
	)

	// Assert
	require.NoError(t, readErr)
	require.EqualValues(t, "foobar", string(content))
}

// Test that the tarball is listed.
func TestListFilesInTarball(t *testing.T) {
	// Arrange
	expectedFilenames := []string{"aaa", "bbb", "ccc"}
	var buffer bytes.Buffer
	tarballWriter := NewTarballWriter(&buffer)
	for _, filename := range expectedFilenames {
		tarballWriter.AddContent(filename, []byte(filename), time.Now())
	}
	tarballWriter.Close()

	// Act
	actualFilenames, err := ListFilesInTarball(&buffer)

	// Assert
	require.NoError(t, err)
	require.EqualValues(t, expectedFilenames, actualFilenames)
}

// Test that the searching function found the file.
func TestSearchFileInTarball(t *testing.T) {
	// Arrange
	expectedFilenames := []string{"aaa", "bbb", "ccc"}
	var buffer bytes.Buffer
	tarballWriter := NewTarballWriter(&buffer)
	for _, filename := range expectedFilenames {
		tarballWriter.AddContent(filename, []byte(filename), time.Now())
	}
	tarballWriter.Close()

	// Act
	content, err := SearchFileInTarball(&buffer, "bbb")

	// Assert
	require.NoError(t, err)
	require.EqualValues(t, "bbb", string(content))
}
