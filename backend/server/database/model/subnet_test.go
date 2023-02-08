package dbmodel

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"math/rand"
	"sort"
	"testing"
	"time"

	"github.com/go-pg/pg/v10"
	"github.com/stretchr/testify/require"
	keaconfig "isc.org/stork/appcfg/kea"
	dhcpmodel "isc.org/stork/datamodel/dhcp"
	dbtest "isc.org/stork/server/database/test"
	storkutil "isc.org/stork/util"
)

// Simple mock for utilizationStatistics for testing purposes.
type utilizationStatsMock struct {
	addressUtilization         float64
	delegatedPrefixUtilization float64
	statistics                 SubnetStats
}

func newUtilizationStatsMock(address, pd float64, stats SubnetStats) utilizationStats {
	return &utilizationStatsMock{
		addressUtilization:         address,
		delegatedPrefixUtilization: pd,
		statistics:                 stats,
	}
}

func (m *utilizationStatsMock) GetAddressUtilization() float64 {
	return m.addressUtilization
}

func (m *utilizationStatsMock) GetDelegatedPrefixUtilization() float64 {
	return m.delegatedPrefixUtilization
}

func (m *utilizationStatsMock) GetStatistics() SubnetStats {
	return m.statistics
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

// Test that global subnets are fetched.
func TestGlobalSubnets(t *testing.T) {
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

	// Get global subnets only. It should return two subnets.
	subnets, err = GetGlobalSubnets(db, 4)
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

	subnets, err = GetGlobalSubnets(db, 6)
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
	err = AddDaemonToSubnet(db, subnet, apps[0].Daemons[0])
	require.NoError(t, err)
	require.NotZero(t, subnet.ID)

	// Make sure that this association is returned when fetching the subnet.
	returnedSubnet, err := GetSubnet(db, subnet.ID)
	require.NoError(t, err)
	require.NotNil(t, returnedSubnet)
	require.Len(t, returnedSubnet.LocalSubnets, 1)
	require.EqualValues(t, 123, returnedSubnet.LocalSubnets[0].LocalSubnetID)
	require.NotNil(t, returnedSubnet.LocalSubnets[0].Daemon)
	require.NotNil(t, returnedSubnet.LocalSubnets[0].Daemon.App)

	// Also make sure that the app can be retrieved using the GetApp function.
	returnedApp := returnedSubnet.GetApp(apps[0].ID)
	require.NotNil(t, returnedApp)
	returnedApp = returnedSubnet.GetApp(apps[1].ID)
	require.Nil(t, returnedApp)

	// Add another app to the same subnet.
	err = AddDaemonToSubnet(db, subnet, apps[1].Daemons[0])
	require.NoError(t, err)
	require.NotZero(t, subnet.ID)

	// Get the subnet again and expect two apps to be returned.
	returnedSubnet, err = GetSubnet(db, subnet.ID)
	require.NoError(t, err)
	require.NotNil(t, returnedSubnet)
	require.Len(t, returnedSubnet.LocalSubnets, 2)
	require.EqualValues(t, 123, returnedSubnet.LocalSubnets[0].LocalSubnetID)
	require.NotNil(t, returnedSubnet.LocalSubnets[0].Daemon)
	require.NotNil(t, returnedSubnet.LocalSubnets[0].Daemon.App)
	require.EqualValues(t, 123, returnedSubnet.LocalSubnets[1].LocalSubnetID)
	require.NotNil(t, returnedSubnet.LocalSubnets[1].Daemon)
	require.NotNil(t, returnedSubnet.LocalSubnets[1].Daemon.App)

	// Remove the association of the first daemon with the subnet.
	ok, err := DeleteDaemonFromSubnet(db, subnet.ID, returnedSubnet.LocalSubnets[0].DaemonID)
	require.NoError(t, err)
	require.True(t, ok)

	// Check again that only one daemon is returned.
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
	err := AddDaemonToSubnet(db, &subnets[0], apps[0].Daemons[0])
	require.NoError(t, err)

	err = AddDaemonToSubnet(db, &subnets[1], apps[0].Daemons[0])
	require.NoError(t, err)

	// Associate the second app with another subnet.
	err = AddDaemonToSubnet(db, &subnets[2], apps[1].Daemons[0])
	require.NoError(t, err)

	// Remove associations of the first app.
	count, err := DeleteDaemonFromSubnets(db, apps[0].Daemons[0].ID)
	require.NoError(t, err)
	require.EqualValues(t, 2, count)

	// Ensure that the associations were removed for the first app.
	returned, err := GetSubnetsByDaemonID(db, apps[0].Daemons[0].ID)
	require.NoError(t, err)
	require.Empty(t, returned)

	// The association should still exist for the second app.
	returned, err = GetSubnetsByDaemonID(db, apps[1].Daemons[0].ID)
	require.NoError(t, err)
	require.Len(t, returned, 1)
}

// Test that subnets can be fetched by daemon ID.
func TestGetSubnetsByDaemonID(t *testing.T) {
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

		// Add association of the daemons to the subnets.
		if i < 2 {
			// First two subnets associated with the first daemon.
			err = AddDaemonToSubnet(db, &subnets[i], apps[0].Daemons[0])
		} else {
			// Last subnet is only associated with the second daemon.
			err = AddDaemonToSubnet(db, &subnets[i], apps[1].Daemons[0])
		}
		require.NoError(t, err)
		require.NotZero(t, subnets[i].ID)
	}

	// Get all IPv4 subnets for the first app.
	returnedSubnets, err := GetSubnetsByDaemonID(db, apps[0].Daemons[0].ID)
	require.NoError(t, err)
	require.Len(t, returnedSubnets, 2)

	require.Len(t, returnedSubnets[0].LocalSubnets, 1)
	require.EqualValues(t, 123, returnedSubnets[0].LocalSubnets[0].LocalSubnetID)
	require.EqualValues(t, apps[0].Daemons[0].ID, returnedSubnets[0].LocalSubnets[0].DaemonID)

	require.Len(t, returnedSubnets[1].LocalSubnets, 1)
	require.EqualValues(t, 234, returnedSubnets[1].LocalSubnets[0].LocalSubnetID)
	require.EqualValues(t, apps[0].Daemons[0].ID, returnedSubnets[1].LocalSubnets[0].DaemonID)

	// Get all IPv4 subnets for the second app.
	returnedSubnets, err = GetSubnetsByDaemonID(db, apps[1].Daemons[0].ID)
	require.NoError(t, err)
	require.Len(t, returnedSubnets, 1)

	require.Len(t, returnedSubnets[0].LocalSubnets, 1)
	require.EqualValues(t, 345, returnedSubnets[0].LocalSubnets[0].LocalSubnetID)
	require.EqualValues(t, apps[1].Daemons[0].ID, returnedSubnets[0].LocalSubnets[0].DaemonID)
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
	filters := &SubnetsByPageFilters{
		Text:   newPtr("192.0"),
		Family: newPtr(int64(4)),
	}

	returned, count, err := GetSubnetsByPage(db, 0, 10, filters, "prefix", SortDirDesc)
	require.NoError(t, err)
	require.EqualValues(t, 2, count)
	require.Len(t, returned, 2)
	// The subnets are returned in descending prefix order.
	require.Equal(t, "192.0.3.0/24", returned[0].Prefix)
	require.Equal(t, "192.0.2.0/24", returned[1].Prefix)

	// This should match multiple pools in the first subnet. However,
	// only one record should be returned.
	filters.Text = newPtr("192.0.2.1")
	returned, count, err = GetSubnetsByPage(db, 0, 10, filters, "prefix", SortDirDesc)
	require.NoError(t, err)
	require.EqualValues(t, 1, count)
	require.Len(t, returned, 1)
	require.Equal(t, "192.0.2.0/24", returned[0].Prefix)

	// This should have no match.
	filters.Text = newPtr("192.0.5.0")
	returned, count, err = GetSubnetsByPage(db, 0, 10, filters, "id", SortDirAsc)
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
	err = AddDaemonToSubnet(db, subnet, apps[0].Daemons[0])
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
	err = AddDaemonToSubnet(db, subnet, apps[0].Daemons[0])
	require.NoError(t, err)
	require.NotZero(t, subnet.ID)

	// check fetching LocalSubnets for given app
	subnets, err := GetAppLocalSubnets(db, apps[0].ID)
	require.NoError(t, err)
	require.Len(t, subnets, 1)

	// check updating stats
	lsn := subnets[0]
	lsn.DaemonID = apps[0].Daemons[0].ID
	lsn.SubnetID = subnet.ID
	stats := make(map[string]interface{})
	stats["hakuna-matata"] = 123
	err = lsn.UpdateStats(db, stats)
	require.NoError(t, err)

	// check stored stats
	localSubnets := []*LocalSubnet{}
	err = db.Model(&localSubnets).Select()
	require.NoError(t, err)
	require.Len(t, localSubnets, 1)
	lsn = localSubnets[0]
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
							LocalHosts: []LocalHost{
								{
									DaemonID:   app.Daemons[0].ID,
									DataSource: HostDataSourceConfig,
								},
							},
						},
					},
					LocalSubnets: []*LocalSubnet{
						{
							LocalSubnetID: 13,
						},
					},
				},
			},
			LocalSharedNetworks: []*LocalSharedNetwork{
				{
					DHCPOptionSetHash: "xyz",
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
					LocalHosts: []LocalHost{
						{
							DaemonID:   app.Daemons[0].ID,
							DataSource: HostDataSourceConfig,
						},
					},
				},
			},
			LocalSubnets: []*LocalSubnet{
				{
					SubnetID: 14,
				},
			},
		},
	}
	// Attempt to create the global shared network and subnet.
	addedSubnets, err := CommitNetworksIntoDB(db, networks, subnets, app.Daemons[0])
	require.NoError(t, err)
	require.Len(t, addedSubnets, 1)

	// There should be two subnets in the database now.
	returnedSubnets, err := GetAllSubnets(db, 0)
	require.NoError(t, err)
	require.Len(t, returnedSubnets, 2)
	require.Len(t, returnedSubnets[0].LocalSubnets, 1)
	require.Len(t, returnedSubnets[1].LocalSubnets, 1)

	// There should be one shared network.
	returnedNetworks, err := GetAllSharedNetworks(db, 4)
	require.NoError(t, err)
	require.Len(t, returnedNetworks, 1)
	require.Len(t, returnedNetworks[0].LocalSharedNetworks, 1)

	returnedHosts, err := GetHostsBySubnetID(db, returnedSubnets[0].ID)
	require.NoError(t, err)
	require.Len(t, returnedHosts, 1)
	require.Len(t, returnedHosts[0].LocalHosts, 1)
	require.EqualValues(t, app.Daemons[0].ID, returnedHosts[0].LocalHosts[0].DaemonID)

	returnedHosts, err = GetHostsBySubnetID(db, returnedSubnets[1].ID)
	require.NoError(t, err)
	require.Len(t, returnedHosts, 1)
	require.Len(t, returnedHosts[0].LocalHosts, 1)
	require.EqualValues(t, app.Daemons[0].ID, returnedHosts[0].LocalHosts[0].DaemonID)

	// Make sure we can commit the networks again without an error.
	addedSubnets, err = CommitNetworksIntoDB(db, networks, subnets, app.Daemons[0])
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
	err = AddDaemonToSubnet(db, subnet, apps[0].Daemons[0])
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
	require.Nil(t, returnedSubnet.Stats)
	require.Zero(t, returnedSubnet.StatsCollectedAt)

	// update utilization in subnet
	returnedSubnet.UpdateStatistics(db, newUtilizationStatsMock(0.01, 0.02, SubnetStats{
		"total-nas":    uint64(100),
		"assigned-nas": uint64(1),
		"total-pds":    uint64(100),
		"assigned-pds": uint64(2),
	}))

	// check if utilization was stored in db
	returnedSubnet2, err := GetSubnet(db, subnet.ID)
	require.NoError(t, err)
	require.NotNil(t, returnedSubnet2)
	require.EqualValues(t, 10, returnedSubnet2.AddrUtilization)
	require.EqualValues(t, 20, returnedSubnet2.PdUtilization)
	require.EqualValues(t, 1, returnedSubnet2.Stats["assigned-nas"])
	require.EqualValues(t, 2, returnedSubnet2.Stats["assigned-pds"])
	require.InDelta(t, time.Now().UTC().Unix(), returnedSubnet2.StatsCollectedAt.Unix(), 10.0)
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
	err := AddDaemonToSubnet(db, &subnets[0], apps[0].Daemons[0])
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
	err = AddDaemonToSubnet(db, &network.Subnets[0], apps[0].Daemons[0])
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

