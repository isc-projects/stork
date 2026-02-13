package agent

import (
	"encoding/json"
	"fmt"
	"net"
	"path"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"gopkg.in/h2non/gock.v1"
	keaconfig "isc.org/stork/daemoncfg/kea"
	keactrl "isc.org/stork/daemonctrl/kea"
	"isc.org/stork/datamodel/daemonname"
	"isc.org/stork/datamodel/protocoltype"
	"isc.org/stork/testutil"
	storkutil "isc.org/stork/util"
)

//go:generate mockgen -package=agent -destination=commandexecutormock_test.go isc.org/stork/util CommandExecutor

// Test the case that the command is successfully sent to Kea.
func TestSendCommand(t *testing.T) {
	// Expect appropriate content type and the body. If they are not matched
	// an error will be raised.
	defer gock.Off()
	gock.New("http://localhost:45634").
		MatchHeader("Content-Type", "application/json").
		JSON(map[string]string{"command": "list-commands"}).
		Post("/").
		Reply(200).
		JSON([]map[string]int{{"result": int(keactrl.ResponseSuccess)}})

	command := keactrl.NewCommandBase(keactrl.ListCommands, daemonname.CA)

	accessPoint := AccessPoint{Type: AccessPointControl, Address: "localhost", Port: 45634, Protocol: "http"}
	daemon := &keaDaemon{
		daemon: daemon{
			Name:         daemonname.CA,
			AccessPoints: []AccessPoint{accessPoint},
		},
		connector: newKeaConnector(accessPoint, HTTPClientConfig{Interceptor: gock.InterceptClient}),
	}
	var response keactrl.Response
	err := daemon.sendCommand(t.Context(), command, &response)
	require.NoError(t, err)
	require.False(t, gock.HasUnmatchedRequest())
}

// Test the case that the command is not successfully sent to Kea because
// there is no control access point.
func TestSendCommandNoAccessPoint(t *testing.T) {
	command := keactrl.NewCommandBase(keactrl.ListCommands, daemonname.CA)

	daemon := &keaDaemon{
		daemon: daemon{
			Name:         daemonname.DHCPv4,
			AccessPoints: []AccessPoint{},
		},
		connector: nil,
	}

	var response keactrl.Response
	err := daemon.sendCommand(t.Context(), command, &response)
	require.ErrorContains(t, err, "no control access point")
}

// Test the case when Kea returns invalid response to the command.
func TestSendCommandInvalidResponse(t *testing.T) {
	// Return invalid response. Arguments must be a map not an integer.
	defer gock.Off()
	gock.New("http://localhost:45634").
		MatchHeader("Content-Type", "application/json").
		JSON(map[string]any{"command": "version-get", "service": []string{"dhcp4"}}).
		Post("/").
		Reply(200).
		JSON([]map[string]interface{}{
			{"result": 0, "text": "1.0.0", "arguments": 1},
		})

	command := keactrl.NewCommandBase(keactrl.VersionGet, daemonname.DHCPv4)

	accessPoint := AccessPoint{Type: AccessPointControl, Address: "localhost", Port: 45634, Protocol: "http"}
	daemon := &keaDaemon{
		daemon: daemon{
			Name:         daemonname.DHCPv4,
			AccessPoints: []AccessPoint{accessPoint},
		},
		connector: newKeaConnector(accessPoint, HTTPClientConfig{Interceptor: gock.InterceptClient}),
	}

	type versionGet struct {
		keactrl.ResponseHeader
		Arguments struct {
			ExtendedVersion string
		}
	}
	var response versionGet
	err := daemon.sendCommand(t.Context(), command, &response)
	require.Error(t, err)
	require.False(t, gock.HasUnmatchedRequest())
}

// Test the case when Kea server is unreachable.
func TestSendCommandNoKea(t *testing.T) {
	command := keactrl.NewCommandBase(keactrl.ListCommands, daemonname.CA)
	accessPoint := AccessPoint{Type: AccessPointControl, Address: "localhost", Port: 45634, Protocol: "http"}
	daemon := &keaDaemon{
		daemon: daemon{
			Name:         daemonname.CA,
			AccessPoints: []AccessPoint{accessPoint},
		},
		connector: newKeaConnector(accessPoint, HTTPClientConfig{}),
	}
	var response keactrl.Response
	err := daemon.sendCommand(t.Context(), command, &response)
	require.Error(t, err)
}

// Test the function which extracts the list of log files from the Kea
// daemon by sending the request to the Kea Control Agent and the
// daemons behind it.
func TestKeaAllowedLogs(t *testing.T) {
	// The first config-get command should go to the Kea Control Agent.
	// The logs should be extracted from there and the subsequent config-get
	// commands should be sent to the daemons with which the CA is configured
	// to communicate.
	defer gock.Off()
	caResponseJSON := `[{
        "result": 0,
        "arguments": {
            "Control-agent": {
                "control-sockets": {
                    "dhcp4": {
                        "socket-name": "/tmp/dhcp4.sock"
                    },
                    "dhcp6": {
                        "socket-name": "/tmp/dhcp6.sock"
                    }
                },
                "loggers": [
                    {
                        "output_options": [
                            {
                                "output": "/tmp/kea-ctrl-agent.log"
                            }
                        ]
                    }
                ]
            }
        }
    }]`
	caResponse := make([]map[string]interface{}, 1)
	err := json.Unmarshal([]byte(caResponseJSON), &caResponse)
	require.NoError(t, err)
	gock.New("https://localhost:45634").
		MatchHeader("Content-Type", "application/json").
		JSON(map[string]string{"command": "config-get"}).
		Post("/").
		Reply(200).
		JSON(caResponse)

	dhcpV4ResponsesJSON := `[
        {
            "result": 0,
            "arguments": {
                "Dhcp4": {
                    "loggers": [
                        {
                            "output_options": [
                                {
                                    "output": "/tmp/kea-dhcp4.log"
                                }
                            ]
                        }
                    ]
                }
            }
        }
	]`
	dhcpV4Responses := make([]map[string]interface{}, 1)
	err = json.Unmarshal([]byte(dhcpV4ResponsesJSON), &dhcpV4Responses)
	require.NoError(t, err)
	gock.New("https://localhost:45634").
		MatchHeader("Content-Type", "application/json").
		JSON(map[string]any{"command": "config-get", "service": []string{"dhcp4"}}).
		Post("/").
		Reply(200).
		JSON(dhcpV4Responses)

	dhcpV6ResponsesJSON := `[
        {
            "result": 0,
            "arguments": {
                "Dhcp6": {
                    "loggers": [
                        {
                            "output_options": [
                                {
                                    "output": "/tmp/kea-dhcp6.log"
                                }
                            ]
                        }
                    ]
                }
            }
        }
    ]`
	dhcpV6Responses := make([]map[string]interface{}, 1)
	err = json.Unmarshal([]byte(dhcpV6ResponsesJSON), &dhcpV6Responses)
	require.NoError(t, err)
	require.NoError(t, err)
	gock.New("https://localhost:45634").
		MatchHeader("Content-Type", "application/json").
		JSON(map[string]any{"command": "config-get", "service": []string{"dhcp6"}}).
		Post("/").
		Reply(200).
		JSON(dhcpV6Responses)

	accessPoint := AccessPoint{Type: AccessPointControl, Address: "localhost", Port: 45634, Protocol: protocoltype.HTTPS}
	connector := newKeaConnector(accessPoint, HTTPClientConfig{Interceptor: gock.InterceptClient})

	monitor := &monitor{daemons: []Daemon{
		&keaDaemon{
			daemon: daemon{
				Name:         daemonname.CA,
				AccessPoints: []AccessPoint{accessPoint},
			},
			connector: connector,
		},
		&keaDaemon{
			daemon: daemon{
				Name:         daemonname.DHCPv4,
				AccessPoints: []AccessPoint{accessPoint},
			},
			connector: connector,
		},
		&keaDaemon{
			daemon: daemon{
				Name:         daemonname.DHCPv6,
				AccessPoints: []AccessPoint{accessPoint},
			},
			connector: connector,
		},
	}}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	agentManager := NewMockAgentManager(ctrl)
	// We should have three log files recorded from the returned configurations.
	// One from CA, one from DHCPv4 and one from DHCPv6.
	agentManager.EXPECT().allowLog(gomock.Any()).Times(3)
	agentManager.EXPECT().allowLeaseTracking().Return(false, 0).AnyTimes()

	monitor.refreshDaemons(t.Context(), agentManager)

	require.NoError(t, err)
	require.False(t, gock.HasUnmatchedRequest())
}

