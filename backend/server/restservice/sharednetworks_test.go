package restservice

import (
	context "context"
	"fmt"
	http "net/http"
	"testing"

	"github.com/stretchr/testify/require"
	dhcpmodel "isc.org/stork/datamodel/dhcp"
	agentcommtest "isc.org/stork/server/agentcomm/test"
	apps "isc.org/stork/server/apps"
	"isc.org/stork/server/apps/kea"
	appstest "isc.org/stork/server/apps/test"
	dbmodel "isc.org/stork/server/database/model"
	dbmodeltest "isc.org/stork/server/database/model/test"
	dbtest "isc.org/stork/server/database/test"
	"isc.org/stork/server/gen/models"
	dhcp "isc.org/stork/server/gen/restapi/operations/d_h_c_p"
	storktest "isc.org/stork/server/test/dbmodel"
	"isc.org/stork/testutil"
	storkutil "isc.org/stork/util"
)

// Check getting shared networks via rest api functions.
func TestGetSharedNetworks(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	settings := RestAPISettings{}
	fa := agentcommtest.NewFakeAgents(nil, nil)
	fec := &storktest.FakeEventCenter{}
	fd := &storktest.FakeDispatcher{}
	rapi, err := NewRestAPI(&settings, dbSettings, db, fa, fec, nil, fd, nil)
	require.NoError(t, err)
	ctx := context.Background()

	// get empty list of subnets
	params := dhcp.GetSharedNetworksParams{}
	rsp := rapi.GetSharedNetworks(ctx, params)
	require.IsType(t, &dhcp.GetSharedNetworksOK{}, rsp)
	okRsp := rsp.(*dhcp.GetSharedNetworksOK)
	require.Len(t, okRsp.Payload.Items, 0)
	require.Zero(t, okRsp.Payload.Total)

	dhcp4, err := dbmodeltest.NewKeaDHCPv4Server(db)
	require.NoError(t, err)

	dhcp4.Configure(`{
		"Dhcp4": {
			"shared-networks": [
			  {
				"name": "frog",
				  "subnet4": [
					{
						"id":     11,
						"subnet": "192.1.0.0/24"
					}
				  ]
			  },
			  {
				  "name": "mouse",
				  "subnet4": [
					{
						"id":     12,
						"subnet": "192.3.0.0/24"
					},
					{
						"id":     13,
						"subnet": "192.2.0.0/24"
					}
				  ]
			  }
		  ]
		}
	}`)

	app, err := dhcp4.GetKea()
	require.NoError(t, err)

	err = kea.CommitAppIntoDB(db, app, &storktest.FakeEventCenter{}, nil, dbmodel.NewDHCPOptionDefinitionLookup())
	require.NoError(t, err)

	a4 := app

	dhcp6, err := dbmodeltest.NewKeaDHCPv6Server(db)
	require.NoError(t, err)

	dhcp6.Configure(`{
		"Dhcp6": {
			"shared-networks": [
				{
					"name": "fox",
					"subnet6": [
						{
							"id":     21,
							"subnet": "6001:db8:1::/64"
						},
						{
							"id":     22,
							"subnet": "5001:db8:1::/64"
						}
					]
				},
				{
					"name": "monkey"
				}
			]
		}
	}`)

	app, err = dhcp6.GetKea()
	require.NoError(t, err)

	err = kea.CommitAppIntoDB(db, app, &storktest.FakeEventCenter{}, nil, dbmodel.NewDHCPOptionDefinitionLookup())
	require.NoError(t, err)

	// get all shared networks
	params = dhcp.GetSharedNetworksParams{}
	rsp = rapi.GetSharedNetworks(ctx, params)
	require.IsType(t, &dhcp.GetSharedNetworksOK{}, rsp)
	okRsp = rsp.(*dhcp.GetSharedNetworksOK)
	require.Len(t, okRsp.Payload.Items, 4)
	require.EqualValues(t, 4, okRsp.Payload.Total)
	for _, net := range okRsp.Payload.Items {
		require.Contains(t, []string{"frog", "mouse", "fox", "monkey"}, net.Name)
		switch net.Name {
		case "frog":
			require.Len(t, net.Subnets, 1)
			require.EqualValues(t, 1, net.Subnets[0].ID)
			require.Equal(t, "192.1.0.0/24", net.Subnets[0].Subnet)
			require.EqualValues(t, storkutil.IPv4, net.Universe)
		case "mouse":
			require.Len(t, net.Subnets, 2)
			// subnets should be sorted by prefix, not by ID
			require.EqualValues(t, 3, net.Subnets[0].ID)
			require.Equal(t, "192.2.0.0/24", net.Subnets[0].Subnet)
			require.EqualValues(t, 2, net.Subnets[1].ID)
			require.Equal(t, "192.3.0.0/24", net.Subnets[1].Subnet)
			require.EqualValues(t, storkutil.IPv4, net.Universe)
		case "fox":
			require.Len(t, net.Subnets, 2)
			// subnets should be sorted by prefix, not by ID
			require.EqualValues(t, 5, net.Subnets[0].ID)
			require.Equal(t, "5001:db8:1::/64", net.Subnets[0].Subnet)
			require.EqualValues(t, 4, net.Subnets[1].ID)
			require.Equal(t, "6001:db8:1::/64", net.Subnets[1].Subnet)
			require.EqualValues(t, storkutil.IPv6, net.Universe)
		case "monkey":
			require.Empty(t, net.Subnets)
		}
	}

	// get shared networks from app a4
	params = dhcp.GetSharedNetworksParams{
		AppID: &a4.ID,
	}
	rsp = rapi.GetSharedNetworks(ctx, params)
	require.IsType(t, &dhcp.GetSharedNetworksOK{}, rsp)
	okRsp = rsp.(*dhcp.GetSharedNetworksOK)
	require.Len(t, okRsp.Payload.Items, 2)
	require.EqualValues(t, 2, okRsp.Payload.Total)
	require.Equal(t, a4.ID, okRsp.Payload.Items[0].Subnets[0].LocalSubnets[0].AppID)
	require.Equal(t, a4.Daemons[0].ID, okRsp.Payload.Items[0].Subnets[0].LocalSubnets[0].DaemonID)
	require.Equal(t, a4.Name, okRsp.Payload.Items[0].Subnets[0].LocalSubnets[0].AppName)
	require.Equal(t, a4.ID, okRsp.Payload.Items[1].Subnets[0].LocalSubnets[0].AppID)
	require.Equal(t, a4.Daemons[0].ID, okRsp.Payload.Items[1].Subnets[0].LocalSubnets[0].DaemonID)
	require.Equal(t, a4.Name, okRsp.Payload.Items[1].Subnets[0].LocalSubnets[0].AppName)
	require.Nil(t, okRsp.Payload.Items[1].Subnets[0].LocalSubnets[0].Stats)
	require.ElementsMatch(t, []string{"mouse", "frog"}, []string{okRsp.Payload.Items[0].Name, okRsp.Payload.Items[1].Name})
}

