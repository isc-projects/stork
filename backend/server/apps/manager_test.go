package apps

import (
	"context"
	"fmt"
	"testing"
	"time"

	pkgerrors "github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"isc.org/stork/datamodel"
	agentcommtest "isc.org/stork/server/agentcomm/test"
	"isc.org/stork/server/apps/kea"
	appstest "isc.org/stork/server/apps/test"
	"isc.org/stork/server/config"
	dbmodel "isc.org/stork/server/database/model"
	dbtest "isc.org/stork/server/database/test"
	storkutil "isc.org/stork/util"
)

// An error returned by the Commit function in fake Kea module.
type lackingStateError struct{}

// Error implementation.
func (lackingStateError) Error() string {
	return "context lacks state"
}

// Fake Kea module exposing a Commit function. It is used to
// test that the manager's Commit() function properly routes
// the calls to the Commit() function in the Kea module.
type fakeKeaModuleCommit struct {
	contexts []context.Context
	ops      []string
	err      error
}

// Creates new instance of the fake Kea module.
func newFakeKeaModuleCommit() *fakeKeaModuleCommit {
	return &fakeKeaModuleCommit{}
}

// Implementation of the fake Commit() function. It records
// the invoked commit operations and passed contexts.
func (fkm *fakeKeaModuleCommit) Commit(ctx context.Context) (context.Context, error) {
	state, ok := config.GetTransactionState[kea.ConfigRecipe](ctx)
	if !ok {
		return ctx, lackingStateError{}
	}
	fkm.contexts = append(fkm.contexts, ctx)
	for _, update := range state.Updates {
		fkm.ops = append(fkm.ops, fmt.Sprintf("%s.%s", update.Target, update.Operation))
	}
	return ctx, fkm.err
}

// Test creating new config manager instance.
func TestNewManager(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	agents := &agentcommtest.FakeAgents{}
	lookup := dbmodel.NewDHCPOptionDefinitionLookup()

	manager := NewManager(&appstest.ManagerAccessorsWrapper{
		DB:        db,
		Agents:    agents,
		DefLookup: lookup,
	})
	require.NotNil(t, manager)
	require.NotNil(t, manager.GetKeaModule())

	impl, ok := manager.(*configManagerImpl)
	require.True(t, ok)
	require.NotNil(t, impl)
	require.Equal(t, db, impl.GetDB())
	require.Equal(t, agents, impl.GetConnectedAgents())
	require.Equal(t, lookup, impl.GetDHCPOptionDefinitionLookup())
}

// Test creating new context with context ID and user ID.
func TestCreateContext(t *testing.T) {
	manager := NewManager(&appstest.ManagerAccessorsWrapper{})
	require.NotNil(t, manager)

	// Gather the generated context ids in the map to ensure
	// that each created context has a unique context ID.
	ids := make(map[int64]bool)
	for i := 0; i < 10; i++ {
		// Create new context with user ID between 0 and 9.
		ctx, err := manager.CreateContext(int64(i))
		require.NoError(t, err)
		require.NotNil(t, ctx)

		// Make sure that the context ID exists.
		ctxid, ok := config.GetValueAsInt64(ctx, config.ContextIDKey)
		require.True(t, ok)
		ids[ctxid] = true

		// Make sure that the user ID exists.
		userid, ok := config.GetValueAsInt64(ctx, config.UserContextKey)
		require.True(t, ok)
		require.EqualValues(t, i, userid)
	}
	// Ensure that each call to CreateContext generated new ID.
	require.Len(t, ids, 10)
}

