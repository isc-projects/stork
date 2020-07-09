package restservice

import (
	"context"
	"testing"

	"github.com/go-pg/pg/v9"
	"github.com/stretchr/testify/require"

	agentcommtest "isc.org/stork/server/agentcomm/test"
	dbmodel "isc.org/stork/server/database/model"
	dbtest "isc.org/stork/server/database/test"
	"isc.org/stork/server/gen/models"
	dhcp "isc.org/stork/server/gen/restapi/operations/d_h_c_p"
	storktest "isc.org/stork/server/test"
)

// This function creates multiple hosts used in tests which fetch and
// filter hosts.
func addTestHosts(t *testing.T, db *pg.DB) (hosts []dbmodel.Host, apps []dbmodel.App) {
	// Add two apps.
	for i := 0; i < 2; i++ {
		m := &dbmodel.Machine{
			ID:        0,
			Address:   "localhost",
			AgentPort: int64(8080 + i),
		}
		err := dbmodel.AddMachine(db, m)
		require.NoError(t, err)

		accessPoints := []*dbmodel.AccessPoint{}
		accessPoints = dbmodel.AppendAccessPoint(accessPoints, dbmodel.AccessPointControl, "cool.example.org", "", int64(1234+i))

		a := dbmodel.App{
			ID:           0,
			MachineID:    m.ID,
			Type:         dbmodel.AppTypeKea,
			Active:       true,
			AccessPoints: accessPoints,
		}

		apps = append(apps, a)
	}

	subnets := []dbmodel.Subnet{
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
		err := dbmodel.AddSubnet(db, &subnet)
		require.NoError(t, err)
		require.NotZero(t, subnet.ID)
		subnets[i] = subnet
	}

	hosts = []dbmodel.Host{
		{
			SubnetID: 1,
			Hostname: "first.example.org",
			HostIdentifiers: []dbmodel.HostIdentifier{
				{
					Type:  "hw-address",
					Value: []byte{1, 2, 3, 4, 5, 6},
				},
				{
					Type:  "circuit-id",
					Value: []byte{1, 2, 3, 4},
				},
			},
			IPReservations: []dbmodel.IPReservation{
				{
					Address: "192.0.2.4",
				},
				{
					Address: "192.0.2.5",
				},
			},
		},
		{
			HostIdentifiers: []dbmodel.HostIdentifier{
				{
					Type:  "hw-address",
					Value: []byte{2, 3, 4, 5, 6, 7},
				},
				{
					Type:  "circuit-id",
					Value: []byte{2, 3, 4, 5},
				},
			},
			IPReservations: []dbmodel.IPReservation{
				{
					Address: "192.0.2.6",
				},
				{
					Address: "192.0.2.7",
				},
			},
		},
		{
			SubnetID: 2,
			HostIdentifiers: []dbmodel.HostIdentifier{
				{
					Type:  "hw-address",
					Value: []byte{1, 2, 3, 4, 5, 6},
				},
			},
			IPReservations: []dbmodel.IPReservation{
				{
					Address: "2001:db8:1::1",
				},
			},
		},
		{
			HostIdentifiers: []dbmodel.HostIdentifier{
				{
					Type:  "duid",
					Value: []byte{1, 2, 3, 4},
				},
			},
			IPReservations: []dbmodel.IPReservation{
				{
					Address: "2001:db8:1::2",
				},
			},
		},
		{
			HostIdentifiers: []dbmodel.HostIdentifier{
				{
					Type:  "duid",
					Value: []byte{2, 2, 2, 2},
				},
			},
			IPReservations: []dbmodel.IPReservation{
				{
					Address: "3000::/48",
				},
			},
		},
	}

	// Add apps to the database.
	for i, a := range apps {
		app := a
		_, err := dbmodel.AddApp(db, &app)
		require.NoError(t, err)
		require.NotZero(t, app.ID)
		apps[i] = app
	}

	// Add hosts to the database.
	for i, h := range hosts {
		host := h
		err := dbmodel.AddHost(db, &host)
		require.NoError(t, err)
		require.NotZero(t, host.ID)
		hosts[i] = host
	}
	return hosts, apps
}