// Test that big numbers are serialized as string to avoid losing the precision.
func TestSerializeLocalSubnetWithLargeNumbersInStatisticsToJSON(t *testing.T) {
	// Arrange
	localSubnet := &LocalSubnet{
		Stats: SubnetStats{
			"maxInt64":             int64(math.MaxInt64),
			"minInt64":             int64(math.MinInt64),
			"maxUint64":            uint64(math.MaxUint64),
			"minUint64":            uint64(0),
			"untyped":              42,
			"int16":                int16(16),
			"int32":                int32(32),
			"bigIntInUint64Bounds": big.NewInt(42),
			"bigIntInInt64Bounds":  big.NewInt(-42),
			"bigIntAboveUint64Bounds": big.NewInt(0).Add(
				big.NewInt(0).SetUint64(math.MaxUint64),
				big.NewInt(0).SetUint64(math.MaxUint64),
			),
			"bigIntBelowInt64Bounds": big.NewInt(0).Add(
				big.NewInt(math.MinInt64), big.NewInt(math.MinInt64),
			),
		},
	}

	var deserialized LocalSubnet

	// Act
	serialized, toJSONErr := json.Marshal(localSubnet)
	fromJSONErr := json.Unmarshal(serialized, &deserialized)

	// Assert
	require.NoError(t, toJSONErr)
	require.NoError(t, fromJSONErr)

	// Deserializer loses the original types (!)
	require.Equal(t, uint64(math.MaxInt64), deserialized.Stats["maxInt64"])
	require.Equal(t, int64(math.MinInt64), deserialized.Stats["minInt64"])
	require.Equal(t, uint64(math.MaxUint64), deserialized.Stats["maxUint64"])
	require.Equal(t, uint64(0), deserialized.Stats["minUint64"])

	require.Equal(t, float64(42), deserialized.Stats["untyped"])
	require.Equal(t, float64(16), deserialized.Stats["int16"])
	require.Equal(t, float64(32), deserialized.Stats["int32"])

	require.Equal(t, uint64(42), deserialized.Stats["bigIntInUint64Bounds"])
	require.Equal(t, int64(-42), deserialized.Stats["bigIntInInt64Bounds"])
	require.Equal(t, big.NewInt(0).Add(
		big.NewInt(0).SetUint64(math.MaxUint64),
		big.NewInt(0).SetUint64(math.MaxUint64),
	), deserialized.Stats["bigIntAboveUint64Bounds"])
	require.Equal(t, big.NewInt(0).Add(
		big.NewInt(math.MinInt64), big.NewInt(math.MinInt64),
	), deserialized.Stats["bigIntBelowInt64Bounds"])
}

