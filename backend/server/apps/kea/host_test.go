package kea

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	require "github.com/stretchr/testify/require"
	keaconfig "isc.org/stork/appcfg/kea"
	keactrl "isc.org/stork/appctrl/kea"
	agentcommtest "isc.org/stork/server/agentcomm/test"
	"isc.org/stork/server/configreview"
	dbmodel "isc.org/stork/server/database/model"
	dbtest "isc.org/stork/server/database/test"
	storktest "isc.org/stork/server/test/dbmodel"
)

// Returns test Kea configuration including multiple IPv4 subnets.
func getTestConfigWithIPv4Subnets(t *testing.T, hostCmds bool) *dbmodel.KeaConfig {
	configStr := `{
        "Dhcp4": {
            "subnet4": [
                {
                    "id": 234,
                    "subnet": "192.0.3.0/24"
                },
                {
                    "id": 345,
                    "subnet": "192.0.4.0/24"
                },
                {
                    "id": 567,
                    "subnet": "192.0.5.0/24"
                },
                {
                    "id": 678,
                    "subnet": "192.0.6.0/24"
                }
            ]%s
        }
    }`
	if hostCmds {
		hooks := `, "hooks-libraries": [
            {
                "library": "libdhcp_host_cmds.so"
            }
        ]`
		configStr = fmt.Sprintf(configStr, hooks)
	} else {
		configStr = fmt.Sprintf(configStr, "")
	}

	cfg, err := dbmodel.NewKeaConfigFromJSON(configStr)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	return cfg
}

// Returns test Kea configuration including one IPv4 subnet.
func getTestConfigWithOneIPv4Subnet(t *testing.T) *dbmodel.KeaConfig {
	configStr := `{
        "Dhcp4": {
            "subnet4": [
                {
                    "id": 123,
                    "subnet": "192.0.2.0/24"
                }
            ],
            "hooks-libraries": [
                {
                    "library": "libdhcp_host_cmds.so"
                }
            ]
        }
    }`

	cfg, err := dbmodel.NewKeaConfigFromJSON(configStr)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	return cfg
}

// Returns test Kea configuration including one IPv6 subnet.
func getTestConfigWithOneIPv6Subnet(t *testing.T) *dbmodel.KeaConfig {
	configStr := `{
        "Dhcp6": {
            "subnet6": [
                {
                    "id": 234,
                    "subnet": "2001:db8:3::/64"
                }
            ],
            "hooks-libraries": [
                {
                    "library": "libdhcp_host_cmds.so"
                }
            ]
        }
    }`

	cfg, err := dbmodel.NewKeaConfigFromJSON(configStr)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	return cfg
}

// Returns test Kea configuration including multiple IPv6 subnets.
func getTestConfigWithIPv6Subnets(t *testing.T) *dbmodel.KeaConfig {
	configStr := `{
        "Dhcp6": {
            "subnet6": [
                {
                    "id": 234,
                    "subnet": "2001:db8:3::/64"
                },
                {
                    "id": 345,
                    "subnet": "2001:db8:4::/64"
                },
                {
                    "id": 567,
                    "subnet": "2001:db8:5::/64"
                },
                {
                    "id": 678,
                    "subnet": "2001:db8:6::/64"
                }
            ],
            "hooks-libraries": [
                {
                    "library": "libdhcp_host_cmds.so"
                }
            ]
        }
    }`

	cfg, err := dbmodel.NewKeaConfigFromJSON(configStr)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	return cfg
}

// Returns test Kea configuration including global host reservations.
func getTestConfigWithIPv4GlobalHosts(t *testing.T) *dbmodel.KeaConfig {
	configStr := `{
        "Dhcp4": {
            "reservations": [
                {
                    "hw-address": "aa:bb:cc:dd:ee:ff",
                    "ip-address": "192.0.2.10",
                    "hostname": "abc.example.org"
                },
                {
                    "hw-address": "ff:ff:ff:ff:ff:ff",
                    "ip-address": "192.0.2.11",
                    "hostname": "foo.example.org"
                }
            ],
            "hooks-libraries": [
                {
                    "library": "libdhcp_host_cmds.so"
                }
            ]
        }
    }`

	cfg, err := dbmodel.NewKeaConfigFromJSON(configStr)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	return cfg
}

// Returns test Kea configuration including global host reservations.
func getTestConfigWithIPv6GlobalHosts(t *testing.T) *dbmodel.KeaConfig {
	configStr := `{
        "Dhcp6": {
            "reservations": [
                {
                    "duid": "aa:bb:cc:dd",
                    "ip-addresses": [ "2001:db8:1::10" ],
                    "prefixes": [ "3000::/64" ],
                    "hostname": "abc.example.org"
                },
                {
                    "duid": "ff:ff:ff:ff",
                    "ip-addresses": [ "2001:db8:1::11" ],
                    "prefixes": [ "3001::/64" ],
                    "hostname": "foo.example.org",
					"option-data": [
						{
							"code": 1024,
							"space": "dhcp6"
						},
						{
							"code": 1025,
							"space": "option-1024"
						},
						{
							"code": 1026,
							"space": "option-1024.1025"
						},
						{
							"code": 1027,
							"space": "option-1024.1025.1026"
						}
					]
                }
            ],
            "hooks-libraries": [
                {
                    "library": "libdhcp_host_cmds.so"
                }
            ]
        }
    }`

	cfg, err := dbmodel.NewKeaConfigFromJSON(configStr)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	return cfg
}

// This function mocks the response of the Kea servers to the reservation-get-page
// command. This command fetches host reservations in chunks from the servers.
// The detailed explanations how this function works are provided within the
// function body.
func mockReservationGetPage(callNo int, cmdResponses []interface{}) {
	// First 15 calls are for IPv4 reservations.
	family := 4
	if callNo/(3*5) > 0 {
		// Next 15 calls are for IPv6 reservations.
		family = 6
	}

	// For each server type we expect 5 subnets and 3 calls for each. Let's
	// reset callNo when we're done with all DHCPv4 specific calls.
	// Then we can use callNo for DHCPv6 in the same way as we use for
	// DHCPv4.
	callNo %= 3 * 5
	// Each subnet contains 10 reservations and we fetch them by chunks of 5.
	// This means that we should get 5 reservations for the first call, 5
	// reservations for the second call and no reservations for the third call
	// for a given subnet. That's why if the call number modulo 3 is 0 we should
	// send an empty response, otherwise we send non-empty response with hosts.
	nonEmptyResponse := (callNo+1)%3 > 0
	// This is the default from value we send back, but it may be modified
	// depending on the current state.
	fromValue := 0
	// We make three calls to get reservations for a particular subnet. Therefore
	// the index of the current subnet is equal to call number divided by 3.
	currentSubnet := callNo / 3
	// Source index indicates which database holds the reservations. Initially
	// it is the first database and we simulate switching to the second
	// database after 3 subnets.
	sourceIndex := currentSubnet/3 + 1

	// By default return no hosts.
	hostsAsJSON := []byte("[ ]")
	hosts := []keaconfig.Reservation{}

	if nonEmptyResponse {
		// Fill in the response with 5 host reservations belonging to a given
		// subnet.
		for i := 0; i < 5; i++ {
			host := keaconfig.Reservation{
				HWAddress: fmt.Sprintf("01:02:03:04:05:%02d", callNo*5+i),
				Hostname:  fmt.Sprintf("host%02d", callNo*5+i),
			}
			switch family {
			case 4:
				// Starting from 192.0.2.10, then 192.0.2.11 .. up to 192.0.2.19.
				// Then 192.0.3.10 etc.
				host.IPAddress = fmt.Sprintf("192.0.%d.%d", 2+currentSubnet, callNo%3*5+10+i)
			default:
				// Starting from 2001:db8:2::10, then 2001:db8:2::11 etc.
				host.IPAddresses = append(host.IPAddresses,
					fmt.Sprintf("2001:db8:%d::%d", 2+currentSubnet, callNo%3*5+10+i))
			}
			hosts = append(hosts, host)
		}
		// Turn the hosts into the JSON representation. The magic numbers 16 and 4 are
		// to make it look better with indentation in case some debugging is needed.
		hostsAsJSON, _ = json.MarshalIndent(hosts, strings.Repeat(" ", 16), strings.Repeat(" ", 4))
		// The from value should be copied in the next request. For the first request
		// within a given subnet it is set to 5. For the second request it is set to 10
		// to mark the last host returned.
		fromValue = callNo*5%15 + len(hosts)
	}

	// Generate the response with filling in the values as appropriate.
	json := []byte(fmt.Sprintf(`[
        {
            "result": 0,
            "text": "Hosts found",
            "arguments": {
                "count": %d,
                "hosts": %s,
                "next": {
                    "from": %d,
                    "source-index": %d
                }
            }
        }
    ]`, len(hosts), string(hostsAsJSON), fromValue, sourceIndex))

	command := keactrl.NewCommandBase(keactrl.ReservationGetPage, fmt.Sprintf("dhcp%d", family))

	_ = keactrl.UnmarshalResponseList(command, json, cmdResponses[0])
}

