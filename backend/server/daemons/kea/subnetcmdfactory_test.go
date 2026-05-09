package kea

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	keaconfig "isc.org/stork/daemoncfg/kea"
	"isc.org/stork/datamodel/daemonname"
	dbmodel "isc.org/stork/server/database/model"
	storkutil "isc.org/stork/util"
)

// Creates a test daemon with the specified name, server tag and hooks. If the
// config backend hook is included, a config database will also be configured.
func newTestDaemonWithConfig(t *testing.T, name daemonname.Name, serverTag *string, hooks ...hook) *dbmodel.Daemon {
	hookLibraries := []map[string]any{}
	var configDatabases []map[string]any
	for _, h := range hooks {
		var library string
		switch h {
		case hookSubnetCmds:
			library = "libdhcp_subnet_cmds.so"
		case hookCbCmds:
			library = "libdhcp_cb_cmds.so"
			configDatabase := map[string]any{
				"name": "keatest", "host": "localhost", "type": "mysql", "port": 5432,
			}
			configDatabases = append(configDatabases, configDatabase)
		default:
			t.Fatalf("unrecognized hook type %d", h)
		}
		hookLibraries = append(hookLibraries, map[string]any{"library": library})
	}

	configHeader := ""
	switch name {
	case daemonname.DHCPv4:
		configHeader = "Dhcp4"
	case daemonname.DHCPv6:
		configHeader = "Dhcp6"
	default:
		t.Fatalf("unrecognized daemon name %s", name)
	}

	configBody := map[string]any{
		"hooks-libraries": hookLibraries,
	}
	if len(configDatabases) > 0 {
		configBody["config-control"] = map[string]any{
			"config-databases": configDatabases,
		}
	}
	if serverTag != nil {
		configBody["server-tag"] = *serverTag
	}

	configMap := map[string]any{
		configHeader: configBody,
	}
	json, err := json.Marshal(configMap)
	require.NoError(t, err)

	config, err := keaconfig.NewConfig(json)
	require.NoError(t, err)
	return &dbmodel.Daemon{
		Name: name,
		KeaDaemon: &dbmodel.KeaDaemon{
			ServerTag: serverTag,
			Config:    &dbmodel.KeaConfig{Config: config},
		},
	}
}

// Constructs a minimal subnet with a matching Daemon entry.
func newTestSubnet(daemons ...*dbmodel.Daemon) *dbmodel.Subnet {
	prefix := "192.0.2.0/24"
	if len(daemons) > 0 && daemons[0].Name == daemonname.DHCPv6 {
		prefix = "2001:db8:1::/64"
	}

	var localSubnets []*dbmodel.LocalSubnet
	for i, d := range daemons {
		d.ID = int64(i + 1)
		localSubnet := &dbmodel.LocalSubnet{
			ID:            2,
			SubnetID:      3,
			LocalSubnetID: 42,
			DaemonID:      d.ID,
			Daemon:        d,
		}
		localSubnets = append(localSubnets, localSubnet)
	}

	return &dbmodel.Subnet{
		ID:           3,
		Prefix:       prefix,
		LocalSubnets: localSubnets,
	}
}

// Tests that the CB hook enum is returned when only the CB hook is loaded.
func TestGetHookForAlteringSubnetsCbCmds(t *testing.T) {
	// Arrange
	daemon := newTestDaemonWithConfig(t, daemonname.DHCPv4, nil, hookCbCmds)

	// Act
	hook, err := getHookForAlteringSubnets(daemon)

	// Assert
	require.NoError(t, err)
	require.Equal(t, hookCbCmds, hook)
}

// Tests that the subnet_cmds hook is returned when only the subnet_cmds hook
// is loaded.
func TestGetHookForAlteringSubnetsSubnetCmds(t *testing.T) {
	// Arrange
	daemon := newTestDaemonWithConfig(t, daemonname.DHCPv4, nil, hookSubnetCmds)

	// Act
	hook, err := getHookForAlteringSubnets(daemon)

	// Assert
	require.NoError(t, err)
	require.Equal(t, hookSubnetCmds, hook)
}

// Tests that CB hook takes precedence over subnet_cmds when both are loaded.
func TestGetHookForAlteringSubnetsCbCmdsPrecedence(t *testing.T) {
	// Arrange
	daemon := newTestDaemonWithConfig(t, daemonname.DHCPv4, nil, hookSubnetCmds, hookCbCmds)

	// Act
	hook, err := getHookForAlteringSubnets(daemon)

	// Assert
	require.NoError(t, err)
	require.Equal(t, hookCbCmds, hook)
}

// Tests that the error is returned when neither hook is loaded.
func TestGetHookForAlteringSubnetsNeither(t *testing.T) {
	// Arrange
	daemon := newTestDaemonWithConfig(t, daemonname.DHCPv4, nil)

	// Act
	hook, err := getHookForAlteringSubnets(daemon)

	// Assert
	require.ErrorContains(t, err, "no subnet_cmds nor cb_cmds hook library found")
	require.Zero(t, hook)
}

