package keactrl

import "isc.org/stork/datamodel/daemonname"

// Lease type specified in the commands.
type LeaseType string

const (
	LeaseTypeNA LeaseType = "IA_NA"
	LeaseTypePD LeaseType = "IA_PD"
)

const (
	Lease4Get            CommandName = "lease4-get"
	Lease6Get            CommandName = "lease6-get"
	Lease4GetByClientID  CommandName = "lease4-get-by-client-id"
	Lease6GetByDUID      CommandName = "lease6-get-by-duid"
	Lease4GetByHostname  CommandName = "lease4-get-by-hostname"
	Lease6GetByHostname  CommandName = "lease6-get-by-hostname"
	Lease4GetByHWAddress CommandName = "lease4-get-by-hw-address"
)

// Creates lease4-get command.
func NewCommandLease4Get(ipAddress string) *Command {
	return newCommand(Lease4Get, daemonname.DHCPv4, map[string]any{"ip-address": ipAddress})
}

// Creates lease6-get command.
func NewCommandLease6Get(leaseType LeaseType, ipAddress string) *Command {
	return newCommand(Lease6Get, daemonname.DHCPv6, map[string]any{
		"type":       leaseType,
		"ip-address": ipAddress,
	})
}

// Creates lease4-get-by-hw-address command.
func NewCommandLease4GetByHWAddress(hwAddress string) *Command {
	return newCommand(Lease4GetByHWAddress, daemonname.DHCPv4, map[string]any{"hw-address": hwAddress})
}

// Creates lease4-get-by-client-id command.
func NewCommandLease4GetByClientID(clientID string) *Command {
	return newCommand(Lease4GetByClientID, daemonname.DHCPv4, map[string]any{"client-id": clientID})
}

// Creates lease6-get-by-duid command.
func NewCommandLease6GetByDUID(duid string) *Command {
	return newCommand(Lease6GetByDUID, daemonname.DHCPv6, map[string]any{"duid": duid})
}

// Creates lease4-get-by-hostname command.
func NewCommandLease4GetByHostname(hostname string) *Command {
	return newCommand(Lease4GetByHostname, daemonname.DHCPv4, map[string]any{"hostname": hostname})
}

// Creates lease6-get-by-hostname command.
func NewCommandLease6GetByHostname(hostname string) *Command {
	return newCommand(Lease6GetByHostname, daemonname.DHCPv6, map[string]any{"hostname": hostname})
}