// Test that the none stats are serialized as nil.
func TestSerializeLocalSubnetWithNoneStatsToJSON(t *testing.T) {
	// Arrange
	localSubnet := &LocalSubnet{
		Stats: nil,
	}

	var deserialized LocalSubnet

	// Act
	serialized, toJSONErr := json.Marshal(localSubnet)
	fromJSONErr := json.Unmarshal(serialized, &deserialized)

	// Assert
	require.NoError(t, toJSONErr)
	require.NoError(t, fromJSONErr)

	require.Nil(t, deserialized.Stats)
}

// Test that the subnet and its pools are updated properly.
func TestUpdateSubnet(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	sharedNetworkFoo := &SharedNetwork{Name: "foo"}
	sharedNetworkBar := &SharedNetwork{Name: "bar"}
	_ = AddSharedNetwork(db, sharedNetworkFoo)
	_ = AddSharedNetwork(db, sharedNetworkBar)

	subnet := &Subnet{
		Prefix:      "fe80::/64",
		ClientClass: "foo",
		AddressPools: []AddressPool{
			{LowerBound: "fe80::1", UpperBound: "fe80::10"},
			{LowerBound: "fe80::100", UpperBound: "fe80::110"},
		},
		PrefixPools: []PrefixPool{
			{Prefix: "fe80:1::/80", DelegatedLen: 96},
			{Prefix: "fe80:2::/80", DelegatedLen: 96},
		},
		SharedNetworkID: sharedNetworkFoo.ID,
		Hosts:           []Host{{Hostname: "foo"}},
	}

	err := AddSubnet(db, subnet)
	require.NoError(t, err)

	// Act
	subnet.ClientClass = "bar"
	subnet.AddressPools = subnet.AddressPools[1:]
	subnet.AddressPools = append(subnet.AddressPools, AddressPool{
		LowerBound: "fe80::1000", UpperBound: "fe80::1010",
	})

	subnet.PrefixPools = subnet.PrefixPools[1:]
	subnet.PrefixPools = append(subnet.PrefixPools, PrefixPool{
		Prefix: "fe80:2::/80", DelegatedLen: 108,
	})

	subnet.SharedNetworkID = sharedNetworkBar.ID
	subnet.Hosts = []Host{{Hostname: "bar"}}

	err = updateSubnetWithPools(db, subnet)

	// Assert
	require.NoError(t, err)
	subnets, _, _ := GetSubnetsByPage(db, 0, 10, nil, "", SortDirAny)
	require.Len(t, subnets, 1)
	subnet = &subnets[0]

	require.EqualValues(t, "bar", subnet.ClientClass)

	require.Len(t, subnet.AddressPools, 2)
	sort.Slice(subnet.AddressPools, func(i, j int) bool {
		return subnet.AddressPools[i].ID < subnet.AddressPools[j].ID
	})
	require.EqualValues(t, 2, subnet.AddressPools[0].ID)
	require.EqualValues(t, 3, subnet.AddressPools[1].ID)
	require.EqualValues(t, "fe80::1000", subnet.AddressPools[1].LowerBound)

	require.Len(t, subnet.PrefixPools, 2)
	sort.Slice(subnet.PrefixPools, func(i, j int) bool {
		return subnet.PrefixPools[i].ID < subnet.PrefixPools[j].ID
	})
	require.EqualValues(t, 2, subnet.PrefixPools[0].ID)
	require.EqualValues(t, 3, subnet.PrefixPools[1].ID)
	require.EqualValues(t, 108, subnet.PrefixPools[1].DelegatedLen)
}

