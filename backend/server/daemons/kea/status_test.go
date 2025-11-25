package kea

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"isc.org/stork/datamodel/daemonname"
	"isc.org/stork/datamodel/protocoltype"
	agentcommtest "isc.org/stork/server/agentcomm/test"
	dbmodel "isc.org/stork/server/database/model"
	dbtest "isc.org/stork/server/database/test"
	storktest "isc.org/stork/server/test/dbmodel"
)

// Returns DHCP server configuration created from a template. The template
// parameters include root parameter, i.e. Dhcp4 or Dhcp6, High Availability
// mode and a variadic list of HA peers. The peers are identified by names:
// server1, server2 ...  server5. The server1 is a primary, the server2
// is a secondary, the server3 is a standby and the remaining ones are the
// backup servers.
func getHATestConfig(rootName, thisServerName, mode string, peerNames ...string) *dbmodel.KeaConfig {
	type peerInfo struct {
		URL  string
		Role string
	}
	// Map server names to peer configurations.
	peers := map[string]peerInfo{
		"server1": {
			URL:  "http://192.0.2.33:8000",
			Role: "primary",
		},
		"server2": {
			URL:  "http://192.0.2.66:8000",
			Role: "secondary",
		},
		"server3": {
			URL:  "http://192.0.2.66:8000",
			Role: "standby",
		},
		"server4": {
			URL:  "http://192.0.2.133:8000",
			Role: "backup",
		},
		"server5": {
			URL:  "http://192.0.2.166:8000",
			Role: "backup",
		},
	}

	// Output configuration of the peers from the template.
	var peersList string
	for _, peerName := range peerNames {
		if peer, ok := peers[peerName]; ok {
			peerTemplate := `
                {
                    "name": "%s",
                    "url":  "%s",
                    "role": "%s"
                }`
			peerTemplate = fmt.Sprintf(peerTemplate, peerName, peer.URL, peer.Role)
			if len(peersList) > 0 {
				peersList += ",\n"
			}
			peersList += peerTemplate
		}
	}

	// Output the server configuration from the template.
	configStr := `{
        "%s": {
            "hooks-libraries": [
                {
                    "library": "libdhcp_ha.so",
                    "parameters": {
                        "high-availability": [{
                            "this-server-name": "%s",
                            "mode": "%s",
                            "peers": [ %s ]
                        }]
                    }
                }
            ]
        }
    }`
	configStr = fmt.Sprintf(configStr, rootName, thisServerName, mode, peersList)

	// Convert the configuration from JSON to KeaConfig.
	var config dbmodel.KeaConfig
	_ = json.Unmarshal([]byte(configStr), &config)
	return &config
}

// Generates a response to the status-get command including two status
// structures, one for DHCPv4 and one for DHCPv6.
func mockGetStatusWithHA(callNo int, cmdResponses []interface{}) {
	var bytes string
	switch callNo {
	case 0:
		bytes = `{
            "result": 0,
            "text": "Everything is fine",
            "arguments": {
                "pid": 1234,
                "uptime": 3024,
                "reload": 1111,
                "ha-servers":
                    {
                        "local": {
                            "role": "primary",
                            "scopes": [ "server1" ],
                            "state": "load-balancing"
                        },
                        "remote": {
                            "age": 10,
                            "in-touch": true,
                            "role": "secondary",
                            "last-scopes": [ "server2" ],
                            "last-state": "load-balancing"
                        }
                    }
              }
         }`
	case 1:
		bytes = `{
             "result": 0,
             "text": "Everything is fine",
             "arguments": {
                 "pid": 2345,
                 "uptime": 3333,
                 "reload": 2222,
                 "ha-servers":
                     {
                         "local": {
                             "role": "standby",
                             "scopes": [ ],
                             "state": "hot-standby"
                         },
                         "remote": {
                             "age": 3,
                             "in-touch": true,
                             "role": "primary",
                             "last-scopes": [ "server1" ],
                             "last-state": "waiting"
                         }
                     }
               }
          }`
	case 2:
		bytes = `{
            "result": 0,
            "text": "Everything is fine",
            "arguments": {
                "pid": 1234,
                "uptime": 3024,
                "reload": 1111,
                "ha-servers":
                    {
                        "local": {
                            "role": "primary",
                            "scopes": [ "server1", "server2" ],
                            "state": "partner-down"
                        },
                        "remote": {
                            "age": 0,
                            "in-touch": false,
                            "role": "secondary",
                            "last-scopes": [ ],
                            "last-state": "unavailable"
                        }
                    }
              }
         }`
	case 3:
		bytes = `{
             "result": 1,
             "text": "Unable to communicate"
         }`
	case 4:
		bytes = `{
            "result": 0,
            "text": "Everything is fine",
            "arguments": {
                "pid": 1234,
                "uptime": 3024,
                "reload": 1111,
                "ha-servers":
                    {
                        "local": {
                            "role": "primary",
                            "scopes": [ "server1", "server2" ],
                            "state": "partner-down"
                        },
                        "remote": {
                            "age": 0,
                            "in-touch": false,
                            "role": "secondary",
                            "last-scopes": [ ],
                            "last-state": "unavailable"
                        }
                    }
              }
         }`
	case 5:
		bytes = `{
             "result": 1,
             "text": "Unable to communicate"
         }`
	}
	err := json.Unmarshal([]byte(bytes), cmdResponses[0])
	err = errors.WithStack(err)
	log.WithError(err).Error("unmarshal error")
}

