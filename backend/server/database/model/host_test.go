package dbmodel

import (
	"testing"

	"github.com/go-pg/pg/v9"
	"github.com/stretchr/testify/require"
	dbops "isc.org/stork/server/database"
	dbtest "isc.org/stork/server/database/test"
)

// This function creates multiple hosts used in tests which fetch and
// filter hosts.
func addTestHosts(t *testing.T, db *pg.DB) []Host {
	subnets := []Subnet{
		{
			ID:     1,
			Prefix: "192.0.2.0/24",
		},
		{
			ID:     2,
			Prefix: "2001:db8:1::/64",
		},
	}
	for i, s := range subnets {
		subnet := s
		err := AddSubnet(db, &subnet)
		require.NoError(t, err)
		require.NotZero(t, subnet.ID)
		subnets[i] = subnet
	}

	hosts := []Host{
		{
			SubnetID: 1,
			HostIdentifiers: []HostIdentifier{
				{
					Type:  "hw-address",
					Value: []byte{1, 2, 3, 4, 5, 6},
				},
				{
					Type:  "circuit-id",
					Value: []byte{1, 2, 3, 4},
				},
			},
			IPReservations: []IPReservation{
				{
					Address: "192.0.2.4/32",
				},
				{
					Address: "192.0.2.5/32",
				},
			},
			Hostname: "first.example.org",
		},
		{
			HostIdentifiers: []HostIdentifier{
				{
					Type:  "hw-address",
					Value: []byte{2, 3, 4, 5, 6, 7},
				},
				{
					Type:  "circuit-id",
					Value: []byte{2, 3, 4, 5},
				},
			},
			IPReservations: []IPReservation{
				{
					Address: "192.0.2.6/32",
				},
				{
					Address: "192.0.2.7/32",
				},
			},
		},
		{
			SubnetID: 2,
			HostIdentifiers: []HostIdentifier{
				{
					Type:  "hw-address",
					Value: []byte{1, 2, 3, 4, 5, 6},
				},
			},
			IPReservations: []IPReservation{
				{
					Address: "2001:db8:1::1/128",
				},
			},
			Hostname: "second.example.org",
		},
		{
			HostIdentifiers: []HostIdentifier{
				{
					Type:  "duid",
					Value: []byte{1, 2, 3, 4},
				},
			},
			IPReservations: []IPReservation{
				{
					Address: "2001:db8:1::2/128",
				},
			},
		},
	}

	for i, h := range hosts {
		host := h
		err := AddHost(db, &host)
		require.NoError(t, err)
		require.NotZero(t, host.ID)
		hosts[i] = host
	}
	return hosts
}

// This test verifies that the new host along with identifiers and reservations
// can be added to the database.
func TestAddHost(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Add a host with two identifiers and two reservations.
	host := &Host{
		Hostname: "host.example.org",
		HostIdentifiers: []HostIdentifier{
			{
				Type:  "hw-address",
				Value: []byte{1, 2, 3, 4, 5, 6},
			},
			{
				Type:  "circuit-id",
				Value: []byte{1, 1, 1, 1, 1},
			},
		},
		IPReservations: []IPReservation{
			{
				Address: "192.0.2.4",
			},
			{
				Address: "2001:db8:1::4",
			},
		},
	}
	err := AddHost(db, host)
	require.NoError(t, err)
	require.NotZero(t, host.ID)

	// Get the host from the database.
	returned, err := GetHost(db, host.ID)
	require.NoError(t, err)
	require.NotNil(t, returned)

	require.Equal(t, host.ID, returned.ID)
	require.Equal(t, "host.example.org", host.Hostname)
	require.Len(t, returned.HostIdentifiers, 2)
	require.Len(t, returned.IPReservations, 2)

	// Make sure that the returned host identifiers match.
	for i := range returned.HostIdentifiers {
		require.Contains(t, returned.HostIdentifiers, host.HostIdentifiers[i])
	}

	// Make sure that the returned reservations match.
	for i := range returned.IPReservations {
		require.Contains(t, returned.IPReservations[i].Address, host.IPReservations[i].Address)
	}
}

