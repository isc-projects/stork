package testutil

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// Check if Sandbox Join works.
func TestSandboxJoin(t *testing.T) {
	sb := NewSandbox()
	defer sb.Close()

	require.DirExists(t, sb.BasePath)

	// check 'a' file in sandbox root
	aFile, err := sb.Join("a")
	require.NoError(t, err)
	require.FileExists(t, aFile)
	require.True(t, strings.HasSuffix(aFile, "/a"))

	// check 'c' file in sandbox subdir 'b'
	cFile, err := sb.Join("b/c")
	require.NoError(t, err)
	require.FileExists(t, cFile)
	require.True(t, strings.HasSuffix(cFile, "/b/c"))

	// check if all has been created for sure and nothing else :)
	dirCount := 0
	fileCount := 0
	filepath.Walk(sb.BasePath, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			dirCount++
		} else {
			fileCount++
		}
		return nil
	})

	// 2 dirs expected: ., ./b
	require.EqualValues(t, 2, dirCount)

	// 2 files expected: ./a, ./b/c
	require.EqualValues(t, 2, fileCount)
}

// Check if Sandbox Write works.
func TestSandboxWrite(t *testing.T) {
	sb := NewSandbox()
	defer sb.Close()

	fpath, err := sb.Write("abc", "def")
	require.NoError(t, err)
	require.Contains(t, fpath, "abc")

	content, err := os.ReadFile(fpath)
	require.NoError(t, err)
	require.EqualValues(t, "def", content)
}

// Check if write returns error if there is a writing failure.
func TestSandboxWriteFail(t *testing.T) {
	// Create a sandbox and then immediately close it, which will force the removal of the
	// whole directory.
	sb := NewSandbox()
	defer sb.Close()

	// Now we ask the code to write path with illegal character. It should fail.
	fpath, err := sb.Write("/", "abc")
	require.Error(t, err)
	require.Empty(t, fpath)
}

// Check if Sandbox JoinDir works.
func TestSandboxJoinDir(t *testing.T) {
	sb := NewSandbox()
	defer sb.Close()

	require.DirExists(t, sb.BasePath)

	// check 'a' dir in sandbox root
	aDir, err := sb.JoinDir("a")
	require.NoError(t, err)
	require.DirExists(t, aDir)
	require.True(t, strings.HasSuffix(aDir, "/a"))

	// check 'c' dir in sandbox subdir 'b'
	cDir, err := sb.JoinDir("b/c")
	require.NoError(t, err)
	require.DirExists(t, cDir)
	require.True(t, strings.HasSuffix(cDir, "/b/c"))

	// check if all has been created for sure and nothing else :)
	dirCount := 0
	fileCount := 0
	filepath.Walk(sb.BasePath, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			dirCount++
		} else {
			fileCount++
		}
		return nil
	})

	// 4 dirs expected: ., ./a, ./b, ./b/c
	require.EqualValues(t, 4, dirCount)

	// 0 files expected
	require.Zero(t, fileCount)
}

// Check if Sandbox Close works.
func TestSandboxClose(t *testing.T) {
	sb := NewSandbox()
	defer sb.Close()

	sb.Join("a")
	sb.Join("b/c")
	sb.JoinDir("d/e/f")

	count := 0
	filepath.Walk(sb.BasePath, func(path string, info os.FileInfo, err error) error {
		count++
		return nil
	})
	// 7 elems expected
	require.EqualValues(t, 7, count)

	sb.Close()

	require.NoDirExists(t, sb.BasePath)
}
