package keactrl

import keaconfig "isc.org/stork/appcfg/kea"

const (
	ReservationAdd     CommandName = "reservation-add"
	ReservationDel     CommandName = "reservation-del"
	ReservationGetPage CommandName = "reservation-get-page"
)

// Creates reservation-add command.
func NewCommandReservationAdd(reservation *keaconfig.HostCmdsReservation, daemonNames ...DaemonName) *Command {
	return NewCommandBase(ReservationAdd, daemonNames...).
		WithArgument("reservation", reservation)
}

// Creates reservation-del command.
func NewCommandReservationDel(reservation *keaconfig.HostCmdsDeletedReservation, daemonNames ...DaemonName) *Command {
	return NewCommandBase(ReservationDel, daemonNames...).WithArguments(reservation)
}

// Creates reservation-get-page command. The arguments from and source-index
// are only included in the command when they are greater than 0.
func NewCommandReservationGetPage(localSubnetID, sourceIndex, from, limit int64, daemons ...DaemonName) *Command {
	command := NewCommandBase(ReservationGetPage, daemons...).
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
