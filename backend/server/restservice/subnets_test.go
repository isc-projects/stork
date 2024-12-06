package restservice

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	dhcpmodel "isc.org/stork/datamodel/dhcp"
	agentcommtest "isc.org/stork/server/agentcomm/test"
	"isc.org/stork/server/apps"
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
	dbmodel.CommitNetworksIntoDB(db, []dbmodel.SharedNetwork{}, subnets)

	subnets, err = dbmodel.GetSubnetsByPrefix(db, "3001:db8:1::/64")
	require.NoError(t, err)
	require.Len(t, subnets, 1)
	subnets[0].Stats = dbmodel.SubnetStats{
		"baz": 4224,
	}
	subnets[0].StatsCollectedAt = time.Time{}.Add(3 * time.Hour)
	subnets[0].AddrUtilization = 240
	subnets[0].PdUtilization = 420
	dbmodel.CommitNetworksIntoDB(db, []dbmodel.SharedNetwork{}, subnets)

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
	require.EqualValues(t, map[string]any(nil), okRsp.Payload.Items[0].Stats)
	require.Nil(t, okRsp.Payload.Items[0].StatsCollectedAt)

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
	require.NotNil(t, okRsp.Payload.Items[1].StatsCollectedAt)
	require.EqualValues(t, time.Time{}.Add(2*time.Hour), *okRsp.Payload.Items[1].StatsCollectedAt)

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
	require.NotNil(t, okRsp.Payload.Items[2].StatsCollectedAt)
	require.EqualValues(t, time.Time{}.Add(3*time.Hour), *okRsp.Payload.Items[2].StatsCollectedAt)
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
	err = dhcp4.Configure(string(testutil.AllKeysDHCPv4JSON))
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

	// Validate the name.
	require.NotNil(t, ls.UserContext)
	require.IsType(t, map[string]any(nil), ls.UserContext)
	require.EqualValues(t, 42, ls.UserContext.(map[string]any)["answer"])

	// Validate the pools.
	require.Len(t, ls.Pools, 2)
	require.NotNil(t, ls.Pools[0].Pool)

	require.Equal(t, "192.1.0.1-192.1.0.200", *ls.Pools[0].Pool)
	require.NotNil(t, ls.Pools[0].KeaConfigPoolParameters)
	require.Equal(t, "phones_server1", *ls.Pools[0].KeaConfigPoolParameters.ClientClass)
	require.Len(t, ls.Pools[0].KeaConfigPoolParameters.RequireClientClasses, 1)
	require.Len(t, ls.Pools[0].KeaConfigPoolParameters.DHCPOptions.Options, 1)
	require.False(t, ls.Pools[0].KeaConfigPoolParameters.DHCPOptions.Options[0].AlwaysSend)
	require.EqualValues(t, 3, ls.Pools[0].KeaConfigPoolParameters.DHCPOptions.Options[0].Code)
	require.Empty(t, ls.Pools[0].KeaConfigPoolParameters.DHCPOptions.Options[0].Encapsulate)
	require.Len(t, ls.Pools[0].KeaConfigPoolParameters.DHCPOptions.Options[0].Fields, 1)
	require.Equal(t, dhcpmodel.IPv4AddressField, ls.Pools[0].KeaConfigPoolParameters.DHCPOptions.Options[0].Fields[0].FieldType)
	require.Len(t, ls.Pools[0].KeaConfigPoolParameters.DHCPOptions.Options[0].Fields[0].Values, 1)
	require.Equal(t, "192.0.3.10", ls.Pools[0].KeaConfigPoolParameters.DHCPOptions.Options[0].Fields[0].Values[0])
	require.EqualValues(t, storkutil.IPv4, ls.Pools[0].KeaConfigPoolParameters.DHCPOptions.Options[0].Universe)
	require.NotNil(t, ls.Pools[0].KeaConfigPoolParameters.PoolID)
	require.EqualValues(t, 7, *ls.Pools[0].KeaConfigPoolParameters.PoolID)

	require.NotNil(t, ls.Pools[1].Pool)
	require.Equal(t, "192.3.0.1-192.3.0.200", *ls.Pools[1].Pool)
	require.Equal(t, "phones_server2", *ls.Pools[1].KeaConfigPoolParameters.ClientClass)
	require.Empty(t, ls.Pools[1].KeaConfigPoolParameters.RequireClientClasses)

	// Validate subnet-level parameters
	require.NotNil(t, ls.KeaConfigSubnetParameters)
	keaParams := ls.KeaConfigSubnetParameters

	require.NotNil(t, keaParams.SubnetLevelParameters)
	subnetParams := keaParams.SubnetLevelParameters

	// 4o6-interface
	require.Nil(t, subnetParams.FourOverSixInterface)
	// 4o6-interface-id
	require.Nil(t, subnetParams.FourOverSixInterfaceID)
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
	require.Nil(t, subnetParams.BootFileName)
	// client-class
	require.Nil(t, subnetParams.ClientClass)
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
	require.Nil(t, subnetParams.DdnsQualifyingSuffix)
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
	// ddns-conflict-resolution-mode
	require.NotNil(t, subnetParams.DdnsConflictResolutionMode)
	require.Equal(t, "check-with-dhcid", *subnetParams.DdnsConflictResolutionMode)
	// ddns-ttl-percent
	require.NotNil(t, subnetParams.DdnsTTLPercent)
	require.EqualValues(t, float32(0.65), *subnetParams.DdnsTTLPercent)
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
	require.Nil(t, subnetParams.ServerHostname)
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
	require.NotNil(t, globalParams.DdnsConflictResolutionMode)
	require.Equal(t, "check-with-dhcid", *globalParams.DdnsConflictResolutionMode)
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

	require.EqualValues(t, subnets[0].SharedNetwork.ID, subnet.SharedNetworkID)
	require.Equal(t, "foo", subnet.SharedNetwork)

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
	err = dhcp6.Configure(string(testutil.AllKeysDHCPv6JSON))
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

	// Validate the name.
	require.NotNil(t, ls.UserContext)
	require.IsType(t, map[string]any(nil), ls.UserContext)
	require.EqualValues(t, 42, ls.UserContext.(map[string]any)["answer"])

	// Validate the prefix delegation pools.
	require.Len(t, ls.PrefixDelegationPools, 1)

	require.NotNil(t, ls.PrefixDelegationPools[0].Prefix)
	require.Equal(t, "2001:db8:1::/48", *ls.PrefixDelegationPools[0].Prefix)
	require.NotNil(t, ls.PrefixDelegationPools[0].DelegatedLength)
	require.EqualValues(t, 64, *ls.PrefixDelegationPools[0].DelegatedLength)
	require.Equal(t, "2001:db8:1::/72", ls.PrefixDelegationPools[0].ExcludedPrefix)

	require.NotNil(t, ls.PrefixDelegationPools[0].KeaConfigPoolParameters)

	poolParams := ls.PrefixDelegationPools[0].KeaConfigPoolParameters
	require.Equal(t, "phones_server1", *poolParams.ClientClass)
	require.Len(t, poolParams.RequireClientClasses, 1)
	require.NotNil(t, poolParams.PoolID)
	require.EqualValues(t, 2, *poolParams.PoolID)

	// DHCP options in a prefix pool.
	require.Len(t, poolParams.DHCPOptions.Options, 1)
	require.False(t, poolParams.DHCPOptions.Options[0].AlwaysSend)
	require.EqualValues(t, 7, poolParams.DHCPOptions.Options[0].Code)
	require.Empty(t, poolParams.DHCPOptions.Options[0].Encapsulate)
	require.Len(t, poolParams.DHCPOptions.Options[0].Fields, 1)
	require.Equal(t, dhcpmodel.BinaryField, poolParams.DHCPOptions.Options[0].Fields[0].FieldType)
	require.Len(t, poolParams.DHCPOptions.Options[0].Fields[0].Values, 1)
	require.Equal(t, "cafe", poolParams.DHCPOptions.Options[0].Fields[0].Values[0])
	require.EqualValues(t, storkutil.IPv6, ls.PrefixDelegationPools[0].KeaConfigPoolParameters.DHCPOptions.Options[0].Universe)

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

// Test that the zero statistics collected timestamp is omitted in the
// serialized JSON.
func TestSubnetToRestAPIEmptyStatsCollectedTimestamp(t *testing.T) {
	// Arrange
	settings := RestAPISettings{}
	rapi, _ := NewRestAPI(&settings)

	subnetDB := &dbmodel.Subnet{}

	// Act
	subnetAPI := rapi.convertSubnetToRestAPI(subnetDB)
	subnetJSON, err := json.Marshal(subnetAPI)

	// Assert
	require.NotNil(t, subnetAPI)
	require.NoError(t, err)
	require.NotContains(t, string(subnetJSON), "statsCollectedAt")
}

