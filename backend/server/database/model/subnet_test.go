package dbmodel

import (
	"testing"

	"github.com/stretchr/testify/require"

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
	// then and the subnet.
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
	err = AddAppToSubnet(db, subnet, apps[0])
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
	err = AddAppToSubnet(db, subnet, apps[1])
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
	err = AddAppToSubnet(db, subnet, apps[0])
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
			err = AddAppToSubnet(db, &subnets[i], apps[0])
		} else {
			// Last subnet is only associated with the second app.
			err = AddAppToSubnet(db, &subnets[i], apps[1])
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
	err = AddAppToSubnet(db, subnet, apps[0])
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
	err = AddAppToSubnet(db, subnet, apps[0])
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
	accessPoints = AppendAccessPoint(accessPoints, AccessPointControl, "localhost", "", 8000)
	app := App{
		MachineID:    m.ID,
		Type:         AppTypeKea,
		AccessPoints: accessPoints,
	}
	// Add the app to the database.
	err = AddApp(db, &app)
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
	err = CommitNetworksIntoDB(db, networks, subnets, &app)
	require.NoError(t, err)

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
	err = CommitNetworksIntoDB(db, networks, subnets, &app)
	require.NoError(t, err)
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
