package dbmodel

import (
	"math"
	"testing"

	"github.com/go-pg/pg/v10"
	require "github.com/stretchr/testify/require"
	agentapi "isc.org/stork/api"
	keaconfig "isc.org/stork/daemoncfg/kea"
	keadata "isc.org/stork/daemondata/kea"
	"isc.org/stork/datamodel/daemonname"
	dbops "isc.org/stork/server/database"
	dbtest "isc.org/stork/server/database/test"

	storkutil "isc.org/stork/util"
)

// Adds daemons to be used with subnet tests.
func addTestLeaseDaemons(t *testing.T, db *dbops.PgDB) (daemons []*Daemon, subnets []*Subnet) {
	// Add two machines with daemons.
	for i := range 2 {
		m := &Machine{
			ID:        0,
			Address:   "localhost",
			AgentPort: int64(8080 + i),
		}
		err := AddMachine(db, m)
		require.NoError(t, err)

		accessPoints := []*AccessPoint{
			{
				Type:    AccessPointControl,
				Address: "cool.example.org",
				Port:    int64(1234 + i),
				Key:     "",
			},
		}

		// Create DHCPv4 daemon
		daemon4 := NewDaemon(m, daemonname.DHCPv4, true, accessPoints)
		daemon4.KeaDaemon.Config = getTestConfigWithIPv4Subnets(t)
		daemon4.KeaDaemon.Config.DHCPv4Config.LeaseDatabase = &keaconfig.Database{
			Type: "memfile",
		}
		err = AddDaemon(db, daemon4)
		require.NoError(t, err)
		daemons = append(daemons, daemon4)

		// Create one subnet for the DHCPv4 daemon
		subnet4 := &Subnet{
			Prefix: "192.0.2.0/24",
			LocalSubnets: []*LocalSubnet{
				{
					DaemonID: daemon4.ID,
					AddressPools: []AddressPool{
						{
							LowerBound: "192.0.2.1",
							UpperBound: "192.0.2.254",
						},
					},
				},
			},
		}
		err = AddSubnet(db, subnet4)
		require.NoError(t, err)
		subnets = append(subnets, subnet4)

		// Create DHCPv6 daemon
		daemon6 := NewDaemon(m, daemonname.DHCPv6, true, accessPoints)
		daemon6.KeaDaemon.Config = getTestConfigWithIPv6Subnets(t)
		daemon6.KeaDaemon.Config.DHCPv6Config.LeaseDatabase = &keaconfig.Database{
			Type: "memfile",
		}
		err = AddDaemon(db, daemon6)
		require.NoError(t, err)
		daemons = append(daemons, daemon6)

		// Create one subnet for the DHCPv6 daemon
		subnet6 := &Subnet{
			Prefix: "2001:db8:1::/64",
			LocalSubnets: []*LocalSubnet{
				{
					DaemonID: daemon6.ID,
					AddressPools: []AddressPool{
						{
							LowerBound: "2001:db8:1::1",
							UpperBound: "2001:db8:1::ffff",
						},
					},
				},
			},
		}
		err = AddSubnet(db, subnet6)
		require.NoError(t, err)
		subnets = append(subnets, subnet6)
	}
	return daemons, subnets
}

