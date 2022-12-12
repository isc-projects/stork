package config

import (
	"testing"

	"github.com/stretchr/testify/require"
	dbmodel "isc.org/stork/server/database/model"
)

// Test AppTag interface implementation.
func TestAppTag(t *testing.T) {
	app := App{
		ID:   11,
		Name: "kea@xyz",
		Type: dbmodel.AppTypeKea,
		Meta: AppMeta{
			Version: "2.1.1",
		},
		Machine: Machine{
			ID: 42,
		},
	}
	require.EqualValues(t, 11, app.GetID())
	require.Equal(t, "kea@xyz", app.GetName())
	require.Equal(t, dbmodel.AppTypeKea, app.GetType())
	require.Equal(t, "2.1.1", app.GetVersion())
	require.EqualValues(t, 42, app.GetMachineID())
}

// Test that getting machine ID is safe even if the machine reference is not
// set explicitly.
func TestAppTagGetMachineIDForDefaultMachine(t *testing.T) {
	// Arrange
	app := App{}

	// Act & Assert
	require.Zero(t, app.GetMachineID())
}

// Test getting control access point.
func TestGetControlAccessPoint(t *testing.T) {
	app := &App{}

	// An error should be returned when there is no control access point.
	address, port, key, secure, err := app.GetControlAccessPoint()
	require.Error(t, err)
	require.Empty(t, address)
	require.Zero(t, port)
	require.Empty(t, key)
	require.False(t, secure)

	// Specify control access point and check it is returned.
	app.AccessPoints = []AccessPoint{
		{
			Type:              dbmodel.AccessPointControl,
			Address:           "cool.example.org",
			Port:              1234,
			Key:               "key",
			UseSecureProtocol: true,
		},
	}
	address, port, key, secure, err = app.GetControlAccessPoint()
	require.NoError(t, err)
	require.Equal(t, "cool.example.org", address)
	require.EqualValues(t, 1234, port)
	require.Equal(t, "key", key)
	require.True(t, secure)
}

// Test getting MachineTag interface from an app.
func TestGetMachineTag(t *testing.T) {
	app := App{
		Machine: Machine{
			ID: 10,
		},
	}
	machine := app.GetMachineTag()
	require.NotNil(t, machine)
	require.EqualValues(t, 10, machine.GetID())
}

// Test getting DaemonTag interfaces from an app.
func TestGetDaemonTags(t *testing.T) {
	app := App{
		Daemons: []Daemon{
			{
				ID: 10,
			},
			{
				ID: 11,
			},
		},
		Machine: Machine{ID: 42},
	}
	daemons := app.GetDaemonTags()
	require.Len(t, daemons, 2)
	require.EqualValues(t, 10, daemons[0].GetID())
	require.EqualValues(t, 11, daemons[1].GetID())
	require.NotNil(t, daemons[1].GetMachineID())
	require.EqualValues(t, 42, *daemons[1].GetMachineID())
}
