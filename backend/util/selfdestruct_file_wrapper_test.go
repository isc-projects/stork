package storkutil

import (
	"io"
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

// Test that constructed wrapper is valid.
func TestWrapperIsConstructed(t *testing.T) {
	// Arrange
	file, _ := ioutil.TempFile("", "*")
	defer file.Close()

	// Act
	wrapper := NewSelfDestructFileWrapper(file)

	// Assert
	require.NotNil(t, wrapper)
}

// Test that the wrapper read the file content.
func TestWrapperIsReadable(t *testing.T) {
	// Arrange
	file, _ := ioutil.TempFile("", "*")
	defer file.Close()
	_, _ = file.WriteString("Hello World!")
	_, _ = file.Seek(0, io.SeekStart)
	wrapper := NewSelfDestructFileWrapper(file)

	// Act
	content, err := ioutil.ReadAll(wrapper)

	// Assert
	require.NoError(t, err)
	require.EqualValues(t, "Hello World!", string(content))
}

// Test that the wrapper is closed with no errors.
func TestWrapperIsCloseable(t *testing.T) {
	// Arrange
	file, _ := ioutil.TempFile("", "*")
	wrapper := NewSelfDestructFileWrapper(file)

	// Act
	closeErr := wrapper.Close()
	_, seekErr := file.Seek(0, io.SeekCurrent)

	// Assert
	require.NoError(t, closeErr)
	require.Error(t, seekErr) // File is closed
}

// Test that the wrapper returns the same name as
// the inner file.
func TestWrapperReturnsProperName(t *testing.T) {
	// Arrange
	file, _ := ioutil.TempFile("", "*")
	wrapper := NewSelfDestructFileWrapper(file)

	// Act
	expectedName := file.Name()
	actualName := wrapper.Name()

	// Assert
	require.EqualValues(t, expectedName, actualName)
}

// Test that the file is self-destructed after the file close.
func TestWrapperDeletesAfterClose(t *testing.T) {
	// Arrange
	file, _ := ioutil.TempFile("", "*")
	wrapper := NewSelfDestructFileWrapper(file)
	path := file.Name()

	// Act
	_ = wrapper.Close()
	_, err := os.Stat(path)

	// Assert
	require.ErrorIs(t, err, os.ErrNotExist)
}

// // Test that the file is self-destructed even if the inner file was closed directly.
// func TestWrapperDeletesAfterDirectlyClose(t *testing.T) {
// 	// Arrange
// 	file, _ := ioutil.TempFile("", "*")
// 	wrapper := NewSelfDestructFileWrapper(file)
// 	path := file.Name()

// 	// Act
// 	_ = file.Close()
// 	_, beforeErr := os.Stat(path)
// 	closeErr := wrapper.Close()
// 	_, afterErr := os.Stat(path)

// 	// Assert
// 	require.NoError(t, beforeErr)
// 	require.NoError(t, closeErr)
// 	require.ErrorIs(t, err, os.ErrNotExist)
// }
