package dbmodel

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	keaconfig "isc.org/stork/appcfg/kea"
	dbops "isc.org/stork/server/database"
	dbtest "isc.org/stork/server/database/test"
)

// Test that subnet with address pools is inserted into the database.
func TestAddSubnetWithAddressPools(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	subnet := &Subnet{
		Prefix: "192.0.2.0/24",
		AddressPools: []AddressPool{
			{
				LowerBound: "192.0.2.1",
				UpperBound: "192.0.2.10",
			},
			{
				LowerBound: "192.0.2.11",
				UpperBound: "192.0.2.20",
			},
			{
				LowerBound: "192.0.2.21",
				UpperBound: "192.0.2.30",
			},
		},
	}
	err := AddSubnet(db, subnet)
	require.NoError(t, err)
	require.NotZero(t, subnet.ID)

	// Get the subnet from the database.
	returned, err := GetSubnet(db, subnet.ID)
	require.NoError(t, err)
	require.NotNil(t, returned)

	// Make sure that the pools are returned too.
	require.Equal(t, subnet.Prefix, returned.Prefix)
	require.Len(t, subnet.AddressPools, 3)
	require.Empty(t, subnet.PrefixPools)
}

// Test that subnet with address and prefix pools is inserted into the database.
func TestAddSubnetWithPrefixPools(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	subnet := &Subnet{
		Prefix: "2001:db8:1::/64",
		AddressPools: []AddressPool{
			{
				LowerBound: "2001:db8:1::1",
				UpperBound: "2001:db8:1::10",
			},
			{
				LowerBound: "2001:db8:1::11",
				UpperBound: "2001:db8:1::20",
			},
			{
				LowerBound: "2001:db8:1::21",
				UpperBound: "2001:db8:1::30",
			},
		},
		PrefixPools: []PrefixPool{
			{
				Prefix:       "3001::/64",
				DelegatedLen: 80,
			},
			{
				Prefix:       "3000::/32",
				DelegatedLen: 120,
			},
			{
				Prefix:       "2001:db8:2::/64",
				DelegatedLen: 96,
			},
		},
	}
	err := AddSubnet(db, subnet)
	require.NoError(t, err)
	require.NotZero(t, subnet.ID)

	// Get the subnet from the database/
	returned, err := GetSubnet(db, subnet.ID)
	require.NoError(t, err)
	require.NotNil(t, returned)
	require.NotZero(t, returned.CreatedAt)
	require.Equal(t, subnet.Prefix, returned.Prefix)
	require.Zero(t, returned.SharedNetworkID)
	require.Nil(t, returned.SharedNetwork)

	// Make sure that the address pools and prefix pools are returned.
	require.Len(t, returned.AddressPools, 3)
	require.Len(t, returned.PrefixPools, 3)

	// Validate returned address pools.
	for i, p := range returned.AddressPools {
		require.NotZero(t, p.CreatedAt)
		require.Equal(t, subnet.AddressPools[i].LowerBound, p.LowerBound)
		require.Equal(t, subnet.AddressPools[i].UpperBound, p.UpperBound)
	}

	// Validate returned prefix pools.
	for i, p := range returned.PrefixPools {
		require.NotZero(t, p.CreatedAt)
		require.Equal(t, subnet.PrefixPools[i].Prefix, p.Prefix)
		require.Equal(t, subnet.PrefixPools[i].DelegatedLen, p.DelegatedLen)
	}
}

// Test that all subnets, all IPv4 subnets or all IPv6 subnets can be fetched.
func TestGetAllSubnets(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Add two IPv4 and two IPv6 subnets.
	subnets := []Subnet{
		{
			Prefix: "192.0.2.0/24",
		},
		{
			Prefix: "2001:db8:1::/64",
		},
		{
			Prefix: "192.0.3.0/24",
		},
		{
			Prefix: "2001:db8:2::/64",
		},
	}
	for _, s := range subnets {
		subnet := s
		err := AddSubnet(db, &subnet)
		require.NoError(t, err)
	}

	// Get all subnets regardless of the family.
	returnedSubnets, err := GetAllSubnets(db, 0)
	require.NoError(t, err)
	require.Len(t, returnedSubnets, 4)

	// They should be ordered the same as they were inserted.
	for i, s := range returnedSubnets {
		require.Equal(t, subnets[i].Prefix, s.Prefix)
	}

	// Get IPv4 subnets only. The order is preserved.
	returnedSubnets, err = GetAllSubnets(db, 4)
	require.NoError(t, err)
	require.Len(t, returnedSubnets, 2)
	require.Equal(t, subnets[0].Prefix, returnedSubnets[0].Prefix)
	require.Equal(t, subnets[2].Prefix, returnedSubnets[1].Prefix)

	// Get IPv6 subnets only. The order is preserved.
	returnedSubnets, err = GetAllSubnets(db, 6)
	require.NoError(t, err)
	require.Len(t, returnedSubnets, 2)
	require.Equal(t, subnets[1].Prefix, returnedSubnets[0].Prefix)
	require.Equal(t, subnets[3].Prefix, returnedSubnets[1].Prefix)
}