// This function mocks the response of the Kea servers to the reservation-get-page
// command. It should be used to test cases that the second attempt to fetch hosts
// reduces the number of hosts in the database.
func mockReservationGetPageReduceHosts(callNo int, cmdResponses []interface{}) {
	var json string
	switch callNo {
	case 1:
		json = `[
            {
                "result": 0,
                "text": "Hosts found",
                "arguments": {
                    "count": 1,
                    "hosts": [
                        {
                            "hw-address": "01:02:03:04:05:06",
                            "ip-address": "192.0.2.10"
                        },
                        {
                            "hw-address": "01:02:03:04:05:07",
                            "ip-address": "192.0.2.11"
                        }
                    ],
                    "next": {
                        "from": 0,
                        "source-index": 1
                    }
                }
            }
        ]`
	case 0, 2, 3, 5:
		json = `[
            {
                "result": 0,
                "text": "Hosts found",
                "arguments": {
                    "count": 0,
                    "hosts": [ ],
                    "next": {
                        "from": 0,
                        "source-index": 1
                    }
                }
            }
        ]`
	case 4:
		json = `[
            {
                "result": 0,
                "text": "Hosts found",
                "arguments": {
                    "count": 1,
                    "hosts": [
                        {
                            "hw-address": "01:02:03:04:05:07",
                            "ip-address": "192.0.2.11"
                        }
                    ],
                    "next": {
                        "from": 0,
                        "source-index": 1
                    }
                }
            }
        ]`
	}

	command := keactrl.NewCommandBase(keactrl.ReservationGetPage, keactrl.DHCPv4)

	_ = keactrl.UnmarshalResponseList(command, []byte(json), cmdResponses[0])
	fmt.Printf("cmdResponses[0]: %+v\n", cmdResponses[0])
}

// This function mocks the response of the Kea server to the reservation-get-page
// command. It is used to test the case when host reservations returned in the
// first response to the reservation-get-page hasn't changed but the reservations
// in the second response have changed.
func mockReservationGetPagePartialChange(callNo int, cmdResponses []interface{}) {
	var json string
	switch callNo {
	// No global hosts, third response returns empty response terminating the
	// process of fetching the hosts. The same for the second iteration.
	case 0, 3, 4, 7:
		json = `[
            {
                "result": 0,
                "text": "Hosts found",
                "arguments": {
                    "count": 0,
                    "hosts": [ ],
                    "next": {
                        "from": 0,
                        "source-index": 1
                    }
                }
            }
        ]`
	// Two hosts in the first response that don't change in the second
	// iteration.
	case 1, 5:
		json = `[
            {
                "result": 0,
                "text": "Hosts found",
                "arguments": {
                    "count": 2,
                    "hosts": [
                        {
                            "hw-address": "01:02:03:04:05:06",
                            "ip-address": "192.0.2.10"
                        },
                        {
                            "hw-address": "01:02:03:04:05:07",
                            "ip-address": "192.0.2.11"
                        }
                    ],
                    "next": {
                        "from": 2,
                        "source-index": 1
                    }
                }
            }
        ]`
	// One host in the second response that changes in the next iteration.
	case 2:
		json = `[
            {
                "result": 0,
                "text": "Hosts found",
                "arguments": {
                    "count": 1,
                    "hosts": [
                        {
                            "hw-address": "01:02:03:04:05:08",
                            "ip-address": "192.0.2.12"
                        }
                    ],
                    "next": {
                        "from": 3,
                        "source-index": 1
                    }
                }
            }
        ]`
	// One host in the second response in the second iteration.
	case 6:
		json = `[
            {
                "result": 0,
                "text": "Hosts found",
                "arguments": {
                    "count": 1,
                    "hosts": [
                        {
                            "hw-address": "01:02:03:04:05:09",
                            "ip-address": "192.0.2.13"
                        }
                    ],
                    "next": {
                        "from": 3,
                        "source-index": 1
                    }
                }
            }
        ]`
	}

	command := keactrl.NewCommandBase(keactrl.ReservationGetPage, keactrl.DHCPv4)

	_ = keactrl.UnmarshalResponseList(command, []byte(json), cmdResponses[0])
	fmt.Printf("cmdResponses[0]: %+v\n", cmdResponses[0])
}

// Verifies that the specified host contains the specified host identifier and
// reserved IP address.
func testHost(t *testing.T, reservation interface{}, identifier string, address string) {
	var (
		host *dbmodel.Host
		err  error
	)
	daemon := dbmodel.NewKeaDaemon(dbmodel.DaemonNameDHCPv4, true)
	if r, ok := reservation.(keaconfig.Reservation); ok {
		host, err = dbmodel.NewHostFromKeaConfigReservation(r, daemon, dbmodel.HostDataSourceConfig, dbmodel.NewDHCPOptionDefinitionLookup())
		require.NoError(t, err)
	} else {
		h := reservation.(dbmodel.Host)
		host = &h
	}
	require.NotNil(t, host)
	require.NotEmpty(t, host.LocalHosts)
	require.Len(t, host.LocalHosts[0].IPReservations, 1)
	require.Equal(t, address, host.LocalHosts[0].IPReservations[0].Address)
	require.Len(t, host.HostIdentifiers, 1)

	// If the caller specified colons within the identifier, they have to
	// be removed before we convert it to binary.
	identifier = strings.ReplaceAll(identifier, ":", "")
	identifierBytes, err := hex.DecodeString(identifier)
	require.NoError(t, err)
	require.Equal(t, identifierBytes, host.HostIdentifiers[0].Value)
}

// Tests that valid reservation-get-page received command was received by
// the fake agents.
func testReservationGetPageReceived(t *testing.T, iterator *hostIterator) {
	agents, ok := iterator.agents.(*agentcommtest.FakeAgents)
	require.True(t, ok)
	// This function is not called before first command is sent.
	require.GreaterOrEqual(t, len(agents.RecordedCommands), 1)
	recordedCommand := agents.GetLastCommand()
	// Check that the correct command name was sent.
	require.Equal(t, keactrl.ReservationGetPage, recordedCommand.GetCommand())
	// This command must always include some arguments.
	require.NotNil(t, recordedCommand.Arguments)
	recordedArguments := recordedCommand.Arguments
	// The subnet-id is always required.
	require.Contains(t, recordedArguments.(map[string]interface{}), "subnet-id")
	// The limit is also always required.
	require.Contains(t, recordedArguments, "limit")
	// The limit is configurable and the limit value sent should be the one
	// that has been configured.
	require.EqualValues(t, iterator.limit, (recordedArguments.(map[string]interface{}))["limit"])
	// Check that the service name is correct.
	recordedDaemons := recordedCommand.Daemons
	require.Len(t, recordedDaemons, 1)
	require.Contains(t, recordedDaemons, iterator.daemon.Name)
}

// Tests that host reservations can be extracted from the Kea app's
// configuration.
func TestDetectHostsFromConfig(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	m := &dbmodel.Machine{
		ID:        0,
		Address:   "localhost",
		AgentPort: 8080,
	}
	err := dbmodel.AddMachine(db, m)
	require.NoError(t, err)

	// Creates new app with provided configurations.
	accessPoints := []*dbmodel.AccessPoint{}
	accessPoints = dbmodel.AppendAccessPoint(accessPoints, dbmodel.AccessPointControl, "localhost", "", 8000, false)
	app := dbmodel.App{
		MachineID:    m.ID,
		Type:         dbmodel.AppTypeKea,
		AccessPoints: accessPoints,
		Daemons: []*dbmodel.Daemon{
			{
				Name:   "dhcp4",
				Active: true,
				KeaDaemon: &dbmodel.KeaDaemon{
					Config: getTestConfigWithIPv4GlobalHosts(t),
				},
			},
			{
				Name:   "dhcp6",
				Active: true,
				KeaDaemon: &dbmodel.KeaDaemon{
					Config: getTestConfigWithIPv6GlobalHosts(t),
				},
			},
		},
	}
	// Add the app to the database.
	_, err = dbmodel.AddApp(db, &app)
	require.NoError(t, err)
	app.Machine = m

	var (
		hosts   []dbmodel.Host
		v4hosts []dbmodel.Host
		v6hosts []dbmodel.Host
	)

	lookup := dbmodel.NewDHCPOptionDefinitionLookup()

	// Detect global hosts in the configurations of the app.
	v4hosts, err = detectGlobalHostsFromConfig(db, app.Daemons[0], lookup)
	require.NoError(t, err)
	hosts = append(hosts, v4hosts...)
	v6hosts, err = detectGlobalHostsFromConfig(db, app.Daemons[1], lookup)
	require.NoError(t, err)
	hosts = append(hosts, v6hosts...)
	require.Len(t, hosts, 4)

	for _, h := range hosts {
		// Hosts are global.
		require.Zero(t, h.SubnetID)
		// Each of them has single DHCP identifier.
		require.Len(t, h.HostIdentifiers, 1)
		// The hosts should be associated with the app.
		require.Len(t, h.LocalHosts, 1)
	}

	// Commit the hosts into the database.
	tx, err := db.Begin()
	require.NoError(t, err)
	err = dbmodel.CommitGlobalHostsIntoDB(tx, v4hosts)
	require.NoError(t, err)
	err = dbmodel.CommitGlobalHostsIntoDB(tx, v6hosts)
	require.NoError(t, err)
	err = tx.Commit()
	require.NoError(t, err)

	// Run the detection again.
	hosts = []dbmodel.Host{}
	for i := range app.Daemons {
		detectedHosts, err := detectGlobalHostsFromConfig(db, app.Daemons[i], lookup)
		require.NoError(t, err)
		hosts = append(hosts, detectedHosts...)
	}
	require.Len(t, hosts, 4)

	// Existing hosts should be returned.
	for _, h := range hosts {
		require.Zero(t, h.SubnetID)
		require.Len(t, h.HostIdentifiers, 1)
		// The hosts should have been already associated with our app.
		require.Len(t, h.LocalHosts, 1)
		require.Contains(t, []int64{app.Daemons[0].ID, app.Daemons[1].ID}, h.LocalHosts[0].DaemonID)
	}
}