// Verify that AddLease adds a lease to the database with correct values (and
// can come back out of the database unmodified).
func TestAddLease(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()
	daemons, subnets := addTestLeaseDaemons(t, db)

	// Add a lease in subnet 1 (192.0.2.0/24).
	lease := &Lease{
		DaemonID: daemons[0].ID,
		SubnetID: subnets[0].ID,
		Lease: keadata.Lease{
			Family:        4,
			HWAddress:     "00:00:00:00:00:01",
			IPAddress:     "192.0.2.9",
			CLTT:          9999,
			State:         keadata.LeaseStateDefault,
			ValidLifetime: 3600,
			LocalSubnetID: 7,
		},
	}
	err := AddLease(db, lease)
	require.NoError(t, err)
	require.NotZero(t, lease.ID)

	// Get the host from the database.
	returned, err := GetLeaseByID(db, lease.ID)
	require.NoError(t, err)
	require.NotNil(t, returned)

	require.Equal(t, lease.ID, returned.ID)
	require.Equal(t, lease.IPAddress, returned.IPAddress)
	require.Equal(t, lease.HWAddress, returned.HWAddress)
	require.Equal(t, lease.CLTT, returned.CLTT)
	require.Equal(t, lease.State, returned.State)
	require.Equal(t, lease.ValidLifetime, returned.ValidLifetime)
	require.Equal(t, lease.LocalSubnetID, returned.LocalSubnetID)

	// Make sure the linked relations worked too
	require.Equal(t, lease.SubnetID, returned.SubnetID)
	require.Equal(t, lease.DaemonID, returned.DaemonID)
	require.NotNil(t, returned.Daemon)
	require.NotNil(t, returned.Subnet)
}

// Verify that AddLease also works if you give it a transaction struct, rather
// than a db struct.
func TestAddLeaseWorksInTransaction(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()
	daemons, subnets := addTestLeaseDaemons(t, db)

	// Add a lease in subnet 2 (2001:db8:1::0).
	lease := &Lease{
		DaemonID: daemons[1].ID,
		SubnetID: subnets[1].ID,
		Lease: keadata.Lease{
			Family:        6,
			DUID:          "0000000000112233445566ff",
			IPAddress:     "2001:db8:1::67",
			CLTT:          9999,
			State:         keadata.LeaseStateExpiredReclaimed,
			ValidLifetime: 3600,
			LocalSubnetID: 4,
			PrefixLength:  128,
		},
	}
	err := db.RunInTransaction(t.Context(), func(tx *pg.Tx) error {
		return AddLease(tx, lease)
	})
	require.NoError(t, err)
	require.NotZero(t, lease.ID)

	// Get the host from the database.
	returned, err := GetLeaseByID(db, lease.ID)
	require.NoError(t, err)
	require.NotNil(t, returned)

	require.Equal(t, lease.ID, returned.ID)
	require.Equal(t, lease.IPAddress, returned.IPAddress)
	require.Equal(t, lease.HWAddress, returned.HWAddress)
	require.Equal(t, lease.CLTT, returned.CLTT)
	require.Equal(t, lease.State, returned.State)
	require.Equal(t, lease.ValidLifetime, returned.ValidLifetime)
	require.Equal(t, lease.LocalSubnetID, returned.LocalSubnetID)

	// Make sure the linked relations worked too
	require.Equal(t, lease.SubnetID, returned.SubnetID)
	require.Equal(t, lease.DaemonID, returned.DaemonID)
	require.NotNil(t, returned.Daemon)
	require.NotNil(t, returned.Subnet)
}

// Confirm that the return values from GetLeaseByID are as expected when there
// is no lease matching the provided ID.
func TestGetLeaseReturnsNoErrorForNonexistentLease(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()
	_, _ = addTestLeaseDaemons(t, db)

	lease, err := GetLeaseByID(db, 67)
	require.Nil(t, lease)
	require.Nil(t, err)
}

// Confirm that AddLease returns an error if you try to insert a structure which
// links to a nonexistent Daemon.
func TestAddLeaseReturnsErrorWhenForeignKeyConstraintFails(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()
	_, subnets := addTestLeaseDaemons(t, db)

	// Add a lease in subnet 1 (192.0.2.0/24).
	lease := &Lease{
		DaemonID: 9001,
		SubnetID: subnets[0].ID,
		Lease: keadata.Lease{
			Family:        4,
			HWAddress:     "00:00:00:00:00:01",
			IPAddress:     "192.0.2.11",
			CLTT:          9999,
			State:         keadata.LeaseStateDefault,
			ValidLifetime: 3600,
			LocalSubnetID: 7,
		},
	}
	err := AddLease(db, lease)
	require.ErrorContains(t, err, "problem inserting lease")
}

