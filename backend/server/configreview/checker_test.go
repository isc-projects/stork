package configreview

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Test that the config checker metadata is constructed properly.
func TestNewCheckerMetadata(t *testing.T) {
	// Act
	metadata := newCheckerMetadata("foo", Triggers{ManualRun, ConfigModified},
		DispatchGroupSelectors{Bind9Daemon, KeaDHCPv4Daemon}, true, CheckerStateInherit)

	// Assert
	require.EqualValues(t, "foo", metadata.Name)
	require.Contains(t, metadata.Triggers, ManualRun)
	require.Contains(t, metadata.Triggers, ConfigModified)
	require.Len(t, metadata.Triggers, 2)
	require.Contains(t, metadata.Selectors, Bind9Daemon)
	require.Contains(t, metadata.Selectors, KeaDHCPv4Daemon)
	require.Len(t, metadata.Selectors, 2)
	require.True(t, metadata.GloballyEnabled)
	require.EqualValues(t, CheckerStateInherit, metadata.State)
}
