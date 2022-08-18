package config

import (
	"context"
	"time"

	"github.com/go-pg/pg/v10"
	"github.com/mitchellh/mapstructure"
	pkgerrors "github.com/pkg/errors"
	agentcomm "isc.org/stork/server/agentcomm"
	dbmodel "isc.org/stork/server/database/model"
)

// A structure describing a single configuration update that may be applied
// to multiple daemons.
type Update = dbmodel.ConfigUpdate

// A structure describing a single configuration change. It includes one or more
// configuration updates.
type TransactionState struct {
	// A flag indicating if the state has been re-created from the information
	// stored in the database (scheduled configuration change).
	Scheduled bool
	// Configuration updates belonging to the transaction.
	Updates []*Update
}

// A type representing a configuration lock key.
type LockKey int64

// Interface of the Kea configuration module.
type KeaModule interface {
	BeginHostAdd(context.Context) (context.Context, error)
	ApplyHostAdd(context.Context, *dbmodel.Host) (context.Context, error)
	BeginHostUpdate(context.Context, int64) (context.Context, error)
	ApplyHostUpdate(context.Context, *dbmodel.Host) (context.Context, error)
	BeginHostDelete(context.Context) (context.Context, error)
	ApplyHostDelete(context.Context, *dbmodel.Host) (context.Context, error)
}

// Interface of the Kea configuration module used by the manager to
// commit configuration changes in Kea servers.
type KeaModuleCommit interface {
	Commit(context.Context) (context.Context, error)
}

// Common configuration manager interface.
type Manager interface {
	ManagerLocker
	// Returns Kea configuration module.
	GetKeaModule() KeaModule
	// Creates new context for applying configuration changes.
	CreateContext(int64) (context.Context, error)
	// Stores the context in the manager for later use.
	RememberContext(context.Context, time.Duration) error
	// Returns stored context for a given context and user ID.
	RecoverContext(int64, int64) (context.Context, context.CancelFunc)
	// Cancels the config update transaction.
	Done(context.Context)
	// Sends configuration changes to the daemons.
	Commit(context.Context) (context.Context, error)
	// Sends scheduled configuration changes to the daemons.
	CommitDue() error
	// Schedules configuration changes to apply them in the future.
	Schedule(context.Context, time.Time) (context.Context, error)
}

// Configuration manager interface exposing functions accessing
// its unexported fields. It is used by the config modules.
type ManagerAccessors interface {
	ManagerLocker
	// Returns an instance of the database handler used by the configuration manager.
	GetDB() *pg.DB
	// Returns an interface to the agents the manager communicates with.
	GetConnectedAgents() agentcomm.ConnectedAgents
}

// Config manager interface exposing locking functions.
type ManagerLocker interface {
	// Locks the daemons' configurations for update.
	Lock(context.Context, ...int64) (context.Context, error)
	// Unlocks the daemons' configurations.
	Unlock(context.Context)
}

// Creates new config update instance.
func NewUpdate(target, operation string, daemonIDs ...int64) *Update {
	return dbmodel.NewConfigUpdate(target, operation, daemonIDs...)
}

// Creates new transaction state with one config update instance. It is
// the most typical use case.
func NewTransactionStateWithUpdate(target, operation string, daemonIDs ...int64) *TransactionState {
	update := NewUpdate(target, operation, daemonIDs...)
	state := &TransactionState{
		Updates: []*Update{
			update,
		},
	}
	return state
}

// Sets a value in the transaction state for a given update index, under the
// specified name in the recipe. It returns an error if the specified index
// is out of bounds.
func (state *TransactionState) SetValueForUpdate(updateIndex int, valueName string, value any) error {
	if len(state.Updates) <= updateIndex {
		return pkgerrors.Errorf("transaction state update index %d is out of bounds", updateIndex)
	}
	state.Updates[updateIndex].Recipe[valueName] = value
	return nil
}

// Gets a value from the transaction state for a given update index, under the
// specified name in the recipe. It returns an error if the specified index
// is out of bounds or when the value doesn't exist.
func (state *TransactionState) GetValueForUpdate(updateIndex int, valueName string) (any, error) {
	if len(state.Updates) <= updateIndex {
		return nil, pkgerrors.Errorf("transaction state update index %d is out of bounds", updateIndex)
	}
	value, ok := state.Updates[updateIndex].Recipe[valueName]
	if !ok {
		return nil, pkgerrors.Errorf("value %s does not exist for update with index %d", valueName, updateIndex)
	}
	return value, nil
}

// Decodes data stored as a map in the context/transaction into a custom structure.
func DecodeContextData(input interface{}, output interface{}) error {
	err := mapstructure.Decode(input, output)
	return pkgerrors.WithStack(err)
}
