package config

import (
	"context"
	"time"

	"github.com/go-pg/pg/v10"
	pkgerrors "github.com/pkg/errors"
	keaconfig "isc.org/stork/appcfg/kea"
	"isc.org/stork/datamodel"
	agentcomm "isc.org/stork/server/agentcomm"
	dbmodel "isc.org/stork/server/database/model"
)

var _ TransactionStateAccessor = (*TransactionState[any])(nil)

// An interface implemented by the TransactionState[T any]. It is used
// to convert the config updates having the specific Recipe type (T) to
// the config updates having the Recipe type any. It is used in the
// config manager's functions that don't need to operate on any specific
// Recipe type.
type TransactionStateAccessor interface {
	// Returns config updates with the Recipe type any.
	GetUpdates() []*Update[any]
}

// A structure describing a single configuration update that may be applied
// to multiple daemons. The T type is the Recipe type. Having this type
// generic allows for using the Update structure for configuring different
// app types.
type Update[T any] struct {
	// Type of the configured daemon, e.g. "kea".
	Target datamodel.AppType
	// Type of the operation to perform, e.g. "host_add".
	Operation string
	// Identifiers of the daemons affected by the update. For example,
	// a host reservation can be shared by two daemons.
	DaemonIDs []int64
	// Holds information required to apply the config update, e.g.
	// commands to be sent to the configured server, information to be
	// inserted into the database etc. The contents of this field are
	// specific to the performed operation.
	Recipe T
}

// A structure describing a single configuration change. It includes one or more
// configuration updates.
type TransactionState[T any] struct {
	// A flag indicating if the state has been re-created from the information
	// stored in the database (scheduled configuration change).
	Scheduled bool
	// Configuration updates belonging to the transaction.
	Updates []*Update[T]
}

// A type representing a configuration lock key.
type LockKey int64

// A generic structure associating an object (entity) with a
// database ID.
//
// Configurations passed to the Config Manager must be associated with
// the daemons they belong to, so that the Config Manager knows which
// config should be sent to which daemon. The keaconfig package has no
// notion of the daemon ID, so the structures representing Kea
// configurations do not contain these identifiers. This structure can be
// used to make associations between any object and an ID. It is used
// for making associations described above, but it is generic and can be
// used for making associations between any object and an ID.
type AnnotatedEntity[T any] struct {
	id     int64
	entity T
}

// Instantiates new AnnotatedEntity.
func NewAnnotatedEntity[T any](id int64, entity T) *AnnotatedEntity[T] {
	return &AnnotatedEntity[T]{
		id:     id,
		entity: entity,
	}
}

// Returns entity database ID.
func (ae AnnotatedEntity[T]) GetID() int64 {
	return ae.id
}

// Returns the annotated entity.
func (ae AnnotatedEntity[T]) GetEntity() T {
	return ae.entity
}

// Interface of the Kea configuration module.
type KeaModule interface {
	BeginGlobalParametersUpdate(context.Context, []int64) (context.Context, error)
	ApplyGlobalParametersUpdate(context.Context, []AnnotatedEntity[*keaconfig.SettableConfig]) (context.Context, error)
	BeginHostAdd(context.Context) (context.Context, error)
	ApplyHostAdd(context.Context, *dbmodel.Host) (context.Context, error)
	BeginHostUpdate(context.Context, int64) (context.Context, error)
	ApplyHostUpdate(context.Context, *dbmodel.Host) (context.Context, error)
	BeginHostDelete(context.Context) (context.Context, error)
	ApplyHostDelete(context.Context, *dbmodel.Host) (context.Context, error)
	BeginSharedNetworkAdd(context.Context) (context.Context, error)
	ApplySharedNetworkAdd(context.Context, *dbmodel.SharedNetwork) (context.Context, error)
	BeginSharedNetworkUpdate(context.Context, int64) (context.Context, error)
	ApplySharedNetworkUpdate(context.Context, *dbmodel.SharedNetwork) (context.Context, error)
	ApplySharedNetworkDelete(context.Context, *dbmodel.SharedNetwork) (context.Context, error)
	BeginSubnetAdd(context.Context) (context.Context, error)
	ApplySubnetAdd(context.Context, *dbmodel.Subnet) (context.Context, error)
	BeginSubnetUpdate(context.Context, int64) (context.Context, error)
	ApplySubnetUpdate(context.Context, *dbmodel.Subnet) (context.Context, error)
	ApplySubnetDelete(context.Context, *dbmodel.Subnet) (context.Context, error)
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

// Configuration manager interface exposing functions used for getting
// state information, (e.g., database connection, connected agents etc.).
type ManagerAccessors interface {
	// Returns an instance of the database handler used by the configuration manager.
	GetDB() *pg.DB
	// Returns an interface to the agents the manager communicates with.
	GetConnectedAgents() agentcomm.ConnectedAgents
	// Returns an interface to the instance providing the DHCP option definition
	// lookup logic.
	GetDHCPOptionDefinitionLookup() keaconfig.DHCPOptionDefinitionLookup
}

// Configuration manager interface exposing functions available to the
// configuration modules.
type ModuleManager interface {
	ManagerLocker
	ManagerAccessors
}

// Config manager interface exposing locking functions.
type ManagerLocker interface {
	// Locks the daemons' configurations for update.
	Lock(context.Context, ...int64) (context.Context, error)
	// Unlocks the daemons' configurations.
	Unlock(context.Context)
}

// Returns config updates with the Recipe type any. This function belongs
// to the TransactionStateAccessor interface.
func (state TransactionState[T]) GetUpdates() (updates []*Update[any]) {
	for _, u := range state.Updates {
		update := Update[any]{
			Target:    u.Target,
			Operation: u.Operation,
			DaemonIDs: u.DaemonIDs,
			Recipe:    u.Recipe,
		}
		updates = append(updates, &update)
	}
	return
}

// Creates new config update instance.
func NewUpdate[T any](target datamodel.AppType, operation string, daemonIDs ...int64) *Update[T] {
	return &Update[T]{
		Target:    target,
		Operation: operation,
		DaemonIDs: daemonIDs,
	}
}

// Creates new transaction state with one config update instance. It is
// the most typical use case.
func NewTransactionStateWithUpdate[T any](target datamodel.AppType, operation string, daemonIDs ...int64) *TransactionState[T] {
	update := NewUpdate[T](target, operation, daemonIDs...)
	state := &TransactionState[T]{}
	state.Updates = append(state.Updates, update)
	return state
}

// Sets a recipe in the transaction state for a given update index. It returns
// an error if the specified index is out of bounds.
func (state *TransactionState[T]) SetRecipeForUpdate(updateIndex int, recipe *T) error {
	if len(state.Updates) <= updateIndex {
		return pkgerrors.Errorf("transaction state update index %d is out of bounds", updateIndex)
	}
	state.Updates[updateIndex].Recipe = *recipe
	return nil
}

// Returns an update recipe for a specified update index. It returns an error
// if the index is out of bounds.
func (state *TransactionState[T]) GetRecipeForUpdate(updateIndex int) (*T, error) {
	if len(state.Updates) <= updateIndex {
		return nil, pkgerrors.Errorf("transaction state update index %d is out of bounds", updateIndex)
	}
	return &state.Updates[updateIndex].Recipe, nil
}
