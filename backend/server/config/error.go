package config

import (
	"fmt"
	"strings"
)

// An error returned when some of the desired daemons were not found in the database.
type SomeDaemonsNotFoundError struct {
	daemonIDs []int64
}

// Create new instance of the SomeDaemonsNotFoundError.
func NewSomeDaemonsNotFoundError(daemonIDs ...int64) error {
	return &SomeDaemonsNotFoundError{
		daemonIDs: daemonIDs,
	}
}

// Returns error string.
func (e SomeDaemonsNotFoundError) Error() string {
	if len(e.daemonIDs) == 0 {
		return "daemons were not found for empty set of IDs"
	}
	ids := make([]string, len(e.daemonIDs))
	for i, id := range e.daemonIDs {
		ids[i] = fmt.Sprint(id)
	}
	return fmt.Sprintf("some daemons were not found for IDs: %s", strings.Join(ids, ", "))
}

// An error returned when unexpected set of configs was specified during
// reconfiguration. It may indicate that some configs are missing or there
// are too many configs.
type InvalidConfigsError struct {
	daemonIDs []int64
}

// Create new instance of the InvalidConfigsError.
func NewInvalidConfigsError(daemonIDs ...int64) error {
	return &InvalidConfigsError{
		daemonIDs: daemonIDs,
	}
}

// Returns error string.
func (e InvalidConfigsError) Error() string {
	if len(e.daemonIDs) == 0 {
		return "no configs specified"
	}
	ids := make([]string, len(e.daemonIDs))
	for i, id := range e.daemonIDs {
		ids[i] = fmt.Sprint(id)
	}
	return fmt.Sprintf("invalid set of daemons specified with IDs: %s", strings.Join(ids, ", "))
}

// An error returned when specified host is not found in the database.
type HostNotFoundError struct {
	hostID int64
}

// Create new instance of the HostNotFoundError.
func NewHostNotFoundError(hostID int64) error {
	return &HostNotFoundError{
		hostID: hostID,
	}
}

// Returns error string.
func (e HostNotFoundError) Error() string {
	return fmt.Sprintf("host with ID %d not found", e.hostID)
}

// An error returned when specified shared network is not found in the database.
type SharedNetworkNotFoundError struct {
	sharedNetworkID int64
}

// Create new instance of the SharedNetworkNotFoundError.
func NewSharedNetworkNotFoundError(sharedNetworkID int64) error {
	return &SharedNetworkNotFoundError{
		sharedNetworkID: sharedNetworkID,
	}
}

// Returns error string.
func (e SharedNetworkNotFoundError) Error() string {
	return fmt.Sprintf("shared network with ID %d not found", e.sharedNetworkID)
}

// An error returned when specified subnet is not found in the database.
type SubnetNotFoundError struct {
	subnetID int64
}

// Create new instance of the SubnetNotFoundError.
func NewSubnetNotFoundError(subnetID int64) error {
	return &SubnetNotFoundError{
		subnetID: subnetID,
	}
}

// Returns error string.
func (e SubnetNotFoundError) Error() string {
	return fmt.Sprintf("subnet with ID %d not found", e.subnetID)
}

// An error returned when some of the daemons have no libdhcp_subnet_cmds hook
// library configured.
type NoSubnetCmdsHookError struct{}

// Create new instance of the NoSubnetCmdsHookError.
func NewNoSubnetCmdsHookError() error {
	return &NoSubnetCmdsHookError{}
}

// Returns error string.
func (e NoSubnetCmdsHookError) Error() string {
	return "libdhcp_subnet_cmds hook library not configured for some of the daemons"
}

// An error returned when it was not possible to lock daemons' configuration.
type LockError struct{}

// Creates new instance of the LockError.
func NewLockError() error {
	return &LockError{}
}

// Returns error string.
func (e LockError) Error() string {
	return "problem with locking daemons configuration"
}
