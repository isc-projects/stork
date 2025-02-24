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

// Creates a migration service with some predefined migrations.
func newTestService() Service {
	service := NewService().(*service)
	// Migration in progress.
	service.migrations["in-progress"] = &migration{
		id:             "in-progress",
		ctx:            context.Background(),
		startDate:      time.Date(2025, 2, 1, 12, 0, 0, 0, time.UTC),
		endDate:        time.Time{},
		generalError:   nil,
		processedItems: 10,
		totalItems:     100,
		errors:         []MigrationError{},
		cancelDate:     time.Time{},
		cancelFunc: func() {
			service.migrations["in-progress"].cancelDate = time.Date(
				2025, 2, 1, 13, 0, 0, 0, time.UTC,
			)
		},
	}
	// Finished migration.
	service.migrations["finished"] = &migration{
		id:             "finished",
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
	service.migrations["canceled"] = &migration{
		id:             "canceled",
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
	service.migrations["canceling"] = &migration{
		id:             "canceling",
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
	service.migrations["general-error"] = &migration{
		id:             "general-error",
		ctx:            context.Background(),
		startDate:      time.Date(2025, 2, 5, 11, 0, 0, 0, time.UTC),
		endDate:        time.Date(2025, 2, 5, 12, 0, 0, 0, time.UTC),
		generalError:   errors.New("general error"),
		processedItems: 10,
		totalItems:     100,
		errors:         []MigrationError{},
		cancelDate:     time.Time{},
	}
	return service
}

// Test that the service instance is created correctly.
func TestNewService(t *testing.T) {
	// Arrange & Act
	service := NewService().(*service)

	// Assert
	require.NotNil(t, service)
	require.NotNil(t, service.migrations)
}

// Test that the migrations are listed correctly regardless their state.
func TestGetMigrations(t *testing.T) {
	// Arrange
	service := newTestService()

	// Act
	migrations := service.GetMigrations()

	// Assert
	require.Len(t, migrations, 5)
	require.EqualValues(t, "in-progress", migrations[0].ID)
	require.EqualValues(t, "finished", migrations[1].ID)
	require.EqualValues(t, "canceled", migrations[2].ID)
	require.EqualValues(t, "canceling", migrations[3].ID)
	require.EqualValues(t, "general-error", migrations[4].ID)
}

// Test that the migration is retrieved correctly.
func TestGetMigration(t *testing.T) {
	// Arrange
	service := newTestService()

	// Act
	migrationInProgress, okInProgress := service.GetMigration("in-progress")
	migrationFinished, okFinished := service.GetMigration("finished")
	migrationCanceled, okCanceled := service.GetMigration("canceled")
	migrationCanceling, okCanceling := service.GetMigration("canceling")
	migrationGeneralError, okGeneralError := service.GetMigration("general-error")
	migrationUnknown, okUnknown := service.GetMigration("unknown")

	// Assert
	require.True(t, okInProgress)
	require.EqualValues(t, "in-progress", migrationInProgress.ID)
	require.True(t, okFinished)
	require.EqualValues(t, "finished", migrationFinished.ID)
	require.True(t, okCanceled)
	require.EqualValues(t, "canceled", migrationCanceled.ID)
	require.True(t, okCanceling)
	require.EqualValues(t, "canceling", migrationCanceling.ID)
	require.True(t, okGeneralError)
	require.EqualValues(t, "general-error", migrationGeneralError.ID)
	require.False(t, okUnknown)
	require.EqualValues(t, MigrationStatus{}, migrationUnknown)
}

// Test that the executing of the stop migration function calls the cancel
// function.
func TestStopMigrationCallsCancelFunction(t *testing.T) {
	// Arrange
	service := newTestService()
	migration, _ := service.GetMigration("in-progress")

	// Act
	status, ok := service.StopMigration(migration.ID)

	// Assert
	require.True(t, ok)
	require.True(t, status.Canceling)
	require.Zero(t, status.GeneralError)
	require.Zero(t, status.EndDate)
}

// Test that the migration that is unknown cannot be stopped.
func TestStopUnknownMigration(t *testing.T) {
	// Arrange
	service := newTestService()

	// Act
	migration, ok := service.StopMigration("unknown")

	// Assert
	require.False(t, ok)
	require.EqualValues(t, MigrationStatus{}, migration)
}

// Test that the finished migration cannot be stopped again.
func TestStopFinishedMigration(t *testing.T) {
	// Arrange
	service := newTestService()
	migration, _ := service.GetMigration("finished")

	// Act
	status, ok := service.StopMigration(migration.ID)

	// Assert
	require.True(t, ok)
	require.False(t, status.Canceling)
	require.Zero(t, status.GeneralError)
	require.NotZero(t, status.EndDate)
}

// Test that the canceling migration cannot be stopped again.
func TestStopCancelingMigration(t *testing.T) {
	// Arrange
	service := newTestService()
	migration, _ := service.GetMigration("canceling")

	// Act
	status, ok := service.StopMigration(migration.ID)

	// Assert
	require.True(t, ok)
	require.True(t, status.Canceling)
	require.Zero(t, status.GeneralError)
	require.Zero(t, status.EndDate)
}

// Test that the finished migrations can be cleared.
func TestClearFinishedMigrations(t *testing.T) {
	// Arrange
	service := newTestService()

	// Act
	service.ClearFinishedMigrations()

	// Assert
	migrations := service.GetMigrations()
	require.Len(t, migrations, 2)
	require.EqualValues(t, "in-progress", migrations[0].ID)
	require.EqualValues(t, "canceling", migrations[1].ID)
}

// Test that the migration is started and executed asynchronously.
// The errors should be aggregated.
func TestStartAndExecuteMigration(t *testing.T) {
	// Arrange
	service := NewService()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	migrator := NewMockMigrator(ctrl)

	migrator.EXPECT().CountTotal().Return(int64(250), nil)

	migrator.EXPECT().LoadItems(gomock.Eq(int64(0))).Return(int64(100), nil)
	migrator.EXPECT().LoadItems(gomock.Eq(int64(100))).Return(int64(100), nil)
	migrator.EXPECT().LoadItems(gomock.Eq(int64(200))).Return(int64(50), nil)
	migrator.EXPECT().LoadItems(gomock.Eq(int64(250))).Return(int64(0), nil)

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
	initialStatus, err := service.StartMigration(ctx, migrator)
	require.NoError(t, err)

	// Check the initial status.
	require.NotEmpty(t, initialStatus.ID)
	require.False(t, initialStatus.Canceling)
	require.Equal(t, "value", initialStatus.Context.Value(contextKey("key")))
	require.Zero(t, initialStatus.EndDate)
	require.Zero(t, initialStatus.GeneralError)
	require.Zero(t, initialStatus.Progress)
	require.Zero(t, initialStatus.EstimatedLeftTime)
	require.Empty(t, initialStatus.Errors)

	// Wait for the first chunk to be processed.
	assertionFinishedChan <- struct{}{}
	require.Eventually(t, func() bool {
		status, _ := service.GetMigration(initialStatus.ID)
		return status.Progress-100.0/250.0 < 1e-6
	}, 100*time.Millisecond, 10*time.Millisecond)

	// Check the status after the first chunk is migrated.
	firstChunkStatus, ok := service.GetMigration(initialStatus.ID)
	require.True(t, ok)
	require.Equal(t, initialStatus.ID, firstChunkStatus.ID)
	require.False(t, firstChunkStatus.Canceling)
	require.Equal(t, "value", firstChunkStatus.Context.Value(contextKey("key")))
	require.Zero(t, firstChunkStatus.EndDate)
	require.Zero(t, firstChunkStatus.GeneralError)
	require.InDelta(t, firstChunkStatus.Progress, 100.0/250.0, 1e-6)
	require.NotZero(t, firstChunkStatus.EstimatedLeftTime)
	require.NotZero(t, firstChunkStatus.ElapsedTime)
	require.Len(t, firstChunkStatus.Errors, 3)
	require.EqualValues(t, 101, firstChunkStatus.Errors[0].ID)
	require.EqualValues(t, 102, firstChunkStatus.Errors[1].ID)
	require.EqualValues(t, 103, firstChunkStatus.Errors[2].ID)

	// Wait for the second chunk to be processed.
	assertionFinishedChan <- struct{}{}
	require.Eventually(t, func() bool {
		status, _ := service.GetMigration(initialStatus.ID)
		return status.Progress-200.0/250.0 < 1e-6
	}, 100*time.Millisecond, 10*time.Millisecond)

	// Check the status after the second chunk is migrated.
	secondChunkStatus, ok := service.GetMigration(initialStatus.ID)
	require.True(t, ok)
	require.Equal(t, initialStatus.ID, secondChunkStatus.ID)
	require.False(t, secondChunkStatus.Canceling)
	require.Equal(t, "value", secondChunkStatus.Context.Value(contextKey("key")))
	require.Zero(t, secondChunkStatus.EndDate)
	require.Zero(t, secondChunkStatus.GeneralError)
	require.InDelta(t, secondChunkStatus.Progress, 200.0/250.0, 1e-6)
	require.NotZero(t, secondChunkStatus.EstimatedLeftTime)
	require.NotZero(t, secondChunkStatus.ElapsedTime)
	require.Len(t, secondChunkStatus.Errors, 6)
	require.EqualValues(t, 201, secondChunkStatus.Errors[3].ID)
	require.EqualValues(t, 202, secondChunkStatus.Errors[4].ID)
	require.EqualValues(t, 203, secondChunkStatus.Errors[5].ID)

	// Wait for the third chunk to be processed.
	assertionFinishedChan <- struct{}{}
	require.Eventually(t, func() bool {
		status, _ := service.GetMigration(initialStatus.ID)
		return status.Progress-1.0 < 1e-6
	}, 100*time.Millisecond, 10*time.Millisecond)

	// Check the status after the third chunk is migrated.
	thirdChunkStatus, ok := service.GetMigration(initialStatus.ID)
	require.True(t, ok)
	require.Equal(t, initialStatus.ID, thirdChunkStatus.ID)
	require.False(t, thirdChunkStatus.Canceling)
	require.Equal(t, "value", thirdChunkStatus.Context.Value(contextKey("key")))
	require.NotZero(t, thirdChunkStatus.EndDate)
	require.Zero(t, thirdChunkStatus.GeneralError)
	require.InDelta(t, thirdChunkStatus.Progress, 1.0, 1e-6)
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
	service := NewService()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	migrator := NewMockMigrator(ctrl)

	migrator.EXPECT().CountTotal().Return(int64(0), errors.New("error"))

	// Act
	status, err := service.StartMigration(context.Background(), migrator)

	// Assert
	require.Error(t, err)
	require.Empty(t, status)
}

// Test that the migration has unique ID even if some migrations are started
// in exactly the same time.
func TestStartMigrationUniqueID(t *testing.T) {
	// Arrange
	service := NewService().(*service)

	// Act
	id1 := service.getUniqueMigrationID(time.Time{})
	service.migrations[id1] = &migration{id: id1}

	id2 := service.getUniqueMigrationID(time.Time{})

	// Assert
	require.NotEqual(t, id1, id2)
}

// Test that the loading error interrupts the migration.
func TestStartMigrationLoadingError(t *testing.T) {
	// Arrange
	service := NewService()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	migrator := NewMockMigrator(ctrl)

	migrator.EXPECT().CountTotal().Return(int64(250), nil)
	migrator.EXPECT().LoadItems(gomock.Eq(int64(0))).Return(int64(100), nil)
	migrator.EXPECT().LoadItems(gomock.Eq(int64(100))).Return(
		int64(100),
		errors.New("loading error"),
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
	initialStatus, err := service.StartMigration(context.Background(), migrator)
	require.NoError(t, err)

	// Check the initial status.
	require.Zero(t, initialStatus.EndDate)
	require.Zero(t, initialStatus.GeneralError)
	require.Empty(t, initialStatus.Errors)

	// Wait for the first chunk to be processed.
	assertionFinishedChan <- struct{}{}
	require.Eventually(t, func() bool {
		status, _ := service.GetMigration(initialStatus.ID)
		return status.GeneralError != nil
	}, 100*time.Millisecond, 10*time.Millisecond)

	// Check the status after the first chunk is migrated.
	firstChunkStatus, ok := service.GetMigration(initialStatus.ID)
	require.True(t, ok)
	require.NotZero(t, firstChunkStatus.EndDate)
	require.ErrorContains(t, firstChunkStatus.GeneralError, "loading error")
	require.Empty(t, firstChunkStatus.Errors)
	require.InDelta(t, 100.0/250.0, firstChunkStatus.Progress, 1e-6)
}

// Test that the migration can be canceled. The context in the returned
// statuses should not include the canceling stuff.
func TestCancelMigration(t *testing.T) {
	// Arrange
	service := NewService()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	migrator := NewMockMigrator(ctrl)

	migrator.EXPECT().CountTotal().Return(int64(250), nil)

	migrator.EXPECT().LoadItems(gomock.Eq(int64(0))).Return(int64(100), nil)
	migrator.EXPECT().LoadItems(gomock.Eq(int64(100))).Return(int64(100), nil)

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
	initialStatus, err := service.StartMigration(context.Background(), migrator)
	require.NoError(t, err)

	// Check the initial status.
	require.False(t, initialStatus.Canceling)
	require.Zero(t, initialStatus.EndDate)
	require.NoError(t, initialStatus.GeneralError)
	require.Nil(t, initialStatus.Context.Done())

	// Wait for the first chunk to be processed.
	assertionFinishedChan <- struct{}{}
	require.Eventually(t, func() bool {
		status, _ := service.GetMigration(initialStatus.ID)
		return status.Progress-100.0/250.0 < 1e-6
	}, 100*time.Millisecond, 10*time.Millisecond)

	// Check the status after the first chunk is migrated.
	firstChunkStatus, ok := service.GetMigration(initialStatus.ID)
	require.True(t, ok)
	require.False(t, firstChunkStatus.Canceling)
	require.Zero(t, firstChunkStatus.EndDate)
	require.NoError(t, firstChunkStatus.GeneralError)
	require.Nil(t, firstChunkStatus.Context.Done())

	// Cancel the migration.
	stopStatus, ok := service.StopMigration(initialStatus.ID)

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
		status, _ := service.GetMigration(initialStatus.ID)
		return status.EndDate != time.Time{}
	}, 100*time.Millisecond, 10*time.Millisecond)

	// Check the status after the second chunk is migrated.
	secondChunkStatus, ok := service.GetMigration(initialStatus.ID)
	require.True(t, ok)
	require.True(t, secondChunkStatus.Canceling)
	require.NotZero(t, secondChunkStatus.EndDate)
	require.ErrorContains(t, secondChunkStatus.GeneralError, "canceled")
	require.Nil(t, secondChunkStatus.Context.Done())
}

// Test that canceling the parent context doesn't cancel the migration.
func TestMigrationParentCancel(t *testing.T) {
	// Arrange
	service := NewService()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	migrator := NewMockMigrator(ctrl)

	migrator.EXPECT().CountTotal().Return(int64(250), nil)

	migrator.EXPECT().LoadItems(gomock.Eq(int64(0))).Return(int64(100), nil)
	migrator.EXPECT().LoadItems(gomock.Eq(int64(100))).Return(int64(100), nil)
	migrator.EXPECT().LoadItems(gomock.Eq(int64(200))).Return(int64(50), nil)
	migrator.EXPECT().LoadItems(gomock.Eq(int64(250))).Return(int64(0), nil)

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
	initialStatus, err := service.StartMigration(parentCtx, migrator)
	require.NoError(t, err)

	// Check the initial status.
	require.False(t, initialStatus.Canceling)
	require.Zero(t, initialStatus.EndDate)
	require.NoError(t, initialStatus.GeneralError)
	require.Nil(t, initialStatus.Context.Done())

	// Wait for the first chunk to be processed.
	assertionFinishedChan <- struct{}{}
	require.Eventually(t, func() bool {
		status, _ := service.GetMigration(initialStatus.ID)
		return status.Progress-100.0/250.0 < 1e-6
	}, 100*time.Millisecond, 10*time.Millisecond)

	// Check the status after the first chunk is migrated.
	firstChunkStatus, ok := service.GetMigration(initialStatus.ID)
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
		status, _ := service.GetMigration(initialStatus.ID)
		return status.EndDate != time.Time{}
	}, 100*time.Millisecond, 10*time.Millisecond)

	// Check the status after the second chunk is migrated.
	secondChunkStatus, ok := service.GetMigration(initialStatus.ID)
	require.True(t, ok)
	require.False(t, secondChunkStatus.Canceling)
	require.NotZero(t, secondChunkStatus.EndDate)
	require.Nil(t, secondChunkStatus.GeneralError)
	require.Nil(t, secondChunkStatus.Context.Done())
}

// Test that the closing of the migration service cancels all migrations and
// waits for them to finish.
func TestConcurrentMigrationsCloseService(t *testing.T) {
	// Arrange
	service := NewService()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	migrator := NewMockMigrator(ctrl)

	migrator.EXPECT().CountTotal().Return(int64(math.MaxInt64), nil)

	// Migrate infinitely.
	migrator.EXPECT().LoadItems(gomock.Any()).Return(int64(100), nil).AnyTimes()
	migrator.EXPECT().Migrate().Return([]MigrationError{}).AnyTimes()

	// Act & Assert
	initialStatus, err := service.StartMigration(context.Background(), migrator)
	require.NoError(t, err)

	// Check the initial status.
	require.False(t, initialStatus.Canceling)
	require.Zero(t, initialStatus.EndDate)
	require.NoError(t, initialStatus.GeneralError)
	require.Nil(t, initialStatus.Context.Done())

	// Close the migration service.
	service.Close()

	// Check the status after the service is closed.
	closedStatus, ok := service.GetMigration(initialStatus.ID)
	require.True(t, ok)
	require.True(t, closedStatus.Canceling)
	require.NotZero(t, closedStatus.EndDate)
	require.ErrorContains(t, closedStatus.GeneralError, "canceled")
}
