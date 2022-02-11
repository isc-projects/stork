package kea

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"isc.org/stork/server/config"
	dbmodel "isc.org/stork/server/database/model"
)

// Test Kea module commit function.
func TestCommit(t *testing.T) {
	module := NewConfigModule(nil)
	require.NotNil(t, module)

	ctx := context.Background()

	_, err := module.Commit(ctx)
	require.Error(t, err)
}

// Test first stage of adding a new host.
func TestBeginHostAdd(t *testing.T) {
	module := NewConfigModule(nil)
	require.NotNil(t, module)

	ctx1 := context.Background()
	ctx2, err := module.BeginHostAdd(ctx1)
	require.NoError(t, err)
	require.Equal(t, ctx1, ctx2)
}

// Test second stage of adding a new host.
func TestApplyHostAdd(t *testing.T) {
	module := NewConfigModule(nil)
	require.NotNil(t, module)

	ctx := context.Background()

	host := &dbmodel.Host{}
	_, err := module.ApplyHostAdd(ctx, host)
	require.NoError(t, err)
}

// Test committing added host.
func TestCommitHostAdd(t *testing.T) {
	module := NewConfigModule(nil)
	require.NotNil(t, module)

	ctx := context.Background()

	host := &dbmodel.Host{}
	state := config.TransactionState{
		Updates: []config.Update{
			{
				Recipe: config.UpdateRecipe{
					Host: host,
				},
			},
		},
	}
	ctx = context.WithValue(ctx, config.StateContextKey, state)

	_, err := module.CommitHostAdd(ctx)
	require.NoError(t, err)
}
