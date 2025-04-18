package configmigrator

import (
	"context"
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

const (
	unknownMigrationID      = MigrationIdentifier(0)
	inProgressMigrationID   = MigrationIdentifier(1)
	finishedMigrationID     = MigrationIdentifier(2)
	canceledMigrationID     = MigrationIdentifier(3)
	cancelingMigrationID    = MigrationIdentifier(4)
	generalErrorMigrationID = MigrationIdentifier(5)
)

// Creates a migration manager with some predefined migrations.
func newTestManager() MigrationManager {
	manager := NewMigrationManager().(*manager)
	// Migration in progress.
	manager.migrations[inProgressMigrationID] = &migration{
		id:             inProgressMigrationID,
		ctx:            context.Background(),
		startDate:      time.Date(2025, 2, 1, 12, 0, 0, 0, time.UTC),
		endDate:        time.Time{},
		generalError:   nil,
		processedItems: 10,
		totalItems:     100,
		errors:         []MigrationError{},
		cancelDate:     time.Time{},
		cancelFunc: func() {
			manager.migrations[inProgressMigrationID].cancelDate = time.Date(
				2025, 2, 1, 13, 0, 0, 0, time.UTC,
			)
		},
	}
	// Finished migration.
	manager.migrations[finishedMigrationID] = &migration{
		id:             finishedMigrationID,
		ctx:            context.Background(),
		startDate:      time.Date(2025, 2, 2, 11, 0, 0, 0, time.UTC),
		endDate:        time.Date(2025, 2, 2, 12, 0, 0, 0, time.UTC),
		generalError:   nil,
		processedItems: 100,
		totalItems:     100,
		errors: []MigrationError{{
			ID:    42,
			Error: errors.New("error"),
			Label: "host-finished",
			Type:  EntityTypeHost,
		}},
		cancelDate: time.Time{},
	}
	// Canceled migration.
	manager.migrations[canceledMigrationID] = &migration{
		id:             canceledMigrationID,
		ctx:            context.Background(),
		startDate:      time.Date(2025, 2, 3, 11, 0, 0, 0, time.UTC),
		cancelDate:     time.Date(2025, 2, 3, 12, 0, 0, 0, time.UTC),
		endDate:        time.Date(2025, 2, 3, 13, 0, 0, 0, time.UTC),
		generalError:   errors.New("canceled"),
		processedItems: 10,
		totalItems:     100,
		errors:         []MigrationError{},
	}
	// Canceling migration.
	manager.migrations[cancelingMigrationID] = &migration{
		id:             cancelingMigrationID,
		ctx:            context.Background(),
		startDate:      time.Date(2025, 2, 4, 11, 0, 0, 0, time.UTC),
		endDate:        time.Time{},
		generalError:   nil,
		cancelDate:     time.Date(2025, 2, 4, 12, 0, 0, 0, time.UTC),
		processedItems: 10,
		totalItems:     100,
		errors:         []MigrationError{},
	}
	// General error occurred.
	manager.migrations[generalErrorMigrationID] = &migration{
		id:             generalErrorMigrationID,
		ctx:            context.Background(),
		startDate:      time.Date(2025, 2, 5, 11, 0, 0, 0, time.UTC),
		endDate:        time.Date(2025, 2, 5, 12, 0, 0, 0, time.UTC),
		generalError:   errors.New("general error"),
		processedItems: 10,
		totalItems:     100,
		errors:         []MigrationError{},
		cancelDate:     time.Time{},
	}
	return manager
}

// Test that the manager instance is created correctly.
func TestNewManager(t *testing.T) {
	// Arrange & Act
	manager := NewMigrationManager().(*manager)

	// Assert
	require.NotNil(t, manager)
	require.NotNil(t, manager.migrations)
}

// Test that the migrations are listed correctly regardless their state.
func TestGetMigrations(t *testing.T) {
	// Arrange
	manager := newTestManager()

	// Act
	migrations := manager.GetMigrations()

	// Assert
	require.Len(t, migrations, 5)
	require.EqualValues(t, inProgressMigrationID, migrations[0].ID)
	require.EqualValues(t, finishedMigrationID, migrations[1].ID)
	require.EqualValues(t, canceledMigrationID, migrations[2].ID)
	require.EqualValues(t, cancelingMigrationID, migrations[3].ID)
	require.EqualValues(t, generalErrorMigrationID, migrations[4].ID)
}

// Test that the migration is retrieved correctly.
func TestGetMigration(t *testing.T) {
	// Arrange
	manager := newTestManager()

	// Act
	migrationInProgress, okInProgress := manager.GetMigration(inProgressMigrationID)
	migrationFinished, okFinished := manager.GetMigration(finishedMigrationID)
	migrationCanceled, okCanceled := manager.GetMigration(canceledMigrationID)
	migrationCanceling, okCanceling := manager.GetMigration(cancelingMigrationID)
	migrationGeneralError, okGeneralError := manager.GetMigration(generalErrorMigrationID)
	migrationUnknown, okUnknown := manager.GetMigration(unknownMigrationID)

	// Assert
	require.True(t, okInProgress)
	require.EqualValues(t, inProgressMigrationID, migrationInProgress.ID)
	require.True(t, okFinished)
	require.EqualValues(t, finishedMigrationID, migrationFinished.ID)
	require.True(t, okCanceled)
	require.EqualValues(t, canceledMigrationID, migrationCanceled.ID)
	require.True(t, okCanceling)
	require.EqualValues(t, cancelingMigrationID, migrationCanceling.ID)
	require.True(t, okGeneralError)
	require.EqualValues(t, generalErrorMigrationID, migrationGeneralError.ID)
	require.False(t, okUnknown)
	require.Nil(t, migrationUnknown)
}

// Test that the executing of the stop migration function calls the cancel
// function.
func TestStopMigrationCallsCancelFunction(t *testing.T) {
	// Arrange
	manager := newTestManager()
	migration, _ := manager.GetMigration(inProgressMigrationID)

	// Act
	status, ok := manager.StopMigration(migration.ID)

	// Assert
	require.True(t, ok)
	require.True(t, status.Canceling)
	require.Zero(t, status.GeneralError)
	require.Zero(t, status.EndDate)
}

// Test that the migration that is unknown cannot be stopped.
func TestStopUnknownMigration(t *testing.T) {
	// Arrange
	manager := newTestManager()

	// Act
	migration, ok := manager.StopMigration(unknownMigrationID)

	// Assert
	require.False(t, ok)
	require.Nil(t, migration)
}

// Test that the finished migration cannot be stopped again.
func TestStopFinishedMigration(t *testing.T) {
	// Arrange
	manager := newTestManager()
	migration, _ := manager.GetMigration(finishedMigrationID)

	// Act
	status, ok := manager.StopMigration(migration.ID)

	// Assert
	require.True(t, ok)
	require.False(t, status.Canceling)
	require.Zero(t, status.GeneralError)
	require.NotZero(t, status.EndDate)
}

// Test that the canceling migration cannot be stopped again.
func TestStopCancelingMigration(t *testing.T) {
	// Arrange
	manager := newTestManager()
	migration, _ := manager.GetMigration(cancelingMigrationID)

	// Act
	status, ok := manager.StopMigration(migration.ID)

	// Assert
	require.True(t, ok)
	require.True(t, status.Canceling)
	require.Zero(t, status.GeneralError)
	require.Zero(t, status.EndDate)
}

// Test that the finished migrations can be cleared.
func TestClearFinishedMigrations(t *testing.T) {
	// Arrange
	manager := newTestManager()

	// Act
	manager.ClearFinishedMigrations()

	// Assert
	migrations := manager.GetMigrations()
	require.Len(t, migrations, 2)
	require.EqualValues(t, inProgressMigrationID, migrations[0].ID)
	require.EqualValues(t, cancelingMigrationID, migrations[1].ID)
}

// Test that the migration is started and executed asynchronously.
// The errors should be aggregated.
func TestStartAndExecuteMigration(t *testing.T) {
	// Arrange
	manager := NewMigrationManager()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	migrator := NewMockMigrator(ctrl)

	gomock.InOrder(
		migrator.EXPECT().CountTotal().Return(int64(250), nil),

		migrator.EXPECT().Begin(),
		migrator.EXPECT().LoadItems(gomock.Eq(int64(0))).Return(int64(100), nil),
		migrator.EXPECT().LoadItems(gomock.Eq(int64(100))).Return(int64(100), nil),
		migrator.EXPECT().LoadItems(gomock.Eq(int64(200))).Return(int64(50), nil),
		migrator.EXPECT().LoadItems(gomock.Eq(int64(250))).Return(int64(0), nil),
		migrator.EXPECT().End(),
	)

	// Blocks the migration runner until the assertion of a particular chunk is
	// finished.
	assertionFinishedChan := make(chan struct{})
	defer close(assertionFinishedChan)
	callIndex := int64(0)

	migrator.EXPECT().Migrate().DoAndReturn(func() []MigrationError {
		// The migrator will wait for the assertion to finish before it
		// continues. Also, the assertion will wait for the migrator to finish
		// migrating the chunk before it continues.
		<-assertionFinishedChan
		// The runner does additional processing after the migrator does its
		// job. So, we cannot immediately run the assertions after the channel
		// is empty. We need to wait for the runner to finish its job by
		// calling t.Eventually.

		// Return some errors.
		callIndex++
		return []MigrationError{
			{ID: callIndex*100 + 1, Error: errors.New("error"), Label: fmt.Sprintf("host-%d", callIndex*100+1), Type: EntityTypeHost},
			{ID: callIndex*100 + 2, Error: errors.New("error"), Label: fmt.Sprintf("host-%d", callIndex*100+2), Type: EntityTypeHost},
			{ID: callIndex*100 + 3, Error: errors.New("error"), Label: fmt.Sprintf("host-%d", callIndex*100+3), Type: EntityTypeHost},
		}
	}).Times(3)

	type contextKey string
	ctx := context.WithValue(context.Background(), contextKey("key"), "value")

	// Act & Assert
	initialStatus, err := manager.StartMigration(ctx, migrator)
	require.NoError(t, err)

	// Check the initial status.
	require.NotEmpty(t, initialStatus.ID)
	require.False(t, initialStatus.Canceling)
	require.Equal(t, "value", initialStatus.Context.Value(contextKey("key")))
	require.Zero(t, initialStatus.EndDate)
	require.Zero(t, initialStatus.GeneralError)
	require.Zero(t, initialStatus.ProcessedItemsCount)
	require.EqualValues(t, 250, initialStatus.TotalItemsCount)
	require.Zero(t, initialStatus.EstimatedLeftTime)
	require.Empty(t, initialStatus.Errors)

	// Wait for the first chunk to be processed.
	assertionFinishedChan <- struct{}{}
	require.Eventually(t, func() bool {
		status, _ := manager.GetMigration(initialStatus.ID)
		return status.ProcessedItemsCount == 100
	}, 100*time.Millisecond, 10*time.Millisecond)

	// Check the status after the first chunk is migrated.
	firstChunkStatus, ok := manager.GetMigration(initialStatus.ID)
	require.True(t, ok)
	require.Equal(t, initialStatus.ID, firstChunkStatus.ID)
	require.False(t, firstChunkStatus.Canceling)
	require.Equal(t, "value", firstChunkStatus.Context.Value(contextKey("key")))
	require.Zero(t, firstChunkStatus.EndDate)
	require.Zero(t, firstChunkStatus.GeneralError)
	require.EqualValues(t, 100, firstChunkStatus.ProcessedItemsCount)
	require.NotZero(t, firstChunkStatus.EstimatedLeftTime)
	require.NotZero(t, firstChunkStatus.ElapsedTime)
	require.Len(t, firstChunkStatus.Errors, 3)
	require.EqualValues(t, 101, firstChunkStatus.Errors[0].ID)
	require.EqualValues(t, 102, firstChunkStatus.Errors[1].ID)
	require.EqualValues(t, 103, firstChunkStatus.Errors[2].ID)

	// Wait for the second chunk to be processed.
	assertionFinishedChan <- struct{}{}
	require.Eventually(t, func() bool {
		status, _ := manager.GetMigration(initialStatus.ID)
		return status.ProcessedItemsCount == 200
	}, 100*time.Millisecond, 10*time.Millisecond)

	// Check the status after the second chunk is migrated.
	secondChunkStatus, ok := manager.GetMigration(initialStatus.ID)
	require.True(t, ok)
	require.Equal(t, initialStatus.ID, secondChunkStatus.ID)
	require.False(t, secondChunkStatus.Canceling)
	require.Equal(t, "value", secondChunkStatus.Context.Value(contextKey("key")))
	require.Zero(t, secondChunkStatus.EndDate)
	require.Zero(t, secondChunkStatus.GeneralError)
	require.EqualValues(t, 200, secondChunkStatus.ProcessedItemsCount)
	require.NotZero(t, secondChunkStatus.EstimatedLeftTime)
	require.NotZero(t, secondChunkStatus.ElapsedTime)
	require.Len(t, secondChunkStatus.Errors, 6)
	require.EqualValues(t, 201, secondChunkStatus.Errors[3].ID)
	require.EqualValues(t, 202, secondChunkStatus.Errors[4].ID)
	require.EqualValues(t, 203, secondChunkStatus.Errors[5].ID)

	// Wait for the third chunk to be processed.
	assertionFinishedChan <- struct{}{}
	require.Eventually(t, func() bool {
		status, _ := manager.GetMigration(initialStatus.ID)
		return status.ProcessedItemsCount == 250
	}, 100*time.Millisecond, 10*time.Millisecond)

	// Check the status after the third chunk is migrated.
	thirdChunkStatus, ok := manager.GetMigration(initialStatus.ID)
	require.True(t, ok)
	require.Equal(t, initialStatus.ID, thirdChunkStatus.ID)
	require.False(t, thirdChunkStatus.Canceling)
	require.Equal(t, "value", thirdChunkStatus.Context.Value(contextKey("key")))
	require.NotZero(t, thirdChunkStatus.EndDate)
	require.Zero(t, thirdChunkStatus.GeneralError)
	require.EqualValues(t, 250, thirdChunkStatus.ProcessedItemsCount)
	require.Zero(t, thirdChunkStatus.EstimatedLeftTime)
	require.NotZero(t, thirdChunkStatus.ElapsedTime)
	require.Len(t, thirdChunkStatus.Errors, 9)
	require.EqualValues(t, 301, thirdChunkStatus.Errors[6].ID)
	require.EqualValues(t, 302, thirdChunkStatus.Errors[7].ID)
	require.EqualValues(t, 303, thirdChunkStatus.Errors[8].ID)
}

// Test that the migration is not started if an error occurs in the initial
// phase.
func TestStartMigrationErrorInInitialPhase(t *testing.T) {
	// Arrange
	manager := NewMigrationManager()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	migrator := NewMockMigrator(ctrl)

	migrator.EXPECT().CountTotal().Return(int64(0), errors.New("error"))

	// Act
	status, err := manager.StartMigration(context.Background(), migrator)

	// Assert
	require.Error(t, err)
	require.Empty(t, status)
}

// Test that the migration has unique ID.
func TestStartMigrationUniqueID(t *testing.T) {
	// Arrange
	manager := NewMigrationManager().(*manager)

	// Act
	id1 := manager.generateUniqueMigrationID()
	manager.migrations[id1] = &migration{id: id1}

	id2 := manager.generateUniqueMigrationID()

	// Assert
	require.NotEqual(t, id1, id2)
	require.EqualValues(t, 1, id1)
	require.EqualValues(t, 2, id2)
}

// Test that the loading error interrupts the migration.
func TestStartMigrationLoadingError(t *testing.T) {
	// Arrange
	manager := NewMigrationManager()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	migrator := NewMockMigrator(ctrl)

	gomock.InOrder(
		migrator.EXPECT().CountTotal().Return(int64(250), nil),
		migrator.EXPECT().Begin(),
		migrator.EXPECT().LoadItems(gomock.Eq(int64(0))).Return(int64(100), nil),
		migrator.EXPECT().LoadItems(gomock.Eq(int64(100))).Return(
			int64(100),
			errors.New("loading error"),
		),
		migrator.EXPECT().End(),
	)

	// Blocks the migration runner until the assertion of a particular chunk is
	// finished.
	assertionFinishedChan := make(chan struct{})
	defer close(assertionFinishedChan)

	migrator.EXPECT().Migrate().Do(func() {
		// The migrator will wait for the assertion to finish before it
		// continues. Also, the assertion will wait for the migrator to finish
		// migrating the chunk before it continues.
		<-assertionFinishedChan
		// The runner does additional processing after the migrator does its
		// job. So, we cannot immediately run the assertions after the channel
		// is empty. We need to wait for the runner to finish its job by
		// calling t.Eventually.
	}).Return([]MigrationError{}).Times(1)

	// Act & Assert
	initialStatus, err := manager.StartMigration(context.Background(), migrator)
	require.NoError(t, err)

	// Check the initial status.
	require.Zero(t, initialStatus.EndDate)
	require.Zero(t, initialStatus.GeneralError)
	require.Empty(t, initialStatus.Errors)

	// Wait for the first chunk to be processed.
	assertionFinishedChan <- struct{}{}
	require.Eventually(t, func() bool {
		status, _ := manager.GetMigration(initialStatus.ID)
		return status.GeneralError != nil
	}, 100*time.Millisecond, 10*time.Millisecond)

	// Check the status after the first chunk is migrated.
	firstChunkStatus, ok := manager.GetMigration(initialStatus.ID)
	require.True(t, ok)
	require.NotZero(t, firstChunkStatus.EndDate)
	require.ErrorContains(t, firstChunkStatus.GeneralError, "loading error")
	require.Empty(t, firstChunkStatus.Errors)
	require.EqualValues(t, 100, firstChunkStatus.ProcessedItemsCount)
}

// Test that the migration can be canceled. The context in the returned
// statuses should not include the canceling stuff.
func TestCancelMigration(t *testing.T) {
	// Arrange
	manager := NewMigrationManager()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	migrator := NewMockMigrator(ctrl)

	gomock.InOrder(
		migrator.EXPECT().CountTotal().Return(int64(250), nil),
		migrator.EXPECT().Begin(),
		migrator.EXPECT().LoadItems(gomock.Eq(int64(0))).Return(int64(100), nil),
		migrator.EXPECT().LoadItems(gomock.Eq(int64(100))).Return(int64(100), nil),
		migrator.EXPECT().End(),
	)

	// Blocks the migration runner until the assertion of a particular chunk is
	// finished.
	assertionFinishedChan := make(chan struct{})
	defer close(assertionFinishedChan)

	migrator.EXPECT().Migrate().Do(func() {
		// The migrator will wait for the assertion to finish before it
		// continues. Also, the assertion will wait for the migrator to finish
		// migrating the chunk before it continues.
		<-assertionFinishedChan
		// The runner does additional processing after the migrator does its
		// job. So, we cannot immediately run the assertions after the channel
		// is empty. We need to wait for the runner to finish its job by
		// calling t.Eventually.
	}).Return([]MigrationError{}).Times(2)

	// Act & Assert
	initialStatus, err := manager.StartMigration(context.Background(), migrator)
	require.NoError(t, err)

	// Check the initial status.
	require.False(t, initialStatus.Canceling)
	require.Zero(t, initialStatus.EndDate)
	require.NoError(t, initialStatus.GeneralError)
	require.Nil(t, initialStatus.Context.Done())

	// Wait for the first chunk to be processed.
	assertionFinishedChan <- struct{}{}
	require.Eventually(t, func() bool {
		status, _ := manager.GetMigration(initialStatus.ID)
		return status.ProcessedItemsCount == 100
	}, 100*time.Millisecond, 10*time.Millisecond)

	// Check the status after the first chunk is migrated.
	firstChunkStatus, ok := manager.GetMigration(initialStatus.ID)
	require.True(t, ok)
	require.False(t, firstChunkStatus.Canceling)
	require.Zero(t, firstChunkStatus.EndDate)
	require.NoError(t, firstChunkStatus.GeneralError)
	require.Nil(t, firstChunkStatus.Context.Done())

	// Cancel the migration.
	stopStatus, ok := manager.StopMigration(initialStatus.ID)

	// Check the canceled status.
	require.True(t, ok)
	require.NotNil(t, stopStatus)
	require.True(t, stopStatus.Canceling)
	require.Zero(t, stopStatus.EndDate)
	require.NoError(t, stopStatus.GeneralError)
	require.Nil(t, stopStatus.Context.Done())

	// Wait for the cancellation to be processed.
	assertionFinishedChan <- struct{}{}
	require.Eventually(t, func() bool {
		status, _ := manager.GetMigration(initialStatus.ID)
		return status.EndDate != time.Time{}
	}, 100*time.Millisecond, 10*time.Millisecond)

	// Check the status after the second chunk is migrated.
	secondChunkStatus, ok := manager.GetMigration(initialStatus.ID)
	require.True(t, ok)
	require.True(t, secondChunkStatus.Canceling)
	require.NotZero(t, secondChunkStatus.EndDate)
	require.ErrorContains(t, secondChunkStatus.GeneralError, "canceled")
	require.Nil(t, secondChunkStatus.Context.Done())
}

// Test that canceling the parent context doesn't cancel the migration.
func TestMigrationParentCancel(t *testing.T) {
	// Arrange
	manager := NewMigrationManager()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	migrator := NewMockMigrator(ctrl)

	gomock.InOrder(
		migrator.EXPECT().CountTotal().Return(int64(250), nil),
		migrator.EXPECT().Begin(),
		migrator.EXPECT().LoadItems(gomock.Eq(int64(0))).Return(int64(100), nil),
		migrator.EXPECT().LoadItems(gomock.Eq(int64(100))).Return(int64(100), nil),
		migrator.EXPECT().LoadItems(gomock.Eq(int64(200))).Return(int64(50), nil),
		migrator.EXPECT().LoadItems(gomock.Eq(int64(250))).Return(int64(0), nil),
		migrator.EXPECT().End(),
	)

	// Blocks the migration runner until the assertion of a particular chunk is
	// finished.
	assertionFinishedChan := make(chan struct{})
	defer close(assertionFinishedChan)

	migrator.EXPECT().Migrate().Do(func() {
		// The migrator will wait for the assertion to finish before it
		// continues. Also, the assertion will wait for the migrator to finish
		// migrating the chunk before it continues.
		<-assertionFinishedChan
		// The runner does additional processing after the migrator does its
		// job. So, we cannot immediately run the assertions after the channel
		// is empty. We need to wait for the runner to finish its job by
		// calling t.Eventually.
	}).Return([]MigrationError{}).Times(3)

	parentCtx, parentCancel := context.WithCancel(context.Background())

	// Act & Assert
	initialStatus, err := manager.StartMigration(parentCtx, migrator)
	require.NoError(t, err)

	// Check the initial status.
	require.False(t, initialStatus.Canceling)
	require.Zero(t, initialStatus.EndDate)
	require.NoError(t, initialStatus.GeneralError)
	require.Nil(t, initialStatus.Context.Done())

	// Wait for the first chunk to be processed.
	assertionFinishedChan <- struct{}{}
	require.Eventually(t, func() bool {
		status, _ := manager.GetMigration(initialStatus.ID)
		return status.ProcessedItemsCount == 100
	}, 100*time.Millisecond, 10*time.Millisecond)

	// Check the status after the first chunk is migrated.
	firstChunkStatus, ok := manager.GetMigration(initialStatus.ID)
	require.True(t, ok)
	require.False(t, firstChunkStatus.Canceling)
	require.Zero(t, firstChunkStatus.EndDate)
	require.NoError(t, firstChunkStatus.GeneralError)
	require.Nil(t, firstChunkStatus.Context.Done())

	// Cancel the parent context.
	parentCancel()

	// Cancellation should have no effect on the migration.
	assertionFinishedChan <- struct{}{}
	assertionFinishedChan <- struct{}{}

	require.Eventually(t, func() bool {
		status, _ := manager.GetMigration(initialStatus.ID)
		return status.EndDate != time.Time{}
	}, 100*time.Millisecond, 10*time.Millisecond)

	// Check the status after the second chunk is migrated.
	secondChunkStatus, ok := manager.GetMigration(initialStatus.ID)
	require.True(t, ok)
	require.False(t, secondChunkStatus.Canceling)
	require.NotZero(t, secondChunkStatus.EndDate)
	require.Nil(t, secondChunkStatus.GeneralError)
	require.Nil(t, secondChunkStatus.Context.Done())
}

// Test that the closing of the migration manager cancels all migrations and
// waits for them to finish.
func TestConcurrentMigrationsCloseManager(t *testing.T) {
	// Arrange
	manager := NewMigrationManager()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	migrator := NewMockMigrator(ctrl)

	migrator.EXPECT().CountTotal().Return(int64(math.MaxInt64), nil).Times(2)
	migrator.EXPECT().Begin().Times(2)
	migrator.EXPECT().End().Times(2)

	// Migrate infinitely.
	migrator.EXPECT().LoadItems(gomock.Any()).Return(int64(100), nil).AnyTimes()
	migrator.EXPECT().Migrate().Return([]MigrationError{}).AnyTimes()

	// Act & Assert
	initialStatus1, err := manager.StartMigration(context.Background(), migrator)
	require.NoError(t, err)
	initialStatus2, err := manager.StartMigration(context.Background(), migrator)
	require.NoError(t, err)

	// Check the initial status.
	require.False(t, initialStatus1.Canceling)
	require.False(t, initialStatus2.Canceling)
	require.Zero(t, initialStatus1.EndDate)
	require.Zero(t, initialStatus2.EndDate)
	require.NoError(t, initialStatus1.GeneralError)
	require.NoError(t, initialStatus2.GeneralError)
	require.Nil(t, initialStatus1.Context.Done())
	require.Nil(t, initialStatus2.Context.Done())

	// Close the migration manager.
	manager.Close()

	// Check the status after the manager is closed.
	closedStatus, ok := manager.GetMigration(initialStatus1.ID)
	require.True(t, ok)
	require.True(t, closedStatus.Canceling)
	require.NotZero(t, closedStatus.EndDate)
	require.ErrorContains(t, closedStatus.GeneralError, "canceled")

	closedStatus, ok = manager.GetMigration(initialStatus2.ID)
	require.True(t, ok)
	require.True(t, closedStatus.Canceling)
	require.NotZero(t, closedStatus.EndDate)
	require.ErrorContains(t, closedStatus.GeneralError, "canceled")
}