// Test that the host can be updated and that this update includes extending
// the list of reservations and identifiers.
func TestUpdateHostExtend(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Add the host with two reservations and two identifiers.
	host := &Host{
		HostIdentifiers: []HostIdentifier{
			{
				Type:  "hw-address",
				Value: []byte{1, 2, 3, 4, 5, 6},
			},
			{
				Type:  "circuit-id",
				Value: []byte{1, 1, 1, 1, 1},
			},
		},
		IPReservations: []IPReservation{
			{
				Address: "192.0.2.4/32",
			},
			{
				Address: "2001:db8:1::4/128",
			},
		},
	}
	err := AddHost(db, host)
	require.NoError(t, err)
	require.NotZero(t, host.ID)

	// Modify the value of the first identifier.
	host.HostIdentifiers[0].Value = []byte{6, 5, 4, 3, 2, 1}
	// Modify the identifier type of the second identifier.
	host.HostIdentifiers[1].Type = "client-id"
	// Add one more identifier.
	host.HostIdentifiers = append(host.HostIdentifiers, HostIdentifier{
		Type:  "flex-id",
		Value: []byte{2, 2, 2, 2, 2},
	})

	// Modify the first reservation.
	host.IPReservations[0].Address = "192.0.3.4/32"
	// Add one more reservation.
	host.IPReservations = append(host.IPReservations, IPReservation{
		Address: "3000::/64",
	})

	// Not only does updating the host modify the host value but also adds
	// or removes reservations and identifiers.
	err = UpdateHost(db, host)
	require.NoError(t, err)
	require.NotZero(t, host.ID)

	// Get the updated host.
	returned, err := GetHost(db, host.ID)
	require.NoError(t, err)
	require.NotNil(t, returned)

	require.Len(t, returned.HostIdentifiers, 3)
	require.Len(t, returned.IPReservations, 3)

	// Make sure that the identifiers and reservations were modified.
	require.ElementsMatch(t, returned.HostIdentifiers, host.HostIdentifiers)
	require.ElementsMatch(t, returned.IPReservations, host.IPReservations)
}

// Test that the host can be updated and that some reservations and
// host identifiers are deleted as a result of this update.
func TestUpdateHostShrink(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	host := &Host{
		HostIdentifiers: []HostIdentifier{
			{
				Type:  "hw-address",
				Value: []byte{1, 2, 3, 4, 5, 6},
			},
			{
				Type:  "circuit-id",
				Value: []byte{1, 1, 1, 1, 1},
			},
		},
		IPReservations: []IPReservation{
			{
				Address: "192.0.2.4/32",
			},
			{
				Address: "2001:db8:1::4/128",
			},
		},
	}
	err := AddHost(db, host)
	require.NoError(t, err)
	require.NotZero(t, host.ID)

	// Remove one host identifier and one reservation.
	host.HostIdentifiers = host.HostIdentifiers[0:1]
	host.IPReservations = host.IPReservations[1:]

	// Updating the host should result in removal of this identifier
	// and the reservation from the database.
	err = UpdateHost(db, host)
	require.NoError(t, err)
	require.NotZero(t, host.ID)

	// Get the updated host.
	returned, err := GetHost(db, host.ID)
	require.NoError(t, err)
	require.NotNil(t, returned)

	// Verify that only one identifier and one reservation have left.
	require.Len(t, returned.HostIdentifiers, 1)
	require.Len(t, returned.IPReservations, 1)

	require.Equal(t, "hw-address", returned.HostIdentifiers[0].Type)
	require.Equal(t, []byte{1, 2, 3, 4, 5, 6}, returned.HostIdentifiers[0].Value)

	require.Equal(t, "2001:db8:1::4/128", returned.IPReservations[0].Address)
}

