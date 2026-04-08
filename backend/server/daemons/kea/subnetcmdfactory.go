package kea

import (
	pkgerrors "github.com/pkg/errors"
	keaconfig "isc.org/stork/daemoncfg/kea"
	keactrl "isc.org/stork/daemonctrl/kea"
	dbmodel "isc.org/stork/server/database/model"
	storkutil "isc.org/stork/util"
)

// Identifies which Kea hook library is used to manage subnets for a
// particular daemon.
type hook int

const (
	// Indicates the libdhcp_subnet_cmds hook library.
	hookSubnetCmds hook = iota
	// Indicates the libdhcp_cb_cmds hook library.
	hookCbCmds
)

// Returns the hook to modify subnets for a daemon. When both libdhcp_cb_cmds
// and libdhcp_subnet_cmds are configured, cb_cmds takes precedence. Returns an
// error wrapping NoSubnetHookError when neither hook is loaded.
func getHookForAlteringSubnets(daemon *dbmodel.Daemon) (hook, error) {
	if daemon == nil || daemon.KeaDaemon == nil || daemon.KeaDaemon.Config == nil {
		return 0, pkgerrors.New("daemon or Kea configuration is nil")
	}
	config := daemon.KeaDaemon.Config.Config

	if _, _, ok := config.GetHookLibrary("libdhcp_cb_cmds"); ok {
		return hookCbCmds, nil
	}
	if _, _, ok := config.GetHookLibrary("libdhcp_subnet_cmds"); ok {
		return hookSubnetCmds, nil
	}
	return 0, pkgerrors.Errorf("no subnet hook nor config backend hook found")
}

// Creates commands to add a subnet via the subnet_cmds hook library.
// Returns a subnet4-add or subnet6-add command, and when the subnet belongs to
// a shared network, also the corresponding network4-subnet-add or
// network6-subnet-add command.
func createSubnetCmdsAddCommands(
	localSubnet *dbmodel.LocalSubnet,
	subnet *dbmodel.Subnet,
	sharedNetworkName string,
	lookup keaconfig.DHCPOptionDefinitionLookup,
) ([]ConfigCommand, error) {
	var commands []ConfigCommand
	switch subnet.GetFamily() {
	case 4:
		subnet4, err := keaconfig.CreateSubnet4(localSubnet.DaemonID, lookup, subnet)
		if err != nil {
			return nil, err
		}
		commands = append(commands, ConfigCommand{
			Command: keactrl.NewCommandSubnet4Add(subnet4, localSubnet.Daemon.Name),
			Daemon:  localSubnet.Daemon,
		})
		if sharedNetworkName != "" {
			commands = append(commands, ConfigCommand{
				Command: keactrl.NewCommandNetwork4SubnetAdd(
					sharedNetworkName,
					localSubnet.LocalSubnetID,
					localSubnet.Daemon.Name,
				),
				Daemon: localSubnet.Daemon,
			})
		}
	default:
		subnet6, err := keaconfig.CreateSubnet6(localSubnet.DaemonID, lookup, subnet)
		if err != nil {
			return nil, err
		}
		commands = append(commands, ConfigCommand{
			Command: keactrl.NewCommandSubnet6Add(subnet6, localSubnet.Daemon.Name),
			Daemon:  localSubnet.Daemon,
		})
		if sharedNetworkName != "" {
			commands = append(commands, ConfigCommand{
				Command: keactrl.NewCommandNetwork6SubnetAdd(
					sharedNetworkName,
					localSubnet.LocalSubnetID,
					localSubnet.Daemon.Name,
				),
				Daemon: localSubnet.Daemon,
			})
		}
	}
	return commands, nil
}

// Creates a remote-subnet4-set or remote-subnet6-set command for a cb_cmds
// daemon. The server tags are derived from the daemon configuration; when the
// server tag is absent, "all" is used.
func createCbCmdsSetCommand(
	localSubnet *dbmodel.LocalSubnet,
	subnet *dbmodel.Subnet,
	sharedNetworkName string,
	serverTags []string,
	lookup keaconfig.DHCPOptionDefinitionLookup,
) (ConfigCommand, error) {
	switch subnet.GetFamily() {
	case 4:
		subnet4, err := keaconfig.CreateSubnet4(localSubnet.DaemonID, lookup, subnet)
		if err != nil {
			return ConfigCommand{}, err
		}
		remoteSubnet := &keaconfig.RemoteSubnet4{
			Subnet4:           subnet4,
			SharedNetworkName: sharedNetworkName,
		}
		return ConfigCommand{
			Command: keactrl.NewCommandRemoteSubnet4Set(
				remoteSubnet,
				serverTags,
				localSubnet.Daemon.Name,
			),
			Daemon: localSubnet.Daemon,
		}, nil
	default:
		subnet6, err := keaconfig.CreateSubnet6(localSubnet.DaemonID, lookup, subnet)
		if err != nil {
			return ConfigCommand{}, err
		}
		remoteSubnet := &keaconfig.RemoteSubnet6{
			Subnet6:           subnet6,
			SharedNetworkName: sharedNetworkName,
		}
		return ConfigCommand{
			Command: keactrl.NewCommandRemoteSubnet6Set(
				remoteSubnet,
				serverTags,
				localSubnet.Daemon.Name,
			),
			Daemon: localSubnet.Daemon,
		}, nil
	}
}

// Creates all commands required to add a subnet for a given local subnet's
// daemon. The hook type is derived from the daemon's configuration.
func createSubnetAddCommands(
	localSubnet *dbmodel.LocalSubnet,
	subnet *dbmodel.Subnet,
	sharedNetworkName string,
	serverTags []string,
	lookup keaconfig.DHCPOptionDefinitionLookup,
) ([]ConfigCommand, error) {
	hook, err := getHookForAlteringSubnets(localSubnet.Daemon)
	if err != nil {
		return nil, err
	}
	switch hook {
	case hookSubnetCmds:
		cmds, err := createSubnetCmdsAddCommands(localSubnet, subnet, sharedNetworkName, lookup)
		if err != nil {
			return nil, err
		}
		return cmds, nil
	case hookCbCmds:
		cmd, err := createCbCmdsSetCommand(localSubnet, subnet, sharedNetworkName, serverTags, lookup)
		if err != nil {
			return nil, err
		}
		return []ConfigCommand{cmd}, nil
	default:
		return nil, pkgerrors.Errorf("unrecognized subnet hook type %d", hook)
	}
}

// Creates commands to make the subnet changes persistent. This command saves
// the subnet directly in the config backend database, so no config-write or
// config-reload is needed afterward.
func createSubnetSaveCommands(daemon *dbmodel.Daemon) ([]ConfigCommand, error) {
	hook, err := getHookForAlteringSubnets(daemon)
	if err != nil {
		return nil, err
	}

	if hook == hookCbCmds {
		// No additional command is needed to save the subnet in the config
		// backend database.
		return []ConfigCommand{}, nil
	}

	var cmds []ConfigCommand
	cmds = append(cmds, ConfigCommand{
		Command: keactrl.NewCommandBase(keactrl.ConfigWrite, daemon.Name),
		Daemon:  daemon,
	})
	version := storkutil.ParseSemanticVersionOrLatest(daemon.Version)
	if version.LessThan(storkutil.NewSemanticVersion(2, 6, 0)) {
		cmds = append(cmds, ConfigCommand{
			Command: keactrl.NewCommandBase(keactrl.ConfigReload, daemon.Name),
			Daemon:  daemon,
		})
	}
	return cmds, nil
}
