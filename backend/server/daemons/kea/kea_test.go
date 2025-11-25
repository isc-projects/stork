package kea

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	keactrl "isc.org/stork/daemonctrl/kea"
	"isc.org/stork/datamodel/daemonname"
	"isc.org/stork/datamodel/protocoltype"
	agentcommtest "isc.org/stork/server/agentcomm/test"
	dbmodel "isc.org/stork/server/database/model"
	dbtest "isc.org/stork/server/database/test"
	storktest "isc.org/stork/server/test/dbmodel"
)

// Kea servers' response to config-get command from CA. The argument indicates if
// it is a response from a single server or two servers.
func mockGetConfigFromCAResponse(daemons int, cmdResponses []interface{}) {
	versionResp := cmdResponses[0].(*VersionGetResponse)
	*versionResp = VersionGetResponse{
		ResponseHeader: keactrl.ResponseHeader{
			Result: 0,
		},
		Arguments: &VersionGetRespArgs{
			Extended: "Extended version",
		},
	}

	configResp := cmdResponses[1].(*keactrl.Response)
	*configResp = keactrl.Response{
		ResponseHeader: keactrl.ResponseHeader{
			Result: 0,
		},
	}
	if daemons > 1 {
		configArgs := map[string]interface{}{
			"Control-agent": map[string]interface{}{
				"control-sockets": map[string]interface{}{
					"dhcp4": map[string]interface{}{
						"socket-name": "aaa",
						"socket-type": "unix",
					},
					"dhcp6": map[string]interface{}{
						"socket-name": "bbbb",
						"socket-type": "unix",
					},
				},
			},
		}
		configBytes, _ := json.Marshal(configArgs)
		configResp.Arguments = configBytes
	} else {
		configArgs := map[string]interface{}{
			"Control-agent": map[string]interface{}{
				"control-sockets": map[string]interface{}{
					"dhcp4": map[string]interface{}{
						"socket-name": "aaa",
						"socket-type": "unix",
					},
				},
				"loggers": []interface{}{
					map[string]interface{}{
						"name":     "kea-ca",
						"severity": "DEBUG",
						"output_options": []interface{}{
							map[string]interface{}{
								"output": "stdout",
							},
						},
					},
					map[string]interface{}{
						"name":     "kea-ca.sockets",
						"severity": "DEBUG",
						"output_options": []interface{}{
							map[string]interface{}{
								"output": "/tmp/kea-ca-sockets.log",
							},
						},
					},
				},
			},
		}
		configBytes, _ := json.Marshal(configArgs)
		configResp.Arguments = configBytes
	}
}

// Kea servers' response to config-get command from other Kea daemons. The
// argument indicates of the IP family related to the daemon.

func mockGetConfigFromOtherDaemonsResponse(family int, cmdResponses []interface{}) {
	// For non-CA daemons, we have version-get, config-get, and status-get responses
	versionResp := cmdResponses[0].(*VersionGetResponse)
	*versionResp = VersionGetResponse{
		ResponseHeader: keactrl.ResponseHeader{
			Result: 0,
		},
		Arguments: &VersionGetRespArgs{
			Extended: "Extended version",
		},
	}

	configResp := cmdResponses[1].(*keactrl.Response)
	*configResp = keactrl.Response{
		ResponseHeader: keactrl.ResponseHeader{
			Result: 0,
		},
	}
	configArgs := map[string]interface{}{
		fmt.Sprintf("Dhcp%d", family): map[string]interface{}{
			"hooks-libraries": []interface{}{
				map[string]interface{}{
					"library": "hook_abc.so",
				},
				map[string]interface{}{
					"library": "hook_def.so",
				},
			},
		},
	}
	configBytes, _ := json.Marshal(configArgs)
	configResp.Arguments = configBytes

	if len(cmdResponses) > 2 {
		statusResp := cmdResponses[2].(*StatusGetResponse)
		*statusResp = StatusGetResponse{
			ResponseHeader: keactrl.ResponseHeader{
				Result: 0,
			},
			Arguments: &StatusGetRespArgs{
				Pid: 123,
			},
		}
	}
}

