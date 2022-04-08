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