// Test that a created context can be remembered and then recovered
// by context ID and user ID.
func TestRememberRecoverContext(t *testing.T) {
	manager := NewManager(&appstest.ManagerAccessorsWrapper{})
	require.NotNil(t, manager)

	// Create first context with user ID 123.
	ctx1, err := manager.CreateContext(int64(123))
	require.NoError(t, err)
	require.NotNil(t, ctx1)

	// Linters do not like when simple types are used for keys in the context.
	type testContextKeyType string

	// Add some additional data specific to this context.
	key := testContextKeyType("foo")
	ctx1 = context.WithValue(ctx1, key, "bar")

	// Retrieve the generated context ID. It will be later needed
	// to recover the context.
	id1, ok := config.GetValueAsInt64(ctx1, config.ContextIDKey)
	require.True(t, ok)

	// Store the context.
	err = manager.RememberContext(ctx1, time.Minute*10)
	require.NoError(t, err)
	defer manager.Done(ctx1)

	// Try to recover the context by retrieved ID and user ID.
	recovered1, cancel1 := manager.RecoverContext(id1, 123)
	require.NotNil(t, recovered1)
	require.NotNil(t, cancel1)

	// The context ID and user ID should be present in the recovered context.
	_, ok = config.GetValueAsInt64(recovered1, config.ContextIDKey)
	require.True(t, ok)
	user1, ok := config.GetValueAsInt64(recovered1, config.UserContextKey)
	require.True(t, ok)
	require.EqualValues(t, 123, user1)

	// Ensure that the context specific information is also present.
	foo, ok := recovered1.Value(key).(string)
	require.True(t, ok)
	require.Equal(t, "bar", foo)

	// Repeat the same test for the second context. context.
	ctx2, err := manager.CreateContext(int64(234))
	require.NoError(t, err)
	require.NotNil(t, ctx2)

	key = testContextKeyType("bar")
	ctx2 = context.WithValue(ctx2, key, "baz")

	id2, ok := config.GetValueAsInt64(ctx2, config.ContextIDKey)
	require.True(t, ok)

	err = manager.RememberContext(ctx2, time.Minute*10)
	require.NoError(t, err)
	defer manager.Done(ctx2)

	recovered2, cancel2 := manager.RecoverContext(id2, 234)
	require.NotNil(t, recovered2)
	require.NotNil(t, cancel2)

	_, ok = config.GetValueAsInt64(recovered2, config.ContextIDKey)
	require.True(t, ok)
	user2, ok := config.GetValueAsInt64(recovered2, config.UserContextKey)
	require.True(t, ok)
	require.EqualValues(t, 234, user2)

	bar, ok := recovered2.Value(key).(string)
	require.True(t, ok)
	require.Equal(t, "baz", bar)
}

// Test the case when a timeout occurs during config update.
func TestContextTimeout(t *testing.T) {
	manager := NewManager(&appstest.ManagerAccessorsWrapper{})
	require.NotNil(t, manager)

	ctx, err := manager.CreateContext(int64(123))
	require.NoError(t, err)
	require.NotNil(t, ctx)

	contextID, ok := config.GetValueAsInt64(ctx, config.ContextIDKey)
	require.True(t, ok)

	// Remember the context.
	err = manager.RememberContext(ctx, time.Second*10)
	require.NoError(t, err)

	// Use the context to lock the daemon 1.
	ctx, err = manager.Lock(ctx, 1)
	require.NoError(t, err)
	defer manager.Unlock(ctx)

	// Remember the context again. It specifies a very short timeout
	// overriding the previous timeout of 10s.
	err = manager.RememberContext(ctx, time.Microsecond)
	require.NoError(t, err)

	// Wait for a timeout. When the timeout elapses, an attempt to recover
	// the context should return nil because the context should be removed
	// after the timeout.
	require.Eventually(t, func() bool {
		ctx, _ := manager.RecoverContext(contextID, 123)
		return ctx == nil
	}, time.Second, time.Millisecond)

	// Try to lock the configuration on daemon 1. It should succeed because
	// the configuration should have been unlocked after the timeout.
	ctxLock, err := manager.CreateContext(int64(234))
	require.NoError(t, err)
	require.NotNil(t, ctxLock)
	require.Eventually(t, func() bool {
		ctxLock, err = manager.Lock(ctxLock, 1)
		defer manager.Unlock(ctxLock)
		return err == nil
	}, time.Second, time.Millisecond)
}

// Test that calling Done() function results in removing the context and
// unlocking the configuration.
func TestDone(t *testing.T) {
	manager := NewManager(&appstest.ManagerAccessorsWrapper{})
	require.NotNil(t, manager)

	ctx, err := manager.CreateContext(int64(123))
	require.NoError(t, err)
	require.NotNil(t, ctx)

	contextID, ok := config.GetValueAsInt64(ctx, config.ContextIDKey)
	require.True(t, ok)

	ctx, err = manager.Lock(ctx, 1)
	require.NoError(t, err)
	defer manager.Unlock(ctx)

	err = manager.RememberContext(ctx, time.Second*10)
	require.NoError(t, err)

	manager.Done(ctx)

	// An attempt to recover the context should return nil.
	ctx, cancel := manager.RecoverContext(contextID, 123)
	require.Nil(t, ctx)
	require.Nil(t, cancel)

	// An attempt to lock the daemon configuration should succeed
	// because the previous lock should have been removed as a result
	// of calling Done().
	ctxLock, err := manager.CreateContext(int64(234))
	require.NoError(t, err)
	require.NotNil(t, ctxLock)
	_, err = manager.Lock(ctxLock, 1)
	require.NoError(t, err)
	manager.Unlock(ctxLock)
}