// Test that top-level subnets are fetched.
func TestTopLevelSubnets(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Add a shared network with two subnets.
	sharedNetwork := SharedNetwork{
		Name:   "foo",
		Family: 4,
	}
	sharedNetwork.Subnets = []Subnet{
		{
			Prefix: "192.0.2.0/24",
		},
		{
			Prefix: "192.0.3.0/24",
		},
	}
	err := AddSharedNetwork(db, &sharedNetwork)
	require.NoError(t, err)

	// Add two top-level subnets.
	subnets := []Subnet{
		{
			Prefix: "192.0.4.0/24",
		},
		{
			Prefix: "192.0.5.0/24",
		},
	}
	for _, s := range subnets {
		subnet := s
		err := AddSubnet(db, &subnet)
		require.NoError(t, err)
	}

	// Get top-level subnets only. It should return two subnets.
	subnets, err = GetTopLevelSubnets(db, 4)
	require.NoError(t, err)
	require.Len(t, subnets, 2)

	// Ensure that both subnets were returned.
	subnetMap := make(map[string]Subnet)
	for i := range subnets {
		subnetMap[subnets[i].Prefix] = subnets[i]
	}
	require.Len(t, subnetMap, 2)
	require.Contains(t, subnetMap, "192.0.4.0/24")
	require.Contains(t, subnetMap, "192.0.5.0/24")

	subnets, err = GetTopLevelSubnets(db, 6)
	require.NoError(t, err)
	require.Empty(t, subnets)
}

// Test that the inserted subnet can be associated with a shared network.
func TestAddSubnetWithExistingSharedNetwork(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// First, add the shared network.
	sharedNetwork := &SharedNetwork{
		Name:   "test",
		Family: 6,
	}
	err := AddSharedNetwork(db, sharedNetwork)
	require.NoError(t, err)
	require.NotZero(t, sharedNetwork.ID)

	// Add a subnet associated with this shared network.
	subnet := &Subnet{
		Prefix:          "2001:db8:1::/64",
		SharedNetworkID: sharedNetwork.ID,
	}
	err = AddSubnet(db, subnet)
	require.NoError(t, err)
	require.NotZero(t, subnet.ID)

	// Get the subnet from the database and make sure that the shared
	// network is also returned.
	returnedSubnet, err := GetSubnet(db, subnet.ID)
	require.NoError(t, err)
	require.NotNil(t, returnedSubnet)
	require.NotNil(t, returnedSubnet.SharedNetwork)
	require.Equal(t, returnedSubnet.SharedNetwork.ID, returnedSubnet.SharedNetworkID)
}

