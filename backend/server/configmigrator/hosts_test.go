package configmigrator

import (
	"bytes"
	"encoding/binary"
	"testing"

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
	expectReservationAddCommandWithError := func(daemon *dbmodel.Daemon, host *dbmodel.Host, result *agentcomm.KeaCmdsResult) {
		reservation, _ := keaconfig.CreateHostCmdsReservation(
			daemon.ID, lookup, host,
		)

		agentMock.EXPECT().ForwardToKeaOverHTTP(
			gomock.Any(),          // Context.
			gomock.Eq(daemon.App), // App.
			gomock.Eq([]keactrl.SerializableCommand{
				keactrl.NewCommandReservationAdd(
					reservation, daemon.Name,
				),
			}), // Commands.
			gomock.Any(), // Responses.
		).Return(result, nil)
	}

	expectReservationAddCommandNoError := func(daemon *dbmodel.Daemon, host *dbmodel.Host) {
		expectReservationAddCommandWithError(daemon, host, &agentcomm.KeaCmdsResult{})
	}

	expectReservationDelCommandWithError := func(daemon *dbmodel.Daemon, host *dbmodel.Host, result *agentcomm.KeaCmdsResult) {
		deletedReservation, _ := keaconfig.CreateHostCmdsDeletedReservation(
			daemon.ID, host, keaconfig.HostCmdsOperationTargetMemory,
		)
		agentMock.EXPECT().ForwardToKeaOverHTTP(
			gomock.Any(),          // Context.
			gomock.Eq(daemon.App), // App.
			gomock.Eq([]keactrl.SerializableCommand{
				keactrl.NewCommandReservationDel(
					deletedReservation, daemon.Name,
				),
			}), // Commands.
			gomock.Any(), // Responses.
		).Return(result, nil)
	}

	expectReservationDelCommandNoError := func(daemon *dbmodel.Daemon, host *dbmodel.Host) {
		expectReservationDelCommandWithError(daemon, host, &agentcomm.KeaCmdsResult{})
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

	createHost := func(daemons ...*dbmodel.Daemon) *dbmodel.Host {
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

		host := &dbmodel.Host{
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
	_ = daemon2

	t.Run("single host, single daemon, all OK", func(t *testing.T) {
		host := createHost(daemon1)

		expectReservationAddCommandNoError(daemon1, host)
		expectReservationDelCommandNoError(daemon1, host)
		expectConfigWriteCommandNoError(daemon1)

		migrator.items = []dbmodel.Host{*host}

		// Act
		errs := migrator.Migrate()

		// Assert
		require.Empty(t, errs)
	})

	t.Run("inactive daemon", func(t *testing.T) {
		host := createHost(inactiveDaemon)

		migrator.items = []dbmodel.Host{*host}

		// Act
		errs := migrator.Migrate()

		// Assert
		require.Empty(t, errs)
	})
}