// Test getting a shared network with a detailed DHCP configuration over the REST API.
func TestGetSharedNetwork4(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	settings := RestAPISettings{}
	fa := agentcommtest.NewFakeAgents(nil, nil)
	fec := &storktest.FakeEventCenter{}
	fd := &storktest.FakeDispatcher{}
	rapi, err := NewRestAPI(&settings, dbSettings, db, fa, fec, nil, fd, dbmodel.NewDHCPOptionDefinitionLookup())
	require.NoError(t, err)
	ctx := context.Background()

	// Create DHCPv4 server in the database.
	dhcp4, err := dbmodeltest.NewKeaDHCPv4Server(db)
	require.NoError(t, err)

	// Apply the configuration including all configuration keys.
	err = dhcp4.Configure(string(testutil.AllKeysDHCPv4JSON))
	require.NoError(t, err)

	app, err := dhcp4.GetKea()
	require.NoError(t, err)

	// Populate subnets and shared networks from the Kea configuration.
	err = kea.CommitAppIntoDB(db, app, &storktest.FakeEventCenter{}, nil, dbmodel.NewDHCPOptionDefinitionLookup())
	require.NoError(t, err)

	sharedNetworks, err := dbmodel.GetAllSharedNetworks(db, 4)
	require.NoError(t, err)
	require.Len(t, sharedNetworks, 1)
	require.Len(t, sharedNetworks[0].LocalSharedNetworks, 1)

	// Get the shared network over the REST API.
	params := dhcp.GetSharedNetworkParams{
		ID: sharedNetworks[0].ID,
	}
	rsp := rapi.GetSharedNetwork(ctx, params)
	require.IsType(t, &dhcp.GetSharedNetworkOK{}, rsp)
	okRsp := rsp.(*dhcp.GetSharedNetworkOK)
	sharedNetwork := okRsp.Payload
	require.NotNil(t, sharedNetwork)

	require.Len(t, sharedNetwork.LocalSharedNetworks, 1)
	ls := sharedNetwork.LocalSharedNetworks[0]

	// Validate shared-network-level parameters
	require.NotNil(t, ls.KeaConfigSharedNetworkParameters)
	networkParams := ls.KeaConfigSharedNetworkParameters.SharedNetworkLevelParameters
	require.NotNil(t, networkParams)

	// allocator
	require.NotNil(t, networkParams.Allocator)
	require.Equal(t, "random", *networkParams.Allocator)
	// authoritative
	require.NotNil(t, networkParams.Authoritative)
	require.False(t, *networkParams.Authoritative)
	// boot-file-name
	require.NotNil(t, networkParams.BootFileName)
	require.Equal(t, "/dev/null", *networkParams.BootFileName)
	// client-class
	require.Nil(t, networkParams.ClientClass)
	// ddns-generated-prefix
	require.NotNil(t, networkParams.DdnsGeneratedPrefix)
	require.Equal(t, "myhost", *networkParams.DdnsGeneratedPrefix)
	// ddns-override-client-update
	require.NotNil(t, networkParams.DdnsOverrideClientUpdate)
	require.False(t, *networkParams.DdnsOverrideClientUpdate)
	// ddns-override-no-update
	require.NotNil(t, networkParams.DdnsOverrideNoUpdate)
	require.False(t, *networkParams.DdnsOverrideNoUpdate)
	// ddns-qualifying-suffix
	require.Nil(t, networkParams.DdnsQualifyingSuffix)
	// ddns-replace-client-name
	require.NotNil(t, networkParams.DdnsReplaceClientName)
	require.Equal(t, "never", *networkParams.DdnsReplaceClientName)
	// ddns-send-updates
	require.NotNil(t, networkParams.DdnsSendUpdates)
	require.True(t, *networkParams.DdnsSendUpdates)
	// ddns-update-on-renew
	require.NotNil(t, networkParams.DdnsUpdateOnRenew)
	require.True(t, *networkParams.DdnsUpdateOnRenew)
	// ddns-use-conflict-resolution
	require.NotNil(t, networkParams.DdnsUseConflictResolution)
	require.True(t, *networkParams.DdnsUseConflictResolution)
	// ddns-conflict-resolution-mode
	require.NotNil(t, networkParams.DdnsConflictResolutionMode)
	require.Equal(t, "check-with-dhcid", *networkParams.DdnsConflictResolutionMode)
	// ddns-ttl-percent
	require.NotNil(t, networkParams.DdnsTTLPercent)
	require.EqualValues(t, float32(0.65), *networkParams.DdnsTTLPercent)
	// hostname-char-replacement
	require.NotNil(t, networkParams.HostnameCharReplacement)
	require.Equal(t, "x", *networkParams.HostnameCharReplacement)
	// hostname-char-set
	require.NotNil(t, networkParams.HostnameCharSet)
	require.Equal(t, "[^A-Za-z0-9.-]", *networkParams.HostnameCharSet)
	// interface
	require.NotNil(t, networkParams.Interface)
	require.Equal(t, "eth0", *networkParams.Interface)
	// match-client-id
	require.NotNil(t, networkParams.MatchClientID)
	require.True(t, *networkParams.MatchClientID)
	// next-server
	require.NotNil(t, networkParams.NextServer)
	require.Equal(t, "192.0.2.123", *networkParams.NextServer)
	// store-extended-info
	require.NotNil(t, networkParams.StoreExtendedInfo)
	require.False(t, *networkParams.StoreExtendedInfo)
	// rebind-timer
	require.NotNil(t, networkParams.RebindTimer)
	require.EqualValues(t, 41, *networkParams.RebindTimer)
	// relay
	require.NotNil(t, networkParams.Relay)
	require.Empty(t, networkParams.Relay.IPAddresses)
	// renew-timer
	require.NotNil(t, networkParams.RenewTimer)
	require.EqualValues(t, 31, *networkParams.RenewTimer)
	// reservation-mode
	require.NotNil(t, networkParams.ReservationMode)
	require.Equal(t, "all", *networkParams.ReservationMode)
	// reservations-global
	require.NotNil(t, networkParams.ReservationsGlobal)
	require.False(t, *networkParams.ReservationsGlobal)
	// reservations-in-subnet
	require.NotNil(t, networkParams.ReservationsInSubnet)
	require.True(t, *networkParams.ReservationsInSubnet)
	// reservations-out-of-pool
	require.NotNil(t, networkParams.ReservationsOutOfPool)
	require.False(t, *networkParams.ReservationsOutOfPool)
	// calculate-tee-times
	require.NotNil(t, networkParams.CalculateTeeTimes)
	require.True(t, *networkParams.CalculateTeeTimes)
	// t1-percent
	require.NotNil(t, networkParams.T1Percent)
	require.EqualValues(t, 0.5, *networkParams.T1Percent)
	// t2-percent
	require.NotNil(t, networkParams.T2Percent)
	require.EqualValues(t, 0.75, *networkParams.T2Percent)
	// cache-max-age
	require.NotNil(t, networkParams.CacheMaxAge)
	require.EqualValues(t, 1000, *networkParams.CacheMaxAge)
	// require-client-classes
	require.Len(t, networkParams.RequireClientClasses, 1)
	require.Equal(t, "late", networkParams.RequireClientClasses[0])
	// server-hostname
	require.Nil(t, networkParams.ServerHostname)
	// valid-lifetime
	require.NotNil(t, networkParams.ValidLifetime)
	require.EqualValues(t, 6001, *networkParams.ValidLifetime)
	// min-valid-lifetime
	require.NotNil(t, networkParams.MinValidLifetime)
	require.EqualValues(t, 4001, *networkParams.MinValidLifetime)
	// max-valid-lifetime
	require.NotNil(t, networkParams.MaxValidLifetime)
	require.EqualValues(t, 8001, *networkParams.MaxValidLifetime)

	// Validate the options.
	require.NotEmpty(t, networkParams.OptionsHash)
	require.Len(t, networkParams.Options, 1)
	require.False(t, networkParams.Options[0].AlwaysSend)
	require.EqualValues(t, 3, networkParams.Options[0].Code)
	require.Empty(t, networkParams.Options[0].Encapsulate)
	require.Len(t, networkParams.Options[0].Fields, 1)
	require.Equal(t, dhcpmodel.IPv4AddressField, networkParams.Options[0].Fields[0].FieldType)
	require.Len(t, networkParams.Options[0].Fields[0].Values, 1)
	require.Equal(t, "192.0.3.2", networkParams.Options[0].Fields[0].Values[0])
	require.EqualValues(t, storkutil.IPv4, networkParams.Options[0].Universe)

	// Validate global parameters.
	globalParams := ls.KeaConfigSharedNetworkParameters.GlobalParameters
	require.NotNil(t, globalParams)

	// allocator
	require.NotNil(t, globalParams.Allocator)
	require.Equal(t, "iterative", *globalParams.Allocator)
	// authoritative
	require.NotNil(t, globalParams.Authoritative)
	require.False(t, *globalParams.Authoritative)
	// boot-file-name
	require.NotNil(t, globalParams.BootFileName)
	require.Equal(t, "/dev/null", *globalParams.BootFileName)
	// ddns-generated-prefix
	require.NotNil(t, globalParams.DdnsGeneratedPrefix)
	require.Equal(t, "myhost", *globalParams.DdnsGeneratedPrefix)
	// ddns-override-client-update
	require.NotNil(t, globalParams.DdnsOverrideClientUpdate)
	require.False(t, *globalParams.DdnsOverrideClientUpdate)
	// ddns-override-no-update
	require.NotNil(t, globalParams.DdnsOverrideNoUpdate)
	require.False(t, *globalParams.DdnsOverrideNoUpdate)
	// ddns-qualifying-suffix
	require.Nil(t, globalParams.DdnsQualifyingSuffix)
	// ddns-replace-client-name
	require.NotNil(t, globalParams.DdnsReplaceClientName)
	require.Equal(t, "never", *globalParams.DdnsReplaceClientName)
	// ddns-send-updates
	require.NotNil(t, globalParams.DdnsSendUpdates)
	require.True(t, *globalParams.DdnsSendUpdates)
	// ddns-update-on-renew
	require.NotNil(t, globalParams.DdnsUpdateOnRenew)
	require.True(t, *globalParams.DdnsUpdateOnRenew)
	// ddns-use-conflict-resolution
	require.NotNil(t, globalParams.DdnsUseConflictResolution)
	require.True(t, *globalParams.DdnsUseConflictResolution)
	// ddns-conflict-resolution-mode
	require.NotNil(t, networkParams.DdnsConflictResolutionMode)
	require.Equal(t, "check-with-dhcid", *networkParams.DdnsConflictResolutionMode)
	// hostname-char-replacement
	require.NotNil(t, globalParams.HostnameCharReplacement)
	require.Equal(t, "x", *globalParams.HostnameCharReplacement)
	// hostname-char-set
	require.NotNil(t, globalParams.HostnameCharSet)
	require.Equal(t, "[^A-Za-z0-9.-]", *globalParams.HostnameCharSet)
	// match-client-id
	require.NotNil(t, globalParams.MatchClientID)
	require.False(t, *globalParams.MatchClientID)
	// next-server
	require.NotNil(t, globalParams.NextServer)
	require.Equal(t, "192.0.2.123", *globalParams.NextServer)
	// store-extended-info
	require.NotNil(t, globalParams.StoreExtendedInfo)
	require.True(t, *globalParams.StoreExtendedInfo)
	// rebind-timer
	require.NotNil(t, globalParams.RebindTimer)
	require.EqualValues(t, 40, *globalParams.RebindTimer)
	// renew-timer
	require.NotNil(t, globalParams.RenewTimer)
	require.EqualValues(t, 30, *globalParams.RenewTimer)
	// reservation-mode
	require.NotNil(t, globalParams.ReservationMode)
	require.Equal(t, "all", *globalParams.ReservationMode)
	// reservations-global
	require.NotNil(t, globalParams.ReservationsGlobal)
	require.False(t, *globalParams.ReservationsGlobal)
	// reservations-in-subnet
	require.NotNil(t, globalParams.ReservationsInSubnet)
	require.True(t, *globalParams.ReservationsInSubnet)
	// reservations-out-of-pool
	require.NotNil(t, globalParams.ReservationsOutOfPool)
	require.False(t, *globalParams.ReservationsOutOfPool)
	// calculate-tee-times
	require.NotNil(t, globalParams.CalculateTeeTimes)
	require.True(t, *globalParams.CalculateTeeTimes)
	// t1-percent
	require.NotNil(t, globalParams.T1Percent)
	require.EqualValues(t, 0.5, *globalParams.T1Percent)
	// t2-percent
	require.NotNil(t, globalParams.T2Percent)
	require.EqualValues(t, 0.75, *globalParams.T2Percent)
	// cache-max-age
	require.NotNil(t, globalParams.CacheMaxAge)
	require.EqualValues(t, 1000, *globalParams.CacheMaxAge)
	// server-hostname
	require.Nil(t, globalParams.ServerHostname)
	// valid-lifetime
	require.NotNil(t, globalParams.ValidLifetime)
	require.EqualValues(t, 6000, *globalParams.ValidLifetime)
	// min-valid-lifetime
	require.NotNil(t, globalParams.MinValidLifetime)
	require.EqualValues(t, 4000, *globalParams.MinValidLifetime)
	// max-valid-lifetime
	require.NotNil(t, globalParams.MaxValidLifetime)
	require.EqualValues(t, 8000, *globalParams.MaxValidLifetime)

	// Validate the options.
	require.NotEmpty(t, globalParams.OptionsHash)
	require.Len(t, globalParams.Options, 1)
	require.False(t, globalParams.Options[0].AlwaysSend)
	require.EqualValues(t, 6, globalParams.Options[0].Code)
	require.Empty(t, globalParams.Options[0].Encapsulate)
	require.Len(t, globalParams.Options[0].Fields, 2)
	require.Equal(t, dhcpmodel.IPv4AddressField, globalParams.Options[0].Fields[0].FieldType)
	require.Equal(t, dhcpmodel.IPv4AddressField, globalParams.Options[0].Fields[1].FieldType)
	require.Len(t, globalParams.Options[0].Fields[0].Values, 1)
	require.Equal(t, "192.0.3.1", globalParams.Options[0].Fields[0].Values[0])
	require.Len(t, globalParams.Options[0].Fields[1].Values, 1)
	require.Equal(t, "192.0.3.2", globalParams.Options[0].Fields[1].Values[0])
	require.EqualValues(t, storkutil.IPv4, globalParams.Options[0].Universe)
}

// Test getting an IPv4 shared network over the REST API when the DHCP configuration
// contains no explicit parameters.
func TestGetSharedNetwork4MinimalParameters(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	settings := RestAPISettings{}
	fa := agentcommtest.NewFakeAgents(nil, nil)
	fec := &storktest.FakeEventCenter{}
	fd := &storktest.FakeDispatcher{}
	rapi, err := NewRestAPI(&settings, dbSettings, db, fa, fec, nil, fd, nil)
	require.NoError(t, err)
	ctx := context.Background()

	// Create a new Kea DHCPv4 server instance in the database.
	dhcp4, err := dbmodeltest.NewKeaDHCPv4Server(db)
	require.NoError(t, err)

	// Apply a minimal server configuration.
	cfg := `{
		"Dhcp4": {
			"shared-networks": [
				{
					"name": "foo",
					"subnet4": [
						{
							"subnet": "192.0.2.0/24"
						}
					]
				}
			]
		}
	}`
	err = dhcp4.Configure(cfg)
	require.NoError(t, err)

	app, err := dhcp4.GetKea()
	require.NoError(t, err)

	// Populate the subnets in the database.
	err = kea.CommitAppIntoDB(db, app, &storktest.FakeEventCenter{}, nil, dbmodel.NewDHCPOptionDefinitionLookup())
	require.NoError(t, err)

	sharedNetworks, err := dbmodel.GetAllSharedNetworks(db, 4)
	require.NoError(t, err)
	require.Len(t, sharedNetworks, 1)

	params := dhcp.GetSharedNetworkParams{
		ID: sharedNetworks[0].ID,
	}
	rsp := rapi.GetSharedNetwork(ctx, params)
	require.IsType(t, &dhcp.GetSharedNetworkOK{}, rsp)
	okRsp := rsp.(*dhcp.GetSharedNetworkOK)
	sharedNetwork := okRsp.Payload
	require.NotNil(t, sharedNetwork)

	require.Len(t, sharedNetwork.LocalSharedNetworks, 1)
	ls := sharedNetwork.LocalSharedNetworks[0]

	require.NotNil(t, ls.KeaConfigSharedNetworkParameters)
}