// Test that the new pools are added and existing ones are untouched. The
// out-of-date entries should be removed.
func TestAddAndClearSubnetPools(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	subnetFoo := &Subnet{
		Prefix: "3001::/64",
		AddressPools: []AddressPool{
			{LowerBound: "3001:1::1", UpperBound: "3001:1::10"},
			{LowerBound: "3001:2::1", UpperBound: "3001:2::10"},
			{LowerBound: "3001:3::1", UpperBound: "3001:3::10"},
		},
		PrefixPools: []PrefixPool{
			{Prefix: "3001:10::/80", DelegatedLen: 96},
			{Prefix: "3001:20::/80", DelegatedLen: 96},
			{Prefix: "3001:30::/80", DelegatedLen: 96},
		},
	}

	subnetBar := &Subnet{
		Prefix: "3002::/64",
		AddressPools: []AddressPool{
			{LowerBound: "3002:1::1", UpperBound: "3002:1::10"},
			{LowerBound: "3002:2::1", UpperBound: "3002:2::10"},
		},
		PrefixPools: []PrefixPool{
			{Prefix: "3002:10::/80", DelegatedLen: 96},
			{Prefix: "3002:20::/80", DelegatedLen: 96},
		},
	}

	_ = AddSubnet(db, subnetFoo)
	_ = AddSubnet(db, subnetBar)

	// Act
	subnetFoo.AddressPools[2].UpperBound = "3001:3::42"
	subnetFoo.PrefixPools[2].DelegatedLen = 116
	subnetFoo.AddressPools = subnetFoo.AddressPools[1:]
	subnetBar.PrefixPools = subnetBar.PrefixPools[1:]
	err := db.RunInTransaction(context.Background(), func(tx *pg.Tx) error {
		if err := addAndClearSubnetPools(tx, subnetFoo); err != nil {
			return err
		}
		return addAndClearSubnetPools(tx, subnetBar)
	})

	// Assert
	require.NoError(t, err)
	subnetFoo, err = GetSubnet(db, subnetFoo.ID)
	require.NoError(t, err)
	subnetBar, err = GetSubnet(db, subnetBar.ID)
	require.NoError(t, err)
	require.Len(t, subnetFoo.AddressPools, 2)
	require.Len(t, subnetBar.AddressPools, 2)
	require.Len(t, subnetFoo.PrefixPools, 3)
	require.Len(t, subnetBar.PrefixPools, 1)
	// Update is not supported.
	require.EqualValues(t, "3001:3::10", subnetFoo.AddressPools[1].UpperBound)
	require.EqualValues(t, 96, subnetFoo.PrefixPools[1].DelegatedLen)
}