// Test that that an error is returned upon an attempt to remember the context
// under the specific context ID when user ID doesn't match.
func TestRememberContextWithMismatchedUserID(t *testing.T) {
	manager := NewManager(&appstest.ManagerAccessorsWrapper{})
	require.NotNil(t, manager)

	// Create context with user ID 123.
	ctx, err := manager.CreateContext(int64(123))
	require.NoError(t, err)
	require.NotNil(t, ctx)

	// Remember the context.
	err = manager.RememberContext(ctx, time.Minute*10)
	require.NoError(t, err)

	// Retrieve the context ID. We are going to use this ID instead of the
	// user ID when trying to replace the remembered context. It should
	// cause the mismatch.
	id, ok := config.GetValueAsInt64(ctx, config.ContextIDKey)
	require.True(t, ok)

	// In unlikely event that both ids happen to be equal, modify the
	// ID to avoid the test failure.
	if id == 123 {
		id++
	}
	ctx = context.WithValue(ctx, config.UserContextKey, id)
	err = manager.RememberContext(ctx, time.Minute*10)
	require.Error(t, err)
}

// Test that nil context is returned when user ID or context ID doesn't
// match the remembered values.
func TestRecoverContextMismatch(t *testing.T) {
	manager := NewManager(&appstest.ManagerAccessorsWrapper{})
	require.NotNil(t, manager)

	// Create first context with user ID 123.
	ctx1, err := manager.CreateContext(int64(123))
	require.NoError(t, err)
	require.NotNil(t, ctx1)
	id1, ok := config.GetValueAsInt64(ctx1, config.ContextIDKey)
	require.True(t, ok)
	err = manager.RememberContext(ctx1, time.Minute*10)
	require.NoError(t, err)

	// Create second context with user ID 234.
	ctx2, err := manager.CreateContext(int64(234))
	require.NoError(t, err)
	require.NotNil(t, ctx2)
	id2, ok := config.GetValueAsInt64(ctx2, config.ContextIDKey)
	require.True(t, ok)
	err = manager.RememberContext(ctx2, time.Minute*10)
	require.NoError(t, err)

	// When a user ID or context ID doesn't match the nil context
	// should be returned.
	recovered, cancel := manager.RecoverContext(id1, 234)
	require.Nil(t, recovered)
	require.Nil(t, cancel)
	recovered, cancel = manager.RecoverContext(id2, 123)
	require.Nil(t, recovered)
	require.Nil(t, cancel)
	recovered, cancel = manager.RecoverContext(111, 111)
	require.Nil(t, recovered)
	require.Nil(t, cancel)
}

// Test that daemon configurations can be locked for updates and then
// unlocked allowing for locking again.
func TestLockUnlock(t *testing.T) {
	manager := NewManager(&appstest.ManagerAccessorsWrapper{})
	require.NotNil(t, manager)

	// Create context and lock daemons 1, 2, 3.
	ctx1, err := manager.CreateContext(123)
	require.NoError(t, err)
	ctx1, err = manager.Lock(ctx1, 1, 2, 3)
	require.NoError(t, err)

	// An attempt to lock one of these daemons should fail.
	_, err = manager.Lock(ctx1, 4, 1)
	require.Error(t, err)

	// Create another context and try to lock unlocked daemon by different user.
	ctx2, err := manager.CreateContext(234)
	require.NoError(t, err)
	ctx2, err = manager.Lock(ctx2, 4)
	require.NoError(t, err)

	// Locking already locked daemon should fail.
	_, err = manager.Lock(ctx2, 1)
	require.Error(t, err)

	// Unlock the daemons locked by the first user.
	manager.Unlock(ctx1)

	// An attempt to lock the daemon should this time pass.
	_, err = manager.Lock(ctx2, 1)
	require.NoError(t, err)
}