// Test that GetConfig returns the correct configuration.
func TestGetConfig(t *testing.T) {
	// Arrange
	ctx := context.Background()

	keaMock := func(callNo int, cmdResponses []any) {
		require.Equal(t, 0, callNo)
		require.Len(t, cmdResponses, 1)

		response := cmdResponses[0].(*keactrl.Response)
		response.Result = keactrl.ResponseSuccess
		response.Arguments = []byte(`{ "Dhcp4": {} }`)
	}
	fa := agentcommtest.NewFakeAgents(keaMock, nil)

	daemon := dbmodel.NewDaemon(&dbmodel.Machine{
		Address:   "192.0.2.0",
		AgentPort: 1111,
	}, daemonname.DHCPv4, true, []*dbmodel.AccessPoint{{
		Type:     dbmodel.AccessPointControl,
		Address:  "192.0.2.0",
		Port:     1234,
		Protocol: protocoltype.HTTPS,
	}})

	// Act
	config, err := GetConfig(ctx, fa, daemon)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, config)
	require.True(t, config.IsDHCPv4())
}

// Check if GetDaemonState returns response to the forwarded command.
func TestGetDaemonStateWith1Daemon(t *testing.T) {
	ctx := context.Background()

	// check getting config of 1 daemon
	keaMock := func(callNo int, cmdResponses []interface{}) {
		require.LessOrEqual(t, callNo, 2)
		mockGetConfigFromCAResponse(1, cmdResponses)
	}
	fa := agentcommtest.NewFakeAgents(keaMock, nil)

	accessPoints := []*dbmodel.AccessPoint{
		{
			Type:     dbmodel.AccessPointControl,
			Address:  "192.0.2.0",
			Port:     1234,
			Protocol: protocoltype.HTTPS,
		},
	}

	daemon := dbmodel.NewDaemon(&dbmodel.Machine{
		Address:   "192.0.2.0",
		AgentPort: 1111,
	}, daemonname.CA, true, accessPoints)

	daemon, meta := GetDaemonWithRefreshedState(ctx, fa, daemon)

	require.Contains(t, fa.RecordedURLs, "https://192.0.2.0:1234/")
	require.Equal(t, keactrl.VersionGet, fa.RecordedCommands[0].GetCommand())
	require.Equal(t, keactrl.ConfigGet, fa.RecordedCommands[1].GetCommand())
	require.NotNil(t, meta)
	require.True(t, meta.IsConfigChanged)
	require.Empty(t, meta.Events)

	// Committing daemon into database assigns it an ID.
	daemon.ID = 1

	// Refresh state again, should be no changes.
	_, meta = GetDaemonWithRefreshedState(ctx, fa, daemon)
	require.NotNil(t, meta)
	require.False(t, meta.IsConfigChanged)
	require.Empty(t, meta.Events)
}

func TestGetDaemonStateWith2Daemons(t *testing.T) {
	ctx := context.Background()

	// check getting configs of 2 daemons
	keaMock := func(callNo int, cmdResponses []interface{}) {
		switch callNo {
		case 0:
			mockGetConfigFromCAResponse(2, cmdResponses)
		case 1:
			mockGetConfigFromOtherDaemonsResponse(4, cmdResponses)
		}
	}
	fa := agentcommtest.NewFakeAgents(keaMock, nil)

	accessPoints := []*dbmodel.AccessPoint{
		{
			Type:     dbmodel.AccessPointControl,
			Address:  "192.0.2.0",
			Port:     1234,
			Protocol: protocoltype.HTTP,
		},
	}

	daemon := dbmodel.NewDaemon(&dbmodel.Machine{
		Address:   "192.0.2.0",
		AgentPort: 1111,
	}, daemonname.CA, true, accessPoints)

	GetDaemonWithRefreshedState(ctx, fa, daemon)

	require.Contains(t, fa.RecordedURLs, "http://192.0.2.0:1234/")
	require.Equal(t, keactrl.VersionGet, fa.RecordedCommands[0].GetCommand())
	require.Equal(t, keactrl.ConfigGet, fa.RecordedCommands[1].GetCommand())
}