// Test that all hosts or all hosts having IP reservations of specified family
// can be fetched.
func TestGetAllHosts(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Add four hosts. Two with IPv4 and two with IPv6 reservations.
	hosts := addTestHosts(t, db)

	// Fetch all hosts having IPv4 reservations.
	returned, err := GetAllHosts(db, 4)
	require.NoError(t, err)
	require.Len(t, returned, 2)

	require.Contains(t, returned, hosts[0])
	require.Contains(t, returned, hosts[1])

	// Fetch all hosts having IPv6 reservations.
	returned, err = GetAllHosts(db, 6)
	require.NoError(t, err)
	require.Len(t, returned, 2)

	require.Contains(t, returned, hosts[2])
	require.Contains(t, returned, hosts[3])

	// Fetch all hosts.
	returned, err = GetAllHosts(db, 0)
	require.NoError(t, err)
	require.Len(t, returned, 4)

	for _, host := range hosts {
		require.Contains(t, returned, host)
	}
}

// Test that hosts can be fetched by subnet ID.
func TestGetHostsBySubnetID(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Add four hosts. Two with IPv4 and two with IPv6 reservations.
	hosts := addTestHosts(t, db)

	// Fetch host having IPv4 reservations.
	returned, err := GetHostsBySubnetID(db, 1)
	require.NoError(t, err)
	require.Len(t, returned, 1)
	require.Contains(t, returned, hosts[0])

	// Fetch host having IPv6 reservations.
	returned, err = GetHostsBySubnetID(db, 2)
	require.NoError(t, err)
	require.Len(t, returned, 1)
	require.Contains(t, returned, hosts[2])
}

// Test that page of the hosts can be fetched without filtering.
func TestGetHostsByPageNoFiltering(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Add four hosts. Two with IPv4 and two with IPv6 reservations.
	_ = addTestHosts(t, db)

	returned, total, err := GetHostsByPage(db, 0, 10, 0, nil, nil, nil, "", SortDirAny)
	require.NoError(t, err)
	require.EqualValues(t, 4, total)
	require.Len(t, returned, 4)
}

// Test that page of the hosts can be fetched with filtering by subnet id.
func TestGetHostsByPageSubnet(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Add four hosts. Two with IPv4 and two with IPv6 reservations.
	hosts := addTestHosts(t, db)

	// Get global hosts only.
	subnetID := int64(0)
	returned, total, err := GetHostsByPage(db, 0, 10, 0, &subnetID, nil, nil, "", SortDirAny)
	require.NoError(t, err)
	require.EqualValues(t, 2, total)
	require.Len(t, returned, 2)
	require.Contains(t, returned, hosts[1])
	require.Contains(t, returned, hosts[3])

	// Get hosts associated with subnet id 1.
	subnetID = int64(1)
	returned, total, err = GetHostsByPage(db, 0, 10, 0, &subnetID, nil, nil, "", SortDirAny)
	require.NoError(t, err)
	require.EqualValues(t, 1, total)
	require.Len(t, returned, 1)
	require.EqualValues(t, hosts[0].ID, returned[0].ID)
	require.EqualValues(t, 1, returned[0].SubnetID)
	require.NotNil(t, returned[0].Subnet)
	require.Equal(t, "192.0.2.0/24", returned[0].Subnet.Prefix)
	require.ElementsMatch(t, returned[0].HostIdentifiers, hosts[0].HostIdentifiers)
	require.ElementsMatch(t, returned[0].IPReservations, hosts[0].IPReservations)

	// Get hosts associated with subnet id 2.
	subnetID = int64(2)
	returned, total, err = GetHostsByPage(db, 0, 10, 0, &subnetID, nil, nil, "", SortDirAny)
	require.NoError(t, err)
	require.EqualValues(t, 1, total)
	require.Len(t, returned, 1)
	require.EqualValues(t, hosts[2].ID, returned[0].ID)
	require.EqualValues(t, 2, returned[0].SubnetID)
	require.NotNil(t, returned[0].Subnet)
	require.Equal(t, "2001:db8:1::/64", returned[0].Subnet.Prefix)
	require.ElementsMatch(t, returned[0].HostIdentifiers, hosts[2].HostIdentifiers)
	require.ElementsMatch(t, returned[0].IPReservations, hosts[2].IPReservations)
}

