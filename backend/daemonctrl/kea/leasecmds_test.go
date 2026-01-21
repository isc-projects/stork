package keactrl

import (
	"testing"

	require "github.com/stretchr/testify/require"
)

// Tests lease4-get command.
func TestNewCommandLease4Get(t *testing.T) {
	command := NewCommandLease4Get("192.0.2.1")
	require.NotNil(t, command)
	require.Len(t, command.Daemons, 1)
	bytes, err := command.Marshal()
	require.NoError(t, err)
	require.JSONEq(t, `{
		"command": "lease4-get",
		"service": ["dhcp4"],
		"arguments": {
			"ip-address": "192.0.2.1"
		}

	}`, string(bytes))
}

// Tests lease6-get command.
func TestNewCommandLease6Get(t *testing.T) {
	command := NewCommandLease6Get(LeaseTypeNA, "2001:db8:1::1")
	require.NotNil(t, command)
	require.Len(t, command.Daemons, 1)
	bytes, err := command.Marshal()
	require.NoError(t, err)
	require.JSONEq(t, `{
		"command": "lease6-get",
		"service": ["dhcp6"],
		"arguments": {
			"type": "IA_NA",
			"ip-address": "2001:db8:1::1"
		}

	}`, string(bytes))
}

// Tests lease4-get-by-hw-address command.
func TestNewCommandLease4GetByHWAddress(t *testing.T) {
	command := NewCommandLease4GetByHWAddress("aa:bb:cc:dd:ee:ff")
	require.NotNil(t, command)
	require.Len(t, command.Daemons, 1)
	bytes, err := command.Marshal()
	require.NoError(t, err)
	require.JSONEq(t, `{
		"command": "lease4-get-by-hw-address",
		"service": ["dhcp4"],
		"arguments": {
			"hw-address": "aa:bb:cc:dd:ee:ff"
		}
	}`, string(bytes))
}

// Tests lease4-get-by-client-id command.
func TestNewCommandLease4GetByClientID(t *testing.T) {
	command := NewCommandLease4GetByClientID("client123")
	require.NotNil(t, command)
	require.Len(t, command.Daemons, 1)
	bytes, err := command.Marshal()
	require.NoError(t, err)
	require.JSONEq(t, `{
		"command": "lease4-get-by-client-id",
		"service": ["dhcp4"],
		"arguments": {
			"client-id": "client123"
		}
	}`, string(bytes))
}

// Tests lease6-get-by-duid command.
func TestNewCommandLease6GetByDUID(t *testing.T) {
	command := NewCommandLease6GetByDUID("00:01:00:01:12:34:56:78:aa:bb:cc:dd:ee:ff")
	require.NotNil(t, command)
	require.Len(t, command.Daemons, 1)
	bytes, err := command.Marshal()
	require.NoError(t, err)
	require.JSONEq(t, `{
		"command": "lease6-get-by-duid",
		"service": ["dhcp6"],
		"arguments": {
			"duid": "00:01:00:01:12:34:56:78:aa:bb:cc:dd:ee:ff"
		}
	}`, string(bytes))
}

// Tests lease4-get-by-hostname command.
func TestNewCommandLease4GetByHostname(t *testing.T) {
	command := NewCommandLease4GetByHostname("example.com")
	require.NotNil(t, command)
	require.Len(t, command.Daemons, 1)
	bytes, err := command.Marshal()
	require.NoError(t, err)
	require.JSONEq(t, `{
		"command": "lease4-get-by-hostname",
		"service": ["dhcp4"],
		"arguments": {
			"hostname": "example.com"
		}
	}`, string(bytes))
}

// Tests lease6-get-by-hostname command.
func TestNewCommandLease6GetByHostname(t *testing.T) {
	command := NewCommandLease6GetByHostname("ipv6.example.com")
	require.NotNil(t, command)
	require.Len(t, command.Daemons, 1)
	bytes, err := command.Marshal()
	require.NoError(t, err)
	require.JSONEq(t, `{
		"command": "lease6-get-by-hostname",
		"service": ["dhcp6"],
		"arguments": {
			"hostname": "ipv6.example.com"
		}
	}`, string(bytes))
}

// Tests lease4-get-by-hostname command.
func TestNewCommandLease4GetByState(t *testing.T) {
	command := NewCommandLease4GetByState(LeaseStateDeclined)
	require.NotNil(t, command)
	require.Len(t, command.Daemons, 1)
	bytes, err := command.Marshal()
	require.NoError(t, err)
	require.JSONEq(t, `{
		"command": "lease4-get-by-state",
		"service": ["dhcp4"],
		"arguments": {
			"state": 1
		}
	}`, string(bytes))
}

// Tests lease6-get-by-hostname command.
func TestNewCommandLease6GetByState(t *testing.T) {
	command := NewCommandLease6GetByState(LeaseStateDeclined)
	require.NotNil(t, command)
	require.Len(t, command.Daemons, 1)
	bytes, err := command.Marshal()
	require.NoError(t, err)
	require.JSONEq(t, `{
		"command": "lease6-get-by-state",
		"service": ["dhcp6"],
		"arguments": {
			"state": 1
		}
	}`, string(bytes))
}

// Tests ParseLeaseState to ensure it handles all valid lease state names.
func TestParseLeaseState(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input                 string
		expectedOutput        LeaseState
		expectedErrorContains string
	}{
		{
			LeaseStateAssignedStr,
			LeaseStateAssigned,
			"",
		},
		{
			LeaseStateDeclinedStr,
			LeaseStateDeclined,
			"",
		},
		{
			LeaseStateExpiredReclaimedStr,
			LeaseStateExpiredReclaimed,
			"",
		},
		{
			LeaseStateReleasedStr,
			LeaseStateReleased,
			"",
		},
		{
			LeaseStateRegisteredStr,
			LeaseStateRegistered,
			"",
		},
		{
			"foo",
			LeaseStateAssigned,
			"foo",
		},
		// spelling error
		{
			"declnied",
			LeaseStateAssigned,
			"declnied",
		},
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			t.Parallel()
			result, err := ParseLeaseState(test.input)
			require.Equal(t, test.expectedOutput, result)
			if test.expectedErrorContains != "" {
				require.ErrorContains(t, err, test.expectedErrorContains)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