// Test that global hosts are not committed to the database when all of
// the DHCP daemons have been marked as having the same config since last
// update.
func TestDetectHostsSameConfig(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	fec := &storktest.FakeEventCenter{}

	m := &dbmodel.Machine{
		ID:        0,
		Address:   "localhost",
		AgentPort: 8080,
	}
	err := dbmodel.AddMachine(db, m)
	require.NoError(t, err)

	// Creates new app with provided configurations.
	accessPoints := []*dbmodel.AccessPoint{}
	accessPoints = dbmodel.AppendAccessPoint(accessPoints, dbmodel.AccessPointControl, "localhost", "", 8000, true)
	app := dbmodel.App{
		MachineID:    m.ID,
		Type:         dbmodel.AppTypeKea,
		AccessPoints: accessPoints,
		Daemons: []*dbmodel.Daemon{
			{
				Name:   dbmodel.DaemonNameDHCPv4,
				Active: true,
				KeaDaemon: &dbmodel.KeaDaemon{
					Config: getTestConfigWithIPv4GlobalHosts(t),
				},
			},
			{
				Name:   dbmodel.DaemonNameDHCPv6,
				Active: true,
				KeaDaemon: &dbmodel.KeaDaemon{
					Config: getTestConfigWithIPv6GlobalHosts(t),
				},
			},
		},
	}
	// Add the app to the database.
	_, err = dbmodel.AddApp(db, &app)
	require.NoError(t, err)
	app.Machine = m

	// Indicate that the DHCP configurations haven't changed.
	state := &AppStateMeta{
		SameConfigDaemons: map[string]bool{
			dbmodel.DaemonNameDHCPv4: true,
			dbmodel.DaemonNameDHCPv6: true,
		},
	}

	// Both configurations are indicated to be the same so the hosts should not
	// be committed to the database.
	lookup := dbmodel.NewDHCPOptionDefinitionLookup()
	err = CommitAppIntoDB(db, &app, fec, state, lookup)
	require.NoError(t, err)

	// Make sure that no hosts have been added.
	hosts, err := dbmodel.GetAllHosts(db, 0)
	require.NoError(t, err)
	require.Empty(t, hosts)

	// Indicate that configuration is the same for DHCPv4 but not for DHCPv6.
	state = &AppStateMeta{
		SameConfigDaemons: map[string]bool{
			dbmodel.DaemonNameDHCPv4: true,
		},
	}
	err = CommitAppIntoDB(db, &app, fec, state, lookup)
	require.NoError(t, err)

	// The hosts should have been added for the DHCPv6 daemon.
	hosts, err = dbmodel.GetAllHosts(db, 0)
	require.NoError(t, err)
	require.Len(t, hosts, 2)
}