// Test that page of the hosts can be fetched with filtering by app id.
func TestGetHostsByPageApp(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Insert apps and hosts into the database.
	apps := addTestSubnetApps(t, db)
	hosts := addTestHosts(t, db)

	// Associate the first host with the first app.
	err := AddAppToHost(db, &hosts[0], apps[0], "api", 1)
	require.NoError(t, err)
	err = AddAppToHost(db, &hosts[1], apps[0], "api", 1)
	require.NoError(t, err)
	err = AddAppToHost(db, &hosts[2], apps[1], "api", 1)
	require.NoError(t, err)
	err = AddAppToHost(db, &hosts[3], apps[1], "api", 1)
	require.NoError(t, err)

	// Get global hosts only.
	returned, total, err := GetHostsByPage(db, 0, 10, apps[0].ID, nil, nil, nil, "", SortDirAny)
	require.NoError(t, err)
	require.EqualValues(t, 2, total)
	require.Len(t, returned, 2)
	require.True(t,
		(returned[0].ID == hosts[0].ID && returned[1].ID == hosts[1].ID) ||
			(returned[0].ID == hosts[1].ID && returned[1].ID == hosts[2].ID))
}

// Test that page of the hosts can be filtered by IP reservations and
// hostnames.
func TestGetHostsByPageFilteringText(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Add four hosts. Two with IPv4 and two with IPv6 reservations.
	hosts := addTestHosts(t, db)

	filterText := "0.2.4"
	returned, total, err := GetHostsByPage(db, 0, 10, 0, nil, &filterText, nil, "", SortDirAny)
	require.NoError(t, err)
	require.EqualValues(t, 1, total)
	require.Len(t, returned, 1)
	require.NotNil(t, returned[0].Subnet)
	require.Equal(t, "192.0.2.0/24", returned[0].Subnet.Prefix)
	// Reset subnet, so as we can use Contain function to compare the rest of the
	// host information.
	returned[0].Subnet = nil
	require.Contains(t, returned, hosts[0])

	filterText = "192.0.2"
	returned, total, err = GetHostsByPage(db, 0, 10, 0, nil, &filterText, nil, "", SortDirAny)
	require.NoError(t, err)
	require.EqualValues(t, 2, total)
	require.Len(t, returned, 2)
	require.NotNil(t, returned[0].Subnet)
	require.Equal(t, "192.0.2.0/24", returned[0].Subnet.Prefix)
	require.Nil(t, returned[1].Subnet)
	// Reset subnet, so as we can use Contain function to compare the rest of the
	// host information.
	returned[0].Subnet = nil
	require.Contains(t, returned, hosts[0])
	require.Contains(t, returned, hosts[1])

	filterText = "0"
	returned, total, err = GetHostsByPage(db, 0, 10, 0, nil, &filterText, nil, "", SortDirAny)
	require.NoError(t, err)
	require.EqualValues(t, 4, total)
	require.Len(t, returned, 4)

	for i := range returned {
		returned[i].Subnet = nil
	}

	require.ElementsMatch(t, returned, hosts)

	// Filter by identifier value.
	filterText = "01:02:03"
	returned, total, err = GetHostsByPage(db, 0, 10, 0, nil, &filterText, nil, "", SortDirAny)
	require.NoError(t, err)
	require.EqualValues(t, 3, total)
	require.Len(t, returned, 3)

	// Filter by identifier type.
	filterText = "dui"
	returned, total, err = GetHostsByPage(db, 0, 10, 0, nil, &filterText, nil, "", SortDirAny)
	require.NoError(t, err)
	require.EqualValues(t, 1, total)
	require.Len(t, returned, 1)
	require.Contains(t, returned, hosts[3])

	// Filter by hostname.
	filterText = "example"
	returned, total, err = GetHostsByPage(db, 0, 10, 0, nil, &filterText, nil, "", SortDirAny)
	require.NoError(t, err)
	require.EqualValues(t, 2, total)
	require.Len(t, returned, 2)
	// Reset subnet, so as we can use Contain function to compare the rest of the
	// host information.
	returned[0].Subnet = nil
	returned[1].Subnet = nil
	require.Contains(t, returned, hosts[0])
	require.Contains(t, returned, hosts[2])
}