// Tests creating subnet_cmds commands for an IPv4 subnet.
func TestCreateSubnetCmdsAddCommandsIPv4(t *testing.T) {
	// Arrange
	daemon := newTestDaemonWithConfig(t, daemonname.DHCPv4, nil, hookSubnetCmds)
	subnet := newTestSubnet(daemon)
	lookup := dbmodel.NewDHCPOptionDefinitionLookup()

	// Act
	cmds, err := createSubnetCmdsAddCommands(subnet.LocalSubnets[0], subnet, "", lookup)

	// Assert
	require.NoError(t, err)
	require.Len(t, cmds, 1)
	marshalled, err := cmds[0].Command.Marshal()
	require.NoError(t, err)
	require.JSONEq(t, `{
		"command": "subnet4-add",
		"service": ["dhcp4"],
		"arguments": {
			"subnet4": [{"id": 42, "subnet": "192.0.2.0/24"}]
		}
	}`, string(marshalled))
	require.Equal(t, daemon, cmds[0].Daemon)
}

// Tests creating subnet_cmds commands for an IPv6 subnet.
func TestCreateSubnetCmdsAddCommandsIPv6(t *testing.T) {
	// Arrange
	daemon := newTestDaemonWithConfig(t, daemonname.DHCPv6, nil, hookSubnetCmds)
	lookup := dbmodel.NewDHCPOptionDefinitionLookup()
	subnet := newTestSubnet(daemon)

	// Act
	cmds, err := createSubnetCmdsAddCommands(subnet.LocalSubnets[0], subnet, "", lookup)

	// Assert
	require.NoError(t, err)
	require.Len(t, cmds, 1)

	marshalled, err := cmds[0].Command.Marshal()
	require.NoError(t, err)
	require.JSONEq(t, `{
		"command": "subnet6-add",
		"service": ["dhcp6"],
		"arguments": {
			"subnet6": [{"id": 42, "subnet": "2001:db8:1::/64"}]
		}
	}`, string(marshalled))
}

// Tests that createSubnetCmdsAddCommands includes the network4-subnet-add
// command when the subnet belongs to a shared network.
func TestCreateSubnetCmdsAddCommandsIPv4WithSharedNetwork(t *testing.T) {
	// Arrange
	daemon := newTestDaemonWithConfig(t, daemonname.DHCPv4, nil, hookSubnetCmds)
	subnet := newTestSubnet(daemon)
	lookup := dbmodel.NewDHCPOptionDefinitionLookup()

	// Act
	cmds, err := createSubnetCmdsAddCommands(subnet.LocalSubnets[0], subnet, "mynet", lookup)

	// Assert
	require.NoError(t, err)
	require.Len(t, cmds, 2)

	marshalled0, err := cmds[0].Command.Marshal()
	require.NoError(t, err)
	require.JSONEq(t, `{
		"command": "subnet4-add",
		"service": ["dhcp4"],
		"arguments": {"subnet4": [{"id": 42, "subnet": "192.0.2.0/24"}]}
	}`, string(marshalled0))

	marshalled1, err := cmds[1].Command.Marshal()
	require.NoError(t, err)
	require.JSONEq(t, `{
		"command": "network4-subnet-add",
		"service": ["dhcp4"],
		"arguments": {"name": "mynet", "id": 42}
	}`, string(marshalled1))
}

// Tests creating a cb_cmds set command for an IPv4 subnet when no server
// tag is configured (defaults to "all").
func TestCreateCbCmdsSetCommandIPv4(t *testing.T) {
	// Arrange
	daemon := newTestDaemonWithConfig(t, daemonname.DHCPv4, nil, hookCbCmds)
	subnet := newTestSubnet(daemon)
	lookup := dbmodel.NewDHCPOptionDefinitionLookup()

	// Act
	cmd, err := createSubnetAddCommands(
		subnet.LocalSubnets[0], subnet, "", []string{"all"}, lookup,
	)

	// Assert
	require.NoError(t, err)
	require.Len(t, cmd, 1)
	marshalled, err := cmd[0].Command.Marshal()
	require.NoError(t, err)
	require.JSONEq(t, `{
		"command": "remote-subnet4-set",
		"service": ["dhcp4"],
		"arguments": {
			"subnets": [{"id": 42, "subnet": "192.0.2.0/24", "shared-network-name": ""}],
			"server-tags": ["all"]
		}
	}`, string(marshalled))
	require.Equal(t, daemon, cmd[0].Daemon)
}