// Test the calls for creating new transaction and creating a subnet.
func TestCreateSubnet4BeginSubmit(t *testing.T) {
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
					"name": "foo"
				}
			],
			"subnet4": [
				{
					"id": 100,
					"subnet": "192.0.100.0/24"
				},
				{
					"id": 101,
					"subnet": "192.0.101.0/24"
				},
				{
					"id": 103,
					"subnet": "192.0.102.0/24"
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

	// Begin transaction.
	params := dhcp.CreateSubnetBeginParams{}
	rsp := rapi.CreateSubnetBegin(ctx, params)
	require.IsType(t, &dhcp.CreateSubnetBeginOK{}, rsp)
	okRsp := rsp.(*dhcp.CreateSubnetBeginOK)
	contents := okRsp.Payload

	// Make sure the server returned transaction ID, subnet, daemons, shared networks and client classes.
	transactionID := contents.ID
	require.NotZero(t, transactionID)
	require.Len(t, contents.Daemons, 2)
	require.Len(t, contents.SharedNetworks4, 1)
	require.Empty(t, contents.SharedNetworks6)
	require.Len(t, contents.ClientClasses, 2)

	keaConfigSubnetParameters := &models.KeaConfigSubnetParameters{
		SubnetLevelParameters: &models.KeaConfigSubnetDerivedParameters{
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
	params2 := dhcp.CreateSubnetSubmitParams{
		ID: transactionID,
		Subnet: &models.Subnet{
			ID:              0,
			Subnet:          "192.0.2.0/24",
			SharedNetworkID: sharedNetworks[0].ID,
			SharedNetwork:   "foo",
			LocalSubnets: []*models.LocalSubnet{
				{
					ID:       1,
					DaemonID: dbapps[0].Daemons[0].ID,
					Pools: []*models.Pool{
						{
							Pool: storkutil.Ptr("192.0.2.10-192.0.2.20"),
							KeaConfigPoolParameters: &models.KeaConfigPoolParameters{
								KeaConfigClientClassParameters: models.KeaConfigClientClassParameters{
									ClientClass:          storkutil.Ptr("foo"),
									RequireClientClasses: []string{"foo", "bar"},
								},
								DHCPOptions: models.DHCPOptions{
									Options: []*models.DHCPOption{
										{
											AlwaysSend: false,
											Code:       3,
											Fields: []*models.DHCPOptionField{
												{
													FieldType: "ipv4-address",
													Values:    []string{"192.0.2.2"},
												},
											},
											Universe: 4,
										},
									},
								},
							},
						},
						{
							Pool: storkutil.Ptr("192.0.2.30-192.0.2.40"),
						},
					},
					UserContext: map[string]any{
						"answer": 42,
					},
					KeaConfigSubnetParameters: keaConfigSubnetParameters,
				},
				{
					ID:       1,
					DaemonID: dbapps[1].Daemons[0].ID,
					Pools: []*models.Pool{
						{
							Pool: storkutil.Ptr("192.0.2.10-192.0.2.20"),
							KeaConfigPoolParameters: &models.KeaConfigPoolParameters{
								KeaConfigClientClassParameters: models.KeaConfigClientClassParameters{
									ClientClass:          storkutil.Ptr("foo"),
									RequireClientClasses: []string{"foo", "bar"},
								},
								DHCPOptions: models.DHCPOptions{
									Options: []*models.DHCPOption{
										{
											AlwaysSend: false,
											Code:       3,
											Fields: []*models.DHCPOptionField{
												{
													FieldType: "ipv4-address",
													Values:    []string{"192.0.2.2"},
												},
											},
											Universe: 4,
										},
									},
								},
							},
						},
						{
							Pool: storkutil.Ptr("192.0.2.30-192.0.2.40"),
						},
					},
					UserContext: map[string]any{
						"answer": 42,
					},
					KeaConfigSubnetParameters: keaConfigSubnetParameters,
				},
			},
		},
	}
	rsp2 := rapi.CreateSubnetSubmit(ctx, params2)
	require.IsType(t, &dhcp.CreateSubnetSubmitOK{}, rsp2)
	require.NotZero(t, rsp2.(*dhcp.CreateSubnetSubmitOK).Payload.SubnetID)

	// It should result in sending commands to two Kea servers. Each server
	// receives the subnet4-add command.
	require.Len(t, fa.RecordedCommands, 6)

	for i, c := range fa.RecordedCommands {
		switch i {
		case 0, 2:
			require.JSONEq(t,
				`{
				"command": "subnet4-add",
				"service": [ "dhcp4" ],
				"arguments": {
					"subnet4": [
						{
							"id": 104,
							"subnet": "192.0.2.0/24",

							"pools": [
								{
									"pool": "192.0.2.10-192.0.2.20",
									"client-class": "foo",
									"require-client-classes": [ "foo", "bar" ],
									"option-data": [
										{
											"code": 3,
											"csv-format": true,
											"data": "192.0.2.2",
											"space": "dhcp4"
										}
									]
								},
								{
									"pool": "192.0.2.30-192.0.2.40"
								}
							],

							"user-context": {
								"answer": 42
							},

							"cache-max-age": 1000,
							"cache-threshold": 0.25,
							"client-class": "foo",
							"require-client-classes": [ "bar" ],
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
							"4o6-interface": "eth0",
							"4o6-interface-id": "ifaceid",
							"4o6-subnet": "2001:db8:1::/64",
							"hostname-char-replacement": "a",
							"hostname-char-set": "b",
							"reservation-mode": "in-pool",
							"reservations-global": false,
							"reservations-in-subnet": true,
							"reservations-out-of-pool": false,
							"calculate-tee-times": true,
							"rebind-timer": 2000,
							"renew-timer": 1000,
							"t1-percent": 0.25,
							"t2-percent": 0.50,
							"max-valid-lifetime": 5000,
							"min-valid-lifetime": 4000,
							"valid-lifetime": 4500,
							"allocator": "random",
							"authoritative": true,
							"boot-file-name": "/tmp/filename",
							"interface": "eth0",
							"match-client-id": true,
							"next-server": "192.0.2.1",
							"relay": {
								"ip-addresses": [ "10.1.1.1" ]
							},
							"server-hostname": "myhost.example.org",
							"store-extended-info": true,
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
					]
				}
			}`,
				c.Marshal())
		case 1, 3:
			require.JSONEq(t,
				`{
						"command": "network4-subnet-add",
						"service": [ "dhcp4" ],
						"arguments": {
							"name": "foo",
							"id": 104
						}
				}`,
				c.Marshal())
		default:
			require.JSONEq(t,
				`{
						"command": "config-write",
						"service": [ "dhcp4" ]
				}`,
				c.Marshal())
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

	// Make sure that the updated host has been stored in the database.
	returnedSubnets, err := dbmodel.GetSubnetsByPrefix(db, "192.0.2.0/24")
	require.NoError(t, err)
	require.Len(t, returnedSubnets, 1)
	require.NotNil(t, returnedSubnets[0].SharedNetwork)

	require.Len(t, returnedSubnets[0].LocalSubnets, 2)
	for _, ls := range returnedSubnets[0].LocalSubnets {
		require.Len(t, ls.AddressPools, 2)
		require.Equal(t, "192.0.2.10", ls.AddressPools[0].LowerBound)
		require.Equal(t, "192.0.2.20", ls.AddressPools[0].UpperBound)
		require.Equal(t, "192.0.2.30", ls.AddressPools[1].LowerBound)
		require.Equal(t, "192.0.2.40", ls.AddressPools[1].UpperBound)

		require.NotNil(t, ls.UserContext)
		require.EqualValues(t, 42, ls.UserContext["answer"])

		require.NotNil(t, ls.KeaParameters)
		require.NotNil(t, ls.KeaParameters.CacheMaxAge)
		require.EqualValues(t, 1000, *ls.KeaParameters.CacheMaxAge)
		require.NotNil(t, ls.KeaParameters.CacheThreshold)
		require.EqualValues(t, 0.25, *ls.KeaParameters.CacheThreshold)
		require.NotNil(t, ls.KeaParameters.ClientClass)
		require.Equal(t, "foo", *ls.KeaParameters.ClientClass)
		require.Len(t, ls.KeaParameters.RequireClientClasses, 1)
		require.EqualValues(t, "bar", ls.KeaParameters.RequireClientClasses[0])
		require.NotNil(t, ls.KeaParameters.DDNSGeneratedPrefix)
		require.Equal(t, "abc", *ls.KeaParameters.DDNSGeneratedPrefix)
		require.NotNil(t, ls.KeaParameters.DDNSOverrideClientUpdate)
		require.True(t, *ls.KeaParameters.DDNSOverrideClientUpdate)
		require.NotNil(t, ls.KeaParameters.DDNSOverrideNoUpdate)
		require.False(t, *ls.KeaParameters.DDNSOverrideNoUpdate)
		require.NotNil(t, ls.KeaParameters.DDNSQualifyingSuffix)
		require.Equal(t, "example.org", *ls.KeaParameters.DDNSQualifyingSuffix)
		require.NotNil(t, ls.KeaParameters.DDNSReplaceClientName)
		require.Equal(t, "never", *ls.KeaParameters.DDNSReplaceClientName)
		require.NotNil(t, ls.KeaParameters.DDNSSendUpdates)
		require.True(t, *ls.KeaParameters.DDNSSendUpdates)
		require.NotNil(t, ls.KeaParameters.DDNSUpdateOnRenew)
		require.True(t, *ls.KeaParameters.DDNSUpdateOnRenew)
		require.NotNil(t, ls.KeaParameters.DDNSUseConflictResolution)
		require.True(t, *ls.KeaParameters.DDNSUseConflictResolution)
		require.NotNil(t, ls.KeaParameters.DDNSTTLPercent)
		require.EqualValues(t, float32(0.65), *ls.KeaParameters.DDNSTTLPercent)
		require.NotNil(t, ls.KeaParameters.FourOverSixInterface)
		require.Equal(t, "eth0", *ls.KeaParameters.FourOverSixInterface)
		require.NotNil(t, ls.KeaParameters.FourOverSixInterfaceID)
		require.Equal(t, "ifaceid", *ls.KeaParameters.FourOverSixInterfaceID)
		require.NotNil(t, ls.KeaParameters.FourOverSixSubnet)
		require.Equal(t, "2001:db8:1::/64", *ls.KeaParameters.FourOverSixSubnet)
		require.NotNil(t, ls.KeaParameters.HostnameCharReplacement)
		require.Equal(t, "a", *ls.KeaParameters.HostnameCharReplacement)
		require.NotNil(t, ls.KeaParameters.HostnameCharSet)
		require.Equal(t, "b", *ls.KeaParameters.HostnameCharSet)
		require.NotNil(t, ls.KeaParameters.ReservationMode)
		require.Equal(t, "in-pool", *ls.KeaParameters.ReservationMode)
		require.NotNil(t, ls.KeaParameters.ReservationsGlobal)
		require.False(t, *ls.KeaParameters.ReservationsGlobal)
		require.NotNil(t, ls.KeaParameters.ReservationsInSubnet)
		require.True(t, *ls.KeaParameters.ReservationsInSubnet)
		require.NotNil(t, ls.KeaParameters.ReservationsOutOfPool)
		require.False(t, *ls.KeaParameters.ReservationsOutOfPool)
		require.NotNil(t, ls.KeaParameters.CalculateTeeTimes)
		require.True(t, *ls.KeaParameters.CalculateTeeTimes)
		require.NotNil(t, ls.KeaParameters.RebindTimer)
		require.EqualValues(t, 2000, *ls.KeaParameters.RebindTimer)
		require.NotNil(t, ls.KeaParameters.RenewTimer)
		require.EqualValues(t, 1000, *ls.KeaParameters.RenewTimer)
		require.NotNil(t, ls.KeaParameters.T1Percent)
		require.EqualValues(t, 0.25, *ls.KeaParameters.T1Percent)
		require.NotNil(t, ls.KeaParameters.T2Percent)
		require.EqualValues(t, 0.50, *ls.KeaParameters.T2Percent)
		require.NotNil(t, ls.KeaParameters.MaxValidLifetime)
		require.EqualValues(t, 5000, *ls.KeaParameters.MaxValidLifetime)
		require.NotNil(t, ls.KeaParameters.MinValidLifetime)
		require.EqualValues(t, 4000, *ls.KeaParameters.MinValidLifetime)
		require.NotNil(t, ls.KeaParameters.ValidLifetime)
		require.EqualValues(t, 4500, *ls.KeaParameters.ValidLifetime)
		require.NotNil(t, ls.KeaParameters.Allocator)
		require.Equal(t, "random", *ls.KeaParameters.Allocator)
		require.NotNil(t, ls.KeaParameters.Authoritative)
		require.True(t, *ls.KeaParameters.Authoritative)
		require.NotNil(t, ls.KeaParameters.Authoritative)
		require.Equal(t, "/tmp/filename", *ls.KeaParameters.BootFileName)
		require.NotNil(t, ls.KeaParameters.Interface)
		require.Equal(t, "eth0", *ls.KeaParameters.Interface)
		require.NotNil(t, ls.KeaParameters.MatchClientID)
		require.True(t, *ls.KeaParameters.MatchClientID)
		require.NotNil(t, ls.KeaParameters.NextServer)
		require.Equal(t, "192.0.2.1", *ls.KeaParameters.NextServer)
		require.NotNil(t, ls.KeaParameters.Relay)
		require.Len(t, ls.KeaParameters.Relay.IPAddresses, 1)
		require.Equal(t, "10.1.1.1", ls.KeaParameters.Relay.IPAddresses[0])
		require.NotNil(t, ls.KeaParameters.ServerHostname)
		require.Equal(t, "myhost.example.org", *ls.KeaParameters.ServerHostname)
		require.NotNil(t, ls.KeaParameters.StoreExtendedInfo)
		require.True(t, *ls.KeaParameters.StoreExtendedInfo)

		// DHCP options
		require.Len(t, ls.DHCPOptionSet.Options, 1)
		require.True(t, ls.DHCPOptionSet.Options[0].AlwaysSend)
		require.EqualValues(t, 3, ls.DHCPOptionSet.Options[0].Code)
		require.Len(t, ls.DHCPOptionSet.Options[0].Fields, 1)
		require.Equal(t, dhcpmodel.IPv4AddressField, ls.DHCPOptionSet.Options[0].Fields[0].FieldType)
		require.Len(t, ls.DHCPOptionSet.Options[0].Fields[0].Values, 1)
		require.Equal(t, "192.0.2.1", ls.DHCPOptionSet.Options[0].Fields[0].Values[0])
		require.Equal(t, dhcpmodel.DHCPv4OptionSpace, ls.DHCPOptionSet.Options[0].Space)
		require.NotEmpty(t, ls.DHCPOptionSet.Hash)
	}
}

// Test error case when a user attempts to begin a new transaction when
// there are no servers with subnet_cmds hook library found.
func TestCreateSubnetBeginSubmitNoServers(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	serverConfig := `{
		"Dhcp4": {}
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

	lookup := dbmodel.NewDHCPOptionDefinitionLookup()
	require.NotNil(t, lookup)

	fa := agentcommtest.NewFakeAgents(nil, nil)

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
	params := dhcp.CreateSubnetBeginParams{}
	rsp := rapi.CreateSubnetBegin(ctx, params)
	require.IsType(t, &dhcp.CreateSubnetBeginDefault{}, rsp)
	defaultRsp := rsp.(*dhcp.CreateSubnetBeginDefault)
	require.Equal(t, http.StatusBadRequest, getStatusCode(*defaultRsp))
	require.Equal(t, "Unable to begin transaction because there are no Kea servers with subnet_cmds hooks library available", *defaultRsp.Payload.Message)
}

// Test error cases for submitting new subnet.
func TestCreateSubnetBeginSubmitError(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Setup fake agents that return an error in response to reservation-add
	// command.
	fa := agentcommtest.NewFakeAgents(func(callNo int, cmdResponses []interface{}) {
		mockStatusError("subnet4-add", cmdResponses)
	}, nil)
	require.NotNil(t, fa)

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
					"name": "foo"
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

	// Begin transaction.
	params := dhcp.CreateSubnetBeginParams{}
	rsp := rapi.CreateSubnetBegin(ctx, params)
	require.IsType(t, &dhcp.CreateSubnetBeginOK{}, rsp)
	okRsp := rsp.(*dhcp.CreateSubnetBeginOK)
	contents := okRsp.Payload

	// Make sure the server returned transaction ID, subnet, daemons, shared networks and client classes.
	transactionID := contents.ID
	require.NotZero(t, transactionID)
	require.Len(t, contents.Daemons, 2)
	require.Len(t, contents.SharedNetworks4, 1)
	require.Empty(t, contents.SharedNetworks6)
	require.Len(t, contents.ClientClasses, 2)

	// Submit transaction without the subnet information.
	t.Run("no subnet", func(t *testing.T) {
		params := dhcp.CreateSubnetSubmitParams{
			ID:     transactionID,
			Subnet: nil,
		}
		rsp := rapi.CreateSubnetSubmit(ctx, params)
		require.IsType(t, &dhcp.CreateSubnetSubmitDefault{}, rsp)
		defaultRsp := rsp.(*dhcp.CreateSubnetSubmitDefault)
		require.Equal(t, http.StatusBadRequest, getStatusCode(*defaultRsp))
		require.Equal(t, "Subnet information not specified", *defaultRsp.Payload.Message)
	})

	// Submit transaction with non-matching transaction ID.
	t.Run("wrong transaction id", func(t *testing.T) {
		params := dhcp.CreateSubnetSubmitParams{
			ID: transactionID + 1,
			Subnet: &models.Subnet{
				ID:     0,
				Subnet: "192.0.2.0/24",
				LocalSubnets: []*models.LocalSubnet{
					{
						DaemonID: dbapps[0].Daemons[0].ID,
					},
					{
						DaemonID: dbapps[1].Daemons[0].ID,
					},
				},
			},
		}
		rsp := rapi.CreateSubnetSubmit(ctx, params)
		require.IsType(t, &dhcp.CreateSubnetSubmitDefault{}, rsp)
		defaultRsp := rsp.(*dhcp.CreateSubnetSubmitDefault)
		require.Equal(t, http.StatusNotFound, getStatusCode(*defaultRsp))
		require.Equal(t, "Transaction expired for the subnet update", *defaultRsp.Payload.Message)
	})

	// Submit transaction with a subnet that is not associated with any daemons.
	// It simulates a failure in "apply" step which typically is caused by
	// some internal server problem rather than malformed request.
	t.Run("no daemons in subnet", func(t *testing.T) {
		params := dhcp.CreateSubnetSubmitParams{
			ID: transactionID,
			Subnet: &models.Subnet{
				ID:           0,
				Subnet:       "192.0.2.0/24",
				LocalSubnets: []*models.LocalSubnet{},
			},
		}
		rsp := rapi.CreateSubnetSubmit(ctx, params)
		require.IsType(t, &dhcp.CreateSubnetSubmitDefault{}, rsp)
		defaultRsp := rsp.(*dhcp.CreateSubnetSubmitDefault)
		require.Equal(t, http.StatusInternalServerError, getStatusCode(*defaultRsp))
		require.Equal(t, "Problem with applying subnet information: applied subnet 192.0.2.0/24 is not associated with any daemon", *defaultRsp.Payload.Message)
	})

	// Submit transaction with valid ID and subnet but expect the agent to
	// return an error code. This is considered a conflict with the state
	// of the Kea servers.
	t.Run("commit failure", func(t *testing.T) {
		params := dhcp.CreateSubnetSubmitParams{
			ID: transactionID,
			Subnet: &models.Subnet{
				ID:     0,
				Subnet: "192.0.2.0/24",
				LocalSubnets: []*models.LocalSubnet{
					{
						DaemonID: dbapps[0].Daemons[0].ID,
					},
					{
						DaemonID: dbapps[1].Daemons[0].ID,
					},
				},
			},
		}
		rsp := rapi.CreateSubnetSubmit(ctx, params)
		require.IsType(t, &dhcp.CreateSubnetSubmitDefault{}, rsp)
		defaultRsp := rsp.(*dhcp.CreateSubnetSubmitDefault)
		require.Equal(t, http.StatusConflict, getStatusCode(*defaultRsp))
		require.Equal(t, fmt.Sprintf("Problem with committing subnet information: subnet4-add command to %s failed: error status (1) returned by Kea dhcp4 daemon with text: 'unable to communicate with the daemon'", app1.GetName()),
			*defaultRsp.Payload.Message)
	})
}

// Test that the transaction to create a subnet can be canceled, resulting
// in the removal of this transaction from the config manager and allowing
// another user to apply config updates.
func TestCreateSubnetBeginCancel(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	serverConfig := `{
		"Dhcp6": {
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
	params := dhcp.CreateSubnetBeginParams{}
	rsp := rapi.CreateSubnetBegin(ctx, params)
	require.IsType(t, &dhcp.CreateSubnetBeginOK{}, rsp)
	okRsp := rsp.(*dhcp.CreateSubnetBeginOK)
	contents := okRsp.Payload

	// Make sure the server returned transaction ID, subnet, daemons and client classes.
	transactionID := contents.ID
	require.NotZero(t, transactionID)
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

	// Cancel the transaction.
	params2 := dhcp.CreateSubnetDeleteParams{
		ID: transactionID,
	}
	rsp2 := rapi.CreateSubnetDelete(ctx, params2)
	require.IsType(t, &dhcp.CreateSubnetDeleteOK{}, rsp2)

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

// Test the calls for creating new transaction and updating a subnet.
func TestUpdateSubnet4BeginSubmit(t *testing.T) {
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
							"subnet": "192.0.2.0/24",
							"option-data": [
								{
									"always-send": true,
									"code": 3,
									"csv-format": true,
									"data": "192.0.2.1",
									"space": "dhcp4"
								}
							],
							"user-context": {
								"foo": "bar"
							}
						}
					]
				},
				{
					"name": "bar",
					"subnet4": []
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

	subnets, err := dbmodel.GetSubnetsByPrefix(db, "192.0.2.0/24")
	require.NoError(t, err)
	require.Len(t, subnets, 1)

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
	params := dhcp.UpdateSubnetBeginParams{
		SubnetID: subnets[0].ID,
	}
	rsp := rapi.UpdateSubnetBegin(ctx, params)
	require.IsType(t, &dhcp.UpdateSubnetBeginOK{}, rsp)
	okRsp := rsp.(*dhcp.UpdateSubnetBeginOK)
	contents := okRsp.Payload

	// Make sure the server returned transaction ID, subnet, daemons, shared networks and client classes.
	transactionID := contents.ID
	require.NotZero(t, transactionID)
	require.NotNil(t, contents.Subnet)
	require.Equal(t, "foo", contents.Subnet.SharedNetwork)
	require.Len(t, contents.Daemons, 2)
	require.Len(t, contents.SharedNetworks4, 2)
	require.Empty(t, contents.SharedNetworks6)
	require.Len(t, contents.ClientClasses, 2)

	keaConfigSubnetParameters := &models.KeaConfigSubnetParameters{
		SubnetLevelParameters: &models.KeaConfigSubnetDerivedParameters{
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
	params2 := dhcp.UpdateSubnetSubmitParams{
		ID: transactionID,
		Subnet: &models.Subnet{
			ID:              subnets[0].ID,
			Subnet:          subnets[0].Prefix,
			SharedNetworkID: subnets[0].SharedNetworkID,
			SharedNetwork:   "foo",
			LocalSubnets: []*models.LocalSubnet{
				{
					ID:          1,
					DaemonID:    subnets[0].LocalSubnets[0].DaemonID,
					UserContext: subnets[0].LocalSubnets[0].UserContext,
					Pools: []*models.Pool{
						{
							Pool: storkutil.Ptr("192.0.2.10-192.0.2.20"),
							KeaConfigPoolParameters: &models.KeaConfigPoolParameters{
								KeaConfigAssortedPoolParameters: models.KeaConfigAssortedPoolParameters{
									PoolID: storkutil.Ptr(int64(1234)),
								},
								KeaConfigClientClassParameters: models.KeaConfigClientClassParameters{
									ClientClass:          storkutil.Ptr("foo"),
									RequireClientClasses: []string{"foo", "bar"},
								},
								DHCPOptions: models.DHCPOptions{
									Options: []*models.DHCPOption{
										{
											AlwaysSend: false,
											Code:       3,
											Fields: []*models.DHCPOptionField{
												{
													FieldType: "ipv4-address",
													Values:    []string{"192.0.2.2"},
												},
											},
											Universe: 4,
										},
									},
								},
							},
						},
						{
							Pool: storkutil.Ptr("192.0.2.30-192.0.2.40"),
						},
					},
					KeaConfigSubnetParameters: keaConfigSubnetParameters,
				},
				{
					ID:          1,
					DaemonID:    subnets[0].LocalSubnets[1].DaemonID,
					UserContext: subnets[0].LocalSubnets[1].UserContext,
					Pools: []*models.Pool{
						{
							Pool: storkutil.Ptr("192.0.2.10-192.0.2.20"),
							KeaConfigPoolParameters: &models.KeaConfigPoolParameters{
								KeaConfigAssortedPoolParameters: models.KeaConfigAssortedPoolParameters{
									PoolID: storkutil.Ptr(int64(1234)),
								},
								KeaConfigClientClassParameters: models.KeaConfigClientClassParameters{
									ClientClass:          storkutil.Ptr("foo"),
									RequireClientClasses: []string{"foo", "bar"},
								},
								DHCPOptions: models.DHCPOptions{
									Options: []*models.DHCPOption{
										{
											AlwaysSend: false,
											Code:       3,
											Fields: []*models.DHCPOptionField{
												{
													FieldType: "ipv4-address",
													Values:    []string{"192.0.2.2"},
												},
											},
											Universe: 4,
										},
									},
								},
							},
						},
						{
							Pool: storkutil.Ptr("192.0.2.30-192.0.2.40"),
						},
					},
					KeaConfigSubnetParameters: keaConfigSubnetParameters,
				},
			},
		},
	}
	rsp2 := rapi.UpdateSubnetSubmit(ctx, params2)
	require.IsType(t, &dhcp.UpdateSubnetSubmitOK{}, rsp2)

	// It should result in sending commands to two Kea servers. Each server
	// receives the subnet4-update command.
	require.Len(t, fa.RecordedCommands, 4)

	for i, c := range fa.RecordedCommands {
		switch {
		case i < 2:
			require.JSONEq(t,
				`{
				"command": "subnet4-update",
				"service": [ "dhcp4" ],
				"arguments": {
					"subnet4": [
						{
							"id": 1,
							"subnet": "192.0.2.0/24",

							"pools": [
								{
									"pool": "192.0.2.10-192.0.2.20",
									"pool-id": 1234,
									"client-class": "foo",
									"require-client-classes": [ "foo", "bar" ],
									"option-data": [
										{
											"code": 3,
											"csv-format": true,
											"data": "192.0.2.2",
											"space": "dhcp4"
										}
									]
								},
								{
									"pool": "192.0.2.30-192.0.2.40"
								}
							],

							"user-context": {
								"foo": "bar"
							},

							"cache-max-age": 1000,
							"cache-threshold": 0.25,
							"client-class": "foo",
							"require-client-classes": [ "bar" ],
							"ddns-generated-prefix": "abc",
							"ddns-override-client-update": true,
							"ddns-override-no-update": false,
							"ddns-qualifying-suffix": "example.org",
							"ddns-replace-client-name": "never",
							"ddns-send-updates": true,
							"ddns-update-on-renew": true,
							"ddns-use-conflict-resolution": true,
							"ddns-conflict-resolution-mode": "check-with-dhcid",
							"4o6-interface": "eth0",
							"4o6-interface-id": "ifaceid",
							"4o6-subnet": "2001:db8:1::/64",
							"hostname-char-replacement": "a",
							"hostname-char-set": "b",
							"reservation-mode": "in-pool",
							"reservations-global": false,
							"reservations-in-subnet": true,
							"reservations-out-of-pool": false,
							"calculate-tee-times": true,
							"rebind-timer": 2000,
							"renew-timer": 1000,
							"t1-percent": 0.25,
							"t2-percent": 0.50,
							"max-valid-lifetime": 5000,
							"min-valid-lifetime": 4000,
							"valid-lifetime": 4500,
							"allocator": "random",
							"authoritative": true,
							"boot-file-name": "/tmp/filename",
							"interface": "eth0",
							"match-client-id": true,
							"next-server": "192.0.2.1",
							"relay": {
								"ip-addresses": [ "10.1.1.1" ]
							},
							"server-hostname": "myhost.example.org",
							"store-extended-info": true,
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
					]
				}
			}`,
				c.Marshal())
		default:
			require.JSONEq(t,
				`{
						"command": "config-write",
						"service": [ "dhcp4" ]
				}`,
				c.Marshal())
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

	// Make sure that the updated host has been stored in the database.
	returnedSubnet, err := dbmodel.GetSubnet(db, subnets[0].ID)
	require.NoError(t, err)
	require.NotNil(t, returnedSubnet)
	require.NotNil(t, returnedSubnet.SharedNetwork)

	require.Len(t, returnedSubnet.LocalSubnets, 2)
	for _, ls := range returnedSubnet.LocalSubnets {
		require.Len(t, ls.AddressPools, 2)
		require.Equal(t, "192.0.2.10", ls.AddressPools[0].LowerBound)
		require.Equal(t, "192.0.2.20", ls.AddressPools[0].UpperBound)
		require.Equal(t, "192.0.2.30", ls.AddressPools[1].LowerBound)
		require.Equal(t, "192.0.2.40", ls.AddressPools[1].UpperBound)

		require.NotNil(t, ls.UserContext)
		require.Contains(t, ls.UserContext, "foo")
		require.Equal(t, "bar", ls.UserContext["foo"])

		require.NotNil(t, ls.KeaParameters)
		require.NotNil(t, ls.KeaParameters.CacheMaxAge)
		require.EqualValues(t, 1000, *ls.KeaParameters.CacheMaxAge)
		require.NotNil(t, ls.KeaParameters.CacheThreshold)
		require.EqualValues(t, 0.25, *ls.KeaParameters.CacheThreshold)
		require.NotNil(t, ls.KeaParameters.ClientClass)
		require.Equal(t, "foo", *ls.KeaParameters.ClientClass)
		require.Len(t, ls.KeaParameters.RequireClientClasses, 1)
		require.EqualValues(t, "bar", ls.KeaParameters.RequireClientClasses[0])
		require.NotNil(t, ls.KeaParameters.DDNSGeneratedPrefix)
		require.Equal(t, "abc", *ls.KeaParameters.DDNSGeneratedPrefix)
		require.NotNil(t, ls.KeaParameters.DDNSOverrideClientUpdate)
		require.True(t, *ls.KeaParameters.DDNSOverrideClientUpdate)
		require.NotNil(t, ls.KeaParameters.DDNSOverrideNoUpdate)
		require.False(t, *ls.KeaParameters.DDNSOverrideNoUpdate)
		require.NotNil(t, ls.KeaParameters.DDNSQualifyingSuffix)
		require.Equal(t, "example.org", *ls.KeaParameters.DDNSQualifyingSuffix)
		require.NotNil(t, ls.KeaParameters.DDNSReplaceClientName)
		require.Equal(t, "never", *ls.KeaParameters.DDNSReplaceClientName)
		require.NotNil(t, ls.KeaParameters.DDNSSendUpdates)
		require.True(t, *ls.KeaParameters.DDNSSendUpdates)
		require.NotNil(t, ls.KeaParameters.DDNSUpdateOnRenew)
		require.True(t, *ls.KeaParameters.DDNSUpdateOnRenew)
		require.NotNil(t, ls.KeaParameters.DDNSUseConflictResolution)
		require.True(t, *ls.KeaParameters.DDNSUseConflictResolution)
		require.NotNil(t, ls.KeaParameters.DDNSConflictResolutionMode)
		require.Equal(t, "check-with-dhcid", *ls.KeaParameters.DDNSConflictResolutionMode)
		require.NotNil(t, ls.KeaParameters.FourOverSixInterface)
		require.Equal(t, "eth0", *ls.KeaParameters.FourOverSixInterface)
		require.NotNil(t, ls.KeaParameters.FourOverSixInterfaceID)
		require.Equal(t, "ifaceid", *ls.KeaParameters.FourOverSixInterfaceID)
		require.NotNil(t, ls.KeaParameters.FourOverSixSubnet)
		require.Equal(t, "2001:db8:1::/64", *ls.KeaParameters.FourOverSixSubnet)
		require.NotNil(t, ls.KeaParameters.HostnameCharReplacement)
		require.Equal(t, "a", *ls.KeaParameters.HostnameCharReplacement)
		require.NotNil(t, ls.KeaParameters.HostnameCharSet)
		require.Equal(t, "b", *ls.KeaParameters.HostnameCharSet)
		require.NotNil(t, ls.KeaParameters.ReservationMode)
		require.Equal(t, "in-pool", *ls.KeaParameters.ReservationMode)
		require.NotNil(t, ls.KeaParameters.ReservationsGlobal)
		require.False(t, *ls.KeaParameters.ReservationsGlobal)
		require.NotNil(t, ls.KeaParameters.ReservationsInSubnet)
		require.True(t, *ls.KeaParameters.ReservationsInSubnet)
		require.NotNil(t, ls.KeaParameters.ReservationsOutOfPool)
		require.False(t, *ls.KeaParameters.ReservationsOutOfPool)
		require.NotNil(t, ls.KeaParameters.CalculateTeeTimes)
		require.True(t, *ls.KeaParameters.CalculateTeeTimes)
		require.NotNil(t, ls.KeaParameters.RebindTimer)
		require.EqualValues(t, 2000, *ls.KeaParameters.RebindTimer)
		require.NotNil(t, ls.KeaParameters.RenewTimer)
		require.EqualValues(t, 1000, *ls.KeaParameters.RenewTimer)
		require.NotNil(t, ls.KeaParameters.T1Percent)
		require.EqualValues(t, 0.25, *ls.KeaParameters.T1Percent)
		require.NotNil(t, ls.KeaParameters.T2Percent)
		require.EqualValues(t, 0.50, *ls.KeaParameters.T2Percent)
		require.NotNil(t, ls.KeaParameters.MaxValidLifetime)
		require.EqualValues(t, 5000, *ls.KeaParameters.MaxValidLifetime)
		require.NotNil(t, ls.KeaParameters.MinValidLifetime)
		require.EqualValues(t, 4000, *ls.KeaParameters.MinValidLifetime)
		require.NotNil(t, ls.KeaParameters.ValidLifetime)
		require.EqualValues(t, 4500, *ls.KeaParameters.ValidLifetime)
		require.NotNil(t, ls.KeaParameters.Allocator)
		require.Equal(t, "random", *ls.KeaParameters.Allocator)
		require.NotNil(t, ls.KeaParameters.Authoritative)
		require.True(t, *ls.KeaParameters.Authoritative)
		require.NotNil(t, ls.KeaParameters.Authoritative)
		require.Equal(t, "/tmp/filename", *ls.KeaParameters.BootFileName)
		require.NotNil(t, ls.KeaParameters.Interface)
		require.Equal(t, "eth0", *ls.KeaParameters.Interface)
		require.NotNil(t, ls.KeaParameters.MatchClientID)
		require.True(t, *ls.KeaParameters.MatchClientID)
		require.NotNil(t, ls.KeaParameters.NextServer)
		require.Equal(t, "192.0.2.1", *ls.KeaParameters.NextServer)
		require.NotNil(t, ls.KeaParameters.Relay)
		require.Len(t, ls.KeaParameters.Relay.IPAddresses, 1)
		require.Equal(t, "10.1.1.1", ls.KeaParameters.Relay.IPAddresses[0])
		require.NotNil(t, ls.KeaParameters.ServerHostname)
		require.Equal(t, "myhost.example.org", *ls.KeaParameters.ServerHostname)
		require.NotNil(t, ls.KeaParameters.StoreExtendedInfo)
		require.True(t, *ls.KeaParameters.StoreExtendedInfo)

		// DHCP options
		require.Len(t, ls.DHCPOptionSet.Options, 1)
		require.True(t, ls.DHCPOptionSet.Options[0].AlwaysSend)
		require.EqualValues(t, 3, ls.DHCPOptionSet.Options[0].Code)
		require.Len(t, ls.DHCPOptionSet.Options[0].Fields, 1)
		require.Equal(t, dhcpmodel.IPv4AddressField, ls.DHCPOptionSet.Options[0].Fields[0].FieldType)
		require.Len(t, ls.DHCPOptionSet.Options[0].Fields[0].Values, 1)
		require.Equal(t, "192.0.2.1", ls.DHCPOptionSet.Options[0].Fields[0].Values[0])
		require.Equal(t, dhcpmodel.DHCPv4OptionSpace, ls.DHCPOptionSet.Options[0].Space)
		require.NotEmpty(t, ls.DHCPOptionSet.Hash)
	}
}

// Test the calls for moving a subnet to a different shared network.
func TestUpdateSubnet4BeginSubmitChangeSharedNetwork(t *testing.T) {
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
				},
				{
					"name": "bar",
					"subnet4": []
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

	subnets, err := dbmodel.GetSubnetsByPrefix(db, "192.0.2.0/24")
	require.NoError(t, err)
	require.Len(t, subnets, 1)

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
	params := dhcp.UpdateSubnetBeginParams{
		SubnetID: subnets[0].ID,
	}
	rsp := rapi.UpdateSubnetBegin(ctx, params)
	require.IsType(t, &dhcp.UpdateSubnetBeginOK{}, rsp)
	okRsp := rsp.(*dhcp.UpdateSubnetBeginOK)
	contents := okRsp.Payload

	// Make sure the server returned transaction ID, subnet, daemons, shared networks and client classes.
	transactionID := contents.ID
	require.NotZero(t, transactionID)
	require.NotNil(t, contents.Subnet)
	require.Equal(t, "foo", contents.Subnet.SharedNetwork)
	require.Len(t, contents.Daemons, 1)
	require.Len(t, contents.SharedNetworks4, 2)
	require.Empty(t, contents.SharedNetworks6)
	require.Empty(t, contents.ClientClasses)

	// Submit transaction.
	params2 := dhcp.UpdateSubnetSubmitParams{
		ID: transactionID,
		Subnet: &models.Subnet{
			ID:              subnets[0].ID,
			Subnet:          subnets[0].Prefix,
			SharedNetworkID: contents.SharedNetworks4[1].ID,
			SharedNetwork:   contents.SharedNetworks4[1].Name,
			LocalSubnets: []*models.LocalSubnet{
				{
					ID:       1,
					DaemonID: dbapps[0].Daemons[0].ID,
					Pools: []*models.Pool{
						{
							Pool: storkutil.Ptr("192.0.2.10-192.0.2.20"),
							KeaConfigPoolParameters: &models.KeaConfigPoolParameters{
								KeaConfigClientClassParameters: models.KeaConfigClientClassParameters{},
								DHCPOptions:                    models.DHCPOptions{},
							},
						},
					},
					KeaConfigSubnetParameters: &models.KeaConfigSubnetParameters{},
				},
			},
		},
	}
	rsp2 := rapi.UpdateSubnetSubmit(ctx, params2)
	require.IsType(t, &dhcp.UpdateSubnetSubmitOK{}, rsp2)

	require.Len(t, fa.RecordedCommands, 4)

	for i, c := range fa.RecordedCommands {
		switch i {
		case 0:
			require.JSONEq(t,
				`{
				"command": "subnet4-update",
				"service": [ "dhcp4" ],
				"arguments": {
					"subnet4": [
						{
							"id": 1,
							"subnet": "192.0.2.0/24",
							"pools": [
								{
									"pool": "192.0.2.10-192.0.2.20"
								}
							]
						}
					]
				}
			}`,
				c.Marshal())
		case 1:
			require.JSONEq(t,
				`{
					"command": "network4-subnet-del",
					"service": [ "dhcp4" ],
					"arguments": {
						"name": "foo",
						"id": 1
					}
				}`,
				c.Marshal())
		case 2:
			require.JSONEq(t,
				`{
					"command": "network4-subnet-add",
					"service": [ "dhcp4" ],
					"arguments": {
						"name": "bar",
						"id": 1
					}
				}`,
				c.Marshal())
		default:
			require.JSONEq(t,
				`{
						"command": "config-write",
						"service": [ "dhcp4" ]
				}`,
				c.Marshal())
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

	// Make sure that the updated subnet has been stored in the database.
	returnedSubnet, err := dbmodel.GetSubnet(db, subnets[0].ID)
	require.NoError(t, err)
	require.NotNil(t, returnedSubnet)
	require.NotNil(t, returnedSubnet.SharedNetwork)
	require.Equal(t, "bar", returnedSubnet.SharedNetwork.Name)
	require.Len(t, returnedSubnet.LocalSubnets, 1)
}

// Test the calls for adding a subnet to a shared network.
func TestUpdateSubnet4BeginSubmitAddToSharedNetwork(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	serverConfig := `{
		"Dhcp4": {
			"subnet4": [
				{
					"id": 1,
					"subnet": "192.0.2.0/24"
				}
			],
			"shared-networks": [
				{
					"name": "foo"
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

	subnets, err := dbmodel.GetSubnetsByPrefix(db, "192.0.2.0/24")
	require.NoError(t, err)
	require.Len(t, subnets, 1)

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
	params := dhcp.UpdateSubnetBeginParams{
		SubnetID: subnets[0].ID,
	}
	rsp := rapi.UpdateSubnetBegin(ctx, params)
	require.IsType(t, &dhcp.UpdateSubnetBeginOK{}, rsp)
	okRsp := rsp.(*dhcp.UpdateSubnetBeginOK)
	contents := okRsp.Payload

	// Make sure the server returned transaction ID, subnet, daemons, shared networks and client classes.
	transactionID := contents.ID
	require.NotZero(t, transactionID)
	require.NotNil(t, contents.Subnet)
	require.Empty(t, contents.Subnet.SharedNetwork)
	require.Len(t, contents.Daemons, 1)
	require.Len(t, contents.SharedNetworks4, 1)
	require.Empty(t, contents.SharedNetworks6)
	require.Empty(t, contents.ClientClasses)

	// Submit transaction.
	params2 := dhcp.UpdateSubnetSubmitParams{
		ID: transactionID,
		Subnet: &models.Subnet{
			ID:              subnets[0].ID,
			Subnet:          subnets[0].Prefix,
			SharedNetworkID: contents.SharedNetworks4[0].ID,
			SharedNetwork:   contents.SharedNetworks4[0].Name,
			LocalSubnets: []*models.LocalSubnet{
				{
					ID:       1,
					DaemonID: dbapps[0].Daemons[0].ID,
					Pools: []*models.Pool{
						{
							Pool: storkutil.Ptr("192.0.2.10-192.0.2.20"),
							KeaConfigPoolParameters: &models.KeaConfigPoolParameters{
								KeaConfigClientClassParameters: models.KeaConfigClientClassParameters{},
								DHCPOptions:                    models.DHCPOptions{},
							},
						},
					},
					KeaConfigSubnetParameters: &models.KeaConfigSubnetParameters{},
				},
			},
		},
	}
	rsp2 := rapi.UpdateSubnetSubmit(ctx, params2)
	require.IsType(t, &dhcp.UpdateSubnetSubmitOK{}, rsp2)

	require.Len(t, fa.RecordedCommands, 3)

	for i, c := range fa.RecordedCommands {
		switch i {
		case 0:
			require.JSONEq(t,
				`{
				"command": "subnet4-update",
				"service": [ "dhcp4" ],
				"arguments": {
					"subnet4": [
						{
							"id": 1,
							"subnet": "192.0.2.0/24",
							"pools": [
								{
									"pool": "192.0.2.10-192.0.2.20"
								}
							]
						}
					]
				}
			}`,
				c.Marshal())
		case 1:
			require.JSONEq(t,
				`{
					"command": "network4-subnet-add",
					"service": [ "dhcp4" ],
					"arguments": {
						"name": "foo",
						"id": 1
					}
				}`,
				c.Marshal())
		default:
			require.JSONEq(t,
				`{
						"command": "config-write",
						"service": [ "dhcp4" ]
				}`,
				c.Marshal())
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

	// Make sure that the updated subnet has been stored in the database.
	returnedSubnet, err := dbmodel.GetSubnet(db, subnets[0].ID)
	require.NoError(t, err)
	require.NotNil(t, returnedSubnet)
	require.NotNil(t, returnedSubnet.SharedNetwork)
	require.Equal(t, "foo", returnedSubnet.SharedNetwork.Name)
	require.Len(t, returnedSubnet.LocalSubnets, 1)
}

// Test the calls for removing a subnet from a shared network.
func TestUpdateSubnet4BeginSubmitRemoveFromSharedNetwork(t *testing.T) {
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

	dbapps, err := dbmodel.GetAllApps(db, true)
	require.NoError(t, err)
	require.Len(t, dbapps, 1)

	subnets, err := dbmodel.GetSubnetsByPrefix(db, "192.0.2.0/24")
	require.NoError(t, err)
	require.Len(t, subnets, 1)

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
	params := dhcp.UpdateSubnetBeginParams{
		SubnetID: subnets[0].ID,
	}
	rsp := rapi.UpdateSubnetBegin(ctx, params)
	require.IsType(t, &dhcp.UpdateSubnetBeginOK{}, rsp)
	okRsp := rsp.(*dhcp.UpdateSubnetBeginOK)
	contents := okRsp.Payload

	// Make sure the server returned transaction ID, subnet, daemons, shared networks and client classes.
	transactionID := contents.ID
	require.NotZero(t, transactionID)
	require.NotNil(t, contents.Subnet)
	require.Equal(t, "foo", contents.Subnet.SharedNetwork)
	require.Len(t, contents.Daemons, 1)
	require.Len(t, contents.SharedNetworks4, 1)
	require.Empty(t, contents.SharedNetworks6)
	require.Empty(t, contents.ClientClasses)

	// Submit transaction.
	params2 := dhcp.UpdateSubnetSubmitParams{
		ID: transactionID,
		Subnet: &models.Subnet{
			ID:     subnets[0].ID,
			Subnet: subnets[0].Prefix,
			LocalSubnets: []*models.LocalSubnet{
				{
					ID:       1,
					DaemonID: dbapps[0].Daemons[0].ID,
					Pools: []*models.Pool{
						{
							Pool: storkutil.Ptr("192.0.2.10-192.0.2.20"),
							KeaConfigPoolParameters: &models.KeaConfigPoolParameters{
								KeaConfigClientClassParameters: models.KeaConfigClientClassParameters{},
								DHCPOptions:                    models.DHCPOptions{},
							},
						},
					},
					KeaConfigSubnetParameters: &models.KeaConfigSubnetParameters{},
				},
			},
		},
	}
	rsp2 := rapi.UpdateSubnetSubmit(ctx, params2)
	require.IsType(t, &dhcp.UpdateSubnetSubmitOK{}, rsp2)

	require.Len(t, fa.RecordedCommands, 3)

	for i, c := range fa.RecordedCommands {
		switch i {
		case 0:
			require.JSONEq(t,
				`{
				"command": "subnet4-update",
				"service": [ "dhcp4" ],
				"arguments": {
					"subnet4": [
						{
							"id": 1,
							"subnet": "192.0.2.0/24",
							"pools": [
								{
									"pool": "192.0.2.10-192.0.2.20"
								}
							]
						}
					]
				}
			}`,
				c.Marshal())
		case 1:
			require.JSONEq(t,
				`{
					"command": "network4-subnet-del",
					"service": [ "dhcp4" ],
					"arguments": {
						"name": "foo",
						"id": 1
					}
				}`,
				c.Marshal())
		default:
			require.JSONEq(t,
				`{
						"command": "config-write",
						"service": [ "dhcp4" ]
				}`,
				c.Marshal())
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

	// Make sure that the updated subnet has been stored in the database.
	returnedSubnet, err := dbmodel.GetSubnet(db, subnets[0].ID)
	require.NoError(t, err)
	require.NotNil(t, returnedSubnet)
	require.Nil(t, returnedSubnet.SharedNetwork)
	require.Len(t, returnedSubnet.LocalSubnets, 1)
}

// Test the calls for creating new transaction and updating a subnet.
func TestUpdateSubnet6BeginSubmit(t *testing.T) {
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
				},
				{
					"name": "bar",
					"subnet6": []
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

	subnets, err := dbmodel.GetSubnetsByPrefix(db, "2001:db8:1::/64")
	require.NoError(t, err)
	require.Len(t, subnets, 1)

	networks, err := dbmodel.GetAllSharedNetworks(db, 6)
	require.NoError(t, err)
	require.Len(t, networks, 2)

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
	params := dhcp.UpdateSubnetBeginParams{
		SubnetID: subnets[0].ID,
	}
	rsp := rapi.UpdateSubnetBegin(ctx, params)
	require.IsType(t, &dhcp.UpdateSubnetBeginOK{}, rsp)
	okRsp := rsp.(*dhcp.UpdateSubnetBeginOK)
	contents := okRsp.Payload

	// Make sure the server returned transaction ID, subnet, daemons and client classes.
	transactionID := contents.ID
	require.NotZero(t, transactionID)
	require.NotNil(t, contents.Subnet)
	require.Len(t, contents.Daemons, 2)
	require.Empty(t, contents.SharedNetworks4)
	require.Len(t, contents.SharedNetworks6, 2)
	require.Len(t, contents.ClientClasses, 2)

	keaConfigSubnetParameters := &models.KeaConfigSubnetParameters{
		SubnetLevelParameters: &models.KeaConfigSubnetDerivedParameters{
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
	params2 := dhcp.UpdateSubnetSubmitParams{
		ID: transactionID,
		Subnet: &models.Subnet{
			ID:              subnets[0].ID,
			Subnet:          subnets[0].Prefix,
			SharedNetworkID: subnets[0].SharedNetworkID,
			SharedNetwork:   "foo",
			LocalSubnets: []*models.LocalSubnet{
				{
					ID:       1,
					DaemonID: dbapps[0].Daemons[0].ID,
					PrefixDelegationPools: []*models.DelegatedPrefixPool{
						{
							Prefix:          storkutil.Ptr("2001:db8:1::/64"),
							DelegatedLength: storkutil.Ptr(int64(80)),
							ExcludedPrefix:  "2001:db8:1:2::/80",
							KeaConfigPoolParameters: &models.KeaConfigPoolParameters{
								KeaConfigAssortedPoolParameters: models.KeaConfigAssortedPoolParameters{
									PoolID: storkutil.Ptr(int64(2345)),
								},
								KeaConfigClientClassParameters: models.KeaConfigClientClassParameters{
									ClientClass:          storkutil.Ptr("foo"),
									RequireClientClasses: []string{"foo", "bar"},
								},
								DHCPOptions: models.DHCPOptions{
									Options: []*models.DHCPOption{
										{
											AlwaysSend: false,
											Code:       23,
											Fields: []*models.DHCPOptionField{
												{
													FieldType: "ipv6-address",
													Values:    []string{"2001:db8:1::1"},
												},
											},
											Universe: 6,
										},
									},
								},
							},
						},
						{
							Prefix:          storkutil.Ptr("2001:db8:2::/64"),
							DelegatedLength: storkutil.Ptr(int64(80)),
						},
					},
					KeaConfigSubnetParameters: keaConfigSubnetParameters,
				},
				{
					ID:       1,
					DaemonID: dbapps[1].Daemons[0].ID,
					PrefixDelegationPools: []*models.DelegatedPrefixPool{
						{
							Prefix:          storkutil.Ptr("2001:db8:1::/64"),
							DelegatedLength: storkutil.Ptr(int64(80)),
							ExcludedPrefix:  "2001:db8:1:2::/80",
							KeaConfigPoolParameters: &models.KeaConfigPoolParameters{
								KeaConfigAssortedPoolParameters: models.KeaConfigAssortedPoolParameters{
									PoolID: storkutil.Ptr(int64(2345)),
								},
								KeaConfigClientClassParameters: models.KeaConfigClientClassParameters{
									ClientClass:          storkutil.Ptr("foo"),
									RequireClientClasses: []string{"foo", "bar"},
								},
								DHCPOptions: models.DHCPOptions{
									Options: []*models.DHCPOption{
										{
											AlwaysSend: false,
											Code:       23,
											Fields: []*models.DHCPOptionField{
												{
													FieldType: "ipv6-address",
													Values:    []string{"2001:db8:1::1"},
												},
											},
											Universe: 6,
										},
									},
								},
							},
						},
						{
							Prefix:          storkutil.Ptr("2001:db8:2::/64"),
							DelegatedLength: storkutil.Ptr(int64(80)),
						},
					},
					KeaConfigSubnetParameters: keaConfigSubnetParameters,
				},
			},
		},
	}
	rsp2 := rapi.UpdateSubnetSubmit(ctx, params2)
	require.IsType(t, &dhcp.UpdateSubnetSubmitOK{}, rsp2)

	// It should result in sending commands to two Kea servers. Each server
	// receives the subnet4-update command.
	require.Len(t, fa.RecordedCommands, 4)

	for i, c := range fa.RecordedCommands {
		switch {
		case i < 2:
			require.JSONEq(t,
				`{
				"command": "subnet6-update",
				"service": [ "dhcp6" ],
				"arguments": {
					"subnet6": [
						{
							"id": 1,
							"subnet": "2001:db8:1::/64",
							"interface-id": "ifaceid",
							"min-preferred-lifetime": 4000,
							"max-preferred-lifetime": 5000,
							"preferred-lifetime": 4500,
							"pd-allocator": "random",
							"pd-pools": [
								{
									"prefix": "2001:db8:1::",
									"prefix-len": 64,
									"delegated-len": 80,
									"excluded-prefix": "2001:db8:1:2::",
									"excluded-prefix-len": 80,
									"client-class": "foo",
									"require-client-classes": [ "foo", "bar" ],
									"pool-id": 2345,
									"option-data": [
										{
											"code": 23,
											"csv-format": true,
											"data": "2001:db8:1::1",
											"space": "dhcp6"
										}
									]
								},
								{
									"prefix": "2001:db8:2::",
									"prefix-len": 64,
									"delegated-len": 80
								}
							]
						}
					]
				}
			}`,
				c.Marshal())
		default:
			require.JSONEq(t,
				`{
						"command": "config-write",
						"service": [ "dhcp6" ]
				}`,
				c.Marshal())
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

	// Make sure that the updated host has been stored in the database.
	returnedSubnet, err := dbmodel.GetSubnet(db, subnets[0].ID)
	require.NoError(t, err)
	require.NotNil(t, returnedSubnet)
	require.NotNil(t, returnedSubnet.SharedNetwork)

	require.Len(t, returnedSubnet.LocalSubnets, 2)
	for _, ls := range returnedSubnet.LocalSubnets {
		require.NotNil(t, ls.KeaParameters)

		require.NotNil(t, ls.KeaParameters.InterfaceID)
		require.Equal(t, "ifaceid", *ls.KeaParameters.InterfaceID)

		require.NotNil(t, ls.KeaParameters.MinPreferredLifetime)
		require.EqualValues(t, 4000, *ls.KeaParameters.MinPreferredLifetime)

		require.NotNil(t, ls.KeaParameters.MinPreferredLifetime)
		require.EqualValues(t, 5000, *ls.KeaParameters.MaxPreferredLifetime)

		require.NotNil(t, ls.KeaParameters.PreferredLifetime)
		require.EqualValues(t, 4500, *ls.KeaParameters.PreferredLifetime)

		require.NotNil(t, ls.KeaParameters.PDAllocator)
		require.Equal(t, "random", *ls.KeaParameters.PDAllocator)

		// DHCP options
		require.Empty(t, ls.DHCPOptionSet.Options)
		require.Empty(t, ls.DHCPOptionSet.Hash)
	}
}

// Test the calls for removing a subnet from a shared network.
func TestUpdateSubnet6BeginSubmitRemoveFromSharedNetwork(t *testing.T) {
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
				},
				{
					"name": "bar",
					"subnet6": []
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

	dbapps, err := dbmodel.GetAllApps(db, true)
	require.NoError(t, err)
	require.Len(t, dbapps, 1)

	subnets, err := dbmodel.GetSubnetsByPrefix(db, "2001:db8:1::/64")
	require.NoError(t, err)
	require.Len(t, subnets, 1)

	networks, err := dbmodel.GetAllSharedNetworks(db, 6)
	require.NoError(t, err)
	require.Len(t, networks, 2)

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
	params := dhcp.UpdateSubnetBeginParams{
		SubnetID: subnets[0].ID,
	}
	rsp := rapi.UpdateSubnetBegin(ctx, params)
	require.IsType(t, &dhcp.UpdateSubnetBeginOK{}, rsp)
	okRsp := rsp.(*dhcp.UpdateSubnetBeginOK)
	contents := okRsp.Payload

	// Make sure the server returned transaction ID, subnet, daemons and client classes.
	transactionID := contents.ID
	require.NotZero(t, transactionID)
	require.NotNil(t, contents.Subnet)
	require.Len(t, contents.Daemons, 1)
	require.Empty(t, contents.SharedNetworks4)
	require.Len(t, contents.SharedNetworks6, 2)
	require.Empty(t, contents.ClientClasses)

	// Submit transaction.
	params2 := dhcp.UpdateSubnetSubmitParams{
		ID: transactionID,
		Subnet: &models.Subnet{
			ID:     subnets[0].ID,
			Subnet: subnets[0].Prefix,
			LocalSubnets: []*models.LocalSubnet{
				{
					ID:       1,
					DaemonID: dbapps[0].Daemons[0].ID,
					PrefixDelegationPools: []*models.DelegatedPrefixPool{
						{
							Prefix:          storkutil.Ptr("2001:db8:1::/64"),
							DelegatedLength: storkutil.Ptr(int64(80)),
						},
					},
					KeaConfigSubnetParameters: &models.KeaConfigSubnetParameters{},
				},
			},
		},
	}
	rsp2 := rapi.UpdateSubnetSubmit(ctx, params2)
	require.IsType(t, &dhcp.UpdateSubnetSubmitOK{}, rsp2)

	// It should result in sending commands to two Kea servers. Each server
	// receives the subnet6-update command.
	require.Len(t, fa.RecordedCommands, 3)

	for i, c := range fa.RecordedCommands {
		switch i {
		case 0:
			require.JSONEq(t,
				`{
				"command": "subnet6-update",
				"service": [ "dhcp6" ],
				"arguments": {
					"subnet6": [
						{
							"id": 1,
							"subnet": "2001:db8:1::/64",
							"pd-pools": [
								{
									"prefix": "2001:db8:1::",
									"prefix-len": 64,
									"delegated-len": 80
								}
							]
						}
					]
				}
			}`,
				c.Marshal())
		case 1:
			require.JSONEq(t,
				`{
					"command": "network6-subnet-del",
					"service": [ "dhcp6" ],
					"arguments": {
						"name": "foo",
						"id": 1
					}
				}`,
				c.Marshal())
		default:
			require.JSONEq(t,
				`{
						"command": "config-write",
						"service": [ "dhcp6" ]
				}`,
				c.Marshal())
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

	// Make sure that the updated host has been stored in the database.
	returnedSubnet, err := dbmodel.GetSubnet(db, subnets[0].ID)
	require.NoError(t, err)
	require.NotNil(t, returnedSubnet)
	require.Nil(t, returnedSubnet.SharedNetwork)
	require.Len(t, returnedSubnet.LocalSubnets, 1)
}

// Test the calls for adding a subnet to a shared network.
func TestUpdateSubnet6BeginSubmitAddToSharedNetwork(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	serverConfig := `{
		"Dhcp6": {
			"shared-networks": [
				{
					"name": "foo"
				}
			],
			"subnet6": [
				{
					"id": 1,
					"subnet": "2001:db8:1::/64"
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

	dbapps, err := dbmodel.GetAllApps(db, true)
	require.NoError(t, err)
	require.Len(t, dbapps, 1)

	subnets, err := dbmodel.GetSubnetsByPrefix(db, "2001:db8:1::/64")
	require.NoError(t, err)
	require.Len(t, subnets, 1)

	networks, err := dbmodel.GetAllSharedNetworks(db, 6)
	require.NoError(t, err)
	require.Len(t, networks, 1)

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
	params := dhcp.UpdateSubnetBeginParams{
		SubnetID: subnets[0].ID,
	}
	rsp := rapi.UpdateSubnetBegin(ctx, params)
	require.IsType(t, &dhcp.UpdateSubnetBeginOK{}, rsp)
	okRsp := rsp.(*dhcp.UpdateSubnetBeginOK)
	contents := okRsp.Payload

	// Make sure the server returned transaction ID, subnet, daemons and client classes.
	transactionID := contents.ID
	require.NotZero(t, transactionID)
	require.NotNil(t, contents.Subnet)
	require.Len(t, contents.Daemons, 1)
	require.Empty(t, contents.SharedNetworks4)
	require.Len(t, contents.SharedNetworks6, 1)
	require.Empty(t, contents.ClientClasses)

	// Submit transaction.
	params2 := dhcp.UpdateSubnetSubmitParams{
		ID: transactionID,
		Subnet: &models.Subnet{
			ID:              subnets[0].ID,
			Subnet:          subnets[0].Prefix,
			SharedNetworkID: contents.SharedNetworks6[0].ID,
			SharedNetwork:   "foo",
			LocalSubnets: []*models.LocalSubnet{
				{
					ID:       1,
					DaemonID: dbapps[0].Daemons[0].ID,
					PrefixDelegationPools: []*models.DelegatedPrefixPool{
						{
							Prefix:          storkutil.Ptr("2001:db8:1::/64"),
							DelegatedLength: storkutil.Ptr(int64(80)),
						},
					},
					KeaConfigSubnetParameters: &models.KeaConfigSubnetParameters{},
				},
			},
		},
	}
	rsp2 := rapi.UpdateSubnetSubmit(ctx, params2)
	require.IsType(t, &dhcp.UpdateSubnetSubmitOK{}, rsp2)

	require.Len(t, fa.RecordedCommands, 3)

	for i, c := range fa.RecordedCommands {
		switch i {
		case 0:
			require.JSONEq(t,
				`{
					"command": "subnet6-update",
					"service": [ "dhcp6" ],
					"arguments": {
						"subnet6": [
							{
								"id": 1,
								"subnet": "2001:db8:1::/64",
								"pd-pools": [
									{
										"prefix": "2001:db8:1::",
										"prefix-len": 64,
										"delegated-len": 80
									}
								]
							}
						]
					}
				}`,
				c.Marshal())
		case 1:
			require.JSONEq(t,
				`{
					"command": "network6-subnet-add",
					"service": [ "dhcp6" ],
					"arguments": {
						"name": "foo",
						"id": 1
					}
				}`,
				c.Marshal())
		default:
			require.JSONEq(t,
				`{
						"command": "config-write",
						"service": [ "dhcp6" ]
				}`,
				c.Marshal())
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

	// Make sure that the updated host has been stored in the database.
	returnedSubnet, err := dbmodel.GetSubnet(db, subnets[0].ID)
	require.NoError(t, err)
	require.NotNil(t, returnedSubnet)
	require.NotNil(t, returnedSubnet.SharedNetwork)
	require.Len(t, returnedSubnet.LocalSubnets, 1)
}

// Test that an error is returned when it is attempted to begin new
// transaction for updating non-existing subnet.
func TestUpdateSubnetBeginNonExistingSubnetID(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	serverConfig := `{
		"Dhcp4": {
			"subnet4": [
				{
					"id": 1,
					"subnet": "192.0.2.0/24"
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

	subnets, err := dbmodel.GetSubnetsByPrefix(db, "192.0.2.0/24")
	require.NoError(t, err)
	require.Len(t, subnets, 1)

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
	params := dhcp.UpdateSubnetBeginParams{
		SubnetID: int64(1024),
	}
	rsp := rapi.UpdateSubnetBegin(ctx, params)
	require.IsType(t, &dhcp.UpdateSubnetBeginDefault{}, rsp)
	defaultRsp := rsp.(*dhcp.UpdateSubnetBeginDefault)
	require.Equal(t, http.StatusBadRequest, getStatusCode(*defaultRsp))
}

// Test that an error is returned when it is attempted to begin new
// transaction when subnet_cmds hook was not configured.
func TestUpdateSubnetBeginNoSubnetCmdsHook(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	serverConfig := `{
		"Dhcp4": {
			"subnet4": [
				{
					"id": 1,
					"subnet": "192.0.2.0/24"
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

	subnets, err := dbmodel.GetSubnetsByPrefix(db, "192.0.2.0/24")
	require.NoError(t, err)
	require.Len(t, subnets, 1)

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
	params := dhcp.UpdateSubnetBeginParams{
		SubnetID: int64(1),
	}
	rsp := rapi.UpdateSubnetBegin(ctx, params)
	require.IsType(t, &dhcp.UpdateSubnetBeginDefault{}, rsp)
	defaultRsp := rsp.(*dhcp.UpdateSubnetBeginDefault)
	require.Equal(t, http.StatusBadRequest, getStatusCode(*defaultRsp))
}

// Test error cases for submitting subnet update.
func TestUpdateSubnetSubmitError(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	serverConfig := `{
		"Dhcp4": {
			"subnet4": [
				{
					"id": 1,
					"subnet": "192.0.2.0/24"
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

	subnets, err := dbmodel.GetSubnetsByPrefix(db, "192.0.2.0/24")
	require.NoError(t, err)
	require.Len(t, subnets, 1)

	fa := agentcommtest.NewFakeAgents(func(callNo int, cmdResponses []interface{}) {
		mockStatusError("subnet4-update", cmdResponses)
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

	// Begin transaction. It will be needed for the actual part of the
	// test that relies on the existence of the transaction.
	params := dhcp.UpdateSubnetBeginParams{
		SubnetID: subnets[0].ID,
	}
	rsp := rapi.UpdateSubnetBegin(ctx, params)
	require.IsType(t, &dhcp.UpdateSubnetBeginOK{}, rsp)
	okRsp := rsp.(*dhcp.UpdateSubnetBeginOK)
	contents := okRsp.Payload

	// Capture transaction ID.
	transactionID := contents.ID
	require.NotZero(t, transactionID)

	// Submit transaction without the subnet information.
	t.Run("no subnet", func(t *testing.T) {
		params := dhcp.UpdateSubnetSubmitParams{
			ID:     transactionID,
			Subnet: nil,
		}
		rsp := rapi.UpdateSubnetSubmit(ctx, params)
		require.IsType(t, &dhcp.UpdateSubnetSubmitDefault{}, rsp)
		defaultRsp := rsp.(*dhcp.UpdateSubnetSubmitDefault)
		require.Equal(t, http.StatusBadRequest, getStatusCode(*defaultRsp))
	})

	// Submit transaction with non-matching transaction ID.
	t.Run("wrong transaction id", func(t *testing.T) {
		params := dhcp.UpdateSubnetSubmitParams{
			ID: transactionID + 1,
			Subnet: &models.Subnet{
				ID:     1,
				Subnet: "192.0.2.0/24",
				LocalSubnets: []*models.LocalSubnet{
					{
						DaemonID: dbapps[0].Daemons[0].ID,
					},
				},
			},
		}
		rsp := rapi.UpdateSubnetSubmit(ctx, params)
		require.IsType(t, &dhcp.UpdateSubnetSubmitDefault{}, rsp)
		defaultRsp := rsp.(*dhcp.UpdateSubnetSubmitDefault)
		require.Equal(t, http.StatusNotFound, getStatusCode(*defaultRsp))
		require.Equal(t, "Transaction expired for the subnet update", *defaultRsp.Payload.Message)
	})

	// Submit transaction with a subnet that is not associated with any daemons.
	// It simulates a failure in "apply" step which typically is caused by
	// some internal server problem rather than malformed request.
	t.Run("no daemons in subnet", func(t *testing.T) {
		params := dhcp.UpdateSubnetSubmitParams{
			ID: transactionID,
			Subnet: &models.Subnet{
				ID:           subnets[0].ID,
				Subnet:       "192.0.2.0/24",
				LocalSubnets: []*models.LocalSubnet{},
			},
		}
		rsp := rapi.UpdateSubnetSubmit(ctx, params)
		require.IsType(t, &dhcp.UpdateSubnetSubmitDefault{}, rsp)
		defaultRsp := rsp.(*dhcp.UpdateSubnetSubmitDefault)
		require.Equal(t, http.StatusInternalServerError, getStatusCode(*defaultRsp))
		require.Equal(t, "Problem with applying subnet information: applied subnet 192.0.2.0/24 is not associated with any daemon", *defaultRsp.Payload.Message)
	})

	// Submit transaction with valid ID and subnet but simulate an error
	// response from Kea.
	t.Run("commit failure", func(t *testing.T) {
		params := dhcp.UpdateSubnetSubmitParams{
			ID: transactionID,
			Subnet: &models.Subnet{
				LocalSubnets: []*models.LocalSubnet{
					{
						DaemonID: dbapps[0].Daemons[0].ID,
					},
				},
			},
		}
		rsp := rapi.UpdateSubnetSubmit(ctx, params)
		require.IsType(t, &dhcp.UpdateSubnetSubmitDefault{}, rsp)
		defaultRsp := rsp.(*dhcp.UpdateSubnetSubmitDefault)
		require.Equal(t, http.StatusConflict, getStatusCode(*defaultRsp))
		require.Equal(t, fmt.Sprintf("Problem with committing subnet information: subnet4-update command to %s failed: error status (1) returned by Kea dhcp4 daemon with text: 'unable to communicate with the daemon'", app.GetName()),
			*defaultRsp.Payload.Message)
	})
}

// Test that the transaction to update a subnet can be canceled, resulting
// in the removal of this transaction from the config manager and allowing
// another user to apply config updates.
func TestUpdateSubnetBeginCancel(t *testing.T) {
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

	subnets, err := dbmodel.GetSubnetsByPrefix(db, "2001:db8:1::/64")
	require.NoError(t, err)
	require.Len(t, subnets, 1)

	networks, err := dbmodel.GetAllSharedNetworks(db, 6)
	require.NoError(t, err)
	require.Len(t, networks, 1)

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
	params := dhcp.UpdateSubnetBeginParams{
		SubnetID: subnets[0].ID,
	}
	rsp := rapi.UpdateSubnetBegin(ctx, params)
	require.IsType(t, &dhcp.UpdateSubnetBeginOK{}, rsp)
	okRsp := rsp.(*dhcp.UpdateSubnetBeginOK)
	contents := okRsp.Payload

	// Make sure the server returned transaction ID, subnet, daemons and client classes.
	transactionID := contents.ID
	require.NotZero(t, transactionID)
	require.NotNil(t, contents.Subnet)
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
	rsp = rapi.UpdateSubnetBegin(ctx2, params)
	require.IsType(t, &dhcp.UpdateSubnetBeginDefault{}, rsp)
	defaultRsp := rsp.(*dhcp.UpdateSubnetBeginDefault)
	require.Equal(t, http.StatusLocked, getStatusCode(*defaultRsp))

	// Cancel the transaction.
	params2 := dhcp.UpdateSubnetDeleteParams{
		ID: transactionID,
	}
	rsp2 := rapi.UpdateSubnetDelete(ctx, params2)
	require.IsType(t, &dhcp.UpdateSubnetDeleteOK{}, rsp2)

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
	rsp = rapi.UpdateSubnetBegin(ctx2, params)
	require.IsType(t, &dhcp.UpdateSubnetBeginOK{}, rsp)
}

// Test successfully deleting a subnet.
func TestDeleteSubnet(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	serverConfig := `{
		"Dhcp4": {
			"subnet4": [
				{
					"id": 1,
					"subnet": "192.0.2.0/24"
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

	subnets, err := dbmodel.GetSubnetsByPrefix(db, "192.0.2.0/24")
	require.NoError(t, err)
	require.Len(t, subnets, 1)

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

	// Attempt to delete the subnet.
	params := dhcp.DeleteSubnetParams{
		ID: subnets[0].ID,
	}
	rsp := rapi.DeleteSubnet(ctx, params)
	require.IsType(t, &dhcp.DeleteSubnetOK{}, rsp)

	// The subnet4-del and config-write commands should be sent to two Kea servers.
	require.Len(t, fa.RecordedCommands, 4)

	for i, c := range fa.RecordedCommands {
		switch {
		case i < 2:
			require.JSONEq(t, `{
				"command": "subnet4-del",
				"service": ["dhcp4"],
				"arguments": {
					"id": 1
				}
		}`, c.Marshal())
		default:
			require.JSONEq(t, `{
				"command": "config-write",
				"service": ["dhcp4"]
			}`, c.Marshal())
		}
	}
	returnedSubnet, err := dbmodel.GetSubnet(db, subnets[0].ID)
	require.NoError(t, err)
	require.Nil(t, returnedSubnet)
}

// Test error cases for deleting a subnet.
func TestDeleteSubnetError(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Setup fake agents that return an error in response to subnet4-del
	// command.
	fa := agentcommtest.NewFakeAgents(func(callNo int, cmdResponses []interface{}) {
		mockStatusError("subnet4-del", cmdResponses)
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

	subnets, err := dbmodel.GetSubnetsByPrefix(db, "192.0.2.0/24")
	require.NoError(t, err)
	require.Len(t, subnets, 1)

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
	t.Run("wrong subnet id", func(t *testing.T) {
		params := dhcp.DeleteSubnetParams{
			ID: 19809865,
		}
		rsp := rapi.DeleteSubnet(ctx, params)
		require.IsType(t, &dhcp.DeleteSubnetDefault{}, rsp)
		defaultRsp := rsp.(*dhcp.DeleteSubnetDefault)
		require.Equal(t, http.StatusNotFound, getStatusCode(*defaultRsp))
		require.Equal(t, "Cannot find a subnet with ID 19809865", *defaultRsp.Payload.Message)
	})

	// Submit transaction with valid ID but expect the agent to return an
	// error code. This is considered a conflict with the state of the
	// Kea servers.
	t.Run("commit failure", func(t *testing.T) {
		params := dhcp.DeleteSubnetParams{
			ID: subnets[0].ID,
		}
		rsp := rapi.DeleteSubnet(ctx, params)
		require.IsType(t, &dhcp.DeleteSubnetDefault{}, rsp)
		defaultRsp := rsp.(*dhcp.DeleteSubnetDefault)
		require.Equal(t, http.StatusConflict, getStatusCode(*defaultRsp))
		require.Equal(t, fmt.Sprintf("Problem with deleting a subnet: network4-subnet-del command to %s failed: error status (1) returned by Kea dhcp4 daemon with text: 'unable to communicate with the daemon'", app.GetName()),
			*defaultRsp.Payload.Message)
	})
}