// Test that page of the hosts can be global/not global hosts.
func TestGetHostsByPageGlobal(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Add four hosts. Two global and two non-global.
	hosts := addTestHosts(t, db)

	// find only global hosts
	global := true
	returned, total, err := GetHostsByPage(db, 0, 10, 0, nil, nil, &global, "", SortDirAny)
	require.NoError(t, err)
	require.EqualValues(t, 2, total)
	require.Len(t, returned, 2)
	require.Contains(t, returned, hosts[1])
	require.Contains(t, returned, hosts[3])

	// find only non-global hosts
	global = false
	returned, total, err = GetHostsByPage(db, 0, 10, 0, nil, nil, &global, "", SortDirAny)
	require.NoError(t, err)
	require.EqualValues(t, 2, total)
	require.Len(t, returned, 2)
	require.Contains(t, []int64{hosts[0].ID, hosts[2].ID}, returned[0].ID)
	require.Contains(t, []int64{hosts[0].ID, hosts[2].ID}, returned[1].ID)
}

// Test hosts can be sorted by different fields.
func TestGetHostsByPageWithSorting(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Add four hosts. Two with IPv4 and two with IPv6 reservations.
	addTestHosts(t, db)

	// check sorting by id asc
	returned, total, err := GetHostsByPage(db, 0, 10, 0, nil, nil, nil, "id", SortDirAsc)
	require.NoError(t, err)
	require.EqualValues(t, 4, total)
	require.Len(t, returned, 4)
	require.EqualValues(t, 1, returned[0].ID)
	require.EqualValues(t, 4, returned[3].ID)

	// check sorting by id desc
	returned, total, err = GetHostsByPage(db, 0, 10, 0, nil, nil, nil, "id", SortDirDesc)
	require.NoError(t, err)
	require.EqualValues(t, 4, total)
	require.Len(t, returned, 4)
	require.EqualValues(t, 4, returned[0].ID)
	require.EqualValues(t, 1, returned[3].ID)

	// check sorting by subnet_id asc
	returned, total, err = GetHostsByPage(db, 0, 10, 0, nil, nil, nil, "subnet_id", SortDirAsc)
	require.NoError(t, err)
	require.EqualValues(t, 4, total)
	require.Len(t, returned, 4)
	require.EqualValues(t, 2, returned[0].ID)
	require.EqualValues(t, 3, returned[3].ID)

	// check sorting by subnet_id desc
	returned, total, err = GetHostsByPage(db, 0, 10, 0, nil, nil, nil, "subnet_id", SortDirDesc)
	require.NoError(t, err)
	require.EqualValues(t, 4, total)
	require.Len(t, returned, 4)
	require.EqualValues(t, 3, returned[0].ID)
	require.EqualValues(t, 4, returned[3].ID)
}

// Test that the host and its identifiers and reservations can be
// deleted.
func TestDeleteHost(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	host := &Host{
		HostIdentifiers: []HostIdentifier{
			{
				Type:  "hw-address",
				Value: []byte{1, 2, 3, 4, 5, 6},
			},
		},
		IPReservations: []IPReservation{
			{
				Address: "192.0.2.4/32",
			},
		},
	}
	err := AddHost(db, host)
	require.NoError(t, err)
	require.NotZero(t, host.ID)

	err = DeleteHost(db, host.ID)
	require.NoError(t, err)

	returned, err := GetHost(db, host.ID)
	require.NoError(t, err)
	require.Nil(t, returned)
}