// Test that the commit call is routed to the Kea module when the
// transaction target is dbmodel.AppTypeKea.
func TestCommitKeaModule(t *testing.T) {
	manager := NewManager(&appstest.ManagerAccessorsWrapper{})
	require.NotNil(t, manager)

	// Replace the interface for committing changes in the Kea
	// configuration module for the fake one.
	impl := manager.(*configManagerImpl)
	require.NotNil(t, impl)
	fkm := newFakeKeaModuleCommit()
	impl.keaCommit = fkm

	ctx, err := impl.CreateContext(123)
	require.NoError(t, err)

	// Create a new transaction with Kea.
	state := config.TransactionState[kea.ConfigRecipe]{
		Updates: []*config.Update[kea.ConfigRecipe]{
			config.NewUpdate[kea.ConfigRecipe](datamodel.AppTypeKea, "host_add"),
		},
	}
	ctx = context.WithValue(ctx, config.StateContextKey, state)

	// Commit the changes. They should result in a call to the Kea
	// module.
	_, err = manager.Commit(ctx)
	require.NoError(t, err)
	require.Len(t, fkm.ops, 1)
	require.Equal(t, "kea.host_add", fkm.ops[0])
}

// Test that an error is returned when unknown tool is specified in the
// Kea context.
func TestCommitUnknownTarget(t *testing.T) {
	manager := NewManager(&appstest.ManagerAccessorsWrapper{})
	require.NotNil(t, manager)

	ctx, err := manager.CreateContext(123)
	require.NoError(t, err)

	// Create a new transaction with unknown target.
	state := config.TransactionState[any]{
		Updates: []*config.Update[any]{
			config.NewUpdate[any]("unknown", "host_add"),
		},
	}
	ctx = context.WithValue(ctx, config.StateContextKey, state)

	// Commit the changes and expect an error.
	_, err = manager.Commit(ctx)
	require.Error(t, err)
}

// Test that due changes from the database are committed.
func TestCommitDue(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Scheduled config changes must be associated with a user.
	user := &dbmodel.SystemUser{
		Login:    "test",
		Lastname: "test",
		Name:     "test",
	}
	_, err := dbmodel.CreateUser(db, user)
	require.NoError(t, err)
	require.NotZero(t, user.ID)

	manager := NewManager(&appstest.ManagerAccessorsWrapper{
		DB: db,
	})
	require.NotNil(t, manager)

	// Replace the interface for committing changes in the Kea
	// configuration module for the fake one.
	impl := manager.(*configManagerImpl)
	require.NotNil(t, impl)
	fkm := newFakeKeaModuleCommit()
	impl.keaCommit = fkm

	// Add three config changes to the database. The first two are due. The
	// third one is still in the future.
	changes := []dbmodel.ScheduledConfigChange{
		{
			DeadlineAt: storkutil.UTCNow().Add(-time.Second * 10),
			UserID:     int64(user.ID),
			Updates: []*dbmodel.ConfigUpdate{
				dbmodel.NewConfigUpdate(dbmodel.AppTypeKea, "host_add"),
			},
		},
		{
			DeadlineAt: storkutil.UTCNow().Add(-time.Second * 100),
			UserID:     int64(user.ID),
			Updates: []*dbmodel.ConfigUpdate{
				dbmodel.NewConfigUpdate(dbmodel.AppTypeKea, "config_edit"),
			},
		},
		{
			DeadlineAt: storkutil.UTCNow().Add(time.Second * 100),
			UserID:     int64(user.ID),
			Updates: []*dbmodel.ConfigUpdate{
				dbmodel.NewConfigUpdate(dbmodel.AppTypeKea, "host_edit"),
			},
		},
	}
	for i := range changes {
		err := dbmodel.AddScheduledConfigChange(db, &changes[i])
		require.NoError(t, err)
	}
	// Commit due changes.
	err = manager.CommitDue()
	require.NoError(t, err)
	require.Len(t, fkm.ops, 2)
	// The changes should be ordered by deadline.
	require.Equal(t, "kea.config_edit", fkm.ops[0])
	require.Equal(t, "kea.host_add", fkm.ops[1])

	require.Len(t, fkm.contexts, 2)
	for _, ctx := range fkm.contexts {
		// Ensure that context ID exists.
		_, ok := config.GetValueAsInt64(ctx, config.ContextIDKey)
		require.True(t, ok)
		// Ensure that the user ID exists.
		userID, ok := config.GetValueAsInt64(ctx, config.UserContextKey)
		require.True(t, ok)
		require.EqualValues(t, user.ID, userID)
		// Ensure that the state exists and is correct.
		state, ok := config.GetTransactionState[kea.ConfigRecipe](ctx)
		require.True(t, ok)
		require.True(t, state.Scheduled)
	}

	// We have processed due config changes. They should no longer be returned.
	changes, err = dbmodel.GetDueConfigChanges(db)
	require.NoError(t, err)
	require.Empty(t, changes)
}