// Test that an app can be associated with the existing subnet and then
// such association can be removed.
func TestAddDeleteAppToSubnet(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Add apps to the database. They must exist to make any association between
	// them and the subnet.
	apps := addTestSubnetApps(t, db)
	require.Len(t, apps, 2)

	// Same story for subnet. It must exist.
	subnet := &Subnet{
		Prefix: "192.0.2.0/24",
	}
	err := AddSubnet(db, subnet)
	require.NoError(t, err)
	require.NotZero(t, subnet.ID)

	// Add association of the app to the subnet.
	err = AddAppToSubnet(db, subnet, apps[0], apps[0].Daemons[0])
	require.NoError(t, err)
	require.NotZero(t, subnet.ID)

	// Make sure that this association is returned when fetching the subnet.
	returnedSubnet, err := GetSubnet(db, subnet.ID)
	require.NoError(t, err)
	require.NotNil(t, returnedSubnet)
	require.Len(t, returnedSubnet.LocalSubnets, 1)
	require.EqualValues(t, 123, returnedSubnet.LocalSubnets[0].LocalSubnetID)
	require.NotNil(t, returnedSubnet.LocalSubnets[0].App)

	// Also make sure that the app can be retrieved using the GetApp function.
	returnedApp := returnedSubnet.GetApp(apps[0].ID)
	require.NotNil(t, returnedApp)
	returnedApp = returnedSubnet.GetApp(apps[1].ID)
	require.Nil(t, returnedApp)

	// Add another app to the same subnet.
	err = AddAppToSubnet(db, subnet, apps[1], apps[1].Daemons[0])
	require.NoError(t, err)
	require.NotZero(t, subnet.ID)

	// Get the subnet again and expect two apps to be returned.
	returnedSubnet, err = GetSubnet(db, subnet.ID)
	require.NoError(t, err)
	require.NotNil(t, returnedSubnet)
	require.Len(t, returnedSubnet.LocalSubnets, 2)
	require.EqualValues(t, 123, returnedSubnet.LocalSubnets[0].LocalSubnetID)
	require.NotNil(t, returnedSubnet.LocalSubnets[0].App)
	require.EqualValues(t, 123, returnedSubnet.LocalSubnets[1].LocalSubnetID)
	require.NotNil(t, returnedSubnet.LocalSubnets[1].App)

	// Remove the association of the first app with the subnet.
	ok, err := DeleteAppFromSubnet(db, subnet.ID, returnedSubnet.LocalSubnets[0].App.ID)
	require.NoError(t, err)
	require.True(t, ok)

	// Check again that only one app is returned.
	returnedSubnet, err = GetSubnet(db, subnet.ID)
	require.NoError(t, err)
	require.NotNil(t, returnedSubnet)
	require.Len(t, returnedSubnet.LocalSubnets, 1)
}

// Test that app's associations with multiple subnets can be removed.
func TestDeleteAppFromSubnets(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Add apps to the database. They must exist to make any association between
	// them and the subnet.
	apps := addTestSubnetApps(t, db)
	require.Len(t, apps, 2)

	subnets := []Subnet{
		{
			Prefix: "192.0.2.0/24",
		},
		{
			Prefix: "192.0.3.0/24",
		},
		{
			Prefix: "192.0.4.0/24",
		},
	}
	for i := range subnets {
		err := AddSubnet(db, &subnets[i])
		require.NoError(t, err)
		require.NotZero(t, subnets[i].ID)
	}

	// Associate the first app with two subnets.
	err := AddAppToSubnet(db, &subnets[0], apps[0], apps[0].Daemons[0])
	require.NoError(t, err)

	err = AddAppToSubnet(db, &subnets[1], apps[0], apps[0].Daemons[0])
	require.NoError(t, err)

	// Associate the second app with another subnet.
	err = AddAppToSubnet(db, &subnets[2], apps[1], apps[1].Daemons[0])
	require.NoError(t, err)

	// Remove associations of the first app.
	count, err := DeleteDaemonFromSubnets(db, apps[0].Daemons[0].ID)
	require.NoError(t, err)
	require.EqualValues(t, 2, count)

	// Ensure that the associations were removed for the first app.
	returned, err := GetSubnetsByAppID(db, apps[0].ID, 4)
	require.NoError(t, err)
	require.Empty(t, returned)

	// The association should still exist for the second app.
	returned, err = GetSubnetsByAppID(db, apps[1].ID, 4)
	require.NoError(t, err)
	require.Len(t, returned, 1)
}

// Test that the subnet can be fetched by local ID and app ID.
func TestGetSubnetByLocalID(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Add apps to the database. They must exist to make any association between
	// then and the subnet.
	apps := addTestSubnetApps(t, db)
	require.Len(t, apps, 2)

	subnet := &Subnet{
		Prefix: "192.0.2.0/24",
	}
	err := AddSubnet(db, subnet)
	require.NoError(t, err)
	require.NotZero(t, subnet.ID)

	// Add association of the app to the subnet.
	err = AddAppToSubnet(db, subnet, apps[0], apps[0].Daemons[0])
	require.NoError(t, err)
	require.NotZero(t, subnet.ID)

	// The subnet should be returned for local subnet ID of 123 and for
	// the app we have added.
	returnedSubnets, err := GetSubnetsByLocalID(db, 123, apps[0].ID, 0)
	require.NoError(t, err)
	require.Len(t, returnedSubnets, 1)

	// It should be also returned among IPv4 subnets.
	returnedSubnets, err = GetSubnetsByLocalID(db, 123, apps[0].ID, 4)
	require.NoError(t, err)
	require.Len(t, returnedSubnets, 1)

	// If the local subnet ID is not matching no subnet should be returned.
	returnedSubnets, err = GetSubnetsByLocalID(db, 234, apps[0].ID, 0)
	require.NoError(t, err)
	require.Empty(t, returnedSubnets)

	// If the app id is not matching the subnet should not be returned.
	returnedSubnets, err = GetSubnetsByLocalID(db, 123, apps[0].ID+1, 0)
	require.NoError(t, err)
	require.Empty(t, returnedSubnets)

	// The IPv6 subnet does not exist.
	returnedSubnets, err = GetSubnetsByLocalID(db, 123, apps[0].ID, 6)
	require.NoError(t, err)
	require.Empty(t, returnedSubnets)
}

