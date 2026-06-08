package kea

import (
	pkgerrors "github.com/pkg/errors"
	keaconfig "isc.org/stork/daemoncfg/kea"
	keactrl "isc.org/stork/daemonctrl/kea"
	dbmodel "isc.org/stork/server/database/model"
	storkutil "isc.org/stork/util"
)

// Creates commands to add a subnet via the subnet_cmds hook library.
// Returns a subnet4-add or subnet6-add command, and when the subnet belongs to
// a shared network, also the corresponding network4-subnet-add or
// network6-subnet-add command.
func createSubnetCmdsSubnetAddCommands(
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

// Creates commands to add a subnet via the cb_cmds hook library. Returns a
// remote-subnet4-set or remote-subnet6-set command.
func createCBCmdsSubnetAddCommands(
	localSubnet *dbmodel.LocalSubnet,
	subnet *dbmodel.Subnet,
	sharedNetworkName string,
	serverTags []string,
	lookup keaconfig.DHCPOptionDefinitionLookup,
) ([]ConfigCommand, error) {
	switch subnet.GetFamily() {
	case 4:
		subnet4, err := keaconfig.CreateSubnet4(localSubnet.DaemonID, lookup, subnet)
		if err != nil {
			return nil, err
		}
		return []ConfigCommand{{
			Command: keactrl.NewCommandRemoteSubnet4Set(
				keaconfig.CreateConfigBackendSubnet4(subnet4, sharedNetworkName),
				serverTags,
				localSubnet.Daemon.Name,
			),
			Daemon: localSubnet.Daemon,
		}}, nil
	default:
		subnet6, err := keaconfig.CreateSubnet6(localSubnet.DaemonID, lookup, subnet)
		if err != nil {
			return nil, err
		}
		return []ConfigCommand{{
			Command: keactrl.NewCommandRemoteSubnet6Set(
				keaconfig.CreateConfigBackendSubnet6(subnet6, sharedNetworkName),
				serverTags,
				localSubnet.Daemon.Name,
			),
			Daemon: localSubnet.Daemon,
		}}, nil
	}
}

// Creates commands to update a subnet via the subnet_cmds hook library.
// Returns subnet4-update/subnet6-update for existing associations and
// subnet4-add/subnet6-add for new associations. If the shared-network
// association changed, it also emits network4-subnet-del/network6-subnet-del
// and network4-subnet-add/network6-subnet-add commands as needed.
func createSubnetCmdsSubnetUpdateCommands(
	localSubnet *dbmodel.LocalSubnet,
	subnet *dbmodel.Subnet,
	sharedNetworkNameBeforeUpdate string,
	sharedNetworkNameAfterUpdate string,
	existingAssociation bool,
	lookup keaconfig.DHCPOptionDefinitionLookup,
) ([]ConfigCommand, error) {
	var commands []ConfigCommand
	switch subnet.GetFamily() {
	case 4:
		subnet4, err := keaconfig.CreateSubnet4(localSubnet.DaemonID, lookup, subnet)
		if err != nil {
			return nil, err
		}
		if existingAssociation {
			commands = append(commands, ConfigCommand{
				Command: keactrl.NewCommandSubnet4Update(subnet4, localSubnet.Daemon.Name),
				Daemon:  localSubnet.Daemon,
			})
		} else {
			commands = append(commands, ConfigCommand{
				Command: keactrl.NewCommandSubnet4Add(subnet4, localSubnet.Daemon.Name),
				Daemon:  localSubnet.Daemon,
			})
		}
		if sharedNetworkNameBeforeUpdate != sharedNetworkNameAfterUpdate {
			if sharedNetworkNameBeforeUpdate != "" {
				commands = append(commands, ConfigCommand{
					Command: keactrl.NewCommandNetwork4SubnetDel(
						sharedNetworkNameBeforeUpdate,
						localSubnet.LocalSubnetID,
						localSubnet.Daemon.Name,
					),
					Daemon: localSubnet.Daemon,
				})
			}
			if sharedNetworkNameAfterUpdate != "" {
				commands = append(commands, ConfigCommand{
					Command: keactrl.NewCommandNetwork4SubnetAdd(
						sharedNetworkNameAfterUpdate,
						localSubnet.LocalSubnetID,
						localSubnet.Daemon.Name,
					),
					Daemon: localSubnet.Daemon,
				})
			}
		}
	default:
		subnet6, err := keaconfig.CreateSubnet6(localSubnet.DaemonID, lookup, subnet)
		if err != nil {
			return nil, err
		}
		if existingAssociation {
			commands = append(commands, ConfigCommand{
				Command: keactrl.NewCommandSubnet6Update(subnet6, localSubnet.Daemon.Name),
				Daemon:  localSubnet.Daemon,
			})
		} else {
			commands = append(commands, ConfigCommand{
				Command: keactrl.NewCommandSubnet6Add(subnet6, localSubnet.Daemon.Name),
				Daemon:  localSubnet.Daemon,
			})
		}
		if sharedNetworkNameBeforeUpdate != sharedNetworkNameAfterUpdate {
			if sharedNetworkNameBeforeUpdate != "" {
				commands = append(commands, ConfigCommand{
					Command: keactrl.NewCommandNetwork6SubnetDel(
						sharedNetworkNameBeforeUpdate,
						localSubnet.LocalSubnetID,
						localSubnet.Daemon.Name,
					),
					Daemon: localSubnet.Daemon,
				})
			}
			if sharedNetworkNameAfterUpdate != "" {
				commands = append(commands, ConfigCommand{
					Command: keactrl.NewCommandNetwork6SubnetAdd(
						sharedNetworkNameAfterUpdate,
						localSubnet.LocalSubnetID,
						localSubnet.Daemon.Name,
					),
					Daemon: localSubnet.Daemon,
				})
			}
		}
	}
	return commands, nil
}

// Creates commands to update a subnet via the cb_cmds hook library.
// Returns a remote-subnet4-set or remote-subnet6-set command.
func createCBCmdsSubnetUpdateCommands(
	localSubnet *dbmodel.LocalSubnet,
	subnet *dbmodel.Subnet,
	sharedNetworkNameAfterUpdate string,
	serverTags []string,
	lookup keaconfig.DHCPOptionDefinitionLookup,
) ([]ConfigCommand, error) {
	switch subnet.GetFamily() {
	case 4:
		subnet4, err := keaconfig.CreateSubnet4(localSubnet.DaemonID, lookup, subnet)
		if err != nil {
			return nil, err
		}
		return []ConfigCommand{{
			Command: keactrl.NewCommandRemoteSubnet4Set(
				keaconfig.CreateConfigBackendSubnet4(subnet4, sharedNetworkNameAfterUpdate),
				serverTags,
				localSubnet.Daemon.Name,
			),
			Daemon: localSubnet.Daemon,
		}}, nil
	default:
		subnet6, err := keaconfig.CreateSubnet6(localSubnet.DaemonID, lookup, subnet)
		if err != nil {
			return nil, err
		}
		return []ConfigCommand{{
			Command: keactrl.NewCommandRemoteSubnet6Set(
				keaconfig.CreateConfigBackendSubnet6(subnet6, sharedNetworkNameAfterUpdate),
				serverTags,
				localSubnet.Daemon.Name,
			),
			Daemon: localSubnet.Daemon,
		}}, nil
	}
}

// Creates all commands required to update a subnet for a given local subnet's
// daemon. The hook type is derived from the daemon's configuration.
func createSubnetUpdateCommands(
	localSubnet *dbmodel.LocalSubnet,
	subnet *dbmodel.Subnet,
	sharedNetworkNameBeforeUpdate string,
	sharedNetworkNameAfterUpdate string,
	existingAssociation bool,
	serverTags []string,
	lookup keaconfig.DHCPOptionDefinitionLookup,
) ([]ConfigCommand, error) {
	hook := localSubnet.Daemon.KeaDaemon.Config.GetHookLibraries().GetSubnetAndSharedNetworkAlteringHookLibrary()

	switch hook {
	case keaconfig.SubnetAndSharedNetworkAlteringHookLibrarySubnetCmds:
		return createSubnetCmdsSubnetUpdateCommands(
			localSubnet,
			subnet,
			sharedNetworkNameBeforeUpdate,
			sharedNetworkNameAfterUpdate,
			existingAssociation,
			lookup,
		)
	case keaconfig.SubnetAndSharedNetworkAlteringHookLibraryCBCmds:
		return createCBCmdsSubnetUpdateCommands(
			localSubnet,
			subnet,
			sharedNetworkNameAfterUpdate,
			serverTags,
			lookup,
		)
	default:
		return nil, pkgerrors.Errorf("cannot determine hook library for altering subnets")
	}
}

// Creates all commands required to remove a subnet for a given daemon. The
// hook type is derived from the daemon's configuration.
func createSubnetDeleteCommands(
	localSubnet *dbmodel.LocalSubnet,
	family int,
	sharedNetworkNameBeforeUpdate string,
) ([]ConfigCommand, error) {
	hook := localSubnet.Daemon.KeaDaemon.Config.GetHookLibraries().GetSubnetAndSharedNetworkAlteringHookLibrary()

	var commands []ConfigCommand
	switch hook {
	case keaconfig.SubnetAndSharedNetworkAlteringHookLibrarySubnetCmds:
		if sharedNetworkNameBeforeUpdate != "" {
			// If the deleted subnet belongs to a shared network we first need to remove
			// this subnet from a shared network. This is a Kea limitation described in
			// https://gitlab.isc.org/isc-projects/kea/-/issues/3455.
			commands = append(commands, ConfigCommand{
				Command: keactrl.NewCommandNetworkSubnetDel(
					family,
					sharedNetworkNameBeforeUpdate,
					localSubnet.LocalSubnetID,
					localSubnet.Daemon.Name,
				),
				Daemon: localSubnet.Daemon,
			})
		}
		commands = append(commands, ConfigCommand{
			Command: keactrl.NewCommandSubnetDel(
				family,
				&keaconfig.SubnetCmdsDeletedSubnet{ID: localSubnet.LocalSubnetID},
				localSubnet.Daemon.Name,
			),
			Daemon: localSubnet.Daemon,
		})
		return commands, nil
	case keaconfig.SubnetAndSharedNetworkAlteringHookLibraryCBCmds:
		switch family {
		case 4:
			commands = append(commands, ConfigCommand{
				Command: keactrl.NewCommandRemoteSubnet4DelByID(localSubnet.LocalSubnetID, localSubnet.Daemon.Name),
				Daemon:  localSubnet.Daemon,
			})
		default:
			commands = append(commands, ConfigCommand{
				Command: keactrl.NewCommandRemoteSubnet6DelByID(localSubnet.LocalSubnetID, localSubnet.Daemon.Name),
				Daemon:  localSubnet.Daemon,
			})
		}
		return commands, nil
	default:
		return nil, pkgerrors.Errorf("cannot determine hook library for altering subnets")
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
	hook := localSubnet.Daemon.KeaDaemon.Config.GetHookLibraries().GetSubnetAndSharedNetworkAlteringHookLibrary()

	switch hook {
	case keaconfig.SubnetAndSharedNetworkAlteringHookLibrarySubnetCmds:
		return createSubnetCmdsSubnetAddCommands(localSubnet, subnet, sharedNetworkName, lookup)
	case keaconfig.SubnetAndSharedNetworkAlteringHookLibraryCBCmds:
		return createCBCmdsSubnetAddCommands(localSubnet, subnet, sharedNetworkName, serverTags, lookup)
	default:
		return nil, pkgerrors.Errorf("cannot determine hook library for altering subnets")
	}
}

// Creates commands to make subnet changes persistent. This is needed for
// daemons with subnet_cmds hooks. For daemons running cb_cmds no additional
// commands are created.
func createSubnetSaveCommands(daemon *dbmodel.Daemon) ([]ConfigCommand, error) {
	hook := daemon.KeaDaemon.Config.GetHookLibraries().GetSubnetAndSharedNetworkAlteringHookLibrary()

	if hook == keaconfig.SubnetAndSharedNetworkAlteringHookLibraryCBCmds {
		// No additional command is needed to save the subnet in the config
		// backend database.
		return []ConfigCommand{}, nil
	}
	if hook == keaconfig.SubnetAndSharedNetworkAlteringHookLibraryNone || hook == keaconfig.SubnetAndSharedNetworkAlteringHookLibraryBoth {
		return nil, pkgerrors.Errorf("cannot determine hook library for altering subnets")
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
