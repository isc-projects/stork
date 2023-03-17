package apps

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"math"
	"math/big"
	"sync"
	"time"

	"github.com/go-pg/pg/v10"
	pkgerrors "github.com/pkg/errors"
	keaconfig "isc.org/stork/appcfg/kea"
	"isc.org/stork/datamodel"
	"isc.org/stork/server/agentcomm"
	"isc.org/stork/server/apps/kea"
	"isc.org/stork/server/config"
	dbmodel "isc.org/stork/server/database/model"
)

// Represents a configuration lock for a user.
type configLock struct {
	key    config.LockKey
	userID int64
}

// Holds a pair of a context and its cancel function.
type contextPair struct {
	context context.Context
	cancel  context.CancelFunc
}

// Configuration manager implementation. The manager is responsible
// for coordinating configuration changes of the monitored daemons.
// It allows for applying the configuration changes instantly or
// scheduling the configuration changes to execute at the specified
// time.
type configManagerImpl struct {
	// Database handle used to schedule configuration changes.
	db *pg.DB
	// Interface to the agents that the manager communicates with.
	agents agentcomm.ConnectedAgents
	// Interface to the instance providing functions to search for
	// option definitions.
	lookup keaconfig.DHCPOptionDefinitionLookup
	// Holds contexts for present transactions. The unique context
	// identifier exchanged between the server and the client is a
	// key of this map.
	contexts map[int64]contextPair
	// A map holding acquired locks for daemons. The map key is an
	// ID of the daemon for which the lock has been acquired.
	locks map[int64]configLock
	// Last generated lock key.
	key config.LockKey
	// Config manager main mutex.
	mutex *sync.RWMutex
	// An interface to the Kea module commit function that the manager
	// calls to commit the changes in the Kea servers.
	keaCommit config.KeaModuleCommit
	// An interface to the configuration manager's module responsible
	// for managing Kea configuration.
	kea config.KeaModule
}

// Generates a key for a newly acquired lock. It is called internally
// by the Lock() function.
func (manager *configManagerImpl) generateKey() config.LockKey {
	manager.key++
	return manager.key
}

// Generates new context ID. This ID is returned to the client when the
// client begins a new configuration change, e.g. editing daemon
// configuration. The client must use this ID in its REST calls to
// identify the configuration change transaction. To avoid possible
// spoofing, the ID is randomized.
func (manager *configManagerImpl) generateContextID() (int64, error) {
	for i := 0; i < 10; i++ {
		// Limit the maximum number to uint32. Previously, we tried larger
		// values, i.e. int64, but it confused the typescript-based REST API
		// client which apparently converts all numbers to float. It
		// results in rounding the large int64 values causing mismatches.
		r, err := rand.Int(rand.Reader, big.NewInt(math.MaxUint32))
		if err != nil {
			return 0, pkgerrors.Wrapf(err, "failed to generate new context id")
		}
		if _, ok := manager.contexts[r.Int64()]; !ok {
			return r.Int64(), err
		}
	}
	return 0, pkgerrors.Errorf("failed to generate a unique context ID after several attempts")
}

// Checks if the confguration of the specified daemon is already locked
// for updates. The first returned parameter is an ID of the user who
// owns the lock. It is equal to 0 when the lock is not present.
func (manager *configManagerImpl) isLocked(daemonID int64) (int64, bool) {
	if lock, ok := manager.locks[daemonID]; ok {
		return lock.userID, true
	}
	return 0, false
}

// Attempts to acquire a lock on the specified daemon's configuration.
// It returns an error if the lock exists already. This function is
// called internally from the Lock() function.
func (manager *configManagerImpl) lock(ctx context.Context, daemonID int64) error {
	// Check if the daemon configuration has been locked already.
	if userID, locked := manager.isLocked(daemonID); locked {
		return pkgerrors.Errorf("configuration for daemon %d is locked for updates by user %d", daemonID, userID)
	}
	// The lock key should have been created in the Lock() function.
	lockKey, ok := ctx.Value(config.LockContextKey).(config.LockKey)
	if !ok {
		return pkgerrors.Errorf("context lacks lock key")
	}
	// The user id should have been set in the CreateContext() function.
	userID, ok := config.GetValueAsInt64(ctx, config.UserContextKey)
	if !ok {
		return pkgerrors.Errorf("context lacks user key")
	}
	// Acquire the lock on the daemon's configuration.
	manager.locks[daemonID] = configLock{
		key:    lockKey,
		userID: userID,
	}
	return nil
}

// Unlocks daemon's configuration if the lock stored in the context matches.
func (manager *configManagerImpl) unlockDaemon(ctx context.Context, daemonID int64) {
	// The lock key is required.
	lockKey, ok := ctx.Value(config.LockContextKey).(config.LockKey)
	if !ok {
		return
	}
	// If the configuration is locked and the lock key matches then unlock.
	if _, locked := manager.isLocked(daemonID); locked {
		if manager.locks[daemonID].key == lockKey {
			delete(manager.locks, daemonID)
			// Unlocked.
			return
		}
	}
	// Didn't unlock.
}