// Tests that host reservations can be retrieved in chunks from the Kea
// DHCPv4 and DHCPv6 servers.
func TestGetPageFromHostCmds(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	fec := &storktest.FakeEventCenter{}

	m := &dbmodel.Machine{
		ID:        0,
		Address:   "localhost",
		AgentPort: 8080,
	}
	err := dbmodel.AddMachine(db, m)
	require.NoError(t, err)

	// Creates new app with provided configurations.
	accessPoints := []*dbmodel.AccessPoint{}
	accessPoints = dbmodel.AppendAccessPoint(accessPoints, dbmodel.AccessPointControl, "localhost", "", 8000, false)
	app := dbmodel.App{
		MachineID:    m.ID,
		Type:         dbmodel.AppTypeKea,
		AccessPoints: accessPoints,
		Daemons: []*dbmodel.Daemon{
			{
				Name:   "dhcp4",
				Active: true,
				KeaDaemon: &dbmodel.KeaDaemon{
					KeaDHCPDaemon: &dbmodel.KeaDHCPDaemon{},
					Config:        getTestConfigWithIPv4Subnets(t, true),
				},
			},
			{
				Name:   "dhcp6",
				Active: true,
				KeaDaemon: &dbmodel.KeaDaemon{
					KeaDHCPDaemon: &dbmodel.KeaDHCPDaemon{},
					Config:        getTestConfigWithIPv6Subnets(t),
				},
			},
		},
	}
	// Add the app to the database.
	_, err = dbmodel.AddApp(db, &app)
	require.NoError(t, err)
	app.Machine = m

	lookup := dbmodel.NewDHCPOptionDefinitionLookup()
	err = CommitAppIntoDB(db, &app, fec, nil, lookup)
	require.NoError(t, err)

	fa := agentcommtest.NewFakeAgents(mockReservationGetPage, nil)

	it := newHostIterator(db, &app, app.Daemons[0], fa, 5)
	require.NotNil(t, it)

	// Should get addresses 192.0.2.10 thru 192.0.2.14
	hosts, done, err := it.getPageFromHostCmds()
	require.NoError(t, err)
	require.False(t, done)
	require.Len(t, hosts, 5)
	require.EqualValues(t, 5, it.from)
	require.EqualValues(t, 1, it.sourceIndex)
	require.Len(t, it.subnets, 4)
	require.EqualValues(t, -1, it.subnetIndex)
	require.Nil(t, it.getCurrentSubnet())
	require.EqualValues(t, 1, it.trace.getResponseCount())
	testHost(t, hosts[0], "01:02:03:04:05:00", "192.0.2.10")
	testHost(t, hosts[1], "01:02:03:04:05:01", "192.0.2.11")
	testHost(t, hosts[2], "01:02:03:04:05:02", "192.0.2.12")
	testHost(t, hosts[3], "01:02:03:04:05:03", "192.0.2.13")
	testHost(t, hosts[4], "01:02:03:04:05:04", "192.0.2.14")
	testReservationGetPageReceived(t, it)
	require.Zero(t, (fa.GetLastCommand().Arguments.(map[string]interface{})["subnet-id"]))
	require.NotContains(t, (fa.GetLastCommand().Arguments.(map[string]interface{})), "from")
	require.Contains(t, (fa.GetLastCommand().Arguments.(map[string]interface{})), "source-index")

	// Should get addresses 192.0.2.15 thru 192.0.2.19
	hosts, done, err = it.getPageFromHostCmds()
	require.False(t, done)
	require.NoError(t, err)
	require.Len(t, hosts, 5)
	require.EqualValues(t, 10, it.from)
	require.EqualValues(t, 1, it.sourceIndex)
	require.Len(t, it.subnets, 4)
	require.EqualValues(t, -1, it.subnetIndex)
	require.Nil(t, it.getCurrentSubnet())
	require.EqualValues(t, 2, it.trace.getResponseCount())
	testHost(t, hosts[0], "01:02:03:04:05:05", "192.0.2.15")
	testHost(t, hosts[1], "01:02:03:04:05:06", "192.0.2.16")
	testHost(t, hosts[2], "01:02:03:04:05:07", "192.0.2.17")
	testHost(t, hosts[3], "01:02:03:04:05:08", "192.0.2.18")
	testHost(t, hosts[4], "01:02:03:04:05:09", "192.0.2.19")
	testReservationGetPageReceived(t, it)
	require.Zero(t, (fa.GetLastCommand().Arguments.(map[string]interface{}))["subnet-id"])
	require.Contains(t, (fa.GetLastCommand().Arguments.(map[string]interface{})), "from")
	require.Contains(t, (fa.GetLastCommand().Arguments.(map[string]interface{})), "source-index")

	// Should get addresses 192.0.3.10 thru 192.0.3.14
	hosts, done, err = it.getPageFromHostCmds()
	require.NoError(t, err)
	require.False(t, done)
	require.Len(t, hosts, 5)
	require.EqualValues(t, 5, it.from)
	require.EqualValues(t, 1, it.sourceIndex)
	require.Len(t, it.subnets, 4)
	require.Zero(t, it.subnetIndex)
	require.NotNil(t, it.getCurrentSubnet())
	require.Equal(t, "192.0.3.0/24", it.getCurrentSubnet().Prefix)
	require.EqualValues(t, 3, it.trace.getResponseCount())
	testHost(t, hosts[0], "01:02:03:04:05:15", "192.0.3.10")
	testHost(t, hosts[1], "01:02:03:04:05:16", "192.0.3.11")
	testHost(t, hosts[2], "01:02:03:04:05:17", "192.0.3.12")
	testHost(t, hosts[3], "01:02:03:04:05:18", "192.0.3.13")
	testHost(t, hosts[4], "01:02:03:04:05:19", "192.0.3.14")
	testReservationGetPageReceived(t, it)
	require.EqualValues(t, 234, (fa.GetLastCommand().Arguments.(map[string]interface{}))["subnet-id"])
	require.NotContains(t, fa.GetLastCommand().Arguments.(map[string]interface{}), "from")
	require.Contains(t, fa.GetLastCommand().Arguments.(map[string]interface{}), "source-index")

	// Should get addresses 192.0.3.15 thru 192.0.3.19
	hosts, done, err = it.getPageFromHostCmds()
	require.NoError(t, err)
	require.False(t, done)
	require.Len(t, hosts, 5)
	require.EqualValues(t, 10, it.from)
	require.EqualValues(t, 1, it.sourceIndex)
	require.Len(t, it.subnets, 4)
	require.Zero(t, it.subnetIndex)
	require.NotNil(t, it.getCurrentSubnet())
	require.Equal(t, "192.0.3.0/24", it.getCurrentSubnet().Prefix)
	require.EqualValues(t, 4, it.trace.getResponseCount())
	testHost(t, hosts[0], "01:02:03:04:05:20", "192.0.3.15")
	testHost(t, hosts[1], "01:02:03:04:05:21", "192.0.3.16")
	testHost(t, hosts[2], "01:02:03:04:05:22", "192.0.3.17")
	testHost(t, hosts[3], "01:02:03:04:05:23", "192.0.3.18")
	testHost(t, hosts[4], "01:02:03:04:05:24", "192.0.3.19")
	testReservationGetPageReceived(t, it)
	require.EqualValues(t, 234, (fa.GetLastCommand().Arguments.(map[string]interface{}))["subnet-id"])
	require.Contains(t, fa.GetLastCommand().Arguments.(map[string]interface{}), "from")
	require.Contains(t, fa.GetLastCommand().Arguments.(map[string]interface{}), "source-index")

	// Should get addresses 192.0.4.10 thru 192.0.4.14
	hosts, done, err = it.getPageFromHostCmds()
	require.NoError(t, err)
	require.False(t, done)
	require.Len(t, hosts, 5)
	require.EqualValues(t, 5, it.from)
	require.EqualValues(t, 1, it.sourceIndex)
	require.Len(t, it.subnets, 4)
	require.EqualValues(t, 1, it.subnetIndex)
	require.NotNil(t, it.getCurrentSubnet())
	require.Equal(t, "192.0.4.0/24", it.getCurrentSubnet().Prefix)
	require.EqualValues(t, 5, it.trace.getResponseCount())
	testHost(t, hosts[0], "01:02:03:04:05:30", "192.0.4.10")
	testHost(t, hosts[1], "01:02:03:04:05:31", "192.0.4.11")
	testHost(t, hosts[2], "01:02:03:04:05:32", "192.0.4.12")
	testHost(t, hosts[3], "01:02:03:04:05:33", "192.0.4.13")
	testHost(t, hosts[4], "01:02:03:04:05:34", "192.0.4.14")
	testReservationGetPageReceived(t, it)
	require.EqualValues(t, 345, (fa.GetLastCommand().Arguments.(map[string]interface{}))["subnet-id"])
	require.NotContains(t, fa.GetLastCommand().Arguments.(map[string]interface{}), "from")
	require.Contains(t, fa.GetLastCommand().Arguments.(map[string]interface{}), "source-index")

	// Should get addresses 192.0.4.15 thru 192.0.4.19
	hosts, done, err = it.getPageFromHostCmds()
	require.NoError(t, err)
	require.False(t, done)
	require.Len(t, hosts, 5)
	require.EqualValues(t, 10, it.from)
	require.EqualValues(t, 1, it.sourceIndex)
	require.Len(t, it.subnets, 4)
	require.EqualValues(t, 1, it.subnetIndex)
	require.NotNil(t, it.getCurrentSubnet())
	require.Equal(t, "192.0.4.0/24", it.getCurrentSubnet().Prefix)
	require.EqualValues(t, 6, it.trace.getResponseCount())
	testHost(t, hosts[0], "01:02:03:04:05:35", "192.0.4.15")
	testHost(t, hosts[1], "01:02:03:04:05:36", "192.0.4.16")
	testHost(t, hosts[2], "01:02:03:04:05:37", "192.0.4.17")
	testHost(t, hosts[3], "01:02:03:04:05:38", "192.0.4.18")
	testHost(t, hosts[4], "01:02:03:04:05:39", "192.0.4.19")
	testReservationGetPageReceived(t, it)
	require.EqualValues(t, 345, (fa.GetLastCommand().Arguments.(map[string]interface{}))["subnet-id"])
	require.Contains(t, fa.GetLastCommand().Arguments.(map[string]interface{}), "from")
	require.Contains(t, fa.GetLastCommand().Arguments.(map[string]interface{}), "source-index")

	// Should get addresses 192.0.5.10 thru 192.0.5.14
	hosts, done, err = it.getPageFromHostCmds()
	require.NoError(t, err)
	require.False(t, done)
	require.Len(t, hosts, 5)
	require.EqualValues(t, 5, it.from)
	require.EqualValues(t, 2, it.sourceIndex)
	require.Len(t, it.subnets, 4)
	require.EqualValues(t, 2, it.subnetIndex)
	require.NotNil(t, it.getCurrentSubnet())
	require.Equal(t, "192.0.5.0/24", it.getCurrentSubnet().Prefix)
	require.EqualValues(t, 7, it.trace.getResponseCount())
	testHost(t, hosts[0], "01:02:03:04:05:45", "192.0.5.10")
	testHost(t, hosts[1], "01:02:03:04:05:46", "192.0.5.11")
	testHost(t, hosts[2], "01:02:03:04:05:47", "192.0.5.12")
	testHost(t, hosts[3], "01:02:03:04:05:48", "192.0.5.13")
	testHost(t, hosts[4], "01:02:03:04:05:49", "192.0.5.14")
	testReservationGetPageReceived(t, it)
	require.EqualValues(t, 567, (fa.GetLastCommand().Arguments.(map[string]interface{}))["subnet-id"])
	require.NotContains(t, fa.GetLastCommand().Arguments.(map[string]interface{}), "from")
	require.Contains(t, fa.GetLastCommand().Arguments.(map[string]interface{}), "source-index")

	// Should get addresses 192.0.5.15 thru 192.0.5.19
	hosts, done, err = it.getPageFromHostCmds()
	require.NoError(t, err)
	require.False(t, done)
	require.Len(t, hosts, 5)
	require.EqualValues(t, 10, it.from)
	require.EqualValues(t, 2, it.sourceIndex)
	require.Len(t, it.subnets, 4)
	require.EqualValues(t, 2, it.subnetIndex)
	require.NotNil(t, it.getCurrentSubnet())
	require.Equal(t, "192.0.5.0/24", it.getCurrentSubnet().Prefix)
	require.EqualValues(t, 8, it.trace.getResponseCount())
	testHost(t, hosts[0], "01:02:03:04:05:50", "192.0.5.15")
	testHost(t, hosts[1], "01:02:03:04:05:51", "192.0.5.16")
	testHost(t, hosts[2], "01:02:03:04:05:52", "192.0.5.17")
	testHost(t, hosts[3], "01:02:03:04:05:53", "192.0.5.18")
	testHost(t, hosts[4], "01:02:03:04:05:54", "192.0.5.19")
	testReservationGetPageReceived(t, it)
	require.EqualValues(t, 567, (fa.GetLastCommand().Arguments.(map[string]interface{}))["subnet-id"])
	require.Contains(t, fa.GetLastCommand().Arguments.(map[string]interface{}), "from")
	require.Contains(t, fa.GetLastCommand().Arguments.(map[string]interface{}), "source-index")

	// Should get addresses 192.0.6.10 thru 192.0.6.14
	hosts, done, err = it.getPageFromHostCmds()
	require.NoError(t, err)
	require.False(t, done)
	require.Len(t, hosts, 5)
	require.EqualValues(t, 5, it.from)
	require.EqualValues(t, 2, it.sourceIndex)
	require.Len(t, it.subnets, 4)
	require.EqualValues(t, 3, it.subnetIndex)
	require.NotNil(t, it.getCurrentSubnet())
	require.Equal(t, "192.0.6.0/24", it.getCurrentSubnet().Prefix)
	require.EqualValues(t, 9, it.trace.getResponseCount())
	testHost(t, hosts[0], "01:02:03:04:05:60", "192.0.6.10")
	testHost(t, hosts[1], "01:02:03:04:05:61", "192.0.6.11")
	testHost(t, hosts[2], "01:02:03:04:05:62", "192.0.6.12")
	testHost(t, hosts[3], "01:02:03:04:05:63", "192.0.6.13")
	testHost(t, hosts[4], "01:02:03:04:05:64", "192.0.6.14")
	testReservationGetPageReceived(t, it)
	require.EqualValues(t, 678, (fa.GetLastCommand().Arguments.(map[string]interface{}))["subnet-id"])
	require.NotContains(t, fa.GetLastCommand().Arguments.(map[string]interface{}), "from")
	require.Contains(t, fa.GetLastCommand().Arguments.(map[string]interface{}), "source-index")

	// Should get addresses 192.0.6.15 thru 192.0.6.19
	hosts, done, err = it.getPageFromHostCmds()
	require.NoError(t, err)
	require.False(t, done)
	require.Len(t, hosts, 5)
	require.EqualValues(t, 10, it.from)
	require.EqualValues(t, 2, it.sourceIndex)
	require.Len(t, it.subnets, 4)
	require.EqualValues(t, 3, it.subnetIndex)
	require.NotNil(t, it.getCurrentSubnet())
	require.Equal(t, "192.0.6.0/24", it.getCurrentSubnet().Prefix)
	require.EqualValues(t, 10, it.trace.getResponseCount())
	testHost(t, hosts[0], "01:02:03:04:05:65", "192.0.6.15")
	testHost(t, hosts[1], "01:02:03:04:05:66", "192.0.6.16")
	testHost(t, hosts[2], "01:02:03:04:05:67", "192.0.6.17")
	testHost(t, hosts[3], "01:02:03:04:05:68", "192.0.6.18")
	testHost(t, hosts[4], "01:02:03:04:05:69", "192.0.6.19")
	testReservationGetPageReceived(t, it)
	require.EqualValues(t, 678, (fa.GetLastCommand().Arguments.(map[string]interface{}))["subnet-id"])
	require.Contains(t, fa.GetLastCommand().Arguments.(map[string]interface{}), "from")
	require.Contains(t, fa.GetLastCommand().Arguments.(map[string]interface{}), "source-index")

	// We have iterated over all subnets already and fetched all
	// reservations. No hosts should be returned, the done flag
	// should indicate that we have reached the end of hosts.
	hosts, done, err = it.getPageFromHostCmds()
	require.NoError(t, err)
	require.True(t, done)
	require.Empty(t, hosts)
	require.EqualValues(t, 10, it.trace.getResponseCount())
	testReservationGetPageReceived(t, it)
	require.EqualValues(t, 678, (fa.GetLastCommand().Arguments.(map[string]interface{}))["subnet-id"])

	it = newHostIterator(db, &app, app.Daemons[1], fa, 5)
	require.NotNil(t, it)

	// Should get addresses 2001:db8:2::10 thru 2001:db8:2::14
	hosts, done, err = it.getPageFromHostCmds()
	require.NoError(t, err)
	require.False(t, done)
	require.Len(t, hosts, 5)
	require.EqualValues(t, 5, it.from)
	require.EqualValues(t, 1, it.sourceIndex)
	require.Len(t, it.subnets, 4)
	require.EqualValues(t, -1, it.subnetIndex)
	require.Nil(t, it.getCurrentSubnet())
	require.EqualValues(t, 1, it.trace.getResponseCount())
	testHost(t, hosts[0], "01:02:03:04:05:00", "2001:db8:2::10")
	testHost(t, hosts[1], "01:02:03:04:05:01", "2001:db8:2::11")
	testHost(t, hosts[2], "01:02:03:04:05:02", "2001:db8:2::12")
	testHost(t, hosts[3], "01:02:03:04:05:03", "2001:db8:2::13")
	testHost(t, hosts[4], "01:02:03:04:05:04", "2001:db8:2::14")
	testReservationGetPageReceived(t, it)
	require.Zero(t, (fa.GetLastCommand().Arguments.(map[string]interface{}))["subnet-id"])
	require.NotContains(t, fa.GetLastCommand().Arguments.(map[string]interface{}), "from")
	require.Contains(t, fa.GetLastCommand().Arguments.(map[string]interface{}), "source-index")

	// Should get addresses 2001:db8:2::15 thru 2001:db8:2::19
	hosts, done, err = it.getPageFromHostCmds()
	require.False(t, done)
	require.NoError(t, err)
	require.Len(t, hosts, 5)
	require.EqualValues(t, 10, it.from)
	require.EqualValues(t, 1, it.sourceIndex)
	require.Len(t, it.subnets, 4)
	require.EqualValues(t, -1, it.subnetIndex)
	require.Nil(t, it.getCurrentSubnet())
	require.EqualValues(t, 2, it.trace.getResponseCount())
	testHost(t, hosts[0], "01:02:03:04:05:05", "2001:db8:2::15")
	testHost(t, hosts[1], "01:02:03:04:05:06", "2001:db8:2::16")
	testHost(t, hosts[2], "01:02:03:04:05:07", "2001:db8:2::17")
	testHost(t, hosts[3], "01:02:03:04:05:08", "2001:db8:2::18")
	testHost(t, hosts[4], "01:02:03:04:05:09", "2001:db8:2::19")
	testReservationGetPageReceived(t, it)
	require.Zero(t, (fa.GetLastCommand().Arguments.(map[string]interface{}))["subnet-id"])
	require.Contains(t, fa.GetLastCommand().Arguments.(map[string]interface{}), "from")
	require.Contains(t, fa.GetLastCommand().Arguments.(map[string]interface{}), "source-index")

	// Should get addresses 2001:db8:3::10 thru 2001:db8:3::14
	hosts, done, err = it.getPageFromHostCmds()
	require.NoError(t, err)
	require.False(t, done)
	require.Len(t, hosts, 5)
	require.EqualValues(t, 5, it.from)
	require.EqualValues(t, 1, it.sourceIndex)
	require.Len(t, it.subnets, 4)
	require.Zero(t, it.subnetIndex)
	require.NotNil(t, it.getCurrentSubnet())
	require.Equal(t, "2001:db8:3::/64", it.getCurrentSubnet().Prefix)
	require.EqualValues(t, 3, it.trace.getResponseCount())
	testHost(t, hosts[0], "01:02:03:04:05:15", "2001:db8:3::10")
	testHost(t, hosts[1], "01:02:03:04:05:16", "2001:db8:3::11")
	testHost(t, hosts[2], "01:02:03:04:05:17", "2001:db8:3::12")
	testHost(t, hosts[3], "01:02:03:04:05:18", "2001:db8:3::13")
	testHost(t, hosts[4], "01:02:03:04:05:19", "2001:db8:3::14")
	testReservationGetPageReceived(t, it)
	require.EqualValues(t, 234, (fa.GetLastCommand().Arguments.(map[string]interface{}))["subnet-id"])
	require.NotContains(t, fa.GetLastCommand().Arguments.(map[string]interface{}), "from")
	require.Contains(t, fa.GetLastCommand().Arguments.(map[string]interface{}), "source-index")

	// Should get addresses 2001:db8:3::15 thru 2001:db8:3::19
	hosts, done, err = it.getPageFromHostCmds()
	require.NoError(t, err)
	require.False(t, done)
	require.Len(t, hosts, 5)
	require.EqualValues(t, 10, it.from)
	require.EqualValues(t, 1, it.sourceIndex)
	require.Len(t, it.subnets, 4)
	require.Zero(t, it.subnetIndex)
	require.NotNil(t, it.getCurrentSubnet())
	require.Equal(t, "2001:db8:3::/64", it.getCurrentSubnet().Prefix)
	require.EqualValues(t, 4, it.trace.getResponseCount())
	testHost(t, hosts[0], "01:02:03:04:05:20", "2001:db8:3::15")
	testHost(t, hosts[1], "01:02:03:04:05:21", "2001:db8:3::16")
	testHost(t, hosts[2], "01:02:03:04:05:22", "2001:db8:3::17")
	testHost(t, hosts[3], "01:02:03:04:05:23", "2001:db8:3::18")
	testHost(t, hosts[4], "01:02:03:04:05:24", "2001:db8:3::19")
	testReservationGetPageReceived(t, it)
	require.EqualValues(t, 234, (fa.GetLastCommand().Arguments.(map[string]interface{}))["subnet-id"])
	require.Contains(t, fa.GetLastCommand().Arguments.(map[string]interface{}), "from")
	require.Contains(t, fa.GetLastCommand().Arguments.(map[string]interface{}), "source-index")

	// Should get addresses 2001:db8:4::10 thru 2001:db8:4::14
	hosts, done, err = it.getPageFromHostCmds()
	require.NoError(t, err)
	require.False(t, done)
	require.Len(t, hosts, 5)
	require.EqualValues(t, 5, it.from)
	require.EqualValues(t, 1, it.sourceIndex)
	require.Len(t, it.subnets, 4)
	require.EqualValues(t, 1, it.subnetIndex)
	require.NotNil(t, it.getCurrentSubnet())
	require.Equal(t, "2001:db8:4::/64", it.getCurrentSubnet().Prefix)
	require.EqualValues(t, 5, it.trace.getResponseCount())
	testHost(t, hosts[0], "01:02:03:04:05:30", "2001:db8:4::10")
	testHost(t, hosts[1], "01:02:03:04:05:31", "2001:db8:4::11")
	testHost(t, hosts[2], "01:02:03:04:05:32", "2001:db8:4::12")
	testHost(t, hosts[3], "01:02:03:04:05:33", "2001:db8:4::13")
	testHost(t, hosts[4], "01:02:03:04:05:34", "2001:db8:4::14")
	testReservationGetPageReceived(t, it)
	require.EqualValues(t, 345, (fa.GetLastCommand().Arguments.(map[string]interface{}))["subnet-id"])
	require.NotContains(t, fa.GetLastCommand().Arguments.(map[string]interface{}), "from")
	require.Contains(t, fa.GetLastCommand().Arguments.(map[string]interface{}), "source-index")

	// Should get addresses 2001:db8:4::15 thru 2001:db8:4::19
	hosts, done, err = it.getPageFromHostCmds()
	require.NoError(t, err)
	require.False(t, done)
	require.Len(t, hosts, 5)
	require.EqualValues(t, 10, it.from)
	require.EqualValues(t, 1, it.sourceIndex)
	require.Len(t, it.subnets, 4)
	require.EqualValues(t, 1, it.subnetIndex)
	require.NotNil(t, it.getCurrentSubnet())
	require.Equal(t, "2001:db8:4::/64", it.getCurrentSubnet().Prefix)
	require.EqualValues(t, 6, it.trace.getResponseCount())
	testHost(t, hosts[0], "01:02:03:04:05:35", "2001:db8:4::15")
	testHost(t, hosts[1], "01:02:03:04:05:36", "2001:db8:4::16")
	testHost(t, hosts[2], "01:02:03:04:05:37", "2001:db8:4::17")
	testHost(t, hosts[3], "01:02:03:04:05:38", "2001:db8:4::18")
	testHost(t, hosts[4], "01:02:03:04:05:39", "2001:db8:4::19")
	testReservationGetPageReceived(t, it)
	require.EqualValues(t, 345, (fa.GetLastCommand().Arguments.(map[string]interface{}))["subnet-id"])
	require.Contains(t, fa.GetLastCommand().Arguments.(map[string]interface{}), "from")
	require.Contains(t, fa.GetLastCommand().Arguments.(map[string]interface{}), "source-index")

	// Should get addresses 2001:db8:5::10 thru 2001:db8:5::14
	hosts, done, err = it.getPageFromHostCmds()
	require.NoError(t, err)
	require.False(t, done)
	require.Len(t, hosts, 5)
	require.EqualValues(t, 5, it.from)
	require.EqualValues(t, 2, it.sourceIndex)
	require.Len(t, it.subnets, 4)
	require.EqualValues(t, 2, it.subnetIndex)
	require.NotNil(t, it.getCurrentSubnet())
	require.Equal(t, "2001:db8:5::/64", it.getCurrentSubnet().Prefix)
	require.EqualValues(t, 7, it.trace.getResponseCount())
	testHost(t, hosts[0], "01:02:03:04:05:45", "2001:db8:5::10")
	testHost(t, hosts[1], "01:02:03:04:05:46", "2001:db8:5::11")
	testHost(t, hosts[2], "01:02:03:04:05:47", "2001:db8:5::12")
	testHost(t, hosts[3], "01:02:03:04:05:48", "2001:db8:5::13")
	testHost(t, hosts[4], "01:02:03:04:05:49", "2001:db8:5::14")
	testReservationGetPageReceived(t, it)
	require.EqualValues(t, 567, (fa.GetLastCommand().Arguments.(map[string]interface{}))["subnet-id"])
	require.NotContains(t, fa.GetLastCommand().Arguments.(map[string]interface{}), "from")
	require.Contains(t, fa.GetLastCommand().Arguments.(map[string]interface{}), "source-index")

	// Should get addresses 2001:db8:5::15 thru 2001:db8:5::19
	hosts, done, err = it.getPageFromHostCmds()
	require.NoError(t, err)
	require.False(t, done)
	require.Len(t, hosts, 5)
	require.EqualValues(t, 10, it.from)
	require.EqualValues(t, 2, it.sourceIndex)
	require.Len(t, it.subnets, 4)
	require.EqualValues(t, 2, it.subnetIndex)
	require.NotNil(t, it.getCurrentSubnet())
	require.Equal(t, "2001:db8:5::/64", it.getCurrentSubnet().Prefix)
	require.EqualValues(t, 8, it.trace.getResponseCount())
	testHost(t, hosts[0], "01:02:03:04:05:50", "2001:db8:5::15")
	testHost(t, hosts[1], "01:02:03:04:05:51", "2001:db8:5::16")
	testHost(t, hosts[2], "01:02:03:04:05:52", "2001:db8:5::17")
	testHost(t, hosts[3], "01:02:03:04:05:53", "2001:db8:5::18")
	testHost(t, hosts[4], "01:02:03:04:05:54", "2001:db8:5::19")
	testReservationGetPageReceived(t, it)
	require.EqualValues(t, 567, (fa.GetLastCommand().Arguments.(map[string]interface{}))["subnet-id"])
	require.Contains(t, fa.GetLastCommand().Arguments.(map[string]interface{}), "from")
	require.Contains(t, fa.GetLastCommand().Arguments.(map[string]interface{}), "source-index")

	// Should get addresses 2001:db8:6::10 thru 2001:db8:6::14
	hosts, done, err = it.getPageFromHostCmds()
	require.NoError(t, err)
	require.False(t, done)
	require.Len(t, hosts, 5)
	require.EqualValues(t, 5, it.from)
	require.EqualValues(t, 2, it.sourceIndex)
	require.Len(t, it.subnets, 4)
	require.EqualValues(t, 3, it.subnetIndex)
	require.NotNil(t, it.getCurrentSubnet())
	require.Equal(t, "2001:db8:6::/64", it.getCurrentSubnet().Prefix)
	require.EqualValues(t, 9, it.trace.getResponseCount())
	testHost(t, hosts[0], "01:02:03:04:05:60", "2001:db8:6::10")
	testHost(t, hosts[1], "01:02:03:04:05:61", "2001:db8:6::11")
	testHost(t, hosts[2], "01:02:03:04:05:62", "2001:db8:6::12")
	testHost(t, hosts[3], "01:02:03:04:05:63", "2001:db8:6::13")
	testHost(t, hosts[4], "01:02:03:04:05:64", "2001:db8:6::14")
	testReservationGetPageReceived(t, it)
	require.EqualValues(t, 678, (fa.GetLastCommand().Arguments.(map[string]interface{}))["subnet-id"])
	require.NotContains(t, fa.GetLastCommand().Arguments.(map[string]interface{}), "from")
	require.Contains(t, fa.GetLastCommand().Arguments.(map[string]interface{}), "source-index")

	// Should get addresses 2001:db8:6::15 thru 2001:db8:6::19
	hosts, done, err = it.getPageFromHostCmds()
	require.NoError(t, err)
	require.False(t, done)
	require.Len(t, hosts, 5)
	require.EqualValues(t, 10, it.from)
	require.EqualValues(t, 2, it.sourceIndex)
	require.Len(t, it.subnets, 4)
	require.EqualValues(t, 3, it.subnetIndex)
	require.NotNil(t, it.getCurrentSubnet())
	require.Equal(t, "2001:db8:6::/64", it.getCurrentSubnet().Prefix)
	require.EqualValues(t, 10, it.trace.getResponseCount())
	testHost(t, hosts[0], "01:02:03:04:05:65", "2001:db8:6::15")
	testHost(t, hosts[1], "01:02:03:04:05:66", "2001:db8:6::16")
	testHost(t, hosts[2], "01:02:03:04:05:67", "2001:db8:6::17")
	testHost(t, hosts[3], "01:02:03:04:05:68", "2001:db8:6::18")
	testHost(t, hosts[4], "01:02:03:04:05:69", "2001:db8:6::19")
	testReservationGetPageReceived(t, it)
	require.EqualValues(t, 678, (fa.GetLastCommand().Arguments.(map[string]interface{}))["subnet-id"])
	require.Contains(t, fa.GetLastCommand().Arguments.(map[string]interface{}), "from")
	require.Contains(t, fa.GetLastCommand().Arguments.(map[string]interface{}), "source-index")

	// We have iterated over all subnets already and fetched all
	// reservations. No hosts should be returned, the done flag
	// should indicate that we have reached the end of hosts.
	hosts, done, err = it.getPageFromHostCmds()
	require.NoError(t, err)
	require.True(t, done)
	require.Empty(t, hosts)
	require.EqualValues(t, 10, it.trace.getResponseCount())
	testReservationGetPageReceived(t, it)
	require.EqualValues(t, 678, (fa.GetLastCommand().Arguments.(map[string]interface{}))["subnet-id"])
}

