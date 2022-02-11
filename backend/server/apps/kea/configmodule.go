package kea

import (
	"context"

	pkgerrors "github.com/pkg/errors"
	config "isc.org/stork/server/config"
	dbmodel "isc.org/stork/server/database/model"
)

// A configuration manager module responsible for Kea configuration.
type ConfigModule struct {
	// A configuration manager owning the module.
	manager config.Manager
}

// Creates new instance of the Kea configuration module.
func NewConfigModule(manager config.Manager) *ConfigModule {
	return &ConfigModule{
		manager: manager,
	}
}

// Commits the Kea configuration changes.
func (kea *ConfigModule) Commit(ctx context.Context) (context.Context, error) {
	var err error
	state, ok := ctx.Value(config.StateContextKey).(config.TransactionState)
	if !ok {
		return ctx, pkgerrors.Errorf("context lacks state")
	}
	for _, pu := range state.Updates {
		switch pu.Operation {
		case "host_add":
			ctx, err = kea.CommitHostAdd(ctx)
		default:
			err = pkgerrors.Errorf("unknown operation %s when called Commit()", pu.Operation)
		}
		if err != nil {
			return ctx, err
		}
	}
	return ctx, err
}

// Begins adding a new host reservation.
func (kea *ConfigModule) BeginHostAdd(ctx context.Context) (context.Context, error) {
	return ctx, nil
}

// Applies new host reservation.
func (kea *ConfigModule) ApplyHostAdd(ctx context.Context, host *dbmodel.Host) (context.Context, error) {
	u := config.Update{
		Target:    "kea",
		Operation: "host_add",
		Recipe: config.UpdateRecipe{
			Host: host,
		},
	}
	state := config.TransactionState{}
	state.Updates = append(state.Updates, u)
	ctx = context.WithValue(ctx, config.StateContextKey, state)

	return ctx, nil
}

// Create the host reservation in the Kea servers.
func (kea *ConfigModule) CommitHostAdd(ctx context.Context) (context.Context, error) {
	state, ok := ctx.Value(config.StateContextKey).(config.TransactionState)
	if !ok {
		return ctx, pkgerrors.Errorf("called CommitHostAdd() without a state")
	}
	for _, update := range state.Updates {
		var host *dbmodel.Host
		if host = update.Recipe.Host; host == nil {
			return ctx, pkgerrors.Errorf("missing host in the context")
		}
	}
	return ctx, nil
}