// Test that subnets can be fetched by app ID.
func TestGetSubnetsByAppID(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Add apps to the database. They must exist to make any association between
	// then and the subnet.
	apps := addTestSubnetApps(t, db)
	require.Len(t, apps, 2)

	// Add several subnets matching configuration of the apps we have added.
	subnets := []Subnet{
		{
			Prefix: "192.0.2.0/24",
		},
		{
			Prefix: "192.0.3.0/24",
		},
		{
			Prefix: "10.0.0.0/8",
		},
	}
	for i := range subnets {
		err := AddSubnet(db, &subnets[i])
		require.NoError(t, err)
		require.NotZero(t, subnets[i].ID)

		// Add association of the apps to the subnet.
		if i < 2 {
			// First two subnets associated with the first app.
			err = AddAppToSubnet(db, &subnets[i], apps[0], apps[0].Daemons[0])
		} else {
			// Last subnet is only associated with the second app.
			err = AddAppToSubnet(db, &subnets[i], apps[1], apps[1].Daemons[0])
		}
		require.NoError(t, err)
		require.NotZero(t, subnets[i].ID)
	}

	// Get all IPv4 subnets for this app.
	returnedSubnets, err := GetSubnetsByAppID(db, apps[0].ID, 4)
	require.NoError(t, err)
	require.Len(t, returnedSubnets, 2)

	require.Len(t, returnedSubnets[0].LocalSubnets, 1)
	require.EqualValues(t, 123, returnedSubnets[0].LocalSubnets[0].LocalSubnetID)
	require.EqualValues(t, apps[0].ID, returnedSubnets[0].LocalSubnets[0].AppID)

	require.Len(t, returnedSubnets[1].LocalSubnets, 1)
	require.EqualValues(t, 234, returnedSubnets[1].LocalSubnets[0].LocalSubnetID)
	require.EqualValues(t, apps[0].ID, returnedSubnets[1].LocalSubnets[0].AppID)

	// Get all IPv4 subnets for the second app.
	returnedSubnets, err = GetSubnetsByAppID(db, apps[1].ID, 4)
	require.NoError(t, err)
	require.Len(t, returnedSubnets, 1)

	require.Len(t, returnedSubnets[0].LocalSubnets, 1)
	require.EqualValues(t, 345, returnedSubnets[0].LocalSubnets[0].LocalSubnetID)
	require.EqualValues(t, apps[1].ID, returnedSubnets[0].LocalSubnets[0].AppID)

	// Get all subnets for the first app.
	returnedSubnets, err = GetSubnetsByAppID(db, apps[0].ID, 0)
	require.NoError(t, err)
	require.Len(t, returnedSubnets, 2)

	// Get IPv6 subnets. They should not exist.
	returnedSubnets, err = GetSubnetsByAppID(db, apps[0].ID, 6)
	require.NoError(t, err)
	require.Empty(t, returnedSubnets)
}