// Test function which fetches host reservations from the Kea server over
// the control channel and stores them in the database.
func TestFetchHostsFromHostCmds(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	fec := &storktest.FakeEventCenter{}

	m := &dbmodel.Machine{
		ID:        0,
		Address:   "localhost",
		AgentPort: 8080,
	}
	err := dbmodel.AddMachine(db, m)
	require.NoError(t, err)

	// Creates new app with provided configurations.
	accessPoints := []*dbmodel.AccessPoint{}
	accessPoints = dbmodel.AppendAccessPoint(accessPoints, dbmodel.AccessPointControl, "localhost", "", 8000, false)
	app := dbmodel.App{
		MachineID:    m.ID,
		Type:         dbmodel.AppTypeKea,
		AccessPoints: accessPoints,
		Daemons: []*dbmodel.Daemon{
			{
				Name:   "dhcp4",
				Active: true,
				KeaDaemon: &dbmodel.KeaDaemon{
					KeaDHCPDaemon: &dbmodel.KeaDHCPDaemon{},
					Config:        getTestConfigWithIPv4Subnets(t, true),
				},
			},
			{
				Name:   "dhcp6",
				Active: true,
				KeaDaemon: &dbmodel.KeaDaemon{
					KeaDHCPDaemon: &dbmodel.KeaDHCPDaemon{},
					Config:        getTestConfigWithIPv6Subnets(t),
				},
			},
		},
	}
	// Add the app to the database.
	_, err = dbmodel.AddApp(db, &app)
	require.NoError(t, err)
	app.Machine = m

	lookup := dbmodel.NewDHCPOptionDefinitionLookup()
	err = CommitAppIntoDB(db, &app, fec, nil, lookup)
	require.NoError(t, err)

	fa := agentcommtest.NewFakeAgents(mockReservationGetPage, nil)
	fd := &storktest.FakeDispatcher{}

	// The puller requires fetch interval to be present in the database.
	err = dbmodel.InitializeSettings(db, 0)
	require.NoError(t, err)

	// Create the puller.
	puller, err := NewHostsPuller(db, fa, fd, lookup)
	require.NoError(t, err)
	require.NotNil(t, puller)

	// Detect hosts two times in the row. This simulates periodic
	// pull of the hosts for the given app.
	for i := 0; i < 2; i++ {
		err = puller.pull()
		require.NoError(t, err)

		hosts, err := dbmodel.GetAllHosts(db, 4)
		require.NoError(t, err)
		require.Len(t, hosts, 50)

		hosts, err = dbmodel.GetAllHosts(db, 6)
		require.NoError(t, err)
		require.Len(t, hosts, 50)

		// Ensure that the traces have been created.
		traces := puller.traces
		require.Contains(t, traces, app.Daemons[0].ID)
		require.EqualValues(t, 10, traces[app.Daemons[0].ID].getResponseCount())
		require.Contains(t, traces, app.Daemons[1].ID)
		require.EqualValues(t, 10, traces[app.Daemons[1].ID].getResponseCount())

		// Here, we test indirectly that hosts update was detected in the first
		// iteration and that it was not detected in the second iteration. The
		// config reviews are only scheduled when some hosts were updated.
		require.Len(t, fd.CallLog, 2)

		// Reset server state so it should send the same set of responses
		// the second time.
		fa.CallNo = 0
	}
}

