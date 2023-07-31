package storktest

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"isc.org/stork/testutil"
)

// Helper function to skip the current test case if it relies on altering file
// permissions, but the user that runs the tests is unaffected by the file
// permissions. It may occur if the user is a super-user or a filesystem
// doesn't support file permissions.
func SkipIfCurrentUserIgnoresFilePermissions(t *testing.T) {
	sb := testutil.NewSandbox()
	defer sb.Close()

	filepath, err := sb.Write("test", "test")
	require.NoError(t, err)

	// Remove all permissions.
	err = os.Chmod(filepath, 0)
	require.NoError(t, err)
	defer func() {
		_ = os.Chmod(filepath, 0o700)
	}()

	// Try to read.
	_, err = os.ReadFile(filepath)
	if err == nil {
		// File permission ignored.
		t.Skip("Skip the test because the current user is unaffected by the file " +
			"permissions (it may be super-user or the filesystem doesn't " +
			"support file permissions)")
	}
}
