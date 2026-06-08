package keactrl

import (
	keaconfig "isc.org/stork/daemoncfg/kea"
	"isc.org/stork/datamodel/daemonname"
)

const (
	RemoteSubnet4Set CommandName = "remote-subnet4-set"
	RemoteSubnet6Set CommandName = "remote-subnet6-set"
	RemoteSubnet4Del CommandName = "remote-subnet4-del-by-id"
	RemoteSubnet6Del CommandName = "remote-subnet6-del-by-id"
)

// Creates a remote-subnet4-set command. The command updates or inserts an IPv4
// subnet in the Kea configuration backend database. It does not include the
// remote parameter, so Kea uses the first configured config database by default.
func NewCommandRemoteSubnet4Set(subnet *keaconfig.ConfigBackendSubnet4, serverTags []string, daemonName daemonname.Name) *Command {
	return NewCommandBase(RemoteSubnet4Set, daemonName).
		WithArrayArgument("subnets", subnet).
		WithArgument("server-tags", serverTags)
}

// Creates a remote-subnet6-set command. The command updates or inserts an IPv6
// subnet in the Kea configuration backend database. It does not include the
// remote parameter, so Kea uses the first configured config database by default.
func NewCommandRemoteSubnet6Set(subnet *keaconfig.ConfigBackendSubnet6, serverTags []string, daemonName daemonname.Name) *Command {
	return NewCommandBase(RemoteSubnet6Set, daemonName).
		WithArrayArgument("subnets", subnet).
		WithArgument("server-tags", serverTags)
}

// Creates a remote-subnet4-del-by-id command. The command deletes an IPv4
// subnet from the Kea configuration backend database. It does not include the
// remote parameter, so Kea uses the first configured config database by
// default.
func NewCommandRemoteSubnet4DelByID(subnetID int64, daemonName daemonname.Name) *Command {
	return NewCommandBase(RemoteSubnet4Del, daemonName).
		WithArrayArgument("subnets", map[string]int64{"id": subnetID})
}

// Creates a remote-subnet6-del-by-id command. The command deletes an IPv6
// subnet from the Kea configuration backend database. It does not include the
// remote parameter, so Kea uses the first configured config database by
// default.
func NewCommandRemoteSubnet6DelByID(subnetID int64, daemonName daemonname.Name) *Command {
	return NewCommandBase(RemoteSubnet6Del, daemonName).
		WithArrayArgument("subnets", map[string]int64{"id": subnetID})
}