// Test that new instance of the puller for fetching host reservations can be
// created and shut down.
func TestNewHostsPuller(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// The puller requires fetch interval to be present in the database.
	err := dbmodel.InitializeSettings(db, 0)
	require.NoError(t, err)

	fd := &storktest.FakeDispatcher{}
	lookup := dbmodel.NewDHCPOptionDefinitionLookup()
	puller, err := NewHostsPuller(db, nil, fd, lookup)
	require.NoError(t, err)
	require.NotNil(t, puller)
	puller.Shutdown()
}

// This test verifies that host reservations can be fetched via the hosts
// puller mechanism.
func TestPullHostsIntoDB(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	fec := &storktest.FakeEventCenter{}

	m := &dbmodel.Machine{
		ID:        0,
		Address:   "localhost",
		AgentPort: 8080,
	}
	err := dbmodel.AddMachine(db, m)
	require.NoError(t, err)

	// Creates new app with provided configurations.
	accessPoints := []*dbmodel.AccessPoint{}
	accessPoints = dbmodel.AppendAccessPoint(accessPoints, dbmodel.AccessPointControl, "localhost", "", 8000, true)
	app := dbmodel.App{
		MachineID:    m.ID,
		Type:         dbmodel.AppTypeKea,
		AccessPoints: accessPoints,
		Daemons: []*dbmodel.Daemon{
			{
				Name:   "dhcp4",
				Active: true,
				KeaDaemon: &dbmodel.KeaDaemon{
					KeaDHCPDaemon: &dbmodel.KeaDHCPDaemon{},
					Config:        getTestConfigWithIPv4Subnets(t, true),
				},
			},
			{
				Name:   "dhcp6",
				Active: true,
				KeaDaemon: &dbmodel.KeaDaemon{
					KeaDHCPDaemon: &dbmodel.KeaDHCPDaemon{},
					Config:        getTestConfigWithIPv6Subnets(t),
				},
			},
		},
	}
	// Add the app to the database.
	_, err = dbmodel.AddApp(db, &app)
	require.NoError(t, err)
	app.Machine = m

	lookup := dbmodel.NewDHCPOptionDefinitionLookup()
	err = CommitAppIntoDB(db, &app, fec, nil, lookup)
	require.NoError(t, err)

	fa := agentcommtest.NewFakeAgents(mockReservationGetPage, nil)

	// The puller requires fetch interval to be present in the database.
	err = dbmodel.InitializeSettings(db, 0)
	require.NoError(t, err)

	// Create the puller. It is configured to fetch the data every 60 seconds
	// so we'd rather call it periodically.
	fd := &storktest.FakeDispatcher{}
	puller, err := NewHostsPuller(db, fa, fd, lookup)
	require.NoError(t, err)
	require.NotNil(t, puller)

	// Detect hosts two times in the row. This simulates periodic
	// pull of the hosts for the given app.
	for i := 0; i < 2; i++ {
		err = puller.pull()
		require.NoError(t, err)

		hosts, err := dbmodel.GetAllHosts(db, 4)
		require.NoError(t, err)
		require.Len(t, hosts, 50)

		hosts, err = dbmodel.GetAllHosts(db, 6)
		require.NoError(t, err)
		require.Len(t, hosts, 50)

		// Reset server state so it should send the same set of responses
		// the second time.
		fa.CallNo = 0
	}

	// Ensure that the config review was scheduled after updating the hosts.
	require.Len(t, fd.CallLog, 2)
	for i, call := range fd.CallLog {
		require.Equal(t, "BeginReview", call.CallName)
		require.Equal(t, app.Daemons[i%2].ID, call.DaemonID)
		require.Equal(t, configreview.Triggers{configreview.DBHostsModified}, call.Triggers)
	}
}