// Generates a response to the status-get command including two status
// structures, one for DHCPv4 and one for DHCPv6. Format supported by
// Kea 1.7.8 onwards.
func mockGetStatusWithHA178(callNo int, cmdResponses []interface{}) {
	var bytes string
	switch callNo {
	case 0:
		bytes = `{
            "result": 0,
            "text": "Everything is fine",
            "arguments": {
                "pid": 1234,
                "uptime": 3024,
                "reload": 1111,
                "high-availability": [
                    {
                        "ha-mode": "load-balancing",
                        "ha-servers":
                            {
                                "local": {
                                "role": "primary",
                                "scopes": [ "server1" ],
                                "state": "load-balancing"
                            },
                            "remote": {
                                "age": 10,
                                "in-touch": true,
                                "role": "secondary",
                                "last-scopes": [ "server2" ],
                                "last-state": "load-balancing",
                                "communication-interrupted": true,
                                "connecting-clients": 1,
                                "unacked-clients": 2,
                                "unacked-clients-left": 3,
                                "analyzed-packets": 10
                            }
                        }
                    }
               ]
         }}`
	case 1:
		bytes = `{
             "result": 0,
             "text": "Everything is fine",
             "arguments": {
                 "pid": 2345,
                 "uptime": 3333,
                 "reload": 2222,
                 "high-availability": [
                     {
                         "ha-mode": "hot-standby",
                         "ha-servers":
                             {
                                 "local": {
                                     "role": "standby",
                                     "scopes": [ ],
                                     "state": "hot-standby"
                                 },
                                 "remote": {
                                     "age": 3,
                                     "in-touch": true,
                                     "role": "primary",
                                     "last-scopes": [ "server1" ],
                                     "last-state": "waiting",
                                     "communication-interrupted": true,
                                     "connecting-clients": 2,
                                     "unacked-clients": 3,
                                     "unacked-clients-left": 4,
                                     "analyzed-packets": 15
                                 }
                             }
                      }
                 ]
          }}`
	case 2:
		bytes = `{
            "result": 0,
            "text": "Everything is fine",
            "arguments": {
                "pid": 1234,
                "uptime": 3024,
                "reload": 1111,
                "high-availability": [
                    {
                        "ha-mode": "hot-standby",
                        "ha-servers":
                            {
                                "local": {
                                    "role": "primary",
                                    "scopes": [ "server1", "server2" ],
                                    "state": "partner-down"
                                },
                                "remote": {
                                    "age": 0,
                                    "in-touch": false,
                                    "role": "secondary",
                                    "last-scopes": [ ],
                                    "last-state": "unavailable",
                                    "communication-interrupted": true,
                                    "connecting-clients": 2,
                                    "unacked-clients": 3,
                                    "unacked-clients-left": 1,
                                    "analyzed-packets": 20
                                }
                           }
                    }
               ]
         }}`
	case 3:
		bytes = `{
             "result": 1,
             "text": "Unable to communicate"
         }`
	}
	err := json.Unmarshal([]byte(bytes), cmdResponses[0])
	err = errors.WithStack(err)
	log.WithError(err).Error("unmarshal error")
}

