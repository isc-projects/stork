package storkutil

import (
	"path"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"isc.org/stork/testutil"
)

// Test that the system command executor is constructed properly.
func TestNewSystemCommandExecutor(t *testing.T) {
	// Arrange & Act
	executor := NewSystemCommandExecutor()

	// Assert
	require.NotNil(t, executor)

	lsPath, err := executor.LookPath("ls")
	require.NotNil(t, lsPath)
	require.Nil(t, err)
	require.True(t, executor.IsFileExist(lsPath))
	sb := testutil.NewSandbox()
	defer sb.Close()
	require.False(t, executor.IsFileExist(path.Join(sb.BasePath, "not-exists")))
}

// Test that the executor returns coorrect file info for a file.
func TestGetFileInfo(t *testing.T) {
	executor := NewSystemCommandExecutor()
	sb := testutil.NewSandbox()
	defer sb.Close()
	_, err := sb.Write("test.txt", "test")
	require.NoError(t, err)

	info, err := executor.GetFileInfo(filepath.Join(sb.BasePath, "test.txt"))
	require.NoError(t, err)
	require.NotNil(t, info)
	require.Equal(t, info.Name(), "test.txt")
	require.Equal(t, info.IsDir(), false)
	require.Equal(t, info.Size(), int64(4))
	require.WithinDuration(t, info.ModTime(), time.Now(), 10*time.Second)
}

// Test that the executor returns an error when attempting to get the file
// information for a non-existent file.
func TestGetFileInfoNotFound(t *testing.T) {
	executor := NewSystemCommandExecutor()
	sb := testutil.NewSandbox()
	defer sb.Close()

	info, err := executor.GetFileInfo(filepath.Join(sb.BasePath, "not-exists"))
	require.ErrorContains(t, err, "cannot get file info")
	require.Nil(t, info)
}