// Test that hosts not returned in the subsequent attempts to fetch hosts
// from host_cmds hooks library are removed from the database.
func TestReduceHostsIntoDB(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	fec := &storktest.FakeEventCenter{}

	m := &dbmodel.Machine{
		ID:        0,
		Address:   "localhost",
		AgentPort: 8080,
	}
	err := dbmodel.AddMachine(db, m)
	require.NoError(t, err)

	// Creates new app with provided configurations.
	accessPoints := []*dbmodel.AccessPoint{}
	accessPoints = dbmodel.AppendAccessPoint(accessPoints, dbmodel.AccessPointControl, "localhost", "", 8000, false)
	app := dbmodel.App{
		MachineID:    m.ID,
		Type:         dbmodel.AppTypeKea,
		AccessPoints: accessPoints,
		Daemons: []*dbmodel.Daemon{
			{
				Name:   "dhcp4",
				Active: true,
				KeaDaemon: &dbmodel.KeaDaemon{
					KeaDHCPDaemon: &dbmodel.KeaDHCPDaemon{},
					Config:        getTestConfigWithOneIPv4Subnet(t),
				},
			},
		},
	}
	// Add the app to the database.
	_, err = dbmodel.AddApp(db, &app)
	require.NoError(t, err)
	app.Machine = m

	lookup := dbmodel.NewDHCPOptionDefinitionLookup()
	err = CommitAppIntoDB(db, &app, fec, nil, lookup)
	require.NoError(t, err)

	// Create server which returns two hosts at the first attempt and
	// one host at the second attempt.
	fa := agentcommtest.NewFakeAgents(mockReservationGetPageReduceHosts, nil)

	// The puller requires fetch interval to be present in the database.
	err = dbmodel.InitializeSettings(db, 0)
	require.NoError(t, err)

	// Create the puller instance.
	fd := &storktest.FakeDispatcher{}
	puller, err := NewHostsPuller(db, fa, fd, lookup)
	require.NoError(t, err)
	require.NotNil(t, puller)

	// Get the hosts from Kea. This should result in having two hosts
	// within the database.
	err = puller.pull()
	require.NoError(t, err)

	hosts, err := dbmodel.GetAllHosts(db, 4)
	require.NoError(t, err)
	require.Len(t, hosts, 2)

	// Repeat the same test, but this time only one host should be returned.
	puller, err = NewHostsPuller(db, fa, fd, lookup)
	require.NoError(t, err)
	require.NotNil(t, puller)

	err = puller.pull()
	require.NoError(t, err)

	// The second host should have been removed from the database.
	hosts, err = dbmodel.GetAllHosts(db, 4)
	require.NoError(t, err)
	require.Len(t, hosts, 1)
}

// Test the case when the puller first fetches the reservations in two
// chunks and then pulls again. During the second pull, the first page
// contains the same host as in the first pull. The second page contains
// a different host than the second page in the first pull. The server
// should ensure that the hosts are properly updated even though they
// are not updated when the first page is received during the second
// run.
func TestPartialHostsChange(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	m := &dbmodel.Machine{
		ID:        0,
		Address:   "localhost",
		AgentPort: 8080,
	}
	err := dbmodel.AddMachine(db, m)
	require.NoError(t, err)

	// Creates new app with provided configurations.
	accessPoints := []*dbmodel.AccessPoint{}
	accessPoints = dbmodel.AppendAccessPoint(accessPoints, dbmodel.AccessPointControl, "localhost", "", 8000, false)
	app := dbmodel.App{
		MachineID:    m.ID,
		Type:         dbmodel.AppTypeKea,
		AccessPoints: accessPoints,
		Daemons: []*dbmodel.Daemon{
			{
				Name:   "dhcp4",
				Active: true,
				KeaDaemon: &dbmodel.KeaDaemon{
					KeaDHCPDaemon: &dbmodel.KeaDHCPDaemon{},
					Config:        getTestConfigWithOneIPv4Subnet(t),
				},
			},
		},
	}
	// Add the app to the database.
	_, err = dbmodel.AddApp(db, &app)
	require.NoError(t, err)
	app.Machine = m

	fec := &storktest.FakeEventCenter{}
	lookup := dbmodel.NewDHCPOptionDefinitionLookup()
	err = CommitAppIntoDB(db, &app, fec, nil, lookup)
	require.NoError(t, err)

	fa := agentcommtest.NewFakeAgents(mockReservationGetPagePartialChange, nil)

	// The puller requires fetch interval to be present in the database.
	err = dbmodel.InitializeSettings(db, 0)
	require.NoError(t, err)

	// Create the puller instance.
	fd := &storktest.FakeDispatcher{}
	puller, err := NewHostsPuller(db, fa, fd, lookup)
	require.NoError(t, err)
	require.NotNil(t, puller)

	// Get the hosts from Kea. There should be three returned in two chunks.
	err = puller.pull()
	require.NoError(t, err)

	hosts, err := dbmodel.GetAllHosts(db, 4)
	require.NoError(t, err)
	require.Len(t, hosts, 3)
	require.Len(t, hosts, 3)
	testHost(t, hosts[0], "01:02:03:04:05:06", "192.0.2.10/32")
	testHost(t, hosts[1], "01:02:03:04:05:07", "192.0.2.11/32")
	testHost(t, hosts[2], "01:02:03:04:05:08", "192.0.2.12/32")

	for _, host := range hosts {
		require.Len(t, host.LocalHosts, 1)
	}

	// Second iteration.
	err = puller.pull()
	require.NoError(t, err)

	// The third host should have been replaced with a new one.
	hosts, err = dbmodel.GetAllHosts(db, 4)
	require.NoError(t, err)
	require.Len(t, hosts, 3)
	testHost(t, hosts[0], "01:02:03:04:05:06", "192.0.2.10/32")
	testHost(t, hosts[1], "01:02:03:04:05:07", "192.0.2.11/32")
	testHost(t, hosts[2], "01:02:03:04:05:09", "192.0.2.13/32")

	for _, host := range hosts {
		require.Len(t, host.LocalHosts, 1)
	}
}

