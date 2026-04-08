package kea

import (
	"testing"

	"github.com/stretchr/testify/require"
	"isc.org/stork/datamodel/daemonname"
	dbmodel "isc.org/stork/server/database/model"
)

// Tests that the server tag defaults to "all" when no server-tag is present in
// the daemon configuration.
func TestBuildConfigBackendIDNoServerTag(t *testing.T) {
	// Arrange
	daemon := newTestDaemonWithConfig(t, daemonname.DHCPv4, "", hookCbCmds)

	// Act
	id, err := buildConfigBackendID(daemon)

	// Assert
	require.NoError(t, err)
	require.Equal(t, "all", id.ServerTag)
	require.Equal(t, "keatest", id.DBName)
	require.Equal(t, "localhost", id.DBHost)
	require.Equal(t, 5432, id.DBPort)
}

// Tests that the server tag is included in the config backend ID.
func TestBuildConfigBackendIDWithServerTag(t *testing.T) {
	// Arrange
	daemon := newTestDaemonWithConfig(t, daemonname.DHCPv4, "server1", hookCbCmds)

	// Act
	id, err := buildConfigBackendID(daemon)

	// Assert
	require.NoError(t, err)
	require.Equal(t, "server1", id.ServerTag)
	require.Equal(t, "keatest", id.DBName)
	require.Equal(t, "localhost", id.DBHost)
	require.Equal(t, 5432, id.DBPort)
}

// Tests that an error is returned when no config databases are configured.
func TestBuildConfigBackendIDNoConfigDB(t *testing.T) {
	// Arrange
	daemon := &dbmodel.Daemon{
		Name:      daemonname.DHCPv4,
		KeaDaemon: &dbmodel.KeaDaemon{},
	}

	err := daemon.SetKeaConfigFromJSON([]byte(`{ "Dhcp4": {} }`))
	require.NoError(t, err)

	// Act
	_, err = buildConfigBackendID(daemon)

	// Assert
	require.ErrorContains(t, err, "no config databases configured")
}

// Tests that target iterator calls a function only once for each unique config
// backend database.
func TestForEachUniqueTargetDeduplicatesCbCmds(t *testing.T) {
	// Arrange
	daemon1 := newTestDaemonWithConfig(t, daemonname.DHCPv4, "server1", hookCbCmds)
	daemon2 := newTestDaemonWithConfig(t, daemonname.DHCPv4, "server1", hookCbCmds)
	daemon2.ID = 2

	subnet := newTestSubnet(daemon1)
	subnet.LocalSubnets = append(subnet.LocalSubnets, &dbmodel.LocalSubnet{
		DaemonID:      daemon2.ID,
		Daemon:        daemon2,
		LocalSubnetID: 42,
		SubnetID:      subnet.ID,
	})

	called := 0

	// Act
	err := forEachUniqueTarget(subnet.LocalSubnets, func(ls *dbmodel.LocalSubnet) error {
		called++
		return nil
	})

	// Assert
	require.NoError(t, err)
	require.Equal(t, 1, called)
}

// Tests that the target iterator calls fn for multiple unique config backend.
func TestForEachUniqueTargetDifferentCbCmdsBackends(t *testing.T) {
	// Arrange
	daemon1 := newTestDaemonWithConfig(t, daemonname.DHCPv4, "server1", hookCbCmds)
	daemon2 := newTestDaemonWithConfig(t, daemonname.DHCPv4, "server2", hookCbCmds)
	daemon2.ID = 2

	subnet := newTestSubnet(daemon1)
	subnet.LocalSubnets = append(subnet.LocalSubnets, &dbmodel.LocalSubnet{
		DaemonID:      daemon2.ID,
		Daemon:        daemon2,
		LocalSubnetID: 42,
		SubnetID:      subnet.ID,
	})

	called := 0

	// Act
	err := forEachUniqueTarget(subnet.LocalSubnets, func(ls *dbmodel.LocalSubnet) error {
		called++
		return nil
	})

	// Assert
	require.NoError(t, err)
	require.Equal(t, 2, called)
}

// Tests that the target iterator processes all daemons with the subnet_cmds
// hook.
func TestForEachUniqueTargetProcessesSubnetCmds(t *testing.T) {
	// Arrange
	daemon1 := newTestDaemonWithConfig(t, daemonname.DHCPv4, "server1", hookSubnetCmds)
	daemon2 := newTestDaemonWithConfig(t, daemonname.DHCPv4, "server2", hookSubnetCmds)
	daemon2.ID = 2

	subnet := newTestSubnet(daemon1)
	subnet.LocalSubnets = append(subnet.LocalSubnets, &dbmodel.LocalSubnet{
		DaemonID:      daemon2.ID,
		Daemon:        daemon2,
		LocalSubnetID: 42,
		SubnetID:      subnet.ID,
	})

	called := 0

	// Act
	err := forEachUniqueTarget(subnet.LocalSubnets, func(ls *dbmodel.LocalSubnet) error {
		called++
		return nil
	})

	// Assert
	require.NoError(t, err)
	require.Equal(t, 2, called)
}

// Tests that the target iterator silently skips local subnets whose daemon
// or Kea configuration is nil.
func TestForEachUniqueTargetSkipsNilConfig(t *testing.T) {
	// Arrange
	daemon := newTestDaemonWithConfig(t, daemonname.DHCPv4, "server1")
	subnet := newTestSubnet(daemon)
	called := 0

	// Act
	err := forEachUniqueTarget(subnet.LocalSubnets, func(ls *dbmodel.LocalSubnet) error {
		called++
		return nil
	})

	// Assert
	require.NoError(t, err)
	require.Equal(t, 0, called)
}
