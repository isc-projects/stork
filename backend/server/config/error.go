package config

import "fmt"

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