// Benchmark measuring a time to add a single subnet.
func BenchmarkAddSubnet(b *testing.B) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(b)
	defer teardown()

	tx, _ := db.Begin()
	defer tx.Rollback()

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
	tx.Commit()
}

// Benchmark measuring a time to associate a subnet with a daemon. This requires
// finding local subnet ID within the app configuration by subnet prefix.
// In order to find a subnet in the app configuration it is possible to use
// indexed and unindexed subnets. Thus, this benchmark contains two test cases,
// one checking performance of the function with indexing and without indexing.
// The function execution time should be significantly longer without indexing.
func BenchmarkAddDaemonToSubnet(b *testing.B) {
	testCases := []string{"without indexing", "with indexing"}

	// Run sub tests.
	for _, testCase := range testCases {
		tc := testCase
		b.Run(tc, func(b *testing.B) {
			db, _, teardown := dbtest.SetupDatabaseTestCase(b)
			defer teardown()

			tx, _ := db.Begin()

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
			tx.Commit()

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
				AddDaemonToSubnet(db, &subnets[subnetIndex], app.Daemons[0])
			}
		})
	}
}

// Test that the shorthand for setting IPv4 family works properly.
func TestSubnetsByPageFiltersSetIPv4Family(t *testing.T) {
	// Arrange
	filters := &SubnetsByPageFilters{}

	// Act
	filters.SetIPv4Family()

	// Assert
	require.EqualValues(t, 4, *filters.Family)
}