// This test verifies that subnets can be filtered by search text.
// In particular, it verifies that matching with address pools works
// as expected and that duplicates are eliminated from the result
// set when the text matches multiple pools in the same subnet.
func TestGetSubnetsByPage(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Add two subnets with multiple address pools.
	subnets := []Subnet{
		{
			Prefix: "192.0.2.0/24",
			AddressPools: []AddressPool{
				{
					LowerBound: "192.0.2.1",
					UpperBound: "192.0.2.10",
				},
				{
					LowerBound: "192.0.2.11",
					UpperBound: "192.0.2.20",
				},
				{
					LowerBound: "192.0.2.21",
					UpperBound: "192.0.2.30",
				},
			},
		},
		{
			Prefix: "192.0.3.0/24",
			AddressPools: []AddressPool{
				{
					LowerBound: "192.0.3.1",
					UpperBound: "192.0.3.10",
				},
				{
					LowerBound: "192.0.3.11",
					UpperBound: "192.0.3.20",
				},
			},
		},
	}
	for i := range subnets {
		err := AddSubnet(db, &subnets[i])
		require.NoError(t, err)
		require.NotZero(t, subnets[i].ID)
	}

	// This should match two subnets.
	filterText := "192.0"
	returned, count, err := GetSubnetsByPage(db, 0, 10, 0, 4, &filterText, "prefix", SortDirDesc)
	require.NoError(t, err)
	require.EqualValues(t, 2, count)
	require.Len(t, returned, 2)
	// The subnets are returned in descending prefix order.
	require.Equal(t, "192.0.3.0/24", returned[0].Prefix)
	require.Equal(t, "192.0.2.0/24", returned[1].Prefix)

	// This should match multiple pools in the first subnet. However,
	// only one record should be returned.
	filterText = "192.0.2.1"
	returned, count, err = GetSubnetsByPage(db, 0, 10, 0, 4, &filterText, "prefix", SortDirDesc)
	require.NoError(t, err)
	require.EqualValues(t, 1, count)
	require.Len(t, returned, 1)
	require.Equal(t, "192.0.2.0/24", returned[0].Prefix)

	// This should have no match.
	filterText = "192.0.5.0"
	returned, count, err = GetSubnetsByPage(db, 0, 10, 0, 4, &filterText, "id", SortDirAsc)
	require.NoError(t, err)
	require.Zero(t, count)
	require.Empty(t, returned)
}

// Test that the subnet can be fetched by local ID and app ID.
func TestGetAppLocalSubnets(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// prepare apps
	apps := addTestSubnetApps(t, db)
	require.Len(t, apps, 2)

	// prepare a subnet
	subnet := &Subnet{
		Prefix: "192.0.2.0/24",
	}
	err := AddSubnet(db, subnet)
	require.NoError(t, err)
	require.NotZero(t, subnet.ID)

	// add association of the app to the subnet - this will create LocalSubnet
	err = AddAppToSubnet(db, subnet, apps[0], apps[0].Daemons[0])
	require.NoError(t, err)
	require.NotZero(t, subnet.ID)

	// check fetching LocalSubnets for given app
	subnets, err := GetAppLocalSubnets(db, apps[0].ID)
	require.NoError(t, err)
	require.Len(t, subnets, 1)
	require.EqualValues(t, 123, subnets[0].LocalSubnetID)
	require.NotNil(t, subnets[0].Subnet)
	require.Equal(t, subnet.ID, subnets[0].Subnet.ID)
}

// Check updating stats in LocalSubnet.
func TestUpdateStats(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// prepare apps
	apps := addTestSubnetApps(t, db)
	require.Len(t, apps, 2)

	// prepare a subnet
	subnet := &Subnet{
		Prefix: "192.0.2.0/24",
	}
	err := AddSubnet(db, subnet)
	require.NoError(t, err)
	require.NotZero(t, subnet.ID)

	// add association of the app to the subnet - this will create LocalSubnet
	err = AddAppToSubnet(db, subnet, apps[0], apps[0].Daemons[0])
	require.NoError(t, err)
	require.NotZero(t, subnet.ID)

	// check fetching LocalSubnets for given app
	subnets, err := GetAppLocalSubnets(db, apps[0].ID)
	require.NoError(t, err)
	require.Len(t, subnets, 1)

	// check updating stats
	lsn := subnets[0]
	lsn.AppID = apps[0].ID
	lsn.SubnetID = subnet.ID
	stats := make(map[string]interface{})
	stats["hakuna-matata"] = 123
	err = lsn.UpdateStats(db, stats)
	require.NoError(t, err)

	// check stored stats
	lsns := []*LocalSubnet{}
	err = db.Model(&lsns).Select()
	require.NoError(t, err)
	require.Len(t, lsns, 1)
	lsn = lsns[0]
	require.NotZero(t, lsn.StatsCollectedAt)
	require.NotEmpty(t, lsn.Stats)
	require.Contains(t, lsn.Stats, "hakuna-matata")
	require.EqualValues(t, 123, lsn.Stats["hakuna-matata"])
}

