package dbmodel

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
	dbtest "isc.org/stork/server/database/test"
	storkutil "isc.org/stork/util"
)

// An error used in the unit tests.
type testError struct{}

// Returns an error as string.
func (err testError) Error() string {
	return "test error"
}

// Test creating a zone inventory state with the specified details.
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

// Test setting status in the state.
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

// Test setting total number of zones in the state.
func TestZoneInventoryStateDetailsSetTotalZones(t *testing.T) {
	details := NewZoneInventoryStateDetails()
	details.SetTotalZones(10)
	require.NotNil(t, details.ZoneCount)
	require.EqualValues(t, 10, *details.ZoneCount)
}

// Test instantiating new inventory state.
func TestNewZoneInventoryState(t *testing.T) {
	details := NewZoneInventoryStateDetails()
	state := NewZoneInventoryState(15, details)
	require.NotNil(t, state)
	require.EqualValues(t, 15, state.DaemonID)
	require.Zero(t, state.CreatedAt)
	require.Equal(t, details, state.State)
}

// Test adding zone inventory state to the database.
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

// Test adding and overriding zone inventory states.
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

// Test getting the zone inventory states from the database.
func TestGetZoneInventoryStates(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Create several states.
	details := []*ZoneInventoryStateDetails{
		{
			Status:    ZoneInventoryStatusBusy,
			Error:     storkutil.Ptr("busy error"),
			ZoneCount: storkutil.Ptr(int64(123)),
		},
		{
			Status:    ZoneInventoryStatusErred,
			Error:     storkutil.Ptr("other error"),
			ZoneCount: storkutil.Ptr(int64(234)),
		},
		{
			Status:    ZoneInventoryStatusUninitialized,
			Error:     storkutil.Ptr("uninitialized error"),
			ZoneCount: storkutil.Ptr(int64(345)),
		},
	}
	// Add the machines and apps and associate them with the states.
	for i := range details {
		machine := &Machine{
			Address:   "localhost",
			AgentPort: int64(8080 + i),
		}
		err := AddMachine(db, machine)
		require.NoError(t, err)

		app := &App{
			MachineID: machine.ID,
			Type:      AppTypeBind9,
			Daemons: []*Daemon{
				NewBind9Daemon(true),
			},
		}
		addedDaemons, err := AddApp(db, app)
		require.NoError(t, err)
		require.Len(t, addedDaemons, 1)

		state := NewZoneInventoryState(addedDaemons[0].ID, details[i])
		err = AddZoneInventoryState(db, state)
		require.NoError(t, err)
	}

	// Get the states from the database.
	states, count, err := GetZoneInventoryStates(db)
	require.NoError(t, err)
	require.Equal(t, 3, count)
	require.Len(t, states, 3)

	// Compare the returned states with the ones inserted to the database.
	for _, d := range details {
		index := slices.IndexFunc(states, func(state ZoneInventoryState) bool {
			return d.Status == state.State.Status
		})
		require.GreaterOrEqual(t, index, 0)
		require.Equal(t, d.Error, states[index].State.Error)
		require.Equal(t, d.ZoneCount, states[index].State.ZoneCount)
		require.Positive(t, states[index].DaemonID)
	}
}