// Generates a response to the status-get command for a server that has two
// HA relationships.
func mockGetStatusWithHAHub(callNo int, cmdResponses []any) {
	var bytes string
	switch callNo {
	case 0:
		bytes = `{
            "result": 0,
            "text": "Everything is fine",
            "arguments": {
                "pid": 2222,
                "uptime": 1020,
                "reload": 70,
                "high-availability": [
                    {
                        "ha-mode": "hot-standby",
                        "ha-servers": {
                            "local": {
                                "server-name": "server2",
                                "role": "standby",
                                "scopes": [ ],
                                "state": "hot-standby"
                            },
                            "remote": {
                                "server-name": "server1",
                                "age": 10,
                                "in-touch": true,
                                "role": "primary",
                                "last-scopes": [ "server1" ],
                                "last-state": "hot-standby"
							}
                        }
                    },
                    {
                        "ha-mode": "hot-standby",
                        "ha-servers": {
                            "local": {
                                "server-name": "server4",
                                "role": "standby",
                                "scopes": [ ],
                                "state": "hot-standby"
                            },
                            "remote": {
                                "server-name": "server3",
                                "age": 10,
                                "in-touch": true,
                                "role": "standby",
                                "last-scopes": [ "server3" ],
                                "last-state": "hot-standby"
                            }
                        }
                    }
                ]
            }
        }`
	default:
		bytes = `{
            "result": 0,
            "text": "Everything is fine",
            "arguments": {
                "pid": 1234,
                "uptime": 3024,
                "reload": 1111,
                "high-availability": [
                    {
                        "ha-mode": "hot-standby",
                        "ha-servers": {
                            "local": {
                                "server-name": "server2",
                                "role": "standby",
                                "scopes": [ "server1" ],
                                "state": "partner-down"
                            },
                            "remote": {
                                "server-name": "server1",
                                "age": 10,
                                "in-touch": true,
                                "role": "secondary",
                                "last-scopes": [ ],
                                "last-state": "unavailable"
							}
                        }
                    },
                    {
                        "ha-mode": "hot-standby",
                        "ha-servers": {
                            "local": {
                                "server-name": "server4",
                                "role": "standby",
                                "scopes": [ "server3" ],
                                "state": "partner-down"
                            },
                            "remote": {
                                "server-name": "server3",
                                "age": 10,
                                "in-touch": true,
                                "role": "primary",
                                "last-scopes": [ ],
                                "last-state": "unavailable"
                            }
                        }
                    }
                ]
            }
        }`
	}
	err := json.Unmarshal([]byte(bytes), cmdResponses[0])
	err = errors.WithStack(err)
	log.WithError(err).Error("unmarshal error")
}

// Generate test response to status-get command including status of the
// HA pair doing load balancing.
func mockGetStatusLoadBalancing(callNo int, cmdResponses []interface{}) {
	bytes := `
        {
            "result": 0,
            "text": "Everything is fine",
            "arguments": {
                "pid": 1234,
                "uptime": 3024,
                "reload": 1111,
                "ha-servers":
                    {
                        "local": {
                            "role": "primary",
                            "scopes": [ "server1" ],
                            "state": "load-balancing"
                        },
                        "remote": {
                            "age": 10,
                            "in-touch": true,
                            "role": "secondary",
                            "last-scopes": [ "server2" ],
                            "last-state": "load-balancing"
                        }
                    }
                }
            }
    `
	err := json.Unmarshal([]byte(bytes), cmdResponses[0])
	err = errors.WithStack(err)
	log.WithError(err).Error("unmarshal error")
}

// Generate test response to status-get command including status of the
// HA pair doing load balancing. Format supported by Kea 1.7.8 onwards.
func mockGetStatusLoadBalancing178(callNo int, cmdResponses []interface{}) {
	bytes := `
        {
            "result": 0,
            "text": "Everything is fine",
            "arguments": {
                "pid": 1234,
                "uptime": 3024,
                "reload": 1111,
                "high-availability": [
                    {
                        "ha-mode": "load-balancing",
                        "ha-servers":
                            {
                                "local": {
                                    "role": "primary",
                                    "scopes": [ "server1" ],
                                    "state": "load-balancing"
                                },
                                "remote": {
                                    "age": 10,
                                    "in-touch": true,
                                    "role": "secondary",
                                    "last-scopes": [ "server2" ],
                                    "last-state": "load-balancing",
                                    "communication-interrupted": true,
                                    "connecting-clients": 1,
                                    "unacked-clients": 2,
                                    "unacked-clients-left": 3,
                                    "analyzed-packets": 10
                                }
                            }
                    }
                ]
            }
        }
    `
	err := json.Unmarshal([]byte(bytes), cmdResponses[0])
	err = errors.WithStack(err)
	log.WithError(err).Error("unmarshal error")
}

