package storkutil

import (
	"archive/tar"
	"bytes"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// Test that the Tarball writer is properly constructed.
func TestConstructNewTarballWriter(t *testing.T) {
	// Act
	writer := NewTarballWriter(bytes.NewBufferString("foo"))

	// Assert
	require.NotNil(t, writer)
}

// Test that the Tarball writer is not constructed for empty input.
func TestConstructNewTarballWriterForEmptyData(t *testing.T) {
	// Act
	writer := NewTarballWriter(nil)

	// Assert
	require.Nil(t, writer)
}

// Test that the tarball content is saved after close.
func TestTarballClose(t *testing.T) {
	// Arrange
	var buffer bytes.Buffer

	// Act
	writer := NewTarballWriter(&buffer)
	writer.Close()

	// Assert
	// The empty tarball always has 32 bytes (using Go TAR and GZIP implementations).
	require.Len(t, buffer.Bytes(), 32)
}

// Test that the binary content is added to the tarball.
func TestTarballAddContent(t *testing.T) {
	// Arrange
	var buffer bytes.Buffer
	writer := NewTarballWriter(&buffer)
	content := []byte("Hello World!")

	// Act
	// We need to use the local time for test purposes because
	// the TAR library converts the time internally from UTC to local.
	err := writer.AddContent("foo", content, time.Date(2013, time.Month(12), 11, 10, 9, 8, 7, time.Local))
	writer.Close()

	// Assert
	require.NoError(t, err)
	fileCount := 0
	err = WalkFilesInTarball(bytes.NewReader(buffer.Bytes()), func(header *tar.Header, read func() ([]byte, error)) bool {
		fileCount++
		expectedTime := time.Date(2013, time.Month(12), 11, 10, 9, 8, 0, time.Local)
		require.EqualValues(t, expectedTime, header.ModTime)
		require.EqualValues(t, "foo", header.Name)
		data, err := read()
		require.NoError(t, err)
		require.EqualValues(t, content, data)
		return true
	})
	require.NoError(t, err)
	require.EqualValues(t, 1, fileCount)
}

// Test that the empty content is added to the tarball.
func TestTarballAddEmptyContent(t *testing.T) {
	// Arrange
	var buffer bytes.Buffer
	writer := NewTarballWriter(&buffer)

	// Act
	content := ""
	err := writer.AddContent("foo", []byte(content), time.Time{})
	writer.Close()

	// Assert
	require.NoError(t, err)
	data, err := SearchFileInTarball(&buffer, "foo")
	require.NoError(t, err)
	require.Empty(t, data)
}

// Test that the file is added to the tarball.
func TestTarballAddFile(t *testing.T) {
	// Arrange
	var buffer bytes.Buffer
	writer := NewTarballWriter(&buffer)
	file, _ := os.CreateTemp("", "*")
	defer (func() {
		file.Close()
		os.Remove(file.Name())
	})()
	file.WriteString("Hello World!")
	stat, _ := file.Stat()

	// Act
	err := writer.AddFile(file.Name(), stat)
	writer.Close()

	// Assert
	require.NoError(t, err)
	data, err := SearchFileInTarball(&buffer, file.Name())
	require.NoError(t, err)
	require.EqualValues(t, []byte("Hello World!"), data)
}
