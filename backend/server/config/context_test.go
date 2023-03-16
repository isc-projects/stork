package config

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"isc.org/stork/datamodel"
)

// Test configuration recipe used in the tests.
type testRecipe struct {
	param string
}

// Test convenience function returning an int64 value from the context.
func TestGetValueAsInt64(t *testing.T) {
	ctx := context.WithValue(context.Background(), ContextIDKey, int64(1234))
	ctx = context.WithValue(ctx, StateContextKey, "test")

	// This value exists and should cast to int64.
	value1, ok := GetValueAsInt64(ctx, ContextIDKey)
	require.True(t, ok)
	require.EqualValues(t, 1234, value1)

	// This value exists but it doesn't cast to int64.
	_, ok = GetValueAsInt64(ctx, StateContextKey)
	require.False(t, ok)

	// This value doesn't exist.
	_, ok = GetValueAsInt64(ctx, UserContextKey)
	require.False(t, ok)
}

// Test convenience function returning transaction state.
func TestGetTransactionState(t *testing.T) {
	state := TransactionState[testRecipe]{
		Scheduled: true,
	}
	ctx := context.WithValue(context.Background(), StateContextKey, state)
	returned, ok := GetTransactionState[testRecipe](ctx)
	require.True(t, ok)
	require.True(t, returned.Scheduled)
}

// Test convenience function returning transaction state when the
// state doesn't exist.
func TestGetTransactionStateNoState(t *testing.T) {
	_, ok := GetTransactionState[testRecipe](context.Background())
	require.False(t, ok)
}

// Test convenience function returning transaction state when the
// state has invalid type.
func TestGetTransactionStateNoCast(t *testing.T) {
	ctx := context.WithValue(context.Background(), StateContextKey, "a string")
	_, ok := GetTransactionState[testRecipe](ctx)
	require.False(t, ok)
}

// Test convenience function returning transaction state.
func TestGetAnyTransactionState(t *testing.T) {
	state := TransactionState[testRecipe]{
		Scheduled: true,
		Updates: []*Update[testRecipe]{
			{
				Recipe: testRecipe{
					param: "foo",
				},
			},
		},
	}
	ctx := context.WithValue(context.Background(), StateContextKey, state)
	returned, ok := GetAnyTransactionState(ctx)
	require.True(t, ok)
	require.Len(t, returned.GetUpdates(), 1)
}

// Test convenience function returning transaction state when the
// state doesn't exist.
func TestGetAnyTransactionStateNoState(t *testing.T) {
	_, ok := GetAnyTransactionState(context.Background())
	require.False(t, ok)
}

// Test convenience function returning transaction state when the
// state has invalid type.
func TestGetAnyTransactionStateNoCast(t *testing.T) {
	ctx := context.WithValue(context.Background(), StateContextKey, "a string")
	_, ok := GetAnyTransactionState(ctx)
	require.False(t, ok)
}

// Test setting and getting recipe for an update in the transaction state.
func TestSetRecipeForUpdateInContext(t *testing.T) {
	state := NewTransactionStateWithUpdate[testRecipe](datamodel.AppTypeKea, "host_update", 1)
	ctx := context.WithValue(context.Background(), StateContextKey, *state)

	recipe := testRecipe{
		param: "foo",
	}
	ctx, err := SetRecipeForUpdate(ctx, 0, &recipe)
	require.NoError(t, err)

	returnedState, ok := GetTransactionState[testRecipe](ctx)
	require.True(t, ok)
	returnedRecipe, err := returnedState.GetRecipeForUpdate(0)
	require.NoError(t, err)
	require.NotNil(t, returnedRecipe)
	require.Equal(t, "foo", returnedRecipe.param)
}

// Test that an error is returned when trying to set a recipe for update in the
// state when the state does not exist.
func TestSetValueForUpdateInContextNoState(t *testing.T) {
	ctx := context.Background()
	recipe := testRecipe{}
	_, err := SetRecipeForUpdate(ctx, 0, &recipe)
	require.Error(t, err)
}

// Test that an error is returned when trying to set a recipe for update in the
// state when update index is out of bounds.
func TestSetValueForUpdateInContextIndexOutOfBounds(t *testing.T) {
	recipe := testRecipe{}
	state := NewTransactionStateWithUpdate[testRecipe](datamodel.AppTypeKea, "host_update", 1)
	ctx := context.WithValue(context.Background(), StateContextKey, *state)
	_, err := SetRecipeForUpdate(ctx, 1, &recipe)
	require.Error(t, err)
}

// Test getting a recipe for update from the context.
func TestGetValueForUpdateInContext(t *testing.T) {
	state := NewTransactionStateWithUpdate[testRecipe](datamodel.AppTypeKea, "host_update", 1)
	ctx := context.WithValue(context.Background(), StateContextKey, *state)

	recipe := testRecipe{
		param: "foo",
	}
	ctx, err := SetRecipeForUpdate(ctx, 0, &recipe)
	require.NoError(t, err)

	returnedRecipe, err := GetRecipeForUpdate[testRecipe](ctx, 0)
	require.NoError(t, err)
	require.Equal(t, "foo", returnedRecipe.param)
}

// Test that an error is returned when trying to get a recipe for update from the
// context when the state does not exist.
func TestGetValueForUpdateInContextNoState(t *testing.T) {
	value, err := GetRecipeForUpdate[any](context.Background(), 0)
	require.Error(t, err)
	require.Nil(t, value)
}

// Test that an error is returned when trying to get a recipe for update from the
// context when update index is out of bounds.
func TestGetValueForUpdateInContextIndexOutOfBounds(t *testing.T) {
	state := NewTransactionStateWithUpdate[testRecipe](datamodel.AppTypeKea, "host_update", 1)
	ctx := context.WithValue(context.Background(), StateContextKey, *state)

	recipe := testRecipe{
		param: "foo",
	}
	ctx, err := SetRecipeForUpdate(ctx, 0, &recipe)
	require.NoError(t, err)

	returnedRecipe, err := GetRecipeForUpdate[testRecipe](ctx, 1)
	require.Error(t, err)
	require.Nil(t, returnedRecipe)
}