// Test that all hosts can be fetched without filtering.
func TestGetHostsNoFiltering(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	settings := RestAPISettings{}
	fa := agentcommtest.NewFakeAgents(nil, nil)
	fec := &storktest.FakeEventCenter{}
	rapi, err := NewRestAPI(&settings, dbSettings, db, fa, fec)
	require.NoError(t, err)
	ctx := context.Background()

	// Add four hosts. Two with IPv4 and two with IPv6 reservations.
	hosts, apps := addTestHosts(t, db)

	err = dbmodel.AddAppToHost(db, &hosts[0], &apps[0], "config", 1)
	require.NoError(t, err)
	err = dbmodel.AddAppToHost(db, &hosts[0], &apps[1], "config", 1)
	require.NoError(t, err)

	params := dhcp.GetHostsParams{}
	rsp := rapi.GetHosts(ctx, params)
	require.IsType(t, &dhcp.GetHostsOK{}, rsp)
	okRsp := rsp.(*dhcp.GetHostsOK)
	require.Len(t, okRsp.Payload.Items, 5)
	require.EqualValues(t, 5, okRsp.Payload.Total)

	items := okRsp.Payload.Items
	require.NotNil(t, items)

	// There should be a total of 5 hosts, 4 of them including IP address
	// reservations and 1 with a prefix reservation.
	require.Len(t, items, 5)
	for i := range items {
		require.NotNil(t, items[i])
		require.NotZero(t, items[i].ID)
		require.EqualValues(t, hosts[i].SubnetID, items[i].SubnetID)

		// Check that the host identifiers types match.
		require.EqualValues(t, len(items[i].HostIdentifiers), len(hosts[i].HostIdentifiers))
		for j := range items[i].HostIdentifiers {
			require.NotNil(t, items[i].HostIdentifiers[j])
			require.EqualValues(t, hosts[i].HostIdentifiers[j].Type, items[i].HostIdentifiers[j].IDType)
		}

		// The total number of reservations, which includes both address and
		// prefix reservations should be equal to the number of reservations for
		// a given host.
		require.EqualValues(t, len(hosts[i].IPReservations),
			len(items[i].AddressReservations)+len(items[i].PrefixReservations))

		// Walk over the address and prefix reservations for a host.
		for _, ips := range [][]*models.IPReservation{items[i].AddressReservations, items[i].PrefixReservations} {
			for j, resrv := range ips {
				require.NotNil(t, resrv)
				require.EqualValues(t, hosts[i].IPReservations[j].Address, resrv.Address)
			}
		}
	}

	// The identifiers should have been converted to hex values.
	require.EqualValues(t, "01:02:03:04:05:06", items[0].HostIdentifiers[0].IDHexValue)
	require.EqualValues(t, "01:02:03:04", items[0].HostIdentifiers[1].IDHexValue)
	require.EqualValues(t, "02:03:04:05:06:07", items[1].HostIdentifiers[0].IDHexValue)
	require.EqualValues(t, "02:03:04:05", items[1].HostIdentifiers[1].IDHexValue)
	require.EqualValues(t, "01:02:03:04:05:06", items[2].HostIdentifiers[0].IDHexValue)
	require.EqualValues(t, "01:02:03:04", items[3].HostIdentifiers[0].IDHexValue)
	require.EqualValues(t, "02:02:02:02", items[4].HostIdentifiers[0].IDHexValue)

	require.Equal(t, "192.0.2.0/24", items[0].SubnetPrefix)
	require.Empty(t, items[1].SubnetPrefix)
	require.NotNil(t, "2001:db8:1::/64", items[2].SubnetPrefix)
	require.Empty(t, items[3].SubnetPrefix)
	require.Empty(t, items[4].SubnetPrefix)

	// Hosts
	require.Equal(t, "first.example.org", items[0].Hostname)

	// The first host should be associated with two apps.
	require.Len(t, items[0].LocalHosts, 2)
	require.NotNil(t, items[0].LocalHosts[0])
	require.EqualValues(t, apps[0].ID, items[0].LocalHosts[0].AppID)
	require.Equal(t, "config", items[0].LocalHosts[0].DataSource)
	require.Equal(t, "cool.example.org:1234", items[0].LocalHosts[0].MachineAddress)
	require.NotNil(t, items[0].LocalHosts[1])
	require.EqualValues(t, apps[1].ID, items[0].LocalHosts[1].AppID)
	require.Equal(t, "config", items[0].LocalHosts[1].DataSource)
	require.Equal(t, "cool.example.org:1235", items[0].LocalHosts[1].MachineAddress)
}

// Test that hosts can be filtered by subnet ID.
func TestGetHostsBySubnetID(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	settings := RestAPISettings{}
	fa := agentcommtest.NewFakeAgents(nil, nil)
	fec := &storktest.FakeEventCenter{}
	rapi, err := NewRestAPI(&settings, dbSettings, db, fa, fec)
	require.NoError(t, err)
	ctx := context.Background()

	// Add four hosts. Two with IPv4 and two with IPv6 reservations.
	_, _ = addTestHosts(t, db)

	subnetID := int64(2)
	params := dhcp.GetHostsParams{
		SubnetID: &subnetID,
	}
	rsp := rapi.GetHosts(ctx, params)
	require.IsType(t, &dhcp.GetHostsOK{}, rsp)
	okRsp := rsp.(*dhcp.GetHostsOK)
	require.Len(t, okRsp.Payload.Items, 1)
	require.EqualValues(t, 1, okRsp.Payload.Total)
}

// Test that hosts can be filtered by text.
func TestGetHostsWithFiltering(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	settings := RestAPISettings{}
	fa := agentcommtest.NewFakeAgents(nil, nil)
	fec := &storktest.FakeEventCenter{}
	rapi, err := NewRestAPI(&settings, dbSettings, db, fa, fec)
	require.NoError(t, err)
	ctx := context.Background()

	// Add four hosts. Two with IPv4 and two with IPv6 reservations.
	_, _ = addTestHosts(t, db)

	filteringText := "2001:db"
	params := dhcp.GetHostsParams{
		Text: &filteringText,
	}
	rsp := rapi.GetHosts(ctx, params)
	require.IsType(t, &dhcp.GetHostsOK{}, rsp)
	okRsp := rsp.(*dhcp.GetHostsOK)
	require.Len(t, okRsp.Payload.Items, 2)
	require.EqualValues(t, 2, okRsp.Payload.Total)
}
