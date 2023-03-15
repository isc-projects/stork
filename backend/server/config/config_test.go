package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Test creating new config update instance.
func TestNewUpdate(t *testing.T) {
	cu := NewUpdate[any]("kea", "host_add", 1, 2, 3)
	require.NotNil(t, cu)
	require.Equal(t, "kea", cu.Target)
	require.Equal(t, "host_add", cu.Operation)
	require.Len(t, cu.DaemonIDs, 3)
	require.Contains(t, cu.DaemonIDs, int64(1))
	require.Contains(t, cu.DaemonIDs, int64(2))
	require.Contains(t, cu.DaemonIDs, int64(3))
}

// Test creating new transaction state instance with one update instance.
func TestNewTransactionStateWithUpdate(t *testing.T) {
	state := NewTransactionStateWithUpdate[any]("keax", "host_update", 2, 3)
	require.NotNil(t, state)
	require.Len(t, state.Updates, 1)
	cu := state.Updates[0]
	require.Equal(t, "keax", cu.Target)
	require.Equal(t, "host_update", cu.Operation)
	require.Len(t, cu.DaemonIDs, 2)
	require.Contains(t, cu.DaemonIDs, int64(2))
	require.Contains(t, cu.DaemonIDs, int64(3))
}

// Test setting and getting a recipe for an update in the transaction state.
func TestSetRecipeForUpdate(t *testing.T) {
	state := TransactionState[testRecipe]{
		Updates: []*Update[testRecipe]{},
	}
	for i := 0; i < 5; i++ {
		update := NewUpdate[testRecipe]("kea", "host_update", int64(i))
		state.Updates = append(state.Updates, update)
	}
	recipe := testRecipe{
		param: "foo",
	}
	err := state.SetRecipeForUpdate(2, &recipe)
	require.NoError(t, err)

	recipe = testRecipe{
		param: "bar",
	}
	err = state.SetRecipeForUpdate(4, &recipe)
	require.NoError(t, err)

	returnedRecipe, err := state.GetRecipeForUpdate(2)
	require.NoError(t, err)
	require.NotNil(t, returnedRecipe)
	require.Equal(t, "foo", returnedRecipe.param)

	returnedRecipe, err = state.GetRecipeForUpdate(4)
	require.NoError(t, err)
	require.NotNil(t, returnedRecipe)
	require.Equal(t, "bar", returnedRecipe.param)

	// Test error cases.
	returnedRecipe, err = state.GetRecipeForUpdate(8)
	require.Error(t, err)
	require.Nil(t, returnedRecipe)
}

// Test getting config updates from state with the recipe of any type.
func TestGetUpdates(t *testing.T) {
	state := TransactionState[testRecipe]{
		Updates: []*Update[testRecipe]{},
	}
	for i := 0; i < 5; i++ {
		update := NewUpdate[testRecipe]("kea", "host_update", int64(i))
		update.Recipe = testRecipe{
			param: "foo",
		}
		state.Updates = append(state.Updates, update)
	}
	anyUpdates := state.GetUpdates()
	require.Len(t, anyUpdates, 5)
	for i, u := range anyUpdates {
		require.Equal(t, "kea", u.Target)
		require.Equal(t, "host_update", u.Operation)
		require.Len(t, u.DaemonIDs, 1)
		require.EqualValues(t, i, u.DaemonIDs[0])
		require.IsType(t, testRecipe{}, u.Recipe)
		recipe := u.Recipe.(testRecipe)
		require.NotNil(t, recipe)
		require.Equal(t, "foo", recipe.param)
	}
}
