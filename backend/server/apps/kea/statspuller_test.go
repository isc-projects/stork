package kea

import (
	"testing"

	//log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"

	"isc.org/stork/server/agentcomm"
	dbmodel "isc.org/stork/server/database/model"
	dbtest "isc.org/stork/server/database/test"
	storktest "isc.org/stork/server/test"
)

// Check creating and shutting down StatsPuller.
func TestStatsPullerBasic(t *testing.T) {
	// prepare db
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// prepare fake agents
	fa := storktest.NewFakeAgents(nil)

	sp := NewStatsPuller(db, fa)
	sp.Shutdown()
}

// Check if pulling stats works.
func TestStatsPullerPullStats(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// prepare fake agents
	fa := storktest.NewFakeAgents(func(callNo int, cmdResponses []interface{}) {
		// DHCPv4
		daemons, _ := agentcomm.NewKeaDaemons("dhcp4")
		command, _ := agentcomm.NewKeaCommand("stat-lease4-get", daemons, nil)
		json := `[{
                            "result": 0,
                            "text": "Everything is fine",
                            "arguments": {
                                "result-set": {
                                    "columns": [ "subnet-id", "total-addresses", "assigned-addresses", "declined-addresses" ],
                                    "rows": [
                                        [ 10, 256, 111, 0 ],
                                        [ 20, 4098, 2034, 4 ]
                                    ],
                                    "timestamp": "2018-05-04 15:03:37.000000"
                                }
                            }
                         }]`
		agentcomm.UnmarshalKeaResponseList(command, json, cmdResponses[0])

		// DHCPv6
		daemons, _ = agentcomm.NewKeaDaemons("dhcp6")
		command, _ = agentcomm.NewKeaCommand("stat-lease6-get", daemons, nil)
		json = `[{
                           "result": 0,
                           "text": "Everything is fine",
                           "arguments": {
                               "result-set": {
                                   "columns": [ "subnet-id", "total-nas", "assigned-nas", "declined-nas", "total-pds", "assigned-pds" ],
                                   "rows": [
                                       [ 10, 4096, 2400, 3, 0, 0],
                                       [ 20, 0, 0, 0, 1048, 233 ],
                                       [ 30, 256, 60, 0, 1048, 15 ]
                                   ],
                                   "timestamp": "2018-05-04 15:03:37.000000"
                               }
                           }
                        }]`
		agentcomm.UnmarshalKeaResponseList(command, json, cmdResponses[1])
	})

	// add one machine with one kea app
	m := &dbmodel.Machine{
		ID:        0,
		Address:   "localhost",
		AgentPort: 8080,
	}
	err := dbmodel.AddMachine(db, m)
	require.NoError(t, err)
	require.NotEqual(t, 0, m.ID)
	a := &dbmodel.App{
		ID:          0,
		MachineID:   m.ID,
		Type:        dbmodel.KeaAppType,
		CtrlAddress: "cool.example.org",
		CtrlPort:    1234,
		CtrlKey:     "",
		Active:      true,
		Details: dbmodel.AppKea{
			Daemons: []*dbmodel.KeaDaemon{
				{
					Active: true,
					Name:   "dhcp4",
				},
				{
					Active: true,
					Name:   "dhcp6",
				},
			},
		},
	}
	err = dbmodel.AddApp(db, a)
	require.NoError(t, err)
	require.NotEqual(t, 0, a.ID)

	// prepare stats puller
	sp := NewStatsPuller(db, fa)
	// shutdown stats puller at the end
	defer sp.Shutdown()

	// invoke pulling stats
	appsOkCnt, err := sp.pullLeaseStats()
	require.NoError(t, err)
	require.Equal(t, 1, appsOkCnt)

	// TODO: check collected stats
}

// Check if empty stats response is handled correctly and when stat plugin is not loaded in Kea.
func TestStatsPullerEmptyResponse(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// prepare fake agents
	fa := storktest.NewFakeAgents(func(callNo int, cmdResponses []interface{}) {
		// DHCPv4
		daemons, _ := agentcomm.NewKeaDaemons("dhcp4")
		command, _ := agentcomm.NewKeaCommand("stat-lease4-get", daemons, nil)
		// simulate empty response
		json := `[{
                            "result": 0,
                            "text": "Everything is fine",
                            "arguments": {}
                         }]`
		agentcomm.UnmarshalKeaResponseList(command, json, cmdResponses[0])

		// DHCPv6
		daemons, _ = agentcomm.NewKeaDaemons("dhcp6")
		command, _ = agentcomm.NewKeaCommand("stat-lease6-get", daemons, nil)
		// simulate not loaded stat plugin in kea
		json = `[{
                           "result": 2,
                           "text": "'stat-lease6-get' command not supported."
                        }]`
		agentcomm.UnmarshalKeaResponseList(command, json, cmdResponses[1])
	})

	// add one machine with one kea app
	m := &dbmodel.Machine{
		ID:        0,
		Address:   "localhost",
		AgentPort: 8080,
	}
	err := dbmodel.AddMachine(db, m)
	require.NoError(t, err)
	require.NotEqual(t, 0, m.ID)
	a := &dbmodel.App{
		ID:          0,
		MachineID:   m.ID,
		Type:        dbmodel.KeaAppType,
		CtrlAddress: "cool.example.org",
		CtrlPort:    1234,
		CtrlKey:     "",
		Active:      true,
		Details: dbmodel.AppKea{
			Daemons: []*dbmodel.KeaDaemon{
				{
					Active: true,
					Name:   "dhcp4",
				},
				{
					Active: true,
					Name:   "dhcp6",
				},
			},
		},
	}
	err = dbmodel.AddApp(db, a)
	require.NoError(t, err)
	require.NotEqual(t, 0, a.ID)

	// prepare stats puller
	sp := NewStatsPuller(db, fa)
	// shutdown stats puller at the end
	defer sp.Shutdown()

	// invoke pulling stats
	appsOkCnt, err := sp.pullLeaseStats()
	require.NoError(t, err)
	require.Equal(t, 1, appsOkCnt)

	// TODO: check collected stats
}