// Removes locks from the specified daemons' configurations if the
// key stored in the context matches.
func (manager *configManagerImpl) unlock(ctx context.Context) {
	if daemonIDs, ok := ctx.Value(config.DaemonsContextKey).([]int64); ok {
		for _, id := range daemonIDs {
			manager.unlockDaemon(ctx, id)
		}
	}
}

// Creates new configuration manager instance. The server parameter is an
// interface to the owner of the state required by the manager (i.e., an
// instance of the Stork Server holding the state.).
func NewManager(server config.ManagerAccessors) config.Manager {
	manager := &configManagerImpl{
		db:       server.GetDB(),
		agents:   server.GetConnectedAgents(),
		lookup:   server.GetDHCPOptionDefinitionLookup(),
		contexts: make(map[int64]contextPair),
		locks:    make(map[int64]configLock),
		mutex:    &sync.RWMutex{},
	}
	keaConfigModule := kea.NewConfigModule(manager)
	manager.kea = keaConfigModule
	manager.keaCommit = keaConfigModule
	return manager
}

// Returns the database handle instance. It is used by the configuration
// modules to access the database.
func (manager *configManagerImpl) GetDB() *pg.DB {
	return manager.db
}

// Returns the interface to the agents the manager communicates with.
func (manager *configManagerImpl) GetConnectedAgents() agentcomm.ConnectedAgents {
	return manager.agents
}

// Returns an interface to the instance providing the DHCP option definition
// lookup logic.
func (manager *configManagerImpl) GetDHCPOptionDefinitionLookup() keaconfig.DHCPOptionDefinitionLookup {
	return manager.lookup
}

// Returns Kea configuration module of the configuration manager.
func (manager *configManagerImpl) GetKeaModule() config.KeaModule {
	return manager.kea
}

// Creates the context for use with the configuration manager. It sets the
// unique context ID and a user identifier used to associate the context
// and the configuration change transaction with a user applying the
// configuration change.
func (manager *configManagerImpl) CreateContext(userID int64) (context.Context, error) {
	id, err := manager.generateContextID()
	if err != nil {
		return nil, err
	}
	ctx := context.WithValue(context.Background(), config.ContextIDKey, id)
	ctx = context.WithValue(ctx, config.UserContextKey, userID)
	return ctx, nil
}

// Stores the context in the manager for later use. If the context exists
// for the given context ID and the user ID matches, the context is replaced.
// If the user ID doesn't match, an error is returned.
func (manager *configManagerImpl) RememberContext(ctx context.Context, timeout time.Duration) error {
	var (
		contextID   int64
		userID      int64
		existingCtx contextPair
		ok          bool
	)
	// Retrieve context ID from the context. It will be used as a key to access
	// the stored context.
	if contextID, ok = config.GetValueAsInt64(ctx, config.ContextIDKey); !ok {
		return pkgerrors.New("context lacks context ID")
	}
	// Retrieve the user ID from the context. First, the user ID is mandatory in
	// the stored context. Also, we have to compare the user id with the corresponding
	// user ID in the already stored context.
	if userID, ok = config.GetValueAsInt64(ctx, config.UserContextKey); !ok {
		return pkgerrors.New("context lacks user ID")
	}
	manager.mutex.RLock()
	// Check if the context has been already stored under this context ID.
	existingCtx, ok = manager.contexts[contextID]
	manager.mutex.RUnlock()
	if ok {
		existingUserID, ok := config.GetValueAsInt64(existingCtx.context, config.UserContextKey)
		if !ok || existingUserID != userID {
			return pkgerrors.New("unable to remember the context because user ID is mismatched")
		}
		existingCtx.cancel()
	}
	// User is still actively working on the configuration. Push the
	// the timeout forward.
	ctx, cancel := context.WithCancel(ctx)
	manager.mutex.Lock()
	manager.contexts[contextID] = contextPair{
		context: ctx,
		cancel:  cancel,
	}
	manager.mutex.Unlock()
	// Run the goroutine watching for a timeout that may occur when
	// a user is inactive.
	go func(ctx context.Context, timeout time.Duration) {
		select {
		case <-ctx.Done():
			// Canceled the context to push the timeout forward.
		case <-time.After(timeout):
			// Timeout occurred because the user has been inactive.
			// Remove the context causing the user to start over.
			manager.Done(ctx)
		}
	}(ctx, timeout)
	return nil
}

// Returns a stored context for a given context ID and user ID. The user ID
// should come from the current user session.
func (manager *configManagerImpl) RecoverContext(contextID, userID int64) (context.Context, context.CancelFunc) {
	manager.mutex.RLock()
	defer manager.mutex.RUnlock()
	if ctx, ok := manager.contexts[contextID]; ok {
		if existingUserID, ok := config.GetValueAsInt64(ctx.context, config.UserContextKey); ok {
			if existingUserID == userID {
				return ctx.context, ctx.cancel
			}
		}
	}
	return nil, nil
}

