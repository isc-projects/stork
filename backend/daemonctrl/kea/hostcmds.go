package keactrl

import (
	keaconfig "isc.org/stork/daemoncfg/kea"
	"isc.org/stork/datamodel/daemonname"
)

const (
	ReservationAdd     CommandName = "reservation-add"
	ReservationDel     CommandName = "reservation-del"
	ReservationGetPage CommandName = "reservation-get-page"
)

// Creates reservation-add command.
func NewCommandReservationAdd(reservation *keaconfig.HostCmdsReservation, daemonName daemonname.Name) *Command {
	return NewCommandBase(ReservationAdd, daemonName).
		WithArgument("reservation", reservation)
}

// Creates reservation-del command.
func NewCommandReservationDel(reservation *keaconfig.HostCmdsDeletedReservation, daemonName daemonname.Name) *Command {
	return NewCommandBase(ReservationDel, daemonName).WithArguments(reservation)
}

// Creates reservation-get-page command. The arguments from and source-index
// are only included in the command when they are greater than 0.
func NewCommandReservationGetPage(localSubnetID, sourceIndex, from, limit int64, daemonName daemonname.Name) *Command {
	command := NewCommandBase(ReservationGetPage, daemonName).
		WithArgument("subnet-id", localSubnetID).
		WithArgument("limit", limit)

	if from > 0 {
		command = command.WithArgument("from", from)
	}
	if sourceIndex > 0 {
		command = command.WithArgument("source-index", sourceIndex)
	}
	return command
}