// Test that an app can be associated with a host.
func TestAddAppToHost(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Insert apps and hosts into the database.
	apps := addTestSubnetApps(t, db)
	hosts := addTestHosts(t, db)

	// Associate the first host with the first app.
	host := hosts[0]
	err := AddAppToHost(db, &host, apps[0], "api", 1)
	require.NoError(t, err)

	// Fetch the host from the database.
	returned, err := GetHost(db, host.ID)
	require.NoError(t, err)
	require.NotNil(t, returned)

	// Make sure that the host includes the local host information which
	// associates the app with the host.
	require.Len(t, returned.LocalHosts, 1)
	require.Equal(t, "api", returned.LocalHosts[0].DataSource)
	require.EqualValues(t, apps[0].ID, returned.LocalHosts[0].AppID)
	// When fetching one selected host the app information should be also
	// returned.
	require.NotNil(t, returned.LocalHosts[0].App)

	// Get all hosts.
	returnedList, err := GetAllHosts(db, 0)
	require.NoError(t, err)
	require.Len(t, returnedList, 4)
	require.Len(t, returnedList[0].LocalHosts, 1)
	require.Equal(t, "api", returnedList[0].LocalHosts[0].DataSource)
	require.EqualValues(t, apps[0].ID, returnedList[0].LocalHosts[0].AppID)
	// When fetching all hosts, the detailed app information should not be returned.
	require.Nil(t, returnedList[0].LocalHosts[0].App)

	// Get the first host by reserved IP address.
	filterText := "192.0.2.4"
	returnedList, total, err := GetHostsByPage(db, 0, 10, 0, nil, &filterText, nil, "", SortDirAny)
	require.NoError(t, err)
	require.EqualValues(t, 1, total)
	require.Len(t, returnedList, 1)
	require.Len(t, returnedList[0].LocalHosts, 1)
	require.Equal(t, "api", returnedList[0].LocalHosts[0].DataSource)
	require.EqualValues(t, apps[0].ID, returnedList[0].LocalHosts[0].AppID)
	// When fetching all hosts, the detailed app information
	// should be returned as well.
	require.NotNil(t, returnedList[0].LocalHosts[0].App)
}

// Tests that a host which is no longer associated with any app is deleted
// from the database.
func TestDeleteDanglingHosts(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Insert apps and hosts into the database.
	apps := addTestSubnetApps(t, db)
	hosts := addTestHosts(t, db)

	// Associate two apps with a host.
	host := hosts[0]
	err := AddAppToHost(db, &host, apps[0], "api", 1)
	require.NoError(t, err)
	err = AddAppToHost(db, &host, apps[1], "api", 1)
	require.NoError(t, err)

	// Delete the first app. The host is still associated with the second
	// app so it should still exist.
	err = DeleteApp(db, apps[0])
	require.NoError(t, err)

	// Make sure it is returned.
	filterText := "192.0.2.4"
	returnedList, total, err := GetHostsByPage(db, 0, 10, 0, nil, &filterText, nil, "", SortDirAny)
	require.NoError(t, err)
	require.EqualValues(t, 1, total)
	require.Len(t, returnedList, 1)

	// Delete the second app. The host is no longer associated with any
	// app and should get deleted automatically.
	err = DeleteApp(db, apps[1])
	require.NoError(t, err)

	// Make sure the host is no longer returned.
	returnedList, total, err = GetHostsByPage(db, 0, 10, 0, nil, &filterText, nil, "", SortDirAny)
	require.NoError(t, err)
	require.EqualValues(t, 0, total)
	require.Empty(t, returnedList)
}

