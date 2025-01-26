package dbmodel

import (
	"testing"

	"github.com/stretchr/testify/require"
	dbtest "isc.org/stork/server/database/test"
)

type testError struct{}

func (err testError) Error() string {
	return "test error"
}

func TestNewZoneInventoryStateDetails(t *testing.T) {
	details := NewZoneInventoryStateDetails()
	require.Equal(t, ZoneInventoryStatusOK, details.Status)
	require.Nil(t, details.Error)
	require.Nil(t, details.ZoneCount)

	state := NewZoneInventoryState(15, details)
	require.NotNil(t, state)
	require.EqualValues(t, 15, state.DaemonID)
	require.Zero(t, state.CreatedAt)
	require.Equal(t, details, state.State)
}

func TestZoneInventoryStateDetailsSetStatus(t *testing.T) {
	details := NewZoneInventoryStateDetails()
	details.SetStatus(ZoneInventoryStatusBusy, testError{})
	require.Equal(t, ZoneInventoryStatusBusy, details.Status)
	require.NotNil(t, details.Error)
	require.Equal(t, "test error", *details.Error)
	details.SetStatus(ZoneInventoryStatusOK, nil)
	require.Equal(t, ZoneInventoryStatusOK, details.Status)
	require.Nil(t, details.Error)
}

func TestZoneInventoryStateDetailsSetTotalZones(t *testing.T) {
	details := NewZoneInventoryStateDetails()
	details.SetTotalZones(10)
	require.NotNil(t, details.ZoneCount)
	require.EqualValues(t, 10, *details.ZoneCount)
}

func TestNewZoneInventoryState(t *testing.T) {
	details := NewZoneInventoryStateDetails()
	state := NewZoneInventoryState(15, details)
	require.NotNil(t, state)
	require.EqualValues(t, 15, state.DaemonID)
	require.Zero(t, state.CreatedAt)
	require.Equal(t, details, state.State)
}

func TestAddZoneInventoryState(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	machine := &Machine{
		ID:        0,
		Address:   "localhost",
		AgentPort: int64(8080),
	}
	err := AddMachine(db, machine)
	require.NoError(t, err)

	app := &App{
		ID:        0,
		MachineID: machine.ID,
		Type:      AppTypeBind9,
		Daemons: []*Daemon{
			NewBind9Daemon(true),
		},
	}
	addedDaemons, err := AddApp(db, app)
	require.NoError(t, err)
	require.Len(t, addedDaemons, 1)

	details := NewZoneInventoryStateDetails()
	details.SetStatus(ZoneInventoryStatusBusy, testError{})
	details.SetTotalZones(123)
	state := NewZoneInventoryState(addedDaemons[0].ID, details)

	err = AddZoneInventoryState(db, state)
	require.NoError(t, err)

	returnedState, err := GetZoneInventoryState(db, addedDaemons[0].ID)
	require.NoError(t, err)
	require.NotNil(t, returnedState)
	require.NotZero(t, returnedState.ID)
	require.Equal(t, addedDaemons[0].ID, returnedState.DaemonID)
	require.NotZero(t, returnedState.CreatedAt)
	require.NotNil(t, returnedState.State)
	require.Equal(t, ZoneInventoryStatusBusy, returnedState.State.Status)
	require.NotNil(t, returnedState.State.Error)
	require.Equal(t, "test error", *returnedState.State.Error)
	require.NotNil(t, returnedState.State.ZoneCount)
	require.EqualValues(t, 123, *returnedState.State.ZoneCount)

	returnedState, err = GetZoneInventoryState(db, addedDaemons[0].ID+1)
	require.NoError(t, err)
	require.Nil(t, returnedState)
}

func TestAddZoneInventoryStateOverride(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	machine := &Machine{
		ID:        0,
		Address:   "localhost",
		AgentPort: int64(8080),
	}
	err := AddMachine(db, machine)
	require.NoError(t, err)

	app := &App{
		ID:        0,
		MachineID: machine.ID,
		Type:      AppTypeBind9,
		Daemons: []*Daemon{
			NewBind9Daemon(true),
		},
	}
	addedDaemons, err := AddApp(db, app)
	require.NoError(t, err)
	require.Len(t, addedDaemons, 1)

	details := NewZoneInventoryStateDetails()
	details.SetStatus(ZoneInventoryStatusBusy, testError{})
	details.SetTotalZones(123)
	state := NewZoneInventoryState(addedDaemons[0].ID, details)

	err = AddZoneInventoryState(db, state)
	require.NoError(t, err)

	state.State.SetStatus(ZoneInventoryStatusOK, nil)
	state.State.SetTotalZones(234)
	err = AddZoneInventoryState(db, state)
	require.NoError(t, err)

	returnedState, err := GetZoneInventoryState(db, addedDaemons[0].ID)
	require.NoError(t, err)
	require.NotNil(t, returnedState)
	require.NotZero(t, returnedState.ID)
	require.Equal(t, addedDaemons[0].ID, returnedState.DaemonID)
	require.NotZero(t, returnedState.CreatedAt)
	require.NotNil(t, returnedState.State)
	require.Equal(t, ZoneInventoryStatusOK, returnedState.State.Status)
	require.Nil(t, returnedState.State.Error)
	require.NotNil(t, returnedState.State.ZoneCount)
	require.EqualValues(t, 234, *returnedState.State.ZoneCount)
}
