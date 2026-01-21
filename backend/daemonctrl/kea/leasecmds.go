package keactrl

import (
	errors "github.com/pkg/errors"

	"isc.org/stork/datamodel/daemonname"
)

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
	Lease4GetByState     CommandName = "lease4-get-by-state"
	Lease6GetByState     CommandName = "lease6-get-by-state"
)

type LeaseState int

const (
	LeaseStateAssigned         = 0
	LeaseStateDeclined         = 1
	LeaseStateExpiredReclaimed = 2
	LeaseStateReleased         = 3
	LeaseStateRegistered       = 4
)

const (
	LeaseStateAssignedStr         = "assigned"
	LeaseStateDeclinedStr         = "declined"
	LeaseStateExpiredReclaimedStr = "expired-reclaimed"
	LeaseStateReleasedStr         = "released"
	LeaseStateRegisteredStr       = "registered"
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

// Create lease4-get-by-state command.
func NewCommandLease4GetByState(state LeaseState) *Command {
	return newCommand(Lease4GetByState, daemonname.DHCPv4, map[string]any{
		"state": int(state),
	})
}

// Create lease6-get-by-state command.
func NewCommandLease6GetByState(state LeaseState) *Command {
	return newCommand(Lease6GetByState, daemonname.DHCPv6, map[string]any{
		"state": int(state),
	})
}

func ParseLeaseState(input string) (LeaseState, error) {
	switch input {
	case LeaseStateAssignedStr:
		return LeaseStateAssigned, nil
	case LeaseStateDeclinedStr:
		return LeaseStateDeclined, nil
	case LeaseStateExpiredReclaimedStr:
		return LeaseStateExpiredReclaimed, nil
	case LeaseStateReleasedStr:
		return LeaseStateReleased, nil
	case LeaseStateRegisteredStr:
		return LeaseStateRegistered, nil
	default:
		return LeaseStateAssigned, errors.Errorf(
			"invalid lease state string: '%s'", input,
		)
	}
}