// Test getting a subnet with a detailed DHCP configuration over the REST API.
func TestGetSharedNetwork6(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	settings := RestAPISettings{}
	fa := agentcommtest.NewFakeAgents(nil, nil)
	fec := &storktest.FakeEventCenter{}
	fd := &storktest.FakeDispatcher{}
	rapi, err := NewRestAPI(&settings, dbSettings, db, fa, fec, nil, fd, dbmodel.NewDHCPOptionDefinitionLookup())
	require.NoError(t, err)
	ctx := context.Background()

	// Create DHCPv6 server in the database.
	dhcp6, err := dbmodeltest.NewKeaDHCPv6Server(db)
	require.NoError(t, err)

	// Apply the configuration including all configuration keys.
	err = dhcp6.Configure(string(testutil.AllKeysDHCPv6JSON))
	require.NoError(t, err)

	app, err := dhcp6.GetKea()
	require.NoError(t, err)

	// Populate subnets and shared networks from the Kea configuration.
	err = kea.CommitAppIntoDB(db, app, &storktest.FakeEventCenter{}, nil, dbmodel.NewDHCPOptionDefinitionLookup())
	require.NoError(t, err)

	sharedNetworks, err := dbmodel.GetAllSharedNetworks(db, 6)
	require.NoError(t, err)
	require.Len(t, sharedNetworks, 1)
	require.Len(t, sharedNetworks[0].LocalSharedNetworks, 1)

	// Get the shared network over the REST API.
	params := dhcp.GetSharedNetworkParams{
		ID: sharedNetworks[0].ID,
	}
	rsp := rapi.GetSharedNetwork(ctx, params)
	require.IsType(t, &dhcp.GetSharedNetworkOK{}, rsp)
	okRsp := rsp.(*dhcp.GetSharedNetworkOK)
	sharedNetwork := okRsp.Payload
	require.NotNil(t, sharedNetwork)

	require.Len(t, sharedNetwork.LocalSharedNetworks, 1)
	ls := sharedNetwork.LocalSharedNetworks[0]

	// Validate shared-network-level parameters
	require.NotNil(t, ls.KeaConfigSharedNetworkParameters)
	networkParams := ls.KeaConfigSharedNetworkParameters.SharedNetworkLevelParameters
	require.NotNil(t, networkParams)

	// pd-allocator
	require.NotNil(t, networkParams.PdAllocator)
	require.Equal(t, "iterative", *networkParams.PdAllocator)
	// preferred-lifetime
	require.NotNil(t, networkParams.PreferredLifetime)
	require.EqualValues(t, 2000, *networkParams.PreferredLifetime)
	// min-preferred-lifetime
	require.NotNil(t, networkParams.MinPreferredLifetime)
	require.EqualValues(t, 1500, *networkParams.MinPreferredLifetime)
	// max-preferred-lifetime
	require.NotNil(t, networkParams.MaxPreferredLifetime)
	require.EqualValues(t, 2500, *networkParams.MaxPreferredLifetime)

	// Validate the options.
	require.NotEmpty(t, networkParams.OptionsHash)
	require.Len(t, networkParams.Options, 1)
	require.False(t, networkParams.Options[0].AlwaysSend)
	require.EqualValues(t, 7, networkParams.Options[0].Code)
	require.Empty(t, networkParams.Options[0].Encapsulate)
	require.Len(t, networkParams.Options[0].Fields, 1)
	require.Equal(t, dhcpmodel.BinaryField, networkParams.Options[0].Fields[0].FieldType)
	require.Len(t, networkParams.Options[0].Fields[0].Values, 1)
	require.Equal(t, "ab", networkParams.Options[0].Fields[0].Values[0])
	require.EqualValues(t, storkutil.IPv6, networkParams.Options[0].Universe)

	// Validate global parameters.
	globalParams := ls.KeaConfigSharedNetworkParameters.GlobalParameters
	require.NotNil(t, globalParams)

	require.NotNil(t, globalParams.PdAllocator)
	require.Equal(t, "random", *globalParams.PdAllocator)
	// preferred-lifetime
	require.NotNil(t, globalParams.PreferredLifetime)
	require.EqualValues(t, 50, *globalParams.PreferredLifetime)
	// min-preferred-lifetime
	require.NotNil(t, globalParams.MinPreferredLifetime)
	require.EqualValues(t, 40, *globalParams.MinPreferredLifetime)
	// max-preferred-lifetime
	require.NotNil(t, globalParams.MaxPreferredLifetime)
	require.EqualValues(t, 60, *globalParams.MaxPreferredLifetime)

	// Validate the options.
	require.NotEmpty(t, globalParams.OptionsHash)
	require.Len(t, globalParams.Options, 1)
	require.False(t, globalParams.Options[0].AlwaysSend)
	require.EqualValues(t, 23, globalParams.Options[0].Code)
	require.Empty(t, globalParams.Options[0].Encapsulate)
	require.Len(t, globalParams.Options[0].Fields, 2)
	require.Equal(t, dhcpmodel.IPv6AddressField, globalParams.Options[0].Fields[0].FieldType)
	require.Equal(t, dhcpmodel.IPv6AddressField, globalParams.Options[0].Fields[1].FieldType)
	require.Len(t, globalParams.Options[0].Fields[0].Values, 1)
	require.Equal(t, "2001:db8:2::45", globalParams.Options[0].Fields[0].Values[0])
	require.Len(t, globalParams.Options[0].Fields[1].Values, 1)
	require.Equal(t, "2001:db8:2::100", globalParams.Options[0].Fields[1].Values[0])
	require.EqualValues(t, storkutil.IPv6, globalParams.Options[0].Universe)
}

// Test getting an IPv6 shared network over the REST API when the DHCP configuration
// contains no explicit parameters.
func TestGetSharedNetwork6MinimalParameters(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	settings := RestAPISettings{}
	fa := agentcommtest.NewFakeAgents(nil, nil)
	fec := &storktest.FakeEventCenter{}
	fd := &storktest.FakeDispatcher{}
	rapi, err := NewRestAPI(&settings, dbSettings, db, fa, fec, nil, fd, nil)
	require.NoError(t, err)
	ctx := context.Background()

	// Create DHCPv6 server in the database.
	dhcp6, err := dbmodeltest.NewKeaDHCPv6Server(db)
	require.NoError(t, err)

	// Apply a minimal configuration.
	cfg := `{
		"Dhcp6": {
			"shared-networks": [
				{
					"name": "foo",
					"subnet6": [
						{
							"subnet": "3000::/64"
						}
					]
				}
			]
		}
	}`
	err = dhcp6.Configure(cfg)
	require.NoError(t, err)

	app, err := dhcp6.GetKea()
	require.NoError(t, err)

	// Populates the subnets in the database.
	err = kea.CommitAppIntoDB(db, app, &storktest.FakeEventCenter{}, nil, dbmodel.NewDHCPOptionDefinitionLookup())
	require.NoError(t, err)

	sharedNetworks, err := dbmodel.GetAllSharedNetworks(db, 6)
	require.NoError(t, err)
	require.Len(t, sharedNetworks, 1)

	// Get the subnet over the REST API.
	params := dhcp.GetSharedNetworkParams{
		ID: sharedNetworks[0].ID,
	}
	rsp := rapi.GetSharedNetwork(ctx, params)
	require.IsType(t, &dhcp.GetSharedNetworkOK{}, rsp)
	okRsp := rsp.(*dhcp.GetSharedNetworkOK)
	sharedNetwork := okRsp.Payload
	require.NotNil(t, sharedNetwork)

	require.Len(t, sharedNetwork.LocalSharedNetworks, 1)
	ls := sharedNetwork.LocalSharedNetworks[0]

	require.NotNil(t, ls.KeaConfigSharedNetworkParameters)
}

