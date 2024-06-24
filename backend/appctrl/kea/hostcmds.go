package keactrl

import keaconfig "isc.org/stork/appcfg/kea"

const (
	ReservationAdd     CommandName = "reservation-add"
	ReservationDel     CommandName = "reservation-del"
	ReservationGetPage CommandName = "reservation-get-page"
)

// Creates reservation-add command.
func NewCommandReservationAdd(reservation *keaconfig.HostCmdsReservation, daemonNames ...string) *Command {
	return NewCommandBase(ReservationAdd, daemonNames...).
		WithArgument("reservation", reservation)
}

// Creates reservation-del command.
func NewCommandReservationDel(reservation *keaconfig.HostCmdsDeletedReservation, daemonNames ...string) *Command {
	return NewCommandBase(ReservationDel, daemonNames...).WithArguments(reservation)
}

// Creates reservation-get-page command.
func NewCommandReservationGetPage(localSubnetID, sourceIndex, from, limit int64, daemons ...string) *Command {
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
