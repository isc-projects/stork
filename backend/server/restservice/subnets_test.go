package restservice

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	dhcpmodel "isc.org/stork/datamodel/dhcp"
	agentcommtest "isc.org/stork/server/agentcomm/test"
	"isc.org/stork/server/apps/kea"
	dbmodel "isc.org/stork/server/database/model"
	dbmodeltest "isc.org/stork/server/database/model/test"
	dbtest "isc.org/stork/server/database/test"
	dhcp "isc.org/stork/server/gen/restapi/operations/d_h_c_p"
	storktest "isc.org/stork/server/test/dbmodel"
	"isc.org/stork/testutil"
	storkutil "isc.org/stork/util"
)

// Check getting subnets via rest api functions.
func TestGetSubnets(t *testing.T) {
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
	params := dhcp.GetSubnetsParams{}
	rsp := rapi.GetSubnets(ctx, params)
	require.IsType(t, &dhcp.GetSubnetsOK{}, rsp)
	okRsp := rsp.(*dhcp.GetSubnetsOK)
	require.Len(t, okRsp.Payload.Items, 0)
	require.Zero(t, okRsp.Payload.Total)

	dhcp4, err := dbmodeltest.NewKeaDHCPv4Server(db)
	require.NoError(t, err)

	dhcp4.Configure(`{
        "Dhcp4": {
            "subnet4": [
                {
                    "id": 1,
                    "subnet": "192.168.0.0/24",
                    "pools": [
                        {
                            "pool": "192.168.0.1-192.168.0.100"
                        },
                        {
                            "pool": "192.168.0.150-192.168.0.200"
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
            "subnet6": [
                {
                    "id": 2,
                    "subnet": "2001:db8:1::/64",
                    "pools": []
                }
            ]
        }
    }`)

	app, err = dhcp6.GetKea()
	require.NoError(t, err)

	err = kea.CommitAppIntoDB(db, app, &storktest.FakeEventCenter{}, nil, dbmodel.NewDHCPOptionDefinitionLookup())
	require.NoError(t, err)

	kea46, err := dbmodeltest.NewKea(db)
	require.NoError(t, err)

	dhcp4, err = kea46.NewKeaDHCPv4Server()
	require.NoError(t, err)

	err = dhcp4.Configure(`{
        "Dhcp4": {
            "subnet4": [
                {
                    "id": 3,
                    "subnet": "192.118.0.0/24",
                    "pools": [
                        {
                            "pool": "192.118.0.1-192.118.0.200"
                        }
                    ]
                }
            ]
        }
    }`)
	require.NoError(t, err)

	dhcp6, err = kea46.NewKeaDHCPv6Server()
	require.NoError(t, err)

	err = dhcp6.Configure(`{
        "Dhcp6": {
            "subnet6": [
                {
                    "id": 4,
                    "subnet": "3001:db8:1::/64",
                    "pools": [
                        {
                            "pool": "3001:db8:1::/80"
                        }
                    ]
                }
            ],
            "shared-networks": [
                {
                    "name": "fox",
                    "subnet6": [
                        {
                            "id":     21,
                            "subnet": "5001:db8:1::/64"
                        }
                    ]
                }
            ]
        }
    }`)
	require.NoError(t, err)

	app, err = dhcp6.GetKea()
	require.NoError(t, err)

	a46 := app

	err = kea.CommitAppIntoDB(db, app, &storktest.FakeEventCenter{}, nil, dbmodel.NewDHCPOptionDefinitionLookup())
	require.NoError(t, err)

	subnets, err := dbmodel.GetSubnetsByPrefix(db, "192.118.0.0/24")
	require.NoError(t, err)
	require.Len(t, subnets, 1)
	subnets[0].Stats = dbmodel.SubnetStats{
		"bar": 24,
	}
	subnets[0].StatsCollectedAt = time.Time{}.Add(2 * time.Hour)
	dbmodel.CommitNetworksIntoDB(db, []dbmodel.SharedNetwork{}, subnets, a46.Daemons[0])

	subnets, err = dbmodel.GetSubnetsByPrefix(db, "3001:db8:1::/64")
	require.NoError(t, err)
	require.Len(t, subnets, 1)
	subnets[0].Stats = dbmodel.SubnetStats{
		"baz": 4224,
	}
	subnets[0].StatsCollectedAt = time.Time{}.Add(3 * time.Hour)
	subnets[0].AddrUtilization = 240
	subnets[0].PdUtilization = 420
	dbmodel.CommitNetworksIntoDB(db, []dbmodel.SharedNetwork{}, subnets, a46.Daemons[1])

	// get all subnets
	params = dhcp.GetSubnetsParams{}
	rsp = rapi.GetSubnets(ctx, params)
	require.IsType(t, &dhcp.GetSubnetsOK{}, rsp)
	okRsp = rsp.(*dhcp.GetSubnetsOK)
	require.Len(t, okRsp.Payload.Items, 5)
	require.EqualValues(t, 5, okRsp.Payload.Total)
	for _, sn := range okRsp.Payload.Items {
		switch sn.LocalSubnets[0].ID {
		case 1:
			require.Len(t, sn.LocalSubnets[0].Pools, 2)
		case 2:
			require.Len(t, sn.LocalSubnets[0].Pools, 0)
		case 21:
			require.Len(t, sn.LocalSubnets[0].Pools, 0)
		default:
			require.Len(t, sn.LocalSubnets[0].Pools, 1)
		}
	}

	// get subnets from app a4
	params = dhcp.GetSubnetsParams{
		AppID: &a4.ID,
	}
	rsp = rapi.GetSubnets(ctx, params)
	require.IsType(t, &dhcp.GetSubnetsOK{}, rsp)
	okRsp = rsp.(*dhcp.GetSubnetsOK)
	require.Len(t, okRsp.Payload.Items, 1)
	require.EqualValues(t, 1, okRsp.Payload.Total)
	require.Len(t, okRsp.Payload.Items[0].LocalSubnets, 1)
	require.Equal(t, a4.ID, okRsp.Payload.Items[0].LocalSubnets[0].AppID)
	require.Equal(t, a4.Name, okRsp.Payload.Items[0].LocalSubnets[0].AppName)
	require.EqualValues(t, 1, okRsp.Payload.Items[0].ID)
	require.EqualValues(t, dbmodel.SubnetStats(nil), okRsp.Payload.Items[0].Stats)
	require.EqualValues(t, time.Time{}, okRsp.Payload.Items[0].StatsCollectedAt)

	// get subnets from app a46
	params = dhcp.GetSubnetsParams{
		AppID: &a46.ID,
	}
	rsp = rapi.GetSubnets(ctx, params)
	require.IsType(t, &dhcp.GetSubnetsOK{}, rsp)
	okRsp = rsp.(*dhcp.GetSubnetsOK)
	require.Len(t, okRsp.Payload.Items, 3)
	require.EqualValues(t, 3, okRsp.Payload.Total)
	// checking if returned subnet-ids have expected values
	require.ElementsMatch(t, []int64{3, 4, 21},
		[]int64{
			okRsp.Payload.Items[0].LocalSubnets[0].ID,
			okRsp.Payload.Items[1].LocalSubnets[0].ID,
			okRsp.Payload.Items[2].LocalSubnets[0].ID,
		})

	// get v4 subnets
	var dhcpVer int64 = 4
	params = dhcp.GetSubnetsParams{
		DhcpVersion: &dhcpVer,
	}
	rsp = rapi.GetSubnets(ctx, params)
	require.IsType(t, &dhcp.GetSubnetsOK{}, rsp)
	okRsp = rsp.(*dhcp.GetSubnetsOK)
	require.Len(t, okRsp.Payload.Items, 2)
	require.EqualValues(t, 2, okRsp.Payload.Total)
	// checking if returned subnet-ids have expected values
	require.True(t,
		(okRsp.Payload.Items[0].LocalSubnets[0].ID == 1 && okRsp.Payload.Items[1].LocalSubnets[0].ID == 3) ||
			(okRsp.Payload.Items[0].LocalSubnets[0].ID == 3 && okRsp.Payload.Items[1].LocalSubnets[0].ID == 1))
	require.EqualValues(t, 24, okRsp.Payload.Items[1].Stats.(dbmodel.SubnetStats)["bar"])
	require.EqualValues(t, time.Time{}.Add(2*time.Hour), okRsp.Payload.Items[1].StatsCollectedAt)

	// get v6 subnets
	dhcpVer = 6
	params = dhcp.GetSubnetsParams{
		DhcpVersion: &dhcpVer,
	}
	rsp = rapi.GetSubnets(ctx, params)
	require.IsType(t, &dhcp.GetSubnetsOK{}, rsp)
	okRsp = rsp.(*dhcp.GetSubnetsOK)
	require.Len(t, okRsp.Payload.Items, 3)
	require.EqualValues(t, 3, okRsp.Payload.Total)
	// checking if returned subnet-ids have expected values
	require.ElementsMatch(t, []int64{2, 4, 21},
		[]int64{
			okRsp.Payload.Items[0].LocalSubnets[0].ID,
			okRsp.Payload.Items[1].LocalSubnets[0].ID,
			okRsp.Payload.Items[2].LocalSubnets[0].ID,
		})
	require.EqualValues(t, 4224, okRsp.Payload.Items[2].Stats.(dbmodel.SubnetStats)["baz"])
	require.EqualValues(t, time.Time{}.Add(3*time.Hour), okRsp.Payload.Items[2].StatsCollectedAt)
	require.EqualValues(t, 24, okRsp.Payload.Items[2].AddrUtilization)
	require.EqualValues(t, 42, okRsp.Payload.Items[2].PdUtilization)

	// get subnets by text '118.0.0/2'
	text := "118.0.0/2"
	params = dhcp.GetSubnetsParams{
		Text: &text,
	}
	rsp = rapi.GetSubnets(ctx, params)
	require.IsType(t, &dhcp.GetSubnetsOK{}, rsp)
	okRsp = rsp.(*dhcp.GetSubnetsOK)
	require.Len(t, okRsp.Payload.Items, 1)
	require.EqualValues(t, 1, okRsp.Payload.Total)
	require.Len(t, okRsp.Payload.Items[0].LocalSubnets, 1)
	require.Equal(t, a46.ID, okRsp.Payload.Items[0].LocalSubnets[0].AppID)
	require.Equal(t, a46.Daemons[0].ID, okRsp.Payload.Items[0].LocalSubnets[0].DaemonID)
	// checking if returned subnet-ids have expected values
	require.EqualValues(t, 3, okRsp.Payload.Items[0].LocalSubnets[0].ID)
	require.Nil(t, okRsp.Payload.Items[0].LocalSubnets[0].Stats)

	// get subnets by text '0.150-192.168'
	text = "0.150-192.168"
	params = dhcp.GetSubnetsParams{
		Text: &text,
	}
	rsp = rapi.GetSubnets(ctx, params)
	require.IsType(t, &dhcp.GetSubnetsOK{}, rsp)
	okRsp = rsp.(*dhcp.GetSubnetsOK)
	require.Len(t, okRsp.Payload.Items, 1)
	require.EqualValues(t, 1, okRsp.Payload.Total)
	require.Len(t, okRsp.Payload.Items[0].LocalSubnets, 1)
	require.Equal(t, a4.ID, okRsp.Payload.Items[0].LocalSubnets[0].AppID)
	require.Equal(t, a4.Daemons[0].ID, okRsp.Payload.Items[0].LocalSubnets[0].DaemonID)
	// checking if returned subnet-ids have expected values
	require.EqualValues(t, 1, okRsp.Payload.Items[0].LocalSubnets[0].ID)
}

