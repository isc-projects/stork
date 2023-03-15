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
func GetTransactionState[T any](ctx context.Context) (state TransactionState[T], ok bool) {
	state, ok = ctx.Value(StateContextKey).(TransactionState[T])
	return
}

// Convenience function retrieving an interface to the transaction state
// from the context. The interface provides the GetUpdates() function
// returning the config updates using the generic interface. It should
// be used in cases when the type of the config update recipe doesn't
// matter.
func GetAnyTransactionState(ctx context.Context) (state TransactionStateAccessor, ok bool) {
	state, ok = ctx.Value(StateContextKey).(TransactionStateAccessor)
	return
}

// Gets a value from the transaction state for a given update index, under the
// specified name in the recipe. It returns an error if the specified index
// is out of bounds or when the value doesn't exist.
func GetRecipeForUpdate[T any](ctx context.Context, updateIndex int) (*T, error) {
	state, ok := GetTransactionState[T](ctx)
	if !ok {
		return nil, pkgerrors.New("transaction state does not exist in the context")
	}
	return state.GetRecipeForUpdate(updateIndex)
}

// Sets a value in the transaction state for a given update index, under the
// specified name in the recipe. It returns an error if the context does not
// contain a transaction state or the specified index is out of bounds. It
// always returns a context with an updated value if the value has been
// successfully set.
func SetRecipeForUpdate[T any](ctx context.Context, updateIndex int, recipe *T) (context.Context, error) {
	state, ok := GetTransactionState[T](ctx)
	if !ok {
		return ctx, pkgerrors.New("transaction state does not exist in the context")
	}
	if err := state.SetRecipeForUpdate(updateIndex, recipe); err != nil {
		return ctx, err
	}
	return context.WithValue(ctx, StateContextKey, state), nil
}