// Test the function which extracts the list of log files from the Kea
// daemon by sending the request to the Kea Control Agent and the
// daemons behind it. This test variant uses output-options alias for
// logger configuration.
func TestKeaAllowedLogsOutputOptionsWithDash(t *testing.T) {
	// The first config-get command should go to the Kea Control Agent.
	// The logs should be extracted from there and the subsequent config-get
	// commands should be sent to the daemons with which the CA is configured
	// to communicate.
	defer gock.Off()
	caResponseJSON := `[{
        "result": 0,
        "arguments": {
            "Control-agent": {
                "control-sockets": {
                    "dhcp4": {
                        "socket-name": "/tmp/dhcp4.sock"
                    },
                    "dhcp6": {
                        "socket-name": "/tmp/dhcp6.sock"
                    }
                },
                "loggers": [
                    {
                        "output-options": [
                            {
                                "output": "/tmp/kea-ctrl-agent.log"
                            }
                        ]
                    }
                ]
            }
        }
    }]`
	caResponse := make([]map[string]interface{}, 1)
	err := json.Unmarshal([]byte(caResponseJSON), &caResponse)
	require.NoError(t, err)
	gock.New("https://localhost:45634").
		MatchHeader("Content-Type", "application/json").
		JSON(map[string]string{"command": "config-get"}).
		Post("/").
		Reply(200).
		JSON(caResponse)

	dhcpV4ResponsesJSON := `[
        {
            "result": 0,
            "arguments": {
                "Dhcp4": {
                    "loggers": [
                        {
                            "output-options": [
                                {
                                    "output": "/tmp/kea-dhcp4.log"
                                }
                            ]
                        }
                    ]
                }
            }
        }
	]`
	dhcpV4Response := make([]map[string]interface{}, 1)
	err = json.Unmarshal([]byte(dhcpV4ResponsesJSON), &dhcpV4Response)
	require.NoError(t, err)
	gock.New("https://localhost:45634").
		MatchHeader("Content-Type", "application/json").
		JSON(map[string]any{"command": "config-get", "service": []string{"dhcp4"}}).
		Post("/").
		Reply(200).
		JSON(dhcpV4Response)

	dhcpV6ResponsesJSON := `[
        {
            "result": 0,
            "arguments": {
                "Dhcp6": {
                    "loggers": [
                        {
                            "output-options": [
                                {
                                    "output": "/tmp/kea-dhcp6.log"
                                }
                            ]
                        }
                    ]
                }
            }
        }
    ]`
	dhcpV6Response := make([]map[string]interface{}, 1)
	err = json.Unmarshal([]byte(dhcpV6ResponsesJSON), &dhcpV6Response)
	require.NoError(t, err)
	gock.New("https://localhost:45634").
		MatchHeader("Content-Type", "application/json").
		JSON(map[string]any{"command": "config-get", "service": []string{"dhcp6"}}).
		Post("/").
		Reply(200).
		JSON(dhcpV6Response)

	accessPoint := AccessPoint{Type: AccessPointControl, Address: "localhost", Port: 45634, Protocol: protocoltype.HTTPS}
	connector := newKeaConnector(accessPoint, HTTPClientConfig{Interceptor: gock.InterceptClient})

	monitor := &monitor{daemons: []Daemon{
		&keaDaemon{
			daemon: daemon{
				Name:         daemonname.CA,
				AccessPoints: []AccessPoint{accessPoint},
			},
			connector: connector,
		},
		&keaDaemon{
			daemon: daemon{
				Name:         daemonname.DHCPv4,
				AccessPoints: []AccessPoint{accessPoint},
			},
			connector: connector,
		},
		&keaDaemon{
			daemon: daemon{
				Name:         daemonname.DHCPv6,
				AccessPoints: []AccessPoint{accessPoint},
			},
			connector: connector,
		},
	}}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	agentManager := NewMockAgentManager(ctrl)
	// We should have three log files recorded from the returned configurations.
	// One from CA, one from DHCPv4 and one from DHCPv6.
	agentManager.EXPECT().allowLog(gomock.Any()).Times(3)
	agentManager.EXPECT().allowLeaseTracking().Return(false, 0).AnyTimes()

	monitor.refreshDaemons(t.Context(), agentManager)

	require.NoError(t, err)
	require.False(t, gock.HasUnmatchedRequest())
}