// Attempts to lock configurations of the specified daemons for update.
// Internally it generates a new lock key and stores it in the context.
// If an attempt to lock any of the configurations fails, it will remove
// already acquired locks and return an error.
func (manager *configManagerImpl) Lock(ctx context.Context, daemonIDs ...int64) (context.Context, error) {
	// Generate a new lock key.
	ctx = context.WithValue(ctx, config.LockContextKey, manager.generateKey())
	manager.mutex.Lock()
	defer manager.mutex.Unlock()
	for _, id := range daemonIDs {
		// Try to acquire a lock for each daemon.
		if err := manager.lock(ctx, id); err != nil {
			// Locking failed. Remove the applied locks.
			for _, uid := range daemonIDs {
				if uid == id {
					break
				}
				manager.unlockDaemon(ctx, uid)
			}
			return ctx, err
		}
	}
	ctx = context.WithValue(ctx, config.DaemonsContextKey, daemonIDs)
	return ctx, nil
}

// Removes locks from the specified daemons' configurations if the
// key stored in the context matches.
func (manager *configManagerImpl) Unlock(ctx context.Context) {
	manager.mutex.Lock()
	defer manager.mutex.Unlock()
	manager.unlock(ctx)
}

// Cancels the context and removes it from the storage of remembered contexts.
// Finally, it unlocks the daemons locked with this context.
func (manager *configManagerImpl) Done(ctx context.Context) {
	manager.mutex.Lock()
	defer manager.mutex.Unlock()
	if contextID, ok := config.GetValueAsInt64(ctx, config.ContextIDKey); ok {
		if contextPair, ok := manager.contexts[contextID]; ok {
			if contextPair.cancel != nil {
				contextPair.cancel()
			}
			delete(manager.contexts, contextID)
		}
	}
	manager.unlock(ctx)
}

// Sends the configuration updates queued in the context to one or multiple daemons
// right away.
func (manager *configManagerImpl) Commit(ctx context.Context) (context.Context, error) {
	state, ok := config.GetAnyTransactionState(ctx)
	if !ok {
		return ctx, pkgerrors.Errorf("context lacks state")
	}
	var err error
	for _, pu := range state.GetUpdates() {
		switch pu.Target {
		case datamodel.AppTypeKea:
			// Kea configuration update. Route the call to Kea module.
			ctx, err = manager.keaCommit.Commit(ctx)
		default:
			err = pkgerrors.Errorf("unknown configured module name %s", pu.Target)
		}
		if err != nil {
			return ctx, err
		}
	}
	return ctx, err
}

// Commit all configuration changes in the database which are due, i.e. for which
// the deadline_at time expired.
func (manager *configManagerImpl) CommitDue() error {
	// Get due configuration changes.
	changes, err := dbmodel.GetDueConfigChanges(manager.GetDB())
	if err != nil {
		return err
	}
	// Nothing to do.
	if len(changes) == 0 {
		return nil
	}
	// Iterate over the changes.
	for _, change := range changes {
		var state any
		// Re-create the transaction state from the serialized data stored in
		// the database.
		switch {
		case change.HasKeaUpdates():
			keaState := config.TransactionState[kea.ConfigRecipe]{
				Scheduled: true,
			}
			for _, u := range change.Updates {
				update := kea.NewConfigUpdateFromDBModel(u)
				if update == nil {
					continue
				}
				keaState.Updates = append(keaState.Updates, update)
			}
			state = keaState
		default:
		}
		var (
			ctx context.Context
			err error
		)
		// Re-create the context.
		ctx, err = manager.CreateContext(change.UserID)
		if err == nil {
			ctx = context.WithValue(ctx, config.StateContextKey, state)
			// Commit the changes in the monitored daemons.
			_, err = manager.Commit(ctx)
		}
		var errtext string
		if err != nil {
			errtext = err.Error()
		}
		// Mark the current config change as executed.
		if err = dbmodel.SetScheduledConfigChangeExecuted(manager.GetDB(), change.ID, errtext); err != nil {
			return err
		}
	}
	return nil
}

// Schedules sending the changes queued in the context to one or multiple daemons.
// The deadline parameter specifies the time when the changes should be committed.
func (manager *configManagerImpl) Schedule(ctx context.Context, deadline time.Time) (context.Context, error) {
	state, ok := config.GetAnyTransactionState(ctx)
	if !ok {
		return ctx, pkgerrors.Errorf("context lacks state")
	}
	userID, ok := config.GetValueAsInt64(ctx, config.UserContextKey)
	if !ok {
		return ctx, pkgerrors.Errorf("context lacks user key")
	}
	// Create the config change entry in the database.
	scc := &dbmodel.ScheduledConfigChange{
		DeadlineAt: deadline,
		UserID:     userID,
	}
	for _, u := range state.GetUpdates() {
		update := &dbmodel.ConfigUpdate{
			Target:    u.Target,
			Operation: u.Operation,
			DaemonIDs: u.DaemonIDs,
		}
		recipe, err := json.Marshal(u.Recipe)
		if err != nil {
			return ctx, pkgerrors.Wrapf(err, "problem converting config update recipe to the raw format")
		}
		update.Recipe = (*json.RawMessage)(&recipe)
		scc.Updates = append(scc.Updates, update)
	}
	if err := dbmodel.AddScheduledConfigChange(manager.db, scc); err != nil {
		return ctx, err
	}
	return ctx, nil
}