// Generates test response to status-get command lacking a status of the
// HA pair.
func mockGetStatusNoHA(callNo int, cmdResponses []interface{}) {
	bytes := `
        {
            "result": 0,
            "text": "Everything is fine",
            "arguments": {
                "pid": 1234,
                "uptime": 3024,
                "reload": 1111
            }
        }
    `
	err := json.Unmarshal([]byte(bytes), cmdResponses[0])
	err = errors.WithStack(err)
	log.WithError(err).Error("unmarshal error")
}

// Generates test response to status-get command indicating an error and
// lacking arguments.
func mockGetStatusError(callNo int, cmdResponses []interface{}) {
	bytes := `
        {
            "result": 1,
            "text": "unable to communicate with the daemon"
        }
    `
	err := json.Unmarshal([]byte(bytes), cmdResponses[0])
	err = errors.WithStack(err)
	log.WithError(err).Error("unmarshal error")
}

// Test status-get command when HA status is returned.
func TestGetDHCPStatus(t *testing.T) {
	fa := agentcommtest.NewFakeAgents(mockGetStatusLoadBalancing, nil)

	accessPoints := []*dbmodel.AccessPoint{
		{
			Type:     dbmodel.AccessPointControl,
			Address:  "",
			Port:     1234,
			Key:      "",
			Protocol: protocoltype.HTTP,
		},
	}

	machine := &dbmodel.Machine{
		Address:   "192.0.2.0",
		AgentPort: 1111,
	}

	daemon := dbmodel.NewDaemon(machine, daemonname.DHCPv4, true, accessPoints)

	status, err := getDHCPStatus(context.Background(), fa, daemon)
	require.NoError(t, err)
	require.NotNil(t, status)

	// Common fields must be always present.
	require.EqualValues(t, 1234, status.Pid)
	require.EqualValues(t, 3024, status.Uptime)
	require.EqualValues(t, 1111, status.Reload)

	// HA status should have been returned.
	require.NotNil(t, status.HAServers)

	// Test HA status of the server receiving the command.
	local := status.HAServers.Local
	require.Equal(t, "primary", local.Role)
	require.Len(t, local.Scopes, 1)
	require.Contains(t, local.Scopes, "server1")
	require.Equal(t, "load-balancing", local.State)

	// Test HA status of the partner.
	remote := status.HAServers.Remote
	require.Equal(t, "secondary", remote.Role)
	require.Len(t, remote.LastScopes, 1)
	require.Contains(t, remote.LastScopes, "server2")
	require.Equal(t, "load-balancing", remote.LastState)
	require.EqualValues(t, 10, remote.Age)
	require.True(t, remote.InTouch)
}

// Test status-get command when HA status is returned by Kea 1.7.8 or
// later.
func TestGetDHCPStatus178(t *testing.T) {
	fa := agentcommtest.NewFakeAgents(mockGetStatusLoadBalancing178, nil)

	accessPoints := []*dbmodel.AccessPoint{
		{
			Type:     dbmodel.AccessPointControl,
			Address:  "",
			Port:     1234,
			Key:      "",
			Protocol: protocoltype.HTTPS,
		},
	}

	machine := &dbmodel.Machine{
		Address:   "192.0.2.0",
		AgentPort: 1111,
	}

	daemon := dbmodel.NewDaemon(machine, daemonname.DHCPv4, true, accessPoints)

	status, err := getDHCPStatus(context.Background(), fa, daemon)
	require.NoError(t, err)
	require.NotNil(t, status)

	// Common fields must be always present.
	require.EqualValues(t, 1234, status.Pid)
	require.EqualValues(t, 3024, status.Uptime)
	require.EqualValues(t, 1111, status.Reload)

	// The HA status should be returned in the high-availability argument.
	require.Nil(t, status.HAServers)

	require.Len(t, status.HA, 1)

	require.Equal(t, "load-balancing", status.HA[0].HAMode)

	// Test HA status of the server receiving the command.
	local := status.HA[0].HAServers.Local
	require.Equal(t, "primary", local.Role)
	require.Len(t, local.Scopes, 1)
	require.Contains(t, local.Scopes, "server1")
	require.Equal(t, "load-balancing", local.State)

	// Test HA status of the partner.
	remote := status.HA[0].HAServers.Remote
	require.Equal(t, "secondary", remote.Role)
	require.Len(t, remote.LastScopes, 1)
	require.Contains(t, remote.LastScopes, "server2")
	require.Equal(t, "load-balancing", remote.LastState)
	require.EqualValues(t, 10, remote.Age)
	require.True(t, remote.InTouch)
	require.NotNil(t, remote.CommInterrupted)
	require.True(t, *remote.CommInterrupted)
	require.EqualValues(t, 1, remote.ConnectingClients)
	require.EqualValues(t, 2, remote.UnackedClients)
	require.EqualValues(t, 3, remote.UnackedClientsLeft)
	require.EqualValues(t, 10, remote.AnalyzedPackets)
}