// Test that the shorthand for setting IPv6 family works properly.
func TestSubnetsByPageFiltersSetIPv6Family(t *testing.T) {
	// Arrange
	filters := &SubnetsByPageFilters{}

	// Act
	filters.SetIPv6Family()

	// Assert
	require.EqualValues(t, 6, *filters.Family)
}

// Test implementation of the keaconfig.Subnet interface (GetID() function).
func TestSubnetGetID(t *testing.T) {
	subnet := Subnet{
		LocalSubnets: []*LocalSubnet{
			{
				SubnetID: 10,
				DaemonID: 110,
			},
			{
				SubnetID: 11,
				DaemonID: 111,
			},
		},
	}
	require.EqualValues(t, 10, subnet.GetID(110))
	require.EqualValues(t, 11, subnet.GetID(111))
	require.Zero(t, subnet.GetID(1000))
}

// Test implementation of the keaconfig.Subnet interface (GetKeaParameters()
// function).
func TestSubnetGetKeaParameters(t *testing.T) {
	subnet := Subnet{
		LocalSubnets: []*LocalSubnet{
			{
				DaemonID: 110,
				KeaParameters: &keaconfig.SubnetParameters{
					Allocator: storkutil.Ptr("random"),
				},
			},
			{
				DaemonID: 111,
				KeaParameters: &keaconfig.SubnetParameters{
					Allocator: storkutil.Ptr("iterative"),
				},
			},
		},
	}
	params0 := subnet.GetKeaParameters(110)
	require.NotNil(t, params0)
	require.Equal(t, "random", *params0.Allocator)
	params1 := subnet.GetKeaParameters(111)
	require.NotNil(t, params1)
	require.Equal(t, "iterative", *params1.Allocator)

	require.Nil(t, subnet.GetKeaParameters(1000))
}

// Test implementation of the dhcpmodel.SubnetAccessor interface (GetPrefix() function).
func TestSubnetGetPrefix(t *testing.T) {
	subnet := Subnet{
		Prefix: "3001::/64",
	}
	require.Equal(t, "3001::/64", subnet.GetPrefix())
}

// Test implementation of the dhcpmodel.SubnetAccessor interface (GetAddressPools() function).
func TestSubnetGetAddressPools(t *testing.T) {
	subnet := Subnet{
		AddressPools: []AddressPool{
			{
				LowerBound: "192.0.2.1",
				UpperBound: "192.0.2.10",
			},
			{
				LowerBound: "192.0.2.20",
				UpperBound: "192.0.2.30",
			},
		},
	}
	pools := subnet.GetAddressPools()
	require.Len(t, pools, 2)
	require.Equal(t, "192.0.2.1", pools[0].GetLowerBound())
	require.Equal(t, "192.0.2.10", pools[0].GetUpperBound())
	require.Equal(t, "192.0.2.20", pools[1].GetLowerBound())
	require.Equal(t, "192.0.2.30", pools[1].GetUpperBound())
}