// Test that global shared networks and subnet instances are committed
// to the database and associated with the given app. This test is very
// simple. More exhaustive tests are implemented in backend/apps.
func TestCommitNetworksIntoDB(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Add the machine.
	m := &Machine{
		ID:        0,
		Address:   "localhost",
		AgentPort: 8080,
	}
	err := AddMachine(db, m)
	require.NoError(t, err)

	// Creates new app. Its configuration doesn't matter in this test.
	var accessPoints []*AccessPoint
	accessPoints = AppendAccessPoint(accessPoints, AccessPointControl, "localhost", "", 8000, false)
	app := App{
		MachineID:    m.ID,
		Type:         AppTypeKea,
		AccessPoints: accessPoints,
		Daemons: []*Daemon{
			{
				Name:   DaemonNameDHCPv4,
				Active: true,
			},
		},
	}
	// Add the app to the database.
	_, err = AddApp(db, &app)
	require.NoError(t, err)

	// Create a shared network and subnet.
	networks := []SharedNetwork{
		{
			Name:   "foo",
			Family: 4,
			Subnets: []Subnet{
				{
					Prefix: "192.0.2.0/24",
					Hosts: []Host{
						{
							HostIdentifiers: []HostIdentifier{
								{
									Type:  "hw-address",
									Value: []byte{1, 2, 3, 4, 5, 6},
								},
							},
							IPReservations: []IPReservation{
								{
									Address: "192.0.2.123/32",
								},
							},
						},
					},
				},
			},
		},
	}
	subnets := []Subnet{
		{
			Prefix: "192.0.3.0/24",
			Hosts: []Host{
				{
					HostIdentifiers: []HostIdentifier{
						{
							Type:  "hw-address",
							Value: []byte{1, 2, 3, 4, 5, 6},
						},
					},
					IPReservations: []IPReservation{
						{
							Address: "192.0.3.123/32",
						},
					},
				},
			},
		},
	}
	// Attempt to create the global shared network and subnet.
	addedSubnets, err := CommitNetworksIntoDB(db, networks, subnets, &app, app.Daemons[0], 1)
	require.NoError(t, err)
	require.Len(t, addedSubnets, 1)

	// There should be two subnets in the database now.
	returnedSubnets, err := GetAllSubnets(db, 0)
	require.NoError(t, err)
	require.Len(t, returnedSubnets, 2)

	returnedHosts, err := GetHostsBySubnetID(db, returnedSubnets[0].ID)
	require.NoError(t, err)
	require.Len(t, returnedHosts, 1)
	require.Len(t, returnedHosts[0].LocalHosts, 1)
	require.EqualValues(t, app.ID, returnedHosts[0].LocalHosts[0].AppID)

	returnedHosts, err = GetHostsBySubnetID(db, returnedSubnets[1].ID)
	require.NoError(t, err)
	require.Len(t, returnedHosts, 1)
	require.Len(t, returnedHosts[0].LocalHosts, 1)
	require.EqualValues(t, app.ID, returnedHosts[0].LocalHosts[0].AppID)

	// Make sure we can commit the networks again without an error.
	addedSubnets, err = CommitNetworksIntoDB(db, networks, subnets, &app, app.Daemons[0], 1)
	require.NoError(t, err)
	require.Len(t, addedSubnets, 0)
}

// Check if getting subnet family works.
func TestGetSubnetFamily(t *testing.T) {
	// create v4 subnet and check its family
	s4 := &Subnet{Prefix: "192.168.0.0/24"}
	require.EqualValues(t, 4, s4.GetFamily())

	// create v6 subnet and check its family
	s6 := &Subnet{Prefix: "2001:db8:1::0/24"}
	require.EqualValues(t, 6, s6.GetFamily())
}

// Check getting subnets with local subnets.
func TestGetSubnetsWithLocalSubnets(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// prepare apps
	apps := addTestSubnetApps(t, db)
	require.Len(t, apps, 2)

	// prepare a subnet
	subnet := &Subnet{
		Prefix: "192.0.2.0/24",
	}
	err := AddSubnet(db, subnet)
	require.NoError(t, err)
	require.NotZero(t, subnet.ID)

	// add association of the app to the subnet - this will create LocalSubnet
	err = AddAppToSubnet(db, subnet, apps[0], apps[0].Daemons[0])
	require.NoError(t, err)
	require.NotZero(t, subnet.ID)

	// get subnet with its local subnet
	subnets, err := GetSubnetsWithLocalSubnets(db)
	require.NoError(t, err)
	require.Len(t, subnets, 1)
	require.EqualValues(t, "192.0.2.0/24", subnets[0].Prefix)
	require.Len(t, subnets[0].LocalSubnets, 1)
	require.EqualValues(t, 123, subnets[0].LocalSubnets[0].LocalSubnetID)
}

