package dbmodel

import (
	"testing"

	require "github.com/stretchr/testify/require"
	dbtest "isc.org/stork/server/database/test"
)

// Test inserting, selecting and deleting configuration reports
// shared by two daemons.
func TestConfigReportSharingDaemons(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Add a machine.
	machine := &Machine{
		Address:   "localhost",
		AgentPort: 8080,
	}
	err := AddMachine(db, machine)
	require.NoError(t, err)

	// Add an app with two daemons.
	app := &App{
		Type:      AppTypeKea,
		MachineID: machine.ID,
		Daemons: []*Daemon{
			NewKeaDaemon("dhcp4", true),
			NewKeaDaemon("dhcp6", true),
		},
	}
	daemons, err := AddApp(db, app)
	require.NoError(t, err)
	require.Len(t, daemons, 2)

	// Add a configuration report shared by both daemons.
	configReport := &ConfigReport{
		ProducerName: "test",
		Content:      "Here is the test report",
		DaemonID:     daemons[0].ID,
		RefDaemons:   daemons,
	}
	err = AddConfigReport(db, configReport)
	require.NoError(t, err)

	// Try to get the configuration report.
	configReports, err := GetConfigReportsByDaemonID(db, daemons[0].ID)
	require.NoError(t, err)
	require.Len(t, configReports, 1)
	require.NotZero(t, configReports[0].DaemonID)
	require.Len(t, configReports[0].RefDaemons, 2)
	require.Equal(t, "dhcp4", configReports[0].RefDaemons[0].Name)
	require.NotNil(t, configReports[0].RefDaemons[0].App)
	require.Equal(t, "dhcp6", configReports[0].RefDaemons[1].Name)
	require.NotNil(t, configReports[0].RefDaemons[1].App)
	require.Equal(t, "Here is the test report", configReports[0].Content)

	// Delete the configuration report.
	err = DeleteConfigReportsByDaemonID(db, daemons[0].ID)
	require.NoError(t, err)

	// The report is no longer returned.
	configReports, err = GetConfigReportsByDaemonID(db, daemons[0].ID)
	require.NoError(t, err)
	require.Empty(t, configReports)
}

// Test inserting, selecting and deleting configuration reports associated
// with distinct daemons.
func TestConfigReportDistinctDaemons(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Add a machine.
	machine := &Machine{
		Address:   "localhost",
		AgentPort: 8080,
	}
	err := AddMachine(db, machine)
	require.NoError(t, err)

	// Add an app.
	app := &App{
		Type:      AppTypeKea,
		MachineID: machine.ID,
		Daemons: []*Daemon{
			NewKeaDaemon("dhcp4", true),
			NewKeaDaemon("dhcp6", true),
		},
	}
	daemons, err := AddApp(db, app)
	require.NoError(t, err)
	require.Len(t, daemons, 2)

	// Associate configuration reports with distinct daemons.
	configReports := []ConfigReport{
		{
			ProducerName: "test",
			Content:      "Here is the first test report",
			DaemonID:     daemons[0].ID,
			RefDaemons: []*Daemon{
				daemons[0],
			},
		},
		{
			ProducerName: "test",
			Content:      "Here is the second test report",
			DaemonID:     daemons[1].ID,
			RefDaemons: []*Daemon{
				daemons[1],
			},
		},
	}
	for i := range configReports {
		err = AddConfigReport(db, &configReports[i])
		require.NoError(t, err)
	}

	// Select configuration reports for both daemons.
	for i, daemon := range daemons {
		returnedConfigReports, err := GetConfigReportsByDaemonID(db, daemon.ID)
		require.NoError(t, err)
		require.Len(t, returnedConfigReports, 1)
		require.Len(t, returnedConfigReports[0].RefDaemons, 1)
		require.NotNil(t, returnedConfigReports[0].RefDaemons[0].App)
		require.Equal(t, configReports[i].Content, returnedConfigReports[0].Content)
	}

	// Delete configuration reports for the first daemon.
	err = DeleteConfigReportsByDaemonID(db, daemons[0].ID)
	require.NoError(t, err)

	// The configuration report for the first daemon no longer exists.
	configReports, err = GetConfigReportsByDaemonID(db, daemons[0].ID)
	require.NoError(t, err)
	require.Empty(t, configReports)

	// It should not affect the report for the second daemon.
	configReports, err = GetConfigReportsByDaemonID(db, daemons[1].ID)
	require.NoError(t, err)
	require.Len(t, configReports, 1)
}

// Test different cases of malformed configuration reports.
func TestInvalidConfigReport(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Add a machine.
	machine := &Machine{
		Address:   "localhost",
		AgentPort: 8080,
	}
	err := AddMachine(db, machine)
	require.NoError(t, err)

	// Add an app.
	app := &App{
		Type:      AppTypeKea,
		MachineID: machine.ID,
		Daemons: []*Daemon{
			NewKeaDaemon("dhcp4", true),
		},
	}
	daemons, err := AddApp(db, app)
	require.NoError(t, err)
	require.Len(t, daemons, 1)

	testCases := []string{
		"empty producer name",
		"empty contents",
		"invalid daemon id",
	}
	configReports := []*ConfigReport{
		{
			ProducerName: "",
			Content:      "Here is the first test report",
			DaemonID:     daemons[0].ID,
			RefDaemons: []*Daemon{
				daemons[0],
			},
		},
		{
			ProducerName: "test",
			Content:      "",
			DaemonID:     daemons[0].ID,
			RefDaemons: []*Daemon{
				daemons[0],
			},
		},
		{
			ProducerName: "test",
			Content:      "contents",
			DaemonID:     111111,
			RefDaemons: []*Daemon{
				daemons[0],
			},
		},
	}

	for i, tc := range testCases {
		tcIndex := i
		t.Run(tc, func(t *testing.T) {
			err = AddConfigReport(db, configReports[tcIndex])
			require.Error(t, err)
		})
	}
}

// This test verifies that deleting configuration reports for non-existing
// daemon does not return an error.
func TestConfigReportDeleteNonExisting(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	err := DeleteConfigReportsByDaemonID(db, 12345)
	require.NoError(t, err)
}
