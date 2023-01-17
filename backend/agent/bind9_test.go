package agent

import (
	"testing"

	"github.com/stretchr/testify/require"
	storkutil "isc.org/stork/util"
)

// Test the function which extracts the list of log files from the Bind9
// application by sending the request to the Kea Control Agent and the
// daemons behind it.
func TestBind9AllowedLogs(t *testing.T) {
	ba := &Bind9App{}
	paths, err := ba.DetectAllowedLogs()
	require.NoError(t, err)
	require.Len(t, paths, 0)
}

// Check if getPotentialNamedConfLocations returns paths.
func TestGetPotentialNamedConfLocations(t *testing.T) {
	paths := getPotentialNamedConfLocations()
	require.Greater(t, len(paths), 1)
}

// Test that the system command executor returns a proper output.
func TestSystemCommandExecutorOutput(t *testing.T) {
	// Arrange
	executor := storkutil.NewSystemCommandExecutor()

	// Act
	output, err := executor.Output("echo", "-n", "foo")

	// Assert
	require.NoError(t, err)
	require.EqualValues(t, "foo", string(output))
}

// Test that the system command executor returns an error for invalid command.
func TestSystemCommandExecutorOnFail(t *testing.T) {
	// Arrange
	executor := storkutil.NewSystemCommandExecutor()

	// Act
	output, err := executor.Output("non-exist-command")

	// Assert
	require.Error(t, err)
	require.Nil(t, output)
}
