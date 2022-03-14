package config

import (
	"context"
	"time"

	"github.com/go-pg/pg/v10"
	dbmodel "isc.org/stork/server/database/model"
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

// A structure describing a single configuration update that may be applied
// to multiple daemons.
type Update = dbmodel.ConfigUpdate

// A structure describing the recipe for running a configuration update. The
// recipe is specific to certain config update operation, e.g. adding a host,
// editing a config, updating a subnet etc.
type UpdateRecipe = dbmodel.ConfigUpdateRecipe

// A structure describing a single configuration change. It includes one or more
// configuration updates.
type TransactionState struct {
	// A flag indicating if the state has been re-created from the information
	// stored in the database (scheduled configuration change).
	Scheduled bool
	// Configuration updates belonging to the transaction.
	Updates []Update
}

// A type representing a configuration lock key.
type LockKey int64

// Interface of the Kea configuration module.
type KeaModule interface {
	BeginHostAdd(context.Context) (context.Context, error)
	ApplyHostAdd(context.Context, *dbmodel.Host) (context.Context, error)
	CommitHostAdd(context.Context) (context.Context, error)
}

// Interface of the Kea configuration module used by the manager to
// commit configuration changes in Kea servers.
type KeaModuleCommit interface {
	Commit(context.Context) (context.Context, error)
}

// Common configuration manager Interface.
type Manager interface {
	// Returns an instance of the database handler used by the configuration manager.
	GetDB() *pg.DB
	// Returns Kea configuration module.
	GetKeaModule() KeaModule
	// Creates new context for applying configuration changes.
	CreateContext(int64) (context.Context, error)
	// Stores the context in the manager for later use.
	RememberContext(context.Context, time.Duration) error
	// Returns stored context for a given context and user ID.
	RecoverContext(int64, int64) (context.Context, context.CancelFunc)
	// Locks the daemons' configurations for update.
	Lock(context.Context, ...int64) (context.Context, error)
	// Unlocks the daemons' configurations.
	Unlock(context.Context)
	// Cancels the config update transaction.
	Done(context.Context)
	// Sends configuration changes to the daemons.
	Commit(context.Context) (context.Context, error)
	// Sends scheduled configuration changes to the daemons.
	CommitDue() error
	// Schedules configuration changes to apply them in the future.
	Schedule(context.Context, time.Time) (context.Context, error)
}
