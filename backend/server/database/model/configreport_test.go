package dbmodel

import (
	"fmt"
	"testing"

	require "github.com/stretchr/testify/require"
	dbtest "isc.org/stork/server/database/test"
)

// Returns a pointer to a given value. It is helpful to create a pointer from
// the constants.
func newPtr[T any](value T) *T {
	return &value
}

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
		CheckerName: "test",
		Content:     newPtr("Here is the test report for {daemon}, {daemon} and {daemon}"),
		DaemonID:    daemons[0].ID,
		RefDaemons:  daemons,
	}
	err = AddConfigReport(db, configReport)
	require.NoError(t, err)

	// Add an empty configuration report.
	configReport = &ConfigReport{
		CheckerName: "empty",
		Content:     nil,
		DaemonID:    daemons[0].ID,
		RefDaemons:  []*Daemon{},
	}
	err = AddConfigReport(db, configReport)
	require.NoError(t, err)

	// Try to get the configuration report.
	configReports, total, err := GetConfigReportsByDaemonID(db, 0, 0, daemons[0].ID, false)
	require.NoError(t, err)
	require.EqualValues(t, 2, total)
	require.Len(t, configReports, 2)
	require.NotZero(t, configReports[0].DaemonID)
	require.Len(t, configReports[0].RefDaemons, 2)
	require.Equal(t, "dhcp4", configReports[0].RefDaemons[0].Name)
	require.NotNil(t, configReports[0].RefDaemons[0].App)
	require.Equal(t, "dhcp6", configReports[0].RefDaemons[1].Name)
	require.NotNil(t, configReports[0].RefDaemons[1].App)
	require.Equal(t, "Here is the test report for <daemon id=\"1\" name=\"dhcp4\" appId=\"1\" appType=\"kea\">, <daemon id=\"2\" name=\"dhcp6\" appId=\"1\" appType=\"kea\"> and {daemon}", *configReports[0].Content)
	require.Equal(t, "empty", configReports[1].CheckerName)
	require.Nil(t, configReports[1].Content)

	// Delete the configuration report.
	err = DeleteConfigReportsByDaemonID(db, daemons[0].ID)
	require.NoError(t, err)

	// The report is no longer returned.
	configReports, total, err = GetConfigReportsByDaemonID(db, 0, 0, daemons[0].ID, false)
	require.NoError(t, err)
	require.Zero(t, total)
	require.Empty(t, configReports)
}

// Test that the config reports containing no issues are skipped if a specific
// flag is provided.
func TestGetConfigReportsExceptEmpty(t *testing.T) {
	// Arrange
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
		},
	}
	daemons, err := AddApp(db, app)
	require.NoError(t, err)
	require.Len(t, daemons, 1)

	// Add a configuration report shared by both daemons.
	configReport := &ConfigReport{
		CheckerName: "test",
		Content:     newPtr("Here is the test report for {daemon}, {daemon} and {daemon}"),
		DaemonID:    daemons[0].ID,
		RefDaemons:  daemons,
	}
	err = AddConfigReport(db, configReport)
	require.NoError(t, err)

	// Add an empty configuration report.
	configReport = &ConfigReport{
		CheckerName: "empty",
		Content:     nil,
		DaemonID:    daemons[0].ID,
		RefDaemons:  []*Daemon{},
	}
	err = AddConfigReport(db, configReport)
	require.NoError(t, err)
	// Act
	reports, total, err := GetConfigReportsByDaemonID(db, 0, 10, daemons[0].ID, true)

	// Assert
	require.Len(t, reports, 1)
	require.EqualValues(t, 1, total)
	require.NoError(t, err)
	require.EqualValues(t, "test", reports[0].CheckerName)
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
			CheckerName: "test",
			Content:     newPtr("Here is the first test report"),
			DaemonID:    daemons[0].ID,
			RefDaemons: []*Daemon{
				daemons[0],
			},
		},
		{
			CheckerName: "test",
			Content:     newPtr("Here is the second test report"),
			DaemonID:    daemons[1].ID,
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
		returnedConfigReports, total, err := GetConfigReportsByDaemonID(db, 0, 0, daemon.ID, false)
		require.NoError(t, err)
		require.EqualValues(t, 1, total)
		require.Len(t, returnedConfigReports, 1)
		require.Len(t, returnedConfigReports[0].RefDaemons, 1)
		require.NotNil(t, returnedConfigReports[0].RefDaemons[0].App)
		require.Equal(t, configReports[i].Content, returnedConfigReports[0].Content)
	}

	// Delete configuration reports for the first daemon.
	err = DeleteConfigReportsByDaemonID(db, daemons[0].ID)
	require.NoError(t, err)

	// The configuration report for the first daemon no longer exists.
	configReports, total, err := GetConfigReportsByDaemonID(db, 0, 0, daemons[0].ID, false)
	require.NoError(t, err)
	require.Zero(t, total)
	require.Empty(t, configReports)

	// It should not affect the report for the second daemon.
	configReports, total, err = GetConfigReportsByDaemonID(db, 0, 0, daemons[1].ID, false)
	require.NoError(t, err)
	require.EqualValues(t, 1, total)
	require.Len(t, configReports, 1)
}

