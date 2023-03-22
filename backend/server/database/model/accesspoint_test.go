package dbmodel

import (
	"testing"

	"github.com/stretchr/testify/require"
	dbtest "isc.org/stork/server/database/test"
)

// Test that the access point is appended properly.
func TestAppendAccessPoint(t *testing.T) {
	// Arrange
	var accessPoints []*AccessPoint

	// Act
	accessPoints = AppendAccessPoint(
		accessPoints,
		AccessPointControl,
		"42.42.42.42",
		"secret",
		4242,
		true,
	)

	// Assert
	require.Len(t, accessPoints, 1)
	require.EqualValues(t, "42.42.42.42", accessPoints[0].Address)
	require.Zero(t, accessPoints[0].AppID)
	require.EqualValues(t, "secret", accessPoints[0].Key)
	require.Zero(t, accessPoints[0].MachineID)
	require.EqualValues(t, 4242, accessPoints[0].Port)
	require.EqualValues(t, AccessPointControl, accessPoints[0].Type)
	require.True(t, accessPoints[0].UseSecureProtocol)
}

// Test that the no output and no error are returned if the entry is not found.
func TestGetAccessPointByIDForMissingEntry(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Act
	accessPoint, err := GetAccessPointByID(db, 42, AccessPointControl)

	// Assert
	require.NoError(t, err)
	require.Nil(t, accessPoint)
}

// Test that the error is returned if any database problem occurs.
func TestGetAccessPointByIDForInvalidDatabase(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)

	// Act
	teardown()
	accessPoint, err := GetAccessPointByID(db, 42, AccessPointControl)

	// Assert
	require.Error(t, err)
	require.Nil(t, accessPoint)
}

// Test that the access point is properly returned.
func TestGetAccessPointByID(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	machine := &Machine{Address: "localhost", AgentPort: 8080}
	_ = AddMachine(db, machine)
	app := &App{
		MachineID: machine.ID,
		Type:      AppTypeBind9,
		AccessPoints: []*AccessPoint{{
			Type:              AccessPointControl,
			Address:           "127.0.0.1",
			Port:              8080,
			Key:               "secret",
			UseSecureProtocol: true,
		}},
	}
	_, _ = AddApp(db, app)

	// Act
	accessPoint, err := GetAccessPointByID(db, app.ID, AccessPointControl)

	// Assert
	require.NoError(t, err)
	require.EqualValues(t, "127.0.0.1", accessPoint.Address)
	require.EqualValues(t, app.ID, accessPoint.AppID)
	require.EqualValues(t, "secret", accessPoint.Key)
	require.EqualValues(t, machine.ID, accessPoint.MachineID)
	require.EqualValues(t, 8080, accessPoint.Port)
	require.EqualValues(t, AccessPointControl, accessPoint.Type)
	require.True(t, accessPoint.UseSecureProtocol)
}