// Tests creating a cb_cmds set command for an IPv4 subnet with an explicit
// server tag.
func TestCreateCbCmdsSetCommandIPv4WithServerTag(t *testing.T) {
	// Arrange
	daemon := newTestDaemonWithConfig(t, daemonname.DHCPv4, storkutil.Ptr("server1"), hookCbCmds)
	lookup := dbmodel.NewDHCPOptionDefinitionLookup()
	subnet := newTestSubnet(daemon)

	// Act
	cmd, err := createSubnetAddCommands(
		subnet.LocalSubnets[0], subnet, "", []string{"server1", "server2"}, lookup,
	)

	// Assert
	require.NoError(t, err)
	require.Len(t, cmd, 1)
	marshalled, err := cmd[0].Command.Marshal()
	require.NoError(t, err)
	require.JSONEq(t, `{
		"command": "remote-subnet4-set",
		"service": ["dhcp4"],
		"arguments": {
			"subnets": [{"id": 42, "subnet": "192.0.2.0/24", "shared-network-name": ""}],
			"server-tags": ["server1", "server2"]
		}
	}`, string(marshalled))
}

// Tests creating a cb_cmds set command for an IPv6 subnet.
func TestCreateCbCmdsSetCommandIPv6(t *testing.T) {
	daemon := newTestDaemonWithConfig(t, daemonname.DHCPv6, nil, hookCbCmds)
	subnet := newTestSubnet(daemon)
	lookup := dbmodel.NewDHCPOptionDefinitionLookup()

	// Act
	cmd, err := createSubnetAddCommands(
		subnet.LocalSubnets[0], subnet, "", []string{"all"}, lookup,
	)

	// Assert
	require.NoError(t, err)
	require.Len(t, cmd, 1)
	marshalled, err := cmd[0].Command.Marshal()
	require.NoError(t, err)
	require.JSONEq(t, `{
		"command": "remote-subnet6-set",
		"service": ["dhcp6"],
		"arguments": {
			"subnets": [{"id": 42, "subnet": "2001:db8:1::/64", "shared-network-name": ""}],
			"server-tags": ["all"]
		}
	}`, string(marshalled))
}

// Tests that the shared-network-name is embedded in the subnet when the subnet
// belongs to a shared network.
func TestCreateCbCmdsSetCommandIPv4WithSharedNetwork(t *testing.T) {
	// Arrange
	daemon := newTestDaemonWithConfig(t, daemonname.DHCPv4, nil, hookCbCmds)
	subnet := newTestSubnet(daemon)
	lookup := dbmodel.NewDHCPOptionDefinitionLookup()

	// Act
	cmd, err := createSubnetAddCommands(
		subnet.LocalSubnets[0], subnet, "mynet", []string{"all", "server"}, lookup,
	)

	// Assert
	require.NoError(t, err)
	require.Len(t, cmd, 1)
	marshalled, err := cmd[0].Command.Marshal()
	require.NoError(t, err)
	require.JSONEq(t, `{
		"command": "remote-subnet4-set",
		"service": ["dhcp4"],
		"arguments": {
			"subnets": [{"id": 42, "subnet": "192.0.2.0/24", "shared-network-name": "mynet"}],
			"server-tags": ["all", "server"]
		}
	}`, string(marshalled))
}

// Tests that createSubnetAddCommands returns subnet4-add for a
// subnet_cmds daemon.
func TestCreateSubnetAddCommandsSubnetCmds(t *testing.T) {
	// Arrange
	daemon := newTestDaemonWithConfig(t, daemonname.DHCPv4, nil, hookSubnetCmds)
	subnet := newTestSubnet(daemon)
	lookup := dbmodel.NewDHCPOptionDefinitionLookup()

	// Act
	cmds, err := createSubnetAddCommands(
		subnet.LocalSubnets[0], subnet, "", nil, lookup,
	)

	// Assert
	require.NoError(t, err)
	require.Len(t, cmds, 1)

	marshalled, err := cmds[0].Command.Marshal()
	require.NoError(t, err)
	require.JSONEq(t, `{
		"command": "subnet4-add",
		"service": ["dhcp4"],
		"arguments": {"subnet4": [{"id": 42, "subnet": "192.0.2.0/24"}]}
	}`, string(marshalled))
}

// Tests that createSubnetAddCommands returns a remote-subnet4-set for a
// cb_cmds daemon.
func TestCreateSubnetAddCommandsCbCmds(t *testing.T) {
	// Arrange
	daemon := newTestDaemonWithConfig(t, daemonname.DHCPv4, storkutil.Ptr("server"), hookCbCmds)
	subnet := newTestSubnet(daemon)
	lookup := dbmodel.NewDHCPOptionDefinitionLookup()

	// Act
	cmds, err := createSubnetAddCommands(
		subnet.LocalSubnets[0], subnet, "", []string{"server"}, lookup,
	)

	// Assert
	require.NoError(t, err)
	require.Len(t, cmds, 1)
	marshalled, err := cmds[0].Command.Marshal()
	require.NoError(t, err)
	require.JSONEq(t, `{
		"command": "remote-subnet4-set",
		"service": ["dhcp4"],
		"arguments": {
			"subnets": [{"id": 42, "subnet": "192.0.2.0/24", "shared-network-name": ""}],
			"server-tags": ["server"]
		}
	}`, string(marshalled))
}