// Test that errors are recorded in the database when committing due
// config changes fails.
func TestCommitDueErrors(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Scheduled config changes must be associated with a user.
	user := &dbmodel.SystemUser{
		Login:    "test",
		Lastname: "test",
		Name:     "test",
	}
	_, err := dbmodel.CreateUser(db, user)
	require.NoError(t, err)
	require.NotZero(t, user.ID)

	manager := NewManager(&appstest.ManagerAccessorsWrapper{
		DB: db,
	})
	require.NotNil(t, manager)

	// Replace the interface for committing changes in the Kea
	// configuration module for the fake one.
	impl := manager.(*configManagerImpl)
	require.NotNil(t, impl)
	fkm := newFakeKeaModuleCommit()
	// Cause the Commit() function to return an error.
	fkm.err = pkgerrors.New("custom test error")
	impl.keaCommit = fkm

	// Add three config changes to the database. The first two are due. The
	// third one is still in the future.
	changes := []dbmodel.ScheduledConfigChange{
		{
			DeadlineAt: storkutil.UTCNow().Add(-time.Second * 10),
			UserID:     int64(user.ID),
			Updates: []*dbmodel.ConfigUpdate{
				dbmodel.NewConfigUpdate(dbmodel.AppTypeKea, "host_add"),
			},
		},
		{
			DeadlineAt: storkutil.UTCNow().Add(-time.Second * 100),
			UserID:     int64(user.ID),
			Updates: []*dbmodel.ConfigUpdate{
				dbmodel.NewConfigUpdate(dbmodel.AppTypeKea, "config_edit"),
			},
		},
	}
	for i := range changes {
		err := dbmodel.AddScheduledConfigChange(db, &changes[i])
		require.NoError(t, err)
	}
	// Commit due changes.
	err = manager.CommitDue()
	require.NoError(t, err)

	// The changes should have been marked as executed.
	returned, err := dbmodel.GetScheduledConfigChanges(db)
	require.NoError(t, err)
	require.Len(t, returned, 2)

	// Make sure that the errors have been recorded and that the executed
	// flags flipped.
	for _, change := range returned {
		require.True(t, change.Executed)
		require.Equal(t, "custom test error", change.Error)
	}
}

// Test that due changes are dropped if the user is deleted.
func TestDeleteUserDropDueChanges(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Scheduled config changes must be associated with a user.
	user := &dbmodel.SystemUser{
		Login:    "test",
		Lastname: "test",
		Name:     "test",
	}
	_, err := dbmodel.CreateUser(db, user)
	require.NoError(t, err)
	require.NotZero(t, user.ID)

	manager := NewManager(&appstest.ManagerAccessorsWrapper{
		DB: db,
	})
	require.NotNil(t, manager)

	// Replace the interface for committing changes in the Kea
	// configuration module for the fake one.
	impl := manager.(*configManagerImpl)
	require.NotNil(t, impl)
	fkm := newFakeKeaModuleCommit()
	impl.keaCommit = fkm

	// Add a config changes to the database. The user is deleted before the
	// change is applied which is still in the future.
	changes := []dbmodel.ScheduledConfigChange{
		{
			DeadlineAt: storkutil.UTCNow().Add(time.Second * 100),
			UserID:     int64(user.ID),
			Updates: []*dbmodel.ConfigUpdate{
				dbmodel.NewConfigUpdate(dbmodel.AppTypeKea, "config_edit"),
			},
		},
	}
	for i := range changes {
		err := dbmodel.AddScheduledConfigChange(db, &changes[i])
		require.NoError(t, err)
	}

	err = dbmodel.DeleteUser(db, user)
	require.NoError(t, err)

	// We have processed due config changes. They should no longer be returned.
	changes, err = dbmodel.GetDueConfigChanges(db)
	require.NoError(t, err)
	require.Empty(t, changes)
}

// Test that it is ok to call CommitDue() when there are no due changes.
func TestCommitDueNoChanges(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	manager := NewManager(&appstest.ManagerAccessorsWrapper{
		DB: db,
	})
	require.NotNil(t, manager)

	// Replace the interface for committing changes in the Kea
	// configuration module for the fake one.
	impl := manager.(*configManagerImpl)
	require.NotNil(t, impl)
	fkm := newFakeKeaModuleCommit()
	impl.keaCommit = fkm

	err := manager.CommitDue()
	require.NoError(t, err)
	require.Empty(t, fkm.ops)
}

