package restservice

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"
	keadata "isc.org/stork/daemondata/kea"
	"isc.org/stork/datamodel/daemonname"
	dbmodel "isc.org/stork/server/database/model"
)

func TestConvertLeaseFromRestAPIWithNilLease(t *testing.T) {
	result, err := convertLeaseFromRestAPI(nil)

	require.Nil(t, result)
	require.ErrorContains(t, err, "nil")
}

func TestConvertLeaseFromRestAPIWithCLTTTooBig(t *testing.T) {
	lease := dbmodel.Lease{
		Lease: keadata.Lease{
			CLTT: math.MaxInt64 + 1,
		},
	}
	result, err := convertLeaseFromRestAPI(&lease)

	require.Nil(t, result)
	require.ErrorContains(t, err, "CLTT")
}

func TestConvertLeaseFromRestAPIWithNilDaemon(t *testing.T) {
	lease := dbmodel.Lease{
		Lease: keadata.Lease{
			CLTT: math.MaxInt64 - 1,
		},
	}
	result, err := convertLeaseFromRestAPI(&lease)

	require.Nil(t, result)
	require.ErrorContains(t, err, "Daemon")
}

func TestConvertLeaseFromRestAPIWithNilSubnet(t *testing.T) {
	lease := dbmodel.Lease{
		Daemon: &dbmodel.Daemon{
			ID:   1,
			Name: daemonname.DHCPv4,
		},
		Lease: keadata.Lease{
			CLTT: math.MaxInt64 - 1,
		},
	}
	result, err := convertLeaseFromRestAPI(&lease)

	require.Nil(t, result)
	require.ErrorContains(t, err, "Subnet")
}

func TestConvertLeaseFromRestAPIWithValidLease(t *testing.T) {
	lease := dbmodel.Lease{
		DaemonID: 1,
		SubnetID: 9,
		Daemon: &dbmodel.Daemon{
			ID:   1,
			Name: daemonname.DHCPv6,
		},
		Lease: keadata.Lease{
			State:         1,
			CLTT:          1776459817,
			IPAddress:     "fe80::9",
			PrefixLength:  128,
			DUID:          "00:00:00:00:00:00:00:00:01:01",
			ValidLifetime: 3600,
		},
		Subnet: &dbmodel.Subnet{
			ID:     9,
			Prefix: "fe80::/64",
		},
	}
	result, err := convertLeaseFromRestAPI(&lease)

	require.Nil(t, err)
	require.NotNil(t, result)
	require.EqualValues(t, lease.CLTT, *result.Cltt)
}

func TestGetLeaseList(t *testing.T) {
	require.Nil(t, nil)
}
