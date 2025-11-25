package restservice

import (
	"context"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"isc.org/stork/datamodel/daemonname"
	"isc.org/stork/datamodel/protocoltype"
	agentcommtest "isc.org/stork/server/agentcomm/test"
	dbmodel "isc.org/stork/server/database/model"
	dbtest "isc.org/stork/server/database/test"
	"isc.org/stork/server/gen/restapi/operations/search"
)

// Check searching via rest api functions.
func TestSearchRecords(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	fa := agentcommtest.NewFakeAgents(nil, nil)
	rapi, err := NewRestAPI(dbSettings, db, fa)
	require.NoError(t, err)
	ctx := context.Background()

	// search with empty text
	params := search.SearchRecordsParams{}
	rsp := rapi.SearchRecords(ctx, params)
	require.IsType(t, &search.SearchRecordsOK{}, rsp)
	okRsp := rsp.(*search.SearchRecordsOK)
	require.Len(t, okRsp.Payload.Apps.Items, 0)
	require.Zero(t, okRsp.Payload.Apps.Total)
	require.Len(t, okRsp.Payload.Groups.Items, 0)
	require.Zero(t, okRsp.Payload.Groups.Total)
	require.Len(t, okRsp.Payload.Hosts.Items, 0)
	require.Zero(t, okRsp.Payload.Hosts.Total)
	require.Len(t, okRsp.Payload.Machines.Items, 0)
	require.Zero(t, okRsp.Payload.Machines.Total)
	require.Len(t, okRsp.Payload.SharedNetworks.Items, 0)
	require.Zero(t, okRsp.Payload.SharedNetworks.Total)
	require.Len(t, okRsp.Payload.Subnets.Items, 0)
	require.Zero(t, okRsp.Payload.Subnets.Total)
	require.Len(t, okRsp.Payload.Users.Items, 0)
	require.Zero(t, okRsp.Payload.Users.Total)

	// add machine
	m := &dbmodel.Machine{
		Address:    "localhost",
		AgentPort:  8080,
		Authorized: true,
	}
	err = dbmodel.AddMachine(db, m)
	require.NoError(t, err)

	unauthorized := &dbmodel.Machine{
		Address:    "localhost",
		AgentPort:  8980,
		Authorized: false,
	}
	err = dbmodel.AddMachine(db, unauthorized)
	require.NoError(t, err)

	// add daemon kea with dhcp4 to machine
	accessPoint1 := &dbmodel.AccessPoint{
		Type:     dbmodel.AccessPointControl,
		Address:  "",
		Port:     1114,
		Key:      "",
		Protocol: protocoltype.HTTPS,
	}

	d4 := dbmodel.NewDaemon(m, daemonname.DHCPv4, true, []*dbmodel.AccessPoint{accessPoint1})
	config4 := []byte(`{
		"Dhcp4": {
			"subnet4": [{
				"id": 1,
				"subnet": "192.168.0.0/24",
				"pools": [{
					"pool": "192.168.0.1-192.168.0.100"
				}, {
					"pool": "192.168.0.150-192.168.0.200"
				}]
			}]
		}
	}`)
	err = d4.SetKeaConfigFromJSON(config4)
	require.NoError(t, err)
	err = dbmodel.AddDaemon(db, d4)
	require.NoError(t, err)

	appSubnets := []dbmodel.Subnet{
		{
			Prefix: "192.168.0.0/24",
			LocalSubnets: []*dbmodel.LocalSubnet{
				{
					DaemonID: d4.ID,
					AddressPools: []dbmodel.AddressPool{
						{
							LowerBound: "192.168.0.1",
							UpperBound: "192.168.0.100",
						},
						{
							LowerBound: "192.168.0.150",
							UpperBound: "192.168.0.200",
						},
					},
				},
			},
		},
	}

	_, err = dbmodel.CommitNetworksIntoDB(db, []dbmodel.SharedNetwork{}, appSubnets)
	require.NoError(t, err)

	// add daemon kea with dhcp6 to machine
	accessPoint6 := &dbmodel.AccessPoint{
		Type:     dbmodel.AccessPointControl,
		Address:  "",
		Port:     1116,
		Key:      "",
		Protocol: protocoltype.HTTP,
	}

	d6 := dbmodel.NewDaemon(m, daemonname.DHCPv6, true, []*dbmodel.AccessPoint{accessPoint6})
	config6 := []byte(`{
		"Dhcp6": {
			"subnet6": [{
				"id": 2,
				"subnet": "2001:db8:1::/64",
				"pools": []
			}]
		}
	}`)
	err = d6.SetKeaConfigFromJSON(config6)
	require.NoError(t, err)
	err = dbmodel.AddDaemon(db, d6)
	require.NoError(t, err)

	appSubnets = []dbmodel.Subnet{
		{
			Prefix: "2001:db8:1::/64",
		},
	}
	_, err = dbmodel.CommitNetworksIntoDB(db, []dbmodel.SharedNetwork{}, appSubnets)
	require.NoError(t, err)

	// add additional daemons kea with dhcp4 and dhcp6 to machine
	accessPoint46_4 := &dbmodel.AccessPoint{
		Type:     dbmodel.AccessPointControl,
		Address:  "",
		Port:     1146,
		Key:      "",
		Protocol: protocoltype.HTTPS,
	}

	d46_4 := dbmodel.NewDaemon(m, daemonname.DHCPv4, true, []*dbmodel.AccessPoint{accessPoint46_4})
	config46_4 := []byte(`{
		"Dhcp4": {
			"subnet4": [{
				"id": 3,
				"subnet": "192.118.0.0/24",
				"pools": [{
					"pool": "192.118.0.1-192.118.0.200"
				}]
			}]
		}
	}`)
	err = d46_4.SetKeaConfigFromJSON(config46_4)
	require.NoError(t, err)
	err = dbmodel.AddDaemon(db, d46_4)
	require.NoError(t, err)

	accessPoint46_6 := &dbmodel.AccessPoint{
		Type:     dbmodel.AccessPointControl,
		Address:  "",
		Port:     1147,
		Key:      "",
		Protocol: protocoltype.HTTPS,
	}

	d46_6 := dbmodel.NewDaemon(m, daemonname.DHCPv6, true, []*dbmodel.AccessPoint{accessPoint46_6})
	config46_6 := []byte(`{
		"Dhcp6": {
			"subnet6": [{
				"id": 4,
				"subnet": "3001:db8:1::/64",
				"pools": [{
					"pool": "3001:db8:1::/80"
				}]
			}],
			"shared-networks": [{
				"name": "fox",
				"subnet6": [{
					"id": 21,
					"subnet": "5001:db8:1::/64"
				}]
			}]
		}
	}`)
	err = d46_6.SetKeaConfigFromJSON(config46_6)
	require.NoError(t, err)
	err = dbmodel.AddDaemon(db, d46_6)
	require.NoError(t, err)

	appNetworks := []dbmodel.SharedNetwork{
		{
			Name:   "fox",
			Family: 6,
			Subnets: []dbmodel.Subnet{
				{
					Prefix: "5001:db8:1::/64",
					LocalSubnets: []*dbmodel.LocalSubnet{
						{
							DaemonID: d46_6.ID,
						},
					},
				},
			},
			LocalSharedNetworks: []*dbmodel.LocalSharedNetwork{
				{
					DaemonID: d46_6.ID,
				},
			},
		},
	}

	appSubnets = []dbmodel.Subnet{
		{
			Prefix: "192.118.0.0/24",
			LocalSubnets: []*dbmodel.LocalSubnet{
				{
					DaemonID: d46_4.ID,
					AddressPools: []dbmodel.AddressPool{
						{
							LowerBound: "192.118.0.1",
							UpperBound: "192.118.0.200",
						},
					},
				},
			},
		},
		{
			Prefix: "3001:db8:1::/64",
			LocalSubnets: []*dbmodel.LocalSubnet{
				{
					DaemonID: d46_6.ID,
					AddressPools: []dbmodel.AddressPool{
						{
							LowerBound: "3001:db8:1::",
							UpperBound: "3001:db8:1:0:ffff::ffff",
						},
					},
				},
			},
		},
	}
	_, err = dbmodel.CommitNetworksIntoDB(db, []dbmodel.SharedNetwork{}, []dbmodel.Subnet{appSubnets[0]})
	require.NoError(t, err)
	_, err = dbmodel.CommitNetworksIntoDB(db, []dbmodel.SharedNetwork{appNetworks[0]}, []dbmodel.Subnet{appSubnets[1]})
	require.NoError(t, err)

	// search for 'fox' - shared network and subnet are expected
	text := "fox"
	params = search.SearchRecordsParams{
		Text: &text,
	}
	rsp = rapi.SearchRecords(ctx, params)
	require.IsType(t, &search.SearchRecordsOK{}, rsp)
	okRsp = rsp.(*search.SearchRecordsOK)
	require.Len(t, okRsp.Payload.Apps.Items, 0)
	require.Zero(t, okRsp.Payload.Apps.Total)
	require.Len(t, okRsp.Payload.Groups.Items, 0)
	require.Zero(t, okRsp.Payload.Groups.Total)
	require.Len(t, okRsp.Payload.Hosts.Items, 0)
	require.Zero(t, okRsp.Payload.Hosts.Total)
	require.Len(t, okRsp.Payload.Machines.Items, 0)
	require.Zero(t, okRsp.Payload.Machines.Total)
	require.Len(t, okRsp.Payload.SharedNetworks.Items, 1)
	require.EqualValues(t, 1, okRsp.Payload.SharedNetworks.Total)
	require.Len(t, okRsp.Payload.Subnets.Items, 1)
	require.EqualValues(t, 1, okRsp.Payload.Subnets.Total)
	require.Len(t, okRsp.Payload.Users.Items, 0)
	require.Zero(t, okRsp.Payload.Users.Total)

	// search for '192.118.0.0/24' - subnet is expected
	text = "192.118.0.0/24"
	params = search.SearchRecordsParams{
		Text: &text,
	}
	rsp = rapi.SearchRecords(ctx, params)
	require.IsType(t, &search.SearchRecordsOK{}, rsp)
	okRsp = rsp.(*search.SearchRecordsOK)
	require.Len(t, okRsp.Payload.Apps.Items, 0)
	require.Zero(t, okRsp.Payload.Apps.Total)
	require.Len(t, okRsp.Payload.Groups.Items, 0)
	require.Zero(t, okRsp.Payload.Groups.Total)
	require.Len(t, okRsp.Payload.Hosts.Items, 0)
	require.Zero(t, okRsp.Payload.Hosts.Total)
	require.Len(t, okRsp.Payload.Machines.Items, 0)
	require.Zero(t, okRsp.Payload.Machines.Total)
	require.Len(t, okRsp.Payload.SharedNetworks.Items, 0)
	require.Zero(t, okRsp.Payload.SharedNetworks.Total)
	require.Len(t, okRsp.Payload.Subnets.Items, 1)
	require.EqualValues(t, 1, okRsp.Payload.Subnets.Total)
	require.Len(t, okRsp.Payload.Users.Items, 0)
	require.Zero(t, okRsp.Payload.Users.Total)

	// search for 'super' - group is expected
	text = "super"
	params = search.SearchRecordsParams{
		Text: &text,
	}
	rsp = rapi.SearchRecords(ctx, params)
	require.IsType(t, &search.SearchRecordsOK{}, rsp)
	okRsp = rsp.(*search.SearchRecordsOK)
	require.Len(t, okRsp.Payload.Apps.Items, 0)
	require.Zero(t, okRsp.Payload.Apps.Total)
	require.Len(t, okRsp.Payload.Groups.Items, 1)
	require.EqualValues(t, 1, okRsp.Payload.Groups.Total)
	require.Len(t, okRsp.Payload.Hosts.Items, 0)
	require.Zero(t, okRsp.Payload.Hosts.Total)
	require.Len(t, okRsp.Payload.Machines.Items, 0)
	require.Zero(t, okRsp.Payload.Machines.Total)
	require.Len(t, okRsp.Payload.SharedNetworks.Items, 0)
	require.Zero(t, okRsp.Payload.SharedNetworks.Total)
	require.Len(t, okRsp.Payload.Subnets.Items, 0)
	require.Zero(t, okRsp.Payload.Subnets.Total)
	require.Len(t, okRsp.Payload.Users.Items, 0)
	require.Zero(t, okRsp.Payload.Users.Total)

	// search for 'admin' - user and group are expected
	text = "admin"
	params = search.SearchRecordsParams{
		Text: &text,
	}
	rsp = rapi.SearchRecords(ctx, params)
	require.IsType(t, &search.SearchRecordsOK{}, rsp)
	okRsp = rsp.(*search.SearchRecordsOK)
	require.Len(t, okRsp.Payload.Apps.Items, 0)
	require.Zero(t, okRsp.Payload.Apps.Total)
	require.Len(t, okRsp.Payload.Groups.Items, 2)
	require.EqualValues(t, 2, okRsp.Payload.Groups.Total)
	require.Len(t, okRsp.Payload.Hosts.Items, 0)
	require.Zero(t, okRsp.Payload.Hosts.Total)
	require.Len(t, okRsp.Payload.Machines.Items, 0)
	require.Zero(t, okRsp.Payload.Machines.Total)
	require.Len(t, okRsp.Payload.SharedNetworks.Items, 0)
	require.Zero(t, okRsp.Payload.SharedNetworks.Total)
	require.Len(t, okRsp.Payload.Subnets.Items, 0)
	require.Zero(t, okRsp.Payload.Subnets.Total)
	require.Len(t, okRsp.Payload.Users.Items, 1)
	require.EqualValues(t, 1, okRsp.Payload.Users.Total)

	// search for 'localhost' - 2 machines are expected; one authorized and one unauthorized
	text = "localhost"
	params = search.SearchRecordsParams{
		Text: &text,
	}
	rsp = rapi.SearchRecords(ctx, params)
	require.IsType(t, &search.SearchRecordsOK{}, rsp)
	okRsp = rsp.(*search.SearchRecordsOK)
	require.Len(t, okRsp.Payload.Apps.Items, 4)
	require.EqualValues(t, 4, okRsp.Payload.Apps.Total)
	require.Len(t, okRsp.Payload.Groups.Items, 0)
	require.Zero(t, okRsp.Payload.Groups.Total)
	require.Len(t, okRsp.Payload.Hosts.Items, 0)
	require.Zero(t, okRsp.Payload.Hosts.Total)
	require.Len(t, okRsp.Payload.Machines.Items, 2)
	require.EqualValues(t, 2, okRsp.Payload.Machines.Total)
	require.Len(t, okRsp.Payload.SharedNetworks.Items, 0)
	require.Zero(t, okRsp.Payload.SharedNetworks.Total)
	require.Len(t, okRsp.Payload.Subnets.Items, 0)
	require.Zero(t, okRsp.Payload.Subnets.Total)
	require.Len(t, okRsp.Payload.Users.Items, 0)
	require.Zero(t, okRsp.Payload.Users.Total)

	// search for 'dhcp' - all daemons are expected
	text = "dhcp"
	params = search.SearchRecordsParams{
		Text: &text,
	}
	rsp = rapi.SearchRecords(ctx, params)
	require.IsType(t, &search.SearchRecordsOK{}, rsp)
	okRsp = rsp.(*search.SearchRecordsOK)
	require.Len(t, okRsp.Payload.Apps.Items, 4)
	require.EqualValues(t, 4, okRsp.Payload.Apps.Total)
	require.Len(t, okRsp.Payload.Groups.Items, 0)
	require.Zero(t, okRsp.Payload.Groups.Total)
	require.Len(t, okRsp.Payload.Hosts.Items, 0)
	require.Zero(t, okRsp.Payload.Hosts.Total)
	require.Len(t, okRsp.Payload.Machines.Items, 0)
	require.Zero(t, okRsp.Payload.Machines.Total)
	require.Len(t, okRsp.Payload.SharedNetworks.Items, 0)
	require.Zero(t, okRsp.Payload.SharedNetworks.Total)
	require.Len(t, okRsp.Payload.Subnets.Items, 0)
	require.Zero(t, okRsp.Payload.Subnets.Total)
	require.Len(t, okRsp.Payload.Users.Items, 0)
	require.Zero(t, okRsp.Payload.Users.Total)
}

// Check handing error in search.
func TestSearchErrorHandling(t *testing.T) {
	err := errors.New("some error")
	rsp := handleSearchError(err, "some msg")
	require.EqualValues(t, "some msg", *rsp.(*search.SearchRecordsDefault).Payload.Message)
}