// This test verifies that an error is returned when the agent is unable to
// fetch the Kea config.
func TestKeaAllowedLogsConfigUnavailable(t *testing.T) {
	defer gock.Off()

	gock.New("https://localhost:45634").
		MatchHeader("Content-Type", "application/json").
		JSON(map[string]interface{}{"command": "config-get"}).
		Post("/").
		Reply(200).
		JSON([]map[string]any{{
			"result": keactrl.ResponseError,
		}})

	accessPoint := AccessPoint{Type: AccessPointControl, Address: "localhost", Port: 45634, Protocol: protocoltype.HTTPS}
	daemon := &keaDaemon{
		daemon: daemon{
			Name:         daemonname.CA,
			AccessPoints: []AccessPoint{accessPoint},
		},
		connector: newKeaConnector(accessPoint, HTTPClientConfig{Interceptor: gock.InterceptClient}),
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	agentManager := NewMockAgentManager(ctrl)

	err := daemon.RefreshState(t.Context(), agentManager)
	require.Error(t, err)
	require.False(t, gock.HasUnmatchedRequest())
}

// Test that cleaning up the daemon doesn't panic.
func TestKeaDaemonCleanup(t *testing.T) {
	daemon := &keaDaemon{}
	require.NotPanics(t, func() {
		daemon.Cleanup()
	})
}

// Test that the client credentials are retrieved properly.
func TestReadClientCredentials(t *testing.T) {
	t.Run("Nil authentication", func(t *testing.T) {
		// Arrange
		var authentication *keaconfig.Authentication

		// Act & Assert
		require.Panics(t, func() {
			_, _ = readClientCredentials(authentication)
		})
	})

	t.Run("No clients", func(t *testing.T) {
		// Arrange
		authentication := &keaconfig.Authentication{
			Clients: nil,
		}

		// Act
		clients, err := readClientCredentials(authentication)

		// Assert
		require.NoError(t, err)
		require.Empty(t, clients)
	})

	t.Run("User and password", func(t *testing.T) {
		// Arrange
		authentication := &keaconfig.Authentication{
			Clients: []keaconfig.ClientCredentials{
				{
					User:     storkutil.Ptr("user"),
					Password: storkutil.Ptr("password"),
				},
			},
		}

		// Act
		clients, err := readClientCredentials(authentication)

		// Assert
		require.NoError(t, err)
		require.Len(t, clients, 1)
		require.Equal(t, "user", clients[0].User)
		require.Equal(t, "password", clients[0].Password)
	})

	t.Run("No client properties are set", func(t *testing.T) {
		// Arrange
		authentication := &keaconfig.Authentication{
			Clients: []keaconfig.ClientCredentials{
				{},
			},
		}

		// Act
		clients, err := readClientCredentials(authentication)

		// Assert
		require.Error(t, err)
		require.Empty(t, clients)
	})

	t.Run("User file is set only", func(t *testing.T) {
		// Arrange
		sb := testutil.NewSandbox()
		defer sb.Close()
		dir, _ := sb.JoinDir("empty")
		authentication := &keaconfig.Authentication{
			Clients: []keaconfig.ClientCredentials{
				{
					UserFile:     storkutil.Ptr(path.Join(dir, "user")),
					PasswordFile: nil,
				},
			},
		}

		// Act
		clients, err := readClientCredentials(authentication)

		// Assert
		require.ErrorContains(t, err, "could not read the user file")
		require.Empty(t, clients)
	})

	t.Run("Non-existing user file and password file", func(t *testing.T) {
		// Arrange
		sb := testutil.NewSandbox()
		defer sb.Close()
		passwordFile, _ := sb.Join("password")

		authentication := &keaconfig.Authentication{
			Clients: []keaconfig.ClientCredentials{
				{
					UserFile:     storkutil.Ptr(path.Join(sb.BasePath, "user")),
					PasswordFile: storkutil.Ptr(passwordFile),
				},
			},
		}

		// Act
		clients, err := readClientCredentials(authentication)

		// Assert
		require.ErrorContains(t, err, "could not read the user file")
		require.Empty(t, clients)
	})

	t.Run("User file and password file - non-existing user file", func(t *testing.T) {
		// Arrange
		sb := testutil.NewSandbox()
		defer sb.Close()
		dir, _ := sb.JoinDir("test")
		authentication := &keaconfig.Authentication{
			Clients: []keaconfig.ClientCredentials{
				{
					UserFile:     storkutil.Ptr(path.Join(dir, "user")),
					PasswordFile: storkutil.Ptr(path.Join(dir, "password")),
				},
			},
		}

		// Act
		clients, err := readClientCredentials(authentication)

		// Assert
		require.ErrorContains(t, err, "could not read the user file")
		require.Empty(t, clients)
	})

	t.Run("User file and password file - non-existing password file", func(t *testing.T) {
		// Arrange
		sb := testutil.NewSandbox()
		defer sb.Close()

		userFile, _ := sb.Join("user")

		authentication := &keaconfig.Authentication{
			Clients: []keaconfig.ClientCredentials{
				{
					UserFile:     storkutil.Ptr(userFile),
					PasswordFile: storkutil.Ptr(path.Join(sb.BasePath, "password")),
				},
			},
		}

		// Act
		clients, err := readClientCredentials(authentication)

		// Assert
		require.ErrorContains(t, err, "could not read the password file")
		require.Empty(t, clients)
	})

	t.Run("User file and password file with default directory", func(t *testing.T) {
		// Arrange
		sb := testutil.NewSandbox()
		defer sb.Close()

		userFile, _ := sb.Write("user", "foo")
		passwordFile, _ := sb.Write("password", "bar")

		authentication := &keaconfig.Authentication{
			Directory: nil,
			Clients: []keaconfig.ClientCredentials{
				{
					UserFile:     storkutil.Ptr(userFile),
					PasswordFile: storkutil.Ptr(passwordFile),
				},
			},
		}

		// Act
		clients, err := readClientCredentials(authentication)

		// Assert
		require.NoError(t, err)
		require.Len(t, clients, 1)
		require.Equal(t, "foo", clients[0].User)
		require.Equal(t, "bar", clients[0].Password)
	})

	t.Run("User file and password file with custom directory", func(t *testing.T) {
		// Arrange
		sb := testutil.NewSandbox()
		defer sb.Close()

		_, _ = sb.Write("user", "foo")
		_, _ = sb.Write("password", "bar")

		authentication := &keaconfig.Authentication{
			Directory: storkutil.Ptr(sb.BasePath),
			Clients: []keaconfig.ClientCredentials{
				{
					UserFile:     storkutil.Ptr("user"),
					PasswordFile: storkutil.Ptr("password"),
				},
			},
		}

		// Act
		clients, err := readClientCredentials(authentication)

		// Assert
		require.NoError(t, err)
		require.Len(t, clients, 1)
		require.Equal(t, "foo", clients[0].User)
		require.Equal(t, "bar", clients[0].Password)
	})

	t.Run("Password file only - non-existing file", func(t *testing.T) {
		// Arrange
		sb := testutil.NewSandbox()
		defer sb.Close()

		authentication := &keaconfig.Authentication{
			Clients: []keaconfig.ClientCredentials{
				{
					UserFile:     nil,
					PasswordFile: storkutil.Ptr(path.Join(sb.BasePath, "password")),
				},
			},
		}

		// Act
		clients, err := readClientCredentials(authentication)

		// Assert
		require.ErrorContains(t, err, "could not read the password file")
		require.Empty(t, clients)
	})

	t.Run("Password file only - default directory", func(t *testing.T) {
		// Arrange
		sb := testutil.NewSandbox()
		defer sb.Close()

		passwordFile, _ := sb.Write("password", "foo:bar")

		authentication := &keaconfig.Authentication{
			Directory: nil,
			Clients: []keaconfig.ClientCredentials{
				{
					UserFile:     nil,
					PasswordFile: storkutil.Ptr(passwordFile),
				},
			},
		}

		// Act
		clients, err := readClientCredentials(authentication)

		// Assert
		require.NoError(t, err)
		require.Len(t, clients, 1)
		require.Equal(t, "foo", clients[0].User)
		require.Equal(t, "bar", clients[0].Password)
	})

	t.Run("Password file only - invalid content", func(t *testing.T) {
		// Arrange
		sb := testutil.NewSandbox()
		defer sb.Close()

		passwordFile, _ := sb.Write("password", "foo-bar")

		authentication := &keaconfig.Authentication{
			Clients: []keaconfig.ClientCredentials{
				{
					UserFile:     nil,
					PasswordFile: storkutil.Ptr(passwordFile),
				},
			},
		}

		// Act
		clients, err := readClientCredentials(authentication)

		// Assert
		require.ErrorContains(t, err, "invalid format of the password file")
		require.Empty(t, clients)
	})

	t.Run("Password file only - custom directory", func(t *testing.T) {
		// Arrange
		sb := testutil.NewSandbox()
		defer sb.Close()

		_, _ = sb.Write("password", "foo:bar")

		authentication := &keaconfig.Authentication{
			Directory: storkutil.Ptr(sb.BasePath),
			Clients: []keaconfig.ClientCredentials{
				{
					UserFile:     nil,
					PasswordFile: storkutil.Ptr("password"),
				},
			},
		}

		// Act
		clients, err := readClientCredentials(authentication)

		// Assert
		require.NoError(t, err)
		require.Len(t, clients, 1)
		require.Equal(t, "foo", clients[0].User)
		require.Equal(t, "bar", clients[0].Password)
	})

	t.Run("User string and password file", func(t *testing.T) {
		// Arrange
		sb := testutil.NewSandbox()
		defer sb.Close()

		passwordFile, _ := sb.Write("password", "bar")

		authentication := &keaconfig.Authentication{
			Clients: []keaconfig.ClientCredentials{
				{
					User:         storkutil.Ptr("foo"),
					PasswordFile: storkutil.Ptr(passwordFile),
				},
			},
		}

		// Act
		clients, err := readClientCredentials(authentication)

		// Assert
		require.NoError(t, err)
		require.Len(t, clients, 1)
		require.Equal(t, "foo", clients[0].User)
		require.Equal(t, "bar", clients[0].Password)
	})

	t.Run("User file and password string", func(t *testing.T) {
		// Arrange
		sb := testutil.NewSandbox()
		defer sb.Close()

		userFile, _ := sb.Write("user", "foo")

		authentication := &keaconfig.Authentication{
			Clients: []keaconfig.ClientCredentials{
				{
					UserFile: storkutil.Ptr(userFile),
					Password: storkutil.Ptr("bar"),
				},
			},
		}

		// Act
		clients, err := readClientCredentials(authentication)

		// Assert
		require.NoError(t, err)
		require.Len(t, clients, 1)
		require.Equal(t, "foo", clients[0].User)
		require.Equal(t, "bar", clients[0].Password)
	})

	t.Run("All methods at once", func(t *testing.T) {
		// Arrange
		sb := testutil.NewSandbox()
		defer sb.Close()

		userFile, _ := sb.Write("user", "foo")
		passwordFile, _ := sb.Write("password", "bar")
		singlePasswordFile, _ := sb.Write("password-single", "baz:boz")

		authentication := &keaconfig.Authentication{
			Clients: []keaconfig.ClientCredentials{
				{
					// User and password as strings.
					User:     storkutil.Ptr("bim"),
					Password: storkutil.Ptr("bom"),
				},
				{
					// User and password as files.
					UserFile:     storkutil.Ptr(userFile),
					PasswordFile: storkutil.Ptr(passwordFile),
				},
				{
					// User and password in a single file.
					UserFile:     nil,
					PasswordFile: storkutil.Ptr(singlePasswordFile),
				},
				{
					// User as a string and password as a file.
					User:         storkutil.Ptr("ding"),
					PasswordFile: storkutil.Ptr(passwordFile),
				},
				{
					// User as a file and password as a string.
					UserFile: storkutil.Ptr(userFile),
					Password: storkutil.Ptr("dong"),
				},
			},
		}

		// Act
		clients, err := readClientCredentials(authentication)

		// Assert
		require.NoError(t, err)
		require.Len(t, clients, 5)

		require.Equal(t, "bim", clients[0].User)
		require.Equal(t, "bom", clients[0].Password)

		require.Equal(t, "foo", clients[1].User)
		require.Equal(t, "bar", clients[1].Password)

		require.Equal(t, "baz", clients[2].User)
		require.Equal(t, "boz", clients[2].Password)

		require.Equal(t, "ding", clients[3].User)
		require.Equal(t, "bar", clients[3].Password)

		require.Equal(t, "foo", clients[4].User)
		require.Equal(t, "dong", clients[4].Password)
	})
}

// Test that the Kea CA prior to 3.0 is detected. It should detect also all
// daemons behind it.
func TestDetectKeaCAPrior3_0(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	sb := testutil.NewSandbox()
	defer sb.Close()

	// Create a configuration file for Kea CA.
	configPath, _ := sb.Write("kea-ctrl-agent.conf", `{
        "Control-agent": {
		    "http-host": "localhost",
    		"http-port": 45634,
            "control-sockets": {
                "dhcp4": {
                    "socket-type": "unix",
                    "socket-name": "/path/to/the/unix/socket-v4"
                },
                "dhcp6": {
                    "socket-type": "unix",
                    "socket-name": "/path/to/the/unix/socket-v6"
                },
                "d2": {
                    "socket-type": "unix",
                    "socket-name": "/path/to/the/unix/socket-d2"
                }
            }
        }
    }`)
	exePath, _ := sb.Join("kea-ctrl-agent")

	// The HTTP response mock.
	defer gock.Off()
	gock.New("http://localhost:45634").
		JSON(map[string]any{"command": "version-get", "service": []string{"d2"}}).
		Post("/").
		Reply(200).
		JSON([]map[string]int{{"result": int(keactrl.ResponseSuccess)}})

	gock.New("http://localhost:45634").
		JSON(map[string]any{"command": "version-get", "service": []string{"dhcp4"}}).
		Post("/").
		Reply(200).
		JSON([]map[string]int{{"result": int(keactrl.ResponseSuccess)}})

	gock.New("http://localhost:45634").
		JSON(map[string]any{"command": "version-get", "service": []string{"dhcp6"}}).
		Post("/").
		Reply(200).
		JSON([]map[string]int{{"result": int(keactrl.ResponseSuccess)}})

	httpConfig := HTTPClientConfig{Timeout: 42 * time.Minute, Interceptor: gock.InterceptClient}

	// Kea process mock.
	process := NewMockSupportedProcess(ctrl)
	process.EXPECT().getName().Return("kea-ctrl-agent", nil)
	process.EXPECT().getDaemonName().Return(daemonname.CA)
	process.EXPECT().getCmdline().Return(
		fmt.Sprintf("%s -c %s", exePath, configPath),
		nil,
	)
	process.EXPECT().getCwd().Return(sb.BasePath, nil)

	// System calls mock.
	commander := NewMockCommandExecutor(ctrl)
	commander.EXPECT().Output(exePath, "-v").Return([]byte("2.3.0\n"), nil)

	monitor := newMonitor("", "", httpConfig)
	monitor.commander = commander

	// Act
	daemons, err := monitor.detectKeaDaemons(t.Context(), process)

	// Assert
	require.False(t, gock.HasUnmatchedRequest())
	require.NoError(t, err)

	// It detects all daemons - CA and three daemons behind it.
	require.Len(t, daemons, 4)
	require.Equal(t, daemonname.CA, daemons[0].GetName())
	require.Equal(t, daemonname.D2, daemons[1].GetName())
	require.Equal(t, daemonname.DHCPv4, daemons[2].GetName())
	require.Equal(t, daemonname.DHCPv6, daemons[3].GetName())

	// All daemons have the same access point - the HTTP of the CA.
	accessPoints := daemons[0].GetAccessPoints()
	require.Equal(t, accessPoints, daemons[1].GetAccessPoints())
	require.Equal(t, accessPoints, daemons[2].GetAccessPoints())
	require.Equal(t, accessPoints, daemons[3].GetAccessPoints())
	require.Len(t, accessPoints, 1)
	accessPoint := accessPoints[0]
	require.Equal(t, AccessPointControl, accessPoint.Type)
	require.Equal(t, "localhost", accessPoint.Address)
	require.EqualValues(t, 45634, accessPoint.Port)
	require.Equal(t, protocoltype.HTTP, accessPoint.Protocol)
}

// Test that the Kea CA post 3.0 is detected. The other daemons behind it should
// not be detected as they are detected separately.
func TestDetectKeaCAPost3_0(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	sb := testutil.NewSandbox()
	defer sb.Close()

	// Create a configuration file for Kea CA.
	configPath, _ := sb.Write("kea-ctrl-agent.conf", `{
        "Control-agent": {
		    "http-host": "localhost",
    		"http-port": 45634,
            "control-sockets": {
                "dhcp4": {
                    "socket-type": "unix",
                    "socket-name": "/path/to/the/unix/socket-v4"
                },
                "dhcp6": {
                    "socket-type": "unix",
                    "socket-name": "/path/to/the/unix/socket-v6"
                },
                "d2": {
                    "socket-type": "unix",
                    "socket-name": "/path/to/the/unix/socket-d2"
                }
            }
        }
    }`)
	exePath, _ := sb.Join("kea-ctrl-agent")

	httpConfig := HTTPClientConfig{}

	// Kea process mock.
	process := NewMockSupportedProcess(ctrl)
	process.EXPECT().getName().Return("kea-ctrl-agent", nil)
	process.EXPECT().getDaemonName().Return(daemonname.CA)
	process.EXPECT().getCmdline().Return(
		fmt.Sprintf("%s -c %s", exePath, configPath),
		nil,
	)
	process.EXPECT().getCwd().Return(sb.BasePath, nil)

	// System calls mock.
	commander := NewMockCommandExecutor(ctrl)
	commander.EXPECT().Output(exePath, "-v").Return([]byte("3.0.0\n"), nil)

	monitor := newMonitor("", "", httpConfig)
	monitor.commander = commander

	// Act
	daemons, err := monitor.detectKeaDaemons(t.Context(), process)

	// Assert
	require.False(t, gock.HasUnmatchedRequest())
	require.NoError(t, err)

	// It detects only CA daemon.
	require.Len(t, daemons, 1)
	require.Equal(t, daemonname.CA, daemons[0].GetName())

	accessPoints := daemons[0].GetAccessPoints()
	require.Len(t, accessPoints, 1)
	accessPoint := accessPoints[0]
	require.Equal(t, AccessPointControl, accessPoint.Type)
	require.Equal(t, "localhost", accessPoint.Address)
	require.EqualValues(t, 45634, accessPoint.Port)
	require.Equal(t, protocoltype.HTTP, accessPoint.Protocol)
	require.Empty(t, accessPoint.Key)
}

// Test that the Kea DHCP prior to 3.0 is not detected.
func TestDetectKeaDHCPPrior3_0(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	sb := testutil.NewSandbox()
	defer sb.Close()

	configPath, _ := sb.Join("kea-dhcp4.conf")
	exePath, _ := sb.Join("kea-dhcp4")

	httpConfig := HTTPClientConfig{Timeout: 42 * time.Minute}

	// Kea DHCP process mock.
	process := NewMockSupportedProcess(ctrl)
	process.EXPECT().getName().Return("kea-dhcp4", nil)
	process.EXPECT().getDaemonName().Return(daemonname.DHCPv4)
	process.EXPECT().getCmdline().Return(
		fmt.Sprintf("%s -c %s", exePath, configPath),
		nil,
	)
	process.EXPECT().getCwd().Return(sb.BasePath, nil)

	// System calls mock.
	commander := NewMockCommandExecutor(ctrl)
	commander.EXPECT().Output(exePath, "-v").Return([]byte("2.3.0\n"), nil)

	monitor := newMonitor("", "", httpConfig)
	monitor.commander = commander

	// Act
	daemons, err := monitor.detectKeaDaemons(t.Context(), process)

	// Assert
	require.False(t, gock.HasUnmatchedRequest())
	require.NoError(t, err)
	// It detects no daemons.
	require.Len(t, daemons, 0)
}

// Test that the Kea DHCP post 3.0 listening on socket is detected.
func TestDetectKeaDHCPOnSocketPost3_0(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	sb := testutil.NewSandbox()
	defer sb.Close()

	// Create a configuration file for Kea.
	configPath, _ := sb.Write("kea-dhcp4.conf", `{
        "Dhcp4": { "control-socket": {
			"socket-type": "unix",
			"socket-name": "/var/run/kea/kea4-ctrl-socket"
		}}
    }`)
	exePath, _ := sb.Join("kea-dhcp4")

	httpConfig := HTTPClientConfig{}

	// Kea process mock.
	process := NewMockSupportedProcess(ctrl)
	process.EXPECT().getName().Return("kea-dhcp4", nil)
	process.EXPECT().getDaemonName().Return(daemonname.DHCPv4)
	process.EXPECT().getCmdline().Return(
		fmt.Sprintf("%s -c %s", exePath, configPath),
		nil,
	)
	process.EXPECT().getCwd().Return(sb.BasePath, nil)

	// System calls mock.
	commander := NewMockCommandExecutor(ctrl)
	commander.EXPECT().Output(exePath, "-v").Return([]byte("3.0.0\n"), nil)

	monitor := newMonitor("", "", httpConfig)
	monitor.commander = commander

	// Act
	daemons, err := monitor.detectKeaDaemons(t.Context(), process)

	// Assert
	require.False(t, gock.HasUnmatchedRequest())
	require.NoError(t, err)

	require.Len(t, daemons, 1)
	require.Equal(t, daemonname.DHCPv4, daemons[0].GetName())

	accessPoints := daemons[0].GetAccessPoints()
	require.Len(t, accessPoints, 1)
	accessPoint := accessPoints[0]
	require.Equal(t, AccessPointControl, accessPoint.Type)
	require.Equal(t, "/var/run/kea/kea4-ctrl-socket", accessPoint.Address)
	require.Zero(t, accessPoint.Port)
	require.Equal(t, protocoltype.Socket, accessPoint.Protocol)
}

// Test that the Kea DHCP listening on socket configured with name only is detected.
func TestDetectKeaDHCPOnSocketNameOnly(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	sb := testutil.NewSandbox()
	defer sb.Close()

	// Create a configuration file for Kea.
	configPath, _ := sb.Write("kea-dhcp4.conf", `{
      "Dhcp4": {
				"control-socket": {
					"socket-type": "unix",
					"socket-name": "kea4-ctrl-socket"
				}
			}
    }`)
	exePath, _ := sb.Join("bin/kea-dhcp4")
	socketPath, _ := sb.Join("var/run/kea/kea4-ctrl-socket")

	httpConfig := HTTPClientConfig{}

	// Kea process mock.
	process := NewMockSupportedProcess(ctrl)
	process.EXPECT().getName().Return("kea-dhcp4", nil)
	process.EXPECT().getDaemonName().Return(daemonname.DHCPv4)
	process.EXPECT().getCmdline().Return(
		fmt.Sprintf("%s -c %s", exePath, configPath),
		nil,
	)
	process.EXPECT().getCwd().Return(sb.BasePath, nil)
	process.EXPECT().getExe().Return(exePath, nil)

	// System calls mock.
	commander := NewMockCommandExecutor(ctrl)
	commander.EXPECT().Output(exePath, "-v").Return([]byte("3.0.0\n"), nil)

	monitor := newMonitor("", "", httpConfig)
	monitor.commander = commander

	// Act
	daemons, err := monitor.detectKeaDaemons(t.Context(), process)

	// Assert
	require.False(t, gock.HasUnmatchedRequest())
	require.NoError(t, err)

	require.Len(t, daemons, 1)
	require.Equal(t, daemonname.DHCPv4, daemons[0].GetName())

	accessPoints := daemons[0].GetAccessPoints()
	require.Len(t, accessPoints, 1)
	accessPoint := accessPoints[0]
	require.Equal(t, AccessPointControl, accessPoint.Type)
	require.Equal(t, socketPath, accessPoint.Address)
	require.Zero(t, accessPoint.Port)
	require.Equal(t, protocoltype.Socket, accessPoint.Protocol)
}

// Test that the Kea DHCP post 3.0 listening on HTTP is detected.
func TestDetectKeaDHCPOnHTTPPost3_0(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	sb := testutil.NewSandbox()
	defer sb.Close()

	// Create a configuration file for Kea.
	configPath, _ := sb.Write("kea-dhcp4.conf", `{
        "Dhcp4": { "control-socket": {
			"socket-type": "https",
            "socket-address": "10.20.30.40",
            "socket-port": 8004
		}}
    }`)
	exePath, _ := sb.Join("kea-dhcp4")

	httpConfig := HTTPClientConfig{}

	// Kea process mock.
	process := NewMockSupportedProcess(ctrl)
	process.EXPECT().getName().Return("kea-dhcp4", nil)
	process.EXPECT().getDaemonName().Return(daemonname.DHCPv4)
	process.EXPECT().getCmdline().Return(
		fmt.Sprintf("%s -c %s", exePath, configPath),
		nil,
	)
	process.EXPECT().getCwd().Return(sb.BasePath, nil)

	// System calls mock.
	commander := NewMockCommandExecutor(ctrl)
	commander.EXPECT().Output(exePath, "-v").Return([]byte("3.0.0\n"), nil)

	monitor := newMonitor("", "", httpConfig)
	monitor.commander = commander

	// Act
	daemons, err := monitor.detectKeaDaemons(t.Context(), process)

	// Assert
	require.False(t, gock.HasUnmatchedRequest())
	require.NoError(t, err)

	require.Len(t, daemons, 1)
	require.Equal(t, daemonname.DHCPv4, daemons[0].GetName())

	accessPoints := daemons[0].GetAccessPoints()
	require.Len(t, accessPoints, 1)
	accessPoint := accessPoints[0]
	require.Equal(t, AccessPointControl, accessPoint.Type)
	require.Equal(t, "10.20.30.40", accessPoint.Address)
	require.EqualValues(t, 8004, accessPoint.Port)
	require.Equal(t, protocoltype.HTTPS, accessPoint.Protocol)
	require.Empty(t, accessPoint.Key)
}

// Test that the Kea CA with Basic Auth credentials is detected and the used
// username is included in the access point.
func TestDetectKeaCAWithCredentials(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	sb := testutil.NewSandbox()
	defer sb.Close()

	// Create a configuration file for Kea CA.
	configPath, _ := sb.Write("kea-ctrl-agent.conf", `{
        "Control-agent": {
		    "http-host": "localhost",
    		"http-port": 45634,
            "control-sockets": {
                "dhcp4": {
                    "socket-type": "unix",
                    "socket-name": "/path/to/the/unix/socket-v4"
                },
                "dhcp6": {
                    "socket-type": "unix",
                    "socket-name": "/path/to/the/unix/socket-v6"
                },
                "d2": {
                    "socket-type": "unix",
                    "socket-name": "/path/to/the/unix/socket-d2"
                }
            },
			"authentication": {
				"type": "basic",
				"realm": "kea-control-agent",
				"clients": [
					{
						"user": "user",
						"password": "password"
					}
				]
			}
        }
    }`)
	exePath, _ := sb.Join("kea-ctrl-agent")

	httpConfig := HTTPClientConfig{}

	// Kea process mock.
	process := NewMockSupportedProcess(ctrl)
	process.EXPECT().getName().Return("kea-ctrl-agent", nil)
	process.EXPECT().getDaemonName().Return(daemonname.CA)
	process.EXPECT().getCmdline().Return(
		fmt.Sprintf("%s -c %s", exePath, configPath),
		nil,
	)
	process.EXPECT().getCwd().Return(sb.BasePath, nil)

	// System calls mock.
	commander := NewMockCommandExecutor(ctrl)
	commander.EXPECT().Output(exePath, "-v").Return([]byte("3.0.0\n"), nil)

	monitor := newMonitor("", "", httpConfig)
	monitor.commander = commander

	// Act
	daemons, err := monitor.detectKeaDaemons(t.Context(), process)

	// Assert
	require.False(t, gock.HasUnmatchedRequest())
	require.NoError(t, err)

	require.Len(t, daemons, 1)
	require.Equal(t, daemonname.CA, daemons[0].GetName())

	accessPoints := daemons[0].GetAccessPoints()
	require.Len(t, accessPoints, 1)
	accessPoint := accessPoints[0]
	require.Equal(t, AccessPointControl, accessPoint.Type)
	require.Equal(t, "localhost", accessPoint.Address)
	require.EqualValues(t, 45634, accessPoint.Port)
	require.Equal(t, protocoltype.HTTP, accessPoint.Protocol)
	require.Equal(t, "user", accessPoint.Key)
}

// Test that the Kea DHCP with Basic Auth credentials is detected and the used
// username is included in the access point.
func TestDetectKeaDHCPWithCredentials(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	sb := testutil.NewSandbox()
	defer sb.Close()

	// Create a configuration file for Kea.
	configPath, _ := sb.Write("kea-dhcp4.conf", `{
        "Dhcp4": {
			"control-socket": {
				"socket-type": "https",
				"socket-address": "10.20.30.40",
				"socket-port": 8004
		}}
    }`)
	exePath, _ := sb.Join("kea-dhcp4")

	httpConfig := HTTPClientConfig{}

	// Kea process mock.
	process := NewMockSupportedProcess(ctrl)
	process.EXPECT().getName().Return("kea-dhcp4", nil)
	process.EXPECT().getDaemonName().Return(daemonname.DHCPv4)
	process.EXPECT().getCmdline().Return(
		fmt.Sprintf("%s -c %s", exePath, configPath),
		nil,
	)
	process.EXPECT().getCwd().Return(sb.BasePath, nil)

	// System calls mock.
	commander := NewMockCommandExecutor(ctrl)
	commander.EXPECT().Output(exePath, "-v").Return([]byte("3.0.0\n"), nil)

	monitor := newMonitor("", "", httpConfig)
	monitor.commander = commander

	// Act
	daemons, err := monitor.detectKeaDaemons(t.Context(), process)

	// Assert
	require.False(t, gock.HasUnmatchedRequest())
	require.NoError(t, err)

	require.Len(t, daemons, 1)
	require.Equal(t, daemonname.DHCPv4, daemons[0].GetName())

	accessPoints := daemons[0].GetAccessPoints()
	require.Len(t, accessPoints, 1)
	accessPoint := accessPoints[0]
	require.Equal(t, AccessPointControl, accessPoint.Type)
	require.Equal(t, "10.20.30.40", accessPoint.Address)
	require.EqualValues(t, 8004, accessPoint.Port)
	require.Equal(t, protocoltype.HTTPS, accessPoint.Protocol)
}

// Test that the error is returned when the process name cannot be retrieved.
func TestDetectKeaProcessNameUnavailable(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	httpConfig := HTTPClientConfig{}

	process := NewMockSupportedProcess(ctrl)
	process.EXPECT().getName().Return("", errors.New("unable to get the process name"))

	commander := NewMockCommandExecutor(ctrl)

	monitor := newMonitor("", "", httpConfig)
	monitor.commander = commander

	// Act
	daemons, err := monitor.detectKeaDaemons(t.Context(), process)

	// Assert
	require.False(t, gock.HasUnmatchedRequest())
	require.ErrorContains(t, err, "unable to get the process name")
	require.Empty(t, daemons)
}

// Test that the error is returned when the daemon name cannot be determined.
func TestDetectKeaDaemonNameUnavailable(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	httpConfig := HTTPClientConfig{}

	// Kea process mock.
	process := NewMockSupportedProcess(ctrl)
	process.EXPECT().getName().Return("kea-ctrl-agent", nil)
	process.EXPECT().getDaemonName().Return(daemonname.Name(""))

	commander := NewMockCommandExecutor(ctrl)

	monitor := newMonitor("", "", httpConfig)
	monitor.commander = commander

	// Act
	daemons, err := monitor.detectKeaDaemons(t.Context(), process)

	// Assert
	require.False(t, gock.HasUnmatchedRequest())
	require.ErrorContains(t, err, "unsupported Kea process: kea-ctrl-agent")
	require.Empty(t, daemons)
}

// Test that the error is returned when the command line cannot be determined.
func TestDetectKeaCommandLineUnavailable(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	httpConfig := HTTPClientConfig{}

	// Kea process mock.
	process := NewMockSupportedProcess(ctrl)
	process.EXPECT().getName().Return("kea-ctrl-agent", nil)
	process.EXPECT().getDaemonName().Return(daemonname.CA)
	process.EXPECT().getCmdline().Return(
		"",
		errors.New("unable to get the command line"),
	)

	commander := NewMockCommandExecutor(ctrl)

	monitor := newMonitor("", "", httpConfig)
	monitor.commander = commander

	// Act
	daemons, err := monitor.detectKeaDaemons(t.Context(), process)

	// Assert
	require.False(t, gock.HasUnmatchedRequest())
	require.ErrorContains(t, err, "unable to get the command line")
	require.Empty(t, daemons)
}

// Test that the error is returned when the current working directory cannot be
// determined.
func TestDetectKeaCwdUnavailable(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	httpConfig := HTTPClientConfig{}

	sb := testutil.NewSandbox()
	defer sb.Close()

	configPath, _ := sb.Join("kea-dhcp4.conf")
	configPath, _ = filepath.Rel(sb.BasePath, configPath)

	exePath, _ := sb.Join("kea-dhcp4")

	// Kea process mock.
	process := NewMockSupportedProcess(ctrl)
	process.EXPECT().getName().Return("kea-ctrl-agent", nil)
	process.EXPECT().getDaemonName().Return(daemonname.CA)
	process.EXPECT().getCmdline().Return(
		fmt.Sprintf("%s -c %s", exePath, configPath),
		nil,
	)
	process.EXPECT().getCwd().Return("", errors.New("unable to get the cwd"))

	commander := NewMockCommandExecutor(ctrl)

	monitor := newMonitor("", "", httpConfig)
	monitor.commander = commander

	// Act
	daemons, err := monitor.detectKeaDaemons(t.Context(), process)

	// Assert
	require.False(t, gock.HasUnmatchedRequest())
	// Error should contain the relative configuration path.
	require.ErrorContains(t, err, "-c kea-dhcp4.conf")
	require.Empty(t, daemons)
}

// Test that the Kea with default configuration path cannot be detected.
func TestDetectKeaWithDefaultConfigurationPath(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	sb := testutil.NewSandbox()
	defer sb.Close()

	// Create a configuration file for Kea.
	exePath, _ := sb.Join("kea-dhcp4")

	httpConfig := HTTPClientConfig{}

	// Kea process mock.
	process := NewMockSupportedProcess(ctrl)
	process.EXPECT().getName().Return("kea-dhcp4", nil)
	process.EXPECT().getDaemonName().Return(daemonname.DHCPv4)
	process.EXPECT().getCmdline().Return(
		// Config path is default.
		exePath,
		nil,
	)
	process.EXPECT().getCwd().Return(sb.BasePath, nil)

	commander := NewMockCommandExecutor(ctrl)

	monitor := newMonitor("", "", httpConfig)
	monitor.commander = commander

	// Act
	daemons, err := monitor.detectKeaDaemons(t.Context(), process)

	// Assert
	require.False(t, gock.HasUnmatchedRequest())
	require.ErrorContains(t, err, "problem parsing Kea command line")
	require.Empty(t, daemons)
}

// Test that the error is returned if the Kea version doesn't follow the
// semantic versioning.
func TestDetectKeaUnparsableVersion(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	sb := testutil.NewSandbox()
	defer sb.Close()

	configPath, _ := sb.Join("kea-ctrl-agent.conf")
	exePath, _ := sb.Join("kea-ctrl-agent")

	httpConfig := HTTPClientConfig{}

	// Kea process mock.
	process := NewMockSupportedProcess(ctrl)
	process.EXPECT().getName().Return("kea-ctrl-agent", nil)
	process.EXPECT().getDaemonName().Return(daemonname.CA)
	process.EXPECT().getCmdline().Return(
		fmt.Sprintf("%s -c %s", exePath, configPath),
		nil,
	)
	process.EXPECT().getCwd().Return(sb.BasePath, nil)

	// System calls mock.
	commander := NewMockCommandExecutor(ctrl)
	commander.EXPECT().Output(exePath, "-v").Return([]byte("git+dirty\n"), nil)

	monitor := newMonitor("", "", httpConfig)
	monitor.commander = commander

	// Act
	daemons, err := monitor.detectKeaDaemons(t.Context(), process)

	// Assert
	require.False(t, gock.HasUnmatchedRequest())
	require.ErrorContains(t, err, "cannot parse Kea version: git+dirty")
	require.Empty(t, daemons)
}

// Test that the Kea with relative configuration path is detected properly.
func TestDetectKeaWithRelativeConfigurationPath(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	sb := testutil.NewSandbox()
	defer sb.Close()

	// Create a configuration file for Kea CA.
	configPath, _ := sb.Write("kea-ctrl-agent.conf", `{
        "Control-agent": {
		    "http-host": "localhost",
    		"http-port": 45634,
            "control-sockets": {
                "dhcp4": {
                    "socket-type": "unix",
                    "socket-name": "/path/to/the/unix/socket-v4"
                },
                "dhcp6": {
                    "socket-type": "unix",
                    "socket-name": "/path/to/the/unix/socket-v6"
                },
                "d2": {
                    "socket-type": "unix",
                    "socket-name": "/path/to/the/unix/socket-d2"
                }
            }
        }
    }`)
	configPath, _ = filepath.Rel(sb.BasePath, configPath)
	exePath, _ := sb.Join("kea-ctrl-agent")

	httpConfig := HTTPClientConfig{}

	// Kea process mock.
	process := NewMockSupportedProcess(ctrl)
	process.EXPECT().getName().Return("kea-ctrl-agent", nil)
	process.EXPECT().getDaemonName().Return(daemonname.CA)
	process.EXPECT().getCmdline().Return(
		fmt.Sprintf("%s -c %s", exePath, configPath),
		nil,
	)
	process.EXPECT().getCwd().Return(sb.BasePath, nil)

	// System calls mock.
	commander := NewMockCommandExecutor(ctrl)
	commander.EXPECT().Output(exePath, "-v").Return([]byte("3.0.0\n"), nil)

	monitor := newMonitor("", "", httpConfig)
	monitor.commander = commander

	// Act
	daemons, err := monitor.detectKeaDaemons(t.Context(), process)

	// Assert
	require.False(t, gock.HasUnmatchedRequest())
	require.NoError(t, err)

	// It detects only CA daemon.
	require.Len(t, daemons, 1)
	require.Equal(t, daemonname.CA, daemons[0].GetName())

	accessPoints := daemons[0].GetAccessPoints()
	require.Len(t, accessPoints, 1)
	accessPoint := accessPoints[0]
	require.Equal(t, AccessPointControl, accessPoint.Type)
	require.Equal(t, "localhost", accessPoint.Address)
	require.EqualValues(t, 45634, accessPoint.Port)
	require.Equal(t, protocoltype.HTTP, accessPoint.Protocol)
	require.Empty(t, accessPoint.Key)
}

// Test that the no error is returned when communication with Kea daemons
// fails. It is only applicable for Kea prior to 3.0.
func TestDetectKeaCommunicationError(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	sb := testutil.NewSandbox()
	defer sb.Close()

	// Create a configuration file for Kea CA.
	configPath, _ := sb.Write("kea-ctrl-agent.conf", `{
        "Control-agent": {
		    "http-host": "localhost",
    		"http-port": 45634,
            "control-sockets": {
                "dhcp4": {
                    "socket-type": "unix",
                    "socket-name": "/path/to/the/unix/socket-v4"
                },
                "dhcp6": {
                    "socket-type": "unix",
                    "socket-name": "/path/to/the/unix/socket-v6"
                },
                "d2": {
                    "socket-type": "unix",
                    "socket-name": "/path/to/the/unix/socket-d2"
                }
            }
        }
    }`)
	exePath, _ := sb.Join("kea-ctrl-agent")

	// The HTTP response mock.
	defer gock.Off()
	gock.New("http://localhost:45634").
		JSON(map[string]any{"command": "version-get", "service": []string{"d2"}}).
		Post("/").
		Reply(200).
		JSON([]map[string]int{{"result": int(keactrl.ResponseSuccess)}})

	gock.New("http://localhost:45634").
		JSON(map[string]any{"command": "version-get", "service": []string{"dhcp4"}}).
		Post("/").
		Reply(400).
		JSON([]map[string]int{{"result": int(keactrl.ResponseError)}})

	gock.New("http://localhost:45634").
		JSON(map[string]any{"command": "version-get", "service": []string{"dhcp6"}}).
		Post("/").
		Reply(200).
		JSON([]map[string]int{{"result": int(keactrl.ResponseCommandUnsupported)}})

	httpConfig := HTTPClientConfig{Timeout: 42 * time.Minute, Interceptor: gock.InterceptClient}

	// Kea process mock.
	process := NewMockSupportedProcess(ctrl)
	process.EXPECT().getName().Return("kea-ctrl-agent", nil)
	process.EXPECT().getDaemonName().Return(daemonname.CA)
	process.EXPECT().getCmdline().Return(
		fmt.Sprintf("%s -c %s", exePath, configPath),
		nil,
	)
	process.EXPECT().getCwd().Return(sb.BasePath, nil)

	// System calls mock.
	commander := NewMockCommandExecutor(ctrl)
	commander.EXPECT().Output(exePath, "-v").Return([]byte("2.3.0\n"), nil)

	monitor := newMonitor("", "", httpConfig)
	monitor.commander = commander

	// Act
	daemons, err := monitor.detectKeaDaemons(t.Context(), process)

	// Assert
	require.False(t, gock.HasUnmatchedRequest())
	require.NoError(t, err)

	// It detects only active daemons - CA and D2.
	require.Len(t, daemons, 2)
	require.Equal(t, daemonname.CA, daemons[0].GetName())
	require.Equal(t, daemonname.D2, daemons[1].GetName())

	// All daemons have the same access point - the HTTP of the CA.
	accessPoints := daemons[0].GetAccessPoints()
	require.Equal(t, accessPoints, daemons[1].GetAccessPoints())
	require.Len(t, accessPoints, 1)
	accessPoint := accessPoints[0]
	require.Equal(t, AccessPointControl, accessPoint.Type)
	require.Equal(t, "localhost", accessPoint.Address)
	require.EqualValues(t, 45634, accessPoint.Port)
	require.Equal(t, protocoltype.HTTP, accessPoint.Protocol)
}

// Test sending data over Kea socket connector.
func TestKeaSocketConnectorSendPayload(t *testing.T) {
	// Arrange
	sb := testutil.NewSandbox()
	defer sb.Close()
	socketPath := path.Join(sb.BasePath, "kea-socket-test.sock")

	var listenConfig net.ListenConfig
	server, err := listenConfig.Listen(t.Context(), "unix", socketPath)
	require.NoError(t, err)
	defer server.Close()

	// Expected command and response.
	command := []byte(`{"command":"ping"}`)
	response := []byte(`{"result":0}`)

	var wgServer sync.WaitGroup
	wgServer.Go(func() {
		// Wait for client connection.
		conn, err := server.Accept()
		require.NoError(t, err)
		defer conn.Close()

		// Read command.
		buf := make([]byte, len(command))
		n, err := conn.Read(buf)
		require.NoError(t, err)
		require.Len(t, command, n)
		require.Equal(t, command, buf)

		// Write response.
		n, err = conn.Write(response)
		require.NoError(t, err)
		require.Len(t, response, n)
	})

	// Construct a socket connector.
	connector := newKeaConnector(AccessPoint{
		Type:     AccessPointControl,
		Address:  socketPath,
		Protocol: protocoltype.Socket,
	}, HTTPClientConfig{})

	// Act
	output, err := connector.sendPayload(t.Context(), command)

	// Assert
	require.NoError(t, err)
	require.Equal(t, response, output)
	// Wait for assertions in the server goroutine.
	wgServer.Wait()
}

// Test sending data over Kea HTTP connector.
func TestKeaHTTPConnectorSendPayload(t *testing.T) {
	// Arrange
	defer gock.Off()

	gock.New("http://localhost:45634").
		MatchHeader("Content-Type", "application/json").
		Post("/").
		Reply(200).
		// The server wraps the response into an array.
		JSON([]map[string]any{{"result": 0}})

	connector := newKeaConnector(AccessPoint{
		Type:     AccessPointControl,
		Address:  "localhost",
		Port:     45634,
		Protocol: protocoltype.HTTP,
	}, HTTPClientConfig{
		Interceptor: gock.InterceptClient,
	})

	command := []byte(`{"command":""ping"}`)

	// Act
	output, err := connector.sendPayload(t.Context(), command)

	// Assert
	require.NoError(t, err)
	// The response is unwrapped from the array.
	require.Equal(t, []byte(`{"result":0}`), output)
}

// Test that the multi-connector sends payload and fallback to the next
// connector if the first one fails.
func TestKeaMultiConnectorSendPayload(t *testing.T) {
	// Arrange
	// The UNIX socket connector will fail as there is no server listening.
	sb := testutil.NewSandbox()
	defer sb.Close()
	socketPath := path.Join(sb.BasePath, "kea-socket-test.sock")

	command := []byte(`{"command":"ping"}`)

	// The HTTP connector will succeed.
	defer gock.Off()

	gock.New("http://localhost:45634").
		MatchHeader("Content-Type", "application/json").
		Post("/").
		Reply(200).
		// The server wraps the response into an array.
		JSON([]map[string]any{{"result": 0}})

	// Act
	multiConnector := newMultiConnector(
		[]AccessPoint{
			{
				Type:     AccessPointControl,
				Address:  socketPath,
				Protocol: protocoltype.Socket,
			},
			{
				Type:     AccessPointControl,
				Address:  "localhost",
				Port:     45634,
				Protocol: protocoltype.HTTP,
			},
		},
		[]HTTPClientConfig{{}, {Interceptor: gock.InterceptClient}},
	)
	output, err := multiConnector.sendPayload(t.Context(), command)

	// Assert
	require.NoError(t, err)
	// The response is unwrapped from the array.
	require.Equal(t, []byte(`{"result":0}`), output)
}

// Test that the multi-connector fails to send payload if all connectors fail.
func TestKeaMultiConnectorSendPayloadFail(t *testing.T) {
	// Arrange
	// The UNIX socket connector will fail as there is no server listening.
	sb := testutil.NewSandbox()
	defer sb.Close()
	socketPath := path.Join(sb.BasePath, "kea-socket-test.sock")

	command := []byte(`{"command":"ping"}`)

	// The HTTP connector will fail.
	defer gock.Off()

	gock.New("http://localhost:45634").
		MatchHeader("Content-Type", "application/json").
		Post("/").
		Reply(500)

	// Act
	multiConnector := newMultiConnector(
		[]AccessPoint{
			{
				Type:     AccessPointControl,
				Address:  socketPath,
				Protocol: protocoltype.Socket,
			},
			{
				Type:     AccessPointControl,
				Address:  "localhost",
				Port:     45634,
				Protocol: protocoltype.HTTP,
			},
		},
		[]HTTPClientConfig{{}, {Interceptor: gock.InterceptClient}},
	)
	_, err := multiConnector.sendPayload(t.Context(), command)

	// Assert
	require.ErrorContains(t, err, "all connectors failed")
}

// Test that the multi-connector cannot send payload if there is no connectors.
func TestKeaMultiConnectorEmpty(t *testing.T) {
	// Arrange
	command := []byte(`{"command":"ping"}`)
	multiConnector := newMultiConnector([]AccessPoint{}, []HTTPClientConfig{})

	// Act
	_, err := multiConnector.sendPayload(t.Context(), command)

	// Assert
	require.ErrorContains(t, err, "no connectors available")
}

// Mock a Kea status-get command HTTP request/response sequence.
//
// Expect "status-get" for the given `service`, exactly `times` times.
// Reply with `response`.
func makeKeaCommandMock(service, response string, times int) error {
	responseJSON := make([]map[string]any, 1)
	err := json.Unmarshal([]byte(response), &responseJSON)
	if err != nil {
		return err
	}
	gock.New("http://localhost:45634").
		MatchHeader("Content-Type", "application/json").
		JSON(map[string]any{"command": "status-get", "service": []string{service}}).
		Times(times).
		Post("/").
		Reply(200).
		JSON(responseJSON)
	return nil
}

// Test to ensure that keaDaemon.ensureWatchingLeasefile works correctly in a variety of conditions (documented within).
func TestEnsureWatchingLeasefile(t *testing.T) {
	// Call the function once from an empty state to start watching a DHCPv4 lease
	// file. There should be no errors.
	t.Run("dhcpv4 start", func(t *testing.T) {
		sb := testutil.NewSandbox()
		defer sb.Close()

		config := keaconfig.Config{
			DHCPv4Config: &keaconfig.DHCPv4Config{
				CommonDHCPConfig: keaconfig.CommonDHCPConfig{
					LeaseDatabase: &keaconfig.Database{
						Type: "memfile",
					},
				},
			},
		}
		defer gock.Off()

		leasefile, err := sb.Join("kea-leases4")
		require.NoError(t, err)

		dhcpV4ResponsesJSON := fmt.Sprintf(`[
			{
				"result": 0,
				"arguments": {
					"csv-lease-file": "%s"
				}
			}
		]`, leasefile)
		err = makeKeaCommandMock("dhcp4", dhcpV4ResponsesJSON, 1)
		require.NoError(t, err)

		accessPoint := AccessPoint{
			Type:     AccessPointControl,
			Address:  "localhost",
			Port:     45634,
			Protocol: protocoltype.HTTP,
		}
		connector := newKeaConnector(
			accessPoint,
			HTTPClientConfig{Interceptor: gock.InterceptClient},
		)

		daemon := &keaDaemon{
			daemon: daemon{
				Name:         daemonname.DHCPv4,
				AccessPoints: []AccessPoint{accessPoint},
			},
			connector: connector,
		}

		err = daemon.ensureWatchingLeasefile(t.Context(), &config, 10)

		require.NoError(t, err)
		require.False(t, gock.HasUnmatchedRequest())
	})
	// Call the function once from an empty state to start watching a DHCPv6
	// lease file. There should be no errors.
	t.Run("dhcpv6 start", func(t *testing.T) {
		sb := testutil.NewSandbox()
		defer sb.Close()
		config := keaconfig.Config{
			DHCPv6Config: &keaconfig.DHCPv6Config{
				CommonDHCPConfig: keaconfig.CommonDHCPConfig{
					LeaseDatabase: &keaconfig.Database{
						Type: "memfile",
					},
				},
			},
		}
		defer gock.Off()

		leasefile, err := sb.Join("kea-leases6")
		require.NoError(t, err)

		dhcpV6ResponsesJSON := fmt.Sprintf(`[
			{
				"result": 0,
				"arguments": {
					"csv-lease-file": "%s"
				}
			}
		]`, leasefile)
		err = makeKeaCommandMock("dhcp6", dhcpV6ResponsesJSON, 1)
		require.NoError(t, err)

		accessPoint := AccessPoint{
			Type:     AccessPointControl,
			Address:  "localhost",
			Port:     45634,
			Protocol: protocoltype.HTTP,
		}
		connector := newKeaConnector(
			accessPoint,
			HTTPClientConfig{Interceptor: gock.InterceptClient},
		)

		daemon := &keaDaemon{
			daemon: daemon{
				Name:         daemonname.DHCPv6,
				AccessPoints: []AccessPoint{accessPoint},
			},
			connector: connector,
		}

		err = daemon.ensureWatchingLeasefile(t.Context(), &config, 10)

		require.NoError(t, err)
		require.False(t, gock.HasUnmatchedRequest())
	})
	// Call the function twice, once to start watching a file, and then again to switch to a different file. There should be no errors.
	t.Run("dhcp6 change file", func(t *testing.T) {
		sb := testutil.NewSandbox()
		defer sb.Close()

		config := keaconfig.Config{
			DHCPv6Config: &keaconfig.DHCPv6Config{
				CommonDHCPConfig: keaconfig.CommonDHCPConfig{
					LeaseDatabase: &keaconfig.Database{
						Type: "memfile",
					},
				},
			},
		}
		defer gock.Off()

		leasefile1, err := sb.Join("kea-leases6")
		require.NoError(t, err)
		leasefile2, err := sb.Join("kea-leases6")
		require.NoError(t, err)

		dhcpV6ResponsesJSONTemplate := `[
			{
				"result": 0,
				"arguments": {
					"csv-lease-file": "%s"
				}
			}
		]`
		dhcpV6ResponsesJSON1 := fmt.Sprintf(dhcpV6ResponsesJSONTemplate, leasefile1)
		dhcpV6ResponsesJSON2 := fmt.Sprintf(dhcpV6ResponsesJSONTemplate, leasefile2)
		err = makeKeaCommandMock("dhcp6", dhcpV6ResponsesJSON1, 1)
		require.NoError(t, err)
		err = makeKeaCommandMock("dhcp6", dhcpV6ResponsesJSON2, 1)
		require.NoError(t, err)

		accessPoint := AccessPoint{
			Type:     AccessPointControl,
			Address:  "localhost",
			Port:     45634,
			Protocol: protocoltype.HTTP,
		}
		connector := newKeaConnector(
			accessPoint,
			HTTPClientConfig{Interceptor: gock.InterceptClient},
		)

		daemon := &keaDaemon{
			daemon: daemon{
				Name:         daemonname.DHCPv6,
				AccessPoints: []AccessPoint{accessPoint},
			},
			connector: connector,
		}

		err = daemon.ensureWatchingLeasefile(t.Context(), &config, 10)
		require.NoError(t, err)
		err = daemon.ensureWatchingLeasefile(t.Context(), &config, 10)
		require.NoError(t, err)

		require.False(t, gock.HasUnmatchedRequest())
	})
	// Call the function twice, once to start watching a lease file and again to
	// stop watching it (because Kea was switched to a SQL database lease backend).
	// There should be no errors.
	t.Run("dhcp6 stop watching", func(t *testing.T) {
		sb := testutil.NewSandbox()
		defer sb.Close()

		config1 := keaconfig.Config{
			DHCPv6Config: &keaconfig.DHCPv6Config{
				CommonDHCPConfig: keaconfig.CommonDHCPConfig{
					LeaseDatabase: &keaconfig.Database{
						Type: "memfile",
					},
				},
			},
		}
		config2 := keaconfig.Config{
			DHCPv6Config: &keaconfig.DHCPv6Config{
				CommonDHCPConfig: keaconfig.CommonDHCPConfig{
					LeaseDatabase: &keaconfig.Database{
						Type: "mysql",
					},
				},
			},
		}
		defer gock.Off()

		leasefile, err := sb.Join("kea-leases6")
		require.NoError(t, err)

		dhcpV6ResponsesJSONTemplate := `[
			{
				"result": 0,
				"arguments": {
					"csv-lease-file": "%s"
				}
			}
		]`
		dhcpV6ResponsesJSON1 := fmt.Sprintf(dhcpV6ResponsesJSONTemplate, leasefile)
		err = makeKeaCommandMock("dhcp6", dhcpV6ResponsesJSON1, 2)
		require.NoError(t, err)

		accessPoint := AccessPoint{
			Type:     AccessPointControl,
			Address:  "localhost",
			Port:     45634,
			Protocol: protocoltype.HTTP,
		}
		connector := newKeaConnector(
			accessPoint,
			HTTPClientConfig{Interceptor: gock.InterceptClient},
		)

		daemon := &keaDaemon{
			daemon: daemon{
				Name:         daemonname.DHCPv6,
				AccessPoints: []AccessPoint{accessPoint},
			},
			connector: connector,
		}

		err = daemon.ensureWatchingLeasefile(t.Context(), &config1, 10)
		require.NoError(t, err)
		err = daemon.ensureWatchingLeasefile(t.Context(), &config2, 10)
		require.NoError(t, err)

		require.False(t, gock.HasUnmatchedRequest())
	})
	// Call the function once with memfile mode on, but persist set to false, to
	// ensure that nothing breaks when you turn off persistence.
	t.Run("dhcp6 persist false", func(t *testing.T) {
		sb := testutil.NewSandbox()
		defer sb.Close()

		nopersist := false
		config := keaconfig.Config{
			DHCPv6Config: &keaconfig.DHCPv6Config{
				CommonDHCPConfig: keaconfig.CommonDHCPConfig{
					LeaseDatabase: &keaconfig.Database{
						Type:    "memfile",
						Persist: &nopersist,
					},
				},
			},
		}
		defer gock.Off()

		leasefile, err := sb.Join("kea-leases6")
		require.NoError(t, err)

		dhcpV6ResponsesJSONTemplate := `[
			{
				"result": 0,
				"arguments": {
					"csv-lease-file": "%s"
				}
			}
		]`
		dhcpV6ResponsesJSON1 := fmt.Sprintf(dhcpV6ResponsesJSONTemplate, leasefile)
		err = makeKeaCommandMock("dhcp6", dhcpV6ResponsesJSON1, 1)
		require.NoError(t, err)

		accessPoint := AccessPoint{
			Type:     AccessPointControl,
			Address:  "localhost",
			Port:     45634,
			Protocol: protocoltype.HTTP,
		}
		connector := newKeaConnector(
			accessPoint,
			HTTPClientConfig{Interceptor: gock.InterceptClient},
		)

		daemon := &keaDaemon{
			daemon: daemon{
				Name:         daemonname.DHCPv6,
				AccessPoints: []AccessPoint{accessPoint},
			},
			connector: connector,
		}

		err = daemon.ensureWatchingLeasefile(t.Context(), &config, 10)
		require.NoError(t, err)

		require.False(t, gock.HasUnmatchedRequest())
	})
	// Call the function once with memfile mode on, but persist set to false, to
	// ensure that nothing breaks when you turn off persistence.
	t.Run("dhcp4 persist false", func(t *testing.T) {
		sb := testutil.NewSandbox()
		defer sb.Close()

		nopersist := false
		config := keaconfig.Config{
			DHCPv4Config: &keaconfig.DHCPv4Config{
				CommonDHCPConfig: keaconfig.CommonDHCPConfig{
					LeaseDatabase: &keaconfig.Database{
						Type:    "memfile",
						Persist: &nopersist,
					},
				},
			},
		}
		defer gock.Off()

		leasefile, err := sb.Join("kea-leases4")
		require.NoError(t, err)

		dhcpV4ResponsesJSONTemplate := `[
			{
				"result": 0,
				"arguments": {
					"csv-lease-file": "%s"
				}
			}
		]`
		dhcpV4ResponsesJSON1 := fmt.Sprintf(dhcpV4ResponsesJSONTemplate, leasefile)
		err = makeKeaCommandMock("dhcp4", dhcpV4ResponsesJSON1, 1)
		require.NoError(t, err)

		accessPoint := AccessPoint{
			Type:     AccessPointControl,
			Address:  "localhost",
			Port:     45634,
			Protocol: protocoltype.HTTP,
		}
		connector := newKeaConnector(
			accessPoint,
			HTTPClientConfig{Interceptor: gock.InterceptClient},
		)

		daemon := &keaDaemon{
			daemon: daemon{
				Name:         daemonname.DHCPv4,
				AccessPoints: []AccessPoint{accessPoint},
			},
			connector: connector,
		}

		err = daemon.ensureWatchingLeasefile(t.Context(), &config, 10)
		require.NoError(t, err)

		require.False(t, gock.HasUnmatchedRequest())
	})
	// Call the function once and feed it a 500 error from Kea, to ensure that
	// it handles the error gracefully and does not panic or otherwise take down the
	// agent.
	t.Run("fetch error", func(t *testing.T) {
		config := keaconfig.Config{
			DHCPv4Config: &keaconfig.DHCPv4Config{
				CommonDHCPConfig: keaconfig.CommonDHCPConfig{
					LeaseDatabase: &keaconfig.Database{
						Type: "memfile",
					},
				},
			},
		}
		defer gock.Off()

		gock.New("http://localhost:45634").
			MatchHeader("Content-Type", "application/json").
			JSON(map[string]any{"command": "status-get", "service": []string{"dhcp4"}}).
			Post("/").
			Reply(500)

		accessPoint := AccessPoint{
			Type:     AccessPointControl,
			Address:  "localhost",
			Port:     45634,
			Protocol: protocoltype.HTTP,
		}
		connector := newKeaConnector(
			accessPoint,
			HTTPClientConfig{Interceptor: gock.InterceptClient},
		)

		daemon := &keaDaemon{
			daemon: daemon{
				Name:         daemonname.DHCPv4,
				AccessPoints: []AccessPoint{accessPoint},
			},
			connector: connector,
		}

		err := daemon.ensureWatchingLeasefile(t.Context(), &config, 10)
		require.ErrorContains(t, err, "500")

		require.False(t, gock.HasUnmatchedRequest())
	})
	// Call the function once and give it a default configuration, but with the
	// value returned by get-status pointing to a file which doesn't exist. It
	// should handle this gracefully, log it, and not take down the agent.
	t.Run("dhcp4 persist true but csv-lease-file missing", func(t *testing.T) {
		persist := true
		config := keaconfig.Config{
			DHCPv4Config: &keaconfig.DHCPv4Config{
				CommonDHCPConfig: keaconfig.CommonDHCPConfig{
					LeaseDatabase: &keaconfig.Database{
						Type:    "memfile",
						Persist: &persist,
					},
				},
			},
		}
		defer gock.Off()

		dhcpV4ResponsesJSON := `[
			{
				"result": 0,
				"arguments": {}
			}
		]`
		err := makeKeaCommandMock("dhcp4", dhcpV4ResponsesJSON, 1)
		require.NoError(t, err)

		accessPoint := AccessPoint{
			Type:     AccessPointControl,
			Address:  "localhost",
			Port:     45634,
			Protocol: protocoltype.HTTP,
		}
		connector := newKeaConnector(
			accessPoint,
			HTTPClientConfig{Interceptor: gock.InterceptClient},
		)

		daemon := &keaDaemon{
			daemon: daemon{
				Name:         daemonname.DHCPv4,
				AccessPoints: []AccessPoint{accessPoint},
			},
			connector: connector,
		}

		err = daemon.ensureWatchingLeasefile(t.Context(), &config, 10)
		require.ErrorContains(t, err, "status API did not return the path to the lease memfile")

		require.False(t, gock.HasUnmatchedRequest())
	})
}