// This test verifies that host reservations are not fetched when the
// libdhcp_host_cmds hooks library is not loaded or when the daemon
// is inactive.
func TestSkipPullingHostsIntoDB(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	fec := &storktest.FakeEventCenter{}

	m := &dbmodel.Machine{
		ID:        0,
		Address:   "localhost",
		AgentPort: 8080,
	}
	err := dbmodel.AddMachine(db, m)
	require.NoError(t, err)

	// Create an app with DHCP configurations lacking the libdhcp_host_cmds
	// hooks library.
	accessPoints := []*dbmodel.AccessPoint{}
	accessPoints = dbmodel.AppendAccessPoint(accessPoints, dbmodel.AccessPointControl, "localhost", "", 8000, true)
	app := dbmodel.App{
		MachineID:    m.ID,
		Type:         dbmodel.AppTypeKea,
		AccessPoints: accessPoints,
		Daemons: []*dbmodel.Daemon{
			{
				Name:   "dhcp4",
				Active: true,
				KeaDaemon: &dbmodel.KeaDaemon{
					KeaDHCPDaemon: &dbmodel.KeaDHCPDaemon{},
					Config:        getTestConfigWithIPv4Subnets(t, false),
				},
			},
			{
				Name:   "dhcp6",
				Active: false,
				KeaDaemon: &dbmodel.KeaDaemon{
					KeaDHCPDaemon: &dbmodel.KeaDHCPDaemon{},
					Config:        getTestConfigWithIPv6Subnets(t),
				},
			},
		},
	}
	// Add the app to the database.
	_, err = dbmodel.AddApp(db, &app)
	require.NoError(t, err)
	app.Machine = m

	lookup := dbmodel.NewDHCPOptionDefinitionLookup()
	err = CommitAppIntoDB(db, &app, fec, nil, lookup)
	require.NoError(t, err)

	fa := agentcommtest.NewFakeAgents(mockReservationGetPage, nil)

	// The puller requires fetch interval to be present in the database.
	err = dbmodel.InitializeSettings(db, 0)
	require.NoError(t, err)

	fd := &storktest.FakeDispatcher{}
	puller, err := NewHostsPuller(db, fa, fd, lookup)
	require.NoError(t, err)
	require.NotNil(t, puller)

	err = puller.pull()
	require.NoError(t, err)

	// Make sure that the hosts were not added to the database.
	hosts, err := dbmodel.GetAllHosts(db, 4)
	require.NoError(t, err)
	require.Empty(t, hosts)

	hosts, err = dbmodel.GetAllHosts(db, 6)
	require.NoError(t, err)
	require.Empty(t, hosts)
}

// Test that host hash tracing works fine.
func TestHostIteratorTrace(t *testing.T) {
	// Create new traces.
	trace0 := newHostIteratorTrace()
	require.NotNil(t, trace0)
	trace1 := newHostIteratorTrace()
	require.NotNil(t, trace1)
	trace2 := newHostIteratorTrace()
	require.NotNil(t, trace1)

	hosts := []keaconfig.Reservation{}

	trace0.addResponse("1234", 0, hosts)
	require.EqualValues(t, 1, trace0.getResponseCount())
	trace0.addResponse("2345", 0, hosts)
	require.EqualValues(t, 2, trace0.getResponseCount())

	// trace1 has no hashes. Comparing them with trace1 hashes should
	// always pass.
	require.True(t, trace1.hasEqualHashes(trace0))
	// Comparing the other way around should fail because hashes from
	// trace0 are missing in trace1.
	require.False(t, trace0.hasEqualHashes(trace1))

	// Add a matching hash.
	trace1.addResponse("1234", 0, hosts)
	require.True(t, trace1.hasEqualHashes(trace0))
	// Add non matching hash.
	trace1.addResponse("5678", 0, hosts)
	require.False(t, trace1.hasEqualHashes(trace0))

	// First add matching hashes, followed by a non-matching one.
	trace2.addResponse("1234", 0, hosts)
	require.True(t, trace2.hasEqualHashes(trace0))
	trace2.addResponse("2345", 0, hosts)
	require.True(t, trace2.hasEqualHashes(trace0))
	trace2.addResponse("3456", 0, hosts)
	require.False(t, trace2.hasEqualHashes(trace0))
}

// Test that host reservation is updated when DHCP identifiers and IP
// addresses remain unchanged, but the hostname changes.
func TestUpdateHost(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	fec := &storktest.FakeEventCenter{}

	m := &dbmodel.Machine{
		ID:        0,
		Address:   "localhost",
		AgentPort: 8080,
	}
	err := dbmodel.AddMachine(db, m)
	require.NoError(t, err)

	accessPoints := []*dbmodel.AccessPoint{}
	accessPoints = dbmodel.AppendAccessPoint(accessPoints, dbmodel.AccessPointControl, "localhost", "", 8000, false)
	app := dbmodel.App{
		MachineID:    m.ID,
		Type:         dbmodel.AppTypeKea,
		AccessPoints: accessPoints,
		Daemons: []*dbmodel.Daemon{
			{
				Name:   "dhcp6",
				Active: true,
				KeaDaemon: &dbmodel.KeaDaemon{
					KeaDHCPDaemon: &dbmodel.KeaDHCPDaemon{},
					Config:        getTestConfigWithOneIPv6Subnet(t),
				},
			},
		},
	}

	_, err = dbmodel.AddApp(db, &app)
	require.NoError(t, err)
	app.Machine = m

	lookup := dbmodel.NewDHCPOptionDefinitionLookup()
	_ = CommitAppIntoDB(db, &app, fec, nil, lookup)

	fa := agentcommtest.NewFakeAgents(func(callNo int, cmdResponses []interface{}) {
		var json string
		// Update option data - read reservation option is unsupported yet.
		switch callNo {
		// Initial data
		case 1:
			json = `[
				{
					"result": 0,
					"text": "Hosts found",
					"arguments": {
						"count": 1,
						"hosts": [
							{
								"hw-address": "01:02:03:04:05:06",
								"ip-address": "fe80::1"
							}
						]
					}
				}
			]`
		// Update hostname
		case 4:
			json = `[
				{
					"result": 0,
					"text": "Hosts found",
					"arguments": {
						"count": 1,
						"hosts": [
							{
								"hw-address": "01:02:03:04:05:06",
								"ip-address": "fe80::1",
								"hostname": "foo.bar"
							}
						]
					}
				}
			]`
		// Break pulling
		case 0, 2, 3, 5:
			json = `[
				{
					"result": 0,
					"text": "Hosts found",
					"arguments": {
						"count": 0,
						"hosts": [ ],
						"next": {
							"from": 0,
							"source-index": 1
						}
					}
				}
			]`
		}

		command := keactrl.NewCommandBase(keactrl.ReservationGetPage, keactrl.DHCPv6)

		err = keactrl.UnmarshalResponseList(command, []byte(json), cmdResponses[0])
		require.NoError(t, err)
		fmt.Printf("cmdResponses[0]: %+v\n", cmdResponses[0])
	}, nil)

	// The puller requires fetch interval to be present in the database.
	err = dbmodel.InitializeSettings(db, 0)
	require.NoError(t, err)

	// Create the puller instance.
	fd := &storktest.FakeDispatcher{}
	puller, err := NewHostsPuller(db, fa, fd, lookup)
	require.NoError(t, err)
	require.NotNil(t, puller)

	for i := 0; i <= 1; i++ {
		err = puller.pull()
		require.NoError(t, err)

		hosts, err := dbmodel.GetAllHosts(db, 6)
		require.NoError(t, err)
		require.Len(t, hosts, 1)
		host := hosts[0]
		require.NotEmpty(t, host.LocalHosts)

		switch i {
		case 0:
			testHost(t, host, "01:02:03:04:05:06", "fe80::1/128")
		case 1:
			testHost(t, host, "01:02:03:04:05:06", "fe80::1/128")
			require.EqualValues(t, "foo.bar", host.LocalHosts[0].Hostname)
		}
	}
}