// Check updating utilization in subnet.
func TestUpdateUtilization(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// prepare a subnet
	subnet := &Subnet{
		Prefix: "192.0.2.0/24",
	}
	err := AddSubnet(db, subnet)
	require.NoError(t, err)
	require.NotZero(t, subnet.ID)

	// get subnet and check if utilization is 0
	returnedSubnet, err := GetSubnet(db, subnet.ID)
	require.NoError(t, err)
	require.NotNil(t, returnedSubnet)
	require.EqualValues(t, 0, returnedSubnet.AddrUtilization)
	require.EqualValues(t, 0, returnedSubnet.PdUtilization)

	// update utilization in subnet
	returnedSubnet.UpdateUtilization(db, 10, 20)

	// check if utilization was stored in db
	returnedSubnet2, err := GetSubnet(db, subnet.ID)
	require.NoError(t, err)
	require.NotNil(t, returnedSubnet2)
	require.EqualValues(t, 10, returnedSubnet2.AddrUtilization)
	require.EqualValues(t, 20, returnedSubnet2.PdUtilization)
}

// Test deleting subnets not assigned to any apps.
func TestDeleteOrphanedSubnets(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Add apps used in the test.
	apps := addTestSubnetApps(t, db)
	require.Len(t, apps, 2)

	// Add three subnets.
	subnets := []Subnet{
		{
			Prefix: "192.0.2.0/24",
		},
		{
			Prefix: "192.0.3.0/24",
		},
		{
			Prefix: "192.0.4.0/24",
		},
	}
	for i := range subnets {
		err := AddSubnet(db, &subnets[i])
		require.NoError(t, err)
		require.NotZero(t, subnets[i].ID)
	}

	// Associate one of the subnets with one of the apps. The
	// other two subnets are orphaned.
	err := AddAppToSubnet(db, &subnets[0], apps[0], apps[0].Daemons[0])
	require.NoError(t, err)

	// Delete subnets not assigned to any apps.
	count, err := DeleteOrphanedSubnets(db)
	require.NoError(t, err)
	require.EqualValues(t, 2, count)

	// Ensure that the non-orphaned subnet hasn't been deleted.
	returned, err := GetSubnet(db, subnets[0].ID)
	require.NoError(t, err)
	require.NotNil(t, returned)

	// Ensure that the orphaned subnets have been deleted.
	returned, err = GetSubnet(db, subnets[1].ID)
	require.NoError(t, err)
	require.Nil(t, returned)

	returned, err = GetSubnet(db, subnets[2].ID)
	require.NoError(t, err)
	require.Nil(t, returned)

	// Deleting orphaned subnets again should affect no subnets.
	count, err = DeleteOrphanedSubnets(db)
	require.NoError(t, err)
	require.Zero(t, count)
}

// Test deleting subnets belonging to a shared network not assigned to any apps.
func TestDeleteOrphanedSharedNetworkSubnets(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Add apps used in the test.
	apps := addTestSubnetApps(t, db)
	require.Len(t, apps, 2)

	// Add a shared network with three subnets.
	network := &SharedNetwork{
		Name:   "foo",
		Family: 4,
		Subnets: []Subnet{
			{
				Prefix: "192.0.2.0/24",
			},
			{
				Prefix: "192.0.3.0/24",
			},
			{
				Prefix: "192.0.4.0/24",
			},
		},
	}
	err := AddSharedNetwork(db, network)
	require.NoError(t, err)

	// Associate one of the subnets with one of the apps. The
	// other two subnets are orphaned.
	err = AddAppToSubnet(db, &network.Subnets[0], apps[0], apps[0].Daemons[0])
	require.NoError(t, err)

	// Delete subnets not assigned to any apps.
	count, err := DeleteOrphanedSubnets(db)
	require.NoError(t, err)
	require.EqualValues(t, 2, count)

	// Ensure that the non-orphaned subnet hasn't been deleted.
	returned, err := GetSubnet(db, network.Subnets[0].ID)
	require.NoError(t, err)
	require.NotNil(t, returned)

	// Ensure that the orphaned subnets have been deleted.
	returned, err = GetSubnet(db, network.Subnets[1].ID)
	require.NoError(t, err)
	require.Nil(t, returned)

	returned, err = GetSubnet(db, network.Subnets[2].ID)
	require.NoError(t, err)
	require.Nil(t, returned)

	// Delete the sole daemon from the subnet.
	count, err = DeleteDaemonFromSubnets(db, apps[0].Daemons[0].ID)
	require.NoError(t, err)
	require.EqualValues(t, 1, count)

	// Delete orphaned subnet. It should leave our shared network empty.
	count, err = DeleteOrphanedSubnets(db)
	require.NoError(t, err)
	require.EqualValues(t, 1, count)

	// Make sure the shared network still exists.
	network, err = GetSharedNetworkWithSubnets(db, network.ID)
	require.NoError(t, err)
	require.NotNil(t, network)
	require.Empty(t, network.Subnets)

	// Deleting empty shared networks should remove our network.
	count, err = DeleteEmptySharedNetworks(db)
	require.NoError(t, err)
	require.EqualValues(t, 1, count)
}