// Test the calls for creating new transaction and adding a shared network.
func TestCreateSharedNetwork4BeginSubmit(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	serverConfig := `{
		"Dhcp4": {
			"client-classes": [
				{
					"name": "devices"
				},
				{
					"name": "printers"
				}
			],
			"shared-networks": [
				{
					"name": "foo",
					"subnet4": [
						{
							"id": 1,
							"subnet": "192.0.2.0/24"
						}
					]
				}
			],
			"hooks-libraries": [
				{
					"library": "libdhcp_subnet_cmds"
				}
			]
		}
	}`

	server1, err := dbmodeltest.NewKeaDHCPv4Server(db)
	require.NoError(t, err)
	err = server1.Configure(serverConfig)
	require.NoError(t, err)

	app, err := server1.GetKea()
	require.NoError(t, err)

	err = kea.CommitAppIntoDB(db, app, &storktest.FakeEventCenter{}, nil, dbmodel.NewDHCPOptionDefinitionLookup())
	require.NoError(t, err)

	server2, err := dbmodeltest.NewKeaDHCPv4Server(db)
	require.NoError(t, err)
	err = server2.Configure(serverConfig)
	require.NoError(t, err)

	app, err = server2.GetKea()
	require.NoError(t, err)

	err = kea.CommitAppIntoDB(db, app, &storktest.FakeEventCenter{}, nil, dbmodel.NewDHCPOptionDefinitionLookup())
	require.NoError(t, err)

	dbapps, err := dbmodel.GetAllApps(db, true)
	require.NoError(t, err)
	require.Len(t, dbapps, 2)

	sharedNetworks, err := dbmodel.GetAllSharedNetworks(db, 0)
	require.NoError(t, err)
	require.Len(t, sharedNetworks, 1)

	// Create fake agents receiving commands.
	fa := agentcommtest.NewFakeAgents(nil, nil)
	require.NotNil(t, fa)

	lookup := dbmodel.NewDHCPOptionDefinitionLookup()
	require.NotNil(t, lookup)

	// Create the config manager.
	cm := apps.NewManager(&appstest.ManagerAccessorsWrapper{
		DB:        db,
		Agents:    fa,
		DefLookup: lookup,
	})
	require.NotNil(t, cm)

	// Create API.
	rapi, err := NewRestAPI(dbSettings, db, fa, cm, lookup)
	require.NoError(t, err)

	// Create session manager.
	ctx, err := rapi.SessionManager.Load(context.Background(), "")
	require.NoError(t, err)

	// Create user session.
	user := &dbmodel.SystemUser{
		ID: 1234,
	}
	err = rapi.SessionManager.LoginHandler(ctx, user)
	require.NoError(t, err)

	// Begin transaction.
	params := dhcp.CreateSharedNetworkBeginParams{}
	rsp := rapi.CreateSharedNetworkBegin(ctx, params)
	require.IsType(t, &dhcp.CreateSharedNetworkBeginOK{}, rsp)
	okRsp := rsp.(*dhcp.CreateSharedNetworkBeginOK)
	contents := okRsp.Payload

	// Make sure the server returned transaction ID, daemons, shared networks and client classes.
	transactionID := contents.ID
	require.NotZero(t, transactionID)
	require.Len(t, contents.Daemons, 2)
	require.Len(t, contents.SharedNetworks4, 1)
	require.Equal(t, "foo", contents.SharedNetworks4[0])
	require.Empty(t, contents.SharedNetworks6)
	require.Len(t, contents.ClientClasses, 2)

	keaConfigSharedNetworkParameters := &models.KeaConfigSharedNetworkParameters{
		SharedNetworkLevelParameters: &models.KeaConfigSubnetDerivedParameters{
			KeaConfigCacheParameters: models.KeaConfigCacheParameters{
				CacheMaxAge:    storkutil.Ptr[int64](1000),
				CacheThreshold: storkutil.Ptr[float32](0.25),
			},
			KeaConfigClientClassParameters: models.KeaConfigClientClassParameters{
				ClientClass:          storkutil.Ptr("foo"),
				RequireClientClasses: []string{"bar"},
			},
			KeaConfigDdnsParameters: models.KeaConfigDdnsParameters{
				DdnsGeneratedPrefix:        storkutil.Ptr("abc"),
				DdnsOverrideClientUpdate:   storkutil.Ptr(true),
				DdnsOverrideNoUpdate:       storkutil.Ptr(false),
				DdnsQualifyingSuffix:       storkutil.Ptr("example.org"),
				DdnsReplaceClientName:      storkutil.Ptr("never"),
				DdnsSendUpdates:            storkutil.Ptr(true),
				DdnsUpdateOnRenew:          storkutil.Ptr(true),
				DdnsUseConflictResolution:  storkutil.Ptr(true),
				DdnsConflictResolutionMode: storkutil.Ptr("check-with-dhcid"),
				DdnsTTLPercent:             storkutil.Ptr(float32(0.65)),
			},
			KeaConfigFourOverSixParameters: models.KeaConfigFourOverSixParameters{
				FourOverSixInterface:   storkutil.Ptr("eth0"),
				FourOverSixInterfaceID: storkutil.Ptr("ifaceid"),
				FourOverSixSubnet:      storkutil.Ptr("2001:db8:1::/64"),
			},
			KeaConfigHostnameCharParameters: models.KeaConfigHostnameCharParameters{
				HostnameCharReplacement: storkutil.Ptr("a"),
				HostnameCharSet:         storkutil.Ptr("b"),
			},
			KeaConfigReservationParameters: models.KeaConfigReservationParameters{
				ReservationMode:       storkutil.Ptr("in-pool"),
				ReservationsGlobal:    storkutil.Ptr(false),
				ReservationsInSubnet:  storkutil.Ptr(true),
				ReservationsOutOfPool: storkutil.Ptr(false),
			},
			KeaConfigTimerParameters: models.KeaConfigTimerParameters{
				CalculateTeeTimes: storkutil.Ptr(true),
				RebindTimer:       storkutil.Ptr[int64](2000),
				RenewTimer:        storkutil.Ptr[int64](1000),
				T1Percent:         storkutil.Ptr[float32](0.25),
				T2Percent:         storkutil.Ptr[float32](0.50),
			},
			KeaConfigValidLifetimeParameters: models.KeaConfigValidLifetimeParameters{
				MaxValidLifetime: storkutil.Ptr[int64](5000),
				MinValidLifetime: storkutil.Ptr[int64](4000),
				ValidLifetime:    storkutil.Ptr[int64](4500),
			},
			KeaConfigAssortedSubnetParameters: models.KeaConfigAssortedSubnetParameters{
				Allocator:     storkutil.Ptr("random"),
				Authoritative: storkutil.Ptr(true),
				BootFileName:  storkutil.Ptr("/tmp/filename"),
				Interface:     storkutil.Ptr("eth0"),
				MatchClientID: storkutil.Ptr(true),
				NextServer:    storkutil.Ptr("192.0.2.1"),
				Relay: &models.KeaConfigAssortedSubnetParametersRelay{
					IPAddresses: []string{"10.1.1.1"},
				},
				ServerHostname:    storkutil.Ptr("myhost.example.org"),
				StoreExtendedInfo: storkutil.Ptr(true),
			},
			DHCPOptions: models.DHCPOptions{
				Options: []*models.DHCPOption{
					{
						AlwaysSend: true,
						Code:       3,
						Fields: []*models.DHCPOptionField{
							{
								FieldType: "ipv4-address",
								Values:    []string{"192.0.2.1"},
							},
						},
						Universe: 4,
					},
				},
			},
		},
	}

	// Submit transaction.
	params2 := dhcp.CreateSharedNetworkSubmitParams{
		ID: transactionID,
		SharedNetwork: &models.SharedNetwork{
			Name:     "bar",
			Universe: int64(4),
			Subnets:  []*models.Subnet{},
			LocalSharedNetworks: []*models.LocalSharedNetwork{
				{
					DaemonID:                         dbapps[0].Daemons[0].ID,
					KeaConfigSharedNetworkParameters: keaConfigSharedNetworkParameters,
				},
				{
					DaemonID:                         dbapps[1].Daemons[0].ID,
					KeaConfigSharedNetworkParameters: keaConfigSharedNetworkParameters,
				},
			},
		},
	}
	rsp2 := rapi.CreateSharedNetworkSubmit(ctx, params2)
	require.IsType(t, &dhcp.CreateSharedNetworkSubmitOK{}, rsp2)
	require.IsType(t, &dhcp.CreateSharedNetworkSubmitOK{}, rsp2)
	okRsp2 := rsp2.(*dhcp.CreateSharedNetworkSubmitOK)

	// Appropriate commands should be sent to two Kea servers.
	require.Len(t, fa.RecordedCommands, 4)

	for i, c := range fa.RecordedCommands {
		switch i {
		case 0, 1:
			require.JSONEq(t, `
				{
					"command": "network4-add",
					"service": ["dhcp4"],
					"arguments": {
						"shared-networks": [
							{
								"cache-threshold": 0.25,
								"cache-max-age": 1000,
								"client-class": "foo",
								"require-client-classes": ["bar"],
								"ddns-generated-prefix": "abc",
								"ddns-override-client-update": true,
								"ddns-override-no-update": false,
								"ddns-qualifying-suffix": "example.org",
								"ddns-replace-client-name": "never",
								"ddns-send-updates": true,
								"ddns-update-on-renew": true,
								"ddns-use-conflict-resolution": true,
								"ddns-conflict-resolution-mode": "check-with-dhcid",
								"ddns-ttl-percent": 0.65,
								"hostname-char-replacement": "a",
								"hostname-char-set": "b",
								"reservation-mode": "in-pool",
								"reservations-global": false,
								"reservations-in-subnet": true,
								"reservations-out-of-pool": false,
								"renew-timer": 1000,
								"rebind-timer": 2000,
								"t1-percent": 0.25,
								"t2-percent": 0.5,
								"calculate-tee-times": true,
								"valid-lifetime": 4500,
								"min-valid-lifetime": 4000,
								"max-valid-lifetime": 5000,
								"allocator": "random",
								"interface": "eth0",
								"store-extended-info": true,
								"option-data": [
									{
										"always-send": true,
										"code": 3,
										"csv-format": true,
										"data": "192.0.2.1",
										"space": "dhcp4"
									}
								],
								"relay": {
									"ip-addresses":["10.1.1.1"]
								},
								"authoritative": true,
								"boot-file-name": "/tmp/filename",
								"match-client-id": true,
								"name": "bar",
								"next-server": "192.0.2.1",
								"server-hostname": "myhost.example.org"
							}
						]
					}
				}
			`, c.Marshal())
		default:
			require.JSONEq(t, `
				{
					"command": "config-write",
					"service": ["dhcp4"]
				}
			`, c.Marshal())
		}
	}

	// Make sure that the transaction is done.
	cctx, _ := cm.RecoverContext(transactionID, int64(user.ID))
	// Remove the context from the config manager before testing that
	// the returned context is nil. If it happens to be non-nil the
	// require.Nil() would otherwise spit out errors about the concurrent
	// access to the context in the manager's goroutine and here.
	if cctx != nil {
		cm.Done(cctx)
	}
	require.Nil(t, cctx)

	// Make sure that the shared network has been stored in the database.
	returnedSharedNetwork, err := dbmodel.GetSharedNetwork(db, okRsp2.Payload.SharedNetworkID)
	require.NoError(t, err)
	require.NotNil(t, returnedSharedNetwork)
	require.Empty(t, returnedSharedNetwork.Subnets)

	require.Len(t, returnedSharedNetwork.LocalSharedNetworks, 2)
	for _, lsn := range returnedSharedNetwork.LocalSharedNetworks {
		require.NotNil(t, lsn.KeaParameters)
		require.NotNil(t, lsn.KeaParameters.CacheMaxAge)
		require.EqualValues(t, 1000, *lsn.KeaParameters.CacheMaxAge)
		require.NotNil(t, lsn.KeaParameters.CacheThreshold)
		require.EqualValues(t, 0.25, *lsn.KeaParameters.CacheThreshold)
		require.NotNil(t, lsn.KeaParameters.ClientClass)
		require.Equal(t, "foo", *lsn.KeaParameters.ClientClass)
		require.Len(t, lsn.KeaParameters.RequireClientClasses, 1)
		require.EqualValues(t, "bar", lsn.KeaParameters.RequireClientClasses[0])
		require.NotNil(t, lsn.KeaParameters.DDNSGeneratedPrefix)
		require.Equal(t, "abc", *lsn.KeaParameters.DDNSGeneratedPrefix)
		require.NotNil(t, lsn.KeaParameters.DDNSOverrideClientUpdate)
		require.True(t, *lsn.KeaParameters.DDNSOverrideClientUpdate)
		require.NotNil(t, lsn.KeaParameters.DDNSOverrideNoUpdate)
		require.False(t, *lsn.KeaParameters.DDNSOverrideNoUpdate)
		require.NotNil(t, lsn.KeaParameters.DDNSQualifyingSuffix)
		require.Equal(t, "example.org", *lsn.KeaParameters.DDNSQualifyingSuffix)
		require.NotNil(t, lsn.KeaParameters.DDNSReplaceClientName)
		require.Equal(t, "never", *lsn.KeaParameters.DDNSReplaceClientName)
		require.NotNil(t, lsn.KeaParameters.DDNSSendUpdates)
		require.True(t, *lsn.KeaParameters.DDNSSendUpdates)
		require.NotNil(t, lsn.KeaParameters.DDNSUpdateOnRenew)
		require.True(t, *lsn.KeaParameters.DDNSUpdateOnRenew)
		require.NotNil(t, lsn.KeaParameters.DDNSUseConflictResolution)
		require.True(t, *lsn.KeaParameters.DDNSUseConflictResolution)
		require.NotNil(t, *lsn.KeaParameters.DDNSTTLPercent)
		require.EqualValues(t, float32(0.65), *lsn.KeaParameters.DDNSTTLPercent)
		require.NotNil(t, lsn.KeaParameters.HostnameCharReplacement)
		require.Equal(t, "a", *lsn.KeaParameters.HostnameCharReplacement)
		require.NotNil(t, lsn.KeaParameters.HostnameCharSet)
		require.Equal(t, "b", *lsn.KeaParameters.HostnameCharSet)
		require.NotNil(t, lsn.KeaParameters.ReservationMode)
		require.Equal(t, "in-pool", *lsn.KeaParameters.ReservationMode)
		require.NotNil(t, lsn.KeaParameters.ReservationsGlobal)
		require.False(t, *lsn.KeaParameters.ReservationsGlobal)
		require.NotNil(t, lsn.KeaParameters.ReservationsInSubnet)
		require.True(t, *lsn.KeaParameters.ReservationsInSubnet)
		require.NotNil(t, lsn.KeaParameters.ReservationsOutOfPool)
		require.False(t, *lsn.KeaParameters.ReservationsOutOfPool)
		require.NotNil(t, lsn.KeaParameters.CalculateTeeTimes)
		require.True(t, *lsn.KeaParameters.CalculateTeeTimes)
		require.NotNil(t, lsn.KeaParameters.RebindTimer)
		require.EqualValues(t, 2000, *lsn.KeaParameters.RebindTimer)
		require.NotNil(t, lsn.KeaParameters.RenewTimer)
		require.EqualValues(t, 1000, *lsn.KeaParameters.RenewTimer)
		require.NotNil(t, lsn.KeaParameters.T1Percent)
		require.EqualValues(t, 0.25, *lsn.KeaParameters.T1Percent)
		require.NotNil(t, lsn.KeaParameters.T2Percent)
		require.EqualValues(t, 0.50, *lsn.KeaParameters.T2Percent)
		require.NotNil(t, lsn.KeaParameters.MaxValidLifetime)
		require.EqualValues(t, 5000, *lsn.KeaParameters.MaxValidLifetime)
		require.NotNil(t, lsn.KeaParameters.MinValidLifetime)
		require.EqualValues(t, 4000, *lsn.KeaParameters.MinValidLifetime)
		require.NotNil(t, lsn.KeaParameters.ValidLifetime)
		require.EqualValues(t, 4500, *lsn.KeaParameters.ValidLifetime)
		require.NotNil(t, lsn.KeaParameters.Allocator)
		require.Equal(t, "random", *lsn.KeaParameters.Allocator)
		require.NotNil(t, lsn.KeaParameters.Authoritative)
		require.True(t, *lsn.KeaParameters.Authoritative)
		require.NotNil(t, lsn.KeaParameters.Authoritative)
		require.Equal(t, "/tmp/filename", *lsn.KeaParameters.BootFileName)
		require.NotNil(t, lsn.KeaParameters.Interface)
		require.Equal(t, "eth0", *lsn.KeaParameters.Interface)
		require.NotNil(t, lsn.KeaParameters.MatchClientID)
		require.True(t, *lsn.KeaParameters.MatchClientID)
		require.NotNil(t, lsn.KeaParameters.NextServer)
		require.Equal(t, "192.0.2.1", *lsn.KeaParameters.NextServer)
		require.NotNil(t, lsn.KeaParameters.Relay)
		require.Len(t, lsn.KeaParameters.Relay.IPAddresses, 1)
		require.Equal(t, "10.1.1.1", lsn.KeaParameters.Relay.IPAddresses[0])
		require.NotNil(t, lsn.KeaParameters.ServerHostname)
		require.Equal(t, "myhost.example.org", *lsn.KeaParameters.ServerHostname)
		require.NotNil(t, lsn.KeaParameters.StoreExtendedInfo)
		require.True(t, *lsn.KeaParameters.StoreExtendedInfo)

		// DHCP options
		require.Len(t, lsn.DHCPOptionSet.Options, 1)
		require.True(t, lsn.DHCPOptionSet.Options[0].AlwaysSend)
		require.EqualValues(t, 3, lsn.DHCPOptionSet.Options[0].Code)
		require.Len(t, lsn.DHCPOptionSet.Options[0].Fields, 1)
		require.Equal(t, dhcpmodel.IPv4AddressField, lsn.DHCPOptionSet.Options[0].Fields[0].FieldType)
		require.Len(t, lsn.DHCPOptionSet.Options[0].Fields[0].Values, 1)
		require.Equal(t, "192.0.2.1", lsn.DHCPOptionSet.Options[0].Fields[0].Values[0])
		require.Equal(t, dhcpmodel.DHCPv4OptionSpace, lsn.DHCPOptionSet.Options[0].Space)
		require.NotEmpty(t, lsn.DHCPOptionSet.Hash)
	}
}

