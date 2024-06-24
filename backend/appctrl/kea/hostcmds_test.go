package keactrl

import (
	"testing"

	require "github.com/stretchr/testify/require"
	keaconfig "isc.org/stork/appcfg/kea"
)

// Tests reservation-add command.
func TestNewCommandReservationAdd(t *testing.T) {
	command := NewCommandReservationAdd(&keaconfig.HostCmdsReservation{
		Reservation: keaconfig.Reservation{
			HWAddress: "00:01:02:03:04:05",
			Hostname:  "foo.example.org",
		},
		SubnetID: 123,
	}, "dhcp4", "dhcp6")
	require.NotNil(t, command)
	require.JSONEq(t, `{
		"command": "reservation-add",
		"service": [ "dhcp4", "dhcp6" ],
		"arguments": {
			"reservation": {
				"hw-address": "00:01:02:03:04:05",
				"hostname": "foo.example.org",
				"subnet-id": 123
			}
		}
	}`, command.Marshal())
}

// Tests reservation-del command.
func TestNewCommandReservationDel(t *testing.T) {
	command := NewCommandReservationDel(&keaconfig.HostCmdsDeletedReservation{
		IdentifierType: "hw-address",
		Identifier:     "00:01:02:03:04:05",
		SubnetID:       123,
	}, "dhcp4", "dhcp6")
	require.NotNil(t, command)
	require.JSONEq(t, `{
		"command": "reservation-del",
		"service": [ "dhcp4", "dhcp6" ],
		"arguments": {
			"identifier-type": "hw-address",
			"identifier": "00:01:02:03:04:05",
			"subnet-id": 123
		}
	}`, command.Marshal())
}

// Tests reservation-get-page command when all arguments are specified.
func TestNewCommandReservationGetPageAllArgs(t *testing.T) {
	command := NewCommandReservationGetPage(234, 1, 5, 100)
	require.NotNil(t, command)
	require.JSONEq(t, `{
		"command": "reservation-get-page",
		"arguments": {
			"subnet-id": 234,
			"source-index": 1,
			"from": 5,
			"limit": 100
		}
	}`, command.Marshal())
}

// Tests reservation-get-page command when mandatory arguments are
// specified and non-mandatory are zero and not included.
func TestNewCommandReservationGetPageAllMandatoryArgs(t *testing.T) {
	command := NewCommandReservationGetPage(234, 0, 0, 100)
	require.NotNil(t, command)
	require.JSONEq(t, `{
		"command": "reservation-get-page",
		"arguments": {
			"subnet-id": 234,
			"limit": 100
		}
	}`, command.Marshal())
}
