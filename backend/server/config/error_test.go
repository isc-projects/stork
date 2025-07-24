package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Test creation of an error which indicates that some daemons were not
// found for the specified IDs.
func TestSomeDaemonsNotFoundError(t *testing.T) {
	err := NewSomeDaemonsNotFoundError(1, 2, 3, 4)
	require.EqualError(t, err, "some daemons were not found for IDs: 1, 2, 3, 4")
}

// Test creation of an error which indicates that daemons were not
// found when no IDs have been specified.
func TestSomeDaemonsNotFoundErrorEmptySet(t *testing.T) {
	err := NewSomeDaemonsNotFoundError()
	require.EqualError(t, err, "daemons were not found for empty set of IDs")
}

// Test creation of an error which indicates that unexpected set of configs
// was specified during reconfiguration.
func TestNewInvalidConfigsError(t *testing.T) {
	err := NewInvalidConfigsError(1, 2, 3, 4)
	require.EqualError(t, err, "invalid set of daemons specified with IDs: 1, 2, 3, 4")
}

// Test creation of an error which indicates that unexpected set of configs
// was specified during reconfiguration when no configs have been specified.
func TestNewInvalidConfigsErrorEmptySet(t *testing.T) {
	err := NewInvalidConfigsError()
	require.EqualError(t, err, "no configs specified")
}

// Test creation of an error which indicates that host was not found.
func TestHostNotFoundError(t *testing.T) {
	err := NewHostNotFoundError(123)
	require.EqualError(t, err, "host with ID 123 not found")
}

// Test creation of an error which indicates that shared network was not found.
func TestSharedNetworkNotFoundError(t *testing.T) {
	err := NewSharedNetworkNotFoundError(234)
	require.EqualError(t, err, "shared network with ID 234 not found")
}

// Test creation of an error which indicates that subnet was not found.
func TestSubnetNotFoundError(t *testing.T) {
	err := NewSubnetNotFoundError(234)
	require.EqualError(t, err, "subnet with ID 234 not found")
}

// Test creation of an error which indicates that libdhcp_subnet_cmds was not configured.
func TestNoSubnetCmdsHookError(t *testing.T) {
	err := NewNoSubnetCmdsHookError()
	require.EqualError(t, err, "libdhcp_subnet_cmds hook library not configured for some of the daemons")
}

// Test creation of an error which indicates a problem with locking
// configuration.
func TestLockError(t *testing.T) {
	err := NewLockError()
	require.EqualError(t, err, "problem with locking daemons configuration")
}