// Test status-get command when HA status is not returned.
func TestGetDHCPStatusNoHA(t *testing.T) {
	fa := agentcommtest.NewFakeAgents(mockGetStatusNoHA, nil)

	accessPoints := []*dbmodel.AccessPoint{
		{
			Type:     dbmodel.AccessPointControl,
			Address:  "",
			Port:     1234,
			Key:      "",
			Protocol: protocoltype.HTTP,
		},
	}

	machine := &dbmodel.Machine{
		Address:   "192.0.2.0",
		AgentPort: 1111,
	}

	daemon := dbmodel.NewDaemon(machine, daemonname.DHCPv4, true, accessPoints)

	status, err := getDHCPStatus(context.Background(), fa, daemon)
	require.NoError(t, err)
	require.NotNil(t, status)

	// Common fields must be always present.
	require.EqualValues(t, 1234, status.Pid)
	require.EqualValues(t, 3024, status.Uptime)
	require.EqualValues(t, 1111, status.Reload)

	// This time, HA status should not be present.
	require.Nil(t, status.HAServers)
	require.Empty(t, status.HA)
}

// Test the case when the Kea CA is unable to communicate with the
// Kea daemon.
func TestGetDHCPStatusError(t *testing.T) {
	fa := agentcommtest.NewFakeAgents(mockGetStatusError, nil)

	accessPoints := []*dbmodel.AccessPoint{
		{
			Type:     dbmodel.AccessPointControl,
			Address:  "",
			Port:     1234,
			Key:      "",
			Protocol: protocoltype.HTTPS,
		},
	}

	machine := &dbmodel.Machine{
		Address:   "192.0.2.0",
		AgentPort: 1111,
	}

	daemon := dbmodel.NewDaemon(machine, daemonname.DHCPv4, true, accessPoints)

	status, err := getDHCPStatus(context.Background(), fa, daemon)
	require.ErrorContains(t, err, "unable to communicate with the daemon")
	require.Nil(t, status)
}

// Test that new instance of the puller for fetching HA services status can
// be created and shut down.
func TestNewHAStatusPuller(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// The puller requires fetch interval to be present in the database.
	err := dbmodel.InitializeSettings(db, 0)
	require.NoError(t, err)

	puller, err := NewHAStatusPuller(db, nil)
	require.NoError(t, err)
	require.NotNil(t, puller)
	defer puller.Shutdown()
}

