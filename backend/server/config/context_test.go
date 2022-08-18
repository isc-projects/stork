package config

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

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
	state := TransactionState{
		Scheduled: true,
	}
	ctx := context.WithValue(context.Background(), StateContextKey, state)
	returned, ok := GetTransactionState(ctx)
	require.True(t, ok)
	require.True(t, returned.Scheduled)
}

// Test convenience function returning transaction state when the
// state doesn't exist.
func TestGetTransactionStateNoState(t *testing.T) {
	_, ok := GetTransactionState(context.Background())
	require.False(t, ok)
}

// Test convenience function returning transaction state when the
// state has invalid type.
func TestGetTransactionStateNoCast(t *testing.T) {
	ctx := context.WithValue(context.Background(), StateContextKey, "a string")
	_, ok := GetTransactionState(ctx)
	require.False(t, ok)
}

// Test setting and getting value for an update in the transaction state.
func TestSetValueForUpdateInContext(t *testing.T) {
	state := NewTransactionStateWithUpdate("kea", "host_update", 1)
	ctx := context.WithValue(context.Background(), StateContextKey, *state)

	ctx, err := SetValueForUpdate(ctx, 0, "foo", "bar")
	require.NoError(t, err)

	returnedState, ok := GetTransactionState(ctx)
	require.True(t, ok)
	v, err := returnedState.GetValueForUpdate(0, "foo")
	require.NoError(t, err)
	require.NotNil(t, v)
	require.Equal(t, "bar", v)
}

// Test that an error is returned when trying to set a value for update in the
// state when the state does not exist.
func TestSetValueForUpdateInContextNoState(t *testing.T) {
	ctx := context.Background()
	_, err := SetValueForUpdate(ctx, 0, "foo", "bar")
	require.Error(t, err)
}

// Test that an error is returned when trying to set a value for update in the
// state when update index is out of bounds.
func TestSetValueForUpdateInContextIndexOutOfBounds(t *testing.T) {
	state := NewTransactionStateWithUpdate("kea", "host_update", 1)
	ctx := context.WithValue(context.Background(), StateContextKey, *state)
	_, err := SetValueForUpdate(ctx, 1, "foo", "bar")
	require.Error(t, err)
}