// This test verifies that it is possible to delete hosts/apps associations
// having non-matching sequence numbers.
func TestDeleteLocalHostsWithOtherSeq(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Insert apps and hosts into the database.
	apps := addTestSubnetApps(t, db)
	hosts := addTestHosts(t, db)

	returned, err := GetAllHosts(db, 4)
	require.NoError(t, err)
	require.Len(t, returned, 2)

	// Insert association of the apps with hosts and give them a sequence number
	// of 123.
	err = AddAppToHost(db, &hosts[0], apps[0], "api", 123)
	require.NoError(t, err)
	err = AddAppToHost(db, &hosts[1], apps[1], "api", 123)
	require.NoError(t, err)

	// Use matching sequence number.
	err = DeleteLocalHostsWithOtherSeq(db, 123, "api")
	require.NoError(t, err)

	// The hosts should still be there.
	returned, err = GetAllHosts(db, 4)
	require.NoError(t, err)
	require.Len(t, returned, 2)

	// The same result of the data source is not matching.
	err = DeleteLocalHostsWithOtherSeq(db, 234, "config")
	require.NoError(t, err)

	returned, err = GetAllHosts(db, 4)
	require.NoError(t, err)
	require.Len(t, returned, 2)

	// This time, use the non-matching sequence number.
	err = DeleteLocalHostsWithOtherSeq(db, 234, "api")
	require.NoError(t, err)

	// The hosts should be gone because all associations of
	// these hosts with apps were removed.
	returned, err = GetAllHosts(db, 4)
	require.NoError(t, err)
	require.Empty(t, returned)
}

// Tests that function getting next sequence number works correctly.
func TestGetNextBulkSeq(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	for i := 0; i < 10; i++ {
		seq, err := GetNextBulkUpdateSeq(db)
		require.NoError(t, err)
		require.EqualValues(t, i+2, seq)
	}
}

// Tests the function checking if the host includes a reservation for the
// given IP address.
func TestHasIPAddress(t *testing.T) {
	host := Host{
		HostIdentifiers: []HostIdentifier{
			{
				Type:  "hw-address",
				Value: []byte{1, 2, 3, 4, 5, 6},
			},
			{
				Type:  "circuit-id",
				Value: []byte{1, 2, 3, 4},
			},
		},
		IPReservations: []IPReservation{
			{
				Address: "192.0.2.4/32",
			},
			{
				Address: "192.0.2.5/32",
			},
		},
	}

	require.True(t, host.HasIPAddress("192.0.2.4"))
	require.True(t, host.HasIPAddress("192.0.2.4/32"))
	require.True(t, host.HasIPAddress("192.0.2.5"))
	require.False(t, host.HasIPAddress("192.0.2.7/32"))
}

// Tests the function checking if the host includes a given identifier.
func TestHasIdentifier(t *testing.T) {
	host := Host{
		HostIdentifiers: []HostIdentifier{
			{
				Type:  "hw-address",
				Value: []byte{1, 2, 3, 4, 5, 6},
			},
			{
				Type:  "circuit-id",
				Value: []byte{1, 2, 3, 4},
			},
		},
		IPReservations: []IPReservation{
			{
				Address: "192.0.2.4/32",
			},
			{
				Address: "192.0.2.5/32",
			},
		},
	}

	exists, equal := host.HasIdentifier("hw-address", []byte{1, 2, 3, 4, 5, 6})
	require.True(t, exists)
	require.True(t, equal)
	require.True(t, host.HasIdentifierType("hw-address"))

	exists, equal = host.HasIdentifier("circuit-id", []byte{1, 2, 3, 4})
	require.True(t, exists)
	require.True(t, equal)
	require.True(t, host.HasIdentifierType("circuit-id"))

	exists, equal = host.HasIdentifier("hw-address", []byte{1, 2, 3, 4})
	require.True(t, exists)
	require.False(t, equal)
	require.True(t, host.HasIdentifierType("hw-address"))

	exists, equal = host.HasIdentifier("duid", []byte{1, 2, 3, 4})
	require.False(t, exists)
	require.False(t, equal)
	require.False(t, host.HasIdentifierType("duid"))
}

