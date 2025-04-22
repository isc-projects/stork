package entitymigrator

import (
	context "context"
	"fmt"
	"sort"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"
	keaconfig "isc.org/stork/appcfg/kea"
	keactrl "isc.org/stork/appctrl/kea"
	"isc.org/stork/server/agentcomm"
	"isc.org/stork/server/config"
	"isc.org/stork/server/configmigrator"
	dbmodel "isc.org/stork/server/database/model"
	dbtest "isc.org/stork/server/database/test"
	storkutil "isc.org/stork/util"
)

//go:generate mockgen -package=entitymigrator -destination=agentcommmock_test.go isc.org/stork/server/agentcomm ConnectedAgents
//go:generate mockgen -package=entitymigrator -destination=daemonlockermock_test.go isc.org/stork/server/config DaemonLocker
//go:generate mockgen -package=entitymigrator -destination=pausermock_test.go isc.org/stork/server/configmigrator/entitymigrator Pauser

// Test that the hosts are migrated and all errors are collected.
func TestMigrate(t *testing.T) {
	// Arrange
	// Mocks and basic data.
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	lookup := dbmodel.NewDHCPOptionDefinitionLookup()
	agentMock := NewMockConnectedAgents(ctrl)
	lockerMock := NewMockDaemonLocker(ctrl)
	statePuller := NewMockPauser(ctrl)
	hostPuller := NewMockPauser(ctrl)

	migrator := NewHostMigrator(
		dbmodel.HostsByPageFilters{}, nil, agentMock,
		dbmodel.NewDHCPOptionDefinitionLookup(),
		lockerMock,
		statePuller,
		hostPuller,
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

	expectForwardToKeaOverHTTP := func(daemon *dbmodel.Daemon, cmds []keactrl.SerializableCommand, err mockErrors) *gomock.Call {
		return agentMock.EXPECT().ForwardToKeaOverHTTP(
			gomock.Any(),          // Context.
			gomock.Eq(daemon.App), // App.
			gomock.Eq(cmds),       // Commands.
			gomock.Any(),          // Responses.
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

	expectReservationAddCommandWithError := func(daemon *dbmodel.Daemon, err mockErrors, hosts ...dbmodel.Host) *gomock.Call {
		var reservations []keactrl.SerializableCommand
		for _, host := range hosts {
			reservation, _ := keaconfig.CreateHostCmdsReservation(
				daemon.ID, lookup, host,
			)
			reservations = append(reservations, keactrl.NewCommandReservationAdd(
				reservation, daemon.Name,
			))
		}

		return expectForwardToKeaOverHTTP(daemon, reservations, err)
	}

	expectReservationAddCommandNoError := func(daemon *dbmodel.Daemon, hosts ...dbmodel.Host) *gomock.Call {
		return expectReservationAddCommandWithError(daemon, mockErrors{}, hosts...)
	}

	expectReservationDelCommandWithError := func(daemon *dbmodel.Daemon, err mockErrors, hosts ...dbmodel.Host) *gomock.Call {
		var reservations []keactrl.SerializableCommand

		for _, host := range hosts {
			deletedReservation, _ := keaconfig.CreateHostCmdsDeletedReservation(
				daemon.ID, host, keaconfig.HostCmdsOperationTargetMemory,
			)
			reservations = append(reservations, keactrl.NewCommandReservationDel(
				deletedReservation, daemon.Name,
			))
		}

		return expectForwardToKeaOverHTTP(daemon, reservations, err)
	}

	expectReservationDelCommandNoError := func(daemon *dbmodel.Daemon, hosts ...dbmodel.Host) *gomock.Call {
		return expectReservationDelCommandWithError(daemon, mockErrors{}, hosts...)
	}

	expectConfigWriteCommandWithError := func(daemon *dbmodel.Daemon, err mockErrors) *gomock.Call {
		return expectForwardToKeaOverHTTP(daemon, []keactrl.SerializableCommand{
			keactrl.NewCommandBase(keactrl.ConfigWrite, daemon.Name),
		}, err)
	}

	expectConfigWriteCommandNoError := func(daemon *dbmodel.Daemon) *gomock.Call {
		return expectConfigWriteCommandWithError(daemon, mockErrors{})
	}

	expectDaemonLock := func(daemon *dbmodel.Daemon, err error) *gomock.Call {
		var lockKey config.LockKey
		if err == nil {
			lockKey = config.LockKey(daemon.ID)
		}

		return lockerMock.EXPECT().
			Lock(gomock.Eq(daemon.ID)).
			Return(lockKey, err)
	}

	expectDaemonLockNoError := func(daemon *dbmodel.Daemon) *gomock.Call {
		return expectDaemonLock(daemon, nil)
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

		host := dbmodel.Host{
			ID: nextHostID,
			HostIdentifiers: []dbmodel.HostIdentifier{{
				ID:     nextHostID,
				Type:   "hw-address",
				Value:  []byte{byte(nextHostID)},
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

	getExpectedLabel := func(err configmigrator.MigrationError) string {
		if err.CauseEntity == configmigrator.ErrorCauseEntityHost {
			return fmt.Sprintf(
				"hw-address=%s",
				storkutil.BytesToHex([]byte{byte(err.ID)}),
			)
		}
		if err.CauseEntity == configmigrator.ErrorCauseEntityDaemon {
			return "dhcp4"
		}
		return "unsupported"
	}

	// Tests migrating a single host with no errors.
	t.Run("single host, single daemon, all OK", func(t *testing.T) {
		daemon := createDaemon()
		host := createHost(daemon)

		gomock.InOrder(
			expectDaemonLockNoError(daemon),
			expectReservationAddCommandNoError(daemon, host),
			expectReservationDelCommandNoError(daemon, host),
			expectConfigWriteCommandNoError(daemon),
		)

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

		gomock.InOrder(
			expectDaemonLockNoError(daemon),
			expectReservationAddCommandNoError(daemon, hosts...),
			expectReservationDelCommandNoError(daemon, hosts...),
			expectConfigWriteCommandNoError(daemon),
		)

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

		gomock.InOrder(
			expectDaemonLockNoError(daemon1),
			expectReservationAddCommandNoError(daemon1, hosts[0], hosts[1], hosts[4]),
			expectReservationDelCommandNoError(daemon1, hosts[0], hosts[1], hosts[4]),
			expectConfigWriteCommandNoError(daemon1),

			expectDaemonLockNoError(daemon2),
			expectReservationAddCommandNoError(daemon2, hosts[2], hosts[3], hosts[4]),
			expectReservationDelCommandNoError(daemon2, hosts[2], hosts[3], hosts[4]),
			expectConfigWriteCommandNoError(daemon2),
		)

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

		gomock.InOrder(
			expectDaemonLockNoError(daemon1),
			// Stork agent returns an error.
			expectReservationAddCommandWithError(daemon1, mockErrors{
				grpcErr: errors.Errorf("error adding reservation"),
			}, host1, host2),
			expectConfigWriteCommandNoError(daemon1),

			// The Kea CA return an error.
			expectDaemonLockNoError(daemon2),
			expectReservationAddCommandWithError(daemon2, mockErrors{
				keaErr: errors.Errorf("error transferring reservation"),
			}, host3, host4),
			expectConfigWriteCommandNoError(daemon2),

			// The Kea daemon returns an error while processing the add command.
			// One host should be still processed.
			expectDaemonLockNoError(daemon3),
			expectReservationAddCommandWithError(daemon3, mockErrors{
				cmdErrs: []error{nil, errors.Errorf("error executing command")},
			}, host5, host6),
			expectReservationDelCommandNoError(daemon3, host5),
			expectConfigWriteCommandNoError(daemon3),

			// The Kea daemon processed the commands but a result of one of them
			// is an error.
			expectDaemonLockNoError(daemon4),
			expectReservationAddCommandWithError(daemon4, mockErrors{
				executionErrs: []error{nil, errors.Errorf("error as result")},
			}, host7, host8),
			expectReservationDelCommandNoError(daemon4, host7),
			expectConfigWriteCommandNoError(daemon4),
		)

		migrator.items = []dbmodel.Host{
			host1, host2, host3, host4, host5, host6, host7, host8,
		}

		// Act
		errs := migrator.Migrate()

		// Assert
		require.Len(t, errs, 4)
		sort.Slice(errs, func(i, j int) bool {
			if errs[i].CauseEntity != errs[j].CauseEntity {
				return len(errs[i].CauseEntity) > len(errs[j].CauseEntity)
			}
			return errs[i].ID < errs[j].ID
		})

		require.EqualValues(t, daemon1.ID, errs[0].ID)
		require.ErrorContains(t, errs[0].Error, "error adding reservation")
		require.EqualValues(t, configmigrator.ErrorCauseEntityDaemon, errs[0].CauseEntity)

		require.EqualValues(t, daemon2.ID, errs[1].ID)
		require.ErrorContains(t, errs[1].Error, "error transferring reservation")
		require.EqualValues(t, configmigrator.ErrorCauseEntityDaemon, errs[1].CauseEntity)

		require.EqualValues(t, host6.ID, errs[2].ID)
		require.ErrorContains(t, errs[2].Error, "error executing command")
		require.EqualValues(t, configmigrator.ErrorCauseEntityHost, errs[2].CauseEntity)

		require.EqualValues(t, host8.ID, errs[3].ID)
		require.ErrorContains(t, errs[3].Error, "error as result")
		require.EqualValues(t, configmigrator.ErrorCauseEntityHost, errs[3].CauseEntity)

		for _, err := range errs {
			require.EqualValues(t, getExpectedLabel(err), err.Label)
		}

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

		gomock.InOrder(
			// Stork agent returns an error.
			expectDaemonLockNoError(daemon1),
			expectReservationAddCommandNoError(daemon1, host1, host2),
			expectReservationDelCommandWithError(daemon1, mockErrors{
				grpcErr: errors.Errorf("error GRPC"),
			}, host1, host2),
			expectConfigWriteCommandNoError(daemon1),

			// The Kea CA return an error.
			expectDaemonLockNoError(daemon2),
			expectReservationAddCommandNoError(daemon2, host3, host4),
			expectReservationDelCommandWithError(daemon2, mockErrors{
				keaErr: errors.Errorf("error Kea CA"),
			}, host3, host4),
			expectConfigWriteCommandNoError(daemon2),

			// The Kea daemon returns an error while processing the add command.
			// One host should be still processed.
			expectDaemonLockNoError(daemon3),
			expectReservationAddCommandNoError(daemon3, host5, host6),
			expectReservationDelCommandWithError(daemon3, mockErrors{
				cmdErrs: []error{nil, errors.Errorf("error Kea command")},
			}, host5, host6),
			expectConfigWriteCommandNoError(daemon3),

			// The Kea daemon processed the commands but a result of one of them
			// is an error.
			expectDaemonLockNoError(daemon4),
			expectReservationAddCommandNoError(daemon4, host7, host8),
			expectReservationDelCommandWithError(daemon4, mockErrors{
				executionErrs: []error{nil, errors.Errorf("error is result")},
			}, host7, host8),
			expectConfigWriteCommandNoError(daemon4),
		)

		migrator.items = []dbmodel.Host{
			host1, host2, host3, host4, host5, host6, host7, host8,
		}

		// Act
		errs := migrator.Migrate()

		// Assert
		require.Len(t, errs, 4)

		sort.Slice(errs, func(i, j int) bool {
			if errs[i].CauseEntity != errs[j].CauseEntity {
				return len(errs[i].CauseEntity) > len(errs[j].CauseEntity)
			}
			return errs[i].ID < errs[j].ID
		})

		require.Equal(t, daemon1.ID, errs[0].ID)
		require.ErrorContains(t, errs[0].Error, "error GRPC")
		require.EqualValues(t, configmigrator.ErrorCauseEntityDaemon, errs[0].CauseEntity)

		require.Equal(t, daemon2.ID, errs[1].ID)
		require.ErrorContains(t, errs[1].Error, "error Kea CA")
		require.EqualValues(t, configmigrator.ErrorCauseEntityDaemon, errs[1].CauseEntity)

		require.Equal(t, host6.ID, errs[2].ID)
		require.ErrorContains(t, errs[2].Error, "error Kea command")
		require.Equal(t, configmigrator.ErrorCauseEntityHost, errs[2].CauseEntity)

		require.Equal(t, host8.ID, errs[3].ID)
		require.ErrorContains(t, errs[3].Error, "error is result")
		require.Equal(t, configmigrator.ErrorCauseEntityHost, errs[3].CauseEntity)

		for _, err := range errs {
			require.EqualValues(t, getExpectedLabel(err), err.Label)
		}

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

		gomock.InOrder(
			expectDaemonLockNoError(daemon1),
			expectReservationAddCommandWithError(daemon1, mockErrors{
				cmdErrs: []error{errors.Errorf("error adding reservation")},
			}, host),
			expectConfigWriteCommandNoError(daemon1),

			expectDaemonLockNoError(daemon2),
			expectConfigWriteCommandNoError(daemon2),

			expectDaemonLockNoError(daemon3),
			expectConfigWriteCommandNoError(daemon3),
		)

		migrator.items = []dbmodel.Host{host}

		// Act
		errs := migrator.Migrate()

		// Assert
		require.Len(t, errs, 1)
		require.EqualValues(t, host.ID, errs[0].ID)
		require.ErrorContains(t, errs[0].Error, "error adding reservation")
		require.Equal(t, configmigrator.ErrorCauseEntityHost, errs[0].CauseEntity)
		require.EqualValues(t, getExpectedLabel(errs[0]), errs[0].Label)

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

		gomock.InOrder(
			expectDaemonLockNoError(daemon1),
			expectReservationAddCommandNoError(daemon1, host),
			expectReservationDelCommandWithError(daemon1, mockErrors{
				cmdErrs: []error{errors.Errorf("error adding reservation")},
			}, host),
			expectConfigWriteCommandNoError(daemon1),

			expectDaemonLockNoError(daemon2),
			expectConfigWriteCommandNoError(daemon2),

			expectDaemonLockNoError(daemon3),
			expectConfigWriteCommandNoError(daemon3),
		)

		migrator.items = []dbmodel.Host{host}

		// Act
		errs := migrator.Migrate()

		// Assert
		require.EqualValues(t, host.ID, errs[0].ID)
		require.ErrorContains(t, errs[0].Error, "error adding reservation")
		require.Equal(t, configmigrator.ErrorCauseEntityHost, errs[0].CauseEntity)
		require.EqualValues(t, getExpectedLabel(errs[0]), errs[0].Label)

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

		gomock.InOrder(
			expectDaemonLockNoError(daemon1),
			expectReservationAddCommandNoError(daemon1, host1, host2),
			expectReservationDelCommandNoError(daemon1, host1, host2),
			expectConfigWriteCommandWithError(daemon1, mockErrors{
				grpcErr: errors.Errorf("error GRPC"),
			}),

			expectDaemonLockNoError(daemon2),
			expectReservationAddCommandNoError(daemon2, host3, host4),
			expectReservationDelCommandNoError(daemon2, host3, host4),
			expectConfigWriteCommandWithError(daemon2, mockErrors{
				keaErr: errors.Errorf("error Kea"),
			}),

			expectDaemonLockNoError(daemon3),
			expectReservationAddCommandNoError(daemon3, host5, host6),
			expectReservationDelCommandNoError(daemon3, host5, host6),
			expectConfigWriteCommandWithError(daemon3, mockErrors{
				cmdErrs: []error{errors.Errorf("error command")},
			}),

			expectDaemonLockNoError(daemon4),
			expectReservationAddCommandNoError(daemon4, host7, host8),
			expectReservationDelCommandNoError(daemon4, host7, host8),
			expectConfigWriteCommandWithError(daemon4, mockErrors{
				executionErrs: []error{errors.Errorf("error execution")},
			}),
		)

		migrator.items = []dbmodel.Host{
			host1, host2, host3, host4, host5, host6, host7, host8,
		}

		// Act
		errs := migrator.Migrate()

		// Assert
		require.Len(t, errs, 4)

		sort.Slice(errs, func(i, j int) bool {
			if errs[i].CauseEntity != errs[j].CauseEntity {
				return len(errs[i].CauseEntity) > len(errs[j].CauseEntity)
			}
			return errs[i].ID < errs[j].ID
		})

		require.EqualValues(t, daemon1.ID, errs[0].ID)
		require.ErrorContains(t, errs[0].Error, "error GRPC")
		require.Equal(t, configmigrator.ErrorCauseEntityDaemon, errs[0].CauseEntity)

		require.EqualValues(t, daemon2.ID, errs[1].ID)
		require.ErrorContains(t, errs[1].Error, "error Kea")
		require.Equal(t, configmigrator.ErrorCauseEntityDaemon, errs[1].CauseEntity)

		require.EqualValues(t, daemon3.ID, errs[2].ID)
		require.ErrorContains(t, errs[2].Error, "error command")
		require.Equal(t, configmigrator.ErrorCauseEntityDaemon, errs[2].CauseEntity)

		require.EqualValues(t, daemon4.ID, errs[3].ID)
		require.ErrorContains(t, errs[3].Error, "error execution")
		require.Equal(t, configmigrator.ErrorCauseEntityDaemon, errs[3].CauseEntity)

		for _, err := range errs {
			require.EqualValues(t, getExpectedLabel(err), err.Label)
		}

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

		gomock.InOrder(
			// The first error is returned as a response to the command.
			expectDaemonLockNoError(daemon1),
			expectReservationAddCommandWithError(daemon1, mockErrors{
				executionErrs: []error{
					errors.Errorf("response error"),
					nil,
					nil,
					nil,
				},
			}, host1, host2, host3, host4),
			expectReservationDelCommandNoError(daemon1, host2, host3, host4),
			expectConfigWriteCommandNoError(daemon1),

			// The second error is returned as a command error.
			expectDaemonLockNoError(daemon2),
			expectReservationAddCommandWithError(daemon2, mockErrors{
				cmdErrs: []error{
					errors.Errorf("command error"),
					nil,
					nil,
				},
			}, host2, host3, host4),
			expectReservationDelCommandNoError(daemon2, host3, host4),
			expectConfigWriteCommandNoError(daemon2),

			// The third error is returned as a Kea CA error.
			expectDaemonLockNoError(daemon3),
			expectReservationAddCommandWithError(daemon3, mockErrors{
				keaErr: errors.Errorf("Kea CA error"),
			}, host3),
			expectConfigWriteCommandNoError(daemon3),

			// The fourth error is returned as a Stork agent error.
			expectDaemonLockNoError(daemon4),
			expectReservationAddCommandWithError(daemon4, mockErrors{
				grpcErr: errors.Errorf("Stork agent error"),
			}, host3, host4),
			expectConfigWriteCommandNoError(daemon4),
		)

		migrator.items = []dbmodel.Host{host1, host2, host3, host4}

		// Act
		errs := migrator.Migrate()

		// Assert
		require.Len(t, errs, 4)

		sort.Slice(errs, func(i, j int) bool {
			if errs[i].CauseEntity != errs[j].CauseEntity {
				return len(errs[i].CauseEntity) < len(errs[j].CauseEntity)
			}
			return errs[i].ID < errs[j].ID
		})

		require.EqualValues(t, host1.ID, errs[0].ID)
		require.ErrorContains(t, errs[0].Error, "response error")
		require.Equal(t, configmigrator.ErrorCauseEntityHost, errs[0].CauseEntity)

		require.EqualValues(t, host2.ID, errs[1].ID)
		require.ErrorContains(t, errs[1].Error, "command error")
		require.Equal(t, configmigrator.ErrorCauseEntityHost, errs[1].CauseEntity)

		require.EqualValues(t, daemon3.ID, errs[2].ID)
		require.ErrorContains(t, errs[2].Error, "Kea CA error")
		require.Equal(t, configmigrator.ErrorCauseEntityDaemon, errs[2].CauseEntity)

		require.EqualValues(t, daemon4.ID, errs[3].ID)
		require.ErrorContains(t, errs[3].Error, "Stork agent error")
		require.Equal(t, configmigrator.ErrorCauseEntityDaemon, errs[3].CauseEntity)

		for _, err := range errs {
			require.EqualValues(t, getExpectedLabel(err), err.Label)
		}

		require.True(t, ctrl.Satisfied())
	})

	// The host to migrate exists only in the database. The host should not be
	// added to the database again nor removed from the configuration.
	t.Run("host exists only in the database", func(t *testing.T) {
		daemon := createDaemon()

		host := createHost(daemon)
		host.LocalHosts[0].DataSource = dbmodel.HostDataSourceAPI

		gomock.InOrder(
			expectDaemonLockNoError(daemon),
			expectConfigWriteCommandNoError(daemon),
		)

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

		gomock.InOrder(
			expectDaemonLockNoError(daemon),
			expectReservationDelCommandNoError(daemon, host),
			expectConfigWriteCommandNoError(daemon),
		)

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

		gomock.InOrder(
			expectDaemonLockNoError(daemon),
			expectConfigWriteCommandNoError(daemon),
		)

		migrator.items = []dbmodel.Host{host}

		// Act
		errs := migrator.Migrate()

		// Assert
		require.Len(t, errs, 1)
		require.EqualValues(t, host.ID, errs[0].ID)
		require.ErrorContains(t,
			errs[0].Error,
			"local subnet id not found in host",
		)
		require.EqualValues(t, getExpectedLabel(errs[0]), errs[0].Label)
		require.Equal(t, configmigrator.ErrorCauseEntityHost, errs[0].CauseEntity)
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

		gomock.InOrder(
			expectDaemonLockNoError(daemon1),
			expectConfigWriteCommandNoError(daemon1),
			expectDaemonLockNoError(daemon2),
			expectConfigWriteCommandNoError(daemon2),
		)

		migrator.items = []dbmodel.Host{host}

		// Act
		errs := migrator.Migrate()

		// Assert
		require.Len(t, errs, 1)
		require.EqualValues(t, host.ID, errs[0].ID)
		require.ErrorContains(t,
			errs[0].Error,
			"local subnet id not found in host",
		)
		require.Equal(t, configmigrator.ErrorCauseEntityHost, errs[0].CauseEntity)
		require.EqualValues(t, getExpectedLabel(errs[0]), errs[0].Label)
	})

	t.Run("daemon is locked only once", func(t *testing.T) {
		daemon := createDaemon()
		hosts := []dbmodel.Host{
			createHost(daemon),
			createHost(daemon),
		}

		gomock.InOrder(
			expectDaemonLockNoError(daemon),
			expectReservationAddCommandNoError(daemon, hosts[:len(hosts)/2]...),
			expectReservationDelCommandNoError(daemon, hosts[:len(hosts)/2]...),
			expectConfigWriteCommandNoError(daemon),
			expectReservationAddCommandNoError(daemon, hosts[len(hosts)/2:]...),
			expectReservationDelCommandNoError(daemon, hosts[len(hosts)/2:]...),
			expectConfigWriteCommandNoError(daemon),
		)

		// Act
		migrator.items = hosts[:len(hosts)/2]
		errs1 := migrator.Migrate()
		migrator.items = hosts[len(hosts)/2:]
		errs2 := migrator.Migrate()

		// Assert
		require.Empty(t, errs1)
		require.Empty(t, errs2)
	})

	t.Run("daemon failed to lock", func(t *testing.T) {
		// Arrange
		daemon := createDaemon()
		migrator.items = []dbmodel.Host{createHost(daemon)}

		expectDaemonLock(daemon, errors.New("daemon lock error"))

		// Act
		errs := migrator.Migrate()

		// Assert
		require.Len(t, errs, 1)
		require.EqualValues(t, daemon.ID, errs[0].ID)
		require.ErrorContains(t, errs[0].Error, "daemon lock error")
		require.EqualValues(t, getExpectedLabel(errs[0]), errs[0].Label)
		require.Equal(t, configmigrator.ErrorCauseEntityDaemon, errs[0].CauseEntity)
	})
}

// Test that the hosts are loaded and counted correctly.
func TestLoadAndCountItems(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	lookup := dbmodel.NewDHCPOptionDefinitionLookup()

	agentMock := NewMockConnectedAgents(ctrl)
	daemonLockerMock := NewMockDaemonLocker(ctrl)
	statePullerMock := NewMockPauser(ctrl)
	hostPullerMock := NewMockPauser(ctrl)

	// Add 20 hosts to the database.
	for i := 0; i < 22; i++ {
		host := &dbmodel.Host{
			HostIdentifiers: []dbmodel.HostIdentifier{
				{
					Type:  "hw-address",
					Value: []byte{byte(i)},
				},
			},
		}

		err := dbmodel.AddHost(db, host)
		require.NoError(t, err)
	}

	migrator := NewHostMigrator(
		dbmodel.HostsByPageFilters{},
		db, agentMock, lookup, daemonLockerMock,
		statePullerMock, hostPullerMock,
	).(*hostMigrator)

	migrator.limit = 5

	t.Run("count total", func(t *testing.T) {
		migrator.totalItemsLoaded = 0

		// Act
		total, err := migrator.CountTotal()

		// Assert
		require.NoError(t, err)
		require.EqualValues(t, 22, total)
	})

	t.Run("load items", func(t *testing.T) {
		migrator.totalItemsLoaded = 0

		// Act
		loaded, err := migrator.LoadItems()

		// Assert
		require.NoError(t, err)
		require.EqualValues(t, 5, loaded)
		require.Len(t, migrator.items, 5)
		for i, host := range migrator.items {
			require.EqualValues(t, i, host.HostIdentifiers[0].Value[0])
		}
	})

	t.Run("paginate", func(t *testing.T) {
		var allLoaded []dbmodel.Host
		migrator.totalItemsLoaded = 0

		// Act
		for {
			loaded, err := migrator.LoadItems()
			require.NoError(t, err)

			if loaded == 0 {
				break
			}

			require.EqualValues(t, loaded, len(migrator.items))
			allLoaded = append(allLoaded, migrator.items...)
		}

		// Assert
		require.EqualValues(t, 22, len(allLoaded))
		for i, host := range allLoaded {
			require.EqualValues(t, i, host.HostIdentifiers[0].Value[0])
		}
	})
}

// Test that the hosts are loaded and counted correctly when the filter is
// applied.
func TestLoadAndCountItemsWithFilter(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	lookup := dbmodel.NewDHCPOptionDefinitionLookup()

	agentMock := NewMockConnectedAgents(ctrl)
	daemonLockerMock := NewMockDaemonLocker(ctrl)
	statePullerMock := NewMockPauser(ctrl)
	hostPullerMock := NewMockPauser(ctrl)

	// Even hosts belong to the subnet, odd hosts don't.
	subnet := &dbmodel.Subnet{Prefix: "10.0.0.0/8"}
	err := dbmodel.AddSubnet(db, subnet)
	require.NoError(t, err)

	// Add 20 hosts to the database.
	for i := 0; i < 22; i++ {
		host := &dbmodel.Host{
			HostIdentifiers: []dbmodel.HostIdentifier{
				{
					Type:  "hw-address",
					Value: []byte{byte(i)},
				},
			},
		}
		if i%2 == 0 {
			host.SubnetID = subnet.ID
		}

		err := dbmodel.AddHost(db, host)
		require.NoError(t, err)
	}

	migrator := NewHostMigrator(
		dbmodel.HostsByPageFilters{
			SubnetID: storkutil.Ptr(subnet.ID),
		},
		db, agentMock, lookup, daemonLockerMock,
		statePullerMock, hostPullerMock,
	).(*hostMigrator)

	migrator.limit = 5

	t.Run("count total", func(t *testing.T) {
		migrator.totalItemsLoaded = 0

		// Act
		total, err := migrator.CountTotal()

		// Assert
		require.NoError(t, err)
		require.EqualValues(t, 11, total)
	})

	t.Run("load items", func(t *testing.T) {
		migrator.totalItemsLoaded = 0

		// Act
		loaded, err := migrator.LoadItems()

		// Assert
		require.NoError(t, err)
		require.EqualValues(t, 5, loaded)
		require.Len(t, migrator.items, 5)
		for i, host := range migrator.items {
			require.EqualValues(t, 2*i, host.HostIdentifiers[0].Value[0])
		}
	})

	t.Run("paginate", func(t *testing.T) {
		var allLoaded []dbmodel.Host
		migrator.totalItemsLoaded = 0

		// Act
		for {
			loaded, err := migrator.LoadItems()
			require.NoError(t, err)

			if loaded == 0 {
				break
			}

			require.EqualValues(t, loaded, len(migrator.items))
			allLoaded = append(allLoaded, migrator.items...)
		}

		// Assert
		require.EqualValues(t, 11, len(allLoaded))
		for i, host := range allLoaded {
			require.EqualValues(t, 2*i, host.HostIdentifiers[0].Value[0])
		}
	})
}

// Test that the begin function pauses the host puller.
func TestHostMigrationBegin(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	lookup := dbmodel.NewDHCPOptionDefinitionLookup()
	agentMock := NewMockConnectedAgents(ctrl)
	daemonLockerMock := NewMockDaemonLocker(ctrl)
	statePullerMock := NewMockPauser(ctrl)
	hostPullerMock := NewMockPauser(ctrl)

	migrator := NewHostMigrator(
		dbmodel.HostsByPageFilters{},
		nil, agentMock, lookup, daemonLockerMock,
		statePullerMock, hostPullerMock,
	).(*hostMigrator)

	gomock.InOrder(
		statePullerMock.EXPECT().Pause(),
		hostPullerMock.EXPECT().Pause(),
	)

	// Act
	err := migrator.Begin()

	// Assert
	require.NoError(t, err)
}

// Test that the end function resumes the host puller.
func TestHostMigrationEnd(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	lookup := dbmodel.NewDHCPOptionDefinitionLookup()
	agentMock := NewMockConnectedAgents(ctrl)
	daemonLockerMock := NewMockDaemonLocker(ctrl)
	statePuller := NewMockPauser(ctrl)
	hostPullerMock := NewMockPauser(ctrl)

	migrator := NewHostMigrator(
		dbmodel.HostsByPageFilters{},
		nil, agentMock, lookup, daemonLockerMock,
		statePuller, hostPullerMock,
	).(*hostMigrator)

	statePuller.EXPECT().Unpause()
	hostPullerMock.EXPECT().Unpause()
	daemonLockerMock.EXPECT().Unlock(
		gomock.Eq(config.LockKey(42)),
		gomock.Eq(int64(24)),
	)

	// Act
	migrator.lockedDemonIDs = map[int64]config.LockKey{
		24: config.LockKey(42),
	}
	err := migrator.End()

	// Assert
	require.NoError(t, err)
}