// Check GetDaemonWithRefreshedState when daemon already exists.
func TestGetDaemonStateForExistingDaemon(t *testing.T) {
	ctx := context.Background()

	// check getting config of 1 daemon
	keaMock := func(callNo int, cmdResponses []interface{}) {
		// Since we're testing with a CA daemon, always use CA response
		mockGetConfigFromCAResponse(1, cmdResponses)
	}
	fa := agentcommtest.NewFakeAgents(keaMock, nil)

	accessPoints := []*dbmodel.AccessPoint{
		{
			Type:     dbmodel.AccessPointControl,
			Address:  "192.0.2.0",
			Port:     1234,
			Protocol: protocoltype.HTTPS,
		},
	}

	daemon := dbmodel.NewDaemon(&dbmodel.Machine{
		Address:   "192.0.2.0",
		AgentPort: 1111,
	}, daemonname.CA, false, accessPoints)
	daemon.ID = 42

	err := daemon.SetKeaConfigFromJSON([]byte(`{
        "Control-agent": {
            "loggers": [
                {
                    "name": "kea-ca",
                    "severity": "debug",
                    "output_options": [
                        {
                            "output": "stdout"
                        }
                    ]
                }
            ]
        }
    }`))
	require.NoError(t, err)
	daemon.LogTargets[0].ID = 1

	// Remember current config hash for daemon.
	caHash := daemon.KeaDaemon.ConfigHash

	daemon, meta := GetDaemonWithRefreshedState(ctx, fa, daemon)
	require.NotNil(t, daemon)
	require.NotNil(t, meta)
	require.True(t, meta.IsConfigChanged)
	require.NotEmpty(t, meta.Events)

	require.Contains(t, fa.RecordedURLs, "https://192.0.2.0:1234/")
	require.Equal(t, keactrl.VersionGet, fa.RecordedCommands[0].GetCommand())
	require.Equal(t, keactrl.ConfigGet, fa.RecordedCommands[1].GetCommand())

	// We successfully communicated with the daemon so it should be in active state.
	require.True(t, daemon.Active)

	// Make sure that logging information is populated correctly.
	require.Len(t, daemon.LogTargets, 2)
	for _, target := range daemon.LogTargets {
		if target.Name == "kea-ca" {
			// The CA daemon should have an updated log target information, but the
			// ID should not change as a result of the update.
			require.EqualValues(t, 1, target.ID)
			require.Equal(t, "debug", target.Severity)
			require.Equal(t, "stdout", target.Output)
		} else {
			// This is new logging target. Its ID should be 0.
			require.Zero(t, target.ID)
			require.Equal(t, "debug", target.Severity)
			require.Equal(t, "/tmp/kea-ca-sockets.log", target.Output)
		}
	}

	// Make sure that the config hash has changed.
	require.NotEmpty(t, daemon.KeaDaemon.ConfigHash)
	require.NotEqual(t, daemon.KeaDaemon.ConfigHash, caHash)

	caConfig := daemon.KeaDaemon.Config

	daemon, meta = GetDaemonWithRefreshedState(ctx, fa, daemon)
	require.NotNil(t, daemon)
	require.NotNil(t, meta)

	require.NotNil(t, daemon.KeaDaemon.Config)
	// Since we're using the same config content, the config should be functionally equivalent
	require.Equal(t, caConfig.Config, daemon.KeaDaemon.Config.Config)
}

// Check GetDaemonHooks when daemon has hooks configured.
func TestGetDaemonHooksFrom1Daemon(t *testing.T) {
	daemon := &dbmodel.Daemon{
		Name:      daemonname.DHCPv4,
		KeaDaemon: &dbmodel.KeaDaemon{},
	}

	// Set configuration with hooks
	err := daemon.SetKeaConfigFromJSON([]byte(`{
		"Dhcp4": {
			"hooks-libraries": [
				{
					"library": "hook_abc.so"
				},
				{
					"library": "hook_def.so"
				}
			]
		}
	}`))
	require.NoError(t, err)

	hooks := GetDaemonHooks(daemon)
	require.Len(t, hooks, 2)
	require.Equal(t, "hook_abc.so", hooks[0])
	require.Equal(t, "hook_def.so", hooks[1])
}