// Test getting a subnet with a detailed DHCP configuration over the REST API.
func TestGetSubnet4(t *testing.T) {
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
	err = dhcp4.Configure(string(testutil.AllKeysDHCPv4))
	require.NoError(t, err)

	app, err := dhcp4.GetKea()
	require.NoError(t, err)

	// Populate subnets and shared networks from the Kea configuration.
	err = kea.CommitAppIntoDB(db, app, &storktest.FakeEventCenter{}, nil, dbmodel.NewDHCPOptionDefinitionLookup())
	require.NoError(t, err)

	subnets, err := dbmodel.GetSubnetsByPrefix(db, "192.0.0.0/8")
	require.NoError(t, err)
	require.Len(t, subnets, 1)

	// Get the subnet over the REST API.
	params := dhcp.GetSubnetParams{
		ID: subnets[0].ID,
	}
	rsp := rapi.GetSubnet(ctx, params)
	require.IsType(t, &dhcp.GetSubnetOK{}, rsp)
	okRsp := rsp.(*dhcp.GetSubnetOK)
	subnet := okRsp.Payload
	require.NotNil(t, subnet)

	require.Len(t, subnet.LocalSubnets, 1)
	ls := subnet.LocalSubnets[0]

	// Validate the pools.
	require.Len(t, ls.Pools, 2)
	require.Equal(t, "192.1.0.1-192.1.0.200", ls.Pools[0])
	require.Equal(t, "192.3.0.1-192.3.0.200", ls.Pools[1])

	// Validate subnet-level parameters
	require.NotNil(t, ls.KeaConfigSubnetParameters)
	keaParams := ls.KeaConfigSubnetParameters

	require.NotNil(t, keaParams.SubnetLevelParameters)
	subnetParams := keaParams.SubnetLevelParameters

	// 4o6-interface
	require.NotNil(t, subnetParams.FourOverSixInterface)
	require.Empty(t, *subnetParams.FourOverSixInterface)
	// 4o6-interface-id
	require.NotNil(t, subnetParams.FourOverSixInterfaceID)
	require.Empty(t, subnetParams.FourOverSixInterfaceID)
	// 4o6-subnet
	require.NotNil(t, subnetParams.FourOverSixSubnet)
	require.Equal(t, "2001:db8:1:1::/64", *subnetParams.FourOverSixSubnet)
	// allocator
	require.NotNil(t, subnetParams.Allocator)
	require.Equal(t, "iterative", *subnetParams.Allocator)
	// authoritative
	require.NotNil(t, subnetParams.Authoritative)
	require.False(t, *subnetParams.Authoritative)
	// boot-file-name
	require.NotNil(t, subnetParams.BootFileName)
	require.Empty(t, subnetParams.BootFileName)
	// client-class
	require.NotNil(t, subnetParams.ClientClass)
	require.Empty(t, subnetParams.ClientClass)
	// ddns-generated-prefix
	require.NotNil(t, subnetParams.DdnsGeneratedPrefix)
	require.Equal(t, "myhost", *subnetParams.DdnsGeneratedPrefix)
	// ddns-override-client-update
	require.NotNil(t, subnetParams.DdnsOverrideClientUpdate)
	require.False(t, *subnetParams.DdnsOverrideClientUpdate)
	// ddns-override-no-update
	require.NotNil(t, subnetParams.DdnsOverrideNoUpdate)
	require.False(t, *subnetParams.DdnsOverrideNoUpdate)
	// ddns-qualifying-suffix
	require.NotNil(t, subnetParams.DdnsQualifyingSuffix)
	require.Empty(t, *subnetParams.DdnsQualifyingSuffix)
	// ddns-replace-client-name
	require.NotNil(t, subnetParams.DdnsReplaceClientName)
	require.Equal(t, "never", *subnetParams.DdnsReplaceClientName)
	// ddns-send-updates
	require.NotNil(t, subnetParams.DdnsSendUpdates)
	require.True(t, *subnetParams.DdnsSendUpdates)
	// ddns-update-on-renew
	require.NotNil(t, subnetParams.DdnsUpdateOnRenew)
	require.True(t, *subnetParams.DdnsUpdateOnRenew)
	// ddns-use-conflict-resolution
	require.NotNil(t, subnetParams.DdnsUseConflictResolution)
	require.True(t, *subnetParams.DdnsUseConflictResolution)
	// hostname-char-replacement
	require.NotNil(t, subnetParams.HostnameCharReplacement)
	require.Equal(t, "x", *subnetParams.HostnameCharReplacement)
	// hostname-char-set
	require.NotNil(t, subnetParams.HostnameCharSet)
	require.Equal(t, "[^A-Za-z0-9.-]", *subnetParams.HostnameCharSet)
	// interface
	require.NotNil(t, subnetParams.Interface)
	require.Equal(t, "eth0", *subnetParams.Interface)
	// match-client-id
	require.NotNil(t, subnetParams.MatchClientID)
	require.True(t, *subnetParams.MatchClientID)
	// next-server
	require.NotNil(t, subnetParams.NextServer)
	require.Equal(t, "0.0.0.0", *subnetParams.NextServer)
	// store-extended-info
	require.NotNil(t, subnetParams.StoreExtendedInfo)
	require.True(t, *subnetParams.StoreExtendedInfo)
	// rebind-timer
	require.NotNil(t, subnetParams.RebindTimer)
	require.EqualValues(t, 40, *subnetParams.RebindTimer)
	// relay
	require.NotNil(t, subnetParams.Relay)
	require.Len(t, subnetParams.Relay.IPAddresses, 1)
	require.Equal(t, "192.168.56.1", subnetParams.Relay.IPAddresses[0])
	// renew-timer
	require.NotNil(t, subnetParams.RenewTimer)
	require.EqualValues(t, 30, *subnetParams.RenewTimer)
	// reservation-mode
	require.NotNil(t, subnetParams.ReservationMode)
	require.Equal(t, "all", *subnetParams.ReservationMode)
	// reservations-global
	require.NotNil(t, subnetParams.ReservationsGlobal)
	require.False(t, *subnetParams.ReservationsGlobal)
	// reservations-in-subnet
	require.NotNil(t, subnetParams.ReservationsInSubnet)
	require.True(t, *subnetParams.ReservationsInSubnet)
	// reservations-out-of-pool
	require.NotNil(t, subnetParams.ReservationsOutOfPool)
	require.False(t, *subnetParams.ReservationsOutOfPool)
	// calculate-tee-times
	require.NotNil(t, subnetParams.CalculateTeeTimes)
	require.True(t, *subnetParams.CalculateTeeTimes)
	// t1-percent
	require.NotNil(t, subnetParams.T1Percent)
	require.EqualValues(t, 0.5, *subnetParams.T1Percent)
	// t2-percent
	require.NotNil(t, subnetParams.T2Percent)
	require.EqualValues(t, 0.75, *subnetParams.T2Percent)
	// cache-max-age
	require.NotNil(t, subnetParams.CacheMaxAge)
	require.EqualValues(t, 1000, *subnetParams.CacheMaxAge)
	// require-client-classes
	require.Len(t, subnetParams.RequireClientClasses, 1)
	require.Equal(t, "late", subnetParams.RequireClientClasses[0])
	// server-hostname
	require.NotNil(t, subnetParams.ServerHostname)
	require.Empty(t, subnetParams.ServerHostname)
	// valid-lifetime
	require.NotNil(t, subnetParams.ValidLifetime)
	require.EqualValues(t, 6000, *subnetParams.ValidLifetime)
	// min-valid-lifetime
	require.NotNil(t, subnetParams.MinValidLifetime)
	require.EqualValues(t, 4000, *subnetParams.MinValidLifetime)
	// max-valid-lifetime
	require.NotNil(t, subnetParams.MaxValidLifetime)
	require.EqualValues(t, 8000, *subnetParams.MaxValidLifetime)

	// Validate the options.
	require.NotEmpty(t, subnetParams.OptionsHash)
	require.Len(t, subnetParams.Options, 1)
	require.False(t, subnetParams.Options[0].AlwaysSend)
	require.EqualValues(t, 3, subnetParams.Options[0].Code)
	require.Empty(t, subnetParams.Options[0].Encapsulate)
	require.Len(t, subnetParams.Options[0].Fields, 1)
	require.Equal(t, dhcpmodel.IPv4AddressField, subnetParams.Options[0].Fields[0].FieldType)
	require.Len(t, subnetParams.Options[0].Fields[0].Values, 1)
	require.Equal(t, "192.0.3.1", subnetParams.Options[0].Fields[0].Values[0])
	require.EqualValues(t, storkutil.IPv4, subnetParams.Options[0].Universe)

	// Validate shared-network-level parameters
	networkParams := ls.KeaConfigSubnetParameters.SharedNetworkLevelParameters
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
	require.NotNil(t, networkParams.ClientClass)
	require.Empty(t, networkParams.ClientClass)
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
	require.NotNil(t, networkParams.DdnsQualifyingSuffix)
	require.Empty(t, *networkParams.DdnsQualifyingSuffix)
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
	require.NotNil(t, networkParams.ServerHostname)
	require.Empty(t, networkParams.ServerHostname)
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
	globalParams := ls.KeaConfigSubnetParameters.GlobalParameters
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
	require.NotNil(t, globalParams.DdnsQualifyingSuffix)
	require.Empty(t, *globalParams.DdnsQualifyingSuffix)
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
	require.NotNil(t, globalParams.ServerHostname)
	require.Empty(t, globalParams.ServerHostname)
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

// Test getting an IPv4 subnet over the REST API when the DHCP configuration
// contains no explicit parameters.
func TestGetSubnet4MinimalParameters(t *testing.T) {
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

	subnets, err := dbmodel.GetSubnetsByPrefix(db, "192.0.2.0/24")
	require.NoError(t, err)
	require.Len(t, subnets, 1)

	params := dhcp.GetSubnetParams{
		ID: subnets[0].ID,
	}
	rsp := rapi.GetSubnet(ctx, params)
	require.IsType(t, &dhcp.GetSubnetOK{}, rsp)
	okRsp := rsp.(*dhcp.GetSubnetOK)
	subnet := okRsp.Payload
	require.NotNil(t, subnet)

	require.Len(t, subnet.LocalSubnets, 1)
	ls := subnet.LocalSubnets[0]

	require.NotNil(t, ls.KeaConfigSubnetParameters)
}

// Test getting a subnet with a detailed DHCP configuration over the REST API.
func TestGetSubnet6(t *testing.T) {
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
	err = dhcp6.Configure(string(testutil.AllKeysDHCPv6))
	require.NoError(t, err)

	app, err := dhcp6.GetKea()
	require.NoError(t, err)

	// Populate subnets and shared networks from the Kea configuration.
	err = kea.CommitAppIntoDB(db, app, &storktest.FakeEventCenter{}, nil, dbmodel.NewDHCPOptionDefinitionLookup())
	require.NoError(t, err)

	subnets, err := dbmodel.GetSubnetsByPrefix(db, "2001:db8::/32")
	require.NoError(t, err)
	require.Len(t, subnets, 1)

	// Get the subnet over the REST API.
	params := dhcp.GetSubnetParams{
		ID: subnets[0].ID,
	}
	rsp := rapi.GetSubnet(ctx, params)
	require.IsType(t, &dhcp.GetSubnetOK{}, rsp)
	okRsp := rsp.(*dhcp.GetSubnetOK)
	subnet := okRsp.Payload
	require.NotNil(t, subnet)

	require.Len(t, subnet.LocalSubnets, 1)
	ls := subnet.LocalSubnets[0]

	// Validate subnet-level parameters
	require.NotNil(t, ls.KeaConfigSubnetParameters)
	keaParams := ls.KeaConfigSubnetParameters

	require.NotNil(t, keaParams.SubnetLevelParameters)
	subnetParams := keaParams.SubnetLevelParameters

	// pd-allocator
	require.NotNil(t, subnetParams.PdAllocator)
	require.Equal(t, "iterative", *subnetParams.Allocator)
	// preferred-lifetime
	require.NotNil(t, subnetParams.PreferredLifetime)
	require.EqualValues(t, 2000, *subnetParams.PreferredLifetime)
	// min-preferred-lifetime
	require.NotNil(t, subnetParams.MinPreferredLifetime)
	require.EqualValues(t, 1500, *subnetParams.MinPreferredLifetime)
	// max-preferred-lifetime
	require.NotNil(t, subnetParams.MaxPreferredLifetime)
	require.EqualValues(t, 2500, *subnetParams.MaxPreferredLifetime)

	// Validate the options.
	require.NotEmpty(t, subnetParams.OptionsHash)
	require.Len(t, subnetParams.Options, 1)
	require.False(t, subnetParams.Options[0].AlwaysSend)
	require.EqualValues(t, 7, subnetParams.Options[0].Code)
	require.Empty(t, subnetParams.Options[0].Encapsulate)
	require.Len(t, subnetParams.Options[0].Fields, 1)
	require.Equal(t, dhcpmodel.BinaryField, subnetParams.Options[0].Fields[0].FieldType)
	require.Len(t, subnetParams.Options[0].Fields[0].Values, 1)
	require.Equal(t, "f0", subnetParams.Options[0].Fields[0].Values[0])
	require.EqualValues(t, storkutil.IPv6, subnetParams.Options[0].Universe)

	// Validate shared-network-level parameters
	networkParams := ls.KeaConfigSubnetParameters.SharedNetworkLevelParameters
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
	globalParams := ls.KeaConfigSubnetParameters.GlobalParameters
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

// Test getting an IPv6 subnet over the REST API when the DHCP configuration
// contains no explicit parameters.
func TestGetSubnet6MinimalParameters(t *testing.T) {
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

	subnets, err := dbmodel.GetSubnetsByPrefix(db, "3000::/64")
	require.NoError(t, err)
	require.Len(t, subnets, 1)

	// Get the subnet over the REST API.
	params := dhcp.GetSubnetParams{
		ID: subnets[0].ID,
	}
	rsp := rapi.GetSubnet(ctx, params)
	require.IsType(t, &dhcp.GetSubnetOK{}, rsp)
	okRsp := rsp.(*dhcp.GetSubnetOK)
	subnet := okRsp.Payload
	require.NotNil(t, subnet)

	require.Len(t, subnet.LocalSubnets, 1)
	ls := subnet.LocalSubnets[0]

	require.NotNil(t, ls.KeaConfigSubnetParameters)
}

// Test that the HTTP Not Found status is returned when getting a subnet which
// doesn't exist.
func TestGetSubnetNonExisting(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	settings := RestAPISettings{}
	fa := agentcommtest.NewFakeAgents(nil, nil)
	fec := &storktest.FakeEventCenter{}
	fd := &storktest.FakeDispatcher{}
	rapi, err := NewRestAPI(&settings, dbSettings, db, fa, fec, nil, fd, nil)
	require.NoError(t, err)
	ctx := context.Background()

	params := dhcp.GetSubnetParams{
		ID: 1000000,
	}
	rsp := rapi.GetSubnet(ctx, params)
	require.IsType(t, &dhcp.GetSubnetDefault{}, rsp)
	defaultRsp := rsp.(*dhcp.GetSubnetDefault)
	require.Equal(t, http.StatusNotFound, getStatusCode(*defaultRsp))
}

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
	require.Len(t, okRsp.Payload.Items, 3)
	require.EqualValues(t, 3, okRsp.Payload.Total)
	for _, net := range okRsp.Payload.Items {
		require.Contains(t, []string{"frog", "mouse", "fox"}, net.Name)
		switch net.Name {
		case "frog":
			require.Len(t, net.Subnets, 1)
			require.EqualValues(t, 1, net.Subnets[0].ID)
			require.Equal(t, "192.1.0.0/24", net.Subnets[0].Subnet)
		case "mouse":
			require.Len(t, net.Subnets, 2)
			// subnets should be sorted by prefix, not by ID
			require.EqualValues(t, 3, net.Subnets[0].ID)
			require.Equal(t, "192.2.0.0/24", net.Subnets[0].Subnet)
			require.EqualValues(t, 2, net.Subnets[1].ID)
			require.Equal(t, "192.3.0.0/24", net.Subnets[1].Subnet)
		case "fox":
			require.Len(t, net.Subnets, 2)
			// subnets should be sorted by prefix, not by ID
			require.EqualValues(t, 5, net.Subnets[0].ID)
			require.Equal(t, "5001:db8:1::/64", net.Subnets[0].Subnet)
			require.EqualValues(t, 4, net.Subnets[1].ID)
			require.Equal(t, "6001:db8:1::/64", net.Subnets[1].Subnet)
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