// Test implementation of the dhcpmodel.SubnetAccessor interface (GetPrefixPools() function).
func TestSubnetGetPrefixPools(t *testing.T) {
	subnet := Subnet{
		PrefixPools: []PrefixPool{
			{
				Prefix:       "2001:db8:1:1::/64",
				DelegatedLen: 80,
			},
			{
				Prefix:       "2001:db8:1:2::/64",
				DelegatedLen: 80,
			},
		},
	}
	pools := subnet.GetPrefixPools()
	require.Len(t, pools, 2)
	require.Equal(t, "2001:db8:1:1::/64", pools[0].GetModel().Prefix)
	require.EqualValues(t, 80, pools[0].GetModel().DelegatedLen)
	require.Equal(t, "2001:db8:1:2::/64", pools[1].GetModel().Prefix)
	require.EqualValues(t, 80, pools[1].GetModel().DelegatedLen)
}

// Test implementation of the dhcpmodel.SubnetAccessor interface (GetDHCPOptions() function).
func TestSubnetGetDHCPOptions(t *testing.T) {
	subnet := Subnet{
		LocalSubnets: []*LocalSubnet{
			{
				DaemonID: 110,
				DHCPOptionSet: []DHCPOption{
					{
						Code:  7,
						Space: dhcpmodel.DHCPv4OptionSpace,
					},
				},
			},
			{
				DaemonID: 111,
				DHCPOptionSet: []DHCPOption{
					{
						Code:  8,
						Space: dhcpmodel.DHCPv4OptionSpace,
					},
				},
			},
		},
	}
	options0 := subnet.GetDHCPOptions(110)
	require.Len(t, options0, 1)
	require.EqualValues(t, 7, options0[0].GetCode())

	options1 := subnet.GetDHCPOptions(111)
	require.Len(t, options1, 1)
	require.EqualValues(t, 8, options1[0].GetCode())

	require.Nil(t, subnet.GetDHCPOptions(1000))
}

// Test that LocalSubnet instance is appended to the Subnet when there is
// no corresponding LocalSubnet, and it is replaced when the corresponding
// LocalSubnet exists.
func TestLocalSubnet(t *testing.T) {
	// Create a subnet with one local subnet.
	subnet := Subnet{
		ID: 123,
		LocalSubnets: []*LocalSubnet{
			{
				DaemonID: 1,
				KeaParameters: &keaconfig.SubnetParameters{
					Allocator: storkutil.Ptr("random"),
				},
			},
		},
	}
	// Create another local shared network and ensure there are now two.
	subnet.SetLocalSubnet(&LocalSubnet{
		DaemonID: 2,
	})
	require.Len(t, subnet.LocalSubnets, 2)
	require.EqualValues(t, 1, subnet.LocalSubnets[0].DaemonID)
	require.EqualValues(t, 2, subnet.LocalSubnets[1].DaemonID)

	// Replace the first instance with a new one.
	subnet.SetLocalSubnet(&LocalSubnet{
		DaemonID: 1,
		KeaParameters: &keaconfig.SubnetParameters{
			Allocator: storkutil.Ptr("iterative"),
		},
	})
	require.Len(t, subnet.LocalSubnets, 2)
	require.EqualValues(t, 1, subnet.LocalSubnets[0].DaemonID)
	require.EqualValues(t, 2, subnet.LocalSubnets[1].DaemonID)
	require.NotNil(t, subnet.LocalSubnets[0].KeaParameters)
	require.Equal(t, "iterative", *subnet.LocalSubnets[0].KeaParameters.Allocator)
}

// Test that LocalSubnets between two Subnet instances can be combined in a
// single instance.
func TestJoinSubnets(t *testing.T) {
	subnet0 := Subnet{
		ID: 1,
		LocalSubnets: []*LocalSubnet{
			{
				DaemonID: 1,
			},
			{
				DaemonID: 2,
			},
		},
	}
	subnet1 := Subnet{
		ID: 1,
		LocalSubnets: []*LocalSubnet{
			{
				DaemonID: 2,
			},
			{
				DaemonID: 3,
			},
		},
	}
	subnet0.Join(&subnet1)
	require.Len(t, subnet0.LocalSubnets, 3)
	require.EqualValues(t, 1, subnet0.LocalSubnets[0].DaemonID)
	require.EqualValues(t, 2, subnet0.LocalSubnets[1].DaemonID)
	require.EqualValues(t, 3, subnet0.LocalSubnets[2].DaemonID)
}