// Test error cases for submitting new shared network.
func TestCreateSharedNetwork4BeginSubmitError(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	serverConfig := `{
		"Dhcp4": {
			"client-classes": [
				{
					"name": "devices"
				},
				{
					"name": "printers"
				}
			],
			"shared-networks": [
				{
					"name": "foo",
					"subnet4": [
						{
							"id": 1,
							"subnet": "192.0.2.0/24"
						}
					]
				}
			],
			"hooks-libraries": [
				{
					"library": "libdhcp_subnet_cmds"
				}
			]
		}
	}`

	server1, err := dbmodeltest.NewKeaDHCPv4Server(db)
	require.NoError(t, err)
	err = server1.Configure(serverConfig)
	require.NoError(t, err)

	app1, err := server1.GetKea()
	require.NoError(t, err)

	err = kea.CommitAppIntoDB(db, app1, &storktest.FakeEventCenter{}, nil, dbmodel.NewDHCPOptionDefinitionLookup())
	require.NoError(t, err)

	server2, err := dbmodeltest.NewKeaDHCPv4Server(db)
	require.NoError(t, err)
	err = server2.Configure(serverConfig)
	require.NoError(t, err)

	app2, err := server2.GetKea()
	require.NoError(t, err)

	err = kea.CommitAppIntoDB(db, app2, &storktest.FakeEventCenter{}, nil, dbmodel.NewDHCPOptionDefinitionLookup())
	require.NoError(t, err)

	dbapps, err := dbmodel.GetAllApps(db, true)
	require.NoError(t, err)
	require.Len(t, dbapps, 2)

	sharedNetworks, err := dbmodel.GetAllSharedNetworks(db, 0)
	require.NoError(t, err)
	require.Len(t, sharedNetworks, 1)

	// Create fake agents receiving commands.
	fa := agentcommtest.NewFakeAgents(func(callNo int, cmdResponses []interface{}) {
		mockStatusError("network4-add", cmdResponses)
	}, nil)
	require.NotNil(t, fa)

	lookup := dbmodel.NewDHCPOptionDefinitionLookup()
	require.NotNil(t, lookup)

	// Create the config manager.
	cm := apps.NewManager(&appstest.ManagerAccessorsWrapper{
		DB:        db,
		Agents:    fa,
		DefLookup: lookup,
	})
	require.NotNil(t, cm)

	// Create API.
	rapi, err := NewRestAPI(dbSettings, db, fa, cm, lookup)
	require.NoError(t, err)

	// Create session manager.
	ctx, err := rapi.SessionManager.Load(context.Background(), "")
	require.NoError(t, err)

	// Create user session.
	user := &dbmodel.SystemUser{
		ID: 1234,
	}
	err = rapi.SessionManager.LoginHandler(ctx, user)
	require.NoError(t, err)

	// Begin transaction.
	params := dhcp.CreateSharedNetworkBeginParams{}
	rsp := rapi.CreateSharedNetworkBegin(ctx, params)
	require.IsType(t, &dhcp.CreateSharedNetworkBeginOK{}, rsp)
	okRsp := rsp.(*dhcp.CreateSharedNetworkBeginOK)
	contents := okRsp.Payload

	// Make sure the server returned transaction ID, daemons, shared networks and client classes.
	transactionID := contents.ID
	require.NotZero(t, transactionID)
	require.Len(t, contents.Daemons, 2)
	require.Len(t, contents.SharedNetworks4, 1)
	require.Equal(t, "foo", contents.SharedNetworks4[0])
	require.Empty(t, contents.SharedNetworks6)
	require.Len(t, contents.ClientClasses, 2)

	// Submit transaction without the shared network information.
	t.Run("no shared network", func(t *testing.T) {
		params := dhcp.CreateSharedNetworkSubmitParams{
			ID:            transactionID,
			SharedNetwork: nil,
		}
		rsp := rapi.CreateSharedNetworkSubmit(ctx, params)
		require.IsType(t, &dhcp.CreateSharedNetworkSubmitDefault{}, rsp)
		defaultRsp := rsp.(*dhcp.CreateSharedNetworkSubmitDefault)
		require.Equal(t, http.StatusBadRequest, getStatusCode(*defaultRsp))
		require.Equal(t, "Shared network information not specified", *defaultRsp.Payload.Message)
	})

	// Submit transaction with non-matching transaction ID.
	t.Run("wrong transaction id", func(t *testing.T) {
		params := dhcp.CreateSharedNetworkSubmitParams{
			ID: transactionID + 1,
			SharedNetwork: &models.SharedNetwork{
				ID:   0,
				Name: "foo",
				LocalSharedNetworks: []*models.LocalSharedNetwork{
					{
						DaemonID: dbapps[0].Daemons[0].ID,
					},
					{
						DaemonID: dbapps[1].Daemons[0].ID,
					},
				},
			},
		}
		rsp := rapi.CreateSharedNetworkSubmit(ctx, params)
		require.IsType(t, &dhcp.CreateSharedNetworkSubmitDefault{}, rsp)
		defaultRsp := rsp.(*dhcp.CreateSharedNetworkSubmitDefault)
		require.Equal(t, http.StatusNotFound, getStatusCode(*defaultRsp))
		require.Equal(t, "Transaction expired for the shared network update", *defaultRsp.Payload.Message)
	})

	// Submit transaction with a shared network that is not associated with
	// any daemons. It simulates a failure in "apply" step which typically
	// is caused by some internal server problem rather than malformed request.
	t.Run("no daemons in shared network", func(t *testing.T) {
		params := dhcp.CreateSharedNetworkSubmitParams{
			ID: transactionID,
			SharedNetwork: &models.SharedNetwork{
				ID:                  0,
				Name:                "foo",
				LocalSharedNetworks: []*models.LocalSharedNetwork{},
			},
		}
		rsp := rapi.CreateSharedNetworkSubmit(ctx, params)
		require.IsType(t, &dhcp.CreateSharedNetworkSubmitDefault{}, rsp)
		defaultRsp := rsp.(*dhcp.CreateSharedNetworkSubmitDefault)
		require.Equal(t, http.StatusInternalServerError, getStatusCode(*defaultRsp))
		require.Equal(t, "Problem with applying shared network information: applied shared network foo is not associated with any daemon", *defaultRsp.Payload.Message)
	})

	// Submit transaction with valid ID and shared network but expect the
	// agent to return an error code. This is considered a conflict with
	// the state of the Kea servers.
	t.Run("commit failure", func(t *testing.T) {
		params := dhcp.CreateSharedNetworkSubmitParams{
			ID: transactionID,
			SharedNetwork: &models.SharedNetwork{
				ID:       0,
				Name:     "foo",
				Universe: 4,
				LocalSharedNetworks: []*models.LocalSharedNetwork{
					{
						DaemonID: dbapps[0].Daemons[0].ID,
					},
					{
						DaemonID: dbapps[1].Daemons[0].ID,
					},
				},
			},
		}
		rsp := rapi.CreateSharedNetworkSubmit(ctx, params)
		require.IsType(t, &dhcp.CreateSharedNetworkSubmitDefault{}, rsp)
		defaultRsp := rsp.(*dhcp.CreateSharedNetworkSubmitDefault)
		require.Equal(t, http.StatusConflict, getStatusCode(*defaultRsp))
		require.Equal(t, fmt.Sprintf("Problem with committing shared network information: network4-add command to %s failed: error status (1) returned by Kea dhcp4 daemon with text: 'unable to communicate with the daemon'", app1.GetName()), *defaultRsp.Payload.Message)
	})
}

// Test that the transaction to update a shared network can be canceled,
// resulting in the removal of this transaction from the config manager
// and allowing another user to apply config updates.
func TestCreateSharedNetworkBeginCancel(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	serverConfig := `{
		"Dhcp4": {
			"client-classes": [
				{
					"name": "devices"
				},
				{
					"name": "printers"
				}
			],
			"shared-networks": [
				{
					"name": "foo",
					"subnet4": [
						{
							"id": 1,
							"subnet": "192.0.2.0/24"
						}
					]
				}
			],
			"hooks-libraries": [
				{
					"library": "libdhcp_subnet_cmds"
				}
			]
		}
	}`

	server1, err := dbmodeltest.NewKeaDHCPv4Server(db)
	require.NoError(t, err)
	err = server1.Configure(serverConfig)
	require.NoError(t, err)

	app, err := server1.GetKea()
	require.NoError(t, err)

	err = kea.CommitAppIntoDB(db, app, &storktest.FakeEventCenter{}, nil, dbmodel.NewDHCPOptionDefinitionLookup())
	require.NoError(t, err)

	server2, err := dbmodeltest.NewKeaDHCPv4Server(db)
	require.NoError(t, err)
	err = server2.Configure(serverConfig)
	require.NoError(t, err)

	app, err = server2.GetKea()
	require.NoError(t, err)

	err = kea.CommitAppIntoDB(db, app, &storktest.FakeEventCenter{}, nil, dbmodel.NewDHCPOptionDefinitionLookup())
	require.NoError(t, err)

	dbapps, err := dbmodel.GetAllApps(db, true)
	require.NoError(t, err)
	require.Len(t, dbapps, 2)

	sharedNetworks, err := dbmodel.GetAllSharedNetworks(db, 0)
	require.NoError(t, err)
	require.Len(t, sharedNetworks, 1)

	// Create fake agents receiving commands.
	fa := agentcommtest.NewFakeAgents(nil, nil)
	require.NotNil(t, fa)

	lookup := dbmodel.NewDHCPOptionDefinitionLookup()
	require.NotNil(t, lookup)

	// Create the config manager.
	cm := apps.NewManager(&appstest.ManagerAccessorsWrapper{
		DB:        db,
		Agents:    fa,
		DefLookup: lookup,
	})
	require.NotNil(t, cm)

	// Create API.
	rapi, err := NewRestAPI(dbSettings, db, fa, cm, lookup)
	require.NoError(t, err)

	// Create session manager.
	ctx, err := rapi.SessionManager.Load(context.Background(), "")
	require.NoError(t, err)

	// Create user session.
	user := &dbmodel.SystemUser{
		ID: 1234,
	}
	err = rapi.SessionManager.LoginHandler(ctx, user)
	require.NoError(t, err)

	// Begin transaction.
	params := dhcp.CreateSharedNetworkBeginParams{}
	rsp := rapi.CreateSharedNetworkBegin(ctx, params)
	require.IsType(t, &dhcp.CreateSharedNetworkBeginOK{}, rsp)
	okRsp := rsp.(*dhcp.CreateSharedNetworkBeginOK)
	contents := okRsp.Payload

	// Make sure the server returned transaction ID.
	transactionID := contents.ID
	require.NotZero(t, transactionID)

	// Cancel the transaction.
	params2 := dhcp.CreateSharedNetworkDeleteParams{
		ID: transactionID,
	}
	rsp2 := rapi.CreateSharedNetworkDelete(ctx, params2)
	require.IsType(t, &dhcp.CreateSharedNetworkDeleteOK{}, rsp2)

	cctx, _ := cm.RecoverContext(transactionID, int64(user.ID))
	// Remove the context from the config manager before testing that
	// the returned context is nil. If it happens to be non-nil the
	// require.Nil() would otherwise spit out errors about the concurrent
	// access to the context in the manager's goroutine and here.
	if cctx != nil {
		cm.Done(cctx)
	}
	require.Nil(t, cctx)
}

