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
// This function creates two machines with two daemons each, one DHCPv4 and one DHCPv6.
// Each daemon has one subnet. The DHCPv4 daemon leases from 192.0.2.0/24. The DHCPv6
// daemon leases from 2001:db8:1::0/64. The local subnet ID for each v4 daemon is 7.
// The local subnet ID for each v6 daemon is 6.
func addTestLeaseDaemons(t *testing.T, db *dbops.PgDB) (daemons []*Daemon, subnets []*Subnet) {
	// Add two machines with two daemons each.
	for i := range 2 {
		m := &Machine{
			ID:        int64(9 + i),
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
					DaemonID:      daemon4.ID,
					LocalSubnetID: 7,
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
					DaemonID:      daemon6.ID,
					LocalSubnetID: 6,
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

// testHelperAddMockLeases is a testing helper function which adds a few example
// leases to the database.  The example leases are in the subnets "192.0.2.0/24" and
// "2001:db8:1::0/64", and this function expects the first subnet provided in
// subnets to also have one of those prefixes.
func testHelperAddMockLeases(t *testing.T, db *dbops.PgDB, daemons []*Daemon, subnets []*Subnet) []*Lease {
	// The tests rely on the indexes of these leases remaining the same. Please only add
	// new leases at the end of the list.
	leases := []*Lease{
		// 0. Valid IPv4 lease.
		{
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
		},
		// 1. Expired IPv4 lease.
		{
			DaemonID: daemons[0].ID,
			SubnetID: subnets[0].ID,
			Lease: keadata.Lease{
				Family:        4,
				HWAddress:     "00:00:00:00:00:02",
				IPAddress:     "192.0.2.11",
				CLTT:          10000,
				State:         keadata.LeaseStateExpiredReclaimed,
				ValidLifetime: 3600,
				LocalSubnetID: 7,
			},
		},
		// 2. Valid IPv6 lease.
		{
			DaemonID: daemons[1].ID,
			SubnetID: subnets[1].ID,
			Lease: keadata.Lease{
				Family:        6,
				DUID:          "01:01:01:01:01:01:01:01",
				IPAddress:     "2001:db8:1::4",
				CLTT:          10001,
				State:         keadata.LeaseStateDefault,
				ValidLifetime: 3600,
				LocalSubnetID: 6,
			},
		},
		// 3. Registered IPv6 lease.
		{
			DaemonID: daemons[1].ID,
			SubnetID: subnets[1].ID,
			Lease: keadata.Lease{
				Family:        6,
				DUID:          "01:01:01:01:01:01:01:02",
				IPAddress:     "2001:db8:1::402",
				CLTT:          10002,
				State:         keadata.LeaseStateRegistered,
				ValidLifetime: 3600,
				LocalSubnetID: 6,
			},
		},
		// 4. Valid IPv6 lease with Client ID.
		{
			DaemonID: daemons[1].ID,
			SubnetID: subnets[1].ID,
			Lease: keadata.Lease{
				Family:        6,
				ClientID:      "01:01:01:01:01:01:01:03",
				IPAddress:     "2001:db8:1::404",
				CLTT:          10002,
				State:         keadata.LeaseStateDefault,
				ValidLifetime: 3600,
				LocalSubnetID: 6,
			},
		},
		// 5. Valid IPv6 lease with hostname.
		{
			DaemonID: daemons[1].ID,
			SubnetID: subnets[1].ID,
			Lease: keadata.Lease{
				Family:        6,
				DUID:          "01:01:01:01:01:01:01:04",
				IPAddress:     "2001:db8:1::408",
				CLTT:          10002,
				Hostname:      "client.example",
				State:         keadata.LeaseStateDefault,
				ValidLifetime: 3600,
				LocalSubnetID: 6,
			},
		},
	}
	for _, lease := range leases {
		err := AddLease(db, lease)
		require.NoError(t, err)
		require.NotZero(t, lease.ID)
	}
	return leases
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

// Confirm that [AddLease] correctly rejects any requests to insert a `nil` lease.
func TestAddLeaseRejectsNilLease(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	err := AddLease(db, nil)
	require.ErrorContains(t, err, "nil lease")
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

// Confirm that [GetLeaseByID] returns an error when there is a database issue *other*
// than "there is no lease with that ID".
func TestGetLeaseByIDDatabaseError(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()
	err := db.Close()
	require.NoError(t, err)

	lease, err := GetLeaseByID(db, 1)
	require.Nil(t, lease)
	require.ErrorContains(t, err, "database is closed")
}

// Verify that [GetLeasesByPage] operates correctly and returns no errors when
// no filters are specified.
func TestGetLeasesByPageNoFilter(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()
	daemons, subnets := addTestLeaseDaemons(t, db)

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

	filters := LeasesByPageFilters{}
	returned, total, err := GetLeasesByPage(db, 0, 10, filters, "", SortDirAny)

	require.NoError(t, err)
	require.EqualValues(t, 1, total)
	require.Len(t, returned, 1)
	require.EqualValues(t, 9999, returned[0].CLTT)
}

// Verify that [GetLeasesByPage] correctly filters the list of leases by subnet.
func TestGetLeasesByPageFilteredBySubnet(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()
	daemons, subnets := addTestLeaseDaemons(t, db)
	leases := testHelperAddMockLeases(t, db, daemons, subnets)

	filters := LeasesByPageFilters{
		SubnetID: &subnets[1].ID,
	}
	returned, total, err := GetLeasesByPage(db, 0, 10, filters, "", SortDirAsc)

	require.NoError(t, err)
	require.EqualValues(t, 4, total)
	require.Len(t, returned, 4)
	require.NotNil(t, returned[0])
	require.Equal(t, leases[2].ID, returned[0].ID)
	require.NotNil(t, returned[1])
	require.Equal(t, leases[3].ID, returned[1].ID)
}

// Verify that [GetLeasesByPage] correctly filters the leases by Kea subnet ID.
func TestGetLeasesByPageFilteredByLocalSubnetID(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()
	daemons, subnets := addTestLeaseDaemons(t, db)
	leases := testHelperAddMockLeases(t, db, daemons, subnets)

	filters := LeasesByPageFilters{
		LocalSubnetID: &subnets[1].LocalSubnets[0].LocalSubnetID,
	}
	returned, total, err := GetLeasesByPage(db, 0, 10, filters, "", SortDirAsc)

	require.NoError(t, err)
	require.EqualValues(t, 4, total)
	require.Len(t, returned, 4)
	require.NotNil(t, returned[0])
	require.Equal(t, leases[2].ID, returned[0].ID)
	require.NotNil(t, returned[1])
	require.Equal(t, leases[3].ID, returned[1].ID)
}

// Verify that [GetLeasesByPage] correctly filters the list of leases by daemon.
func TestGetLeasesByPageFilteredByDaemon(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()
	daemons, subnets := addTestLeaseDaemons(t, db)
	leases := testHelperAddMockLeases(t, db, daemons, subnets)

	filters := LeasesByPageFilters{
		DaemonID: &daemons[1].ID,
	}
	returned, total, err := GetLeasesByPage(db, 0, 10, filters, "", SortDirAsc)

	require.NoError(t, err)
	require.EqualValues(t, 4, total)
	require.Len(t, returned, 4)
	require.NotNil(t, returned[0])
	require.Equal(t, leases[2].ID, returned[0].ID)
	require.NotNil(t, returned[1])
	require.Equal(t, leases[3].ID, returned[1].ID)
}

// Verify that [GetLeasesByPage] correctly filters the list of leases by machine.
func TestGetLeasesByPageFilteredByMachine(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()
	daemons, subnets := addTestLeaseDaemons(t, db)
	leases := testHelperAddMockLeases(t, db, daemons, subnets)

	filters := LeasesByPageFilters{
		MachineID: &daemons[0].Machine.ID,
	}
	returned, total, err := GetLeasesByPage(db, 0, 10, filters, "", SortDirAsc)

	require.NoError(t, err)
	require.EqualValues(t, 6, total)
	require.Len(t, returned, 6)
	require.NotNil(t, returned[0])
	require.Equal(t, leases[0].ID, returned[0].ID)
	require.NotNil(t, returned[1])
	require.Equal(t, leases[1].ID, returned[1].ID)
	require.NotNil(t, returned[1])
	require.Equal(t, leases[2].ID, returned[2].ID)
	require.NotNil(t, returned[1])
	require.Equal(t, leases[3].ID, returned[3].ID)
}

// Verify that [GetLeasesByPage] correctly filters the list of leases by text.
func TestGetLeasesByPageFilteredByText(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()
	daemons, subnets := addTestLeaseDaemons(t, db)
	leases := testHelperAddMockLeases(t, db, daemons, subnets)

	testCases := []struct {
		description string
		filterText  string
		expectedID  int64
	}{
		{
			description: "filter by IP address",
			filterText:  "192.0.2.9",
			expectedID:  leases[0].ID,
		},
		// TODO: this one fails
		{
			description: "filter by DUID",
			filterText:  "01:01:01:01:01:01:01:01",
			expectedID:  leases[2].ID,
		},
		{
			description: "filter by MAC address",
			filterText:  "00:00:00:00:00:01",
			expectedID:  leases[0].ID,
		},
		// TODO: this one fails
		{
			description: "filter by Client ID",
			filterText:  "01:01:01:01:01:01:01:03",
			expectedID:  leases[4].ID,
		},
		{
			description: "filter by hostname",
			filterText:  "client.example",
			expectedID:  leases[5].ID,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			filters := LeasesByPageFilters{
				FilterText: &tc.filterText,
			}
			returned, total, err := GetLeasesByPage(db, 0, 10, filters, "", SortDirAsc)

			require.NoError(t, err)
			require.EqualValues(t, 1, total)
			require.Len(t, returned, 1)
			require.NotNil(t, returned[0])
			require.Equal(t, tc.expectedID, returned[0].ID)
		})
	}
}

// Verify that [GetLeasesByPage] returns ([], 0, nil) when there are no rows.
func TestGetLeasesByPageReturns0WhenNoRows(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	filters := LeasesByPageFilters{}

	returned, total, err := GetLeasesByPage(db, 0, 10, filters, "", SortDirAsc)

	require.NoError(t, err)
	require.EqualValues(t, 0, total)
	require.NotNil(t, returned)
	require.Len(t, returned, 0)
}

// Verify that [GetLeasesByPage] propagates an error when the DB has some issue other
// than [pg.ErrNoRows].
func TestGetLeasesByPageWhenDBHasError(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	db.Close()
	filters := LeasesByPageFilters{}
	returned, total, err := GetLeasesByPage(db, 0, 10, filters, "", SortDirAsc)

	require.ErrorContains(t, err, "database is closed")
	require.EqualValues(t, 0, total)
	require.Nil(t, returned)
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