// Test the functions which compares two hosts for equality and which
// compare IP reservations for equality.
func TestHostsEqual(t *testing.T) {
	host1 := Host{
		HostIdentifiers: []HostIdentifier{
			{
				Type:  "hw-address",
				Value: []byte{1, 2, 3, 4, 5, 6},
			},
			{
				Type:  "circuit-id",
				Value: []byte{1, 2, 3, 4},
			},
		},
		IPReservations: []IPReservation{
			{
				Address: "192.0.2.4/32",
			},
			{
				Address: "192.0.2.5/32",
			},
		},
	}

	host2 := Host{
		HostIdentifiers: []HostIdentifier{
			{
				Type:  "circuit-id",
				Value: []byte{1, 2, 3, 4},
			},
			{
				Type:  "hw-address",
				Value: []byte{1, 2, 3, 4, 5, 6},
			},
		},
		IPReservations: []IPReservation{
			{
				Address: "192.0.2.5/32",
			},
			{
				Address: "192.0.2.4/32",
			},
		},
	}

	require.True(t, host1.Equal(&host2))
	require.True(t, host2.Equal(&host1))
	require.True(t, host1.HasEqualIPReservations(&host2))
	require.True(t, host2.HasEqualIPReservations(&host1))

	host1.HostIdentifiers = append(host1.HostIdentifiers, HostIdentifier{
		Type:  "client-id",
		Value: []byte{1, 1, 1, 1},
	})
	host1.IPReservations = append(host1.IPReservations, IPReservation{
		Address: "192.0.2.6",
	})

	require.False(t, host1.Equal(&host2))
	require.False(t, host2.Equal(&host1))
	require.False(t, host1.HasEqualIPReservations(&host2))
	require.False(t, host2.HasEqualIPReservations(&host1))
}

func TestHostIdentifierToHex(t *testing.T) {
	id := HostIdentifier{
		Value: []byte{1, 2, 3, 4, 5, 0xa, 0xb},
	}
	require.Equal(t, "01:02:03:04:05:0a:0b", id.ToHex(":"))
	require.Equal(t, "01020304050a0b", id.ToHex(""))
}

// Tests that global host reservations and their associations with the apps
// are properly stored in the database.
func TestCommitGlobalHostsIntoDB(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()
	tx, _, commit, err := dbops.Transaction(db)
	require.NoError(t, err)

	apps := addTestSubnetApps(t, db)

	// Create two global hosts. The global hosts have no subnet ID.
	hosts := []Host{
		{
			HostIdentifiers: []HostIdentifier{
				{
					Type:  "hw-address",
					Value: []byte{1, 2, 3, 4, 5, 6},
				},
			},
			IPReservations: []IPReservation{
				{
					Address: "192.0.2.56",
				},
			},
		},
		{
			HostIdentifiers: []HostIdentifier{
				{
					Type:  "client-id",
					Value: []byte{1, 2, 3, 4},
				},
			},
			IPReservations: []IPReservation{
				{
					Address: "192.0.2.156",
				},
			},
		},
	}
	// Add the hosts and their associations with the app to the database.
	err = CommitGlobalHostsIntoDB(tx, hosts, apps[0], "api", 1)
	require.NoError(t, err)
	require.NoError(t, commit())

	// Fetch global hosts.
	returned, err := GetHostsBySubnetID(db, 0)
	require.NoError(t, err)
	require.Len(t, returned, 2)

	// Make sure that the returned hosts are associated with the given app
	// and that they remain global, i.e. subnet id is unspecified.
	for _, h := range returned {
		require.Len(t, h.LocalHosts, 1)
		require.EqualValues(t, apps[0].ID, h.LocalHosts[0].AppID)
		require.Zero(t, h.SubnetID)
	}
}