// Benchmark measuring a time to add a single subnet.
func BenchmarkAddSubnet(b *testing.B) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(b)
	defer teardown()

	tx, rollback, commit, _ := dbops.Transaction(db)
	defer rollback()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		prefix := fmt.Sprintf("%d.%d.%d.", uint8(i>>16), uint8(i>>8), uint8(i))
		subnet := &Subnet{
			Prefix: prefix + "0/24",
			AddressPools: []AddressPool{
				{
					LowerBound: prefix + "1",
					UpperBound: prefix + "10",
				},
			},
		}
		AddSubnet(tx, subnet)
	}
	commit()
}

// Benchmark measuring a time to associate a subnet with an app. This requires
// finding local subnet ID within the app configuration by subnet prefix.
// In order to find a subnet in the app configuration it is possible to use
// indexed and unindexed subnets. Thus, this benchmark contains two test cases,
// one checking performance of the function with indexing and without indexing.
// The function execution time should be significantly longer without indexing.
func BenchmarkAddAppToSubnet(b *testing.B) {
	testCases := []string{"without indexing", "with indexing"}

	// Run sub tests.
	for _, testCase := range testCases {
		tc := testCase
		b.Run(tc, func(b *testing.B) {
			db, _, teardown := dbtest.SetupDatabaseTestCase(b)
			defer teardown()

			tx, rollback, commit, _ := dbops.Transaction(db)
			defer rollback()

			// Add many subnets to the database.
			subnets := []Subnet{}
			keaSubnets := []interface{}{}
			for i := 0; i < 10000; i++ {
				prefix := fmt.Sprintf("%d.%d.%d.", uint8(i>>16), uint8(i>>8), uint8(i))
				subnet := Subnet{
					Prefix: prefix + "0/24",
				}
				keaSubnet := map[string]interface{}{
					"id":     i + 1,
					"subnet": prefix + "0/24",
				}
				AddSubnet(tx, &subnet)
				subnets = append(subnets, subnet)
				keaSubnets = append(keaSubnets, keaSubnet)
			}
			commit()

			// Also create the configuration including these subnets for the app.
			rawConfig := &map[string]interface{}{
				"Dhcp4": map[string]interface{}{
					"subnet4": keaSubnets,
				},
			}
			daemon := NewKeaDaemon("dhcp4", true)
			daemon.SetConfig(NewKeaConfig(rawConfig))

			// When measuring time with indexing, we need to build indexes before
			// running the actual benchmark.
			if tc == "with indexing" {
				indexedSubnets := keaconfig.NewIndexedSubnets(daemon.KeaDaemon.Config)
				daemon.KeaDaemon.KeaDHCPDaemon.IndexedSubnets = indexedSubnets
			}

			// Add machine/app.
			machine := &Machine{
				ID:        0,
				Address:   "localhost",
				AgentPort: 8080,
			}
			AddMachine(db, machine)
			app := &App{
				ID:        0,
				Type:      AppTypeKea,
				MachineID: machine.ID,
				Daemons: []*Daemon{
					daemon,
				},
			}
			AddApp(db, app)

			// Run the actual benchmark.
			rand.Seed(time.Now().UTC().UnixNano())
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				subnetIndex := rand.Intn(len(subnets))
				AddAppToSubnet(db, &subnets[subnetIndex], app, app.Daemons[0])
			}
		})
	}
}
