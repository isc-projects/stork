package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Test creating new config update instance.
func TestNewUpdate(t *testing.T) {
	cu := NewUpdate("kea", "host_add", 1, 2, 3)
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
	state := NewTransactionStateWithUpdate("keax", "host_update", 2, 3)
	require.NotNil(t, state)
	require.Len(t, state.Updates, 1)
	cu := state.Updates[0]
	require.Equal(t, "keax", cu.Target)
	require.Equal(t, "host_update", cu.Operation)
	require.Len(t, cu.DaemonIDs, 2)
	require.Contains(t, cu.DaemonIDs, int64(2))
	require.Contains(t, cu.DaemonIDs, int64(3))
}

// Test setting and getting value for an update in the transaction state.
func TestSetValueForUpdate(t *testing.T) {
	state := TransactionState{
		Updates: []*Update{},
	}
	for i := 0; i < 5; i++ {
		update := NewUpdate("kea", "host_update", int64(i))
		state.Updates = append(state.Updates, update)
	}
	err := state.SetValueForUpdate(2, "foo", "bar")
	require.NoError(t, err)
	err = state.SetValueForUpdate(4, "foobar", "baz")
	require.NoError(t, err)

	v, err := state.GetValueForUpdate(2, "foo")
	require.NoError(t, err)
	require.NotNil(t, v)
	require.Equal(t, "bar", v)

	v, err = state.GetValueForUpdate(4, "foobar")
	require.NoError(t, err)
	require.NotNil(t, v)
	require.Equal(t, "baz", v)

	// Test error cases.
	v, err = state.GetValueForUpdate(4, "foo")
	require.Error(t, err)
	require.Nil(t, v)

	v, err = state.GetValueForUpdate(0, "bba")
	require.Error(t, err)
	require.Nil(t, v)

	v, err = state.GetValueForUpdate(7, "foo")
	require.Error(t, err)
	require.Nil(t, v)
}

// Test that the DecodeContextData decodes a map into structure.
func TestDecodeContextData(t *testing.T) {
	input := map[string]interface{}{
		"foo": "bar",
	}
	output := struct {
		Foo string
	}{}
	err := DecodeContextData(input, &output)
	require.NoError(t, err)
	require.Equal(t, "bar", output.Foo)
}