// Verify that the conversion from the gRPC API structure to this one copies all
// the data it needs to, into the correct places.
func TestFromGRPC(t *testing.T) {
	// Arrange
	v4 := agentapi.Lease{
		Family:        agentapi.Lease_V4,
		IpAddress:     "192.168.1.32",
		HwAddress:     "00:00:00:00:00:00",
		Expire:        1000,
		Cltt:          100,
		ValidLifetime: 900,
		SubnetID:      10,
		State:         1,
	}
	v6 := agentapi.Lease{
		Family:        agentapi.Lease_V6,
		IpAddress:     "fd75:9fa5:e76b:0:0:0:0:20",
		Duid:          "00:00:00:00:00:00:00:00",
		Expire:        1002,
		Cltt:          101,
		ValidLifetime: 901,
		SubnetID:      9,
		State:         2,
		PrefixLen:     128,
	}

	expectedv4 := Lease{
		Lease: keadata.Lease{
			Family:        storkutil.IPv4,
			HWAddress:     v4.HwAddress,
			IPAddress:     v4.IpAddress,
			CLTT:          v4.Cltt,
			ValidLifetime: 900,
			LocalSubnetID: v4.SubnetID,
			State:         1,
		},
		DaemonID: 99,
		SubnetID: 1,
	}
	expectedv6 := Lease{
		Lease: keadata.Lease{
			Family:        storkutil.IPv6,
			DUID:          v6.Duid,
			IPAddress:     v6.IpAddress,
			CLTT:          v6.Cltt,
			ValidLifetime: 901,
			LocalSubnetID: v6.SubnetID,
			State:         2,
			PrefixLength:  128,
		},
		DaemonID: 100,
		SubnetID: 2,
	}

	// Act
	actualv4 := NewLeaseFromGRPC(&v4, 99, 1)
	actualv6 := NewLeaseFromGRPC(&v6, 100, 2)

	// Assert
	require.Equal(t, &expectedv4, actualv4)
	require.Equal(t, &expectedv6, actualv6)
}

// Check the various error conditions for LeaseFromGRPC (described within).
func TestFromGRPCWithErrors(t *testing.T) {
	t.Parallel()
	t.Run("nil input", func(t *testing.T) {
		t.Parallel()
		require.Nil(t, NewLeaseFromGRPC(nil, 1, 1))
	})
	t.Run("valid lifetime too big", func(t *testing.T) {
		t.Parallel()
		badLft := agentapi.Lease{
			Family:        agentapi.Lease_V4,
			IpAddress:     "192.168.1.32",
			HwAddress:     "00:00:00:00:00:00",
			Expire:        1000,
			Cltt:          100,
			ValidLifetime: math.MaxUint32 + 2,
			SubnetID:      10,
			State:         1,
		}
		require.Nil(t, NewLeaseFromGRPC(&badLft, 1, 1))
	})
	t.Run("prefix length too long", func(t *testing.T) {
		t.Parallel()
		badPrefixLen := agentapi.Lease{
			Family:        agentapi.Lease_V6,
			IpAddress:     "fd75:9fa5:e76b:0:0:0:0:20",
			Duid:          "00:00:00:00:00:00:00:00",
			Expire:        1002,
			Cltt:          101,
			ValidLifetime: 901,
			SubnetID:      9,
			State:         2,
			PrefixLen:     9001,
		}
		require.Nil(t, NewLeaseFromGRPC(&badPrefixLen, 1, 1))
	})
	t.Run("no IPv5", func(t *testing.T) {
		t.Parallel()
		badIPVersion := agentapi.Lease{
			Family:        5,
			IpAddress:     "fd75:9fa5:e76b:0:0:0:0:20",
			Duid:          "00:00:00:00:00:00:00:00",
			Expire:        1002,
			Cltt:          101,
			ValidLifetime: 901,
			SubnetID:      9,
			State:         2,
			PrefixLen:     128,
		}
		require.Nil(t, NewLeaseFromGRPC(&badIPVersion, 1, 1))
	})
}