// Test getting the configuration reports with paging.
func TestConfigReportsPaging(t *testing.T) {
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

	// Add several configuration reports.
	for i := 0; i < 10; i++ {
		configReport := &ConfigReport{
			CheckerName: fmt.Sprintf("test%d", i),
			Content:     newPtr(fmt.Sprintf("Here is the test report no %d", i)),
			DaemonID:    daemons[0].ID,
			RefDaemons:  daemons,
		}
		err = AddConfigReport(db, configReport)
		require.NoError(t, err)
	}

	// When specifying the offset and limit of 0, all reports should be returned.
	configReports, total, err := GetConfigReportsByDaemonID(db, 0, 0, daemons[0].ID, false)
	require.NoError(t, err)
	require.EqualValues(t, 10, total)
	require.Len(t, configReports, 10)
	// Put the report IDs into the map to make sure that all reports were returned.
	allReportIDs := make(map[int64]bool)
	prevID := int64(0)
	for _, r := range configReports {
		allReportIDs[r.ID] = true
		// Make sure that reports are ordered by ID.
		require.Greater(t, r.ID, prevID)
		prevID = r.ID
	}
	require.Len(t, allReportIDs, 10)

	// Get the reports from the first to fifth.
	configReports, total, err = GetConfigReportsByDaemonID(db, 0, 5, daemons[0].ID, false)
	require.NoError(t, err)
	require.EqualValues(t, 10, total)
	require.Len(t, configReports, 5)
	// Store report IDs in another map and make sure we have 5 different IDs.
	pagedReportIDs := make(map[int64]bool)
	prevID = int64(0)
	for _, r := range configReports {
		pagedReportIDs[r.ID] = true
		// Make sure that reports are ordered by ID.
		require.Greater(t, r.ID, prevID)
		prevID = r.ID
	}
	require.Len(t, pagedReportIDs, 5)

	// Get the reports from fifth to the last one.
	configReports, total, err = GetConfigReportsByDaemonID(db, 5, 10, daemons[0].ID, false)
	require.NoError(t, err)
	require.EqualValues(t, 10, total)
	require.Len(t, configReports, 5)
	// Store all new IDs in the map.
	for _, r := range configReports {
		pagedReportIDs[r.ID] = true
		// Make sure that reports are ordered by ID.
		require.Greater(t, r.ID, prevID)
		prevID = r.ID
	}
	// Make sure that we have all reports returned with paging.
	require.Len(t, pagedReportIDs, 10)
	require.Equal(t, allReportIDs, pagedReportIDs)
}

// Test that the config reports are counted properly for different filters.
func TestCountConfigReports(t *testing.T) {
	// Arrange
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
		},
	}
	daemons, err := AddApp(db, app)
	require.NoError(t, err)
	require.Len(t, daemons, 1)

	// Add some empty reports
	for i := 0; i < 10; i++ {
		configReport := &ConfigReport{
			CheckerName: fmt.Sprintf("empty %d", i),
			Content:     nil,
			DaemonID:    daemons[0].ID,
			RefDaemons:  []*Daemon{},
		}

		err = AddConfigReport(db, configReport)
		require.NoError(t, err)
	}

	// Add some reports with issues
	for i := 0; i < 10; i++ {
		configReport := &ConfigReport{
			CheckerName: fmt.Sprintf("checker %d", i),
			Content:     newPtr(fmt.Sprintf("content %d", i)),
			DaemonID:    daemons[0].ID,
			RefDaemons:  []*Daemon{},
		}

		err = AddConfigReport(db, configReport)
		require.NoError(t, err)
	}

	// Act
	totalReports, reportsErr := CountConfigReportsByDaemonID(db, daemons[0].ID, false)
	totalIssues, issuesErr := CountConfigReportsByDaemonID(db, daemons[0].ID, true)

	// Assert
	require.NoError(t, reportsErr)
	require.NoError(t, issuesErr)

	require.EqualValues(t, 20, totalReports)
	require.EqualValues(t, 10, totalIssues)
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
		"empty checker name",
		"empty content",
		"invalid daemon id",
	}
	configReports := []*ConfigReport{
		{
			CheckerName: "",
			Content:     newPtr("Here is the first test report"),
			DaemonID:    daemons[0].ID,
			RefDaemons: []*Daemon{
				daemons[0],
			},
		},
		{
			CheckerName: "test",
			Content:     newPtr(""),
			DaemonID:    daemons[0].ID,
			RefDaemons: []*Daemon{
				daemons[0],
			},
		},
		{
			CheckerName: "test",
			Content:     newPtr("contents"),
			DaemonID:    111111,
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

// This test verifies that it is possible to delete a daemon
// having configuration reviews.
func TestDeleteAppWithConfigReview(t *testing.T) {
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

	// Add config report for the daemon.
	configReport := &ConfigReport{
		CheckerName: "test",
		Content:     newPtr("Here is the first test report"),
		DaemonID:    daemons[0].ID,
		RefDaemons: []*Daemon{
			daemons[0],
		},
	}
	err = AddConfigReport(db, configReport)
	require.NoError(t, err)

	// Make sure we can delete an app. The associated config
	// reports should be deleted.
	err = DeleteApp(db, app)
	require.NoError(t, err)
}