// Test the calls for creating new transaction and updating a shared network.
func TestUpdateSharedNetwork4BeginSubmit(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	serverConfig := `{
		"Dhcp4": {
			"client-classes": [
				{
					"name": "devices"
				},
				{
					"name": "printers"
				}
			],
			"shared-networks": [
				{
					"name": "foo",
					"subnet4": [
						{
							"id": 1,
							"subnet": "192.0.2.0/24"
						}
					],
					"option-data": [
						{
							"always-send": true,
							"code": 3,
							"csv-format": true,
							"data": "192.0.2.1",
							"space": "dhcp4"
						}
					]
				}
			],
			"hooks-libraries": [
				{
					"library": "libdhcp_subnet_cmds"
				}
			]
		}
	}`

	server1, err := dbmodeltest.NewKeaDHCPv4Server(db)
	require.NoError(t, err)
	err = server1.Configure(serverConfig)
	require.NoError(t, err)

	app, err := server1.GetKea()
	require.NoError(t, err)

	err = kea.CommitAppIntoDB(db, app, &storktest.FakeEventCenter{}, nil, dbmodel.NewDHCPOptionDefinitionLookup())
	require.NoError(t, err)

	server2, err := dbmodeltest.NewKeaDHCPv4Server(db)
	require.NoError(t, err)
	err = server2.Configure(serverConfig)
	require.NoError(t, err)

	app, err = server2.GetKea()
	require.NoError(t, err)

	err = kea.CommitAppIntoDB(db, app, &storktest.FakeEventCenter{}, nil, dbmodel.NewDHCPOptionDefinitionLookup())
	require.NoError(t, err)

	dbapps, err := dbmodel.GetAllApps(db, true)
	require.NoError(t, err)
	require.Len(t, dbapps, 2)

	sharedNetworks, err := dbmodel.GetAllSharedNetworks(db, 0)
	require.NoError(t, err)
	require.Len(t, sharedNetworks, 1)

	// Create fake agents receiving commands.
	fa := agentcommtest.NewFakeAgents(nil, nil)
	require.NotNil(t, fa)

	lookup := dbmodel.NewDHCPOptionDefinitionLookup()
	require.NotNil(t, lookup)

	// Create the config manager.
	cm := apps.NewManager(&appstest.ManagerAccessorsWrapper{
		DB:        db,
		Agents:    fa,
		DefLookup: lookup,
	})
	require.NotNil(t, cm)

	// Create API.
	rapi, err := NewRestAPI(dbSettings, db, fa, cm, lookup)
	require.NoError(t, err)

	// Create session manager.
	ctx, err := rapi.SessionManager.Load(context.Background(), "")
	require.NoError(t, err)

	// Create user session.
	user := &dbmodel.SystemUser{
		ID: 1234,
	}
	err = rapi.SessionManager.LoginHandler(ctx, user)
	require.NoError(t, err)

	// Begin transaction.
	params := dhcp.UpdateSharedNetworkBeginParams{
		SharedNetworkID: sharedNetworks[0].ID,
	}
	rsp := rapi.UpdateSharedNetworkBegin(ctx, params)
	require.IsType(t, &dhcp.UpdateSharedNetworkBeginOK{}, rsp)
	okRsp := rsp.(*dhcp.UpdateSharedNetworkBeginOK)
	contents := okRsp.Payload

	sharedNetwork := contents.SharedNetwork

	// Make sure the server returned transaction ID, daemons, shared networks and client classes.
	transactionID := contents.ID
	require.NotZero(t, transactionID)
	require.NotNil(t, sharedNetwork)
	require.Len(t, sharedNetwork.Subnets, 1)
	require.Len(t, contents.Daemons, 2)
	require.Len(t, contents.SharedNetworks4, 1)
	require.Equal(t, "foo", contents.SharedNetworks4[0])
	require.Empty(t, contents.SharedNetworks6)
	require.Len(t, contents.ClientClasses, 2)

	keaConfigSharedNetworkParameters := &models.KeaConfigSharedNetworkParameters{
		SharedNetworkLevelParameters: &models.KeaConfigSubnetDerivedParameters{
			KeaConfigCacheParameters: models.KeaConfigCacheParameters{
				CacheMaxAge:    storkutil.Ptr[int64](1000),
				CacheThreshold: storkutil.Ptr[float32](0.25),
			},
			KeaConfigClientClassParameters: models.KeaConfigClientClassParameters{
				ClientClass:          storkutil.Ptr("foo"),
				RequireClientClasses: []string{"bar"},
			},
			KeaConfigDdnsParameters: models.KeaConfigDdnsParameters{
				DdnsGeneratedPrefix:        storkutil.Ptr("abc"),
				DdnsOverrideClientUpdate:   storkutil.Ptr(true),
				DdnsOverrideNoUpdate:       storkutil.Ptr(false),
				DdnsQualifyingSuffix:       storkutil.Ptr("example.org"),
				DdnsReplaceClientName:      storkutil.Ptr("never"),
				DdnsSendUpdates:            storkutil.Ptr(true),
				DdnsUpdateOnRenew:          storkutil.Ptr(true),
				DdnsUseConflictResolution:  storkutil.Ptr(true),
				DdnsConflictResolutionMode: storkutil.Ptr("check-with-dhcid"),
			},
			KeaConfigFourOverSixParameters: models.KeaConfigFourOverSixParameters{
				FourOverSixInterface:   storkutil.Ptr("eth0"),
				FourOverSixInterfaceID: storkutil.Ptr("ifaceid"),
				FourOverSixSubnet:      storkutil.Ptr("2001:db8:1::/64"),
			},
			KeaConfigHostnameCharParameters: models.KeaConfigHostnameCharParameters{
				HostnameCharReplacement: storkutil.Ptr("a"),
				HostnameCharSet:         storkutil.Ptr("b"),
			},
			KeaConfigReservationParameters: models.KeaConfigReservationParameters{
				ReservationMode:       storkutil.Ptr("in-pool"),
				ReservationsGlobal:    storkutil.Ptr(false),
				ReservationsInSubnet:  storkutil.Ptr(true),
				ReservationsOutOfPool: storkutil.Ptr(false),
			},
			KeaConfigTimerParameters: models.KeaConfigTimerParameters{
				CalculateTeeTimes: storkutil.Ptr(true),
				RebindTimer:       storkutil.Ptr[int64](2000),
				RenewTimer:        storkutil.Ptr[int64](1000),
				T1Percent:         storkutil.Ptr[float32](0.25),
				T2Percent:         storkutil.Ptr[float32](0.50),
			},
			KeaConfigValidLifetimeParameters: models.KeaConfigValidLifetimeParameters{
				MaxValidLifetime: storkutil.Ptr[int64](5000),
				MinValidLifetime: storkutil.Ptr[int64](4000),
				ValidLifetime:    storkutil.Ptr[int64](4500),
			},
			KeaConfigAssortedSubnetParameters: models.KeaConfigAssortedSubnetParameters{
				Allocator:     storkutil.Ptr("random"),
				Authoritative: storkutil.Ptr(true),
				BootFileName:  storkutil.Ptr("/tmp/filename"),
				Interface:     storkutil.Ptr("eth0"),
				MatchClientID: storkutil.Ptr(true),
				NextServer:    storkutil.Ptr("192.0.2.1"),
				Relay: &models.KeaConfigAssortedSubnetParametersRelay{
					IPAddresses: []string{"10.1.1.1"},
				},
				ServerHostname:    storkutil.Ptr("myhost.example.org"),
				StoreExtendedInfo: storkutil.Ptr(true),
			},
			DHCPOptions: models.DHCPOptions{
				Options: []*models.DHCPOption{
					{
						AlwaysSend: true,
						Code:       3,
						Fields: []*models.DHCPOptionField{
							{
								FieldType: "ipv4-address",
								Values:    []string{"192.0.2.1"},
							},
						},
						Universe: 4,
					},
				},
			},
		},
	}

	// Submit transaction.
	params2 := dhcp.UpdateSharedNetworkSubmitParams{
		ID: transactionID,
		SharedNetwork: &models.SharedNetwork{
			ID:       sharedNetworks[0].ID,
			Name:     sharedNetworks[0].Name,
			Universe: int64(sharedNetworks[0].Family),
			Subnets: []*models.Subnet{
				{
					ID:     sharedNetwork.Subnets[0].ID,
					Subnet: sharedNetwork.Subnets[0].Subnet,
					LocalSubnets: []*models.LocalSubnet{
						{
							ID:       1,
							DaemonID: dbapps[0].Daemons[0].ID,
							KeaConfigSubnetParameters: &models.KeaConfigSubnetParameters{
								SubnetLevelParameters: &models.KeaConfigSubnetDerivedParameters{
									KeaConfigClientClassParameters: models.KeaConfigClientClassParameters{
										ClientClass: storkutil.Ptr("baz"),
									},
								},
							},
						},
						{
							ID:       1,
							DaemonID: dbapps[1].Daemons[0].ID,
							KeaConfigSubnetParameters: &models.KeaConfigSubnetParameters{
								SubnetLevelParameters: &models.KeaConfigSubnetDerivedParameters{
									KeaConfigClientClassParameters: models.KeaConfigClientClassParameters{
										ClientClass: storkutil.Ptr("baz"),
									},
								},
							},
						},
					},
				},
				{
					Subnet: "192.0.3.0/24",
					LocalSubnets: []*models.LocalSubnet{
						{
							ID:       2,
							DaemonID: dbapps[0].Daemons[0].ID,
						},
						{
							ID:       2,
							DaemonID: dbapps[1].Daemons[0].ID,
						},
					},
				},
			},
			LocalSharedNetworks: []*models.LocalSharedNetwork{
				{
					DaemonID:                         dbapps[0].Daemons[0].ID,
					KeaConfigSharedNetworkParameters: keaConfigSharedNetworkParameters,
				},
				{
					DaemonID:                         dbapps[1].Daemons[0].ID,
					KeaConfigSharedNetworkParameters: keaConfigSharedNetworkParameters,
				},
			},
		},
	}
	rsp2 := rapi.UpdateSharedNetworkSubmit(ctx, params2)
	require.IsType(t, &dhcp.UpdateSharedNetworkSubmitOK{}, rsp2)

	// Appropriate commands should be sent to two Kea servers.
	require.Len(t, fa.RecordedCommands, 6)

	for i, c := range fa.RecordedCommands {
		switch i {
		case 0, 2:
			require.JSONEq(t, `
				{
					"command": "network4-del",
					"service": ["dhcp4"],
					"arguments": {
						"name": "foo",
						"subnets-action":
						"delete"
					}
				}
				`, c.Marshal(),
			)
		case 1, 3:
			require.JSONEq(t, `
				{
					"command": "network4-add",
					"service": ["dhcp4"],
					"arguments": {
						"shared-networks": [
							{
								"cache-threshold": 0.25,
								"cache-max-age": 1000,
								"client-class": "foo",
								"require-client-classes": ["bar"],
								"ddns-generated-prefix": "abc",
								"ddns-override-client-update": true,
								"ddns-override-no-update": false,
								"ddns-qualifying-suffix": "example.org",
								"ddns-replace-client-name": "never",
								"ddns-send-updates": true,
								"ddns-update-on-renew": true,
								"ddns-use-conflict-resolution": true,
								"ddns-conflict-resolution-mode": "check-with-dhcid",
								"hostname-char-replacement": "a",
								"hostname-char-set": "b",
								"reservation-mode": "in-pool",
								"reservations-global": false,
								"reservations-in-subnet": true,
								"reservations-out-of-pool": false,
								"renew-timer": 1000,
								"rebind-timer": 2000,
								"t1-percent": 0.25,
								"t2-percent": 0.5,
								"calculate-tee-times": true,
								"valid-lifetime": 4500,
								"min-valid-lifetime": 4000,
								"max-valid-lifetime": 5000,
								"allocator": "random",
								"interface": "eth0",
								"store-extended-info": true,
								"option-data": [
									{
										"always-send": true,
										"code": 3,
										"csv-format": true,
										"data": "192.0.2.1",
										"space": "dhcp4"
									}
								],
								"relay": {
									"ip-addresses":["10.1.1.1"]
								},
								"authoritative": true,
								"boot-file-name": "/tmp/filename",
								"match-client-id": true,
								"name": "foo",
								"next-server": "192.0.2.1",
								"server-hostname": "myhost.example.org",
								"subnet4": [
									{
										"id": 1,
										"subnet": "192.0.2.0/24",
										"client-class": "baz"
									},
									{
										"id": 2,
										"subnet": "192.0.3.0/24"
									}
								]
							}
						]
					}
				}
			`, c.Marshal())
		default:
			require.JSONEq(t, `
				{
					"command": "config-write",
					"service": ["dhcp4"]
				}
			`, c.Marshal())
		}
	}

	// Make sure that the transaction is done.
	cctx, _ := cm.RecoverContext(transactionID, int64(user.ID))
	// Remove the context from the config manager before testing that
	// the returned context is nil. If it happens to be non-nil the
	// require.Nil() would otherwise spit out errors about the concurrent
	// access to the context in the manager's goroutine and here.
	if cctx != nil {
		cm.Done(cctx)
	}
	require.Nil(t, cctx)

	// Make sure that the updated shared network has been stored in the database.
	returnedSharedNetwork, err := dbmodel.GetSharedNetwork(db, sharedNetworks[0].ID)
	require.NoError(t, err)
	require.NotNil(t, returnedSharedNetwork)

	// Subnets
	require.Len(t, returnedSharedNetwork.Subnets, 2)

	var (
		subnet0 *dbmodel.Subnet
		subnet1 *dbmodel.Subnet
	)
	for i := range returnedSharedNetwork.Subnets {
		switch returnedSharedNetwork.Subnets[i].Prefix {
		case "192.0.2.0/24":
			subnet0 = &returnedSharedNetwork.Subnets[i]
		default:
			subnet1 = &returnedSharedNetwork.Subnets[i]
		}
	}
	require.NotNil(t, subnet0)
	require.EqualValues(t, sharedNetwork.Subnets[0].ID, subnet0.ID)
	require.Equal(t, "192.0.2.0/24", subnet0.Prefix)
	require.Len(t, subnet0.LocalSubnets, 2)

	for _, ls := range subnet0.LocalSubnets {
		require.EqualValues(t, 1, ls.LocalSubnetID)
		require.NotNil(t, ls.KeaParameters.ClientClass)
		require.Equal(t, "baz", *ls.KeaParameters.ClientClass)
	}

	require.NotNil(t, subnet1)
	require.Equal(t, "192.0.3.0/24", subnet1.Prefix)
	require.Len(t, subnet1.LocalSubnets, 2)

	for _, ls := range subnet1.LocalSubnets {
		require.EqualValues(t, 2, ls.LocalSubnetID)
	}

	require.Len(t, returnedSharedNetwork.LocalSharedNetworks, 2)
	for _, lsn := range returnedSharedNetwork.LocalSharedNetworks {
		require.NotNil(t, lsn.KeaParameters)
		require.NotNil(t, lsn.KeaParameters.CacheMaxAge)
		require.EqualValues(t, 1000, *lsn.KeaParameters.CacheMaxAge)
		require.NotNil(t, lsn.KeaParameters.CacheThreshold)
		require.EqualValues(t, 0.25, *lsn.KeaParameters.CacheThreshold)
		require.NotNil(t, lsn.KeaParameters.ClientClass)
		require.Equal(t, "foo", *lsn.KeaParameters.ClientClass)
		require.Len(t, lsn.KeaParameters.RequireClientClasses, 1)
		require.EqualValues(t, "bar", lsn.KeaParameters.RequireClientClasses[0])
		require.NotNil(t, lsn.KeaParameters.DDNSGeneratedPrefix)
		require.Equal(t, "abc", *lsn.KeaParameters.DDNSGeneratedPrefix)
		require.NotNil(t, lsn.KeaParameters.DDNSOverrideClientUpdate)
		require.True(t, *lsn.KeaParameters.DDNSOverrideClientUpdate)
		require.NotNil(t, lsn.KeaParameters.DDNSOverrideNoUpdate)
		require.False(t, *lsn.KeaParameters.DDNSOverrideNoUpdate)
		require.NotNil(t, lsn.KeaParameters.DDNSQualifyingSuffix)
		require.Equal(t, "example.org", *lsn.KeaParameters.DDNSQualifyingSuffix)
		require.NotNil(t, lsn.KeaParameters.DDNSReplaceClientName)
		require.Equal(t, "never", *lsn.KeaParameters.DDNSReplaceClientName)
		require.NotNil(t, lsn.KeaParameters.DDNSSendUpdates)
		require.True(t, *lsn.KeaParameters.DDNSSendUpdates)
		require.NotNil(t, lsn.KeaParameters.DDNSUpdateOnRenew)
		require.True(t, *lsn.KeaParameters.DDNSUpdateOnRenew)
		require.NotNil(t, lsn.KeaParameters.DDNSUseConflictResolution)
		require.True(t, *lsn.KeaParameters.DDNSUseConflictResolution)
		require.NotNil(t, lsn.KeaParameters.HostnameCharReplacement)
		require.Equal(t, "a", *lsn.KeaParameters.HostnameCharReplacement)
		require.NotNil(t, lsn.KeaParameters.HostnameCharSet)
		require.Equal(t, "b", *lsn.KeaParameters.HostnameCharSet)
		require.NotNil(t, lsn.KeaParameters.ReservationMode)
		require.Equal(t, "in-pool", *lsn.KeaParameters.ReservationMode)
		require.NotNil(t, lsn.KeaParameters.ReservationsGlobal)
		require.False(t, *lsn.KeaParameters.ReservationsGlobal)
		require.NotNil(t, lsn.KeaParameters.ReservationsInSubnet)
		require.True(t, *lsn.KeaParameters.ReservationsInSubnet)
		require.NotNil(t, lsn.KeaParameters.ReservationsOutOfPool)
		require.False(t, *lsn.KeaParameters.ReservationsOutOfPool)
		require.NotNil(t, lsn.KeaParameters.CalculateTeeTimes)
		require.True(t, *lsn.KeaParameters.CalculateTeeTimes)
		require.NotNil(t, lsn.KeaParameters.RebindTimer)
		require.EqualValues(t, 2000, *lsn.KeaParameters.RebindTimer)
		require.NotNil(t, lsn.KeaParameters.RenewTimer)
		require.EqualValues(t, 1000, *lsn.KeaParameters.RenewTimer)
		require.NotNil(t, lsn.KeaParameters.T1Percent)
		require.EqualValues(t, 0.25, *lsn.KeaParameters.T1Percent)
		require.NotNil(t, lsn.KeaParameters.T2Percent)
		require.EqualValues(t, 0.50, *lsn.KeaParameters.T2Percent)
		require.NotNil(t, lsn.KeaParameters.MaxValidLifetime)
		require.EqualValues(t, 5000, *lsn.KeaParameters.MaxValidLifetime)
		require.NotNil(t, lsn.KeaParameters.MinValidLifetime)
		require.EqualValues(t, 4000, *lsn.KeaParameters.MinValidLifetime)
		require.NotNil(t, lsn.KeaParameters.ValidLifetime)
		require.EqualValues(t, 4500, *lsn.KeaParameters.ValidLifetime)
		require.NotNil(t, lsn.KeaParameters.Allocator)
		require.Equal(t, "random", *lsn.KeaParameters.Allocator)
		require.NotNil(t, lsn.KeaParameters.Authoritative)
		require.True(t, *lsn.KeaParameters.Authoritative)
		require.NotNil(t, lsn.KeaParameters.Authoritative)
		require.Equal(t, "/tmp/filename", *lsn.KeaParameters.BootFileName)
		require.NotNil(t, lsn.KeaParameters.Interface)
		require.Equal(t, "eth0", *lsn.KeaParameters.Interface)
		require.NotNil(t, lsn.KeaParameters.MatchClientID)
		require.True(t, *lsn.KeaParameters.MatchClientID)
		require.NotNil(t, lsn.KeaParameters.NextServer)
		require.Equal(t, "192.0.2.1", *lsn.KeaParameters.NextServer)
		require.NotNil(t, lsn.KeaParameters.Relay)
		require.Len(t, lsn.KeaParameters.Relay.IPAddresses, 1)
		require.Equal(t, "10.1.1.1", lsn.KeaParameters.Relay.IPAddresses[0])
		require.NotNil(t, lsn.KeaParameters.ServerHostname)
		require.Equal(t, "myhost.example.org", *lsn.KeaParameters.ServerHostname)
		require.NotNil(t, lsn.KeaParameters.StoreExtendedInfo)
		require.True(t, *lsn.KeaParameters.StoreExtendedInfo)

		// DHCP options
		require.Len(t, lsn.DHCPOptionSet.Options, 1)
		require.True(t, lsn.DHCPOptionSet.Options[0].AlwaysSend)
		require.EqualValues(t, 3, lsn.DHCPOptionSet.Options[0].Code)
		require.Len(t, lsn.DHCPOptionSet.Options[0].Fields, 1)
		require.Equal(t, dhcpmodel.IPv4AddressField, lsn.DHCPOptionSet.Options[0].Fields[0].FieldType)
		require.Len(t, lsn.DHCPOptionSet.Options[0].Fields[0].Values, 1)
		require.Equal(t, "192.0.2.1", lsn.DHCPOptionSet.Options[0].Fields[0].Values[0])
		require.Equal(t, dhcpmodel.DHCPv4OptionSpace, lsn.DHCPOptionSet.Options[0].Space)
		require.NotEmpty(t, lsn.DHCPOptionSet.Hash)
	}
}

