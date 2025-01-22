package configmigrator

import (
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
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	agentMock := NewMockConnectedAgents(ctrl)
	lookup := dbmodel.NewDHCPOptionDefinitionLookup()

	daemon1 := &dbmodel.Daemon{
		ID:     1,
		Name:   dbmodel.DaemonNameDHCPv4,
		App:    &dbmodel.App{},
		Active: true,
	}

	hostInDaemon1 := &dbmodel.Host{
		ID:       1,
		SubnetID: 42,
		HostIdentifiers: []dbmodel.HostIdentifier{
			{
				ID:     1,
				Type:   "hw-address",
				Value:  []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06},
				HostID: 1,
			},
		},
		LocalHosts: []dbmodel.LocalHost{
			{
				ID:         1,
				DaemonID:   daemon1.ID,
				Daemon:     daemon1,
				DataSource: dbmodel.HostDataSourceConfig,
			},
		},
	}

	migrator := NewHostMigrator(
		dbmodel.HostsByPageFilters{}, nil, agentMock,
		dbmodel.NewDHCPOptionDefinitionLookup(),
	).(*hostMigrator)

	t.Run("single host, single daemon, all OK", func(t *testing.T) {
		reservation, _ := keaconfig.CreateHostCmdsReservation(
			daemon1.ID, lookup, hostInDaemon1,
		)

		agentMock.EXPECT().ForwardToKeaOverHTTP(
			gomock.Any(),           // Context.
			gomock.Eq(daemon1.App), // App.
			gomock.Eq([]keactrl.SerializableCommand{
				keactrl.NewCommandReservationAdd(
					reservation, dbmodel.DaemonNameDHCPv4,
				),
			}), // Commands.
			gomock.Any(), // Responses.
		).Return(&agentcomm.KeaCmdsResult{}, nil)

		deletedReservation, _ := keaconfig.CreateHostCmdsDeletedReservation(
			daemon1.ID, hostInDaemon1, keaconfig.HostCmdsOperationTargetMemory,
		)
		agentMock.EXPECT().ForwardToKeaOverHTTP(
			gomock.Any(),           // Context.
			gomock.Eq(daemon1.App), // App.
			gomock.Eq([]keactrl.SerializableCommand{
				keactrl.NewCommandReservationDel(
					deletedReservation, "dhcp4",
				),
			}), // Commands.
			gomock.Any(), // Responses.
		).Return(&agentcomm.KeaCmdsResult{}, nil)

		agentMock.EXPECT().ForwardToKeaOverHTTP(
			gomock.Any(),           // Context.
			gomock.Eq(daemon1.App), // App.
			gomock.Eq([]keactrl.SerializableCommand{
				keactrl.NewCommandBase(keactrl.ConfigWrite, "dhcp4"),
			}), // Commands.
			gomock.Any(), // Responses.
		).Return(&agentcomm.KeaCmdsResult{}, nil)

		migrator.items = []dbmodel.Host{*hostInDaemon1}

		// Act
		errs := migrator.Migrate()

		// Assert
		require.Empty(t, errs)
	})
}
