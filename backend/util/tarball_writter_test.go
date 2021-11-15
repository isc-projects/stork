package storkutil

import (
	"bytes"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// Test that the Tarball writer is properly constructed.
func TestConstructNewTarballWriter(t *testing.T) {
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
	require.Len(t, buffer.Bytes(), 32)
}

// Test that the binary content is added to the tarball.
func TestTarballAddContent(t *testing.T) {
	// Arrange
	var buffer bytes.Buffer
	writer := NewTarballWriter(&buffer)

	// Act
	content := "Hello World!"
	err := writer.AddContent("foo", []byte(content), time.Time{})
	writer.Close()

	// Assert
	require.NoError(t, err)
	require.Len(t, buffer.Bytes(), 94)
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
	require.Len(t, buffer.Bytes(), 70)
}

// Test that the the file is added to the tarbal.
func TestTarballAddFile(t *testing.T) {
	// Arrange
	var buffer bytes.Buffer
	writer := NewTarballWriter(&buffer)
	file, _ := ioutil.TempFile("", "*")
	defer os.Remove(file.Name())
	defer file.Close()
	file.WriteString("Hello World!")
	stat, _ := file.Stat()

	// Act
	err := writer.AddFile(file.Name(), stat)
	writer.Close()

	// Assert
	require.NoError(t, err)
	require.GreaterOrEqual(t, buffer.Len(), 100)
}