// Test that HA status can be fetched and updated via the HA status puller
// mechanism. This is a generic test which can be used to validate the
// behavior for two different formats of the status-get response, one for
// Kea versions earlier than 1.7.8 and the second for Kea version 1.7.8
// and later.
func testPullHAStatus(t *testing.T, version178 bool) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Add a machine.
	m := &dbmodel.Machine{
		Address:   "localhost",
		AgentPort: 8080,
	}
	err := dbmodel.AddMachine(db, m)
	require.NoError(t, err)

	// Add Kea daemons to the machine
	accessPoints := []*dbmodel.AccessPoint{
		{
			Type:     dbmodel.AccessPointControl,
			Address:  "",
			Port:     1234,
			Key:      "",
			Protocol: protocoltype.HTTP,
		},
	}

	// Create DHCPv4 daemon
	daemon4 := dbmodel.NewDaemon(m, daemonname.DHCPv4, true, accessPoints)
	daemon4.KeaDaemon.Config = getHATestConfig("Dhcp4", "server1", "load-balancing",
		"server1", "server2", "server4")
	err = dbmodel.AddDaemon(db, daemon4)
	require.NoError(t, err)

	// Create DHCPv6 daemon
	daemon6 := dbmodel.NewDaemon(m, daemonname.DHCPv6, true, accessPoints)
	daemon6.KeaDaemon.Config = getHATestConfig("Dhcp6", "server3", "hot-standby",
		"server1", "server3", "server4")
	err = dbmodel.AddDaemon(db, daemon6)
	require.NoError(t, err)

	// Commit services for the daemons
	daemons := []*dbmodel.Daemon{daemon4, daemon6}
	for _, d := range daemons {
		// Detect services from the daemon configuration
		services, err := DetectHAServices(db, d)
		require.NoError(t, err)
		err = dbmodel.CommitServicesIntoDB(db, services, d)
		require.NoError(t, err)
	}

	// The puller requires fetch interval to be present in the database.
	err = dbmodel.InitializeSettings(db, 0)
	require.NoError(t, err)

	var fa *agentcommtest.FakeAgents

	// Configure the fake control agents to mimic returning a status of
	// two HA services for Kea.
	if version178 {
		fa = agentcommtest.NewFakeAgents(mockGetStatusWithHA178, nil)
	} else {
		fa = agentcommtest.NewFakeAgents(mockGetStatusWithHA, nil)
	}

	// Create the puller which normally fetches the HA status periodically.
	puller, err := NewHAStatusPuller(db, fa)
	require.NoError(t, err)
	require.NotNil(t, puller)

	// No need to wait for the puller to fetch the status.
	err = puller.pullData()
	require.NoError(t, err)

	// We should have two services in the database. One for DHCPv4 and one
	// for DHCPv6.
	services, err := dbmodel.GetDetailedAllServices(db)
	require.NoError(t, err)
	require.Len(t, services, 2)

	// The first one is the DHCPv4 service.
	service := services[0]
	require.NotNil(t, service.HAService)
	// Our daemon has the primary role in this service.
	require.EqualValues(t, daemon4.ID, service.HAService.PrimaryID)
	// The status should have been collected for primary and secondary.
	require.False(t, service.HAService.PrimaryStatusCollectedAt.IsZero())
	require.False(t, service.HAService.SecondaryStatusCollectedAt.IsZero())
	// The "age" value indicates how long ago the secondary status have
	// been fetched. Therefore, the timestamp of the secondary status should
	// be earlier than the status of the primary.
	require.True(t, service.HAService.PrimaryStatusCollectedAt.After(service.HAService.SecondaryStatusCollectedAt))
	// Both servers should be in load balancing state.
	require.Equal(t, "load-balancing", service.HAService.PrimaryLastState)
	require.Equal(t, "load-balancing", service.HAService.SecondaryLastState)
	require.ElementsMatch(t, []string{"server1"}, service.HAService.PrimaryLastScopes)
	require.ElementsMatch(t, []string{"server2"}, service.HAService.SecondaryLastScopes)
	require.True(t, service.HAService.PrimaryReachable)
	require.True(t, service.HAService.SecondaryReachable)
	// The failover event hasn't been observed yet.
	require.True(t, service.HAService.PrimaryLastFailoverAt.IsZero())
	require.True(t, service.HAService.SecondaryLastFailoverAt.IsZero())

	// These fields are only available in Kea 1.7.8+.
	if version178 {
		require.NotNil(t, service.HAService.SecondaryCommInterrupted)
		require.True(t, *service.HAService.SecondaryCommInterrupted)
		require.EqualValues(t, 1, service.HAService.SecondaryConnectingClients)
		require.EqualValues(t, 2, service.HAService.SecondaryUnackedClients)
		require.EqualValues(t, 3, service.HAService.SecondaryUnackedClientsLeft)
		require.EqualValues(t, 10, service.HAService.SecondaryAnalyzedPackets)
	}

	// The second service is the DHCPv6 service.
	service = services[1]

	require.NotNil(t, service.HAService)
	require.EqualValues(t, daemon6.ID, service.HAService.SecondaryID)
	// The status should have been collected for standby and primary.
	require.False(t, service.HAService.PrimaryStatusCollectedAt.IsZero())
	require.False(t, service.HAService.SecondaryStatusCollectedAt.IsZero())
	// The "age" value indicates how long ago the secondary status have
	// been fetched. Therefore, the timestamp of the primary status should
	// be earlier than the status of the primary.
	require.True(t, service.HAService.SecondaryStatusCollectedAt.After(service.HAService.PrimaryStatusCollectedAt))
	require.Equal(t, "waiting", service.HAService.PrimaryLastState)
	require.Equal(t, "hot-standby", service.HAService.SecondaryLastState)
	require.ElementsMatch(t, []string{"server1"}, service.HAService.PrimaryLastScopes)
	require.Empty(t, service.HAService.SecondaryLastScopes)
	require.True(t, service.HAService.PrimaryReachable)
	require.True(t, service.HAService.SecondaryReachable)

	// These fields are only available in Kea 1.7.8+.
	if version178 {
		require.NotNil(t, service.HAService.PrimaryCommInterrupted)
		require.True(t, *service.HAService.PrimaryCommInterrupted)
		require.EqualValues(t, 2, service.HAService.PrimaryConnectingClients)
		require.EqualValues(t, 3, service.HAService.PrimaryUnackedClients)
		require.EqualValues(t, 4, service.HAService.PrimaryUnackedClientsLeft)
		require.EqualValues(t, 15, service.HAService.PrimaryAnalyzedPackets)
	}

	// Pull the data again.
	err = puller.pullData()
	require.NoError(t, err)

	// There should still be two services, one for DHCPv4 and one for DHCPv6.
	services, err = dbmodel.GetDetailedAllServices(db)
	require.NoError(t, err)
	require.Len(t, services, 2)

	// Validate the values of the DHCPv4 service.
	service = services[0]
	require.NotNil(t, service.HAService)
	require.EqualValues(t, daemon4.ID, service.HAService.PrimaryID)
	require.False(t, service.HAService.PrimaryStatusCollectedAt.IsZero())
	require.False(t, service.HAService.SecondaryStatusCollectedAt.IsZero())

	// The primary state is now partner-down and the secondary state is unknown.
	require.Equal(t, "partner-down", service.HAService.PrimaryLastState)
	require.Equal(t, "unavailable", service.HAService.SecondaryLastState)
	require.ElementsMatch(t, []string{"server1", "server2"}, service.HAService.PrimaryLastScopes)
	require.Empty(t, service.HAService.SecondaryLastScopes)
	require.True(t, service.HAService.PrimaryReachable)
	require.False(t, service.HAService.SecondaryReachable)
	// The partner-down state is the indication that the failover took place.
	// This should be recorded for the primary server.
	require.False(t, service.HAService.PrimaryLastFailoverAt.IsZero())
	require.True(t, service.HAService.SecondaryLastFailoverAt.IsZero())

	// These fields are only available in Kea 1.7.8+.
	if version178 {
		require.NotNil(t, service.HAService.SecondaryCommInterrupted)
		require.True(t, *service.HAService.SecondaryCommInterrupted)
		// In the partner-down state they should be all reset.
		require.Zero(t, service.HAService.SecondaryConnectingClients)
		require.Zero(t, service.HAService.SecondaryUnackedClients)
		require.Zero(t, service.HAService.SecondaryUnackedClientsLeft)
		require.Zero(t, service.HAService.SecondaryAnalyzedPackets)
	}

	// The second service is the DHCPv6 service. The status should
	// remain the same for the DHCPv6 server because we were unable to communicate
	// with the server. The state may be overridden if the partner of that server
	// returns a different state for this server.
	service = services[1]
	require.NotNil(t, service.HAService)
	require.EqualValues(t, daemon6.ID, service.HAService.SecondaryID)
	require.False(t, service.HAService.PrimaryStatusCollectedAt.IsZero())
	require.False(t, service.HAService.SecondaryStatusCollectedAt.IsZero())
	require.True(t, service.HAService.SecondaryStatusCollectedAt.After(service.HAService.PrimaryStatusCollectedAt))
	require.ElementsMatch(t, []string{"server1"}, service.HAService.PrimaryLastScopes)
	require.Empty(t, service.HAService.SecondaryLastScopes)
	require.True(t, service.HAService.PrimaryReachable)
	require.False(t, service.HAService.SecondaryReachable)

	// These fields are only available in Kea 1.7.8+.
	if version178 {
		require.NotNil(t, service.HAService.PrimaryCommInterrupted)
		require.True(t, *service.HAService.PrimaryCommInterrupted)
		require.EqualValues(t, 2, service.HAService.PrimaryConnectingClients)
		require.EqualValues(t, 3, service.HAService.PrimaryUnackedClients)
		require.EqualValues(t, 4, service.HAService.PrimaryUnackedClientsLeft)
		require.EqualValues(t, 15, service.HAService.PrimaryAnalyzedPackets)
	}
}