// Tests that Kea daemon can be added and then updated in the database.
func TestCommitDaemonIntoDB(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	fec := &storktest.FakeEventCenter{}

	machine := &dbmodel.Machine{
		ID:        0,
		Address:   "localhost",
		AgentPort: 8080,
	}
	err := dbmodel.AddMachine(db, machine)
	require.NoError(t, err)
	require.NotZero(t, machine.ID)

	// add daemon with particular access point
	accessPoints := []*dbmodel.AccessPoint{
		{
			Type:     dbmodel.AccessPointControl,
			Address:  "",
			Port:     1234,
			Protocol: protocoltype.HTTP,
		},
	}
	daemon := dbmodel.NewDaemon(machine, daemonname.CA, true, accessPoints)

	lookup := dbmodel.NewDHCPOptionDefinitionLookup()
	daemons := []*dbmodel.Daemon{daemon}
	states := []DaemonStateMeta{{IsConfigChanged: true}}
	err = CommitDaemonsIntoDB(db, daemons, fec, states, lookup)
	require.NoError(t, err)

	// now change access point (different port) and trigger updating daemon in database
	daemon.AccessPoints[0].Port = 2345
	daemon.AccessPoints[0].Protocol = protocoltype.HTTPS
	states = []DaemonStateMeta{{IsConfigChanged: true}}
	err = CommitDaemonsIntoDB(db, daemons, fec, states, lookup)
	require.NoError(t, err)

	returned, err := dbmodel.GetDaemonByID(db, daemon.ID)
	require.NoError(t, err)
	require.NotNil(t, returned)
	require.Len(t, returned.AccessPoints, 1)
	require.EqualValues(t, 2345, returned.AccessPoints[0].Port)
	require.Equal(t, protocoltype.HTTPS, returned.AccessPoints[0].Protocol)
}

// Test that the changed configuration is detected correctly.
func TestIsDaemonConfigChanged(t *testing.T) {
	t.Run("Missing both KeaDaemon", func(t *testing.T) {
		// Arrange
		daemonOld := &dbmodel.Daemon{}
		daemonNew := &dbmodel.Daemon{}

		// Act & Assert
		require.False(t, isDaemonConfigChanged(daemonOld, daemonNew))
	})

	t.Run("Missing old KeaDaemon", func(t *testing.T) {
		// Arrange
		daemonOld := &dbmodel.Daemon{}
		daemonNew := &dbmodel.Daemon{KeaDaemon: &dbmodel.KeaDaemon{}}

		// Act & Assert
		require.True(t, isDaemonConfigChanged(daemonOld, daemonNew))
	})

	t.Run("Missing new KeaDaemon", func(t *testing.T) {
		// Arrange
		daemonOld := &dbmodel.Daemon{KeaDaemon: &dbmodel.KeaDaemon{}}
		daemonNew := &dbmodel.Daemon{}

		// Act & Assert
		require.True(t, isDaemonConfigChanged(daemonOld, daemonNew))
	})

	t.Run("Same empty config hash", func(t *testing.T) {
		// Arrange
		daemonOld := &dbmodel.Daemon{KeaDaemon: &dbmodel.KeaDaemon{}}
		daemonNew := &dbmodel.Daemon{KeaDaemon: &dbmodel.KeaDaemon{}}

		// Act & Assert
		require.False(t, isDaemonConfigChanged(daemonOld, daemonNew))
	})

	t.Run("Same config hash", func(t *testing.T) {
		// Arrange
		daemonOld := &dbmodel.Daemon{KeaDaemon: &dbmodel.KeaDaemon{ConfigHash: "abc"}}
		daemonNew := &dbmodel.Daemon{KeaDaemon: &dbmodel.KeaDaemon{ConfigHash: "abc"}}

		// Act & Assert
		require.False(t, isDaemonConfigChanged(daemonOld, daemonNew))
	})

	t.Run("Different config hash", func(t *testing.T) {
		// Arrange
		daemonOld := &dbmodel.Daemon{KeaDaemon: &dbmodel.KeaDaemon{ConfigHash: "abc"}}
		daemonNew := &dbmodel.Daemon{KeaDaemon: &dbmodel.KeaDaemon{ConfigHash: "def"}}

		// Act & Assert
		require.True(t, isDaemonConfigChanged(daemonOld, daemonNew))
	})
}
