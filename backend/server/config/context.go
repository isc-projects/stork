package config

import (
	"context"

	pkgerrors "github.com/pkg/errors"
)

// Type of the context keys used by a config manager. The manager and the
// functions it calls use the golang context to pass the data around and hold
// the critical information for the config change transactions. The context
// keys point to various types of information held in the context.
type ContextKey int

const (
	// A context key for getting a config update state.
	StateContextKey ContextKey = iota
	// A context key for accessing context ID for the config change transaction.
	ContextIDKey
	// A context key for accessing user ID for the config change transaction.
	UserContextKey
	// A context key for accessing a lock for the config change transaction.
	LockContextKey
	// A context key for accessing a list of daemon IDs.
	DaemonsContextKey
)

// Convenience function retrieving a value from the context. If the context
// doesn't contain the specified key, the second returned parameter is false.
func GetValueAsInt64(ctx context.Context, key ContextKey) (value int64, ok bool) {
	value, ok = ctx.Value(key).(int64)
	return
}

// Convenience function retrieving a transaction state from the context. If
// the context doesn't contain the transaction state, the second returned
// parameter is false.
func GetTransactionState(ctx context.Context) (state TransactionState, ok bool) {
	state, ok = ctx.Value(StateContextKey).(TransactionState)
	return
}

// Sets a value in the transaction state for a given update index, under the
// specified name in the recipe. It returns an error if the context does not
// contain a transaction state or the specified index is out of bounds. It
// always returns a context with an updated value if the value has been
// successfully set.
func SetValueForUpdate(ctx context.Context, updateIndex int, valueName string, value any) (context.Context, error) {
	state, ok := GetTransactionState(ctx)
	if !ok {
		return ctx, pkgerrors.New("transaction state does not exist in the context")
	}
	if err := state.SetValueForUpdate(updateIndex, valueName, value); err != nil {
		return ctx, err
	}
	return context.WithValue(ctx, StateContextKey, state), nil
}