// Test that HA status can be fetched and updated via the HA status puller
// mechanism.
func TestPullHAStatus(t *testing.T) {
	testPullHAStatus(t, false)
}

// Test that HA status can be fetched and updated via the HA status puller
// mechanism for Kea versions 1.7.8 and later.
func TestPullHAStatus178(t *testing.T) {
	testPullHAStatus(t, true)
}

// Test that HA status for the hub-and-spoke case is properly propagated.
func TestPullHAStatusHub(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Create the hub
	m := &dbmodel.Machine{
		Address:   "localhost",
		AgentPort: 8080,
	}
	err := dbmodel.AddMachine(db, m)
	require.NoError(t, err)

	accessPoints := []*dbmodel.AccessPoint{
		{
			Type:     dbmodel.AccessPointControl,
			Address:  "",
			Port:     1234,
			Key:      "",
			Protocol: protocoltype.HTTP,
		},
	}

	dhcp4 := dbmodel.NewDaemon(m, daemonname.DHCPv4, true, accessPoints)

	fec := &storktest.FakeEventCenter{}
	lookup := dbmodel.NewDHCPOptionDefinitionLookup()

	configData := []byte(`{
        "Dhcp4": {
            "hooks-libraries": [
                {
                    "library": "libdhcp_ha.so",
                    "parameters": {
                        "high-availability": [
                            {
                                "this-server-name": "server2",
                                "mode": "hot-standby",
                                "peers": [
                                    {
                                        "name": "server1",
                                        "url":  "http://192.0.2.33:8000",
                                        "role": "primary"
                                    },
                                    {
                                        "name": "server2",
                                        "url":  "http://192.0.2.66:8000",
                                        "role": "standby"
                                    }
                                ]
                            },
                            {
                                "this-server-name": "server4",
                                "mode": "hot-standby",
                                "peers": [
                                    {
                                        "name": "server3",
                                        "url":  "http://192.0.2.99:8000",
                                        "role": "primary"
                                    },
                                    {
                                        "name": "server4",
                                        "url":  "http://192.0.2.133:8000",
                                        "role": "standby"
                                    }
                                ]
                            }
                        ]
                    }
                }
            ]
        }
    }`)

	err = dhcp4.SetKeaConfigFromJSON(configData)
	require.NoError(t, err)

	err = dbmodel.AddDaemon(db, dhcp4)
	require.NoError(t, err)

	err = CommitDaemonsIntoDB(db,
		[]*dbmodel.Daemon{dhcp4},
		fec,
		[]DaemonStateMeta{{IsConfigChanged: true}},
		lookup,
	)
	require.NoError(t, err)

	services, err := dbmodel.GetDetailedAllServices(db)
	require.NoError(t, err)
	require.Len(t, services, 2)

	// The puller requires fetch interval to be present in the database.
	err = dbmodel.InitializeSettings(db, 0)
	require.NoError(t, err)

	fa := agentcommtest.NewFakeAgents(mockGetStatusWithHAHub, nil)

	// Create the puller which normally fetches the HA status periodically.
	puller, err := NewHAStatusPuller(db, fa)
	require.NoError(t, err)
	require.NotNil(t, puller)

	// No need to wait for the puller to fetch the status.
	err = puller.pullData()
	require.NoError(t, err)

	services, err = dbmodel.GetDetailedAllServices(db)
	require.NoError(t, err)
	require.Len(t, services, 2)

	require.NotNil(t, services[0].HAService)
	require.NotNil(t, services[1].HAService)
	require.Equal(t, "hot-standby", services[0].HAService.HAMode)
	require.Equal(t, "hot-standby", services[1].HAService.HAMode)
	require.Equal(t, "hot-standby", services[0].HAService.PrimaryLastState)
	require.Equal(t, "hot-standby", services[1].HAService.PrimaryLastState)
	require.Contains(t, services[0].HAService.PrimaryLastScopes, "server1")
	require.Contains(t, services[1].HAService.PrimaryLastScopes, "server3")
	require.Empty(t, services[0].HAService.SecondaryLastScopes)
	require.Empty(t, services[1].HAService.SecondaryLastScopes)

	// Pull the data again. it should affect the state of our services.
	err = puller.pullData()
	require.NoError(t, err)

	services, err = dbmodel.GetDetailedAllServices(db)
	require.NoError(t, err)
	require.Len(t, services, 2)

	// The statue should have changed.
	require.NotNil(t, services[0].HAService)
	require.NotNil(t, services[1].HAService)
	require.Equal(t, "hot-standby", services[0].HAService.HAMode)
	require.Equal(t, "hot-standby", services[1].HAService.HAMode)
	require.Equal(t, "unavailable", services[0].HAService.PrimaryLastState)
	require.Equal(t, "unavailable", services[1].HAService.PrimaryLastState)
	require.Contains(t, services[0].HAService.SecondaryLastScopes, "server1")
	require.Contains(t, services[1].HAService.SecondaryLastScopes, "server3")
	require.Empty(t, services[0].HAService.PrimaryLastScopes)
	require.Empty(t, services[1].HAService.PrimaryLastScopes)
}