// Test the calls for creating new transaction and updating a shared network.
func TestUpdateSharedNetwork6BeginSubmit(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	serverConfig := `{
		"Dhcp6": {
			"client-classes": [
				{
					"name": "devices"
				},
				{
					"name": "printers"
				}
			],
			"shared-networks": [
				{
					"name": "foo",
					"subnet6": [
						{
							"id": 1,
							"subnet": "2001:db8:1::/64"
						}
					]
				}
			],
			"hooks-libraries": [
				{
					"library": "libdhcp_subnet_cmds"
				}
			]
		}
	}`

	server1, err := dbmodeltest.NewKeaDHCPv6Server(db)
	require.NoError(t, err)
	err = server1.Configure(serverConfig)
	require.NoError(t, err)

	app, err := server1.GetKea()
	require.NoError(t, err)

	err = kea.CommitAppIntoDB(db, app, &storktest.FakeEventCenter{}, nil, dbmodel.NewDHCPOptionDefinitionLookup())
	require.NoError(t, err)

	server2, err := dbmodeltest.NewKeaDHCPv6Server(db)
	require.NoError(t, err)
	err = server2.Configure(serverConfig)
	require.NoError(t, err)

	app, err = server2.GetKea()
	require.NoError(t, err)

	err = kea.CommitAppIntoDB(db, app, &storktest.FakeEventCenter{}, nil, dbmodel.NewDHCPOptionDefinitionLookup())
	require.NoError(t, err)

	dbapps, err := dbmodel.GetAllApps(db, true)
	require.NoError(t, err)
	require.Len(t, dbapps, 2)

	sharedNetworks, err := dbmodel.GetAllSharedNetworks(db, 6)
	require.NoError(t, err)
	require.Len(t, sharedNetworks, 1)

	// Create fake agents receiving commands.
	fa := agentcommtest.NewFakeAgents(nil, nil)
	require.NotNil(t, fa)

	lookup := dbmodel.NewDHCPOptionDefinitionLookup()
	require.NotNil(t, lookup)

	// Create the config manager.
	cm := apps.NewManager(&appstest.ManagerAccessorsWrapper{
		DB:        db,
		Agents:    fa,
		DefLookup: lookup,
	})
	require.NotNil(t, cm)

	// Create API.
	rapi, err := NewRestAPI(dbSettings, db, fa, cm, lookup)
	require.NoError(t, err)

	// Create session manager.
	ctx, err := rapi.SessionManager.Load(context.Background(), "")
	require.NoError(t, err)

	// Create user session.
	user := &dbmodel.SystemUser{
		ID: 1234,
	}
	err = rapi.SessionManager.LoginHandler(ctx, user)
	require.NoError(t, err)

	// Begin transaction.
	params := dhcp.UpdateSharedNetworkBeginParams{
		SharedNetworkID: sharedNetworks[0].ID,
	}
	rsp := rapi.UpdateSharedNetworkBegin(ctx, params)
	require.IsType(t, &dhcp.UpdateSharedNetworkBeginOK{}, rsp)
	okRsp := rsp.(*dhcp.UpdateSharedNetworkBeginOK)
	contents := okRsp.Payload

	// Make sure the server returned transaction ID, shared networks, daemons and client classes.
	transactionID := contents.ID
	require.NotZero(t, transactionID)
	require.NotNil(t, contents.SharedNetwork)
	require.Len(t, contents.Daemons, 2)
	require.Empty(t, contents.SharedNetworks4)
	require.Len(t, contents.SharedNetworks6, 1)
	require.Equal(t, "foo", contents.SharedNetworks6[0])
	require.Len(t, contents.ClientClasses, 2)

	keaConfigSharedNetworkParameters := &models.KeaConfigSharedNetworkParameters{
		SharedNetworkLevelParameters: &models.KeaConfigSubnetDerivedParameters{
			KeaConfigPreferredLifetimeParameters: models.KeaConfigPreferredLifetimeParameters{
				MaxPreferredLifetime: storkutil.Ptr[int64](5000),
				MinPreferredLifetime: storkutil.Ptr[int64](4000),
				PreferredLifetime:    storkutil.Ptr[int64](4500),
			},
			KeaConfigAssortedSubnetParameters: models.KeaConfigAssortedSubnetParameters{
				InterfaceID: storkutil.Ptr("ifaceid"),
				PdAllocator: storkutil.Ptr("random"),
			},
		},
	}

	// Submit transaction.
	params2 := dhcp.UpdateSharedNetworkSubmitParams{
		ID: transactionID,
		SharedNetwork: &models.SharedNetwork{
			ID:       sharedNetworks[0].ID,
			Name:     sharedNetworks[0].Name,
			Universe: int64(sharedNetworks[0].Family),
			LocalSharedNetworks: []*models.LocalSharedNetwork{
				{
					DaemonID:                         dbapps[0].Daemons[0].ID,
					KeaConfigSharedNetworkParameters: keaConfigSharedNetworkParameters,
				},
				{
					DaemonID:                         dbapps[1].Daemons[0].ID,
					KeaConfigSharedNetworkParameters: keaConfigSharedNetworkParameters,
				},
			},
		},
	}
	rsp2 := rapi.UpdateSharedNetworkSubmit(ctx, params2)
	require.IsType(t, &dhcp.UpdateSharedNetworkSubmitOK{}, rsp2)

	// Appropriate commands should be sent to two Kea servers.
	require.Len(t, fa.RecordedCommands, 6)

	for i, c := range fa.RecordedCommands {
		switch i {
		case 0, 2:
			require.JSONEq(t, `
				{
					"command": "network6-del",
					"service": ["dhcp6"],
					"arguments": {
						"name": "foo",
						"subnets-action": "delete"
					}
				}
			`, c.Marshal())
		case 1, 3:
			require.JSONEq(t, `
				{
					"command": "network6-add",
					"service": ["dhcp6"],
					"arguments": {
						"shared-networks": [
							{
								"preferred-lifetime": 4500,
								"min-preferred-lifetime": 4000,
								"max-preferred-lifetime": 5000,
								"pd-allocator": "random",
								"interface-id": "ifaceid",
								"name": "foo"
							}
						]
					}
				}
			`, c.Marshal())
		default:
			require.JSONEq(t, `
				{
					"command": "config-write",
					"service": ["dhcp6"]
				}
			`, c.Marshal())
		}
	}

	// Make sure that the transaction is done.
	cctx, _ := cm.RecoverContext(transactionID, int64(user.ID))
	// Remove the context from the config manager before testing that
	// the returned context is nil. If it happens to be non-nil the
	// require.Nil() would otherwise spit out errors about the concurrent
	// access to the context in the manager's goroutine and here.
	if cctx != nil {
		cm.Done(cctx)
	}
	require.Nil(t, cctx)

	// Make sure that the updated shared network has been stored in the database.
	returnedSharedNetwork, err := dbmodel.GetSharedNetwork(db, sharedNetworks[0].ID)
	require.NoError(t, err)
	require.NotNil(t, returnedSharedNetwork)

	require.Len(t, returnedSharedNetwork.LocalSharedNetworks, 2)
	for _, lsn := range returnedSharedNetwork.LocalSharedNetworks {
		require.NotNil(t, lsn.KeaParameters)

		require.NotNil(t, lsn.KeaParameters.InterfaceID)
		require.Equal(t, "ifaceid", *lsn.KeaParameters.InterfaceID)

		require.NotNil(t, lsn.KeaParameters.MinPreferredLifetime)
		require.EqualValues(t, 4000, *lsn.KeaParameters.MinPreferredLifetime)

		require.NotNil(t, lsn.KeaParameters.MinPreferredLifetime)
		require.EqualValues(t, 5000, *lsn.KeaParameters.MaxPreferredLifetime)

		require.NotNil(t, lsn.KeaParameters.PreferredLifetime)
		require.EqualValues(t, 4500, *lsn.KeaParameters.PreferredLifetime)

		require.NotNil(t, lsn.KeaParameters.PDAllocator)
		require.Equal(t, "random", *lsn.KeaParameters.PDAllocator)

		// DHCP options
		require.Empty(t, lsn.DHCPOptionSet.Options)
		require.Empty(t, lsn.DHCPOptionSet.Hash)
	}
}

