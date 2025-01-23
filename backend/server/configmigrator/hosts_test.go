package configmigrator

import (
	"bytes"
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
	expectReservationAddCommandWithError := func(daemon *dbmodel.Daemon, result *agentcomm.KeaCmdsResult, err error, hosts ...dbmodel.Host) {
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
		).Return(result, err)
	}

	expectReservationAddCommandNoError := func(daemon *dbmodel.Daemon, hosts ...dbmodel.Host) {
		expectReservationAddCommandWithError(daemon, &agentcomm.KeaCmdsResult{}, nil, hosts...)
	}

	expectReservationDelCommandWithError := func(daemon *dbmodel.Daemon, result *agentcomm.KeaCmdsResult, hosts ...dbmodel.Host) {
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
		).Return(result, nil)
	}

	expectReservationDelCommandNoError := func(daemon *dbmodel.Daemon, hosts ...dbmodel.Host) {
		expectReservationDelCommandWithError(daemon, &agentcomm.KeaCmdsResult{}, hosts...)
	}

	expectConfigWriteCommandWithError := func(daemon *dbmodel.Daemon, result *agentcomm.KeaCmdsResult) {
		agentMock.EXPECT().ForwardToKeaOverHTTP(
			gomock.Any(),          // Context.
			gomock.Eq(daemon.App), // App.
			gomock.Eq([]keactrl.SerializableCommand{
				keactrl.NewCommandBase(keactrl.ConfigWrite, daemon.Name),
			}), // Commands.
			gomock.Any(), // Responses.
		).Return(result, nil)
	}

	expectConfigWriteCommandNoError := func(daemon *dbmodel.Daemon) {
		expectConfigWriteCommandWithError(daemon, &agentcomm.KeaCmdsResult{})
	}

	// Entities.
	nextHostID := int64(1)
	nextLocalHostID := int64(1)

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

	inactiveDaemon := &dbmodel.Daemon{
		ID:     0,
		Name:   dbmodel.DaemonNameDHCPv4,
		App:    &dbmodel.App{},
		Active: false,
	}

	daemon1 := &dbmodel.Daemon{
		ID:     1,
		Name:   dbmodel.DaemonNameDHCPv4,
		App:    &dbmodel.App{},
		Active: true,
	}

	daemon2 := &dbmodel.Daemon{
		ID:     2,
		Name:   dbmodel.DaemonNameDHCPv4,
		App:    &dbmodel.App{},
		Active: true,
	}

	daemon3 := &dbmodel.Daemon{
		ID:     3,
		Name:   dbmodel.DaemonNameDHCPv4,
		App:    &dbmodel.App{},
		Active: true,
	}

	_ = daemon2

	// Tests migrating a single host with no errors.
	t.Run("single host, single daemon, all OK", func(t *testing.T) {
		host := createHost(daemon1)

		expectReservationAddCommandNoError(daemon1, host)
		expectReservationDelCommandNoError(daemon1, host)
		expectConfigWriteCommandNoError(daemon1)

		migrator.items = []dbmodel.Host{host}

		// Act
		errs := migrator.Migrate()

		// Assert
		require.Empty(t, errs)
	})

	// Tests that the inactive daemon is skipped and generates no API calls.
	t.Run("inactive daemon", func(t *testing.T) {
		host := createHost(inactiveDaemon)

		migrator.items = []dbmodel.Host{host}

		// Act
		errs := migrator.Migrate()

		// Assert
		require.Empty(t, errs)
	})

	// Tests that the migration doesn't fail for no hosts.
	t.Run("no hosts", func(t *testing.T) {
		migrator.items = []dbmodel.Host{}

		// Act
		errs := migrator.Migrate()

		// Assert
		require.Empty(t, errs)
	})

	// Tests migrating multiple hosts belonging to a single daemon. The hosts
	// should be added to host database and deleted from the configuration in
	// a single batch.
	t.Run("multiple hosts, single daemons, all OK", func(t *testing.T) {
		hosts := []dbmodel.Host{
			createHost(daemon1),
			createHost(daemon1),
			createHost(daemon1),
			createHost(daemon1),
		}

		expectReservationAddCommandNoError(daemon1, hosts...)
		expectReservationDelCommandNoError(daemon1, hosts...)
		expectConfigWriteCommandNoError(daemon1)

		migrator.items = hosts

		// Act
		errs := migrator.Migrate()

		// Assert
		require.Empty(t, errs)
	})

	// Tests migrating multiple hosts belonging to multiple daemons. The hosts
	// should be processed in separate batches for each daemon.
	t.Run("multiple hosts, multiple daemons, all OK", func(t *testing.T) {
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
	})

	// Test that if the error occurs during adding a reservation to the
	// database, it isn't removed from the configuration.
	t.Run("error adding reservation", func(t *testing.T) {
		host1 := createHost(daemon1)
		host2 := createHost(daemon1)
		host3 := createHost(daemon2)
		host4 := createHost(daemon2)
		host5 := createHost(daemon3)
		host6 := createHost(daemon3)

		// Kea CA returns an error.
		expectReservationAddCommandWithError(daemon1, &agentcomm.KeaCmdsResult{
			Error: errors.Errorf("error adding reservation"),
		}, nil, host1, host2)
		expectConfigWriteCommandNoError(daemon1)

		// The Stork agent return an error.
		expectReservationAddCommandWithError(daemon2,
			&agentcomm.KeaCmdsResult{},
			errors.Errorf("error transferring reservation"),
			host3, host4,
		)
		expectConfigWriteCommandNoError(daemon2)

		// The Kea daemon returns an error while processing the add command.
		expectReservationAddCommandWithError(daemon3, &agentcomm.KeaCmdsResult{
			CmdsErrors: []error{nil, errors.Errorf("error executing command")},
		}, nil, host5, host6)
		expectReservationDelCommandNoError(daemon3, host5)
		expectConfigWriteCommandNoError(daemon3)

		migrator.items = []dbmodel.Host{host1, host2, host3, host4, host5, host6}

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
	})
}
