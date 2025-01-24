package configmigrator

import (
	"bytes"
	context "context"
	"encoding/binary"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"
	keaconfig "isc.org/stork/appcfg/kea"
	keactrl "isc.org/stork/appctrl/kea"
	"isc.org/stork/server/agentcomm"
	dbmodel "isc.org/stork/server/database/model"
)

//go:generate mockgen -package=configmigrator -destination=agentcommmock_test.go isc.org/stork/server/agentcomm ConnectedAgents

// Test that the hosts are migrated and all errors are collected.
func TestMigrate(t *testing.T) {
	// Arrange
	// Mocks and basic data.
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	agentMock := NewMockConnectedAgents(ctrl)
	lookup := dbmodel.NewDHCPOptionDefinitionLookup()

	migrator := NewHostMigrator(
		dbmodel.HostsByPageFilters{}, nil, agentMock,
		dbmodel.NewDHCPOptionDefinitionLookup(),
	).(*hostMigrator)

	// Assertion helpers.
	// The errors in communication between the Stork server and Kea DHCP daemon
	// may occur in many places. Every case generates a different error object.
	type mockErrors struct {
		// The error occurred in GRPC communication between the Stork server
		// and Stork agent.
		grpcErr error
		// The error occurred in communication between the Kea CA and Kea DHCP
		// daemon.
		keaErr error
		// The errors occurred in the Kea DHCP daemon before or after executing
		// the command. It must have the same length as the number of commands
		// sent to the Kea DHCP daemon.
		cmdErrs []error
		// The errors occurred in the Kea DHCP daemon in executing the command
		// handler. It must have the same length as the number of commands sent
		// to the Kea DHCP daemon.
		executionErrs []error
	}

	expectReservationAddCommandWithError := func(daemon *dbmodel.Daemon, err mockErrors, hosts ...dbmodel.Host) {
		var reservations []keactrl.SerializableCommand
		for _, host := range hosts {
			reservation, _ := keaconfig.CreateHostCmdsReservation(
				daemon.ID, lookup, host,
			)
			reservations = append(reservations, keactrl.NewCommandReservationAdd(
				reservation, daemon.Name,
			))
		}

		agentMock.EXPECT().ForwardToKeaOverHTTP(
			gomock.Any(),            // Context.
			gomock.Eq(daemon.App),   // App.
			gomock.Eq(reservations), // Commands.
			gomock.Any(),            // Responses.
		).Do(func(ctx context.Context, app *dbmodel.App, cmds []keactrl.SerializableCommand, cmdResponses ...any) {
			for i := range cmdResponses {
				if i >= len(err.executionErrs) || err.executionErrs[i] == nil {
					continue
				}

				r := cmdResponses[i].(*keactrl.ResponseList)
				require.Empty(t, *r)

				(*r) = append(*r, keactrl.Response{
					ResponseHeader: keactrl.ResponseHeader{
						Result: keactrl.ResponseError,
						Text:   err.executionErrs[i].Error(),
					},
				})
			}
		}).Return(&agentcomm.KeaCmdsResult{
			Error:      err.keaErr,
			CmdsErrors: err.cmdErrs,
		}, err.grpcErr)
	}

	expectReservationAddCommandNoError := func(daemon *dbmodel.Daemon, hosts ...dbmodel.Host) {
		expectReservationAddCommandWithError(daemon, mockErrors{}, hosts...)
	}

	expectReservationDelCommandWithError := func(daemon *dbmodel.Daemon, err mockErrors, hosts ...dbmodel.Host) {
		var reservations []keactrl.SerializableCommand

		for _, host := range hosts {
			deletedReservation, _ := keaconfig.CreateHostCmdsDeletedReservation(
				daemon.ID, host, keaconfig.HostCmdsOperationTargetMemory,
			)
			reservations = append(reservations, keactrl.NewCommandReservationDel(
				deletedReservation, daemon.Name,
			))
		}

		agentMock.EXPECT().ForwardToKeaOverHTTP(
			gomock.Any(),            // Context.
			gomock.Eq(daemon.App),   // App.
			gomock.Eq(reservations), // Commands.
			gomock.Any(),            // Responses.
		).Do(func(ctx context.Context, app *dbmodel.App, cmds []keactrl.SerializableCommand, cmdResponses ...any) {
			for i := range cmdResponses {
				if i >= len(err.executionErrs) || err.executionErrs[i] == nil {
					continue
				}

				r := cmdResponses[i].(*keactrl.ResponseList)
				require.Empty(t, *r)

				(*r) = append(*r, keactrl.Response{
					ResponseHeader: keactrl.ResponseHeader{
						Result: keactrl.ResponseError,
						Text:   err.executionErrs[i].Error(),
					},
				})
			}
		}).Return(&agentcomm.KeaCmdsResult{
			Error:      err.keaErr,
			CmdsErrors: err.cmdErrs,
		}, err.grpcErr)
	}

	expectReservationDelCommandNoError := func(daemon *dbmodel.Daemon, hosts ...dbmodel.Host) {
		expectReservationDelCommandWithError(daemon, mockErrors{}, hosts...)
	}

	expectConfigWriteCommandWithError := func(daemon *dbmodel.Daemon, err mockErrors) {
		agentMock.EXPECT().ForwardToKeaOverHTTP(
			gomock.Any(),          // Context.
			gomock.Eq(daemon.App), // App.
			gomock.Eq([]keactrl.SerializableCommand{
				keactrl.NewCommandBase(keactrl.ConfigWrite, daemon.Name),
			}), // Commands.
			gomock.Any(), // Responses.
		).Do(func(ctx context.Context, app *dbmodel.App, cmds []keactrl.SerializableCommand, cmdResponses ...any) {
			for i := range cmdResponses {
				if i >= len(err.executionErrs) || err.executionErrs[i] == nil {
					continue
				}

				r := cmdResponses[i].(*keactrl.ResponseList)
				require.Empty(t, *r)

				(*r) = append(*r, keactrl.Response{
					ResponseHeader: keactrl.ResponseHeader{
						Result: keactrl.ResponseError,
						Text:   err.executionErrs[i].Error(),
					},
				})
			}
		}).Return(&agentcomm.KeaCmdsResult{
			Error:      err.keaErr,
			CmdsErrors: err.cmdErrs,
		}, err.grpcErr)
	}

	expectConfigWriteCommandNoError := func(daemon *dbmodel.Daemon) {
		expectConfigWriteCommandWithError(daemon, mockErrors{})
	}

	// Entities.
	// Each sub-test must have separate entities to avoid calling the mock
	// handlers from the previous test.
	nextHostID := int64(1)
	nextLocalHostID := int64(1)
	nextDaemonID := int64(1)

	createHost := func(daemons ...*dbmodel.Daemon) dbmodel.Host {
		var localHosts []dbmodel.LocalHost
		for _, daemon := range daemons {
			localHosts = append(localHosts, dbmodel.LocalHost{
				ID:         nextLocalHostID,
				DaemonID:   daemon.ID,
				Daemon:     daemon,
				DataSource: dbmodel.HostDataSourceConfig,
			})
			nextLocalHostID++
		}

		var identifier bytes.Buffer
		_ = binary.Write(&identifier, binary.LittleEndian, 1024+nextHostID)

		host := dbmodel.Host{
			ID: nextHostID,
			HostIdentifiers: []dbmodel.HostIdentifier{{
				ID:     nextHostID,
				Type:   "hw-address",
				Value:  identifier.Bytes(),
				HostID: 1,
			}},
			LocalHosts: localHosts,
		}
		nextHostID++

		return host
	}

	createDaemon := func() *dbmodel.Daemon {
		daemon := &dbmodel.Daemon{
			ID:     nextDaemonID,
			Name:   dbmodel.DaemonNameDHCPv4,
			App:    &dbmodel.App{ID: nextDaemonID},
			Active: true,
		}
		nextDaemonID++
		return daemon
	}

	// Tests migrating a single host with no errors.
	t.Run("single host, single daemon, all OK", func(t *testing.T) {
		daemon := createDaemon()
		host := createHost(daemon)

		expectReservationAddCommandNoError(daemon, host)
		expectReservationDelCommandNoError(daemon, host)
		expectConfigWriteCommandNoError(daemon)

		migrator.items = []dbmodel.Host{host}

		// Act
		errs := migrator.Migrate()

		// Assert
		require.Empty(t, errs)
		require.True(t, ctrl.Satisfied())
	})

	// Tests that the inactive daemon is skipped and generates no API calls.
	t.Run("inactive daemon", func(t *testing.T) {
		daemonInactive := createDaemon()
		daemonInactive.Active = false
		host := createHost(daemonInactive)

		migrator.items = []dbmodel.Host{host}

		// Act
		errs := migrator.Migrate()

		// Assert
		require.Empty(t, errs)
		require.True(t, ctrl.Satisfied())
	})

	// Tests that the migration doesn't fail for no hosts.
	t.Run("no hosts", func(t *testing.T) {
		migrator.items = []dbmodel.Host{}

		// Act
		errs := migrator.Migrate()

		// Assert
		require.Empty(t, errs)
		require.True(t, ctrl.Satisfied())
	})

	// Tests migrating multiple hosts belonging to a single daemon. The hosts
	// should be added to host database and deleted from the configuration in
	// a single batch.
	t.Run("multiple hosts, single daemons, all OK", func(t *testing.T) {
		daemon := createDaemon()
		hosts := []dbmodel.Host{
			createHost(daemon),
			createHost(daemon),
			createHost(daemon),
			createHost(daemon),
		}

		expectReservationAddCommandNoError(daemon, hosts...)
		expectReservationDelCommandNoError(daemon, hosts...)
		expectConfigWriteCommandNoError(daemon)

		migrator.items = hosts

		// Act
		errs := migrator.Migrate()

		// Assert
		require.Empty(t, errs)
		require.True(t, ctrl.Satisfied())
	})

	// Tests migrating multiple hosts belonging to multiple daemons. The hosts
	// should be processed in separate batches for each daemon.
	t.Run("multiple hosts, multiple daemons, all OK", func(t *testing.T) {
		daemon1 := createDaemon()
		daemon2 := createDaemon()
		hosts := []dbmodel.Host{
			createHost(daemon1),
			createHost(daemon1),
			createHost(daemon2),
			createHost(daemon2),
			createHost(daemon1, daemon2),
		}

		expectReservationAddCommandNoError(daemon1, hosts[0], hosts[1], hosts[4])
		expectReservationDelCommandNoError(daemon1, hosts[0], hosts[1], hosts[4])
		expectConfigWriteCommandNoError(daemon1)

		expectReservationAddCommandNoError(daemon2, hosts[2], hosts[3], hosts[4])
		expectReservationDelCommandNoError(daemon2, hosts[2], hosts[3], hosts[4])
		expectConfigWriteCommandNoError(daemon2)

		migrator.items = hosts

		// Act
		errs := migrator.Migrate()

		// Assert
		require.Empty(t, errs)
		require.True(t, ctrl.Satisfied())
	})

	// Test that if the error occurs during adding a reservation to the
	// database, it isn't removed from the configuration.
	// Test considers all error reasons: an error occurred in the Stork
	// agent, an error occurred in the Kea CA, an error occurred in the
	// DHCP daemon, and an error returned as a command response. The errors
	// occurred in the Stork agent and the Kea CA should cause all hosts from
	// a given daemon to be marked as failed with the same error.
	t.Run("error adding reservation", func(t *testing.T) {
		daemon1 := createDaemon()
		daemon2 := createDaemon()
		daemon3 := createDaemon()
		daemon4 := createDaemon()

		host1 := createHost(daemon1)
		host2 := createHost(daemon1)
		host3 := createHost(daemon2)
		host4 := createHost(daemon2)
		host5 := createHost(daemon3)
		host6 := createHost(daemon3)
		host7 := createHost(daemon4)
		host8 := createHost(daemon4)

		// Stork agent returns an error.
		expectReservationAddCommandWithError(daemon1, mockErrors{
			grpcErr: errors.Errorf("error adding reservation"),
		}, host1, host2)
		expectConfigWriteCommandNoError(daemon1)

		// The Kea CA return an error.
		expectReservationAddCommandWithError(daemon2, mockErrors{
			keaErr: errors.Errorf("error transferring reservation"),
		}, host3, host4)
		expectConfigWriteCommandNoError(daemon2)

		// The Kea daemon returns an error while processing the add command.
		// One host should be still processed.
		expectReservationAddCommandWithError(daemon3, mockErrors{
			cmdErrs: []error{nil, errors.Errorf("error executing command")},
		}, host5, host6)
		expectReservationDelCommandNoError(daemon3, host5)
		expectConfigWriteCommandNoError(daemon3)

		// The Kea daemon processed the commands but a result of one of them
		// is an error.
		expectReservationAddCommandWithError(daemon4, mockErrors{
			executionErrs: []error{nil, errors.Errorf("error as result")},
		}, host7, host8)
		expectReservationDelCommandNoError(daemon4, host7)
		expectConfigWriteCommandNoError(daemon4)

		migrator.items = []dbmodel.Host{
			host1, host2, host3, host4, host5, host6, host7, host8,
		}

		// Act
		errs := migrator.Migrate()

		// Assert
		require.Contains(t, errs, host1.ID)
		require.ErrorContains(t, errs[host1.ID], "error adding reservation")
		require.Contains(t, errs, host2.ID)
		require.ErrorContains(t, errs[host2.ID], "error adding reservation")

		require.Contains(t, errs, host3.ID)
		require.ErrorContains(t, errs[host3.ID], "error transferring reservation")
		require.Contains(t, errs, host4.ID)
		require.ErrorContains(t, errs[host4.ID], "error transferring reservation")

		require.NotContains(t, errs, host5.ID)
		require.Contains(t, errs, host6.ID)
		require.ErrorContains(t, errs[host6.ID], "error executing command")

		require.NotContains(t, errs, host7.ID)
		require.Contains(t, errs, host8.ID)
		require.ErrorContains(t, errs[host8.ID], "error as result")

		require.True(t, ctrl.Satisfied())
	})

	// Test that if the error occurs during deleting a reservation from the
	// the configuration, the migration continues.
	// Test considers all error reasons: an error occurred in the Stork
	// agent, an error occurred in the Kea CA, an error occurred in the
	// DHCP daemon, and an error returned as a command response. The errors
	// occurred in the Stork agent and the Kea CA should cause all hosts from
	// a given daemon to be marked as failed with the same error.
	t.Run("error deleting reservation", func(t *testing.T) {
		daemon1 := createDaemon()
		daemon2 := createDaemon()
		daemon3 := createDaemon()
		daemon4 := createDaemon()

		host1 := createHost(daemon1)
		host2 := createHost(daemon1)
		host3 := createHost(daemon2)
		host4 := createHost(daemon2)
		host5 := createHost(daemon3)
		host6 := createHost(daemon3)
		host7 := createHost(daemon4)
		host8 := createHost(daemon4)

		// Stork agent returns an error.
		expectReservationAddCommandNoError(daemon1, host1, host2)
		expectReservationDelCommandWithError(daemon1, mockErrors{
			grpcErr: errors.Errorf("error GRPC"),
		}, host1, host2)
		expectConfigWriteCommandNoError(daemon1)

		// The Kea CA return an error.
		expectReservationAddCommandNoError(daemon2, host3, host4)
		expectReservationDelCommandWithError(daemon2, mockErrors{
			keaErr: errors.Errorf("error Kea CA"),
		}, host3, host4)
		expectConfigWriteCommandNoError(daemon2)

		// The Kea daemon returns an error while processing the add command.
		// One host should be still processed.
		expectReservationAddCommandNoError(daemon3, host5, host6)
		expectReservationDelCommandWithError(daemon3, mockErrors{
			cmdErrs: []error{nil, errors.Errorf("error Kea daemon")},
		}, host5, host6)
		expectConfigWriteCommandNoError(daemon3)

		// The Kea daemon processed the commands but a result of one of them
		// is an error.
		expectReservationAddCommandNoError(daemon4, host7, host8)
		expectReservationDelCommandWithError(daemon4, mockErrors{
			executionErrs: []error{nil, errors.Errorf("error is result")},
		}, host7, host8)
		expectConfigWriteCommandNoError(daemon4)

		migrator.items = []dbmodel.Host{
			host1, host2, host3, host4, host5, host6, host7, host8,
		}

		// Act
		errs := migrator.Migrate()

		// Assert
		require.Contains(t, errs, host1.ID)
		require.ErrorContains(t, errs[host1.ID], "error GRPC")
		require.Contains(t, errs, host2.ID)
		require.ErrorContains(t, errs[host2.ID], "error GRPC")

		require.Contains(t, errs, host3.ID)
		require.ErrorContains(t, errs[host3.ID], "error Kea CA")
		require.Contains(t, errs, host4.ID)
		require.ErrorContains(t, errs[host4.ID], "error Kea CA")

		require.NotContains(t, errs, host5.ID)
		require.Contains(t, errs, host6.ID)
		require.ErrorContains(t, errs[host6.ID], "error Kea daemon")

		require.NotContains(t, errs, host7.ID)
		require.Contains(t, errs, host8.ID)
		require.ErrorContains(t, errs[host8.ID], "error is result")

		require.True(t, ctrl.Satisfied())
	})

	// Test that if the error occurs for a host that belongs to multiple
	// daemons, the next commands for this host are no longer created for
	// further processed daemons.
	t.Run("error adding reservation - multiple daemons", func(t *testing.T) {
		daemon1 := createDaemon()
		daemon2 := createDaemon()
		daemon3 := createDaemon()

		host := createHost(daemon1, daemon2, daemon3)

		expectReservationAddCommandWithError(daemon1, mockErrors{
			cmdErrs: []error{errors.Errorf("error adding reservation")},
		}, host)

		expectConfigWriteCommandNoError(daemon1)
		expectConfigWriteCommandNoError(daemon2)
		expectConfigWriteCommandNoError(daemon3)

		migrator.items = []dbmodel.Host{host}

		// Act
		errs := migrator.Migrate()

		// Assert
		require.Contains(t, errs, host.ID)
		require.ErrorContains(t, errs[host.ID], "error adding reservation")

		require.True(t, ctrl.Satisfied())
	})

	// Test that if the error occurs for a host that belongs to multiple
	// daemons, the next commands for this host are no longer created for
	// further processed daemons.
	t.Run("error removing reservation - multiple daemons", func(t *testing.T) {
		daemon1 := createDaemon()
		daemon2 := createDaemon()
		daemon3 := createDaemon()

		host := createHost(daemon1, daemon2, daemon3)

		expectReservationAddCommandNoError(daemon1, host)
		expectReservationDelCommandWithError(daemon1, mockErrors{
			cmdErrs: []error{errors.Errorf("error adding reservation")},
		}, host)

		expectConfigWriteCommandNoError(daemon1)
		expectConfigWriteCommandNoError(daemon2)
		expectConfigWriteCommandNoError(daemon3)

		migrator.items = []dbmodel.Host{host}

		// Act
		errs := migrator.Migrate()

		// Assert
		require.Contains(t, errs, host.ID)
		require.ErrorContains(t, errs[host.ID], "error adding reservation")

		require.True(t, ctrl.Satisfied())
	})

	// Test that if the error occurs during saving the configuration, the
	// migration continues but all hosts from a given daemon to be marked as
	// failed with the same error.
	t.Run("error saving configuration", func(t *testing.T) {
		daemon1 := createDaemon()
		daemon2 := createDaemon()
		daemon3 := createDaemon()
		daemon4 := createDaemon()

		host1 := createHost(daemon1)
		host2 := createHost(daemon1)
		host3 := createHost(daemon2)
		host4 := createHost(daemon2)
		host5 := createHost(daemon3)
		host6 := createHost(daemon3)
		host7 := createHost(daemon4)
		host8 := createHost(daemon4)

		expectReservationAddCommandNoError(daemon1, host1, host2)
		expectReservationDelCommandNoError(daemon1, host1, host2)
		expectConfigWriteCommandWithError(daemon1, mockErrors{
			grpcErr: errors.Errorf("error GRPC"),
		})

		expectReservationAddCommandNoError(daemon2, host3, host4)
		expectReservationDelCommandNoError(daemon2, host3, host4)
		expectConfigWriteCommandWithError(daemon2, mockErrors{
			keaErr: errors.Errorf("error Kea"),
		})

		expectReservationAddCommandNoError(daemon3, host5, host6)
		expectReservationDelCommandNoError(daemon3, host5, host6)
		expectConfigWriteCommandWithError(daemon3, mockErrors{
			cmdErrs: []error{errors.Errorf("error command")},
		})

		expectReservationAddCommandNoError(daemon4, host7, host8)
		expectReservationDelCommandNoError(daemon4, host7, host8)
		expectConfigWriteCommandWithError(daemon4, mockErrors{
			executionErrs: []error{errors.Errorf("error execution")},
		})

		migrator.items = []dbmodel.Host{
			host1, host2, host3, host4, host5, host6, host7, host8,
		}

		// Act
		errs := migrator.Migrate()

		// Assert
		require.Contains(t, errs, host1.ID)
		require.ErrorContains(t, errs[host1.ID], "error GRPC")
		require.Contains(t, errs, host2.ID)
		require.ErrorContains(t, errs[host2.ID], "error GRPC")

		require.Contains(t, errs, host3.ID)
		require.ErrorContains(t, errs[host3.ID], "error Kea")
		require.Contains(t, errs, host4.ID)
		require.ErrorContains(t, errs[host4.ID], "error Kea")

		require.Contains(t, errs, host5.ID)
		require.ErrorContains(t, errs[host5.ID], "error command")
		require.Contains(t, errs, host6.ID)
		require.ErrorContains(t, errs[host6.ID], "error command")

		require.Contains(t, errs, host7.ID)
		require.ErrorContains(t, errs[host7.ID], "error execution")
		require.Contains(t, errs, host8.ID)
		require.ErrorContains(t, errs[host8.ID], "error execution")

		require.True(t, ctrl.Satisfied())
	})

	// Test that the migration keeps the first occurred error for a host.
	t.Run("multiple errors for a host", func(t *testing.T) {
		daemon1 := createDaemon()
		daemon2 := createDaemon()
		daemon3 := createDaemon()
		daemon4 := createDaemon()

		host1 := createHost(daemon1, daemon2, daemon3)
		host2 := createHost(daemon1, daemon2, daemon3)
		host3 := createHost(daemon1, daemon2, daemon3, daemon4)
		host4 := createHost(daemon1, daemon2, daemon4)

		// The first error is returned as a response to the command.
		expectReservationAddCommandWithError(daemon1, mockErrors{
			executionErrs: []error{
				errors.Errorf("response error"),
				nil,
				nil,
				nil,
			},
		}, host1, host2, host3, host4)
		expectReservationDelCommandNoError(daemon1, host2, host3, host4)
		expectConfigWriteCommandNoError(daemon1)

		// The second error is returned as a command error.
		expectReservationAddCommandWithError(daemon2, mockErrors{
			cmdErrs: []error{
				errors.Errorf("command error"),
				nil,
				nil,
			},
		}, host2, host3, host4)
		expectReservationDelCommandNoError(daemon2, host3, host4)
		expectConfigWriteCommandNoError(daemon2)

		// The third error is returned as a Kea CA error.
		expectReservationAddCommandWithError(daemon3, mockErrors{
			keaErr: errors.Errorf("Kea CA error"),
		}, host3)
		expectConfigWriteCommandNoError(daemon3)

		// The fourth error is returned as a Stork agent error.
		expectReservationAddCommandWithError(daemon4, mockErrors{
			grpcErr: errors.Errorf("Stork agent error"),
		}, host4)
		expectConfigWriteCommandNoError(daemon4)

		migrator.items = []dbmodel.Host{host1, host2, host3, host4}

		// Act
		errs := migrator.Migrate()

		// Assert
		require.Contains(t, errs, host1.ID)
		require.ErrorContains(t, errs[host1.ID], "response error")

		require.Contains(t, errs, host2.ID)
		require.ErrorContains(t, errs[host2.ID], "command error")

		require.Contains(t, errs, host3.ID)
		require.ErrorContains(t, errs[host3.ID], "Kea CA error")

		require.Contains(t, errs, host4.ID)
		require.ErrorContains(t, errs[host4.ID], "Stork agent error")

		require.True(t, ctrl.Satisfied())
	})

	// The host to migrate exists only in the database. The host should not be
	// added to the database again nor removed from the configuration.
	t.Run("host exists only in the database", func(t *testing.T) {
		daemon := createDaemon()

		host := createHost(daemon)
		host.LocalHosts[0].DataSource = dbmodel.HostDataSourceAPI

		expectConfigWriteCommandNoError(daemon)

		migrator.items = []dbmodel.Host{host}

		// Act
		errs := migrator.Migrate()

		// Assert
		require.Empty(t, errs)
		require.True(t, ctrl.Satisfied())
	})

	// The host to migrate is duplicated in the configuration file and the
	// database. The host should not be added to the database again but removed
	// from the configuration.
	t.Run("host exists in the database and the configuration", func(t *testing.T) {
		daemon := createDaemon()

		host := createHost(daemon, daemon)
		host.LocalHosts[0].DataSource = dbmodel.HostDataSourceAPI

		expectReservationDelCommandNoError(daemon, host)
		expectConfigWriteCommandNoError(daemon)

		migrator.items = []dbmodel.Host{host}

		// Act
		errs := migrator.Migrate()

		// Assert
		require.Empty(t, errs)
		require.True(t, ctrl.Satisfied())
	})

	// Host cannot be converted to the GRPC API format that interrupts the
	// command preparation.
	t.Run("invalid host", func(t *testing.T) {
		// The host has assigned a subnet that is not assigned to any daemon.
		daemon := createDaemon()
		host := createHost(daemon)
		host.Subnet = &dbmodel.Subnet{ID: 42}

		expectConfigWriteCommandNoError(daemon)

		migrator.items = []dbmodel.Host{host}

		// Act
		errs := migrator.Migrate()

		// Assert
		require.Contains(t, errs, host.ID)
		require.ErrorContains(t,
			errs[host.ID],
			"local subnet id not found in host",
		)
		require.True(t, ctrl.Satisfied())
	})

	// Host belonging to multiple daemons cannot be converted to the GRPC API
	// format that interrupts the command preparation.
	t.Run("invalid host - multiple daemons", func(t *testing.T) {
		// The host has assigned a subnet that is not assigned to any daemon.
		daemon1 := createDaemon()
		daemon2 := createDaemon()

		host := createHost(daemon1, daemon2)
		host.Subnet = &dbmodel.Subnet{ID: 42}

		expectConfigWriteCommandNoError(daemon1)
		expectConfigWriteCommandNoError(daemon2)

		migrator.items = []dbmodel.Host{host}

		// Act
		errs := migrator.Migrate()

		// Assert
		require.Contains(t, errs, host.ID)
		require.ErrorContains(t,
			errs[host.ID],
			"local subnet id not found in host",
		)
		require.True(t, ctrl.Satisfied())
	})
}