// Test that the transaction to update a shared network can be canceled, resulting
// in the removal of this transaction from the config manager and allowing
// another user to apply config updates.
func TestUpdateSharedNetworkBeginCancel(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	serverConfig := `{
		"Dhcp6": {
			"shared-networks": [
				{
					"name": "foo",
					"subnet6": [
						{
							"id": 1,
							"subnet": "2001:db8:1::/64"
						}
					]
				}
			],
			"hooks-libraries": [
				{
					"library": "libdhcp_subnet_cmds"
				}
			]
		}
	}`

	server1, err := dbmodeltest.NewKeaDHCPv6Server(db)
	require.NoError(t, err)
	err = server1.Configure(serverConfig)
	require.NoError(t, err)

	app, err := server1.GetKea()
	require.NoError(t, err)

	err = kea.CommitAppIntoDB(db, app, &storktest.FakeEventCenter{}, nil, dbmodel.NewDHCPOptionDefinitionLookup())
	require.NoError(t, err)

	server2, err := dbmodeltest.NewKeaDHCPv6Server(db)
	require.NoError(t, err)
	err = server2.Configure(serverConfig)
	require.NoError(t, err)

	app, err = server2.GetKea()
	require.NoError(t, err)

	err = kea.CommitAppIntoDB(db, app, &storktest.FakeEventCenter{}, nil, dbmodel.NewDHCPOptionDefinitionLookup())
	require.NoError(t, err)

	dbapps, err := dbmodel.GetAllApps(db, true)
	require.NoError(t, err)
	require.Len(t, dbapps, 2)

	sharedNetworks, err := dbmodel.GetAllSharedNetworks(db, 6)
	require.NoError(t, err)
	require.Len(t, sharedNetworks, 1)

	// Create fake agents receiving commands.
	fa := agentcommtest.NewFakeAgents(nil, nil)
	require.NotNil(t, fa)

	lookup := dbmodel.NewDHCPOptionDefinitionLookup()
	require.NotNil(t, lookup)

	// Create the config manager.
	cm := apps.NewManager(&appstest.ManagerAccessorsWrapper{
		DB:        db,
		Agents:    fa,
		DefLookup: lookup,
	})
	require.NotNil(t, cm)

	// Create API.
	rapi, err := NewRestAPI(dbSettings, db, fa, cm, lookup)
	require.NoError(t, err)

	// Create session manager.
	ctx, err := rapi.SessionManager.Load(context.Background(), "")
	require.NoError(t, err)

	// Create user session.
	user := &dbmodel.SystemUser{
		ID: 1234,
	}
	err = rapi.SessionManager.LoginHandler(ctx, user)
	require.NoError(t, err)

	// Begin transaction.
	params := dhcp.UpdateSharedNetworkBeginParams{
		SharedNetworkID: sharedNetworks[0].ID,
	}
	rsp := rapi.UpdateSharedNetworkBegin(ctx, params)
	require.IsType(t, &dhcp.UpdateSharedNetworkBeginOK{}, rsp)
	okRsp := rsp.(*dhcp.UpdateSharedNetworkBeginOK)
	contents := okRsp.Payload

	// Make sure the server returned transaction ID, shared networks, daemons and client classes.
	transactionID := contents.ID
	require.NotZero(t, transactionID)
	require.NotNil(t, contents.SharedNetwork)
	require.Len(t, contents.Daemons, 2)

	// Try to start another session by another user.
	ctx2, err := rapi.SessionManager.Load(context.Background(), "")
	require.NoError(t, err)

	// Create user session.
	user = &dbmodel.SystemUser{
		ID: 2345,
	}
	err = rapi.SessionManager.LoginHandler(ctx2, user)
	require.NoError(t, err)

	// It should fail because the first session locked the daemons for
	// update.
	rsp = rapi.UpdateSharedNetworkBegin(ctx2, params)
	require.IsType(t, &dhcp.UpdateSharedNetworkBeginDefault{}, rsp)
	defaultRsp := rsp.(*dhcp.UpdateSharedNetworkBeginDefault)
	require.Equal(t, http.StatusLocked, getStatusCode(*defaultRsp))
	require.Equal(t, fmt.Sprintf("Unable to edit the shared network with ID %d because it may be currently edited by another user", sharedNetworks[0].ID), *defaultRsp.Payload.Message)

	// Cancel the transaction.
	params2 := dhcp.UpdateSharedNetworkDeleteParams{
		ID: transactionID,
	}
	rsp2 := rapi.UpdateSharedNetworkDelete(ctx, params2)
	require.IsType(t, &dhcp.UpdateSharedNetworkDeleteOK{}, rsp2)

	cctx, _ := cm.RecoverContext(transactionID, int64(user.ID))
	// Remove the context from the config manager before testing that
	// the returned context is nil. If it happens to be non-nil the
	// require.Nil() would otherwise spit out errors about the concurrent
	// access to the context in the manager's goroutine and here.
	if cctx != nil {
		cm.Done(cctx)
	}
	require.Nil(t, cctx)

	// After we released the lock, another user should be able to apply
	// his changes.
	rsp = rapi.UpdateSharedNetworkBegin(ctx2, params)
	require.IsType(t, &dhcp.UpdateSharedNetworkBeginOK{}, rsp)
}

// Test successfully deleting a shared network.
func TestDeleteSharedNetwork(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	serverConfig := `{
		"Dhcp4": {
			"shared-networks": [
				{
					"name": "foo",
					"subnet4": [
						{
							"id": 1,
							"subnet": "192.0.2.0/24"
						}
					]
				}
			],
			"hooks-libraries": [
				{
					"library": "libdhcp_subnet_cmds"
				}
			]
		}
	}`

	server1, err := dbmodeltest.NewKeaDHCPv4Server(db)
	require.NoError(t, err)
	err = server1.Configure(serverConfig)
	require.NoError(t, err)

	app, err := server1.GetKea()
	require.NoError(t, err)

	err = kea.CommitAppIntoDB(db, app, &storktest.FakeEventCenter{}, nil, dbmodel.NewDHCPOptionDefinitionLookup())
	require.NoError(t, err)

	server2, err := dbmodeltest.NewKeaDHCPv4Server(db)
	require.NoError(t, err)
	err = server2.Configure(serverConfig)
	require.NoError(t, err)

	app, err = server2.GetKea()
	require.NoError(t, err)

	err = kea.CommitAppIntoDB(db, app, &storktest.FakeEventCenter{}, nil, dbmodel.NewDHCPOptionDefinitionLookup())
	require.NoError(t, err)

	dbapps, err := dbmodel.GetAllApps(db, true)
	require.NoError(t, err)
	require.Len(t, dbapps, 2)

	sharedNetworks, err := dbmodel.GetAllSharedNetworks(db, 4)
	require.NoError(t, err)
	require.Len(t, sharedNetworks, 1)

	// Create fake agents receiving commands.
	fa := agentcommtest.NewFakeAgents(nil, nil)
	require.NotNil(t, fa)

	lookup := dbmodel.NewDHCPOptionDefinitionLookup()
	require.NotNil(t, lookup)

	// Create the config manager.
	cm := apps.NewManager(&appstest.ManagerAccessorsWrapper{
		DB:        db,
		Agents:    fa,
		DefLookup: lookup,
	})
	require.NotNil(t, cm)

	// Create API.
	rapi, err := NewRestAPI(dbSettings, db, fa, cm, lookup)
	require.NoError(t, err)

	// Create session manager.
	ctx, err := rapi.SessionManager.Load(context.Background(), "")
	require.NoError(t, err)

	// Create user session.
	user := &dbmodel.SystemUser{
		ID: 1234,
	}
	err = rapi.SessionManager.LoginHandler(ctx, user)
	require.NoError(t, err)

	// Attempt to delete the shared network.
	params := dhcp.DeleteSharedNetworkParams{
		ID: sharedNetworks[0].ID,
	}
	rsp := rapi.DeleteSharedNetwork(ctx, params)
	require.IsType(t, &dhcp.DeleteSharedNetworkOK{}, rsp)

	// The network4-del and config-write commands should be sent to two Kea servers.
	require.Len(t, fa.RecordedCommands, 4)

	for i, c := range fa.RecordedCommands {
		switch {
		case i < 2:
			require.JSONEq(t, `{
				"command": "network4-del",
				"service": ["dhcp4"],
				"arguments": {
					"name": "foo",
					"subnets-action": "delete"
				}
		}`, c.Marshal())
		default:
			require.JSONEq(t, `{
				"command": "config-write",
				"service": ["dhcp4"]
			}`, c.Marshal())
		}
	}
	returnedSharedNetwork, err := dbmodel.GetSharedNetwork(db, sharedNetworks[0].ID)
	require.NoError(t, err)
	require.Nil(t, returnedSharedNetwork)
}

// Test error cases for deleting a shared network.
func TestDeleteSharedNetworkError(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Setup fake agents that return an error in response to network4-del
	// command.
	fa := agentcommtest.NewFakeAgents(func(callNo int, cmdResponses []interface{}) {
		mockStatusError("network4-del", cmdResponses)
	}, nil)
	require.NotNil(t, fa)

	serverConfig := `{
		"Dhcp4": {
			"shared-networks": [
				{
					"name": "foo",
					"subnet4": [
						{
							"id": 1,
							"subnet": "192.0.2.0/24"
						}
					]
				}
			],
			"hooks-libraries": [
				{
					"library": "libdhcp_subnet_cmds"
				}
			]
		}
	}`

	server1, err := dbmodeltest.NewKeaDHCPv4Server(db)
	require.NoError(t, err)
	err = server1.Configure(serverConfig)
	require.NoError(t, err)

	app, err := server1.GetKea()
	require.NoError(t, err)

	err = kea.CommitAppIntoDB(db, app, &storktest.FakeEventCenter{}, nil, dbmodel.NewDHCPOptionDefinitionLookup())
	require.NoError(t, err)

	dbapps, err := dbmodel.GetAllApps(db, true)
	require.NoError(t, err)
	require.Len(t, dbapps, 1)

	sharedNetworks, err := dbmodel.GetAllSharedNetworks(db, 4)
	require.NoError(t, err)
	require.Len(t, sharedNetworks, 1)

	lookup := dbmodel.NewDHCPOptionDefinitionLookup()
	require.NotNil(t, lookup)

	// Create the config manager.
	cm := apps.NewManager(&appstest.ManagerAccessorsWrapper{
		DB:        db,
		Agents:    fa,
		DefLookup: lookup,
	})
	require.NotNil(t, cm)

	// Create API.
	rapi, err := NewRestAPI(dbSettings, db, fa, cm, lookup)
	require.NoError(t, err)

	// Create session manager.
	ctx, err := rapi.SessionManager.Load(context.Background(), "")
	require.NoError(t, err)

	// Create user session.
	user := &dbmodel.SystemUser{
		ID: 1234,
	}
	err = rapi.SessionManager.LoginHandler(ctx, user)
	require.NoError(t, err)

	// Submit transaction with non-matching subnet ID.
	t.Run("wrong shared network id", func(t *testing.T) {
		params := dhcp.DeleteSharedNetworkParams{
			ID: 19809865,
		}
		rsp := rapi.DeleteSharedNetwork(ctx, params)
		require.IsType(t, &dhcp.DeleteSharedNetworkDefault{}, rsp)
		defaultRsp := rsp.(*dhcp.DeleteSharedNetworkDefault)
		require.Equal(t, http.StatusNotFound, getStatusCode(*defaultRsp))
		require.Equal(t, "Cannot find a shared network with ID 19809865", *defaultRsp.Payload.Message)
	})

	// Submit transaction with valid ID but expect the agent to return an
	// error code. This is considered a conflict with the state of the
	// Kea servers.
	t.Run("commit failure", func(t *testing.T) {
		params := dhcp.DeleteSharedNetworkParams{
			ID: sharedNetworks[0].ID,
		}
		rsp := rapi.DeleteSharedNetwork(ctx, params)
		require.IsType(t, &dhcp.DeleteSharedNetworkDefault{}, rsp)
		defaultRsp := rsp.(*dhcp.DeleteSharedNetworkDefault)
		require.Equal(t, http.StatusConflict, getStatusCode(*defaultRsp))
		require.Equal(t, fmt.Sprintf("Problem with deleting a shared network: network4-del command to %s failed: error status (1) returned by Kea dhcp4 daemon with text: 'unable to communicate with the daemon'", app.GetName()), *defaultRsp.Payload.Message)
	})
}
