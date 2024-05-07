package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

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
