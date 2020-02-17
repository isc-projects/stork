package dbmodel

import (
	"testing"

	"github.com/stretchr/testify/require"
	dbops "isc.org/stork/server/database"
	dbtest "isc.org/stork/server/database/test"
)

// Adds apps to be used with subnet tests.
func addTestSubnetApps(t *testing.T, db *dbops.PgDB) (apps []*App) {
	// Add two apps.
	for i := 0; i < 2; i++ {
		m := &Machine{
			ID:        0,
			Address:   "localhost",
			AgentPort: int64(8080 + i),
		}
		err := AddMachine(db, m)
		require.NoError(t, err)

		a := &App{
			ID:          0,
			MachineID:   m.ID,
			Type:        KeaAppType,
			CtrlAddress: "cool.example.org",
			CtrlPort:    int64(1234 + i),
			Active:      true,
			Details: AppKea{
				Daemons: []*KeaDaemon{
					{
						Name:   "dhcp4",
						Config: getTestConfigWithIPv4Subnets(t),
					},
				},
			},
		}

		apps = append(apps, a)
	}

	// Add the apps to be database.
	for _, app := range apps {
		err := AddApp(db, app)
		require.NoError(t, err)
	}
	return apps
}

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
	require.NotZero(t, returned.Created)
	require.Equal(t, subnet.Prefix, returned.Prefix)
	require.Zero(t, returned.SharedNetworkID)
	require.Nil(t, returned.SharedNetwork)

	// Make sure that the address pools and prefix pools are returned.
	require.Len(t, returned.AddressPools, 3)
	require.Len(t, returned.PrefixPools, 3)

	// Validate returned address pools.
	for i, p := range returned.AddressPools {
		require.NotZero(t, p.Created)
		require.Equal(t, subnet.AddressPools[i].LowerBound, p.LowerBound)
		require.Equal(t, subnet.AddressPools[i].UpperBound, p.UpperBound)
	}

	// Validate returned prefix pools.
	for i, p := range returned.PrefixPools {
		require.NotZero(t, p.Created)
		require.Equal(t, subnet.PrefixPools[i].Prefix, p.Prefix)
		require.Equal(t, subnet.PrefixPools[i].DelegatedLen, p.DelegatedLen)
	}
}

// Test that the inserted subnet can be associated with a shared network.
func TestAddSubnetWithExistingSharedNetwork(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// First, add the shared network.
	sharedNetwork := &SharedNetwork{
		Name: "test",
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
	require.Len(t, returnedSubnet.Apps, 1)
	// Local subnet ID should have been copied from the app_to_subnet table.
	require.EqualValues(t, 123, returnedSubnet.Apps[0].LocalSubnetID)

	// Add another app to the same subnet.
	err = AddAppToSubnet(db, subnet, apps[1])
	require.NoError(t, err)
	require.NotZero(t, subnet.ID)

	// Get the subnet again and expect two apps to be returned.
	returnedSubnet, err = GetSubnet(db, subnet.ID)
	require.NoError(t, err)
	require.NotNil(t, returnedSubnet)
	require.Len(t, returnedSubnet.Apps, 2)
	require.EqualValues(t, 123, returnedSubnet.Apps[0].LocalSubnetID)
	require.EqualValues(t, 123, returnedSubnet.Apps[1].LocalSubnetID)

	// Remove the association of the first app with the subnet.
	ok, err := DeleteAppFromSubnet(db, subnet.ID, returnedSubnet.Apps[0].ID)
	require.NoError(t, err)
	require.True(t, ok)

	// Check again that only one app is returned.
	returnedSubnet, err = GetSubnet(db, subnet.ID)
	require.NoError(t, err)
	require.NotNil(t, returnedSubnet)
	require.Len(t, returnedSubnet.Apps, 1)
}