// Test that config changes can be scheduled to apply later.
func TestSchedule(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	manager := NewManager(&appstest.ManagerAccessorsWrapper{
		DB: db,
	})
	require.NotNil(t, manager)

	// Create a context with a config change.
	ctx, err := manager.CreateContext(1)
	require.NoError(t, err)

	// Simulate adding an app to the database and then storing the app information
	// as part of the scheduled config change.
	machine := &dbmodel.Machine{
		ID:        1,
		Address:   "localhost",
		AgentPort: 8080,
		State: dbmodel.MachineState{
			Hostname: "cool.example.org",
		},
	}

	var accessPoints []*dbmodel.AccessPoint
	accessPoints = dbmodel.AppendAccessPoint(accessPoints, dbmodel.AccessPointControl, "cool.example.org", "", 1234, true)
	app := &dbmodel.App{
		ID:           1,
		Name:         "foo",
		MachineID:    machine.ID,
		Machine:      machine,
		Type:         dbmodel.AppTypeKea,
		Active:       true,
		AccessPoints: accessPoints,
		Meta: dbmodel.AppMeta{
			Version: "2.0.1",
		},
		Daemons: []*dbmodel.Daemon{
			{
				ID:      1,
				AppID:   1,
				Name:    "dhcp4",
				Version: "2.0.1",
				Active:  true,
			},
			{
				ID:      2,
				AppID:   1,
				Name:    "ca",
				Version: "2.0.1",
				Active:  true,
			},
		},
	}

	// Create the state.
	state := config.TransactionState[kea.ConfigRecipe]{
		Updates: []*config.Update[kea.ConfigRecipe]{
			config.NewUpdate[kea.ConfigRecipe](dbmodel.AppTypeKea, "host_add"),
		},
	}
	// Store the app in the state/context.
	state.Updates[0].Recipe.Commands = []kea.ConfigCommand{
		{
			App: app,
		},
	}

	ctx = context.WithValue(ctx, config.StateContextKey, state)

	// Schedule the change.
	_, err = manager.Schedule(ctx, storkutil.UTCNow().Add(time.Second*100))
	require.NoError(t, err)

	// Ensure that the change has been added to the database.
	changes, err := dbmodel.GetScheduledConfigChanges(db)
	require.NoError(t, err)
	require.Len(t, changes, 1)
	require.Len(t, changes[0].Updates, 1)
	require.Equal(t, dbmodel.AppTypeKea, changes[0].Updates[0].Target)
	require.Equal(t, "host_add", changes[0].Updates[0].Operation)
	require.NotNil(t, changes[0].Updates[0].Recipe)

	update := kea.NewConfigUpdateFromDBModel(changes[0].Updates[0])
	require.NotNil(t, update)

	// Make sure the app information was retrieved.
	require.Len(t, update.Recipe.Commands, 1)
	require.NotNil(t, update.Recipe.Commands[0].App)
	appReturned := update.Recipe.Commands[0].App

	require.EqualValues(t, 1, appReturned.GetID())
	require.Equal(t, "foo", appReturned.GetName())
	require.Equal(t, dbmodel.AppTypeKea, appReturned.GetType())
	require.Equal(t, "2.0.1", appReturned.GetVersion())

	parsedMachine := appReturned.GetMachineTag()
	require.EqualValues(t, 1, parsedMachine.GetID())
	require.Equal(t, "localhost", parsedMachine.GetAddress())
	require.EqualValues(t, 8080, parsedMachine.GetAgentPort())
	require.Equal(t, "cool.example.org", parsedMachine.GetHostname())

	tags := appReturned.GetDaemonTags()
	require.Len(t, tags, 2)

	require.EqualValues(t, 1, tags[0].GetID())
	require.Equal(t, "dhcp4", tags[0].GetName())
	require.EqualValues(t, 1, tags[0].GetAppID())
	require.Equal(t, dbmodel.AppTypeKea, tags[0].GetAppType())

	require.EqualValues(t, 2, tags[1].GetID())
	require.Equal(t, "ca", tags[1].GetName())
	require.EqualValues(t, 1, tags[1].GetAppID())
	require.Equal(t, dbmodel.AppTypeKea, tags[1].GetAppType())
}
